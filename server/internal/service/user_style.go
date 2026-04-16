// server/internal/service/user_style.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// UserStyleService 用户风格业务逻辑层
type UserStyleService struct {
	dao             *dao.UserStyleDAO
	writingStyleDAO *dao.WritingStyleDAO
	dispatcher      *agent.Dispatcher
	modelRegistry   *ModelRegistryService
}

// SetModelRegistry 延迟注入模型注册中心
func (s *UserStyleService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// NewUserStyleService 创建 UserStyleService 实例
func NewUserStyleService(dispatcher *agent.Dispatcher) *UserStyleService {
	return &UserStyleService{
		dao:             dao.NewUserStyleDAO(),
		writingStyleDAO: dao.NewWritingStyleDAO(),
		dispatcher:      dispatcher,
	}
}

// List 获取用户的所有风格模板
func (s *UserStyleService) List(userID uint) ([]model.UserStyle, error) {
	return s.dao.ListByUserID(userID)
}

// Get 获取单个风格（校验归属）
func (s *UserStyleService) Get(id, userID uint) (*model.UserStyle, error) {
	style, err := s.dao.GetByID(id)
	if err != nil {
		return nil, err
	}
	if style.UserID != userID {
		return nil, fmt.Errorf("forbidden")
	}
	return style, nil
}

// Create 创建风格
func (s *UserStyleService) Create(style *model.UserStyle) error {
	s.validateEnums(style)
	return s.dao.Create(style)
}

// Update 更新风格（校验归属）
func (s *UserStyleService) Update(style *model.UserStyle, userID uint) error {
	existing, err := s.dao.GetByID(style.ID)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	s.validateEnums(style)
	style.UserID = userID // 防止篡改
	return s.dao.Update(style)
}

// Delete 删除风格（校验归属）
func (s *UserStyleService) Delete(id, userID uint) error {
	existing, err := s.dao.GetByID(id)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	return s.dao.Delete(id)
}

// validateEnums 校验枚举字段，不合法则回退默认值
func (s *UserStyleService) validateEnums(style *model.UserStyle) {
	if _, ok := model.ValidNarrativeVoices[style.NarrativeVoice]; !ok {
		style.NarrativeVoice = model.NarrativeThirdLimited
	}
	if _, ok := model.ValidTones[style.Tone]; !ok {
		style.Tone = model.ToneNeutral
	}
	if _, ok := model.ValidLanguageLevels[style.LanguageLevel]; !ok {
		style.LanguageLevel = model.LangStandard
	}
}

// aiGenerateResponse AI 生成风格的 JSON 响应结构
type aiGenerateResponse struct {
	NarrativeVoice    string `json:"narrative_voice"`
	Tone              string `json:"tone"`
	LanguageLevel     string `json:"language_level"`
	ReferenceAuthors  string `json:"reference_authors"`
	ForbiddenPatterns string `json:"forbidden_patterns"`
	CustomRules       string `json:"custom_rules"`
	CustomPrompt      string `json:"custom_prompt"`
}

// AIGenerate 根据描述调用 AI 生成风格配置
func (s *UserStyleService) AIGenerate(ctx context.Context, userID uint, description string) (*model.UserStyle, error) {
	systemPrompt := `你是一个写作风格分析专家。根据用户的描述，生成一套完整的写作风格配置。

输出严格 JSON 格式，字段如下：
- narrative_voice: 叙事视角，枚举值 first/third_limited/third_omniscient/multi_pov
- tone: 文风调性，枚举值 serious/humorous/lyrical/sharp/warm/neutral
- language_level: 语言风格，枚举值 literary/standard/colloquial/web_novel
- reference_authors: 参考作家（字符串）
- forbidden_patterns: 禁用句式（字符串）
- custom_rules: 自定义规范（字符串）
- custom_prompt: 完整的写作风格指令 prompt（字符串，供 AI 写作时直接使用）

仅输出 JSON，不要其他内容。`

	// 动态获取默认模型
	defaultModel := "deepseek"
	if s.modelRegistry != nil {
		defaultModel = s.modelRegistry.GetDefaultModel(model.CapTextGen)
	}

	task := &model.AITask{
		UserID:    userID,
		TaskType:  model.TaskTypeTextGen,
		ModelName: defaultModel,
		Prompt:    systemPrompt + "\n\n用户描述：" + description,
	}

	result, err := s.dispatcher.ExecuteSingle(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("AI 生成失败: %w", err)
	}

	textResp, ok := result.(*agent.TextResponse)
	if !ok {
		return nil, fmt.Errorf("AI 返回格式异常")
	}

	// 清理 markdown 代码块包裹
	content := strings.TrimSpace(textResp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var resp aiGenerateResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		log.Printf("[UserStyle] AI 返回 JSON 解析失败: %v, content: %s", err, content)
		return nil, fmt.Errorf("AI 返回格式解析失败")
	}

	style := &model.UserStyle{
		UserID:            userID,
		Description:       description,
		NarrativeVoice:    resp.NarrativeVoice,
		Tone:              resp.Tone,
		LanguageLevel:     resp.LanguageLevel,
		ReferenceAuthors:  resp.ReferenceAuthors,
		ForbiddenPatterns: resp.ForbiddenPatterns,
		CustomRules:       resp.CustomRules,
		CustomPrompt:      resp.CustomPrompt,
		IsAIGenerated:     true,
	}
	s.validateEnums(style)

	return style, nil
}

// FormatForPrompt 将用户风格格式化为 prompt 文本（供绑定后的小说使用）
func (s *UserStyleService) FormatForPrompt(style *model.UserStyle) string {
	var parts []string

	if label, ok := model.ValidNarrativeVoices[style.NarrativeVoice]; ok {
		parts = append(parts, fmt.Sprintf("- 叙事视角：%s", label))
	}
	if label, ok := model.ValidTones[style.Tone]; ok {
		parts = append(parts, fmt.Sprintf("- 文风调性：%s", label))
	}
	if label, ok := model.ValidLanguageLevels[style.LanguageLevel]; ok {
		parts = append(parts, fmt.Sprintf("- 语言风格：%s", label))
	}
	if style.ReferenceAuthors != "" {
		parts = append(parts, fmt.Sprintf("- 参考作家风格：%s", style.ReferenceAuthors))
	}
	if style.ForbiddenPatterns != "" {
		parts = append(parts, fmt.Sprintf("- 禁止使用的句式/表达：%s", style.ForbiddenPatterns))
	}
	if style.CustomRules != "" {
		parts = append(parts, fmt.Sprintf("- 额外写作规范：%s", style.CustomRules))
	}
	if style.CustomPrompt != "" {
		parts = append(parts, fmt.Sprintf("\n【自定义风格指令】\n%s", style.CustomPrompt))
	}

	return strings.Join(parts, "\n")
}

// BindToNovel 将用户风格绑定到小说的 WritingStyle
func (s *UserStyleService) BindToNovel(novelID, userStyleID uint) error {
	return s.writingStyleDAO.UpdateBoundUserStyleID(novelID, &userStyleID)
}

// UnbindFromNovel 解绑小说的用户风格
func (s *UserStyleService) UnbindFromNovel(novelID uint) error {
	return s.writingStyleDAO.UpdateBoundUserStyleID(novelID, nil)
}
