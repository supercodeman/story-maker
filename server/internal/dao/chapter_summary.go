// server/internal/dao/chapter_summary.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// ChapterSummaryDAO 递归摘要树数据访问层
type ChapterSummaryDAO struct {
	db *gorm.DB
}

// NewChapterSummaryDAO 创建 ChapterSummaryDAO 实例
func NewChapterSummaryDAO() *ChapterSummaryDAO {
	return &ChapterSummaryDAO{db: model.DB}
}

// Create 创建摘要节点
func (d *ChapterSummaryDAO) Create(node *model.ChapterSummaryNode) error {
	return d.db.Create(node).Error
}

// Update 更新摘要节点
func (d *ChapterSummaryDAO) Update(node *model.ChapterSummaryNode) error {
	return d.db.Save(node).Error
}

// Delete 删除摘要节点
func (d *ChapterSummaryDAO) Delete(id uint) error {
	return d.db.Delete(&model.ChapterSummaryNode{}, id).Error
}

// ListByNovelAndLevel 按小说 ID 和层级查询摘要节点
func (d *ChapterSummaryDAO) ListByNovelAndLevel(novelID uint, level int) ([]model.ChapterSummaryNode, error) {
	var nodes []model.ChapterSummaryNode
	err := d.db.Where("novel_id = ? AND level = ?", novelID, level).
		Order("start_chapter ASC").
		Find(&nodes).Error
	return nodes, err
}

// GetByRange 按小说 ID、层级和章节范围查询摘要节点
func (d *ChapterSummaryDAO) GetByRange(novelID uint, level, startChapter, endChapter int) (*model.ChapterSummaryNode, error) {
	var node model.ChapterSummaryNode
	err := d.db.Where("novel_id = ? AND level = ? AND start_chapter = ? AND end_chapter = ?",
		novelID, level, startChapter, endChapter).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// DeleteByNovel 删除小说的所有摘要节点（用于全量重建）
func (d *ChapterSummaryDAO) DeleteByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.ChapterSummaryNode{}).Error
}

// ListByNovel 获取小说的所有摘要节点
func (d *ChapterSummaryDAO) ListByNovel(novelID uint) ([]model.ChapterSummaryNode, error) {
	var nodes []model.ChapterSummaryNode
	err := d.db.Where("novel_id = ?", novelID).
		Order("level ASC, start_chapter ASC").
		Find(&nodes).Error
	return nodes, err
}
