// server/internal/model/portfolio.go
package model

import "time"

// Portfolio 作品集表，归属于某个工作空间
type Portfolio struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID uint      `gorm:"index;not null" json:"workspace_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CoverImage  string    `gorm:"size:500" json:"cover_image"`
	Status      string    `gorm:"size:20;not null;default:draft" json:"status"` // draft, published
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PortfolioStatus 作品集状态枚举
const (
	PortfolioStatusDraft     = "draft"
	PortfolioStatusPublished = "published"
)

// ValidPortfolioStatuses 合法的作品集状态白名单
var ValidPortfolioStatuses = map[string]bool{
	PortfolioStatusDraft:     true,
	PortfolioStatusPublished: true,
}
