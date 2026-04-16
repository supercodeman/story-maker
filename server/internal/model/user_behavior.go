// server/internal/model/user_behavior.go
package model

import "time"

// ========== 行为事件类型枚举 ==========

const (
	BehaviorAIAccept      = "ai_accept"      // 采纳 AI 结果
	BehaviorAIReject      = "ai_reject"      // 拒绝 AI 结果
	BehaviorAIModify      = "ai_modify"      // 采纳后修改（含 diff）
	BehaviorChapterSave   = "chapter_save"   // 章节保存（含 diff 摘要）
	BehaviorSuggestAccept = "suggest_accept"  // 采纳联想
	BehaviorSuggestReject = "suggest_reject"  // 忽略联想
)

// ValidBehaviorTypes 合法行为类型白名单
var ValidBehaviorTypes = map[string]bool{
	BehaviorAIAccept:      true,
	BehaviorAIReject:      true,
	BehaviorAIModify:      true,
	BehaviorChapterSave:   true,
	BehaviorSuggestAccept: true,
	BehaviorSuggestReject: true,
}

// UserBehaviorEvent 用户行为事件表（追加写入，定期归档）
type UserBehaviorEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_ube_user_novel;not null" json:"user_id"`
	NovelID   uint      `gorm:"index:idx_ube_user_novel;not null" json:"novel_id"`
	ChapterID uint      `gorm:"index" json:"chapter_id"`
	EventType string    `gorm:"size:30;not null;index" json:"event_type"`
	Payload   string    `gorm:"type:text" json:"payload"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// UserPreference 用户偏好摘要表（每个用户每本小说一条，定期更新）
type UserPreference struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex:idx_up_user_novel;not null" json:"user_id"`
	NovelID          uint      `gorm:"uniqueIndex:idx_up_user_novel;not null" json:"novel_id"`
	VocabProfile     string    `gorm:"type:text" json:"vocab_profile"`      // JSON: 高频词汇、偏好用词
	StyleProfile     string    `gorm:"type:text" json:"style_profile"`      // JSON: 句式偏好
	NarrativeProfile string    `gorm:"type:text" json:"narrative_profile"`  // JSON: 叙事偏好
	AIFeedbackProfile string   `gorm:"type:text" json:"ai_feedback_profile"` // JSON: AI 反馈偏好
	PromptSummary    string    `gorm:"type:text" json:"prompt_summary"`     // LLM 生成的偏好摘要
	EventCount       int       `gorm:"default:0" json:"event_count"`
	Version          int       `gorm:"default:1" json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedAt        time.Time `json:"created_at"`
}
