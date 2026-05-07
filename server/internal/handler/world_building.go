// server/internal/handler/world_building.go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"story-maker/server/internal/model"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// WorldBuildingHandler 世界构建请求处理层
type WorldBuildingHandler struct {
	svc *service.WorldBuildingService
}

// NewWorldBuildingHandler 创建 WorldBuildingHandler 实例
func NewWorldBuildingHandler(svc *service.WorldBuildingService) *WorldBuildingHandler {
	return &WorldBuildingHandler{svc: svc}
}

// StartPhase 启动世界构建阶段
// POST /api/v1/world-building/start
func (h *WorldBuildingHandler) StartPhase(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.WorldBuildingStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	taskID, err := h.svc.StartPhase(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// ReviewResult 触发审查（生成任务完成后调用）
// POST /api/v1/world-building/review
func (h *WorldBuildingHandler) ReviewResult(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		NovelID     uint   `json:"novel_id" binding:"required"`
		PortfolioID uint   `json:"portfolio_id" binding:"required"`
		Phase       string `json:"phase" binding:"required"`
		TaskID      uint   `json:"task_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	reviewTaskID, err := h.svc.ReviewPhaseResult(c.Request.Context(), userID, req.NovelID, req.PortfolioID, req.Phase, req.TaskID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"review_task_id": reviewTaskID})
}

// ProcessReview 处理审查结果（审查任务完成后调用）
// POST /api/v1/world-building/process-review
func (h *WorldBuildingHandler) ProcessReview(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		NovelID        uint                    `json:"novel_id" binding:"required"`
		Phase          string                  `json:"phase" binding:"required"`
		GenerateTaskID uint                    `json:"generate_task_id" binding:"required"`
		ReviewTaskID   uint                    `json:"review_task_id" binding:"required"`
		Config         *model.ReflectionConfig `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.ProcessReviewResult(c.Request.Context(), userID, req.NovelID, req.Phase, req.GenerateTaskID, req.ReviewTaskID, req.Config)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// Optimize 基于审查意见优化（用户确认继续优化时调用）
// POST /api/v1/world-building/optimize
func (h *WorldBuildingHandler) Optimize(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		NovelID     uint   `json:"novel_id" binding:"required"`
		PortfolioID uint   `json:"portfolio_id" binding:"required"`
		Phase       string `json:"phase" binding:"required"`
		PrevContent string `json:"prev_content" binding:"required"`
		ReviewJSON  string `json:"review_json" binding:"required"`
		MaxRounds   int    `json:"max_rounds"`
		Threshold   float64 `json:"threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 解析审查结果
	var reviewResult model.ReviewResult
	if err := json.Unmarshal([]byte(req.ReviewJSON), &reviewResult); err != nil {
		BadRequest(c, "invalid review_json: "+err.Error())
		return
	}

	cfg := &model.ReflectionConfig{
		MaxRounds: req.MaxRounds,
		Threshold: req.Threshold,
	}

	taskID, err := h.svc.OptimizePhase(c.Request.Context(), userID, req.NovelID, req.PortfolioID, req.Phase, req.PrevContent, &reviewResult, cfg)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// AcceptResult 接受阶段结果，写入数据库
// POST /api/v1/world-building/accept
func (h *WorldBuildingHandler) AcceptResult(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		NovelID uint    `json:"novel_id" binding:"required"`
		Phase   string  `json:"phase" binding:"required"`
		Content string  `json:"content" binding:"required"`
		Score   float64 `json:"score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.AcceptPhaseResult(c.Request.Context(), userID, req.NovelID, req.Phase, req.Content, req.Score); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "accepted"})
}

// GetStatus 获取阶段状态
// GET /api/v1/world-building/status?novel_id=x&phase=y
func (h *WorldBuildingHandler) GetStatus(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Query("novel_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel_id")
		return
	}
	phase := c.Query("phase")
	if phase == "" {
		BadRequest(c, "phase is required")
		return
	}

	status, err := h.svc.GetPhaseStatus(uint(novelID), phase)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, status)
}

// GetSummary 获取世界构建概览
// GET /api/v1/world-building/summary?novel_id=x
func (h *WorldBuildingHandler) GetSummary(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Query("novel_id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel_id")
		return
	}

	summary, err := h.svc.GetWorldBuildingSummary(uint(novelID))
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, summary)
}
