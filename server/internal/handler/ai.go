// server/internal/handler/ai.go
package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"ai-curton/server/internal/service"
	"github.com/gin-gonic/gin"
)

// AIHandler AI 请求处理器
type AIHandler struct {
	aiService *service.AIService
}

// NewAIHandler 创建 AIHandler 实例
func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// ChatMessage 对话历史消息（前端传入）
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateTextRequest 文本生成请求
type GenerateTextRequest struct {
	PortfolioID uint          `json:"portfolio_id"`
	ModelName   string        `json:"model_name"`
	Prompt      string        `json:"prompt" binding:"required"`
	History     []ChatMessage `json:"history"` // 多轮对话历史
}

// GenerateImageRequest 图像生成请求
type GenerateImageRequest struct {
	PortfolioID uint   `json:"portfolio_id"`
	ModelName   string `json:"model_name"`
	Prompt      string `json:"prompt" binding:"required"`
}

// AdjustCharacterRequest 角色调整请求
type AdjustCharacterRequest struct {
	PortfolioID uint   `json:"portfolio_id"`
	ModelName   string `json:"model_name"`
	Prompt      string `json:"prompt" binding:"required"`
}

// GenerateAudioRequest 音频生成请求
type GenerateAudioRequest struct {
	PortfolioID uint    `json:"portfolio_id"`
	ChapterID   uint    `json:"chapter_id"`
	Text        string  `json:"text" binding:"required"`
	VoiceID     string  `json:"voice_id"`
	Speed       float64 `json:"speed"`
	Emotion     string  `json:"emotion"`
}

// GenerateVideoRequest 视频生成请求
type GenerateVideoRequest struct {
	PortfolioID uint   `json:"portfolio_id"`
	ChapterID   uint   `json:"chapter_id"`
	ModelName   string `json:"model_name"`
	Prompt      string `json:"prompt" binding:"required"`
}

// TaskResponse 任务响应
type TaskResponse struct {
	TaskID uint   `json:"task_id"`
	Status string `json:"status"`
}

// GenerateText 文本生成接口
// POST /api/v1/ai/text/generate
func (h *AIHandler) GenerateText(c *gin.Context) {
	var req GenerateTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	// 转换 history
	var history []service.ChatMessage
	for _, h := range req.History {
		history = append(history, service.ChatMessage{Role: h.Role, Content: h.Content})
	}

	taskID, err := h.aiService.SubmitTextTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt, history)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// GenerateImage 图像生成接口
// POST /api/v1/ai/image/generate
func (h *AIHandler) GenerateImage(c *gin.Context) {
	var req GenerateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitImageTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// AdjustCharacter 角色调整接口
// POST /api/v1/ai/character/adjust
func (h *AIHandler) AdjustCharacter(c *gin.Context) {
	var req AdjustCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitCharacterAdjustTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// GenerateAudio 音频生成接口
// POST /api/v1/ai/audio/generate
func (h *AIHandler) GenerateAudio(c *gin.Context) {
	var req GenerateAudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	// 将参数序列化为 JSON 作为 prompt 传入
	promptJSON, _ := json.Marshal(map[string]interface{}{
		"text":       req.Text,
		"voice_id":   req.VoiceID,
		"speed":      req.Speed,
		"emotion":    req.Emotion,
		"chapter_id": req.ChapterID,
	})

	taskID, err := h.aiService.SubmitAudioTask(c.Request.Context(), userID, req.PortfolioID, string(promptJSON))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// GenerateVideo 视频生成接口
// POST /api/v1/ai/video/generate
func (h *AIHandler) GenerateVideo(c *gin.Context) {
	var req GenerateVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitVideoTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// GetTask 获取任务详情
// GET /api/v1/ai/tasks/:id
func (h *AIHandler) GetTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "invalid task id")
		return
	}

	userID := c.GetUint("user_id")

	task, err := h.aiService.GetTask(c.Request.Context(), uint(taskID), userID)
	if err != nil {
		Error(c, 404, err.Error())
		return
	}

	Success(c, task)
}

// ListTasks 获取任务列表
// GET /api/v1/ai/tasks?portfolio_id=x&task_types=type1,type2
func (h *AIHandler) ListTasks(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 解析查询参数
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

	// 解析 task_types 过滤参数
	var taskTypes []string
	if tt := c.Query("task_types"); tt != "" {
		for _, t := range strings.Split(tt, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				taskTypes = append(taskTypes, t)
			}
		}
	}

	// 解析 butler_session_id 过滤参数
	butlerSessionID := c.Query("butler_session_id")

	// 解析 novel_id 过滤参数
	var novelID uint
	if nidStr := c.Query("novel_id"); nidStr != "" {
		nid, err := strconv.ParseUint(nidStr, 10, 32)
		if err == nil {
			novelID = uint(nid)
		}
	}

	tasks, total, err := h.aiService.ListTasks(c.Request.Context(), userID, portfolioID, page, pageSize, taskTypes, butlerSessionID, novelID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// CancelTask 取消任务
// DELETE /api/v1/ai/tasks/:id
func (h *AIHandler) CancelTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "invalid task id")
		return
	}

	userID := c.GetUint("user_id")

	if err := h.aiService.CancelTask(c.Request.Context(), uint(taskID), userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "task cancelled", nil)
}
