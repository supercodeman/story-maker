// server/internal/handler/knowledge.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识库请求处理层
type KnowledgeHandler struct {
	svc *service.KnowledgeService
}

// NewKnowledgeHandler 创建 KnowledgeHandler 实例
func NewKnowledgeHandler(svc *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc}
}

// Create 创建知识条目
// POST /api/v1/novels/:id/knowledge
func (h *KnowledgeHandler) Create(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.CreateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	k, err := h.svc.Create(uint(novelID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, k)
}

// List 获取知识条目列表
// GET /api/v1/novels/:id/knowledge?category=xxx&status=xxx
func (h *KnowledgeHandler) List(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	category := c.Query("category")
	status := c.Query("status")

	items, err := h.svc.List(uint(novelID), category, status)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}

// Get 获取知识条目详情
// GET /api/v1/knowledge/:kid
func (h *KnowledgeHandler) Get(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	k, err := h.svc.Get(uint(kid))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, k)
}

// Update 更新知识条目
// PUT /api/v1/knowledge/:kid
func (h *KnowledgeHandler) Update(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	var req service.UpdateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	k, err := h.svc.Update(uint(kid), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, k)
}

// Delete 删除知识条目
// DELETE /api/v1/knowledge/:kid
func (h *KnowledgeHandler) Delete(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	if err := h.svc.Delete(uint(kid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Confirm 确认待审核条目
// POST /api/v1/knowledge/:kid/confirm
func (h *KnowledgeHandler) Confirm(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	if err := h.svc.Confirm(uint(kid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// BatchConfirm 批量确认小说下所有待审核条目
// POST /api/v1/novels/:id/knowledge/batch-confirm
func (h *KnowledgeHandler) BatchConfirm(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	if err := h.svc.BatchConfirm(uint(novelID)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Search 搜索知识条目
// GET /api/v1/novels/:id/knowledge/search?keyword=xxx
func (h *KnowledgeHandler) Search(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		BadRequest(c, "keyword is required")
		return
	}

	items, err := h.svc.Search(uint(novelID), keyword)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}

// Extract AI 提取知识条目
// POST /api/v1/novels/:id/knowledge/extract
func (h *KnowledgeHandler) Extract(c *gin.Context) {
	userID := c.GetUint("user_id")
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.ExtractKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	taskID, err := h.svc.ExtractFromChapter(c.Request.Context(), userID, uint(novelID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// ParseExtract 解析 AI 提取结果并写入知识库
// POST /api/v1/novels/:id/knowledge/parse-extract
func (h *KnowledgeHandler) ParseExtract(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req struct {
		ChapterID uint `json:"chapter_id" binding:"required"`
		TaskID    uint `json:"task_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	items, err := h.svc.ParseExtractResult(uint(novelID), req.ChapterID, req.TaskID)
	if err != nil {
		if err.Error() == "task is not completed" {
			Error(c, http.StatusConflict, err.Error())
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}
