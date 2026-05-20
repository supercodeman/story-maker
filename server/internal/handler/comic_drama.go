// server/internal/handler/comic_drama.go
package handler

import (
	"strconv"

	"story-maker/server/internal/model"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// ComicDramaHandler 漫剧 HTTP 处理器
type ComicDramaHandler struct {
	svc *service.ComicDramaService
}

func NewComicDramaHandler(svc *service.ComicDramaService) *ComicDramaHandler {
	return &ComicDramaHandler{svc: svc}
}

type createComicDramaReq struct {
	NovelID   uint   `json:"novel_id" binding:"required"`
	ChapterID uint   `json:"chapter_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Config    string `json:"config"`
}

type updateConfigReq struct {
	Config string `json:"config" binding:"required"`
}

type updateScriptReq struct {
	Scripts []*model.ComicScript `json:"scripts" binding:"required"`
}

type updateStoryboardShotReq struct {
	Updates map[string]interface{} `json:"updates" binding:"required"`
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	raw := c.Param(name)
	val, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}

// Create 创建漫剧
func (h *ComicDramaHandler) Create(c *gin.Context) {
	var req createComicDramaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	userID := c.GetUint("user_id")
	drama, err := h.svc.CreateComicDrama(c.Request.Context(), userID, &service.CreateComicDramaReq{
		NovelID:   req.NovelID,
		ChapterID: req.ChapterID,
		Title:     req.Title,
		Config:    req.Config,
	})
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, drama)
}

// Get 获取漫剧详情
func (h *ComicDramaHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	drama, err := h.svc.GetComicDrama(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, drama)
}

// List 获取漫剧列表
func (h *ComicDramaHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := h.svc.ListComicDramas(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"list": list, "total": total, "page": page})
}

// Delete 删除漫剧
func (h *ComicDramaHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.DeleteComicDrama(c.Request.Context(), id, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

// UpdateConfig 更新漫剧配置
func (h *ComicDramaHandler) UpdateConfig(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	var req updateConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.UpdateConfig(c.Request.Context(), id, userID, req.Config); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

// StartPipeline 启动漫剧生成流水线
func (h *ComicDramaHandler) StartPipeline(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.StartPipeline(c.Request.Context(), id, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "pipeline started"})
}

// AdvanceStage 手动推进到下一阶段
func (h *ComicDramaHandler) AdvanceStage(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.AdvanceStage(c.Request.Context(), id, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "stage advanced"})
}

// RetryFailed 重试失败的任务
func (h *ComicDramaHandler) RetryFailed(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.RetryFailed(c.Request.Context(), id, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "retry submitted"})
}

// GetScript 获取剧本
func (h *ComicDramaHandler) GetScript(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	script, err := h.svc.GetScript(c.Request.Context(), dramaID, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, script)
}

// UpdateScript 更新剧本
func (h *ComicDramaHandler) UpdateScript(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	var req updateScriptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.UpdateScript(c.Request.Context(), dramaID, userID, req.Scripts); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

// ApproveScript 审核通过剧本
func (h *ComicDramaHandler) ApproveScript(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.ApproveScript(c.Request.Context(), dramaID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "script approved"})
}

// GetStoryboard 获取分镜列表
func (h *ComicDramaHandler) GetStoryboard(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	storyboards, err := h.svc.GetStoryboard(c.Request.Context(), dramaID, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, storyboards)
}

// UpdateStoryboardShot 更新单个分镜
func (h *ComicDramaHandler) UpdateStoryboardShot(c *gin.Context) {
	shotID, err := parseUintParam(c, "shot_id")
	if err != nil {
		BadRequest(c, "invalid shot_id")
		return
	}
	var req updateStoryboardShotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.UpdateStoryboardShot(c.Request.Context(), shotID, userID, req.Updates); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, nil)
}

// ApproveStoryboard 审核通过分镜
func (h *ComicDramaHandler) ApproveStoryboard(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.ApproveStoryboard(c.Request.Context(), dramaID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "storyboard approved"})
}

// GetCharacters 获取角色定妆照列表
func (h *ComicDramaHandler) GetCharacters(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	characters, err := h.svc.GetCharacters(c.Request.Context(), dramaID, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, characters)
}

// RegenerateCharacter 重新生成单个角色定妆照
func (h *ComicDramaHandler) RegenerateCharacter(c *gin.Context) {
	charID, err := parseUintParam(c, "char_id")
	if err != nil {
		BadRequest(c, "invalid char_id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.RegenerateCharacter(c.Request.Context(), charID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "regeneration submitted"})
}

// ApproveCharacters 审核通过角色定妆照
func (h *ComicDramaHandler) ApproveCharacters(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.ApproveCharacters(c.Request.Context(), dramaID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "characters approved"})
}

// GetSegments 获取合成分段列表
func (h *ComicDramaHandler) GetSegments(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	segments, err := h.svc.GetSegments(c.Request.Context(), dramaID, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, segments)
}

// TriggerCompose 触发最终合成
func (h *ComicDramaHandler) TriggerCompose(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	if err := h.svc.TriggerCompose(c.Request.Context(), dramaID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "compose triggered"})
}

// GetDownloadURL 获取成品下载链接
func (h *ComicDramaHandler) GetDownloadURL(c *gin.Context) {
	dramaID, err := parseUintParam(c, "id")
	if err != nil {
		BadRequest(c, "invalid id")
		return
	}
	userID := c.GetUint("user_id")
	url, err := h.svc.GetDownloadURL(c.Request.Context(), dramaID, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"url": url})
}
