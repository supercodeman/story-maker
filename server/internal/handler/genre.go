// server/internal/handler/genre.go
package handler

import (
	"net/http"
	"strconv"

	"story-maker/server/internal/model"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// GenreHandler 赛道请求处理器
type GenreHandler struct {
	genreSvc *service.GenreService
}

// NewGenreHandler 创建 GenreHandler 实例
func NewGenreHandler(genreSvc *service.GenreService) *GenreHandler {
	return &GenreHandler{genreSvc: genreSvc}
}

// ListTree 获取赛道树
// GET /api/v1/genres
func (h *GenreHandler) ListTree(c *gin.Context) {
	tree, err := h.genreSvc.GetGenreTree()
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, tree)
}

// Get 获取赛道详情
// GET /api/v1/genres/:id
func (h *GenreHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid genre id")
		return
	}

	genre, err := h.genreSvc.GetGenre(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, "genre not found")
		return
	}
	Success(c, genre)
}

// AdminCreate 创建赛道（管理员）
// POST /api/v1/admin/genres
func (h *GenreHandler) AdminCreate(c *gin.Context) {
	var req struct {
		ParentID  uint   `json:"parent_id"`
		Name      string `json:"name" binding:"required"`
		Slug      string `json:"slug" binding:"required"`
		Icon      string `json:"icon"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	genre := &model.Genre{
		ParentID:  req.ParentID,
		Name:      req.Name,
		Slug:      req.Slug,
		Icon:      req.Icon,
		SortOrder: req.SortOrder,
	}
	if err := h.genreSvc.CreateGenre(genre); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, genre)
}

// AdminUpdate 更新赛道（管理员）
// PUT /api/v1/admin/genres/:id
func (h *GenreHandler) AdminUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid genre id")
		return
	}

	genre, err := h.genreSvc.GetGenre(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, "genre not found")
		return
	}

	var req struct {
		ParentID  *uint   `json:"parent_id"`
		Name      string  `json:"name"`
		Slug      string  `json:"slug"`
		Icon      string  `json:"icon"`
		SortOrder *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	if req.ParentID != nil {
		genre.ParentID = *req.ParentID
	}
	if req.Name != "" {
		genre.Name = req.Name
	}
	if req.Slug != "" {
		genre.Slug = req.Slug
	}
	if req.Icon != "" {
		genre.Icon = req.Icon
	}
	if req.SortOrder != nil {
		genre.SortOrder = *req.SortOrder
	}

	if err := h.genreSvc.UpdateGenre(genre); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, genre)
}

// AdminDelete 删除赛道（管理员）
// DELETE /api/v1/admin/genres/:id
func (h *GenreHandler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid genre id")
		return
	}

	if err := h.genreSvc.DeleteGenre(uint(id)); err != nil {
		InternalError(c, err.Error())
		return
	}
	SuccessWithMessage(c, "genre deleted", nil)
}
