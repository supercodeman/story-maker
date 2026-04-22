// server/internal/service/suggestion.go
package service

import (
	"context"
	"fmt"
	"log"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// SuggestionService 联想生成服务
type SuggestionService struct {
	dispatcher    *agent.Dispatcher
	prefDAO       *dao.UserPreferenceDAO
	styleSvc      *WritingStyleService
	intentSvc     *IntentService
	modelRegistry *ModelRegistryService
}

// NewSuggestionService 创建 SuggestionService 实例
func NewSuggestionService(dispatcher *agent.Dispatcher, styleSvc *WritingStyleService, intentSvc *IntentService) *SuggestionService {
	return &SuggestionService{
		dispatcher: dispatcher,
		prefDAO:    dao.NewUserPreferenceDAO(),
		styleSvc:   styleSvc,
		intentSvc:  intentSvc,
	}
}

// SetModelRegistry 注入模型注册服务
func (s *SuggestionService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// SuggestRequest 联想请求
type SuggestRequest struct {
	NovelID       uint   `json:"novel_id" binding:"required"`
	PrecedingText string `json:"preceding_text" binding:"required"`
}

// Suggest 生成联想文本
// 输入：用户ID、小说ID、光标前文本（最多500字）
// 输出：联想文本（50-150字）
func (s *SuggestionService) Suggest(ctx context.Context, userID, novelID uint, precedingText string) (string, error) {
	// 截取最后 500 字
	runes := []rune(precedingText)
	if len(runes) > 500 {
		runes = runes[len(runes)-500:]
	}
	trimmedText := string(runes)

	// 构建轻量 Prompt
	var promptParts []string

	// 查询用户偏好
	pref, err := s.prefDAO.GetByUserNovel(userID, novelID)
	if err == nil && pref != nil && pref.PromptSummary != "" {
		promptParts = append(promptParts, fmt.Sprintf("【作者偏好】%s", pref.PromptSummary))
	}

	// 查询写作风格（轻量版，不传 scenePresetID）
	if s.styleSvc != nil {
		styleText := s.styleSvc.FormatStyleForPrompt(novelID, nil)
		if styleText != "" {
			// 截取前 200 字，避免 prompt 过长
			styleRunes := []rune(styleText)
			if len(styleRunes) > 200 {
				styleRunes = styleRunes[:200]
			}
			promptParts = append(promptParts, fmt.Sprintf("【写作规范】%s", string(styleRunes)))
		}
	}

	// 意图推断
	if s.intentSvc != nil {
		intent := s.intentSvc.InferIntent(trimmedText, novelID)
		intentText := s.intentSvc.FormatIntentForPrompt(intent)
		if intentText != "" {
			promptParts = append(promptParts, intentText)
		}
	}

	// 构建最终 prompt
	systemPrompt := "你是一个小说续写助手。根据上文内容自然续写50-150字，保持风格和情节一致。只输出续写内容，不要任何解释。"
	if len(promptParts) > 0 {
		for _, p := range promptParts {
			systemPrompt += "\n" + p
		}
	}

	userPrompt := trimmedText

	// 两层降级，参照 dispatcher 模式：
	// 1. 同 Provider 内模型降级（FallbackModels）
	// 2. 跨 Provider 降级
	req := &agent.TextRequest{
		Prompt:       userPrompt,
		CharacterCtx: systemPrompt,
		MaxTokens:    200,
		Temperature:  0.7,
	}

	primaryModel := "qwen"
	if s.modelRegistry != nil {
		primaryModel = s.modelRegistry.GetDefaultModel(model.CapTextGen)
	}
	result, err := s.callWithFallback(ctx, primaryModel, req)
	if err != nil {
		return "", err
	}

	// 限制输出长度
	resultRunes := []rune(result)
	if len(resultRunes) > 200 {
		result = string(resultRunes[:200])
	}
	return result, nil
}

// callWithFallback 先在 primaryProvider 内做模型降级，再跨 provider 降级
func (s *SuggestionService) callWithFallback(ctx context.Context, primaryProvider string, req *agent.TextRequest) (string, error) {
	// 尝试主 provider（默认模型）
	result, err := s.tryProvider(ctx, primaryProvider, "", req)
	if err == nil {
		return result, nil
	}
	if !agent.IsRetryableError(err) {
		return "", err
	}
	log.Printf("[Suggestion] %s default model failed: %v, trying fallback models", primaryProvider, err)

	// 同 provider 内模型降级
	provider, pErr := s.dispatcher.GetProvider(primaryProvider)
	if pErr == nil {
		for _, fbModel := range provider.FallbackModels() {
			result, err = s.tryProvider(ctx, primaryProvider, fbModel, req)
			if err == nil {
				return result, nil
			}
			if !agent.IsRetryableError(err) {
				return "", err
			}
			log.Printf("[Suggestion] %s/%s failed: %v", primaryProvider, fbModel, err)
		}
	}

	// 跨 provider 降级
	var crossProviders []string
	if s.modelRegistry != nil {
		crossProviders = s.modelRegistry.GetFallbackProviders(0, primaryProvider, model.CapTextGen)
	} else {
		crossProviders = []string{"qwen", "zhipu", "deepseek", "kimi"}
	}
	for _, name := range crossProviders {
		if name == primaryProvider {
			continue
		}
		result, err = s.tryProvider(ctx, name, "", req)
		if err == nil {
			return result, nil
		}
		if !agent.IsRetryableError(err) {
			continue
		}
		log.Printf("[Suggestion] cross-provider %s failed: %v", name, err)
	}

	return "", fmt.Errorf("all providers exhausted, last error: %w", err)
}

// tryProvider 尝试用指定 provider 和模型版本调用
func (s *SuggestionService) tryProvider(ctx context.Context, providerName, modelVersion string, req *agent.TextRequest) (string, error) {
	provider, err := s.dispatcher.GetProviderWithKey(ctx, providerName)
	if err != nil {
		return "", err
	}

	callReq := *req
	if modelVersion != "" {
		callReq.Model = modelVersion
	}

	resp, err := provider.GenerateText(ctx, &callReq)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
