// server/internal/dao/chapter_review.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// ChapterReviewDAO 章节审核评分数据访问层
type ChapterReviewDAO struct {
	db *gorm.DB
}

// NewChapterReviewDAO 创建 ChapterReviewDAO 实例
func NewChapterReviewDAO() *ChapterReviewDAO {
	return &ChapterReviewDAO{db: model.DB}
}

// Create 创建审核评分记录
func (d *ChapterReviewDAO) Create(review *model.ChapterReview) error {
	return d.db.Create(review).Error
}

// ListByWorkflow 按工作流 ID 查询审核记录
func (d *ChapterReviewDAO) ListByWorkflow(workflowID uint) ([]model.ChapterReview, error) {
	var reviews []model.ChapterReview
	err := d.db.Where("workflow_id = ?", workflowID).
		Order("round ASC").
		Find(&reviews).Error
	return reviews, err
}

// ListByNovel 按小说 ID 查询审核记录
func (d *ChapterReviewDAO) ListByNovel(novelID uint) ([]model.ChapterReview, error) {
	var reviews []model.ChapterReview
	err := d.db.Where("novel_id = ?", novelID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

// GetLatestByChapter 获取章节最近一条审核记录
func (d *ChapterReviewDAO) GetLatestByChapter(chapterID uint) (*model.ChapterReview, error) {
	var review model.ChapterReview
	err := d.db.Where("chapter_id = ?", chapterID).
		Order("created_at DESC").
		First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// GetAvgScoreByNovel 获取小说的平均审核评分
func (d *ChapterReviewDAO) GetAvgScoreByNovel(novelID uint) (float64, error) {
	var result struct {
		AvgScore float64
	}
	err := d.db.Model(&model.ChapterReview{}).
		Where("novel_id = ?", novelID).
		Select("COALESCE(AVG(overall_score), 0) as avg_score").
		Scan(&result).Error
	return result.AvgScore, err
}
