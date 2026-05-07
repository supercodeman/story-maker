// server/internal/handler/novel.go
package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"story-maker/server/internal/middleware"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// NovelHandler 小说工坊请求处理层
type NovelHandler struct {
	svc       *service.NovelService
	searchSvc *service.NovelSearchService
}

// NewNovelHandler 创建 NovelHandler 实例
func NewNovelHandler(svc *service.NovelService, searchSvc *service.NovelSearchService) *NovelHandler {
	return &NovelHandler{svc: svc, searchSvc: searchSvc}
}

// ========== Novel 路由 ==========

// CreateNovel 创建小说
// POST /api/v1/novels
func (h *NovelHandler) CreateNovel(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.CreateNovelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	novel, err := h.svc.CreateNovel(userID, req.PortfolioID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, novel)
}

// ListNovels 获取小说列表
// GET /api/v1/novels?portfolio_id=x&source=butler
func (h *NovelHandler) ListNovels(c *gin.Context) {
	pidStr := c.Query("portfolio_id")
	if pidStr == "" {
		BadRequest(c, "portfolio_id is required")
		return
	}

	pid, err := strconv.ParseUint(pidStr, 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio_id")
		return
	}

	source := c.Query("source")

	novels, err := h.svc.ListNovels(uint(pid), source)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, novels)
}

// GetNovel 获取小说详情
// GET /api/v1/novels/:id
func (h *NovelHandler) GetNovel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	novel, err := h.svc.GetNovel(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, "novel not found")
		return
	}

	Success(c, novel)
}

// UpdateNovel 更新小说
// PUT /api/v1/novels/:id
func (h *NovelHandler) UpdateNovel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.UpdateNovelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	novel, err := h.svc.UpdateNovel(uint(id), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, novel)
}

// GetTokenUsage 获取小说 token 使用情况
// GET /api/v1/novels/:id/token-usage
func (h *NovelHandler) GetTokenUsage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	usage, err := h.svc.GetTokenUsage(c.Request.Context(), uint(id))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, usage)
}

// UpdateTokenBudget 更新小说 token 预算
// PUT /api/v1/novels/:id/token-budget
func (h *NovelHandler) UpdateTokenBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req struct {
		TokenBudget int `json:"token_budget"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.UpdateTokenBudget(uint(id), req.TokenBudget); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"token_budget": req.TokenBudget})
}

// DeleteNovel 删除小说
// DELETE /api/v1/novels/:id
func (h *NovelHandler) DeleteNovel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	if err := h.svc.DeleteNovel(uint(id)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "novel deleted"})
}

// ========== Chapter 路由 ==========

// CreateChapter 创建章节
// POST /api/v1/novels/:id/chapters
func (h *NovelHandler) CreateChapter(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.CreateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	chapter, err := h.svc.CreateChapter(userID, uint(novelID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, chapter)
}

// ListChapters 获取章节列表
// GET /api/v1/novels/:id/chapters
func (h *NovelHandler) ListChapters(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	chapters, err := h.svc.ListChapters(uint(novelID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, chapters)
}

// UpdateChapter 更新章节
// PUT /api/v1/chapters/:id
func (h *NovelHandler) UpdateChapter(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	var req service.UpdateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	chapter, err := h.svc.UpdateChapter(userID, uint(id), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, chapter)
}

// DeleteChapter 删除章节
// DELETE /api/v1/chapters/:id
func (h *NovelHandler) DeleteChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	if err := h.svc.DeleteChapter(uint(id)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "chapter deleted"})
}

// ReorderChapters 重新排序章节
// PUT /api/v1/novels/:id/chapters/reorder
func (h *NovelHandler) ReorderChapters(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.ReorderChaptersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.ReorderChapters(uint(novelID), &req); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "chapters reordered"})
}

// ========== AI 操作路由 ==========

// ChapterAIAction 对章节执行 AI 操作
// POST /api/v1/chapters/:id/ai
func (h *NovelHandler) ChapterAIAction(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	var req service.ChapterAIActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	taskID, err := h.svc.ChapterAIAction(c.Request.Context(), userID, uint(id), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// AcceptAIResult 采纳 AI 结果
// POST /api/v1/chapters/:id/accept
func (h *NovelHandler) AcceptAIResult(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	var req service.AcceptAIResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.AcceptAIResult(c.Request.Context(), userID, uint(id), req.TaskID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "AI result accepted"})
}

// RejectAIResult 拒绝 AI 结果
// POST /api/v1/chapters/:id/reject
func (h *NovelHandler) RejectAIResult(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	var req service.AcceptAIResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.RejectAIResult(c.Request.Context(), userID, uint(id), req.TaskID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "AI result rejected"})
}

// ========== 版本管理路由 ==========

// ListVersions 获取章节版本历史
// GET /api/v1/chapters/:id/versions
func (h *NovelHandler) ListVersions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	versions, err := h.svc.ListVersions(uint(id))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, versions)
}

// RevertToVersion 回退到指定版本
// POST /api/v1/chapters/:id/revert
func (h *NovelHandler) RevertToVersion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	var req service.RevertVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.svc.RevertToVersion(uint(id), req.VersionID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "reverted to version"})
}

// ========== 大纲生成路由 ==========

// GenerateOutline 生成小说大纲
// POST /api/v1/outline/generate
func (h *NovelHandler) GenerateOutline(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.GenerateOutlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	taskID, err := h.svc.GenerateOutline(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// AdoptOutline 采用大纲，创建小说和章节
// POST /api/v1/outline/adopt
func (h *NovelHandler) AdoptOutline(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.AdoptOutlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	novel, err := h.svc.AdoptOutline(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, novel)
}

// OutlineChapterAI 大纲页面章节级 AI 操作
// POST /api/v1/outline/chapter-ai
func (h *NovelHandler) OutlineChapterAI(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.OutlineChapterAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	taskID, err := h.svc.OutlineChapterAIAction(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// ExpandChapters 扩写章节目录
// POST /api/v1/novels/:id/expand-chapters
func (h *NovelHandler) ExpandChapters(c *gin.Context) {
	userID := c.GetUint("user_id")
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.ExpandChaptersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	req.NovelID = uint(novelID)

	taskID, err := h.svc.ExpandChapters(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// SearchNovels 搜索热门小说（多引擎降级）
// GET /api/v1/outline/search-novels?keyword=xxx
func (h *NovelHandler) SearchNovels(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		BadRequest(c, "keyword is required")
		return
	}

	results, source, warning := h.searchSvc.Search(c.Request.Context(), keyword, 5)

	resp := gin.H{"results": results}
	if source != "" {
		resp["source"] = source
	}
	if warning != "" {
		resp["warning"] = warning
	}
	Success(c, resp)
}

// TriggerFactColdStart 触发全量冷启动事实采集
// POST /api/v1/novels/:id/facts/cold-start
func (h *NovelHandler) TriggerFactColdStart(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	userID := c.GetUint("user_id")

	// 异步执行，立即返回
	go func() {
		ctx := context.Background()
		if err := h.svc.TriggerFactColdStart(ctx, uint(novelID), userID); err != nil {
			log.Printf("[handler] 全量冷启动失败: novel_id=%d err=%v", novelID, err)
		}
	}()

	Success(c, gin.H{"message": "冷启动采集已触发"})
}

// RepairButlerLinks 修复历史管家任务的 novel_id 关联
// POST /api/v1/novels/repair-butler
func (h *NovelHandler) RepairButlerLinks(c *gin.Context) {
	var req struct {
		PortfolioID uint `json:"portfolio_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	repaired, err := h.svc.RepairButlerNovelLinks(c.Request.Context(), req.PortfolioID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"repaired": repaired})
}
