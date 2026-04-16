// server/internal/dao/asset.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// AssetDAO 资源数据访问层
type AssetDAO struct {
	db *gorm.DB
}

// NewAssetDAO 创建 AssetDAO 实例
func NewAssetDAO() *AssetDAO {
	return &AssetDAO{db: model.DB}
}

// Create 创建资源记录
func (d *AssetDAO) Create(a *model.Asset) error {
	return d.db.Create(a).Error
}

// GetByID 根据 ID 获取资源
func (d *AssetDAO) GetByID(id uint) (*model.Asset, error) {
	var a model.Asset
	err := d.db.First(&a, id).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByPortfolioID 获取作品集下的所有资源
func (d *AssetDAO) ListByPortfolioID(portfolioID uint) ([]model.Asset, error) {
	var assets []model.Asset
	err := d.db.Where("portfolio_id = ?", portfolioID).
		Order("created_at DESC").
		Find(&assets).Error
	return assets, err
}

// Delete 删除资源记录
func (d *AssetDAO) Delete(id uint) error {
	return d.db.Delete(&model.Asset{}, id).Error
}

// ListByChapterID 获取章节下的资源列表（按类型可选过滤）
func (d *AssetDAO) ListByChapterID(chapterID uint, assetType string) ([]model.Asset, error) {
	var assets []model.Asset
	query := d.db.Where("chapter_id = ?", chapterID)
	if assetType != "" {
		query = query.Where("type = ?", assetType)
	}
	err := query.Order("created_at DESC").Find(&assets).Error
	return assets, err
}

// ListByChapterIDs 批量获取多个章节的资源
func (d *AssetDAO) ListByChapterIDs(chapterIDs []uint, assetType string) ([]model.Asset, error) {
	var assets []model.Asset
	query := d.db.Where("chapter_id IN ?", chapterIDs)
	if assetType != "" {
		query = query.Where("type = ?", assetType)
	}
	err := query.Order("chapter_id ASC, created_at DESC").Find(&assets).Error
	return assets, err
}
