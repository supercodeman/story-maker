// server/internal/handler/butler_iterative.go
// 管家多轮迭代 Handler
package handler

import (
	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// ButlerIterativeHandler 管家多轮迭代请求处理层
type ButlerIterativeHandler struct {
	svc *service.ButlerIterativeService
}

// NewButlerIterativeHandler 创建实例
func NewButlerIterativeHandler(svc *service.ButlerIterativeService) *ButlerIterativeHandler {
	return &ButlerIterativeHandler{svc: svc}
}

// StartIteration 启动管家多轮迭代
// POST /api/v1/outline/butler-iterate
func (h *ButlerIterativeHandler) StartIteration(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.ButlerIterateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	iterationID, err := h.svc.StartIteration(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"iteration_id": iterationID})
}

// GetIterationStatus 查询迭代状态
// GET /api/v1/outline/butler-iterate/:iteration_id
func (h *ButlerIterativeHandler) GetIterationStatus(c *gin.Context) {
	iterationID := c.Param("iteration_id")
	if iterationID == "" {
		BadRequest(c, "iteration_id is required")
		return
	}

	status, err := h.svc.GetIterationStatus(iterationID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, status)
}
