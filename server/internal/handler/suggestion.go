// server/internal/handler/suggestion.go
package handler

import (
	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// SuggestionHandler 联想 API 请求处理层
type SuggestionHandler struct {
	svc *service.SuggestionService
}

// NewSuggestionHandler 创建 SuggestionHandler 实例
func NewSuggestionHandler(svc *service.SuggestionService) *SuggestionHandler {
	return &SuggestionHandler{svc: svc}
}

// Suggest 生成联想文本
// POST /api/v1/novels/:id/suggest
func (h *SuggestionHandler) Suggest(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.SuggestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	suggestion, err := h.svc.Suggest(c.Request.Context(), userID, req.NovelID, req.PrecedingText)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"suggestion": suggestion})
}
