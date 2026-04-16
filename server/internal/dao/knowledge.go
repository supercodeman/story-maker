// server/internal/dao/knowledge.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// KnowledgeDAO 知识库数据访问层
type KnowledgeDAO struct {
	db *gorm.DB
}

// NewKnowledgeDAO 创建 KnowledgeDAO 实例
func NewKnowledgeDAO() *KnowledgeDAO {
	return &KnowledgeDAO{db: model.DB}
}

// Create 创建知识条目
func (d *KnowledgeDAO) Create(k *model.NovelKnowledge) error {
	return d.db.Create(k).Error
}

// BatchCreate 批量创建知识条目
func (d *KnowledgeDAO) BatchCreate(items []model.NovelKnowledge) error {
	if len(items) == 0 {
		return nil
	}
	return d.db.Create(&items).Error
}

// Get 根据 ID 获取知识条目
func (d *KnowledgeDAO) Get(id uint) (*model.NovelKnowledge, error) {
	var k model.NovelKnowledge
	err := d.db.First(&k, id).Error
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// Update 更新知识条目
func (d *KnowledgeDAO) Update(k *model.NovelKnowledge) error {
	return d.db.Save(k).Error
}

// Delete 删除知识条目
func (d *KnowledgeDAO) Delete(id uint) error {
	return d.db.Delete(&model.NovelKnowledge{}, id).Error
}

// ListByNovel 获取小说的所有知识条目（按类别+优先级排序）
func (d *KnowledgeDAO) ListByNovel(novelID uint) ([]model.NovelKnowledge, error) {
	var items []model.NovelKnowledge
	err := d.db.Where("novel_id = ?", novelID).
		Order("category ASC, priority DESC, created_at ASC").
		Find(&items).Error
	return items, err
}

// ListByNovelAndCategory 按类别筛选知识条目
func (d *KnowledgeDAO) ListByNovelAndCategory(novelID uint, category string) ([]model.NovelKnowledge, error) {
	var items []model.NovelKnowledge
	err := d.db.Where("novel_id = ? AND category = ?", novelID, category).
		Order("priority DESC, created_at ASC").
		Find(&items).Error
	return items, err
}

// ListByNovelAndStatus 按状态筛选知识条目
func (d *KnowledgeDAO) ListByNovelAndStatus(novelID uint, status string) ([]model.NovelKnowledge, error) {
	var items []model.NovelKnowledge
	err := d.db.Where("novel_id = ? AND status = ?", novelID, status).
		Order("category ASC, priority DESC, created_at ASC").
		Find(&items).Error
	return items, err
}

// ListConfirmedByNovel 获取小说的所有已确认知识条目（用于 AI 上下文注入）
func (d *KnowledgeDAO) ListConfirmedByNovel(novelID uint) ([]model.NovelKnowledge, error) {
	var items []model.NovelKnowledge
	err := d.db.Where("novel_id = ? AND status = ?", novelID, model.KnowledgeStatusConfirmed).
		Order("category ASC, priority DESC").
		Find(&items).Error
	return items, err
}

// CountByNovel 统计小说的知识条目数
func (d *KnowledgeDAO) CountByNovel(novelID uint) (int64, error) {
	var count int64
	err := d.db.Model(&model.NovelKnowledge{}).Where("novel_id = ?", novelID).Count(&count).Error
	return count, err
}

// DeleteByNovel 删除小说的所有知识条目（级联删除用）
func (d *KnowledgeDAO) DeleteByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.NovelKnowledge{}).Error
}

// DeleteByNovelAndCategory 删除小说指定类别的知识条目
func (d *KnowledgeDAO) DeleteByNovelAndCategory(novelID uint, category string) error {
	return d.db.Where("novel_id = ? AND category = ?", novelID, category).Delete(&model.NovelKnowledge{}).Error
}

// ConfirmPending 将待审核条目确认
func (d *KnowledgeDAO) ConfirmPending(id uint) error {
	return d.db.Model(&model.NovelKnowledge{}).Where("id = ?", id).
		Update("status", model.KnowledgeStatusConfirmed).Error
}

// BatchConfirmByNovel 批量确认小说下所有待审核条目
func (d *KnowledgeDAO) BatchConfirmByNovel(novelID uint) error {
	return d.db.Model(&model.NovelKnowledge{}).
		Where("novel_id = ? AND status = ?", novelID, model.KnowledgeStatusPending).
		Update("status", model.KnowledgeStatusConfirmed).Error
}

// SearchByTags 按标签关键词搜索知识条目
func (d *KnowledgeDAO) SearchByTags(novelID uint, keyword string) ([]model.NovelKnowledge, error) {
	var items []model.NovelKnowledge
	err := d.db.Where("novel_id = ? AND (tags LIKE ? OR title LIKE ? OR content LIKE ?)",
		novelID, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Order("priority DESC").
		Find(&items).Error
	return items, err
}
