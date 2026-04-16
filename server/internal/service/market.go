// server/internal/service/market.go
package service

import (
	"errors"
	"fmt"
	"math"
	"time"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// MarketService 市场业务逻辑层
type MarketService struct {
	marketDAO *dao.MarketDAO
	memoryDAO *dao.WritingMemoryDAO
	genreDAO  *dao.GenreDAO
	walletDAO *dao.WalletDAO
	db        *gorm.DB
}

// NewMarketService 创建 MarketService 实例
func NewMarketService() *MarketService {
	return &MarketService{
		marketDAO: dao.NewMarketDAO(),
		memoryDAO: dao.NewWritingMemoryDAO(),
		genreDAO:  dao.NewGenreDAO(),
		walletDAO: dao.NewWalletDAO(),
		db:        model.DB,
	}
}

// ListMarketMemories 市场浏览（支持赛道筛选）
func (s *MarketService) ListMarketMemories(category, keyword, orderBy string, page, pageSize int, genreID uint) ([]model.WritingMemory, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	// 赛道筛选：先查出该赛道下的记忆 ID 列表
	var memoryIDs []uint
	if genreID > 0 {
		var err error
		memoryIDs, err = s.genreDAO.ListMemoryIDsByGenre(genreID)
		if err != nil {
			return nil, 0, err
		}
		if len(memoryIDs) == 0 {
			return []model.WritingMemory{}, 0, nil
		}
	}

	return s.memoryDAO.ListPublished(category, keyword, orderBy, page, pageSize, memoryIDs)
}

// GetMarketMemory 获取市场记忆详情（含版权隔离）
func (s *MarketService) GetMarketMemory(memoryID, userID uint) (map[string]interface{}, error) {
	memory, err := s.memoryDAO.Get(memoryID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":             memory.ID,
		"user_id":        memory.UserID,
		"category":       memory.Category,
		"title":          memory.Title,
		"description":    memory.Description,
		"preview_text":   memory.PreviewText,
		"quality":        memory.Quality,
		"quality_detail": memory.QualityDetail,
		"quality_grade":  memory.QualityGrade,
		"price":          memory.Price,
		"sales_count":    memory.SalesCount,
		"avg_rating":     memory.AvgRating,
		"rating_count":   memory.RatingCount,
		"tags":           memory.Tags,
		"sample_len":     memory.SampleLen,
		"status":         memory.Status,
		"created_at":     memory.CreatedAt,
	}

	// 创建者可见所有字段
	if memory.UserID == userID {
		result["features"] = memory.Features
		result["prompt_tpl"] = memory.PromptTpl
		result["anchor_texts"] = memory.AnchorTexts
		return result, nil
	}

	// 购买者可见 PromptTpl + AnchorTexts + Features
	if s.marketDAO.HasLicense(userID, memoryID) {
		result["features"] = memory.Features
		result["prompt_tpl"] = memory.PromptTpl
		result["anchor_texts"] = memory.AnchorTexts
		result["licensed"] = true
		return result, nil
	}

	// 未购买者只能看基础信息
	result["licensed"] = false
	return result, nil
}

// PurchaseMemory 购买记忆
func (s *MarketService) PurchaseMemory(buyerID, memoryID uint) (*model.MemoryOrder, error) {
	// 查询记忆
	memory, err := s.memoryDAO.Get(memoryID)
	if err != nil {
		return nil, fmt.Errorf("memory not found: %w", err)
	}

	// 校验状态
	if memory.Status != model.MemoryStatusPublished {
		return nil, errors.New("memory is not published")
	}

	// 不能购买自己的记忆
	if memory.UserID == buyerID {
		return nil, errors.New("cannot purchase your own memory")
	}

	// 检查是否已购买
	if s.marketDAO.HasLicense(buyerID, memoryID) {
		return nil, errors.New("already purchased")
	}

	// 获取买家钱包
	buyerWallet, err := s.walletDAO.GetOrCreate(buyerID)
	if err != nil {
		return nil, err
	}

	amount := int64(memory.Price)
	platformFee := int64(math.Round(float64(amount) * model.PlatformFeeRate))
	sellerIncome := amount - platformFee

	// 生成订单号
	orderNo := fmt.Sprintf("MO%d%d", time.Now().UnixMilli(), memoryID)

	var order model.MemoryOrder
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 扣减买家积分
		if err := s.walletDAO.Deduct(tx, buyerID, amount, buyerWallet.Version); err != nil {
			return fmt.Errorf("deduct failed: %w", err)
		}

		// 增加卖家积分
		if err := s.walletDAO.Credit(tx, memory.UserID, sellerIncome); err != nil {
			return fmt.Errorf("credit failed: %w", err)
		}

		// 创建订单
		order = model.MemoryOrder{
			OrderNo:      orderNo,
			BuyerID:      buyerID,
			SellerID:     memory.UserID,
			MemoryID:     memoryID,
			Amount:       amount,
			PlatformFee:  platformFee,
			SellerIncome: sellerIncome,
			Status:       model.OrderStatusPaid,
		}
		if err := s.marketDAO.CreateOrder(tx, &order); err != nil {
			return err
		}

		// 创建授权
		license := &model.MemoryLicense{
			UserID:   buyerID,
			MemoryID: memoryID,
			OrderID:  order.ID,
		}
		if err := s.marketDAO.CreateLicense(tx, license); err != nil {
			return err
		}

		// 查询双方最新余额
		var buyerW, sellerW model.UserWallet
		tx.Where("user_id = ?", buyerID).First(&buyerW)
		tx.Where("user_id = ?", memory.UserID).First(&sellerW)

		// 创建买家流水
		if err := s.walletDAO.CreateTransaction(tx, &model.WalletTransaction{
			UserID:      buyerID,
			Type:        model.TxTypePurchase,
			Amount:      -amount,
			Balance:     buyerW.Balance,
			RefType:     "memory_order",
			RefID:       order.ID,
			Description: fmt.Sprintf("购买记忆「%s」", memory.Title),
		}); err != nil {
			return err
		}

		// 创建卖家流水
		if err := s.walletDAO.CreateTransaction(tx, &model.WalletTransaction{
			UserID:      memory.UserID,
			Type:        model.TxTypeIncome,
			Amount:      sellerIncome,
			Balance:     sellerW.Balance,
			RefType:     "memory_order",
			RefID:       order.ID,
			Description: fmt.Sprintf("记忆「%s」被购买，收入 %d 积分", memory.Title, sellerIncome),
		}); err != nil {
			return err
		}

		// 更新销量
		return s.memoryDAO.IncrSalesCount(memoryID)
	})

	if err != nil {
		return nil, err
	}
	return &order, nil
}

// SubmitReview 提交评价
func (s *MarketService) SubmitReview(userID, memoryID uint, rating int, comment string) error {
	// 校验评分范围
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	// 检查是否有授权
	if !s.marketDAO.HasLicense(userID, memoryID) {
		return errors.New("you must purchase the memory before reviewing")
	}

	// 检查是否已评价
	if s.marketDAO.HasReviewed(userID, memoryID) {
		return errors.New("already reviewed")
	}

	review := &model.MemoryReview{
		UserID:   userID,
		MemoryID: memoryID,
		Rating:   rating,
		Comment:  comment,
	}
	if err := s.marketDAO.CreateReview(review); err != nil {
		return err
	}

	// 更新记忆平均评分
	avgRating, count, err := s.marketDAO.GetAvgRating(memoryID)
	if err == nil {
		s.memoryDAO.UpdateRating(memoryID, avgRating, count)
	}

	return nil
}

// ListReviews 获取评价列表
func (s *MarketService) ListReviews(memoryID uint, page, pageSize int) ([]model.MemoryReview, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	return s.marketDAO.ListReviews(memoryID, page, pageSize)
}

// HasLicense 检查用户是否拥有授权
func (s *MarketService) HasLicense(userID, memoryID uint) bool {
	return s.marketDAO.HasLicense(userID, memoryID)
}
