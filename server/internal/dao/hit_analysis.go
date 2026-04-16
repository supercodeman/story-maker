// server/internal/dao/hit_analysis.go
package dao

import (
	"context"

	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// HitAnalysisDAO 爆款拆解数据访问层
type HitAnalysisDAO struct {
	db *gorm.DB
}

// NewHitAnalysisDAO 创建 HitAnalysisDAO 实例
func NewHitAnalysisDAO() *HitAnalysisDAO {
	return &HitAnalysisDAO{db: model.DB}
}

// Create 创建拆解记录
func (d *HitAnalysisDAO) Create(ctx context.Context, ha *model.HitAnalysis) error {
	return d.db.WithContext(ctx).Create(ha).Error
}

// Get 根据 ID 获取拆解记录
func (d *HitAnalysisDAO) Get(ctx context.Context, id uint) (*model.HitAnalysis, error) {
	var ha model.HitAnalysis
	err := d.db.WithContext(ctx).First(&ha, id).Error
	if err != nil {
		return nil, err
	}
	return &ha, nil
}

// Update 更新拆解记录
func (d *HitAnalysisDAO) Update(ctx context.Context, ha *model.HitAnalysis) error {
	return d.db.WithContext(ctx).Save(ha).Error
}

// Delete 删除拆解记录
func (d *HitAnalysisDAO) Delete(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.HitAnalysis{}, id).Error
}

// ListByUser 获取用户的拆解记录列表
func (d *HitAnalysisDAO) ListByUser(ctx context.Context, userID uint) ([]model.HitAnalysis, error) {
	var list []model.HitAnalysis
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

// ClearSourceText 清除原文（版权合规）
func (d *HitAnalysisDAO) ClearSourceText(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Model(&model.HitAnalysis{}).
		Where("id = ?", id).
		Update("source_text", "").Error
}
