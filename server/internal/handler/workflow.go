// server/internal/handler/workflow.go
package handler

import (
	"strconv"

	"ai-curton/server/internal/service"
	"github.com/gin-gonic/gin"
)

// WorkflowHandler 工作流请求处理器
type WorkflowHandler struct {
	workflowService *service.WorkflowService
}

// NewWorkflowHandler 创建 WorkflowHandler 实例
func NewWorkflowHandler(workflowService *service.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{workflowService: workflowService}
}

// SubmitRequest 提交工作流请求体
type SubmitRequest struct {
	PortfolioID  uint                   `json:"portfolio_id" binding:"required"`
	WorkflowType string                 `json:"workflow_type" binding:"required"`
	ModelName    string                 `json:"model_name"`
	Params       map[string]interface{} `json:"params"`
}

// Submit 提交工作流
// POST /api/v1/ai/workflows/submit
func (h *WorkflowHandler) Submit(c *gin.Context) {
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	svcReq := &service.SubmitWorkflowRequest{
		PortfolioID:  req.PortfolioID,
		WorkflowType: req.WorkflowType,
		ModelName:    req.ModelName,
		Params:       req.Params,
	}

	workflowID, err := h.workflowService.SubmitWorkflow(c.Request.Context(), userID, svcReq)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"workflow_id": workflowID,
		"status":      "pending",
	})
}

// Get 获取工作流详情
// GET /api/v1/ai/workflows/:id
func (h *WorkflowHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "invalid workflow id")
		return
	}

	userID := c.GetUint("user_id")

	workflow, nodes, err := h.workflowService.GetWorkflow(c.Request.Context(), uint(id), userID)
	if err != nil {
		Error(c, 404, err.Error())
		return
	}

	Success(c, gin.H{
		"workflow": workflow,
		"nodes":    nodes,
	})
}

// ListActive 查询指定小说下的活跃工作流
// GET /api/v1/ai/workflows/active?novel_id=X
func (h *WorkflowHandler) ListActive(c *gin.Context) {
	novelIDStr := c.Query("novel_id")
	if novelIDStr == "" {
		BadRequest(c, "novel_id is required")
		return
	}
	novelID, err := strconv.ParseUint(novelIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "invalid novel_id")
		return
	}

	userID := c.GetUint("user_id")

	workflows, err := h.workflowService.ListActiveByNovel(c.Request.Context(), uint(novelID), userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"workflows": workflows,
	})
}

// Cancel 取消工作流
// DELETE /api/v1/ai/workflows/:id
func (h *WorkflowHandler) Cancel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "invalid workflow id")
		return
	}

	userID := c.GetUint("user_id")

	if err := h.workflowService.CancelWorkflow(c.Request.Context(), uint(id), userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "workflow cancelled", nil)
}
