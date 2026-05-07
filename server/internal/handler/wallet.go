// server/internal/handler/wallet.go
package handler

import (
	"strconv"

	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// WalletHandler 钱包请求处理器
type WalletHandler struct {
	walletSvc *service.WalletService
}

// NewWalletHandler 创建 WalletHandler 实例
func NewWalletHandler(walletSvc *service.WalletService) *WalletHandler {
	return &WalletHandler{walletSvc: walletSvc}
}

// GetWallet 获取钱包信息
// GET /api/v1/wallet
func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID := c.GetUint("user_id")
	wallet, err := h.walletSvc.GetWallet(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, wallet)
}

// ListTransactions 流水列表
// GET /api/v1/wallet/transactions
func (h *WalletHandler) ListTransactions(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	txns, total, err := h.walletSvc.ListTransactions(userID, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"list":  txns,
		"total": total,
		"page":  page,
	})
}

// Recharge 充值
// POST /api/v1/wallet/recharge
func (h *WalletHandler) Recharge(c *gin.Context) {
	var req struct {
		UserID uint  `json:"user_id" binding:"required"`
		Amount int64 `json:"amount" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.walletSvc.Recharge(req.UserID, req.Amount); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "recharge success", nil)
}
