// server/internal/handler/writing_style.go
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"story-maker/server/internal/model"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WritingStyleHandler 写作风格请求处理层
type WritingStyleHandler struct {
	svc *service.WritingStyleService
}

// NewWritingStyleHandler 创建 WritingStyleHandler 实例
func NewWritingStyleHandler(svc *service.WritingStyleService) *WritingStyleHandler {
	return &WritingStyleHandler{svc: svc}
}

// ========== WritingStyle 路由 ==========

// GetStyle 获取小说的写作风格配置
// GET /api/v1/novels/:id/writing-style
func (h *WritingStyleHandler) GetStyle(c *gin.Context) {
	_ = c.GetUint("user_id") // 鉴权：确保已登录

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	style, err := h.svc.GetByNovelID(uint(novelID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不存在时返回空对象（前端可据此判断是否已配置）
			Success(c, nil)
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, style)
}

// UpsertStyleRequest 创建/更新写作风格请求
type UpsertStyleRequest struct {
	NarrativeVoice    string `json:"narrative_voice" binding:"omitempty,oneof=first third_limited third_omniscient multi_pov"`
	Tone              string `json:"tone" binding:"omitempty,oneof=serious humorous lyrical sharp warm neutral"`
	LanguageLevel     string `json:"language_level" binding:"omitempty,oneof=literary standard colloquial web_novel"`
	ReferenceAuthors  string `json:"reference_authors" binding:"max=500"`
	ForbiddenPatterns string `json:"forbidden_patterns"`
	CustomRules       string `json:"custom_rules"`
}

// UpsertStyle 创建或更新写作风格
// PUT /api/v1/novels/:id/writing-style
func (h *WritingStyleHandler) UpsertStyle(c *gin.Context) {
	_ = c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req UpsertStyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	style := &model.WritingStyle{
		NovelID:           uint(novelID),
		NarrativeVoice:    req.NarrativeVoice,
		Tone:              req.Tone,
		LanguageLevel:     req.LanguageLevel,
		ReferenceAuthors:  req.ReferenceAuthors,
		ForbiddenPatterns: req.ForbiddenPatterns,
		CustomRules:       req.CustomRules,
	}

	// 设置默认值
	if style.NarrativeVoice == "" {
		style.NarrativeVoice = model.NarrativeThirdLimited
	}
	if style.Tone == "" {
		style.Tone = model.ToneNeutral
	}
	if style.LanguageLevel == "" {
		style.LanguageLevel = model.LangStandard
	}

	if err := h.svc.Upsert(style); err != nil {
		InternalError(c, err.Error())
		return
	}

	// Upsert 后重新查询，确保返回正确的 ID
	saved, err := h.svc.GetByNovelID(uint(novelID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, saved)
}

// DeleteStyle 删除写作风格配置
// DELETE /api/v1/novels/:id/writing-style
func (h *WritingStyleHandler) DeleteStyle(c *gin.Context) {
	_ = c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	if err := h.svc.Delete(uint(novelID)); err != nil {
		InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

// ========== ScenePreset 路由 ==========

// ListPresets 获取小说下的所有场景预设
// GET /api/v1/novels/:id/scene-presets
func (h *WritingStyleHandler) ListPresets(c *gin.Context) {
	_ = c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	presets, err := h.svc.ListPresets(uint(novelID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, presets)
}

// CreatePresetRequest 创建场景预设请求
type CreatePresetRequest struct {
	SceneType string `json:"scene_type" binding:"required,oneof=battle dialogue psychology environment flashback daily"`
	Name      string `json:"name" binding:"required,max=100"`
	Rules     string `json:"rules" binding:"required"`
}

// CreatePreset 创建场景预设
// POST /api/v1/novels/:id/scene-presets
func (h *WritingStyleHandler) CreatePreset(c *gin.Context) {
	_ = c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req CreatePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	preset := &model.ScenePreset{
		NovelID:   uint(novelID),
		SceneType: req.SceneType,
		Name:      req.Name,
		Rules:     req.Rules,
	}

	if err := h.svc.CreatePreset(preset); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, preset)
}

// UpdatePresetRequest 更新场景预设请求（使用指针区分"未传"和"传了空值"）
type UpdatePresetRequest struct {
	SceneType *string `json:"scene_type" binding:"omitempty,oneof=battle dialogue psychology environment flashback daily"`
	Name      *string `json:"name" binding:"omitempty,max=100"`
	Rules     *string `json:"rules"`
}

// UpdatePreset 更新场景预设
// PUT /api/v1/novels/:id/scene-presets/:pid
func (h *WritingStyleHandler) UpdatePreset(c *gin.Context) {
	_ = c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	presetID, err := strconv.ParseUint(c.Param("pid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid preset id")
		return
	}

	preset, err := h.svc.GetPreset(uint(presetID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "preset not found"})
			return
		}
		InternalError(c, err.Error())
		return
	}

	// 校验 preset 归属于当前 novel
	if preset.NovelID != uint(novelID) {
		c.JSON(http.StatusForbidden, gin.H{"code": 1, "message": "preset does not belong to this novel"})
		return
	}

	var req UpdatePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 使用指针判断：nil 表示未传，非 nil 表示要更新（包括空字符串）
	if req.SceneType != nil {
		preset.SceneType = *req.SceneType
	}
	if req.Name != nil {
		preset.Name = *req.Name
	}
	if req.Rules != nil {
		preset.Rules = *req.Rules
	}

	if err := h.svc.UpdatePreset(preset); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, preset)
}

// DeletePreset 删除场景预设
// DELETE /api/v1/novels/:id/scene-presets/:pid
func (h *WritingStyleHandler) DeletePreset(c *gin.Context) {
	_ = c.GetUint("user_id")

	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	presetID, err := strconv.ParseUint(c.Param("pid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid preset id")
		return
	}

	// 校验 preset 归属
	preset, err := h.svc.GetPreset(uint(presetID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "preset not found"})
			return
		}
		InternalError(c, err.Error())
		return
	}
	if preset.NovelID != uint(novelID) {
		c.JSON(http.StatusForbidden, gin.H{"code": 1, "message": "preset does not belong to this novel"})
		return
	}

	if err := h.svc.DeletePreset(uint(presetID)); err != nil {
		InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}
