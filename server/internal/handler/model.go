// server/internal/handler/model.go
package handler

import (
	"strconv"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// ModelHandler 模型管理 Handler
type ModelHandler struct {
	modelRegistry *service.ModelRegistryService
	providers     map[string]agent.AIProvider
}

// NewModelHandler 创建 ModelHandler 实例
func NewModelHandler(modelRegistry *service.ModelRegistryService, providers map[string]agent.AIProvider) *ModelHandler {
	return &ModelHandler{
		modelRegistry: modelRegistry,
		providers:     providers,
	}
}

// GetAvailableModels GET /api/v1/models/available?capability=text_gen
func (h *ModelHandler) GetAvailableModels(c *gin.Context) {
	userID := c.GetUint("user_id")
	capability := c.Query("capability")

	models, err := h.modelRegistry.GetAvailableModels(c.Request.Context(), userID, capability)
	if err != nil {
		InternalError(c, "failed to get available models")
		return
	}

	Success(c, models)
}

// GetModelStatus GET /api/v1/models/status（管理员接口，返回完整调试信息）
func (h *ModelHandler) GetModelStatus(c *gin.Context) {
	details, err := h.modelRegistry.GetAllModelStatus(c.Request.Context())
	if err != nil {
		InternalError(c, "failed to get model status")
		return
	}

	Success(c, details)
}

// TriggerHealthCheck POST /api/v1/models/check（手动触发探测）
func (h *ModelHandler) TriggerHealthCheck(c *gin.Context) {
	if err := h.modelRegistry.HealthCheckAll(c.Request.Context(), h.providers); err != nil {
		InternalError(c, "health check failed")
		return
	}

	Success(c, gin.H{"message": "health check completed"})
}

// AddModel POST /api/v1/models（新增模型）
func (h *ModelHandler) AddModel(c *gin.Context) {
	var req service.AddModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request body")
		return
	}
	if req.Provider == "" || req.Capability == "" {
		BadRequest(c, "provider and capability are required")
		return
	}

	if err := h.modelRegistry.AddModel(c.Request.Context(), &req); err != nil {
		InternalError(c, "failed to add model")
		return
	}

	Success(c, gin.H{"message": "model added"})
}

// DeleteModel DELETE /api/v1/models/:id（删除模型）
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid model id")
		return
	}

	if err := h.modelRegistry.DeleteModel(c.Request.Context(), uint(id)); err != nil {
		InternalError(c, "failed to delete model")
		return
	}

	Success(c, gin.H{"message": "model deleted"})
}

// TestModel POST /api/v1/models/test（测试单个模型）
func (h *ModelHandler) TestModel(c *gin.Context) {
	var req struct {
		Provider   string `json:"provider"`
		ModelName  string `json:"model_name"`
		Capability string `json:"capability"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request body")
		return
	}
	if req.Provider == "" || req.Capability == "" {
		BadRequest(c, "provider and capability are required")
		return
	}

	result, err := h.modelRegistry.TestSingleModel(c.Request.Context(), req.Provider, req.ModelName, req.Capability, h.providers)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// UpdatePriority PUT /api/v1/models/:id/priority（更新模型优先级）
func (h *ModelHandler) UpdatePriority(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid model id")
		return
	}

	var req struct {
		Priority int `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid request body")
		return
	}

	if err := h.modelRegistry.UpdateModelPriority(c.Request.Context(), uint(id), req.Priority); err != nil {
		InternalError(c, "failed to update priority")
		return
	}

	Success(c, gin.H{"message": "priority updated"})
}
