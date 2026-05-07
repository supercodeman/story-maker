// server/internal/handler/api_key.go
package handler

import (
	"net/http"
	"strconv"

	"story-maker/server/internal/service"
	"github.com/gin-gonic/gin"
)

// APIKeyHandler API Key 请求处理器
type APIKeyHandler struct {
	keyService *service.APIKeyService
}

// NewAPIKeyHandler 创建 APIKeyHandler 实例
func NewAPIKeyHandler(keyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{keyService: keyService}
}

// CreateKeyRequest 创建 API Key 请求
type CreateKeyRequest struct {
	Provider string `json:"provider" binding:"required"`
	KeyValue string `json:"key_value" binding:"required"`
}

// UpdateKeyRequest 更新 API Key 请求
type UpdateKeyRequest struct {
	KeyValue  string `json:"key_value"`
	IsDefault *bool  `json:"is_default"`
}

// ListKeys 获取用户的 API Key 列表
// GET /api/v1/apikeys
func (h *APIKeyHandler) ListKeys(c *gin.Context) {
	userID := c.GetUint("user_id")

	keys, err := h.keyService.GetKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// CreateKey 创建 API Key
// POST /api/v1/apikeys
func (h *APIKeyHandler) CreateKey(c *gin.Context) {
	var req CreateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	key, err := h.keyService.CreateKey(c.Request.Context(), userID, req.Provider, req.KeyValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, key)
}

// UpdateKey 更新 API Key
// PUT /api/v1/apikeys/:id
func (h *APIKeyHandler) UpdateKey(c *gin.Context) {
	keyIDStr := c.Param("id")
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	var req UpdateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	if err := h.keyService.UpdateKey(c.Request.Context(), uint(keyID), userID, req.KeyValue, req.IsDefault); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key updated"})
}

// DeleteKey 删除 API Key
// DELETE /api/v1/apikeys/:id
func (h *APIKeyHandler) DeleteKey(c *gin.Context) {
	keyIDStr := c.Param("id")
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	userID := c.GetUint("user_id")

	if err := h.keyService.DeleteKey(c.Request.Context(), uint(keyID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}
