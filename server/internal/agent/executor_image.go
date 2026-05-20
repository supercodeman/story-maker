// server/internal/agent/executor_image.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"story-maker/server/internal/model"
)

// TextGeneratorFunc 文本生成函数类型（用于 auto prompt 提取，不走降级链）
type TextGeneratorFunc func(ctx context.Context, req *TextRequest) (*TextResponse, error)

// ImageGenTaskExecutor 文生图任务执行器（使用 ImageGenProvider）
type ImageGenTaskExecutor struct {
	provider      ImageGenProvider
	assetWriter   AssetWriter
	textGenerator TextGeneratorFunc
}

// NewImageGenTaskExecutor 创建文生图任务执行器
func NewImageGenTaskExecutor(provider ImageGenProvider, assetWriter AssetWriter) *ImageGenTaskExecutor {
	return &ImageGenTaskExecutor{provider: provider, assetWriter: assetWriter}
}

// SetTextGenerator 注入文本生成能力（用于 auto prompt 提取）
func (e *ImageGenTaskExecutor) SetTextGenerator(fn TextGeneratorFunc) {
	e.textGenerator = fn
}

// imageGenTaskParams 文生图任务参数（从 AITask.Prompt JSON 解析）
type imageGenTaskParams struct {
	Prompt            string   `json:"prompt"`
	AspectRatio       string   `json:"aspect_ratio"`
	N                 int      `json:"n"`
	CharacterRefPaths []string `json:"character_ref_paths"`
}

// Execute 执行文生图任务
func (e *ImageGenTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	if e.provider == nil {
		return nil, fmt.Errorf("image generation provider not configured")
	}

	var params imageGenTaskParams
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &params); err != nil {
		params = imageGenTaskParams{Prompt: ec.Task.Prompt}
	}
	if params.Prompt == "" {
		return nil, fmt.Errorf("prompt is empty")
	}

	// auto 模式：调用文本 AI 从章节内容提取画面描述
	prompts, err := e.resolvePrompts(ctx, params)
	if err != nil {
		return nil, err
	}

	// 对每个 prompt 生成图片
	var allResults []map[string]interface{}
	for _, prompt := range prompts {
		resp, err := e.provider.GenerateImages(ctx, &T2IRequest{
			Prompt:            prompt,
			AspectRatio:       params.AspectRatio,
			N:                 1,
			CharacterRefPaths: params.CharacterRefPaths,
		})
		if err != nil {
			return nil, err
		}

		for _, img := range resp.Images {
			if e.assetWriter != nil {
				metadata, _ := json.Marshal(map[string]interface{}{
					"provider":     e.provider.Name(),
					"prompt":       prompt,
					"aspect_ratio": params.AspectRatio,
				})
				asset := &model.Asset{
					PortfolioID: ec.Task.PortfolioID,
					Type:        model.AssetTypeImage,
					FilePath:    img.FilePath,
					ChapterID:   ec.Task.ChapterID,
					CreatedBy:   ec.Task.UserID,
					Metadata:    string(metadata),
				}
				if err := e.assetWriter.Create(asset); err != nil {
					return nil, fmt.Errorf("image generated but failed to create asset record: %w", err)
				}
			}
			allResults = append(allResults, map[string]interface{}{
				"url":       img.URL,
				"file_path": img.FilePath,
			})
		}
	}

	return map[string]interface{}{
		"images": allResults,
	}, nil
}

// resolvePrompts 解析 prompt：auto 模式调 AI 提取，否则直接返回
func (e *ImageGenTaskExecutor) resolvePrompts(ctx context.Context, params imageGenTaskParams) ([]string, error) {
	if params.Prompt != "auto" {
		return []string{params.Prompt}, nil
	}

	if e.textGenerator == nil {
		return nil, fmt.Errorf("auto prompt extraction requires text generator")
	}

	// 调用文本 AI 提取画面描述
	n := params.N
	if n <= 0 {
		n = 2
	}
	if n > 4 {
		n = 4
	}

	systemPrompt := fmt.Sprintf(`你是一个插画描述专家。根据以下小说章节内容，提取 %d 个最具画面感的场景，为每个场景生成一段英文图片描述（适合 AI 绘画）。
要求：
1. 每段描述独立成行，不要编号
2. 描述要具体、有画面感，包含人物外貌、动作、场景、光线等细节
3. 使用英文输出
4. 每段描述 50-100 词`, n)

	resp, err := e.textGenerator(ctx, &TextRequest{
		Prompt:      systemPrompt,
		MaxTokens:   2000,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("auto prompt extraction failed: %w", err)
	}

	// 按行分割，过滤空行
	lines := strings.Split(strings.TrimSpace(resp.Content), "\n")
	var prompts []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 去掉可能的编号前缀
		line = strings.TrimLeft(line, "0123456789.-) ")
		if len(line) > 10 {
			prompts = append(prompts, line)
		}
	}

	if len(prompts) == 0 {
		return nil, fmt.Errorf("auto prompt extraction returned no valid descriptions")
	}
	if len(prompts) > n {
		prompts = prompts[:n]
	}

	return prompts, nil
}

// CharacterTaskExecutor 角色调整任务执行器（character_adjust）
type CharacterTaskExecutor struct{}

func (e *CharacterTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	req := &CharacterAdjustRequest{
		Prompt: ec.Task.Prompt,
	}

	resp, err := ec.Provider.AdjustCharacter(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image_url": resp.ImageURL,
		"file_path": resp.FilePath,
	}, nil
}

// ImageEditTaskExecutor 图像编辑任务执行器（image_edit，走 AIProvider.GenerateImage）
type ImageEditTaskExecutor struct{}

func (e *ImageEditTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	req := &ImageRequest{
		Model:  ec.ModelVersion,
		Prompt: ec.Task.Prompt,
		Width:  1024,
		Height: 1024,
	}

	resp, err := ec.Provider.GenerateImage(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image_url": resp.ImageURL,
		"file_path": resp.FilePath,
	}, nil
}
