// server/internal/model/plot_structure.go
package model

import "time"

// PlotStructureTemplate 剧情结构模板
type PlotStructureTemplate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`           // 三幕式、英雄之旅...
	Category    string    `gorm:"size:30;not null" json:"category"`        // classic/web_novel/suspense/romance/custom
	Description string    `gorm:"type:text" json:"description"`
	Structure   string    `gorm:"type:text;not null" json:"structure"`     // JSON: [{phase, name, description, ratio, beats}]
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`
	UserID      uint      `gorm:"index;default:0" json:"user_id"`         // 0=系统预置
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
