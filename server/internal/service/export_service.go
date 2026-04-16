// server/internal/service/export_service.go
package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
	"ai-curton/server/internal/storage"
)

// ExportService 导出服务，负责全本小说 Word 和音频导出
type ExportService struct {
	novelDAO *dao.NovelDAO
	assetDAO *dao.AssetDAO
	storage  storage.Storage
	notifier agent.Notifier
	tts      agent.TTSProvider
}

// NewExportService 创建导出服务实例
func NewExportService(store storage.Storage, notifier agent.Notifier, tts agent.TTSProvider) *ExportService {
	return &ExportService{
		novelDAO: dao.NewNovelDAO(),
		assetDAO: dao.NewAssetDAO(),
		storage:  store,
		notifier: notifier,
		tts:      tts,
	}
}

// chapterAudioInfo 章节音频信息
type chapterAudioInfo struct {
	ChapterTitle string
	FilePath     string
}

// NovelExportData 导出数据结构
type NovelExportData struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Chapters    []ChapterExportData  `json:"chapters"`
}

// ChapterExportData 章节导出数据
type ChapterExportData struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// GetNovelExportData 获取小说导出数据（供前端组装 Word）
func (s *ExportService) GetNovelExportData(novelID uint) (*NovelExportData, error) {
	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return nil, fmt.Errorf("novel not found: %w", err)
	}

	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}

	data := &NovelExportData{
		Title:       novel.Title,
		Description: novel.Description,
		Chapters:    make([]ChapterExportData, len(chapters)),
	}
	for i, ch := range chapters {
		data.Chapters[i] = ChapterExportData{Title: ch.Title, Content: ch.Content}
	}
	return data, nil
}

// ExportNovelToWord 导出全本小说为 Word 文档（图文混排）
// Deprecated: 已改为前端生成，保留用于音频导出等场景的参考
func (s *ExportService) ExportNovelToWord(ctx context.Context, userID, novelID uint) (string, error) {
	novel, err := s.novelDAO.GetNovel(novelID)
	if err != nil {
		return "", fmt.Errorf("novel not found: %w", err)
	}

	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return "", fmt.Errorf("failed to list chapters: %w", err)
	}

	// 收集章节ID，查询关联的图片资产
	chapterIDs := make([]uint, len(chapters))
	for i, ch := range chapters {
		chapterIDs[i] = ch.ID
	}
	imageAssets, _ := s.assetDAO.ListByChapterIDs(chapterIDs, model.AssetTypeImage)

	// 按章节ID分组图片
	chapterImages := make(map[uint][]model.Asset)
	for _, asset := range imageAssets {
		if asset.ChapterID != nil {
			chapterImages[*asset.ChapterID] = append(chapterImages[*asset.ChapterID], asset)
		}
	}

	// 生成 HTML 格式文档（可被 Word 打开，后续可替换为 unioffice 生成 docx）
	var buf bytes.Buffer
	buf.WriteString("<html><head><meta charset='utf-8'><title>")
	buf.WriteString(novel.Title)
	buf.WriteString("</title></head><body>")

	// 封面
	buf.WriteString(fmt.Sprintf("<h1 style='text-align:center'>%s</h1>", novel.Title))
	if novel.Description != "" {
		buf.WriteString(fmt.Sprintf("<p style='text-align:center;color:#666'>%s</p>", novel.Description))
	}
	buf.WriteString("<hr/>")

	// 目录
	buf.WriteString("<h2>目录</h2><ul>")
	for _, ch := range chapters {
		buf.WriteString(fmt.Sprintf("<li>%s</li>", ch.Title))
	}
	buf.WriteString("</ul><hr/>")

	// 各章节正文 + 图片
	for _, ch := range chapters {
		buf.WriteString(fmt.Sprintf("<h2>%s</h2>", ch.Title))
		if images, ok := chapterImages[ch.ID]; ok {
			for _, img := range images {
				buf.WriteString(fmt.Sprintf("<p><img src='%s' style='max-width:100%%'/></p>", img.FilePath))
			}
		}
		content := strings.ReplaceAll(ch.Content, "\n", "<br/>")
		buf.WriteString(fmt.Sprintf("<div>%s</div><hr/>", content))
	}

	buf.WriteString("</body></html>")

	storagePath := fmt.Sprintf("exports/%d/%s.html", novelID, novel.Title)
	fileURL, err := s.storage.Upload(ctx, &buf, storagePath)
	if err != nil {
		return "", fmt.Errorf("failed to save export file: %w", err)
	}

	if s.notifier != nil {
		_ = s.notifier.NotifyUserWithType(userID, "export_complete", map[string]interface{}{
			"type":         "word",
			"novel_id":     novelID,
			"download_url": fileURL,
		})
	}

	return fileURL, nil
}

// ExportNovelAudio 导出全本音频（分章节 MP3 + 合并全本 + ZIP 打包）
func (s *ExportService) ExportNovelAudio(ctx context.Context, userID, novelID uint, voiceID string, speed float64) (string, error) {
	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return "", fmt.Errorf("failed to list chapters: %w", err)
	}
	if len(chapters) == 0 {
		return "", fmt.Errorf("novel has no chapters")
	}

	// 查询已有的音频资产
	chapterIDs := make([]uint, len(chapters))
	for i, ch := range chapters {
		chapterIDs[i] = ch.ID
	}
	existingAudios, _ := s.assetDAO.ListByChapterIDs(chapterIDs, model.AssetTypeAudio)
	audioMap := make(map[uint]model.Asset)
	for _, a := range existingAudios {
		if a.ChapterID != nil {
			audioMap[*a.ChapterID] = a
		}
	}

	// 逐章节生成或复用音频
	var audios []chapterAudioInfo

	for i, ch := range chapters {
		if s.notifier != nil {
			_ = s.notifier.NotifyUserWithType(userID, "export_progress", map[string]interface{}{
				"message":  fmt.Sprintf("正在处理第 %d/%d 章音频: %s", i+1, len(chapters), ch.Title),
				"progress": int(float64(i) / float64(len(chapters)) * 100),
			})
		}

		// 复用已有音频
		if existing, ok := audioMap[ch.ID]; ok {
			audios = append(audios, chapterAudioInfo{ChapterTitle: ch.Title, FilePath: existing.FilePath})
			continue
		}

		// 生成新音频
		if s.tts == nil || ch.Content == "" {
			log.Printf("[export] 跳过章节 %s: TTS未配置或内容为空", ch.Title)
			continue
		}

		resp, err := s.tts.GenerateSpeech(ctx, &agent.TTSRequest{
			Text:    ch.Content,
			VoiceID: voiceID,
			Speed:   speed,
		})
		if err != nil {
			log.Printf("[export] 章节 %s 音频生成失败: %v", ch.Title, err)
			continue
		}

		// 创建 Asset 记录
		chapterID := ch.ID
		metaJSON, _ := json.Marshal(map[string]string{
			"voice_id": voiceID,
			"source":   "export",
		})
		asset := &model.Asset{
			PortfolioID: chapters[0].NovelID,
			Type:        model.AssetTypeAudio,
			FilePath:    resp.FilePath,
			Metadata:    string(metaJSON),
			Duration:    resp.Duration,
			ChapterID:   &chapterID,
			CreatedBy:   userID,
		}
		_ = s.assetDAO.Create(asset)

		audios = append(audios, chapterAudioInfo{ChapterTitle: ch.Title, FilePath: resp.FilePath})
	}

	if len(audios) == 0 {
		return "", fmt.Errorf("no audio files generated")
	}

	// 打包为 ZIP
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	for i, audio := range audios {
		reader, err := s.storage.Download(ctx, audio.FilePath)
		if err != nil {
			log.Printf("[export] 下载音频失败 %s: %v", audio.FilePath, err)
			continue
		}
		fileName := fmt.Sprintf("%02d_%s.mp3", i+1, sanitizeFileName(audio.ChapterTitle))
		w, err := zipWriter.Create(fileName)
		if err != nil {
			reader.Close()
			continue
		}
		io.Copy(w, reader)
		reader.Close()
	}

	// 尝试 ffmpeg 合并全本音频
	mergedPath := s.tryMergeAudio(ctx, audios)
	if mergedPath != "" {
		if reader, err := s.storage.Download(ctx, mergedPath); err == nil {
			if w, err := zipWriter.Create("全本合并.mp3"); err == nil {
				io.Copy(w, reader)
			}
			reader.Close()
		}
	}

	zipWriter.Close()

	novel, _ := s.novelDAO.GetNovel(novelID)
	title := "novel"
	if novel != nil {
		title = novel.Title
	}
	zipPath := fmt.Sprintf("exports/%d/%s_audio.zip", novelID, title)
	fileURL, err := s.storage.Upload(ctx, &zipBuf, zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to save zip: %w", err)
	}

	if s.notifier != nil {
		_ = s.notifier.NotifyUserWithType(userID, "export_complete", map[string]interface{}{
			"type":         "audio",
			"novel_id":     novelID,
			"download_url": fileURL,
		})
	}

	return fileURL, nil
}

// tryMergeAudio 尝试使用 ffmpeg 合并音频文件
func (s *ExportService) tryMergeAudio(ctx context.Context, audios []chapterAudioInfo) string {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		log.Println("[export] ffmpeg not found, skipping audio merge")
		return ""
	}

	ts := time.Now().UnixMilli()

	// 创建 ffmpeg concat 列表
	var listContent strings.Builder
	for _, audio := range audios {
		absPath, _ := filepath.Abs(filepath.Join("./uploads", audio.FilePath))
		listContent.WriteString(fmt.Sprintf("file '%s'\n", absPath))
	}

	listPath := fmt.Sprintf("./uploads/exports/merge_list_%d.txt", ts)
	listBuf := bytes.NewBufferString(listContent.String())
	_, _ = s.storage.Upload(ctx, listBuf, fmt.Sprintf("exports/merge_list_%d.txt", ts))

	outputPath := fmt.Sprintf("exports/merged_%d.mp3", ts)
	absOutput, _ := filepath.Abs(filepath.Join("./uploads", outputPath))

	cmd := exec.CommandContext(ctx, "ffmpeg", "-f", "concat", "-safe", "0", "-i", listPath, "-c", "copy", absOutput)
	if err := cmd.Run(); err != nil {
		log.Printf("[export] ffmpeg merge failed: %v", err)
		return ""
	}

	return outputPath
}

// sanitizeFileName 清理文件名中的非法字符
func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return replacer.Replace(name)
}

// GetChapterAssets 获取章节的多媒体资产列表
func (s *ExportService) GetChapterAssets(chapterID uint, assetType string) ([]model.Asset, error) {
	return s.assetDAO.ListByChapterID(chapterID, assetType)
}
