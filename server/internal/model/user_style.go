// server/internal/model/user_style.go
package model

import "time"

// UserStyle 用户风格模板库（用户级）
type UserStyle struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"index;not null" json:"user_id"`
	Name              string    `gorm:"size:100;not null" json:"name"`
	Description       string    `gorm:"size:500" json:"description"`
	NarrativeVoice    string    `gorm:"size:30;not null;default:third_limited" json:"narrative_voice"`
	Tone              string    `gorm:"size:30;not null;default:neutral" json:"tone"`
	LanguageLevel     string    `gorm:"size:30;not null;default:standard" json:"language_level"`
	ReferenceAuthors  string    `gorm:"size:500" json:"reference_authors"`
	ForbiddenPatterns string    `gorm:"type:text" json:"forbidden_patterns"`
	CustomRules       string    `gorm:"type:text" json:"custom_rules"`
	CustomPrompt      string    `gorm:"type:text" json:"custom_prompt"`
	IsAIGenerated     bool      `gorm:"default:false" json:"is_ai_generated"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
