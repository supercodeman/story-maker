package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// comicComposeInput 漫剧合成输入
type comicComposeInput struct {
	ComicDramaID uint             `json:"comic_drama_id"`
	OutputDir    string           `json:"output_dir"`
	Segments     []composeSegment `json:"segments"`
	Transition   string           `json:"transition"`
	FPS          int              `json:"fps"`
}

// composeSegment 合成片段
type composeSegment struct {
	MediaPath  string  `json:"media_path"`
	AudioPath  string  `json:"audio_path"`
	Duration   float64 `json:"duration"`
	MediaType  string  `json:"media_type"`
	Transition string  `json:"transition"`
}

// comicComposeResult 漫剧合成结果
type comicComposeResult struct {
	OutputPath   string  `json:"output_path"`
	Duration     float64 `json:"duration"`
	ComicDramaID uint    `json:"comic_drama_id"`
}

const ffmpegTimeout = 10 * time.Minute

// ComicComposeExecutor 漫剧视频合成执行器，使用 ffmpeg 将多段媒体+音频合成最终视频
type ComicComposeExecutor struct {
	ffmpegPath string
}

// NewComicComposeExecutor 创建漫剧合成执行器
func NewComicComposeExecutor() *ComicComposeExecutor {
	return &ComicComposeExecutor{ffmpegPath: "ffmpeg"}
}

// Execute 执行视频合成任务
func (e *ComicComposeExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	var input comicComposeInput
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &input); err != nil {
		return nil, fmt.Errorf("failed to parse compose input: %w", err)
	}
	if len(input.Segments) == 0 {
		return nil, fmt.Errorf("segments list is empty")
	}
	if input.OutputDir == "" {
		return nil, fmt.Errorf("output_dir is required")
	}
	fps := input.FPS
	if fps <= 0 {
		fps = 30
	}

	if err := os.MkdirAll(input.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	segmentPaths, err := e.prepareSegments(ctx, &input, fps)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare segments: %w", err)
	}

	concatPath, err := e.concatSegments(ctx, &input, segmentPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to concat segments: %w", err)
	}

	outputPath, err := e.overlayAudio(ctx, &input, concatPath)
	if err != nil {
		return nil, fmt.Errorf("failed to overlay audio: %w", err)
	}

	var totalDuration float64
	for _, seg := range input.Segments {
		totalDuration += seg.Duration
	}

	return &comicComposeResult{
		OutputPath:   outputPath,
		Duration:     totalDuration,
		ComicDramaID: input.ComicDramaID,
	}, nil
}

// prepareSegments 将每个片段转换为统一格式的视频文件
func (e *ComicComposeExecutor) prepareSegments(ctx context.Context, input *comicComposeInput, fps int) ([]string, error) {
	var paths []string
	for i, seg := range input.Segments {
		outPath := filepath.Join(input.OutputDir, fmt.Sprintf("seg_%03d.mp4", i))
		switch seg.MediaType {
		case "static_image", "image_motion":
			if err := e.imageToVideo(ctx, seg.MediaPath, outPath, seg.Duration, fps); err != nil {
				return nil, fmt.Errorf("segment %d image_to_video failed: %w", i, err)
			}
		case "video", "dynamic_image":
			if err := e.normalizeVideo(ctx, seg.MediaPath, outPath, fps); err != nil {
				return nil, fmt.Errorf("segment %d normalize failed: %w", i, err)
			}
		default:
			return nil, fmt.Errorf("segment %d: unsupported media_type %q", i, seg.MediaType)
		}
		paths = append(paths, outPath)
	}
	return paths, nil
}

// imageToVideo 将静态图片转换为带缓慢推拉效果的视频
func (e *ComicComposeExecutor) imageToVideo(ctx context.Context, imagePath, outputPath string, duration float64, fps int) error {
	totalFrames := int(duration * float64(fps))
	filter := fmt.Sprintf(
		"zoompan=z='min(zoom+0.0015,1.3)':x='iw/2-(iw/zoom/2)':y='ih/2-(ih/zoom/2)':d=%d:s=1920x1080:fps=%d",
		totalFrames, fps,
	)
	args := []string{
		"-y", "-i", imagePath,
		"-vf", filter,
		"-c:v", "libx264",
		"-t", fmt.Sprintf("%.2f", duration),
		"-pix_fmt", "yuv420p",
		outputPath,
	}
	return e.runFFmpeg(ctx, args)
}

// normalizeVideo 将视频标准化为统一分辨率和帧率
func (e *ComicComposeExecutor) normalizeVideo(ctx context.Context, inputPath, outputPath string, fps int) error {
	args := []string{
		"-y", "-i", inputPath,
		"-vf", fmt.Sprintf("scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2,fps=%d", fps),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-an",
		outputPath,
	}
	return e.runFFmpeg(ctx, args)
}

// concatSegments 拼接多个视频片段
func (e *ComicComposeExecutor) concatSegments(ctx context.Context, input *comicComposeInput, segmentPaths []string) (string, error) {
	if len(segmentPaths) == 1 {
		return segmentPaths[0], nil
	}
	defaultTransition := input.Transition
	if defaultTransition == "" || defaultTransition == "none" {
		return e.concatWithDemuxer(ctx, input.OutputDir, segmentPaths)
	}
	return e.concatWithXfade(ctx, input, segmentPaths, defaultTransition)
}

// concatWithDemuxer 使用 concat demuxer 无转场拼接
func (e *ComicComposeExecutor) concatWithDemuxer(ctx context.Context, outputDir string, segmentPaths []string) (string, error) {
	listPath := filepath.Join(outputDir, "concat_list.txt")
	var lines []string
	for _, p := range segmentPaths {
		lines = append(lines, fmt.Sprintf("file '%s'", p))
	}
	if err := os.WriteFile(listPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return "", fmt.Errorf("failed to write concat list: %w", err)
	}
	outputPath := filepath.Join(outputDir, "concat_no_audio.mp4")
	args := []string{"-y", "-f", "concat", "-safe", "0", "-i", listPath, "-c", "copy", outputPath}
	if err := e.runFFmpeg(ctx, args); err != nil {
		return "", err
	}
	return outputPath, nil
}

// concatWithXfade 使用 xfade 滤镜实现转场拼接
func (e *ComicComposeExecutor) concatWithXfade(ctx context.Context, input *comicComposeInput, segmentPaths []string, defaultTransition string) (string, error) {
	current := segmentPaths[0]
	transitionDuration := 0.5
	for i := 1; i < len(segmentPaths); i++ {
		outputPath := filepath.Join(input.OutputDir, fmt.Sprintf("xfade_%03d.mp4", i))
		transition := defaultTransition
		if i-1 < len(input.Segments) && input.Segments[i-1].Transition != "" {
			transition = input.Segments[i-1].Transition
		}
		offset := input.Segments[i-1].Duration - transitionDuration
		if offset < 0 {
			offset = 0
		}
		filter := fmt.Sprintf("xfade=transition=%s:duration=%.2f:offset=%.2f", transition, transitionDuration, offset)
		args := []string{"-y", "-i", current, "-i", segmentPaths[i], "-filter_complex", filter, "-c:v", "libx264", "-pix_fmt", "yuv420p", outputPath}
		if err := e.runFFmpeg(ctx, args); err != nil {
			return "", fmt.Errorf("xfade step %d failed: %w", i, err)
		}
		current = outputPath
	}
	return current, nil
}

// overlayAudio 将合并后的音频叠加到视频上
func (e *ComicComposeExecutor) overlayAudio(ctx context.Context, input *comicComposeInput, videoPath string) (string, error) {
	var hasAudio bool
	for _, seg := range input.Segments {
		if seg.AudioPath != "" {
			hasAudio = true
			break
		}
	}
	if !hasAudio {
		return videoPath, nil
	}

	mergedAudioPath := filepath.Join(input.OutputDir, "merged_audio.aac")
	if err := e.mergeAudioTracks(ctx, input, mergedAudioPath); err != nil {
		return "", fmt.Errorf("failed to merge audio: %w", err)
	}

	outputPath := filepath.Join(input.OutputDir, fmt.Sprintf("comic_drama_%d_final.mp4", input.ComicDramaID))
	args := []string{"-y", "-i", videoPath, "-i", mergedAudioPath, "-c:v", "copy", "-c:a", "aac", "-shortest", outputPath}
	if err := e.runFFmpeg(ctx, args); err != nil {
		return "", err
	}
	return outputPath, nil
}

// mergeAudioTracks 将多段音频按时间偏移合并为单轨
func (e *ComicComposeExecutor) mergeAudioTracks(ctx context.Context, input *comicComposeInput, outputPath string) error {
	var inputs []string
	var filterParts []string
	var mixInputs []string
	audioIdx := 0
	var offset float64

	for _, seg := range input.Segments {
		if seg.AudioPath != "" {
			inputs = append(inputs, "-i", seg.AudioPath)
			delayMs := int(offset * 1000)
			filterParts = append(filterParts, fmt.Sprintf("[%d:a]adelay=%d|%d[a%d]", audioIdx, delayMs, delayMs, audioIdx))
			mixInputs = append(mixInputs, fmt.Sprintf("[a%d]", audioIdx))
			audioIdx++
		}
		offset += seg.Duration
	}

	if audioIdx == 0 {
		return fmt.Errorf("no audio tracks to merge")
	}
	if audioIdx == 1 {
		args := append([]string{"-y"}, inputs...)
		args = append(args, "-c:a", "aac", outputPath)
		return e.runFFmpeg(ctx, args)
	}

	filterComplex := strings.Join(filterParts, ";") + ";" +
		strings.Join(mixInputs, "") +
		fmt.Sprintf("amix=inputs=%d:duration=longest[out]", audioIdx)
	args := append([]string{"-y"}, inputs...)
	args = append(args, "-filter_complex", filterComplex, "-map", "[out]", "-c:a", "aac", outputPath)
	return e.runFFmpeg(ctx, args)
}

// runFFmpeg 执行 ffmpeg 命令
func (e *ComicComposeExecutor) runFFmpeg(ctx context.Context, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, ffmpegTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}
	return nil
}
