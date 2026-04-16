// server/internal/model/world_building.go
package model

import "time"

// NovelWorldSetting 世界观设定表，存储小说的世界观、时代背景、力量体系等
type NovelWorldSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NovelID   uint      `gorm:"not null;index:idx_ws_novel" json:"novel_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Category  string    `gorm:"size:30;not null;index:idx_ws_novel" json:"category"` // era_background / geography / power_system / social_rule / culture
	Title     string    `gorm:"size:200;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Version   int       `gorm:"default:1" json:"version"`
	Score     float64   `gorm:"default:0" json:"score"` // 反思审查最终评分
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 世界观设定分类枚举
const (
	WorldCategoryEraBackground = "era_background" // 时代背景
	WorldCategoryGeography     = "geography"       // 地理环境
	WorldCategoryPowerSystem   = "power_system"    // 力量体系
	WorldCategorySocialRule    = "social_rule"      // 社会规则
	WorldCategoryCulture       = "culture"          // 文化风俗
)

// ValidWorldCategories 合法的世界观分类白名单
var ValidWorldCategories = map[string]bool{
	WorldCategoryEraBackground: true,
	WorldCategoryGeography:     true,
	WorldCategoryPowerSystem:   true,
	WorldCategorySocialRule:    true,
	WorldCategoryCulture:       true,
}

// WorldCategoryLabel 世界观分类中文标签
var WorldCategoryLabel = map[string]string{
	WorldCategoryEraBackground: "时代背景",
	WorldCategoryGeography:     "地理环境",
	WorldCategoryPowerSystem:   "力量体系",
	WorldCategorySocialRule:    "社会规则",
	WorldCategoryCulture:       "文化风俗",
}

// NovelForeshadow 伏笔设定表，存储小说中预埋的伏笔线索
type NovelForeshadow struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	NovelID       uint      `gorm:"not null;index" json:"novel_id"`
	UserID        uint      `gorm:"not null;index" json:"user_id"`
	Title         string    `gorm:"size:200;not null" json:"title"`
	Description   string    `gorm:"type:text;not null" json:"description"`
	PlantChapter  string    `gorm:"size:200" json:"plant_chapter"`  // 预计埋设章节
	RevealChapter string    `gorm:"size:200" json:"reveal_chapter"` // 预计揭示章节
	Status        string    `gorm:"size:20;not null;default:planned" json:"status"` // planned / planted / revealed / abandoned
	Version       int       `gorm:"default:1" json:"version"`
	Score         float64   `gorm:"default:0" json:"score"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// 伏笔状态枚举
const (
	ForeshadowStatusPlanned   = "planned"   // 已规划
	ForeshadowStatusPlanted   = "planted"   // 已埋设
	ForeshadowStatusRevealed  = "revealed"  // 已揭示
	ForeshadowStatusAbandoned = "abandoned" // 已废弃
)

// NovelPlotOutline 剧情大纲表，存储按幕次组织的剧情结构
type NovelPlotOutline struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NovelID   uint      `gorm:"not null;index" json:"novel_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Act       int       `gorm:"not null" json:"act"`         // 幕次
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	Title     string    `gorm:"size:200;not null" json:"title"`
	Summary   string    `gorm:"type:text" json:"summary"`
	KeyEvents string    `gorm:"type:text" json:"key_events"` // JSON: 关键事件列表
	Version   int       `gorm:"default:1" json:"version"`
	Score     float64   `gorm:"default:0" json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReflectionLog 反思审查记录表，记录每轮生成-审查的过程
type ReflectionLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	NovelID    uint      `gorm:"not null;index" json:"novel_id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	Phase      string    `gorm:"size:30;not null;index" json:"phase"` // worldview / character / relation / foreshadow / plot
	Round      int       `gorm:"not null" json:"round"`
	Content    string    `gorm:"type:longtext" json:"content"`     // 该轮生成内容快照
	ReviewJSON string    `gorm:"type:text" json:"review_json"`     // 审查结果 JSON（各维度分数+意见）
	TotalScore float64   `gorm:"default:0" json:"total_score"`
	TaskID     uint      `gorm:"default:0" json:"task_id"`         // 关联的生成 AITask
	ReviewTask uint      `gorm:"default:0" json:"review_task_id"`  // 关联的审查 AITask
	CreatedAt  time.Time `json:"created_at"`
}

// 反思阶段枚举
const (
	ReflectionPhaseWorldview  = "worldview"
	ReflectionPhaseCharacter  = "character"
	ReflectionPhaseRelation   = "relation"
	ReflectionPhaseForeshadow = "foreshadow"
	ReflectionPhasePlot       = "plot"
)

// ReflectionConfig 反思循环配置
type ReflectionConfig struct {
	MaxRounds  int     `json:"max_rounds"`  // 最大轮次，默认 3
	Threshold  float64 `json:"threshold"`   // 达标阈值，默认 6.0
	AutoMode   bool    `json:"auto_mode"`   // true=全自动，false=半自动（每轮用户确认）
	ModelName  string  `json:"model_name"`  // 指定模型，空则用默认
}

// DefaultReflectionConfig 默认反思配置
func DefaultReflectionConfig() ReflectionConfig {
	return ReflectionConfig{
		MaxRounds: 3,
		Threshold: 6.0,
		AutoMode:  true,
	}
}

// ReviewResult 审查结果结构
type ReviewResult struct {
	Dimensions []ReviewDimension `json:"dimensions"`
	TotalScore float64           `json:"total_score"`
	Summary    string            `json:"summary"`    // 总体评价
	Suggestion string            `json:"suggestion"` // 修改建议
}

// ReviewDimension 评分维度
type ReviewDimension struct {
	Name    string  `json:"name"`    // 维度名称
	Score   float64 `json:"score"`   // 1-10 分
	Comment string  `json:"comment"` // 评语
}
