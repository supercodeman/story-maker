// server/internal/handler/portfolio.go
package handler

import (
	"net/http"
	"strconv"

	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// PortfolioHandler 作品集请求处理层
type PortfolioHandler struct {
	svc *service.PortfolioService
}

// NewPortfolioHandler 创建 PortfolioHandler 实例
func NewPortfolioHandler() *PortfolioHandler {
	return &PortfolioHandler{svc: service.NewPortfolioService()}
}

// List 获取作品集列表（按 workspace_id 过滤）
// GET /api/v1/portfolios?workspace_id=xxx
func (h *PortfolioHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsIDStr := c.Query("workspace_id")
	if wsIDStr == "" {
		BadRequest(c, "workspace_id is required")
		return
	}

	wsID, err := strconv.ParseUint(wsIDStr, 10, 64)
	if err != nil {
		BadRequest(c, "invalid workspace_id")
		return
	}

	portfolios, err := h.svc.List(uint(wsID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, portfolios)
}

// Create 创建作品集
// POST /api/v1/portfolios
func (h *PortfolioHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.CreatePortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	p, err := h.svc.Create(userID, &req)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, p)
}

// Get 获取作品集详情
// GET /api/v1/portfolios/:id
func (h *PortfolioHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio id")
		return
	}

	p, err := h.svc.GetByID(uint(pID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, p)
}

// Update 更新作品集
// PUT /api/v1/portfolios/:id
func (h *PortfolioHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio id")
		return
	}

	var req service.UpdatePortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	p, err := h.svc.Update(uint(pID), userID, &req)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, p)
}

// Delete 删除作品集
// DELETE /api/v1/portfolios/:id
func (h *PortfolioHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio id")
		return
	}

	if err := h.svc.Delete(uint(pID), userID); err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, gin.H{"message": "portfolio deleted"})
}
