// server/internal/model/market.go
package model

import "time"

// MemoryOrder 记忆交易订单
type MemoryOrder struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OrderNo      string    `gorm:"size:32;uniqueIndex;not null" json:"order_no"`
	BuyerID      uint      `gorm:"index;not null" json:"buyer_id"`
	SellerID     uint      `gorm:"index;not null" json:"seller_id"`
	MemoryID     uint      `gorm:"index;not null" json:"memory_id"`
	Amount       int64     `gorm:"not null" json:"amount"`
	PlatformFee  int64     `gorm:"not null" json:"platform_fee"`
	SellerIncome int64     `gorm:"not null" json:"seller_income"`
	Status       string    `gorm:"size:20;not null;default:pending" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MemoryLicense 使用授权
type MemoryLicense struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_user_memory;not null" json:"user_id"`
	MemoryID  uint      `gorm:"uniqueIndex:idx_user_memory;not null" json:"memory_id"`
	OrderID   uint      `gorm:"not null" json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MemoryReview 记忆评价
type MemoryReview struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_review_user_mem;not null" json:"user_id"`
	MemoryID  uint      `gorm:"uniqueIndex:idx_review_user_mem;not null" json:"memory_id"`
	Rating    int       `gorm:"not null" json:"rating"`
	Comment   string    `gorm:"size:500" json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// 订单状态常量
const (
	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusRefunded = "refunded"
)

// 平台抽成比例（20%）
const PlatformFeeRate = 0.20
