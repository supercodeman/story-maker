// server/internal/service/user_preference.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// UserPreferenceService 用户偏好提取服务
type UserPreferenceService struct {
	prefDAO       *dao.UserPreferenceDAO
	behaviorDAO   *dao.UserBehaviorDAO
	dispatcher    *agent.Dispatcher
	modelRegistry *ModelRegistryService
}

// SetModelRegistry 注入模型注册中心（可选依赖）
func (s *UserPreferenceService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// NewUserPreferenceService 创建 UserPreferenceService 实例
func NewUserPreferenceService(dispatcher *agent.Dispatcher) *UserPreferenceService {
	return &UserPreferenceService{
		prefDAO:     dao.NewUserPreferenceDAO(),
		behaviorDAO: dao.NewUserBehaviorDAO(),
		dispatcher:  dispatcher,
	}
}

// GetByUserNovel 获取用户偏好
func (s *UserPreferenceService) GetByUserNovel(userID, novelID uint) (*model.UserPreference, error) {
	return s.prefDAO.GetByUserNovel(userID, novelID)
}

// ExtractPreference 提取用户偏好（规则统计 + LLM 摘要）
func (s *UserPreferenceService) ExtractPreference(userID, novelID uint) error {
	// 查询最近 100 条行为事件
	events, err := s.behaviorDAO.ListEventsByUserNovel(userID, novelID, time.Time{}, 100)
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	// 1. 规则统计提取
	vocabProfile := s.extractVocabProfile(events)
	styleProfile := s.extractStyleProfile(events)
	feedbackProfile := s.extractFeedbackProfile(events)

	vocabJSON, _ := json.Marshal(vocabProfile)
	styleJSON, _ := json.Marshal(styleProfile)
	feedbackJSON, _ := json.Marshal(feedbackProfile)

	// 2. LLM 摘要提取
	promptSummary := s.generatePromptSummary(vocabProfile, styleProfile, feedbackProfile, events)

	// 3. 获取当前偏好版本
	existing, _ := s.prefDAO.GetByUserNovel(userID, novelID)
	version := 1
	if existing != nil {
		version = existing.Version + 1
	}

	pref := &model.UserPreference{
		UserID:            userID,
		NovelID:           novelID,
		VocabProfile:      string(vocabJSON),
		StyleProfile:      string(styleJSON),
		AIFeedbackProfile: string(feedbackJSON),
		PromptSummary:     promptSummary,
		EventCount:        len(events),
		Version:           version,
		UpdatedAt:         time.Now(),
	}

	return s.prefDAO.Upsert(pref)
}

// ========== 规则统计提取 ==========

// vocabStats 词汇统计结果
type vocabStats struct {
	TopWords []wordFreq `json:"top_words"`
}

type wordFreq struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// extractVocabProfile 从章节保存事件中提取高频词汇
func (s *UserPreferenceService) extractVocabProfile(events []model.UserBehaviorEvent) *vocabStats {
	wordMap := make(map[string]int)

	for _, e := range events {
		if e.EventType != model.BehaviorChapterSave {
			continue
		}
		var payload struct {
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(e.Payload), &payload) != nil || payload.Content == "" {
			continue
		}
		// 简单分词：按标点和空格切分
		words := splitToWords(payload.Content)
		for _, w := range words {
			if utf8.RuneCountInString(w) >= 2 {
				wordMap[w]++
			}
		}
	}

	// 排序取 Top 50
	var freqs []wordFreq
	for w, c := range wordMap {
		freqs = append(freqs, wordFreq{Word: w, Count: c})
	}
	sort.Slice(freqs, func(i, j int) bool { return freqs[i].Count > freqs[j].Count })
	if len(freqs) > 50 {
		freqs = freqs[:50]
	}

	return &vocabStats{TopWords: freqs}
}

// styleStats 句式统计结果
type styleStats struct {
	AvgSentenceLen float64 `json:"avg_sentence_len"` // 平均句长（字数）
	LongRatio      float64 `json:"long_ratio"`       // 长句占比（>30字）
	AvgParaLen     float64 `json:"avg_para_len"`     // 平均段落长度
}

// extractStyleProfile 从章节保存事件中提取句式统计
func (s *UserPreferenceService) extractStyleProfile(events []model.UserBehaviorEvent) *styleStats {
	var totalSentences, longSentences int
	var totalSentenceLen, totalParas, totalParaLen int

	for _, e := range events {
		if e.EventType != model.BehaviorChapterSave {
			continue
		}
		var payload struct {
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(e.Payload), &payload) != nil || payload.Content == "" {
			continue
		}

		// 按段落分割
		paras := strings.Split(payload.Content, "\n")
		for _, p := range paras {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			totalParas++
			paraLen := utf8.RuneCountInString(p)
			totalParaLen += paraLen

			// 按句号、问号、感叹号分句
			sentences := splitSentences(p)
			for _, sent := range sentences {
				sentLen := utf8.RuneCountInString(sent)
				if sentLen < 2 {
					continue
				}
				totalSentences++
				totalSentenceLen += sentLen
				if sentLen > 30 {
					longSentences++
				}
			}
		}
	}

	stats := &styleStats{}
	if totalSentences > 0 {
		stats.AvgSentenceLen = float64(totalSentenceLen) / float64(totalSentences)
		stats.LongRatio = float64(longSentences) / float64(totalSentences) * 100
	}
	if totalParas > 0 {
		stats.AvgParaLen = float64(totalParaLen) / float64(totalParas)
	}
	return stats
}

// feedbackStats AI 反馈统计
type feedbackStats struct {
	TotalActions int            `json:"total_actions"`
	AcceptRate   float64        `json:"accept_rate"`
	ModifyRate   float64        `json:"modify_rate"`
	ActionStats  map[string]int `json:"action_stats"` // 各 action 的采纳/拒绝计数
}

// extractFeedbackProfile 统计 AI 反馈偏好
func (s *UserPreferenceService) extractFeedbackProfile(events []model.UserBehaviorEvent) *feedbackStats {
	stats := &feedbackStats{ActionStats: make(map[string]int)}

	for _, e := range events {
		switch e.EventType {
		case model.BehaviorAIAccept:
			stats.TotalActions++
			stats.ActionStats["accept"]++
		case model.BehaviorAIReject:
			stats.TotalActions++
			stats.ActionStats["reject"]++
		case model.BehaviorAIModify:
			stats.ActionStats["modify"]++
		}
	}

	if stats.TotalActions > 0 {
		stats.AcceptRate = float64(stats.ActionStats["accept"]) / float64(stats.TotalActions) * 100
		stats.ModifyRate = float64(stats.ActionStats["modify"]) / float64(stats.TotalActions) * 100
	}
	return stats
}

// ========== LLM 摘要提取 ==========

// generatePromptSummary 调用 LLM 生成偏好摘要
func (s *UserPreferenceService) generatePromptSummary(vocab *vocabStats, style *styleStats, feedback *feedbackStats, events []model.UserBehaviorEvent) string {
	if s.dispatcher == nil {
		return s.buildRuleSummary(vocab, style, feedback)
	}

	// 提取最近 5 条 ai_modify 的 diff 摘要
	var modifyPatterns []string
	for _, e := range events {
		if e.EventType == model.BehaviorAIModify && len(modifyPatterns) < 5 {
			var payload struct {
				DiffSummary string `json:"diff_summary"`
			}
			if json.Unmarshal([]byte(e.Payload), &payload) == nil && payload.DiffSummary != "" {
				modifyPatterns = append(modifyPatterns, payload.DiffSummary)
			}
		}
	}

	// 构建 top words 字符串
	var topWords []string
	for _, w := range vocab.TopWords {
		if len(topWords) >= 20 {
			break
		}
		topWords = append(topWords, w.Word)
	}

	prompt := fmt.Sprintf(`根据以下用户写作行为数据，总结该用户在这部小说中的写作偏好（200字以内）：

【用词统计】高频词：%s
【句式统计】平均句长：%.0f字，长句占比：%.0f%%
【AI 反馈】采纳率：%.0f%%，修改率：%.0f%%
【修改倾向】用户采纳 AI 结果后常做的修改：%s

请输出简洁的偏好描述，用于指导 AI 后续生成时贴合该用户的风格。`,
		strings.Join(topWords, "、"),
		style.AvgSentenceLen,
		style.LongRatio,
		feedback.AcceptRate,
		feedback.ModifyRate,
		strings.Join(modifyPatterns, "；"),
	)

	// 尝试调用 LLM（两层降级：同 provider 模型降级 + 跨 provider 降级）
	req := &agent.TextRequest{
		Prompt:      prompt,
		MaxTokens:   300,
		Temperature: 0.3,
	}
	primaryModel := "qwen"
	if s.modelRegistry != nil {
		primaryModel = s.modelRegistry.GetDefaultModel(model.CapTextGen)
	}
	result, err := s.callWithFallback(context.Background(), primaryModel, req)
	if err != nil {
		log.Printf("[UserPreference] all providers failed: %v, using rule summary", err)
		return s.buildRuleSummary(vocab, style, feedback)
	}
	return result
}

// callWithFallback 同 provider 模型降级 + 跨 provider 降级
func (s *UserPreferenceService) callWithFallback(ctx context.Context, primaryProvider string, req *agent.TextRequest) (string, error) {
	// 主 provider 默认模型
	result, err := s.tryProvider(ctx, primaryProvider, "", req)
	if err == nil {
		return result, nil
	}
	if !agent.IsRetryableError(err) {
		return "", err
	}
	log.Printf("[UserPreference] %s default failed: %v, trying fallback", primaryProvider, err)

	// 账号级错误，跳过同 Provider 降级
	skipIntraFallback := agent.IsAccountLevelError(err)
	if skipIntraFallback {
		log.Printf("[UserPreference] %s 账号级错误，跳过同 Provider 降级", primaryProvider)
	}

	// 同 provider 内模型降级
	if !skipIntraFallback {
		provider, pErr := s.dispatcher.GetProvider(primaryProvider)
		if pErr == nil {
			for _, fbModel := range provider.FallbackModels() {
				result, err = s.tryProvider(ctx, primaryProvider, fbModel, req)
				if err == nil {
					return result, nil
				}
				if agent.IsAccountLevelError(err) {
					log.Printf("[UserPreference] %s/%s 账号级错误，跳过剩余同 Provider 降级", primaryProvider, fbModel)
					break
				}
				log.Printf("[UserPreference] %s/%s failed: %v", primaryProvider, fbModel, err)
			}
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
		log.Printf("[UserPreference] cross-provider %s failed: %v", name, err)
	}

	return "", fmt.Errorf("all providers exhausted: %w", err)
}

func (s *UserPreferenceService) tryProvider(ctx context.Context, providerName, modelVersion string, req *agent.TextRequest) (string, error) {
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

// buildRuleSummary 纯规则生成的偏好摘要（LLM 不可用时的 fallback）
func (s *UserPreferenceService) buildRuleSummary(vocab *vocabStats, style *styleStats, feedback *feedbackStats) string {
	var parts []string

	if style.AvgSentenceLen > 0 {
		sentStyle := "短句为主"
		if style.AvgSentenceLen > 20 {
			sentStyle = "长句为主"
		} else if style.AvgSentenceLen > 12 {
			sentStyle = "长短句交替"
		}
		parts = append(parts, fmt.Sprintf("句式偏好：%s（平均%.0f字/句）", sentStyle, style.AvgSentenceLen))
	}

	if feedback.TotalActions > 0 {
		parts = append(parts, fmt.Sprintf("AI采纳率：%.0f%%", feedback.AcceptRate))
	}

	if len(vocab.TopWords) > 0 {
		var words []string
		for i, w := range vocab.TopWords {
			if i >= 10 {
				break
			}
			words = append(words, w.Word)
		}
		parts = append(parts, fmt.Sprintf("常用词：%s", strings.Join(words, "、")))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "；")
}

// ========== 文本处理工具函数 ==========

// splitToWords 简单中文分词（按标点和空格切分）
func splitToWords(text string) []string {
	// 按常见标点和空格切分
	separators := []rune{
		'，', '。', '！', '？', '；', '：', '、',
		'\u201C', '\u201D', // ""
		'\u2018', '\u2019', // ''
		'（', '）', '【', '】', '《', '》',
		'\n', '\r', '\t', ' ',
	}
	sepSet := make(map[rune]bool, len(separators))
	for _, s := range separators {
		sepSet[s] = true
	}

	var words []string
	var current strings.Builder

	for _, r := range text {
		if sepSet[r] {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// splitSentences 按句号、问号、感叹号分句
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	// 最后一段（无句号结尾）
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}
