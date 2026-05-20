package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// charRefInput 角色定妆照生成输入
type charRefInput struct {
	Characters  []charRefCharacter `json:"characters"`
	AspectRatio string             `json:"aspect_ratio"`
}

// charRefCharacter 单个角色描述
type charRefCharacter struct {
	Name        string `json:"name"`
	Appearance  string `json:"appearance"`
	StylePrompt string `json:"style_prompt"`
}

// charRefResult 单个角色定妆照结果
type charRefResult struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	FilePath string `json:"file_path"`
}

// ComicCharRefExecutor 角色定妆照生成执行器
type ComicCharRefExecutor struct {
	imageProvider ImageGenProvider
}

// NewComicCharRefExecutor 创建角色定妆照执行器
func NewComicCharRefExecutor(provider ImageGenProvider) *ComicCharRefExecutor {
	return &ComicCharRefExecutor{imageProvider: provider}
}

func (e *ComicCharRefExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	if e.imageProvider == nil {
		return nil, fmt.Errorf("ImageGenProvider not configured for character reference generation")
	}

	var input charRefInput
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &input); err != nil {
		return nil, fmt.Errorf("failed to parse char ref input: %w", err)
	}
	if len(input.Characters) == 0 {
		return nil, fmt.Errorf("characters list is empty")
	}

	aspectRatio := input.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "2:3"
	}

	var results []charRefResult
	for _, char := range input.Characters {
		prompt := e.buildCharPrompt(char)
		resp, err := e.imageProvider.GenerateImages(ctx, &T2IRequest{
			Prompt:      prompt,
			AspectRatio: aspectRatio,
			N:           1,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate ref image for character %q: %w", char.Name, err)
		}
		if len(resp.Images) == 0 {
			return nil, fmt.Errorf("no image returned for character %q", char.Name)
		}
		results = append(results, charRefResult{
			Name:     char.Name,
			ImageURL: resp.Images[0].URL,
			FilePath: resp.Images[0].FilePath,
		})
	}

	return map[string]interface{}{
		"character_refs": results,
	}, nil
}

// buildCharPrompt 构建角色定妆照生成 prompt
func (e *ComicCharRefExecutor) buildCharPrompt(char charRefCharacter) string {
	var parts []string
	parts = append(parts, "character reference sheet, half-body portrait")
	if char.Appearance != "" {
		parts = append(parts, char.Appearance)
	}
	if char.StylePrompt != "" {
		parts = append(parts, char.StylePrompt)
	}
	parts = append(parts, "high quality, detailed, consistent style, clean background, front view")
	return strings.Join(parts, ", ")
}
