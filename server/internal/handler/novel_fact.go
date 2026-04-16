// server/internal/handler/novel_fact.go
package handler

import (
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// NovelFactHandler 小说记忆事实请求处理层
type NovelFactHandler struct {
	svc *service.NovelFactService
}

// NewNovelFactHandler 创建 NovelFactHandler 实例
func NewNovelFactHandler(svc *service.NovelFactService) *NovelFactHandler {
	return &NovelFactHandler{svc: svc}
}

// List 获取事实列表
// GET /api/v1/novels/:id/facts?fact_type=xxx
func (h *NovelFactHandler) List(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	factType := c.Query("fact_type")

	items, err := h.svc.ListFacts(uint(novelID), factType)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}

// Get 获取事实详情
// GET /api/v1/facts/:fid
func (h *NovelFactHandler) Get(c *gin.Context) {
	fid, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid fact id")
		return
	}

	fact, err := h.svc.GetFact(uint(fid))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, fact)
}

// Create 创建事实
// POST /api/v1/novels/:id/facts
func (h *NovelFactHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.CreateFactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	fact, err := h.svc.CreateFact(c.Request.Context(), uint(novelID), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, fact)
}

// Update 更新事实
// PUT /api/v1/facts/:fid
func (h *NovelFactHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	fid, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid fact id")
		return
	}

	var req service.UpdateFactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	fact, err := h.svc.UpdateFact(c.Request.Context(), uint(fid), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, fact)
}

// Delete 删除事实
// DELETE /api/v1/facts/:fid
func (h *NovelFactHandler) Delete(c *gin.Context) {
	fid, err := strconv.ParseUint(c.Param("fid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid fact id")
		return
	}

	if err := h.svc.DeleteFact(c.Request.Context(), uint(fid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}
