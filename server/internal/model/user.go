// server/internal/model/user.go
package model

import "time"

// User 用户模型
type User struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Username        string     `gorm:"uniqueIndex;size:50" json:"username"`
	Email           string     `gorm:"uniqueIndex;size:100" json:"email"`
	PasswordHash    string     `gorm:"size:255" json:"-"`
	Role            string     `gorm:"size:20;default:creator" json:"role"` // admin, creator, viewer
	WriterLevel     string     `gorm:"size:20;default:beginner" json:"writer_level"`  // beginner / advanced
	LevelUnlockAt   *time.Time `gorm:"" json:"level_unlock_at,omitempty"`             // 解锁时间
	LevelSource     string     `gorm:"size:20;default:''" json:"level_source"`        // growth / purchase
	TotalWordCount  int64      `gorm:"default:0" json:"total_word_count"`             // 累计创作字数
	TotalChapters   int        `gorm:"default:0" json:"total_chapters"`               // 累计章节数
	CompletedNovels int        `gorm:"default:0" json:"completed_novels"`             // 完本数
	ViewMode        string     `gorm:"size:20;default:simple" json:"view_mode"`       // simple / advanced
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// 写手等级常量
const (
	WriterLevelBeginner = "beginner"
	WriterLevelAdvanced = "advanced"
	LevelSourceGrowth   = "growth"
	LevelSourcePurchase = "purchase"
	LevelSourceAdmin    = "admin"
	ViewModeSimple      = "simple"
	ViewModeAdvanced    = "advanced"
)

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
