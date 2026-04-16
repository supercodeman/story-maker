// server/internal/handler/hit_analysis.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// HitAnalysisHandler 爆款拆解请求处理层
type HitAnalysisHandler struct {
	svc *service.HitAnalysisService
}

// NewHitAnalysisHandler 创建 HitAnalysisHandler 实例
func NewHitAnalysisHandler(svc *service.HitAnalysisService) *HitAnalysisHandler {
	return &HitAnalysisHandler{svc: svc}
}

// Submit 提交爆款拆解
// POST /api/v1/hit-analysis
func (h *HitAnalysisHandler) Submit(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.SubmitHitAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ha, err := h.svc.Submit(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, ha)
}

// List 获取拆解记录列表
// GET /api/v1/hit-analysis
func (h *HitAnalysisHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	list, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, list)
}

// Get 获取拆解记录详情
// GET /api/v1/hit-analysis/:id
func (h *HitAnalysisHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid hit analysis id")
		return
	}

	ha, err := h.svc.Get(c.Request.Context(), uint(id), userID)
	if err != nil {
		if err.Error() == "permission denied" {
			Error(c, http.StatusForbidden, err.Error())
			return
		}
		Error(c, http.StatusNotFound, "hit analysis not found")
		return
	}
	Success(c, ha)
}

// Delete 删除拆解记录
// DELETE /api/v1/hit-analysis/:id
func (h *HitAnalysisHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid hit analysis id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id), userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}
