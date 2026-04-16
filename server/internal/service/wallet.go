// server/internal/service/wallet.go
package service

import (
	"errors"
	"fmt"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// WalletService 钱包业务逻辑层
type WalletService struct {
	walletDAO *dao.WalletDAO
	db        *gorm.DB
}

// NewWalletService 创建 WalletService 实例
func NewWalletService() *WalletService {
	return &WalletService{
		walletDAO: dao.NewWalletDAO(),
		db:        model.DB,
	}
}

// GetWallet 获取用户钱包
func (s *WalletService) GetWallet(userID uint) (*model.UserWallet, error) {
	return s.walletDAO.GetOrCreate(userID)
}

// Recharge 充值（MVP 阶段管理员手动操作）
func (s *WalletService) Recharge(userID uint, amount int64) error {
	if amount <= 0 {
		return errors.New("recharge amount must be positive")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.walletDAO.Recharge(tx, userID, amount); err != nil {
			return err
		}

		// 查询充值后余额
		var wallet model.UserWallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}

		return s.walletDAO.CreateTransaction(tx, &model.WalletTransaction{
			UserID:      userID,
			Type:        model.TxTypeRecharge,
			Amount:      amount,
			Balance:     wallet.Balance,
			RefType:     "admin",
			Description: fmt.Sprintf("管理员充值 %d 积分", amount),
		})
	})
}

// ListTransactions 获取流水列表
func (s *WalletService) ListTransactions(userID uint, page, pageSize int) ([]model.WalletTransaction, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	return s.walletDAO.ListTransactions(userID, page, pageSize)
}
