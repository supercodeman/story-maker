// server/internal/model/writing_style.go
package model

import "time"

// WritingStyle 写作风格配置，与 Novel 一对一
type WritingStyle struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	NovelID           uint      `gorm:"uniqueIndex;not null" json:"novel_id"`
	NarrativeVoice    string    `gorm:"size:30;not null;default:third_limited" json:"narrative_voice"`
	Tone              string    `gorm:"size:30;not null;default:neutral" json:"tone"`
	LanguageLevel     string    `gorm:"size:30;not null;default:standard" json:"language_level"`
	ReferenceAuthors  string    `gorm:"size:500" json:"reference_authors"`
	ForbiddenPatterns string    `gorm:"type:text" json:"forbidden_patterns"`
	CustomRules       string    `gorm:"type:text" json:"custom_rules"`
	BoundUserStyleID *uint     `gorm:"index" json:"bound_user_style_id"` // 绑定的用户风格ID，nil 表示未绑定
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ScenePreset 场景预设，与 Novel 一对多
type ScenePreset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NovelID   uint      `gorm:"index;not null" json:"novel_id"`
	SceneType string    `gorm:"size:30;not null" json:"scene_type"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Rules     string    `gorm:"type:text;not null" json:"rules"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 叙事视角枚举
const (
	NarrativeFirst         = "first"           // 第一人称
	NarrativeThirdLimited  = "third_limited"   // 第三人称有限
	NarrativeThirdOmni     = "third_omniscient" // 第三人称全知
	NarrativeMultiPOV      = "multi_pov"       // 多视角
)

// ValidNarrativeVoices 合法叙事视角白名单
var ValidNarrativeVoices = map[string]string{
	NarrativeFirst:        "第一人称",
	NarrativeThirdLimited: "第三人称有限",
	NarrativeThirdOmni:    "第三人称全知",
	NarrativeMultiPOV:     "多视角",
}

// 文风调性枚举
const (
	ToneSerious  = "serious"  // 严肃克制
	ToneHumorous = "humorous" // 幽默诙谐
	ToneLyrical  = "lyrical"  // 抒情优美
	ToneSharp    = "sharp"    // 犀利冷峻
	ToneWarm     = "warm"     // 温暖治愈
	ToneNeutral  = "neutral"  // 中性
)

// ValidTones 合法文风调性白名单
var ValidTones = map[string]string{
	ToneSerious:  "严肃克制",
	ToneHumorous: "幽默诙谐",
	ToneLyrical:  "抒情优美",
	ToneSharp:    "犀利冷峻",
	ToneWarm:     "温暖治愈",
	ToneNeutral:  "中性",
}

// 语言水平枚举
const (
	LangLiterary    = "literary"    // 文学性
	LangStandard    = "standard"    // 标准
	LangColloquial  = "colloquial"  // 口语化
	LangWebNovel    = "web_novel"   // 网文风
)

// ValidLanguageLevels 合法语言水平白名单
var ValidLanguageLevels = map[string]string{
	LangLiterary:   "文学性",
	LangStandard:   "标准",
	LangColloquial: "口语化",
	LangWebNovel:   "网文风",
}

// 场景类型枚举
const (
	SceneBattle     = "battle"     // 战斗
	SceneDialogue   = "dialogue"   // 对话
	ScenePsychology = "psychology" // 心理描写
	SceneEnvironment = "environment" // 环境描写
	SceneFlashback  = "flashback"  // 回忆
	SceneDaily      = "daily"      // 日常
)

// ValidSceneTypes 合法场景类型白名单
var ValidSceneTypes = map[string]string{
	SceneBattle:      "战斗",
	SceneDialogue:    "对话",
	ScenePsychology:  "心理描写",
	SceneEnvironment: "环境描写",
	SceneFlashback:   "回忆",
	SceneDaily:       "日常",
}
