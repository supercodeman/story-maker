// server/internal/agent/mock.go
package agent

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// MockProvider Mock AI 模型适配器，用于开发测试
type MockProvider struct{}

// NewMockProvider 创建 Mock Provider 实例
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// GenerateText 模拟文本生成，延迟 2-3 秒返回漫画脚本
func (m *MockProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	delay := time.Duration(2000+rand.Intn(1000)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	content := fmt.Sprintf(`【AI 生成漫画脚本】

场景一：城市天际线，黄昏
画面描述：高楼林立的城市轮廓在夕阳下呈现金色光芒，一个身影站在楼顶。

对话：
角色A："这座城市的故事，才刚刚开始。"

场景二：街道特写
画面描述：霓虹灯闪烁的街道，行人匆匆。

---
提示词：%s
生成时间：%s`, req.Prompt, time.Now().Format("2006-01-02 15:04:05"))

	return &TextResponse{Content: content}, nil
}

// GenerateImage 模拟图像生成，延迟 3-5 秒返回占位图 URL
func (m *MockProvider) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	delay := time.Duration(3000+rand.Intn(2000)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	width := req.Width
	if width <= 0 {
		width = 1024
	}
	height := req.Height
	if height <= 0 {
		height = 1024
	}

	imageURL := fmt.Sprintf("https://placehold.co/%dx%d/1a1d2e/7c8cf8?text=AI+Generated", width, height)

	return &ImageResponse{
		ImageURL: imageURL,
	}, nil
}

// AdjustCharacter 模拟角色调整，延迟 2 秒返回结果
func (m *MockProvider) AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error) {
	delay := time.Duration(2000+rand.Intn(1000)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	imageURL := "https://placehold.co/1024x1024/232640/67e8f9?text=Character+Adjusted"

	return &ImageResponse{
		ImageURL: imageURL,
	}, nil
}

// Name 返回 Provider 名称
func (m *MockProvider) Name() string {
	return "mock"
}

// FallbackModels Mock 无降级模型
func (m *MockProvider) FallbackModels() []string {
	return []string{}
}

// Capabilities 返回 Mock 支持的所有能力
func (m *MockProvider) Capabilities() []string {
	return []string{
		"text_gen",
		"text_polish",
		"storyboard",
		"image_gen",
		"image_edit",
		"character_adjust",
		"chapter_summary_polish",
		"chapter_polish",
		"chapter_expand",
		"chapter_continue",
		"outline_generate",
		"outline_title_polish",
		"outline_summary_polish",
		"outline_summary_expand",
		"knowledge_extract",
		"outline_generate_characters",
		"butler_generate_topic",
		"butler_generate_storyline",
		"butler_generate_characters",
		"embedding",
	}
}

// Embedding 模拟 Embedding 生成
func (m *MockProvider) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	dimension := 1024
	vectors := make([][]float64, len(req.Texts))
	for i := range req.Texts {
		vec := make([]float64, dimension)
		for j := range vec {
			vec[j] = float64(rand.Intn(100)) / 100.0
		}
		vectors[i] = vec
	}
	return &EmbeddingResponse{
		Vectors:   vectors,
		Dimension: dimension,
	}, nil
}
