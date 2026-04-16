// server/internal/dao/novel.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// NovelDAO 小说数据访问层
type NovelDAO struct {
	db *gorm.DB
}

// NewNovelDAO 创建 NovelDAO 实例
func NewNovelDAO() *NovelDAO {
	return &NovelDAO{db: model.DB}
}

// ========== Novel CRUD ==========

// CreateNovel 创建小说
func (d *NovelDAO) CreateNovel(novel *model.Novel) error {
	return d.db.Create(novel).Error
}

// GetNovel 根据 ID 获取小说
func (d *NovelDAO) GetNovel(id uint) (*model.Novel, error) {
	var novel model.Novel
	err := d.db.First(&novel, id).Error
	if err != nil {
		return nil, err
	}
	return &novel, nil
}

// UpdateNovel 更新小说
func (d *NovelDAO) UpdateNovel(novel *model.Novel) error {
	return d.db.Save(novel).Error
}

// DeleteNovel 删除小说
func (d *NovelDAO) DeleteNovel(id uint) error {
	return d.db.Delete(&model.Novel{}, id).Error
}

// ListNovelsByPortfolio 获取作品集下的所有小说，可选按 source 过滤
func (d *NovelDAO) ListNovelsByPortfolio(portfolioID uint, source string) ([]model.Novel, error) {
	var novels []model.Novel
	query := d.db.Where("portfolio_id = ?", portfolioID)
	if source != "" {
		query = query.Where("source = ?", source)
	}
	err := query.Order("created_at DESC").Find(&novels).Error
	return novels, err
}

// ListRepairedNovels 查找修复工具创建的临时小说
func (d *NovelDAO) ListRepairedNovels(portfolioID uint) ([]model.Novel, error) {
	var novels []model.Novel
	err := d.db.Where("portfolio_id = ? AND source = ? AND description = ?",
		portfolioID, model.NovelSourceButler, "由修复工具从历史管家任务中恢复").
		Find(&novels).Error
	return novels, err
}

// DeleteNovelWithChapters 删除小说及其所有章节
func (d *NovelDAO) DeleteNovelWithChapters(novelID uint) error {
	// 先删章节
	if err := d.db.Where("novel_id = ?", novelID).Delete(&model.Chapter{}).Error; err != nil {
		return err
	}
	// 再删小说
	return d.db.Delete(&model.Novel{}, novelID).Error
}

// UpdateTokenBudget 更新小说 token 预算
func (d *NovelDAO) UpdateTokenBudget(novelID uint, budget int) error {
	return d.db.Model(&model.Novel{}).Where("id = ?", novelID).Update("token_budget", budget).Error
}

// UpdateTokenUsed 更新小说已用 token 缓存
func (d *NovelDAO) UpdateTokenUsed(novelID uint, used int) error {
	return d.db.Model(&model.Novel{}).Where("id = ?", novelID).Update("token_used", used).Error
}

// ========== Chapter CRUD ==========

// CreateChapter 创建章节
func (d *NovelDAO) CreateChapter(chapter *model.Chapter) error {
	return d.db.Create(chapter).Error
}

// GetChapter 根据 ID 获取章节
func (d *NovelDAO) GetChapter(id uint) (*model.Chapter, error) {
	var chapter model.Chapter
	err := d.db.First(&chapter, id).Error
	if err != nil {
		return nil, err
	}
	return &chapter, nil
}

// UpdateChapter 更新章节
func (d *NovelDAO) UpdateChapter(chapter *model.Chapter) error {
	return d.db.Save(chapter).Error
}

// DeleteChapter 删除章节
func (d *NovelDAO) DeleteChapter(id uint) error {
	return d.db.Delete(&model.Chapter{}, id).Error
}

// ListChaptersByNovel 获取小说下的所有章节（按 sort_order 排序）
func (d *NovelDAO) ListChaptersByNovel(novelID uint) ([]model.Chapter, error) {
	var chapters []model.Chapter
	err := d.db.Where("novel_id = ?", novelID).
		Order("sort_order ASC").
		Find(&chapters).Error
	return chapters, err
}

// GetMaxSortOrder 获取小说下最大的排序序号
func (d *NovelDAO) GetMaxSortOrder(novelID uint) (int, error) {
	var maxOrder *int
	err := d.db.Model(&model.Chapter{}).
		Where("novel_id = ?", novelID).
		Select("MAX(sort_order)").
		Scan(&maxOrder).Error
	if err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return 0, nil
	}
	return *maxOrder, nil
}

// ReorderChapters 批量更新章节排序
func (d *NovelDAO) ReorderChapters(novelID uint, chapterIDs []uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		for i, cid := range chapterIDs {
			if err := tx.Model(&model.Chapter{}).
				Where("id = ? AND novel_id = ?", cid, novelID).
				Update("sort_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CountChaptersByNovel 统计小说的章节数
func (d *NovelDAO) CountChaptersByNovel(novelID uint) (int64, error) {
	var count int64
	err := d.db.Model(&model.Chapter{}).Where("novel_id = ?", novelID).Count(&count).Error
	return count, err
}

// SumWordCountByNovel 统计小说的总字数
func (d *NovelDAO) SumWordCountByNovel(novelID uint) (int, error) {
	var total *int
	err := d.db.Model(&model.Chapter{}).
		Where("novel_id = ?", novelID).
		Select("COALESCE(SUM(word_count), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// GetPreviousChapters 获取当前章节之前的章节列表（用于 AI 上下文）
// 返回最近的 limit 章，按 sort_order 升序排列
func (d *NovelDAO) GetPreviousChapters(novelID uint, sortOrder int, limit int) ([]model.Chapter, error) {
	var chapters []model.Chapter
	err := d.db.Where("novel_id = ? AND sort_order < ?", novelID, sortOrder).
		Order("sort_order DESC").
		Limit(limit).
		Find(&chapters).Error
	if err != nil {
		return nil, err
	}
	// 反转为升序
	for i, j := 0, len(chapters)-1; i < j; i, j = i+1, j-1 {
		chapters[i], chapters[j] = chapters[j], chapters[i]
	}
	return chapters, nil
}

// GetNextChapters 获取当前章节之后的章节列表（用于概要润色上下文）
// 返回最近的 limit 章，按 sort_order 升序排列
func (d *NovelDAO) GetNextChapters(novelID uint, sortOrder int, limit int) ([]model.Chapter, error) {
	var chapters []model.Chapter
	err := d.db.Where("novel_id = ? AND sort_order > ?", novelID, sortOrder).
		Order("sort_order ASC").
		Limit(limit).
		Find(&chapters).Error
	return chapters, err
}

// BatchCreateChapters 批量创建章节
func (d *NovelDAO) BatchCreateChapters(chapters []model.Chapter) error {
	return d.db.Create(&chapters).Error
}

// ========== ChapterVersion CRUD ==========

// CreateVersion 创建版本记录
func (d *NovelDAO) CreateVersion(version *model.ChapterVersion) error {
	return d.db.Create(version).Error
}

// GetVersion 根据 ID 获取版本
func (d *NovelDAO) GetVersion(id uint) (*model.ChapterVersion, error) {
	var version model.ChapterVersion
	err := d.db.First(&version, id).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// ListVersionsByChapter 获取章节的所有版本（按版本号倒序）
func (d *NovelDAO) ListVersionsByChapter(chapterID uint) ([]model.ChapterVersion, error) {
	var versions []model.ChapterVersion
	err := d.db.Where("chapter_id = ?", chapterID).
		Order("version DESC").
		Find(&versions).Error
	return versions, err
}

// GetLatestVersion 获取章节的最新版本
func (d *NovelDAO) GetLatestVersion(chapterID uint) (*model.ChapterVersion, error) {
	var version model.ChapterVersion
	err := d.db.Where("chapter_id = ?", chapterID).
		Order("version DESC").
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}
