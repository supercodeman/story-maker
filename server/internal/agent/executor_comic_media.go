package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// comicMediaInput 漫剧媒体生成输入
type comicMediaInput struct {
	FrameDesc          string  `json:"frame_desc"`
	MediaType          string  `json:"media_type"`
	Importance         string  `json:"importance"`
	ReferenceImagePath string  `json:"reference_image_path"`
	Duration           float64 `json:"duration"`
	StoryboardID       uint    `json:"storyboard_id"`
}

// comicMediaResult 漫剧媒体生成结果
type comicMediaResult struct {
	MediaURL     string  `json:"media_url"`
	FilePath     string  `json:"file_path"`
	Duration     float64 `json:"duration"`
	MediaType    string  `json:"media_type"`
	StoryboardID uint    `json:"storyboard_id"`
}

// ComicMediaExecutor 漫剧媒体生成执行器，根据重要性和类型选择生成静态图或视频
type ComicMediaExecutor struct {
	videoProvider VideoProvider
	imageProvider ImageGenProvider
}

// NewComicMediaExecutor 创建漫剧媒体执行器
func NewComicMediaExecutor(video VideoProvider, image ImageGenProvider) *ComicMediaExecutor {
	return &ComicMediaExecutor{videoProvider: video, imageProvider: image}
}

// Execute 执行媒体生成任务
func (e *ComicMediaExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	var input comicMediaInput
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &input); err != nil {
		return nil, fmt.Errorf("failed to parse media input: %w", err)
	}
	if input.FrameDesc == "" {
		return nil, fmt.Errorf("frame_desc is required for media generation")
	}

	// 低重要性的静态图/微动图降级为静态图片生成，节省视频生成资源
	switch {
	case input.Importance == "low" && (input.MediaType == "image_motion" || input.MediaType == "static_image"):
		return e.generateStaticImage(ctx, &input)
	default:
		return e.generateVideo(ctx, ec, &input)
	}
}

// generateStaticImage 生成静态图片（低重要性场景降级策略）
func (e *ComicMediaExecutor) generateStaticImage(ctx context.Context, input *comicMediaInput) (*comicMediaResult, error) {
	if e.imageProvider == nil {
		return nil, fmt.Errorf("ImageGenProvider not configured")
	}
	resp, err := e.imageProvider.GenerateImages(ctx, &T2IRequest{
		Prompt:      input.FrameDesc,
		AspectRatio: "16:9",
		N:           1,
	})
	if err != nil {
		return nil, fmt.Errorf("static image generation failed: %w", err)
	}
	if len(resp.Images) == 0 {
		return nil, fmt.Errorf("no image returned for frame")
	}
	return &comicMediaResult{
		MediaURL:     resp.Images[0].URL,
		FilePath:     resp.Images[0].FilePath,
		Duration:     input.Duration,
		MediaType:    "static_image",
		StoryboardID: input.StoryboardID,
	}, nil
}

// generateVideo 生成视频（默认路径，根据重要性选择模型质量）
func (e *ComicMediaExecutor) generateVideo(ctx context.Context, ec *ExecContext, input *comicMediaInput) (*comicMediaResult, error) {
	if e.videoProvider == nil {
		return nil, fmt.Errorf("VideoProvider not configured")
	}
	modelName := e.selectModel(ec.ModelVersion, input.Importance)
	resp, err := e.videoProvider.GenerateVideo(ctx, &VideoGenRequest{
		Prompt:             input.FrameDesc,
		Model:              modelName,
		ReferenceImagePath: input.ReferenceImagePath,
		Duration:           input.Duration,
	})
	if err != nil {
		return nil, fmt.Errorf("video generation failed: %w", err)
	}
	return &comicMediaResult{
		MediaURL:     resp.VideoURL,
		FilePath:     resp.FilePath,
		Duration:     resp.Duration,
		MediaType:    input.MediaType,
		StoryboardID: input.StoryboardID,
	}, nil
}

// selectModel 根据重要性和 ExecContext 中的模型版本选择视频生成模型
func (e *ComicMediaExecutor) selectModel(modelVersion string, importance string) string {
	switch importance {
	case "high":
		if modelVersion != "" {
			return modelVersion
		}
		return "video-high-quality"
	default:
		return "video-standard"
	}
}
