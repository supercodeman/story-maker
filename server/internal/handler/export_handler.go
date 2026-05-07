// server/internal/handler/export_handler.go
package handler

import (
	"strconv"

	"story-maker/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ExportHandler 导出请求处理器
type ExportHandler struct {
	exportService *service.ExportService
}

// NewExportHandler 创建 ExportHandler 实例
func NewExportHandler(exportService *service.ExportService) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

// ExportWordRequest Word 导出请求
type ExportWordRequest struct {
	NovelID uint `json:"novel_id" binding:"required"`
}

// ExportAudioRequest 音频导出请求
type ExportAudioRequest struct {
	NovelID uint    `json:"novel_id" binding:"required"`
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
}

// ExportWord 返回小说数据供前端组装 Word 文档
// POST /api/v1/novels/:id/export/word
func (h *ExportHandler) ExportWord(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	data, err := h.exportService.GetNovelExportData(uint(novelID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, data)
}

// ExportAudio 导出全本音频
// POST /api/v1/novels/:id/export/audio
func (h *ExportHandler) ExportAudio(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req ExportAudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许不传 body，使用默认参数
		req = ExportAudioRequest{}
	}

	userID := c.GetUint("user_id")
	voiceID := req.VoiceID
	if voiceID == "" {
		voiceID = "female-yujie"
	}
	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}

	// 异步执行导出
	go func() {
		_, _ = h.exportService.ExportNovelAudio(c.Request.Context(), userID, uint(novelID), voiceID, speed)
	}()

	Success(c, gin.H{
		"message": "export started",
		"type":    "audio",
	})
}

// GetChapterAssets 获取章节的多媒体资产列表
// GET /api/v1/chapters/:id/assets
func (h *ExportHandler) GetChapterAssets(c *gin.Context) {
	chapterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "invalid chapter id")
		return
	}

	assetType := c.Query("type") // 可选过滤：audio, video, image

	assets, err := h.exportService.GetChapterAssets(uint(chapterID), assetType)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, assets)
}
