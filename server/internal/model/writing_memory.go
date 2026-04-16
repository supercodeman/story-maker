// server/internal/model/writing_memory.go
package model

import "time"

// WritingMemory 写作记忆主表
type WritingMemory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	Category    string    `gorm:"size:30;not null;index:idx_mem_cat_status" json:"category"`
	Title       string    `gorm:"size:200;not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	SourceText  string    `gorm:"type:mediumtext" json:"-"`
	SampleHash  string    `gorm:"size:64;uniqueIndex" json:"-"`
	SampleLen   int       `gorm:"default:0" json:"sample_len"`
	Features    string    `gorm:"type:text" json:"features"`
	PromptTpl   string    `gorm:"type:text" json:"prompt_tpl,omitempty"`
	AnchorTexts string    `gorm:"type:text" json:"anchor_texts,omitempty"`
	PreviewText string    `gorm:"size:500" json:"preview_text"`
	Version     int       `gorm:"default:1" json:"version"`
	Quality     float64   `gorm:"default:0" json:"quality"`
	QualityDetail string  `gorm:"type:text" json:"quality_detail,omitempty"`
	QualityGrade  string  `gorm:"size:5;default:''" json:"quality_grade"`
	Status      string    `gorm:"size:20;default:draft;index:idx_mem_cat_status" json:"status"`
	IsPublic    bool      `gorm:"default:false;index" json:"is_public"`
	Price       int       `gorm:"default:0" json:"price"`
	SalesCount  int       `gorm:"default:0" json:"sales_count"`
	AvgRating   float64   `gorm:"default:0" json:"avg_rating"`
	RatingCount int       `gorm:"default:0" json:"rating_count"`
	Tags              string    `gorm:"size:500" json:"tags"`
	ExtractWorkflowID uint      `gorm:"default:0" json:"extract_workflow_id"`
	ExtractStatus     string    `gorm:"size:20;default:pending" json:"extract_status"` // pending/running/completed/failed
	ExtractError      string    `gorm:"size:500" json:"extract_error,omitempty"`
	ReviewWorkflowID  uint      `gorm:"default:0" json:"review_workflow_id"`
	ReviewReason      string    `gorm:"size:500" json:"review_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WritingMemoryVersion 记忆版本表
type WritingMemoryVersion struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MemoryID  uint      `gorm:"index;not null" json:"memory_id"`
	Version   int       `gorm:"not null" json:"version"`
	Features  string    `gorm:"type:text" json:"features"`
	PromptTpl string    `gorm:"type:text" json:"prompt_tpl"`
	ChangeLog string    `gorm:"size:500" json:"change_log"`
	CreatedAt time.Time `json:"created_at"`
}

// MemoryEmbedding 记忆向量表
type MemoryEmbedding struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MemoryID  uint      `gorm:"index;not null" json:"memory_id"`
	ChunkIdx  int       `gorm:"not null" json:"chunk_idx"`
	ChunkText string    `gorm:"type:text" json:"chunk_text"`
	Vector    string    `gorm:"type:mediumtext" json:"-"`
	Dimension int       `gorm:"not null" json:"dimension"`
	CreatedAt time.Time `json:"created_at"`
}

// NovelMemoryBinding 小说-记忆绑定表（每个小说每个类别最多绑定一个记忆）
type NovelMemoryBinding struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NovelID   uint      `gorm:"uniqueIndex:idx_novel_category;not null" json:"novel_id"`
	Category  string    `gorm:"size:30;uniqueIndex:idx_novel_category;not null" json:"category"`
	MemoryID  uint      `gorm:"index;not null" json:"memory_id"`
	CreatedAt time.Time `json:"created_at"`
}

// 记忆类别枚举
const (
	MemoryCategoryStyle          = "style"
	MemoryCategoryCharacter      = "character"
	MemoryCategoryWorldview      = "worldview"
	MemoryCategoryPlotPreference = "plot_preference"
)

// ValidMemoryCategories 合法记忆类别白名单
var ValidMemoryCategories = map[string]string{
	MemoryCategoryStyle:          "写作风格",
	MemoryCategoryCharacter:      "人设模板",
	MemoryCategoryWorldview:      "世界观框架",
	MemoryCategoryPlotPreference: "剧情偏好",
}

// 记忆状态枚举
const (
	MemoryStatusDraft     = "draft"
	MemoryStatusReviewing = "reviewing"
	MemoryStatusPublished = "published"
	MemoryStatusRejected  = "rejected"
	MemoryStatusArchived  = "archived"
)

// ValidMemoryStatuses 合法记忆状态白名单
var ValidMemoryStatuses = map[string]string{
	MemoryStatusDraft:     "草稿",
	MemoryStatusReviewing: "审核中",
	MemoryStatusPublished: "已上架",
	MemoryStatusRejected:  "已拒绝",
	MemoryStatusArchived:  "已下架",
}

// ========== 质量评级体系 ==========

// QualityDetail 多维质量评分
type QualityDetail struct {
	Consistency     int    `json:"consistency"`     // 风格一致性 0-100
	Reproducibility int    `json:"reproducibility"` // 可复现性 0-100
	Uniqueness      int    `json:"uniqueness"`      // 独特性 0-100
	Practicality    int    `json:"practicality"`    // 实用性 0-100
	PreviewText     string `json:"preview_text"`    // 生成的预览片段
	Evaluation      string `json:"evaluation"`      // 评价说明
}

// 评级常量
const (
	QualityGradeS = "S" // 90+
	QualityGradeA = "A" // 80-89
	QualityGradeB = "B" // 70-79
	QualityGradeC = "C" // 60-69
	QualityGradeD = "D" // <60
)

// CalcQualityGrade 根据多维评分计算综合评级
func CalcQualityGrade(d *QualityDetail) (float64, string) {
	avg := float64(d.Consistency+d.Reproducibility+d.Uniqueness+d.Practicality) / 4.0
	switch {
	case avg >= 90:
		return avg, QualityGradeS
	case avg >= 80:
		return avg, QualityGradeA
	case avg >= 70:
		return avg, QualityGradeB
	case avg >= 60:
		return avg, QualityGradeC
	default:
		return avg, QualityGradeD
	}
}

// ========== 风格子维度增强 ==========

// StyleFeatures style 类记忆的细化特征结构
type StyleFeatures struct {
	Tone              StyleDimension `json:"tone"`               // 文风：调性、情感基调
	Rhythm            StyleDimension `json:"rhythm"`             // 句式：节奏、长短句偏好
	Vocabulary        StyleDimension `json:"vocabulary"`         // 语感：用词特点、修辞手法
	DialogueStyle     StyleDimension `json:"dialogue_style"`     // 对话：角色对白风格
	ForbiddenPatterns []string       `json:"forbidden_patterns"` // 应避免的表达
	ReferenceStyle    string         `json:"reference_style"`    // 最接近的知名作家风格
}

// StyleDimension 单个子维度
type StyleDimension struct {
	Description string   `json:"description"` // 维度描述
	Score       int      `json:"score"`       // 维度评分 0-100
	Examples    []string `json:"examples"`    // 从样本中提取的典型句子
	PromptPart  string   `json:"prompt_part"` // 该维度对应的写作指令片段
}
