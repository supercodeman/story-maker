// server/internal/handler/novel_overview.go
package handler

import (
	"strconv"

	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// OverviewHandler 总览请求处理层
type OverviewHandler struct {
	svc *service.OverviewService
}

// NewOverviewHandler 创建 OverviewHandler 实例
func NewOverviewHandler(svc *service.OverviewService) *OverviewHandler {
	return &OverviewHandler{svc: svc}
}

// GetOverview 获取总览数据
// GET /api/v1/novels/:id/overview
func (h *OverviewHandler) GetOverview(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	data, err := h.svc.GetOverview(uint(novelID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, data)
}

// CreateRelation 创建人物关系
// POST /api/v1/novels/:id/overview/relations
func (h *OverviewHandler) CreateRelation(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.CreateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	r, err := h.svc.CreateRelation(uint(novelID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, r)
}

// UpdateRelation 更新人物关系
// PUT /api/v1/novels/:id/overview/relations/:rid
func (h *OverviewHandler) UpdateRelation(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	rid, err := strconv.ParseUint(c.Param("rid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid relation id")
		return
	}

	var req service.UpdateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	r, err := h.svc.UpdateRelation(uint(novelID), uint(rid), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, r)
}

// DeleteRelation 删除人物关系
// DELETE /api/v1/novels/:id/overview/relations/:rid
func (h *OverviewHandler) DeleteRelation(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	rid, err := strconv.ParseUint(c.Param("rid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid relation id")
		return
	}

	if err := h.svc.DeleteRelation(uint(novelID), uint(rid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Extract AI 提取总览元数据
// POST /api/v1/novels/:id/overview/extract
func (h *OverviewHandler) Extract(c *gin.Context) {
	userID := c.GetUint("user_id")
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.ExtractOverviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	taskID, err := h.svc.ExtractOverview(c.Request.Context(), userID, uint(novelID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// ParseExtract 解析 AI 提取结果并入库
// POST /api/v1/novels/:id/overview/extract/parse
func (h *OverviewHandler) ParseExtract(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req struct {
		TaskID uint `json:"task_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.ParseExtractResult(uint(novelID), req.TaskID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// SubmitRevision 提交变更（触发分析工作流）
// POST /api/v1/novels/:id/overview/revision
func (h *OverviewHandler) SubmitRevision(c *gin.Context) {
	userID := c.GetUint("user_id")
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	// 从查询参数获取 portfolio_id
	portfolioID, err := strconv.ParseUint(c.Query("portfolio_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio_id")
		return
	}

	var req service.SubmitRevisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	workflowID, err := h.svc.SubmitRevision(c.Request.Context(), userID, uint(novelID), uint(portfolioID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"workflow_id": workflowID})
}

// ExecuteRevision 确认执行变更（触发执行工作流）
// POST /api/v1/novels/:id/overview/revision/execute
func (h *OverviewHandler) ExecuteRevision(c *gin.Context) {
	userID := c.GetUint("user_id")
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	portfolioID, err := strconv.ParseUint(c.Query("portfolio_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio_id")
		return
	}

	var req service.ExecuteRevisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	workflowID, err := h.svc.ExecuteRevision(c.Request.Context(), userID, uint(novelID), uint(portfolioID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"workflow_id": workflowID})
}
