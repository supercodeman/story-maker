// server/internal/model/genre.go
package model

import "time"

// Genre 赛道分类表（支持多级分类）
type Genre struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ParentID  uint      `gorm:"default:0;index:idx_parent" json:"parent_id"` // 父赛道 ID，0 为顶级
	Name      string    `gorm:"size:50;not null" json:"name"`
	Slug      string    `gorm:"size:50;not null;uniqueIndex" json:"slug"` // URL 友好标识
	Icon      string    `gorm:"size:50;default:''" json:"icon"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// GenreTree 赛道树节点（用于 API 返回）
type GenreTree struct {
	Genre
	Children []GenreTree `json:"children,omitempty"`
}

// MemoryGenre 记忆-赛道关联表（多对多）
type MemoryGenre struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	MemoryID uint `gorm:"not null;uniqueIndex:idx_mem_genre" json:"memory_id"`
	GenreID  uint `gorm:"not null;uniqueIndex:idx_mem_genre;index:idx_genre" json:"genre_id"`
}
