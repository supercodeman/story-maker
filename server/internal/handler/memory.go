// server/internal/handler/memory.go
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MemoryHandler 记忆管理请求处理器
type MemoryHandler struct {
	memorySvc *service.MemoryService
	marketSvc *service.MarketService
	genreSvc  *service.GenreService
}

// NewMemoryHandler 创建 MemoryHandler 实例
func NewMemoryHandler(memorySvc *service.MemoryService, marketSvc *service.MarketService, genreSvc *service.GenreService) *MemoryHandler {
	return &MemoryHandler{memorySvc: memorySvc, marketSvc: marketSvc, genreSvc: genreSvc}
}

// Create 创建记忆
// POST /api/v1/memories
func (h *MemoryHandler) Create(c *gin.Context) {
	var req service.CreateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.CreateMemory(c.Request.Context(), userID, &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, memory)
}

// List 我的记忆列表
// GET /api/v1/memories
func (h *MemoryHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	category := c.Query("category")

	memories, err := h.memorySvc.ListMyMemories(userID, category)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, memories)
}

// Get 记忆详情
// GET /api/v1/memories/:mid
func (h *MemoryHandler) Get(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.GetMemory(uint(mid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "memory not found")
			return
		}
		InternalError(c, err.Error())
		return
	}

	// 权限校验：只有创建者或有授权的用户可查看完整信息
	if memory.UserID != userID && !h.marketSvc.HasLicense(userID, memory.ID) {
		// 返回公开信息
		Success(c, gin.H{
			"id":             memory.ID,
			"category":       memory.Category,
			"title":          memory.Title,
			"description":    memory.Description,
			"preview_text":   memory.PreviewText,
			"quality":        memory.Quality,
			"quality_detail": memory.QualityDetail,
			"quality_grade":  memory.QualityGrade,
			"tags":           memory.Tags,
		})
		return
	}

	Success(c, memory)
}

// Update 编辑记忆信息
// PUT /api/v1/memories/:mid
func (h *MemoryHandler) Update(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.GetMemory(uint(mid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "memory not found")
			return
		}
		InternalError(c, err.Error())
		return
	}
	if memory.UserID != userID {
		Error(c, http.StatusForbidden, "permission denied")
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Tags        string `json:"tags"`
		GenreIDs    []uint `json:"genre_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.memorySvc.UpdateMemory(memory, req.Title, req.Description, req.Tags); err != nil {
		InternalError(c, err.Error())
		return
	}

	// 更新赛道关联
	if req.GenreIDs != nil {
		if err := h.genreSvc.SetMemoryGenres(memory.ID, req.GenreIDs); err != nil {
			InternalError(c, err.Error())
			return
		}
	}

	Success(c, memory)
}

// Delete 删除记忆
// DELETE /api/v1/memories/:mid
func (h *MemoryHandler) Delete(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.GetMemory(uint(mid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "memory not found")
			return
		}
		InternalError(c, err.Error())
		return
	}
	if memory.UserID != userID {
		Error(c, http.StatusForbidden, "permission denied")
		return
	}

	if err := h.memorySvc.DeleteMemory(uint(mid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "memory deleted", nil)
}

// Refine 追加样本重新提取
// POST /api/v1/memories/:mid/refine
func (h *MemoryHandler) Refine(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.GetMemory(uint(mid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "memory not found")
			return
		}
		InternalError(c, err.Error())
		return
	}
	if memory.UserID != userID {
		Error(c, http.StatusForbidden, "permission denied")
		return
	}

	var req struct {
		AdditionalText string `json:"additional_text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.memorySvc.RefineMemory(memory, req.AdditionalText); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, memory)
}

// Publish 申请上架
// POST /api/v1/memories/:mid/publish
func (h *MemoryHandler) Publish(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.GetMemory(uint(mid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "memory not found")
			return
		}
		InternalError(c, err.Error())
		return
	}
	if memory.UserID != userID {
		Error(c, http.StatusForbidden, "permission denied")
		return
	}

	var req struct {
		Price int `json:"price" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.memorySvc.PublishMemory(memory, req.Price); err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, "submitted for review", nil)
}

// Archive 下架
// POST /api/v1/memories/:mid/archive
func (h *MemoryHandler) Archive(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.GetMemory(uint(mid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "memory not found")
			return
		}
		InternalError(c, err.Error())
		return
	}
	if memory.UserID != userID {
		Error(c, http.StatusForbidden, "permission denied")
		return
	}

	if err := h.memorySvc.ArchiveMemory(uint(mid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "memory archived", nil)
}

// Preview 生成预览文本
// POST /api/v1/memories/:mid/preview
func (h *MemoryHandler) Preview(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	userID := c.GetUint("user_id")
	memory, err := h.memorySvc.GetMemory(uint(mid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Error(c, http.StatusNotFound, "memory not found")
			return
		}
		InternalError(c, err.Error())
		return
	}
	if memory.UserID != userID {
		Error(c, http.StatusForbidden, "permission denied")
		return
	}

	var req struct {
		ModelName string `json:"model_name"`
	}
	c.ShouldBindJSON(&req)

	preview, err := h.memorySvc.GeneratePreview(c.Request.Context(), memory, req.ModelName)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"preview_text": preview})
}

// ListBindings 查询小说绑定的记忆
// GET /api/v1/novels/:id/memory-bindings
func (h *MemoryHandler) ListBindings(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	bindings, err := h.memorySvc.ListBindings(uint(novelID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 附带记忆详情
	memories, _ := h.memorySvc.GetBindingMemories(uint(novelID))

	Success(c, gin.H{
		"bindings": bindings,
		"memories": memories,
	})
}

// SetBindings 设置小说-记忆绑定
// PUT /api/v1/novels/:id/memory-bindings
func (h *MemoryHandler) SetBindings(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req struct {
		Bindings []struct {
			Category string `json:"category" binding:"required"`
			MemoryID uint   `json:"memory_id"`
		} `json:"bindings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	for _, b := range req.Bindings {
		if b.MemoryID == 0 {
			// 解绑
			h.memorySvc.RemoveBinding(uint(novelID), b.Category)
		} else {
			// 绑定
			if err := h.memorySvc.SetBinding(uint(novelID), b.Category, b.MemoryID); err != nil {
				BadRequest(c, err.Error())
				return
			}
		}
	}

	SuccessWithMessage(c, "bindings updated", nil)
}

// ListAccessible 获取用户可用的记忆（自己的 + 已购买的）
// GET /api/v1/memories/accessible
func (h *MemoryHandler) ListAccessible(c *gin.Context) {
	userID := c.GetUint("user_id")
	category := c.Query("category")

	memories, err := h.memorySvc.ListAccessibleMemories(userID, category)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, memories)
}

// AdminListReviewing 管理员查询所有审核中的记忆
// GET /api/v1/admin/memories/reviewing
func (h *MemoryHandler) AdminListReviewing(c *gin.Context) {
	memories, err := h.memorySvc.ListReviewingMemories()
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, memories)
}

// AdminApprove 管理员强制通过审核
// POST /api/v1/admin/memories/:mid/approve
func (h *MemoryHandler) AdminApprove(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	if err := h.memorySvc.AdminApproveMemory(uint(mid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "memory approved", nil)
}

// AdminReject 管理员强制拒绝审核
// POST /api/v1/admin/memories/:mid/reject
func (h *MemoryHandler) AdminReject(c *gin.Context) {
	mid, err := strconv.ParseUint(c.Param("mid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid memory id")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if err := h.memorySvc.AdminRejectMemory(uint(mid), req.Reason); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "memory rejected", nil)
}
