// server/internal/service/plot_structure.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// PlotStructureService 剧情结构模板服务层
type PlotStructureService struct {
	dao           *dao.PlotStructureDAO
	dispatcher    *agent.Dispatcher
	modelRegistry *ModelRegistryService
}

// SetModelRegistry 延迟注入模型注册中心
func (s *PlotStructureService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// NewPlotStructureService 创建 PlotStructureService 实例
func NewPlotStructureService(plotDAO *dao.PlotStructureDAO, dispatcher *agent.Dispatcher) *PlotStructureService {
	return &PlotStructureService{dao: plotDAO, dispatcher: dispatcher}
}

// ========== 请求参数定义 ==========

// CreatePlotTemplateRequest 创建剧情结构模板请求
type CreatePlotTemplateRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description"`
	Structure   string `json:"structure" binding:"required"` // JSON 格式
}

// UpdatePlotTemplateRequest 更新剧情结构模板请求
type UpdatePlotTemplateRequest struct {
	Name        string `json:"name" binding:"omitempty,max=100"`
	Description string `json:"description"`
	Structure   string `json:"structure"`
}

// AIGenerateTemplateRequest AI 辅助生成模板请求
type AIGenerateTemplateRequest struct {
	Description string `json:"description" binding:"required"` // 用户描述
	ModelName   string `json:"model_name"`
}

// ========== CRUD ==========

// List 获取系统模板 + 当前用户自定义模板
func (s *PlotStructureService) List(userID uint) ([]model.PlotStructureTemplate, error) {
	return s.dao.List(userID)
}

// Get 获取模板详情
func (s *PlotStructureService) Get(id uint) (*model.PlotStructureTemplate, error) {
	return s.dao.Get(id)
}

// Create 创建用户自定义模板
func (s *PlotStructureService) Create(userID uint, req *CreatePlotTemplateRequest) (*model.PlotStructureTemplate, error) {
	// 校验 Structure JSON 格式
	if !json.Valid([]byte(req.Structure)) {
		return nil, errors.New("structure must be valid JSON")
	}

	tpl := &model.PlotStructureTemplate{
		Name:        req.Name,
		Category:    "custom",
		Description: req.Description,
		Structure:   req.Structure,
		IsSystem:    false,
		UserID:      userID,
	}
	if err := s.dao.Create(tpl); err != nil {
		return nil, err
	}
	return tpl, nil
}

// Update 更新用户自定义模板（不允许修改系统模板）
func (s *PlotStructureService) Update(userID, id uint, req *UpdatePlotTemplateRequest) (*model.PlotStructureTemplate, error) {
	tpl, err := s.dao.Get(id)
	if err != nil {
		return nil, err
	}
	if tpl.IsSystem {
		return nil, errors.New("cannot modify system template")
	}
	if tpl.UserID != userID {
		return nil, errors.New("permission denied")
	}

	if req.Name != "" {
		tpl.Name = req.Name
	}
	tpl.Description = req.Description
	if req.Structure != "" {
		if !json.Valid([]byte(req.Structure)) {
			return nil, errors.New("structure must be valid JSON")
		}
		tpl.Structure = req.Structure
	}

	if err := s.dao.Update(tpl); err != nil {
		return nil, err
	}
	return tpl, nil
}

// Delete 删除用户自定义模板（不允许删除系统模板）
func (s *PlotStructureService) Delete(userID, id uint) error {
	tpl, err := s.dao.Get(id)
	if err != nil {
		return err
	}
	if tpl.IsSystem {
		return errors.New("cannot delete system template")
	}
	if tpl.UserID != userID {
		return errors.New("permission denied")
	}
	return s.dao.Delete(id)
}

// ========== AI 辅助生成 ==========

// AIGenerate AI 辅助生成剧情结构模板，返回 taskID 供前端轮询
func (s *PlotStructureService) AIGenerate(ctx context.Context, userID, portfolioID uint, req *AIGenerateTemplateRequest) (uint, error) {
	modelName := req.ModelName
	if modelName == "" {
		if s.modelRegistry != nil {
			modelName = s.modelRegistry.GetDefaultModel(model.CapTextGen)
		} else {
			modelName = "zhipu"
		}
	}

	systemPrompt := `你是一位专业的小说结构设计师。根据用户的描述，生成一个剧情结构模板。
你必须严格按照以下 JSON 格式输出，不要包含任何其他文字：
{
  "name": "模板名称",
  "description": "模板描述",
  "structure": [
    {"phase":1,"name":"阶段名称","description":"阶段描述","ratio":0.25,"beats":["节拍点1","节拍点2"]}
  ]
}
要求：
- ratio 所有阶段之和必须等于 1.0
- 每个阶段至少包含 2 个 beats
- 结构要符合叙事逻辑，有清晰的起承转合`

	userPrompt := fmt.Sprintf("请根据以下描述生成剧情结构模板：\n%s", req.Description)

	historyData := map[string]interface{}{
		"system_prompt": systemPrompt,
	}
	historyJSON, _ := json.Marshal(historyData)

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    "plot_template_generate",
		ModelName:   modelName,
		Prompt:      userPrompt,
		History:     string(historyJSON),
		Status:      model.TaskStatusPending,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, fmt.Errorf("dispatch plot template generate task failed: %w", err)
	}

	return task.ID, nil
}

// ========== 种子数据 ==========

// SeedSystemTemplates 初始化系统预置模板
func (s *PlotStructureService) SeedSystemTemplates() {
	templates := []model.PlotStructureTemplate{
		{
			Name:        "三幕式",
			Category:    "classic",
			Description: "好莱坞经典三幕结构：建置→对抗→解决",
			Structure:   `[{"phase":1,"name":"第一幕·建置","description":"介绍主角、世界观，引入激励事件","ratio":0.25,"beats":["日常世界","激励事件","拒绝召唤","跨越门槛"]},{"phase":2,"name":"第二幕·对抗","description":"递进挑战，盟友与敌人出现","ratio":0.50,"beats":["试炼之路","中点反转","至暗时刻","灵魂黑夜"]},{"phase":3,"name":"第三幕·解决","description":"高潮对决，主题升华","ratio":0.25,"beats":["最终决战","高潮","结局","新世界"]}]`,
			IsSystem:    true,
		},
		{
			Name:        "英雄之旅",
			Category:    "classic",
			Description: "坎贝尔十二阶段：从平凡世界到英雄归来",
			Structure:   `[{"phase":1,"name":"启程","description":"英雄离开日常世界","ratio":0.25,"beats":["平凡世界","冒险召唤","拒绝召唤","遇见导师","跨越第一道门槛"]},{"phase":2,"name":"启蒙","description":"英雄在特殊世界中经历考验","ratio":0.50,"beats":["考验、盟友与敌人","接近最深的洞穴","严峻考验","报酬"]},{"phase":3,"name":"回归","description":"英雄带着宝物返回","ratio":0.25,"beats":["返回之路","复活","携万能药回归"]}]`,
			IsSystem:    true,
		},
		{
			Name:        "起承转合",
			Category:    "classic",
			Description: "东方叙事四段式：起→承→转→合",
			Structure:   `[{"phase":1,"name":"起","description":"开篇引入，建立背景和人物","ratio":0.20,"beats":["背景铺陈","人物登场","初始矛盾"]},{"phase":2,"name":"承","description":"承接发展，深化矛盾","ratio":0.30,"beats":["情节推进","关系深化","伏笔埋设","小高潮"]},{"phase":3,"name":"转","description":"情节转折，出人意料","ratio":0.30,"beats":["重大转折","真相揭露","危机爆发","人物蜕变"]},{"phase":4,"name":"合","description":"收束结局，主题升华","ratio":0.20,"beats":["最终对决","矛盾化解","结局呈现","余韵留白"]}]`,
			IsSystem:    true,
		},
		{
			Name:        "悬疑推理",
			Category:    "suspense",
			Description: "谜题→线索→误导→揭秘的推理结构",
			Structure:   `[{"phase":1,"name":"谜题呈现","description":"案件发生，悬念建立","ratio":0.20,"beats":["日常打破","核心谜题出现","初步调查","第一个线索"]},{"phase":2,"name":"线索收集","description":"深入调查，线索交织","ratio":0.30,"beats":["多条线索并行","嫌疑人浮现","关键证据","矛盾加深"]},{"phase":3,"name":"误导与反转","description":"假象迷惑，真相隐藏","ratio":0.30,"beats":["红鲱鱼误导","错误推理","新证据推翻","真凶线索浮现"]},{"phase":4,"name":"真相揭秘","description":"抽丝剥茧，水落石出","ratio":0.20,"beats":["关键突破","推理链完成","真相大白","正义实现"]}]`,
			IsSystem:    true,
		},
		{
			Name:        "网文爽文节奏",
			Category:    "web_novel",
			Description: "金手指→打脸→升级→新地图循环",
			Structure:   `[{"phase":1,"name":"开局金手指","description":"主角获得核心优势","ratio":0.15,"beats":["废柴开局","奇遇获得金手指","初次试探","小试牛刀"]},{"phase":2,"name":"打脸升级","description":"以弱胜强，不断打脸","ratio":0.35,"beats":["遭人轻视","实力碾压","围观震惊","收获奖励"]},{"phase":3,"name":"势力扩张","description":"建立势力，扩大影响","ratio":0.30,"beats":["收服小弟","建立根据地","势力冲突","更强敌人出现"]},{"phase":4,"name":"新地图循环","description":"进入更大舞台，循环升级","ratio":0.20,"beats":["旧地图毕业","新世界开启","更高层次挑战","终极对决"]}]`,
			IsSystem:    true,
		},
		{
			Name:        "情感线双线",
			Category:    "romance",
			Description: "相遇→误会→靠近→危机→和好的情感结构",
			Structure:   `[{"phase":1,"name":"命运相遇","description":"两人初次相遇，产生火花","ratio":0.20,"beats":["各自日常","命运交汇","第一印象","心动萌芽"]},{"phase":2,"name":"误会与靠近","description":"在误会中逐渐了解彼此","ratio":0.30,"beats":["误会产生","被迫接触","逐渐了解","暧昧升温"]},{"phase":3,"name":"感情危机","description":"外部或内部因素导致感情危机","ratio":0.30,"beats":["感情确认","第三者/外部阻力","信任崩塌","痛苦分离"]},{"phase":4,"name":"和好如初","description":"克服障碍，感情升华","ratio":0.20,"beats":["各自成长","真相大白","重新靠近","幸福结局"]}]`,
			IsSystem:    true,
		},
	}

	for _, tpl := range templates {
		if err := s.dao.FirstOrCreateByName(&tpl); err != nil {
			log.Printf("[seed] failed to seed plot structure template '%s': %v", tpl.Name, err)
		}
	}
	log.Println("[seed] plot structure templates seeded")
}

// ========== 辅助方法 ==========

// FormatStructureForPrompt 将结构模板格式化为 prompt 文本
func (s *PlotStructureService) FormatStructureForPrompt(templateID uint) (string, error) {
	tpl, err := s.dao.Get(templateID)
	if err != nil {
		return "", err
	}

	var phases []struct {
		Phase       int      `json:"phase"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Ratio       float64  `json:"ratio"`
		Beats       []string `json:"beats"`
	}
	if err := json.Unmarshal([]byte(tpl.Structure), &phases); err != nil {
		return "", fmt.Errorf("invalid structure JSON: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【剧情结构模板：%s】\n", tpl.Name))
	sb.WriteString(fmt.Sprintf("说明：%s\n\n", tpl.Description))

	for _, p := range phases {
		sb.WriteString(fmt.Sprintf("阶段%d「%s」（占比 %.0f%%）：%s\n", p.Phase, p.Name, p.Ratio*100, p.Description))
		sb.WriteString("  节拍点：")
		sb.WriteString(strings.Join(p.Beats, " → "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("请严格按照以上结构模板的阶段划分和节拍点来组织大纲章节，确保每个阶段的章节数量比例与 ratio 一致。")
	return sb.String(), nil
}
