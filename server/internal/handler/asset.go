// server/internal/handler/asset.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/model"
	"ai-curton/server/internal/service"
	"ai-curton/server/internal/storage"

	"github.com/gin-gonic/gin"
)

// AssetHandler 资源请求处理层
type AssetHandler struct {
	svc     *service.AssetService
	storage storage.Storage
}

// NewAssetHandler 创建 AssetHandler 实例
func NewAssetHandler(store storage.Storage) *AssetHandler {
	return &AssetHandler{
		svc:     service.NewAssetService(store),
		storage: store,
	}
}

// List 获取作品集下的资源列表
// GET /api/v1/portfolios/:id/assets
func (h *AssetHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	pID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio id")
		return
	}

	assets, err := h.svc.List(uint(pID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, assets)
}

// Upload 上传资源文件
// POST /api/v1/assets/upload
// Form fields: file(文件), portfolio_id(uint), type(string)
func (h *AssetHandler) Upload(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 解析 portfolio_id
	portfolioIDStr := c.PostForm("portfolio_id")
	if portfolioIDStr == "" {
		BadRequest(c, "portfolio_id is required")
		return
	}
	portfolioID, err := strconv.ParseUint(portfolioIDStr, 10, 64)
	if err != nil {
		BadRequest(c, "invalid portfolio_id")
		return
	}

	// 解析资源类型
	assetType := c.PostForm("type")
	if assetType == "" {
		assetType = model.AssetTypeImage // 默认为图片类型
	}

	// 获取上传文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	// 校验文件大小（20MB 限制）
	if header.Size > model.MaxUploadSize {
		BadRequest(c, "file size exceeds 20MB limit")
		return
	}

	// 校验文件 MIME 类型（白名单校验，防止恶意文件上传）
	contentType := header.Header.Get("Content-Type")
	if !model.AllowedUploadMIMETypes[contentType] {
		BadRequest(c, "only jpg/png/webp/gif files are allowed")
		return
	}

	// 从 form 中解析可选的 metadata
	metadata := map[string]string{}
	if m := c.PostForm("metadata"); m != "" {
		// 简单处理：metadata 作为单个字符串存入
		metadata["raw"] = m
	}

	asset, err := h.svc.Upload(c.Request.Context(), userID, uint(portfolioID), assetType, header.Filename, file, metadata)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, asset)
}

// Get 获取资源详情
// GET /api/v1/assets/:id
func (h *AssetHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	aID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid asset id")
		return
	}

	asset, err := h.svc.GetByID(uint(aID), userID)
	if err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, asset)
}

// Delete 删除资源
// DELETE /api/v1/assets/:id
func (h *AssetHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	aID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid asset id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(aID), userID); err != nil {
		Error(c, http.StatusForbidden, err.Error())
		return
	}

	Success(c, gin.H{"message": "asset deleted"})
}
