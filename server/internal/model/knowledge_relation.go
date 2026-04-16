// server/internal/model/knowledge_relation.go
package model

import "time"

// 关系类型常量
const (
	RelMasterOf  = "master_of"
	RelEnemyOf   = "enemy_of"
	RelAllyOf    = "ally_of"
	RelFamilyOf  = "family_of"
	RelLocatedIn = "located_in"
	RelMemberOf  = "member_of"
	RelCreatedBy = "created_by"
	RelEvolvesTo = "evolves_to"
)

// ValidKnowledgeRelationTypes 知识图谱关系类型白名单
var ValidKnowledgeRelationTypes = map[string]bool{
	RelMasterOf:  true,
	RelEnemyOf:   true,
	RelAllyOf:    true,
	RelFamilyOf:  true,
	RelLocatedIn: true,
	RelMemberOf:  true,
	RelCreatedBy: true,
	RelEvolvesTo: true,
}

// KnowledgeRelation 知识图谱实体关系边
type KnowledgeRelation struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	NovelID      uint      `gorm:"index;not null" json:"novel_id"`
	FromEntityID uint      `gorm:"index;not null" json:"from_entity_id"` // 关联 NovelKnowledge.ID
	ToEntityID   uint      `gorm:"index;not null" json:"to_entity_id"`
	RelationType string    `gorm:"size:50;not null" json:"relation_type"`
	Description  string    `gorm:"size:500" json:"description"`
	ChapterRef   string    `gorm:"size:100" json:"chapter_ref"` // 来源章节
	CreatedAt    time.Time `json:"created_at"`
}
