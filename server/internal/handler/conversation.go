// server/internal/handler/conversation.go
package handler

import (
	"strconv"

	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// ConversationHandler 会话请求处理器
type ConversationHandler struct {
	convService *service.ConversationService
}

// NewConversationHandler 创建 ConversationHandler 实例
func NewConversationHandler(convService *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{convService: convService}
}

// CreateConversationRequest 创建会话请求
type CreateConversationRequest struct {
	PortfolioID uint   `json:"portfolio_id" binding:"required"`
	ModelName   string `json:"model_name"`
	Title       string `json:"title"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// CreateConversation 创建会话
// POST /api/v1/conversations
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	conv, err := h.convService.CreateConversation(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Title)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, conv)
}

// GetConversation 获取会话详情（含消息列表）
// GET /api/v1/conversations/:id
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "invalid conversation id")
		return
	}

	userID := c.GetUint("user_id")
	conv, msgs, err := h.convService.GetConversation(c.Request.Context(), uint(convID), userID)
	if err != nil {
		Error(c, 404, err.Error())
		return
	}

	Success(c, gin.H{
		"conversation": conv,
		"messages":     msgs,
	})
}

// ListConversations 获取会话列表
// GET /api/v1/conversations
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var portfolioID *uint
	if pidStr := c.Query("portfolio_id"); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 32)
		if err == nil {
			pidUint := uint(pid)
			portfolioID = &pidUint
		}
	}

	convs, total, err := h.convService.ListConversations(c.Request.Context(), userID, portfolioID, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"conversations": convs,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// SendMessage 在会话中发送消息
// POST /api/v1/conversations/:id/messages
func (h *ConversationHandler) SendMessage(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "invalid conversation id")
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	taskID, err := h.convService.SendMessage(c.Request.Context(), uint(convID), userID, req.Content)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
	})
}

// ArchiveConversation 归档会话
// DELETE /api/v1/conversations/:id
func (h *ConversationHandler) ArchiveConversation(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "invalid conversation id")
		return
	}

	userID := c.GetUint("user_id")
	if err := h.convService.ArchiveConversation(c.Request.Context(), uint(convID), userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "conversation archived", nil)
}
