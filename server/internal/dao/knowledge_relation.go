// server/internal/dao/knowledge_relation.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// KnowledgeRelationDAO 知识图谱关系边数据访问层
type KnowledgeRelationDAO struct {
	db *gorm.DB
}

// NewKnowledgeRelationDAO 创建 KnowledgeRelationDAO 实例
func NewKnowledgeRelationDAO() *KnowledgeRelationDAO {
	return &KnowledgeRelationDAO{db: model.DB}
}

// Create 创建关系记录
func (d *KnowledgeRelationDAO) Create(rel *model.KnowledgeRelation) error {
	return d.db.Create(rel).Error
}

// BatchCreate 批量创建关系记录
func (d *KnowledgeRelationDAO) BatchCreate(rels []model.KnowledgeRelation) error {
	if len(rels) == 0 {
		return nil
	}
	return d.db.Create(&rels).Error
}

// ListByNovel 按小说 ID 查询所有关系
func (d *KnowledgeRelationDAO) ListByNovel(novelID uint) ([]model.KnowledgeRelation, error) {
	var rels []model.KnowledgeRelation
	err := d.db.Where("novel_id = ?", novelID).
		Order("created_at DESC").
		Find(&rels).Error
	return rels, err
}

// ListByEntity 按实体 ID 查询关联关系（depth=1 直接关联）
func (d *KnowledgeRelationDAO) ListByEntity(entityID uint, depth int) ([]model.KnowledgeRelation, error) {
	if depth <= 0 {
		depth = 1
	}

	// 第一跳：直接关联
	var rels []model.KnowledgeRelation
	err := d.db.Where("from_entity_id = ? OR to_entity_id = ?", entityID, entityID).
		Find(&rels).Error
	if err != nil || depth <= 1 {
		return rels, err
	}

	// 多跳查询：收集已发现的实体 ID，逐层扩展
	visited := map[uint]bool{entityID: true}
	allRels := append([]model.KnowledgeRelation{}, rels...)

	for hop := 1; hop < depth; hop++ {
		// 收集当前层新发现的实体 ID
		var newEntityIDs []uint
		for _, r := range rels {
			if !visited[r.FromEntityID] {
				visited[r.FromEntityID] = true
				newEntityIDs = append(newEntityIDs, r.FromEntityID)
			}
			if !visited[r.ToEntityID] {
				visited[r.ToEntityID] = true
				newEntityIDs = append(newEntityIDs, r.ToEntityID)
			}
		}
		if len(newEntityIDs) == 0 {
			break
		}

		// 查询下一跳
		var nextRels []model.KnowledgeRelation
		err = d.db.Where("from_entity_id IN ? OR to_entity_id IN ?", newEntityIDs, newEntityIDs).
			Find(&nextRels).Error
		if err != nil {
			break
		}
		rels = nextRels
		allRels = append(allRels, nextRels...)
	}

	return allRels, nil
}

// Delete 删除关系记录
func (d *KnowledgeRelationDAO) Delete(id uint) error {
	return d.db.Delete(&model.KnowledgeRelation{}, id).Error
}

// DeleteByNovel 删除小说的所有关系
func (d *KnowledgeRelationDAO) DeleteByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.KnowledgeRelation{}).Error
}
