// server/internal/model/wallet.go
package model

import "time"

// UserWallet 用户钱包
type UserWallet struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Balance       int64     `gorm:"default:0;not null" json:"balance"`
	FrozenBalance int64     `gorm:"default:0;not null" json:"frozen_balance"`
	TotalIncome   int64     `gorm:"default:0;not null" json:"total_income"`
	TotalSpent    int64     `gorm:"default:0;not null" json:"total_spent"`
	Version       int64     `gorm:"default:0;not null" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// WalletTransaction 钱包流水
type WalletTransaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	Type        string    `gorm:"size:20;not null" json:"type"`
	Amount      int64     `gorm:"not null" json:"amount"`
	Balance     int64     `gorm:"not null" json:"balance"`
	RefType     string    `gorm:"size:30" json:"ref_type"`
	RefID       uint      `gorm:"default:0" json:"ref_id"`
	Description string    `gorm:"size:200" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// 流水类型常量
const (
	TxTypeRecharge      = "recharge"
	TxTypePurchase      = "purchase"
	TxTypeIncome        = "income"
	TxTypeWithdraw      = "withdraw"
	TxTypeRefund        = "refund"
	TxTypeLevelPurchase = "level_purchase" // 大神写手解锁购买
	TxTypeTokenConsume  = "token_consume"  // Token 消耗扣费
)
