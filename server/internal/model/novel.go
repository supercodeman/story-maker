// server/internal/model/novel.go
package model

import "time"

// Novel 小说表，归属于某个作品集
type Novel struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PortfolioID  uint      `gorm:"index;not null" json:"portfolio_id"`
	Title        string    `gorm:"size:200;not null" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	Status       string    `gorm:"size:20;not null;default:draft" json:"status"`   // draft, writing, completed
	Source       string    `gorm:"size:20;not null;default:manual" json:"source"` // manual, butler, outline
	ChapterCount int       `gorm:"default:0" json:"chapter_count"`
	WordCount    int       `gorm:"default:0" json:"word_count"`
	TokenBudget  int       `gorm:"default:0" json:"token_budget"` // Token 预算上限，0 表示不限制
	TokenUsed    int       `gorm:"default:0" json:"token_used"`   // 缓存的已用 token 数
	// 管家创作各步骤的最终结果（仅 source=butler 时使用）
	ButlerTopic      string `gorm:"type:text" json:"butler_topic,omitempty"`
	ButlerStoryline  string `gorm:"type:text" json:"butler_storyline,omitempty"`
	ButlerCharacters string `gorm:"type:text" json:"butler_characters,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Novel 状态枚举
const (
	NovelStatusDraft     = "draft"
	NovelStatusWriting   = "writing"
	NovelStatusCompleted = "completed"
)

// Novel 来源枚举
const (
	NovelSourceManual  = "manual"
	NovelSourceButler  = "butler"
	NovelSourceOutline = "outline"
)

// ValidNovelStatuses 合法的小说状态白名单
var ValidNovelStatuses = map[string]bool{
	NovelStatusDraft:     true,
	NovelStatusWriting:   true,
	NovelStatusCompleted: true,
}

// Chapter 章节表，归属于某部小说
type Chapter struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	NovelID        uint      `gorm:"index;not null" json:"novel_id"`
	Title          string    `gorm:"size:200;not null" json:"title"`
	SortOrder      int       `gorm:"index;not null;default:0" json:"sort_order"`
	Summary        string    `gorm:"type:text" json:"summary"`
	Content        string    `gorm:"type:longtext" json:"content"`
	WordCount      int       `gorm:"default:0" json:"word_count"`
	Status         string    `gorm:"size:20;not null;default:draft" json:"status"` // draft, polished, final
	CurrentVersion int       `gorm:"default:1" json:"current_version"`
	ScenePresetID  *uint     `json:"scene_preset_id"` // 绑定的默认场景预设，可选
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Chapter 状态枚举
const (
	ChapterStatusDraft    = "draft"
	ChapterStatusPolished = "polished"
	ChapterStatusFinal    = "final"
)

// ChapterVersion 章节版本历史表
type ChapterVersion struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ChapterID uint      `gorm:"index;not null" json:"chapter_id"`
	Version   int       `gorm:"not null" json:"version"`
	Content   string    `gorm:"type:longtext" json:"content"`
	Summary   string    `gorm:"type:text" json:"summary"`
	Source    string    `gorm:"size:30;not null" json:"source"` // manual, ai_outline, ai_polish, ai_expand, ai_continue
	TaskID    *uint     `json:"task_id"`                        // 关联 AITask，手动编辑时为 nil
	WordCount int       `gorm:"default:0" json:"word_count"`
	CreatedAt time.Time `json:"created_at"`
}

// ChapterVersion 来源枚举
const (
	VersionSourceManual     = "manual"
	VersionSourceAIOutline  = "ai_outline"
	VersionSourceAIPolish   = "ai_polish"
	VersionSourceAIExpand   = "ai_expand"
	VersionSourceAIContinue = "ai_continue"
)
