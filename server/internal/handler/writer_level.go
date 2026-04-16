// server/internal/handler/writer_level.go
package handler

import (
	"strconv"

	"ai-curton/server/internal/middleware"
	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// WriterLevelHandler 写手等级相关请求处理
type WriterLevelHandler struct {
	levelService *service.WriterLevelService
}

// NewWriterLevelHandler 创建 WriterLevelHandler 实例
func NewWriterLevelHandler() *WriterLevelHandler {
	return &WriterLevelHandler{
		levelService: service.NewWriterLevelService(),
	}
}

// GetLevelInfo 获取当前用户等级信息
// GET /api/v1/user/level
func (h *WriterLevelHandler) GetLevelInfo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		Unauthorized(c, "User not authenticated")
		return
	}

	info, err := h.levelService.GetLevelInfo(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, info)
}

// PurchaseUpgrade 付费解锁大神写手
// POST /api/v1/user/level/purchase
func (h *WriterLevelHandler) PurchaseUpgrade(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.levelService.PurchaseUpgrade(userID); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "Upgraded to advanced writer successfully"})
}

// UpdateViewModeRequest 切换视图模式请求
type UpdateViewModeRequest struct {
	ViewMode string `json:"view_mode" binding:"required,oneof=simple advanced"`
}

// UpdateViewMode 切换视图模式
// PUT /api/v1/user/view-mode
func (h *WriterLevelHandler) UpdateViewMode(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateViewModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request: view_mode must be simple or advanced")
		return
	}

	if err := h.levelService.UpdateViewMode(userID, req.ViewMode); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "View mode updated", "view_mode": req.ViewMode})
}

// AdminSetWriterLevelRequest 管理员设置写手等级请求
type AdminSetWriterLevelRequest struct {
	WriterLevel string `json:"writer_level" binding:"required,oneof=beginner advanced"`
}

// AdminSetWriterLevel 管理员手动设置用户写手等级
// PUT /api/v1/admin/users/:uid/writer-level
func (h *WriterLevelHandler) AdminSetWriterLevel(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 64)
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}

	var req AdminSetWriterLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request: writer_level must be beginner or advanced")
		return
	}

	if err := h.levelService.AdminSetLevel(uint(uid), req.WriterLevel); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "Writer level updated", "writer_level": req.WriterLevel})
}
