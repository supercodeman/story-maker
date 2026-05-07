// server/internal/service/token_billing.go
package service

import (
	"fmt"
	"log"

	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// 高 Token 消耗任务类型（需要按 Token 计费）
var tokenBilledTaskTypes = map[string]bool{
	model.TaskTypeKnowledgeExtract:         true, // AI 知识提取
	model.TaskTypeOverviewExtract:          true, // 全章生成
	model.TaskTypeChapterExpand:            true, // 章节扩写
	model.TaskTypeOutlineGenerate:          true, // 大纲生成（多轮迭代）
	model.TaskTypeOutlineSummaryExpand:     true, // 大纲扩写
	model.TaskTypeOutlineGenerateCharacters: true, // AI 生成人物
	model.TaskTypeButlerGenerateTopic:      true, // AI 生成主题
	model.TaskTypeButlerGenerateStoryline:  true, // AI 生成故事线
	model.TaskTypeButlerGenerateCharacters: true, // AI 生成角色
}

// TokenRate Token 计费比率（每 1000 token 消耗多少积分）
const TokenRate = 1 // 1 积分 / 1000 tokens

// TokenBillingService Token 计费服务
type TokenBillingService struct {
	walletDAO *dao.WalletDAO
	db        *gorm.DB
}

// NewTokenBillingService 创建 TokenBillingService 实例
func NewTokenBillingService() *TokenBillingService {
	return &TokenBillingService{
		walletDAO: dao.NewWalletDAO(),
		db:        model.DB,
	}
}

// OnTaskCompleted 任务完成时的计费回调
func (s *TokenBillingService) OnTaskCompleted(task *model.AITask) {
	// 仅对高消耗任务类型计费
	if !tokenBilledTaskTypes[task.TaskType] {
		return
	}

	if task.TotalTokens <= 0 {
		return
	}

	// 计算费用：每 1000 token 收取 TokenRate 积分，向上取整
	cost := int64((task.TotalTokens + 999) / 1000 * TokenRate)
	if cost <= 0 {
		return
	}

	// 执行扣费
	if err := s.deductForTask(task.UserID, cost, task); err != nil {
		log.Printf("[token_billing] failed to deduct for task %d (user %d): %v", task.ID, task.UserID, err)
	}
}

// deductForTask 为任务扣费
func (s *TokenBillingService) deductForTask(userID uint, cost int64, task *model.AITask) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 获取钱包
		wallet, err := s.walletDAO.GetOrCreate(userID)
		if err != nil {
			return err
		}

		// 扣费（余额不足也记录，不阻断任务）
		if wallet.Balance < cost {
			log.Printf("[token_billing] user %d balance insufficient (need %d, have %d), skipping deduct", userID, cost, wallet.Balance)
			return nil
		}

		if err := s.walletDAO.Deduct(tx, userID, cost, wallet.Version); err != nil {
			return err
		}

		// 查询扣费后余额
		var updatedWallet model.UserWallet
		if err := tx.Where("user_id = ?", userID).First(&updatedWallet).Error; err != nil {
			return err
		}

		// 记录流水
		return s.walletDAO.CreateTransaction(tx, &model.WalletTransaction{
			UserID:      userID,
			Type:        model.TxTypeTokenConsume,
			Amount:      -cost,
			Balance:     updatedWallet.Balance,
			RefType:     "ai_task",
			RefID:       task.ID,
			Description: fmt.Sprintf("AI任务[%s]消耗 %d tokens，扣费 %d 积分", task.TaskType, task.TotalTokens, cost),
		})
	})
}

// IsTokenBilledTask 判断任务类型是否需要 Token 计费
func IsTokenBilledTask(taskType string) bool {
	return tokenBilledTaskTypes[taskType]
}
