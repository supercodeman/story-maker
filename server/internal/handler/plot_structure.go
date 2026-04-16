// server/internal/handler/plot_structure.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// PlotStructureHandler 剧情结构模板请求处理层
type PlotStructureHandler struct {
	svc *service.PlotStructureService
}

// NewPlotStructureHandler 创建 PlotStructureHandler 实例
func NewPlotStructureHandler(svc *service.PlotStructureService) *PlotStructureHandler {
	return &PlotStructureHandler{svc: svc}
}

// List 获取模板列表（系统 + 当前用户自定义）
// GET /api/v1/plot-templates
func (h *PlotStructureHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	templates, err := h.svc.List(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, templates)
}

// Get 获取模板详情
// GET /api/v1/plot-templates/:id
func (h *PlotStructureHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid template id")
		return
	}

	tpl, err := h.svc.Get(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, "template not found")
		return
	}
	Success(c, tpl)
}

// Create 创建用户自定义模板
// POST /api/v1/plot-templates
func (h *PlotStructureHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.CreatePlotTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	tpl, err := h.svc.Create(userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, tpl)
}

// AIGenerate AI 辅助生成模板
// POST /api/v1/plot-templates/ai-generate
func (h *PlotStructureHandler) AIGenerate(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.AIGenerateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 从 query 或 body 获取 portfolio_id
	var portfolioID uint
	if pidStr := c.Query("portfolio_id"); pidStr != "" {
		pid, _ := strconv.ParseUint(pidStr, 10, 64)
		portfolioID = uint(pid)
	}

	taskID, err := h.svc.AIGenerate(c.Request.Context(), userID, portfolioID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"task_id": taskID})
}

// Update 更新用户自定义模板
// PUT /api/v1/plot-templates/:id
func (h *PlotStructureHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid template id")
		return
	}

	var req service.UpdatePlotTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	tpl, err := h.svc.Update(userID, uint(id), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, tpl)
}

// Delete 删除用户自定义模板
// DELETE /api/v1/plot-templates/:id
func (h *PlotStructureHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid template id")
		return
	}

	if err := h.svc.Delete(userID, uint(id)); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}
