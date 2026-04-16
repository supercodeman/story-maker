// server/internal/dao/market.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// MarketDAO 市场数据访问层
type MarketDAO struct {
	db *gorm.DB
}

// NewMarketDAO 创建 MarketDAO 实例
func NewMarketDAO() *MarketDAO {
	return &MarketDAO{db: model.DB}
}

// ========== MemoryOrder ==========

// CreateOrder 创建订单
func (d *MarketDAO) CreateOrder(tx *gorm.DB, order *model.MemoryOrder) error {
	return tx.Create(order).Error
}

// GetOrder 获取订单
func (d *MarketDAO) GetOrder(id uint) (*model.MemoryOrder, error) {
	var order model.MemoryOrder
	err := d.db.First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetOrderByNo 根据订单号获取
func (d *MarketDAO) GetOrderByNo(orderNo string) (*model.MemoryOrder, error) {
	var order model.MemoryOrder
	err := d.db.Where("order_no = ?", orderNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// ========== MemoryLicense ==========

// CreateLicense 创建授权
func (d *MarketDAO) CreateLicense(tx *gorm.DB, license *model.MemoryLicense) error {
	return tx.Create(license).Error
}

// HasLicense 检查用户是否拥有记忆授权
func (d *MarketDAO) HasLicense(userID, memoryID uint) bool {
	var count int64
	d.db.Model(&model.MemoryLicense{}).
		Where("user_id = ? AND memory_id = ?", userID, memoryID).
		Count(&count)
	return count > 0
}

// ========== MemoryReview ==========

// CreateReview 创建评价
func (d *MarketDAO) CreateReview(review *model.MemoryReview) error {
	return d.db.Create(review).Error
}

// HasReviewed 检查用户是否已评价
func (d *MarketDAO) HasReviewed(userID, memoryID uint) bool {
	var count int64
	d.db.Model(&model.MemoryReview{}).
		Where("user_id = ? AND memory_id = ?", userID, memoryID).
		Count(&count)
	return count > 0
}

// ListReviews 获取记忆的评价列表
func (d *MarketDAO) ListReviews(memoryID uint, page, pageSize int) ([]model.MemoryReview, int64, error) {
	var reviews []model.MemoryReview
	var total int64

	q := d.db.Where("memory_id = ?", memoryID)
	if err := q.Model(&model.MemoryReview{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&reviews).Error
	return reviews, total, err
}

// GetAvgRating 计算记忆的平均评分
func (d *MarketDAO) GetAvgRating(memoryID uint) (float64, int, error) {
	var result struct {
		AvgRating float64
		Count     int
	}
	err := d.db.Model(&model.MemoryReview{}).
		Where("memory_id = ?", memoryID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as count").
		Scan(&result).Error
	return result.AvgRating, result.Count, err
}
