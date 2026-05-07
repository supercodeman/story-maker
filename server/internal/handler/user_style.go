// server/internal/handler/user_style.go
package handler

import (
	"errors"
	"strconv"

	"story-maker/server/internal/model"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserStyleHandler 用户风格请求处理层
type UserStyleHandler struct {
	svc *service.UserStyleService
}

// NewUserStyleHandler 创建 UserStyleHandler 实例
func NewUserStyleHandler(svc *service.UserStyleService) *UserStyleHandler {
	return &UserStyleHandler{svc: svc}
}

// List 获取当前用户的风格列表
// GET /api/v1/user-styles
func (h *UserStyleHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	styles, err := h.svc.List(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, styles)
}

// CreateRequest 创建风格请求
type CreateUserStyleRequest struct {
	Name              string `json:"name" binding:"required,max=100"`
	Description       string `json:"description" binding:"max=500"`
	NarrativeVoice    string `json:"narrative_voice" binding:"omitempty,oneof=first third_limited third_omniscient multi_pov"`
	Tone              string `json:"tone" binding:"omitempty,oneof=serious humorous lyrical sharp warm neutral"`
	LanguageLevel     string `json:"language_level" binding:"omitempty,oneof=literary standard colloquial web_novel"`
	ReferenceAuthors  string `json:"reference_authors" binding:"max=500"`
	ForbiddenPatterns string `json:"forbidden_patterns"`
	CustomRules       string `json:"custom_rules"`
	CustomPrompt      string `json:"custom_prompt"`
}

// Create 创建风格
// POST /api/v1/user-styles
func (h *UserStyleHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req CreateUserStyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	style := &model.UserStyle{
		UserID:            userID,
		Name:              req.Name,
		Description:       req.Description,
		NarrativeVoice:    req.NarrativeVoice,
		Tone:              req.Tone,
		LanguageLevel:     req.LanguageLevel,
		ReferenceAuthors:  req.ReferenceAuthors,
		ForbiddenPatterns: req.ForbiddenPatterns,
		CustomRules:       req.CustomRules,
		CustomPrompt:      req.CustomPrompt,
	}

	if err := h.svc.Create(style); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, style)
}

// UpdateRequest 更新风格请求
type UpdateUserStyleRequest struct {
	Name              string `json:"name" binding:"required,max=100"`
	Description       string `json:"description" binding:"max=500"`
	NarrativeVoice    string `json:"narrative_voice" binding:"omitempty,oneof=first third_limited third_omniscient multi_pov"`
	Tone              string `json:"tone" binding:"omitempty,oneof=serious humorous lyrical sharp warm neutral"`
	LanguageLevel     string `json:"language_level" binding:"omitempty,oneof=literary standard colloquial web_novel"`
	ReferenceAuthors  string `json:"reference_authors" binding:"max=500"`
	ForbiddenPatterns string `json:"forbidden_patterns"`
	CustomRules       string `json:"custom_rules"`
	CustomPrompt      string `json:"custom_prompt"`
}

// Update 更新风格
// PUT /api/v1/user-styles/:id
func (h *UserStyleHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid style id")
		return
	}

	var req UpdateUserStyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	style := &model.UserStyle{
		ID:                uint(id),
		Name:              req.Name,
		Description:       req.Description,
		NarrativeVoice:    req.NarrativeVoice,
		Tone:              req.Tone,
		LanguageLevel:     req.LanguageLevel,
		ReferenceAuthors:  req.ReferenceAuthors,
		ForbiddenPatterns: req.ForbiddenPatterns,
		CustomRules:       req.CustomRules,
		CustomPrompt:      req.CustomPrompt,
	}

	if err := h.svc.Update(style, userID); err != nil {
		if err.Error() == "forbidden" {
			Error(c, 403, "forbidden")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, 404, "style not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, style)
}

// Delete 删除风格
// DELETE /api/v1/user-styles/:id
func (h *UserStyleHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid style id")
		return
	}

	if err := h.svc.Delete(uint(id), userID); err != nil {
		if err.Error() == "forbidden" {
			Error(c, 403, "forbidden")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, 404, "style not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// AIGenerateRequest AI 生成风格请求
type AIGenerateRequest struct {
	Description string `json:"description" binding:"required,max=500"`
}

// AIGenerate AI 根据描述生成风格配置
// POST /api/v1/user-styles/ai-generate
func (h *UserStyleHandler) AIGenerate(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req AIGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	style, err := h.svc.AIGenerate(c.Request.Context(), userID, req.Description)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, style)
}

// BindStyleRequest 绑定用户风格请求
type BindStyleRequest struct {
	UserStyleID uint `json:"user_style_id" binding:"required"`
}

// BindStyle 将小说绑定到用户风格
// PUT /api/v1/novels/:id/bind-style
func (h *UserStyleHandler) BindStyle(c *gin.Context) {
	userID := c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req BindStyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 校验风格归属
	_, err = h.svc.Get(req.UserStyleID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, 404, "style not found")
			return
		}
		if err.Error() == "forbidden" {
			Error(c, 403, "style does not belong to you")
			return
		}
		InternalError(c, err.Error())
		return
	}

	// 更新 WritingStyle 的 bound_user_style_id
	if err := h.svc.BindToNovel(uint(novelID), req.UserStyleID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// UnbindStyle 解绑小说的用户风格
// DELETE /api/v1/novels/:id/bind-style
func (h *UserStyleHandler) UnbindStyle(c *gin.Context) {
	_ = c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	if err := h.svc.UnbindFromNovel(uint(novelID)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}
