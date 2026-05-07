// server/internal/handler/user_behavior.go
package handler

import (
	"story-maker/server/internal/model"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// UserBehaviorHandler 用户行为上报请求处理层
type UserBehaviorHandler struct {
	svc *service.UserBehaviorService
}

// NewUserBehaviorHandler 创建 UserBehaviorHandler 实例
func NewUserBehaviorHandler(svc *service.UserBehaviorService) *UserBehaviorHandler {
	return &UserBehaviorHandler{svc: svc}
}

// RecordEvent 上报用户行为事件
// POST /api/v1/behavior/events
func (h *UserBehaviorHandler) RecordEvent(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.RecordBehaviorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 白名单校验事件类型
	if !model.ValidBehaviorTypes[req.EventType] {
		BadRequest(c, "invalid event_type")
		return
	}

	if err := h.svc.RecordEvent(userID, req.NovelID, req.ChapterID, req.EventType, req.Payload); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "event recorded"})
}
