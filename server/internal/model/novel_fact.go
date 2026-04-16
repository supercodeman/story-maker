// server/internal/model/novel_fact.go
package model

import "time"

// NovelMemoryFact 小说动态记忆事实表，存储从章节中自动提取的结构化事实
type NovelMemoryFact struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	NovelID      uint      `gorm:"not null;index:idx_fact_novel" json:"novel_id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	ChapterID    uint      `gorm:"index" json:"chapter_id"`                          // 来源章节
	FactType     string    `gorm:"size:30;not null;index:idx_fact_novel" json:"fact_type"`
	Title        string    `gorm:"size:200;not null" json:"title"`                   // 事实标题（如人物名）
	Content      string    `gorm:"type:text;not null" json:"content"`                // 事实描述
	SourceText   string    `gorm:"type:text" json:"-"`                               // 原文片段
	MilvusID     int64     `gorm:"default:0" json:"-"`                               // Milvus 向量 ID
	IsSuperseded bool      `gorm:"default:false;index" json:"-"`                     // 被更新版本取代
	SupersededBy uint      `gorm:"default:0" json:"-"`                               // 取代者 ID
	Version      int       `gorm:"default:1" json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 事实类型枚举
const (
	FactTypeCharacterTrait    = "character_trait"     // 人物特征/性格变化
	FactTypePlotEvent         = "plot_event"          // 关键剧情事件
	FactTypeForeshadow        = "foreshadow"          // 伏笔埋设/回收
	FactTypeWorldviewRule     = "worldview_rule"      // 世界观规则
	FactTypeRelationshipChange = "relationship_change" // 人物关系变化
)

// ValidFactTypes 合法的事实类型白名单
var ValidFactTypes = map[string]bool{
	FactTypeCharacterTrait:     true,
	FactTypePlotEvent:          true,
	FactTypeForeshadow:         true,
	FactTypeWorldviewRule:      true,
	FactTypeRelationshipChange: true,
}

// FactTypeLabel 事实类型中文标签（用于 Prompt 注入时的分组标题）
var FactTypeLabel = map[string]string{
	FactTypeCharacterTrait:     "人物特征",
	FactTypePlotEvent:          "剧情事件",
	FactTypeForeshadow:         "伏笔追踪",
	FactTypeWorldviewRule:      "世界观规则",
	FactTypeRelationshipChange: "人物关系",
}
