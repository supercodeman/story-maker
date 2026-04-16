// server/internal/dao/wallet.go
package dao

import (
	"errors"

	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// WalletDAO 钱包数据访问层
type WalletDAO struct {
	db *gorm.DB
}

// NewWalletDAO 创建 WalletDAO 实例
func NewWalletDAO() *WalletDAO {
	return &WalletDAO{db: model.DB}
}

// GetOrCreate 获取或创建用户钱包
func (d *WalletDAO) GetOrCreate(userID uint) (*model.UserWallet, error) {
	var wallet model.UserWallet
	err := d.db.Where("user_id = ?", userID).First(&wallet).Error
	if err == nil {
		return &wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	wallet = model.UserWallet{UserID: userID}
	if err := d.db.Create(&wallet).Error; err != nil {
		// 并发创建时可能冲突，重新查询
		return d.GetOrCreate(userID)
	}
	return &wallet, nil
}

// Deduct 扣减积分（乐观锁）
func (d *WalletDAO) Deduct(tx *gorm.DB, userID uint, amount int64, version int64) error {
	result := tx.Model(&model.UserWallet{}).
		Where("user_id = ? AND version = ? AND balance >= ?", userID, version, amount).
		Updates(map[string]interface{}{
			"balance":     gorm.Expr("balance - ?", amount),
			"total_spent": gorm.Expr("total_spent + ?", amount),
			"version":     gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("insufficient balance or version conflict")
	}
	return nil
}

// Credit 增加积分（乐观锁）
func (d *WalletDAO) Credit(tx *gorm.DB, userID uint, amount int64) error {
	// 确保钱包存在
	tx.Where("user_id = ?", userID).FirstOrCreate(&model.UserWallet{UserID: userID})

	result := tx.Model(&model.UserWallet{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"balance":      gorm.Expr("balance + ?", amount),
			"total_income": gorm.Expr("total_income + ?", amount),
			"version":      gorm.Expr("version + 1"),
		})
	return result.Error
}

// Recharge 充值（管理员操作）
func (d *WalletDAO) Recharge(tx *gorm.DB, userID uint, amount int64) error {
	tx.Where("user_id = ?", userID).FirstOrCreate(&model.UserWallet{UserID: userID})

	return tx.Model(&model.UserWallet{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"balance": gorm.Expr("balance + ?", amount),
			"version": gorm.Expr("version + 1"),
		}).Error
}

// CreateTransaction 创建流水记录
func (d *WalletDAO) CreateTransaction(tx *gorm.DB, txn *model.WalletTransaction) error {
	return tx.Create(txn).Error
}

// ListTransactions 获取用户流水列表
func (d *WalletDAO) ListTransactions(userID uint, page, pageSize int) ([]model.WalletTransaction, int64, error) {
	var txns []model.WalletTransaction
	var total int64

	q := d.db.Where("user_id = ?", userID)
	if err := q.Model(&model.WalletTransaction{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&txns).Error
	return txns, total, err
}
