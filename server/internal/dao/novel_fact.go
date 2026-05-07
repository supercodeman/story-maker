// server/internal/dao/novel_fact.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// NovelFactDAO 小说动态记忆事实数据访问层
type NovelFactDAO struct {
	db *gorm.DB
}

// NewNovelFactDAO 创建 NovelFactDAO 实例
func NewNovelFactDAO() *NovelFactDAO {
	return &NovelFactDAO{db: model.DB}
}

// Create 创建事实记录
func (d *NovelFactDAO) Create(fact *model.NovelMemoryFact) error {
	return d.db.Create(fact).Error
}

// BatchCreate 批量创建事实记录
func (d *NovelFactDAO) BatchCreate(facts []*model.NovelMemoryFact) error {
	if len(facts) == 0 {
		return nil
	}
	return d.db.Create(&facts).Error
}

// UpdateMilvusID 更新事实的 Milvus 向量 ID
func (d *NovelFactDAO) UpdateMilvusID(factID uint, milvusID int64) error {
	return d.db.Model(&model.NovelMemoryFact{}).
		Where("id = ?", factID).
		Update("milvus_id", milvusID).Error
}

// FindByNovelAndTitle 按小说ID、事实类型、标题查找未被取代的事实
func (d *NovelFactDAO) FindByNovelAndTitle(novelID uint, factType, title string) (*model.NovelMemoryFact, error) {
	var fact model.NovelMemoryFact
	err := d.db.Where("novel_id = ? AND fact_type = ? AND title = ? AND is_superseded = false",
		novelID, factType, title).First(&fact).Error
	if err != nil {
		return nil, err
	}
	return &fact, nil
}

// Supersede 标记旧事实为已取代，并设置取代者 ID
func (d *NovelFactDAO) Supersede(oldFactID, newFactID uint) error {
	return d.db.Model(&model.NovelMemoryFact{}).
		Where("id = ?", oldFactID).
		Updates(map[string]interface{}{
			"is_superseded": true,
			"superseded_by": newFactID,
		}).Error
}

// GetByIDs 根据 ID 列表批量获取事实
func (d *NovelFactDAO) GetByIDs(ids []uint) ([]model.NovelMemoryFact, error) {
	var facts []model.NovelMemoryFact
	if len(ids) == 0 {
		return facts, nil
	}
	err := d.db.Where("id IN ?", ids).Find(&facts).Error
	return facts, err
}

// CountByNovel 统计小说的事实数量
func (d *NovelFactDAO) CountByNovel(novelID uint) (int64, error) {
	var count int64
	err := d.db.Model(&model.NovelMemoryFact{}).
		Where("novel_id = ? AND is_superseded = false", novelID).
		Count(&count).Error
	return count, err
}

// ListByNovelActive 获取小说所有未被取代的事实
func (d *NovelFactDAO) ListByNovelActive(novelID uint) ([]model.NovelMemoryFact, error) {
	var facts []model.NovelMemoryFact
	err := d.db.Where("novel_id = ? AND is_superseded = false", novelID).
		Order("created_at DESC").
		Find(&facts).Error
	return facts, err
}

// GetByID 按 ID 获取单条事实
func (d *NovelFactDAO) GetByID(id uint) (*model.NovelMemoryFact, error) {
	var fact model.NovelMemoryFact
	if err := d.db.Where("id = ?", id).First(&fact).Error; err != nil {
		return nil, err
	}
	return &fact, nil
}

// ListByNovelActiveWithFilter 获取小说未被取代的事实，支持按 fact_type 过滤
func (d *NovelFactDAO) ListByNovelActiveWithFilter(novelID uint, factType string) ([]model.NovelMemoryFact, error) {
	var facts []model.NovelMemoryFact
	query := d.db.Where("novel_id = ? AND is_superseded = false", novelID)
	if factType != "" {
		query = query.Where("fact_type = ?", factType)
	}
	err := query.Order("created_at DESC").Find(&facts).Error
	return facts, err
}

// DeleteByNovel 删除小说的所有事实（用于冷启动重置）
func (d *NovelFactDAO) DeleteByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.NovelMemoryFact{}).Error
}
