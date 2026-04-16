// server/internal/model/novel_overview.go
package model

import "time"

// NovelCharacterRelation 人物关系表，存储人物之间的关系边
type NovelCharacterRelation struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	NovelID         uint      `gorm:"index;not null" json:"novel_id"`
	FromKnowledgeID uint      `gorm:"not null" json:"from_knowledge_id"` // 关联 NovelKnowledge(character)
	ToKnowledgeID   uint      `gorm:"not null" json:"to_knowledge_id"`
	RelationType    string    `gorm:"size:30;not null" json:"relation_type"` // ally/enemy/mentor/lover/family/rival/custom
	Label           string    `gorm:"size:100" json:"label"`                // 自定义关系描述
	ChapterRef      string    `gorm:"size:500" json:"chapter_ref"`          // 关联章节 ID 列表
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// 关系类型枚举
const (
	RelationTypeAlly   = "ally"
	RelationTypeEnemy  = "enemy"
	RelationTypeMentor = "mentor"
	RelationTypeLover  = "lover"
	RelationTypeFamily = "family"
	RelationTypeRival  = "rival"
	RelationTypeCustom = "custom"
)

// ValidRelationTypes 合法的关系类型白名单
var ValidRelationTypes = map[string]bool{
	RelationTypeAlly:   true,
	RelationTypeEnemy:  true,
	RelationTypeMentor: true,
	RelationTypeLover:  true,
	RelationTypeFamily: true,
	RelationTypeRival:  true,
	RelationTypeCustom: true,
}

// RelationTypeLabel 关系类型中文标签
var RelationTypeLabel = map[string]string{
	RelationTypeAlly:   "盟友",
	RelationTypeEnemy:  "敌人",
	RelationTypeMentor: "师徒",
	RelationTypeLover:  "恋人",
	RelationTypeFamily: "亲属",
	RelationTypeRival:  "对手",
	RelationTypeCustom: "自定义",
}
