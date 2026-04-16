// server/internal/service/intent.go
package service

import (
	"strings"
	"unicode/utf8"

	"ai-curton/server/internal/dao"
)

// WritingIntent 创作意图推断结果
type WritingIntent struct {
	Scene      string   `json:"scene"`      // 场景类型：battle/dialogue/psychology/environment/transition
	Emotion    string   `json:"emotion"`    // 情感基调：tense/warm/sad/humorous/neutral
	Characters []string `json:"characters"` // 当前涉及的角色
	Direction  string   `json:"direction"`  // 情节走向：escalation/resolution/transition
}

// IntentService 意图理解服务（纯规则 + 关键词，不调 LLM）
type IntentService struct {
	knowledgeDAO *dao.KnowledgeDAO
}

// NewIntentService 创建 IntentService 实例
func NewIntentService() *IntentService {
	return &IntentService{
		knowledgeDAO: dao.NewKnowledgeDAO(),
	}
}

// 场景关键词映射
var sceneKeywords = map[string][]string{
	"battle":      {"剑", "杀", "血", "战", "攻", "防", "拳", "刀", "枪", "打", "斩", "劈", "挡", "闪", "冲", "击", "伤", "死", "武", "功力"},
	"dialogue":    {"说", "道", "问", "答", "笑", "叹", "喊", "叫", "回答", "开口", "说道", "问道", "笑道"},
	"psychology":  {"想", "思", "念", "忆", "感", "觉", "心", "意", "情", "忧", "惧", "怒", "喜", "悲", "恨", "爱"},
	"environment": {"天", "地", "山", "水", "风", "云", "雨", "雪", "月", "日", "花", "树", "林", "海", "河", "城"},
}

// 情感关键词映射
var emotionKeywords = map[string][]string{
	"tense":   {"紧张", "危险", "急", "快", "猛", "烈", "凶", "狠", "恐", "惊", "慌"},
	"warm":    {"温暖", "柔", "轻", "暖", "甜", "美", "幸福", "安", "静", "和"},
	"sad":     {"悲", "伤", "泪", "哭", "痛", "苦", "凄", "惨", "孤", "寂"},
	"humorous": {"笑", "乐", "趣", "逗", "滑稽", "搞笑", "哈", "嘿"},
}

// 情节走向关键词
var directionKeywords = map[string][]string{
	"escalation": {"突然", "忽然", "猛然", "骤然", "竟然", "居然", "却", "但是", "然而", "不料"},
	"resolution": {"终于", "最终", "结束", "完成", "解决", "平息", "安定", "和解"},
	"transition": {"之后", "随后", "接着", "然后", "于是", "便", "就这样", "不久"},
}

// InferIntent 从文本推断创作意图
// text: 最近 200 字的文本内容
// novelID: 小说 ID（用于匹配角色名）
func (s *IntentService) InferIntent(text string, novelID uint) *WritingIntent {
	// 截取最后 200 字
	runes := []rune(text)
	if len(runes) > 200 {
		runes = runes[len(runes)-200:]
	}
	recentText := string(runes)

	intent := &WritingIntent{
		Scene:    s.inferScene(recentText),
		Emotion:  s.inferEmotion(recentText),
		Direction: s.inferDirection(recentText),
	}

	// 匹配角色名
	intent.Characters = s.matchCharacters(recentText, novelID)

	return intent
}

// FormatIntentForPrompt 将意图格式化为 Prompt 注入文本
func (s *IntentService) FormatIntentForPrompt(intent *WritingIntent) string {
	if intent == nil {
		return ""
	}

	var parts []string
	sceneLabels := map[string]string{
		"battle": "战斗", "dialogue": "对话", "psychology": "心理",
		"environment": "环境描写", "transition": "过渡",
	}
	emotionLabels := map[string]string{
		"tense": "紧张", "warm": "温暖", "sad": "悲伤",
		"humorous": "幽默", "neutral": "平和",
	}
	directionLabels := map[string]string{
		"escalation": "升级", "resolution": "收束", "transition": "过渡",
	}

	if label, ok := sceneLabels[intent.Scene]; ok {
		parts = append(parts, "场景："+label)
	}
	if label, ok := emotionLabels[intent.Emotion]; ok {
		parts = append(parts, "情感："+label)
	}
	if label, ok := directionLabels[intent.Direction]; ok {
		parts = append(parts, "走向："+label)
	}
	if len(intent.Characters) > 0 {
		parts = append(parts, "角色："+strings.Join(intent.Characters, "、"))
	}

	if len(parts) == 0 {
		return ""
	}
	return "【当前创作意图】" + strings.Join(parts, "，")
}

// inferScene 推断场景类型
func (s *IntentService) inferScene(text string) string {
	maxScore := 0
	bestScene := "transition"

	for scene, keywords := range sceneKeywords {
		score := countKeywordHits(text, keywords)
		if score > maxScore {
			maxScore = score
			bestScene = scene
		}
	}

	// 至少命中 2 个关键词才认为有明确场景
	if maxScore < 2 {
		return "transition"
	}
	return bestScene
}

// inferEmotion 推断情感基调
func (s *IntentService) inferEmotion(text string) string {
	maxScore := 0
	bestEmotion := "neutral"

	for emotion, keywords := range emotionKeywords {
		score := countKeywordHits(text, keywords)
		if score > maxScore {
			maxScore = score
			bestEmotion = emotion
		}
	}

	if maxScore < 2 {
		return "neutral"
	}
	return bestEmotion
}

// inferDirection 推断情节走向
func (s *IntentService) inferDirection(text string) string {
	maxScore := 0
	bestDir := "transition"

	for dir, keywords := range directionKeywords {
		score := countKeywordHits(text, keywords)
		if score > maxScore {
			maxScore = score
			bestDir = dir
		}
	}
	return bestDir
}

// matchCharacters 匹配文本中出现的角色名
func (s *IntentService) matchCharacters(text string, novelID uint) []string {
	if s.knowledgeDAO == nil {
		return nil
	}

	// 查询小说的人物知识条目
	knowledges, err := s.knowledgeDAO.ListByNovelAndCategory(novelID, "character")
	if err != nil || len(knowledges) == 0 {
		return nil
	}

	var matched []string
	for _, k := range knowledges {
		// 用知识条目的 Title（角色名）做字符串匹配
		if k.Title != "" && strings.Contains(text, k.Title) {
			matched = append(matched, k.Title)
		}
	}

	// 去重（理论上 Title 不会重复，但防御性处理）
	seen := make(map[string]bool)
	var unique []string
	for _, name := range matched {
		if !seen[name] {
			seen[name] = true
			unique = append(unique, name)
		}
	}
	return unique
}

// countKeywordHits 统计文本中关键词命中次数
func countKeywordHits(text string, keywords []string) int {
	count := 0
	for _, kw := range keywords {
		if utf8.RuneCountInString(kw) == 1 {
			// 单字关键词：统计出现次数
			count += strings.Count(text, kw)
		} else {
			// 多字关键词：出现即计 1 次
			if strings.Contains(text, kw) {
				count++
			}
		}
	}
	return count
}
