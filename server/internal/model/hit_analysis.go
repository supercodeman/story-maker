// server/internal/model/hit_analysis.go
package model

import "time"

// HitAnalysis 爆款拆解记录
type HitAnalysis struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	PortfolioID uint      `gorm:"index" json:"portfolio_id"`
	Title       string    `gorm:"size:200;not null" json:"title"`        // 被拆解的小说标题
	Author      string    `gorm:"size:100" json:"author"`
	SourceText  string    `gorm:"type:longtext" json:"source_text"`      // 分析素材（完成后清除）
	Report      string    `gorm:"type:longtext" json:"report"`           // JSON: 结构化拆解报告
	WorkflowID  uint      `gorm:"index;default:0" json:"workflow_id"`
	Status      string    `gorm:"size:20;default:pending" json:"status"` // pending/running/completed/failed
	ModelName   string    `gorm:"size:50" json:"model_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
