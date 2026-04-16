// server/internal/model/knowledge.go
package model

import "time"

// NovelKnowledge 小说知识条目表，支持结构化知识管理
type NovelKnowledge struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	NovelID    uint      `gorm:"not null;index:idx_novel_category" json:"novel_id"`
	Category   string    `gorm:"size:30;not null;index:idx_novel_category" json:"category"` // character/worldview/plotline/foreshadow/style/custom
	Title      string    `gorm:"size:200;not null" json:"title"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	Tags       string    `gorm:"size:500" json:"tags"`        // 逗号分隔的标签，用于关键词匹配
	ChapterRef string    `gorm:"size:500" json:"chapter_ref"` // 关联章节 ID 列表，逗号分隔
	Priority   int       `gorm:"default:0;index" json:"priority"`
	Status     string    `gorm:"size:20;not null;default:confirmed" json:"status"` // confirmed/pending
	SortOrder  int       `gorm:"default:0" json:"sort_order"`                     // 情节线排序
	Resolved   bool      `gorm:"default:false" json:"resolved"`                   // 伏笔回收标记
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// 知识条目类别枚举
const (
	KnowledgeCategoryCharacter  = "character"  // 人物档案
	KnowledgeCategoryWorldview  = "worldview"  // 世界观设定
	KnowledgeCategoryPlotline   = "plotline"   // 主线/支线剧情
	KnowledgeCategoryForeshadow = "foreshadow" // 伏笔追踪
	KnowledgeCategoryStyle      = "style"      // 文风规范
	KnowledgeCategoryCustom     = "custom"     // 自定义
)

// ValidKnowledgeCategories 合法的知识类别白名单
var ValidKnowledgeCategories = map[string]bool{
	KnowledgeCategoryCharacter:  true,
	KnowledgeCategoryWorldview:  true,
	KnowledgeCategoryPlotline:   true,
	KnowledgeCategoryForeshadow: true,
	KnowledgeCategoryStyle:      true,
	KnowledgeCategoryCustom:     true,
}

// 知识条目状态枚举
const (
	KnowledgeStatusConfirmed = "confirmed" // 用户确认
	KnowledgeStatusPending   = "pending"   // AI 提取待审核
)

// ValidKnowledgeStatuses 合法的知识状态白名单
var ValidKnowledgeStatuses = map[string]bool{
	KnowledgeStatusConfirmed: true,
	KnowledgeStatusPending:   true,
}

// KnowledgeCategoryLabel 类别中文标签（用于 Prompt 注入时的分组标题）
var KnowledgeCategoryLabel = map[string]string{
	KnowledgeCategoryCharacter:  "人物档案",
	KnowledgeCategoryWorldview:  "世界观设定",
	KnowledgeCategoryPlotline:   "剧情线索",
	KnowledgeCategoryForeshadow: "伏笔追踪",
	KnowledgeCategoryStyle:      "文风规范",
	KnowledgeCategoryCustom:     "自定义设定",
}
