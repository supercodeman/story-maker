// server/internal/handler/character.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/service"
	"ai-curton/server/internal/storage"

	"github.com/gin-gonic/gin"
)

// CharacterHandler 角色请求处理层
type CharacterHandler struct {
	svc     *service.CharacterService
	storage storage.Storage
}

// NewCharacterHandler 创建 CharacterHandler 实例
func NewCharacterHandler(store storage.Storage) *CharacterHandler {
	return &CharacterHandler{
		svc:     service.NewCharacterService(),
		storage: store,
	}
}

// List 获取作品集下的角色列表
// GET /api/v1/portfolios/:id/characters
func (h *CharacterHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio id")
		return
	}

	characters, err := h.svc.List(uint(pID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, characters)
}

// Create 创建角色
// POST /api/v1/portfolios/:id/characters
func (h *CharacterHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio id")
		return
	}

	var req service.CreateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ch, err := h.svc.Create(uint(pID), userID, &req)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, ch)
}

// Get 获取角色详情
// GET /api/v1/characters/:id
func (h *CharacterHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid character id")
		return
	}

	ch, err := h.svc.GetByID(uint(chID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, ch)
}

// Update 更新角色
// PUT /api/v1/characters/:id
func (h *CharacterHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid character id")
		return
	}

	var req service.UpdateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ch, err := h.svc.Update(uint(chID), userID, &req)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, ch)
}

// Delete 删除角色
// DELETE /api/v1/characters/:id
func (h *CharacterHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid character id")
		return
	}

	if err := h.svc.Delete(uint(chID), userID); err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, gin.H{"message": "character deleted"})
}

// UploadReference 上传角色参考图
// POST /api/v1/characters/:id/reference
func (h *CharacterHandler) UploadReference(c *gin.Context) {
	userID := c.GetUint("user_id")
	chID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid character id")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	// 校验文件大小（20MB 限制）
	if header.Size > 20*1024*1024 {
		BadRequest(c, "file size exceeds 20MB limit")
		return
	}

	// 校验文件类型
	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	if !allowedTypes[contentType] {
		BadRequest(c, "only jpg/png/webp/gif files are allowed")
		return
	}

	// 构造存储路径：characters/{character_id}/{filename}
	storagePath := "characters/" + strconv.FormatUint(chID, 10) + "/" + header.Filename
	url, err := h.storage.Upload(c.Request.Context(), file, storagePath)
	if err != nil {
		Error(c, http.StatusInternalServerError, "failed to upload file")
		return
	}

	// 将参考图路径添加到角色记录
	ch, err := h.svc.AddReferenceImage(uint(chID), userID, url)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, ch)
}
