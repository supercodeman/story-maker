// server/internal/service/writer_level.go
package service

import (
	"errors"
	"fmt"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// 成长解锁阈值
const (
	UnlockWordCount  = 100000 // 10 万字
	UnlockChapters   = 50     // 50 章
	UnlockNovels     = 1      // 完本 1 部
	LevelPurchaseFee = 9900   // 解锁费用（单位：积分，99 元）
)

// LevelInfo 等级信息（含进度）
type LevelInfo struct {
	WriterLevel string `json:"writer_level"`
	ViewMode    string `json:"view_mode"`
	LevelSource string `json:"level_source,omitempty"`
	Progress    struct {
		TotalWordCount  int64 `json:"total_word_count"`
		TotalChapters   int   `json:"total_chapters"`
		CompletedNovels int   `json:"completed_novels"`
		WordTarget      int64 `json:"word_target"`
		ChapterTarget   int   `json:"chapter_target"`
		NovelTarget     int   `json:"novel_target"`
	} `json:"progress"`
}

// WriterLevelService 写手等级服务
type WriterLevelService struct {
	userDAO   *dao.UserDAO
	walletDAO *dao.WalletDAO
	db        *gorm.DB
}

// NewWriterLevelService 创建 WriterLevelService 实例
func NewWriterLevelService() *WriterLevelService {
	return &WriterLevelService{
		userDAO:   dao.NewUserDAO(),
		walletDAO: dao.NewWalletDAO(),
		db:        model.DB,
	}
}

// GetLevelInfo 获取用户等级信息（含进度）
func (s *WriterLevelService) GetLevelInfo(userID uint) (*LevelInfo, error) {
	user, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 管理员自动修复：如果是 admin 但还是 beginner，自动升级
	if user.Role == "admin" && user.WriterLevel != model.WriterLevelAdvanced {
		_ = s.userDAO.UpdateWriterLevel(userID, model.WriterLevelAdvanced, model.LevelSourceAdmin)
		user.WriterLevel = model.WriterLevelAdvanced
		user.LevelSource = model.LevelSourceAdmin
		user.ViewMode = model.ViewModeAdvanced
		_ = s.userDAO.UpdateViewMode(userID, model.ViewModeAdvanced)
	}

	info := &LevelInfo{
		WriterLevel: user.WriterLevel,
		ViewMode:    user.ViewMode,
		LevelSource: user.LevelSource,
	}
	info.Progress.TotalWordCount = user.TotalWordCount
	info.Progress.TotalChapters = user.TotalChapters
	info.Progress.CompletedNovels = user.CompletedNovels
	info.Progress.WordTarget = UnlockWordCount
	info.Progress.ChapterTarget = UnlockChapters
	info.Progress.NovelTarget = UnlockNovels

	return info, nil
}

// CheckAndUpgrade 检查是否满足成长解锁条件，满足则自动升级
func (s *WriterLevelService) CheckAndUpgrade(userID uint) (upgraded bool, err error) {
	user, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		return false, err
	}

	// 已经是大神，无需升级
	if user.WriterLevel == model.WriterLevelAdvanced {
		return false, nil
	}

	// 检查是否满足任一解锁条件
	if user.TotalWordCount < UnlockWordCount &&
		user.TotalChapters < UnlockChapters &&
		user.CompletedNovels < UnlockNovels {
		return false, nil
	}

	// 满足条件，执行升级
	if err := s.userDAO.UpdateWriterLevel(userID, model.WriterLevelAdvanced, model.LevelSourceGrowth); err != nil {
		return false, fmt.Errorf("failed to upgrade writer level: %w", err)
	}

	return true, nil
}

// PurchaseUpgrade 付费解锁大神写手
func (s *WriterLevelService) PurchaseUpgrade(userID uint) error {
	user, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.WriterLevel == model.WriterLevelAdvanced {
		return errors.New("already advanced writer")
	}

	// 管理员免费开通
	if user.Role == "admin" {
		return s.userDAO.UpdateWriterLevel(userID, model.WriterLevelAdvanced, model.LevelSourceAdmin)
	}

	// 事务：扣费 + 升级
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 获取钱包
		wallet, err := s.walletDAO.GetOrCreate(userID)
		if err != nil {
			return err
		}

		// 扣费
		if err := s.walletDAO.Deduct(tx, userID, LevelPurchaseFee, wallet.Version); err != nil {
			return fmt.Errorf("insufficient balance: %w", err)
		}

		// 查询扣费后余额
		var updatedWallet model.UserWallet
		if err := tx.Where("user_id = ?", userID).First(&updatedWallet).Error; err != nil {
			return err
		}

		// 记录流水
		if err := s.walletDAO.CreateTransaction(tx, &model.WalletTransaction{
			UserID:      userID,
			Type:        model.TxTypeLevelPurchase,
			Amount:      -LevelPurchaseFee,
			Balance:     updatedWallet.Balance,
			RefType:     "level_upgrade",
			Description: "购买大神写手解锁",
		}); err != nil {
			return err
		}

		// 升级等级
		return tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"writer_level":    model.WriterLevelAdvanced,
			"level_source":    model.LevelSourcePurchase,
			"level_unlock_at": tx.NowFunc(),
		}).Error
	})
}

// UpdateStats 更新创作统计（章节保存时调用）
func (s *WriterLevelService) UpdateStats(userID uint, wordDelta int64, chapterDelta int) error {
	if wordDelta == 0 && chapterDelta == 0 {
		return nil
	}
	return s.userDAO.UpdateWriterStats(userID, wordDelta, chapterDelta)
}

// UpdateViewMode 切换视图模式
func (s *WriterLevelService) UpdateViewMode(userID uint, mode string) error {
	if mode != model.ViewModeSimple && mode != model.ViewModeAdvanced {
		return errors.New("invalid view mode, must be simple or advanced")
	}

	user, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// 只有大神写手才能切换到高级视图
	if mode == model.ViewModeAdvanced && user.WriterLevel != model.WriterLevelAdvanced {
		return errors.New("advanced view requires advanced writer level")
	}

	return s.userDAO.UpdateViewMode(userID, mode)
}

// AdminSetLevel 管理员手动设置用户写手等级（免费，不扣费）
func (s *WriterLevelService) AdminSetLevel(userID uint, level string) error {
	if level != model.WriterLevelBeginner && level != model.WriterLevelAdvanced {
		return errors.New("invalid writer level")
	}

	_, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	return s.userDAO.UpdateWriterLevel(userID, level, model.LevelSourceAdmin)
}
