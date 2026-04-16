// server/internal/model/chapter_summary.go
package model

import "time"

// ChapterSummaryNode 递归摘要树节点
// Level 0 = 章节原始摘要, Level 1 = 5章聚合, Level 2 = 25章聚合...
type ChapterSummaryNode struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	NovelID      uint      `gorm:"index;not null" json:"novel_id"`
	Level        int       `gorm:"not null" json:"level"`
	StartChapter int       `gorm:"not null" json:"start_chapter"` // 覆盖的起始章节 sort_order
	EndChapter   int       `gorm:"not null" json:"end_chapter"`   // 覆盖的结束章节 sort_order
	Summary      string    `gorm:"type:text;not null" json:"summary"`
	ParentID     *uint     `gorm:"index" json:"parent_id"` // 上层摘要节点 ID
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
