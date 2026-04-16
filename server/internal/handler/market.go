// server/internal/handler/market.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// MarketHandler 市场请求处理器
type MarketHandler struct {
	marketSvc *service.MarketService
}

// NewMarketHandler 创建 MarketHandler 实例
func NewMarketHandler(marketSvc *service.MarketService) *MarketHandler {
	return &MarketHandler{marketSvc: marketSvc}
}

// ListMemories 市场浏览
// GET /api/v1/market/memories
func (h *MarketHandler) ListMemories(c *gin.Context) {
	category := c.Query("category")
	keyword := c.Query("keyword")
	orderBy := c.DefaultQuery("order_by", "sales")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	genreID, _ := strconv.ParseUint(c.Query("genre_id"), 10, 64)

	memories, total, err := h.marketSvc.ListMarketMemories(category, keyword, orderBy, page, pageSize, uint(genreID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"list":  memories,
		"total": total,
		"page":  page,
	})
}

// GetMemory 市场记忆详情
// GET /api/v1/market/memories/:mid
func (h *MarketHandler) GetMemory(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	result, err := h.marketSvc.GetMarketMemory(uint(mid), userID)
	if err != nil {
		Error(c, http.StatusNotFound, "memory not found")
		return
	}

	Success(c, result)
}

// Buy 购买记忆
// POST /api/v1/market/memories/:mid/buy
func (h *MarketHandler) Buy(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	order, err := h.marketSvc.PurchaseMemory(userID, uint(mid))
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, order)
}

// ListReviews 评价列表
// GET /api/v1/market/memories/:mid/reviews
func (h *MarketHandler) ListReviews(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	reviews, total, err := h.marketSvc.ListReviews(uint(mid), page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"list":  reviews,
		"total": total,
		"page":  page,
	})
}

// SubmitReview 提交评价
// POST /api/v1/market/memories/:mid/reviews
func (h *MarketHandler) SubmitReview(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")

	var req struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.marketSvc.SubmitReview(userID, uint(mid), req.Rating, req.Comment); err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, "review submitted", nil)
}
