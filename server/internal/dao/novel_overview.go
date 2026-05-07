// server/internal/dao/novel_overview.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// OverviewDAO 总览数据访问层
type OverviewDAO struct {
	db *gorm.DB
}

// NewOverviewDAO 创建 OverviewDAO 实例
func NewOverviewDAO() *OverviewDAO {
	return &OverviewDAO{db: model.DB}
}

// ========== 人物关系 CRUD ==========

// CreateRelation 创建人物关系
func (d *OverviewDAO) CreateRelation(r *model.NovelCharacterRelation) error {
	return d.db.Create(r).Error
}

// UpdateRelation 更新人物关系
func (d *OverviewDAO) UpdateRelation(r *model.NovelCharacterRelation) error {
	return d.db.Save(r).Error
}

// DeleteRelation 删除人物关系
func (d *OverviewDAO) DeleteRelation(id uint) error {
	return d.db.Delete(&model.NovelCharacterRelation{}, id).Error
}

// GetRelation 根据 ID 获取人物关系
func (d *OverviewDAO) GetRelation(id uint) (*model.NovelCharacterRelation, error) {
	var r model.NovelCharacterRelation
	err := d.db.First(&r, id).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRelationsByNovel 获取小说的所有人物关系
func (d *OverviewDAO) ListRelationsByNovel(novelID uint) ([]model.NovelCharacterRelation, error) {
	var items []model.NovelCharacterRelation
	err := d.db.Where("novel_id = ?", novelID).
		Order("created_at ASC").
		Find(&items).Error
	return items, err
}

// DeleteRelationsByNovel 删除小说的所有人物关系（级联删除）
func (d *OverviewDAO) DeleteRelationsByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.NovelCharacterRelation{}).Error
}

// ========== 聚合查询 ==========

// OverviewData 总览聚合数据
type OverviewData struct {
	Plotlines   []model.NovelKnowledge         `json:"plotlines"`
	Characters  []model.NovelKnowledge         `json:"characters"`
	Foreshadows []model.NovelKnowledge         `json:"foreshadows"`
	Relations   []model.NovelCharacterRelation  `json:"relations"`
}

// GetOverviewData 一次查询返回总览所需的全部元数据
func (d *OverviewDAO) GetOverviewData(novelID uint) (*OverviewData, error) {
	data := &OverviewData{}

	// 查询情节线（只返回已确认的条目）
	if err := d.db.Where("novel_id = ? AND category = ? AND status = ?", novelID, model.KnowledgeCategoryPlotline, model.KnowledgeStatusConfirmed).
		Order("sort_order ASC, priority DESC, created_at ASC").
		Find(&data.Plotlines).Error; err != nil {
		return nil, err
	}

	// 查询人物
	if err := d.db.Where("novel_id = ? AND category = ? AND status = ?", novelID, model.KnowledgeCategoryCharacter, model.KnowledgeStatusConfirmed).
		Order("priority DESC, created_at ASC").
		Find(&data.Characters).Error; err != nil {
		return nil, err
	}

	// 查询伏笔
	if err := d.db.Where("novel_id = ? AND category = ? AND status = ?", novelID, model.KnowledgeCategoryForeshadow, model.KnowledgeStatusConfirmed).
		Order("sort_order ASC, priority DESC, created_at ASC").
		Find(&data.Foreshadows).Error; err != nil {
		return nil, err
	}

	// 查询人物关系
	var err error
	data.Relations, err = d.ListRelationsByNovel(novelID)
	if err != nil {
		return nil, err
	}

	return data, nil
}
