// server/internal/agent/kimi.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	kimiBaseURL       = "https://api.moonshot.cn/v1"
	kimiChatEndpoint  = "/chat/completions"
	kimiImageEndpoint = "/images/generations"
	kimiDefaultModel  = "moonshot-v1-32k"
	kimiTimeout       = 300 * time.Second
)

// KimiProvider Kimi 模型适配器
type KimiProvider struct {
	apiKey string
	client *resty.Client
}

// NewKimiProvider 创建 Kimi Provider 实例
func NewKimiProvider(apiKey string) *KimiProvider {
	client := resty.New().
		SetBaseURL(kimiBaseURL).
		SetTimeout(kimiTimeout).
		SetHeader("Content-Type", "application/json")

	return &KimiProvider{
		apiKey: apiKey,
		client: client,
	}
}

// SetAPIKey 动态设置 API Key（支持用户自有 Key 切换）
func (k *KimiProvider) SetAPIKey(apiKey string) {
	k.apiKey = apiKey
}

// kimiChatRequest Kimi Chat API 请求体
type kimiChatRequest struct {
	Model       string            `json:"model"`
	Messages    []kimiChatMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
}

type kimiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// kimiChatResponse Kimi Chat API 响应体
type kimiChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// kimiImageRequest Kimi 图像生成请求体
type kimiImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
	N      int    `json:"n"`
}

// kimiImageResponse Kimi 图像生成响应体
type kimiImageResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateText 调用 Kimi chat/completions 接口生成文本
func (k *KimiProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	if k.apiKey == "" {
		return nil, errors.New("kimi api key is not set")
	}

	// 构建消息列表
	messages := make([]kimiChatMessage, 0, len(req.History)+2)

	if req.CharacterCtx != "" {
		messages = append(messages, kimiChatMessage{
			Role:    "system",
			Content: req.CharacterCtx,
		})
	}

	// 追加历史对话
	for _, h := range req.History {
		messages = append(messages, kimiChatMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	messages = append(messages, kimiChatMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 16384
	}

	temperature := req.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}

	chatReq := kimiChatRequest{
		Model:       kimiDefaultModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// 优先使用请求指定的模型版本
	if req.Model != "" {
		chatReq.Model = req.Model
	}

	resp, err := k.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+k.apiKey).
		SetBody(chatReq).
		Post(kimiChatEndpoint)

	if err != nil {
		return nil, fmt.Errorf("kimi text request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("kimi text API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	var chatResp kimiChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("kimi text response parse failed: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("kimi text API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("kimi text API returned empty choices")
	}

	textResp := &TextResponse{
		Content: chatResp.Choices[0].Message.Content,
	}

	// 提取 token 消耗统计
	if chatResp.Usage != nil {
		textResp.Usage = &TokenUsage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		}
	}

	return textResp, nil
}

// GenerateImage 调用 Kimi 图像生成接口
func (k *KimiProvider) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	if k.apiKey == "" {
		return nil, errors.New("kimi api key is not set")
	}

	// 构建尺寸字符串
	size := "1024x1024"
	if req.Width > 0 && req.Height > 0 {
		size = fmt.Sprintf("%dx%d", req.Width, req.Height)
	}

	// 组装提示词：如果有参考图 URL，追加到提示词中
	prompt := req.Prompt
	if req.ReferenceURL != "" {
		prompt = fmt.Sprintf("%s\n\nReference image: %s", prompt, req.ReferenceURL)
	}
	if req.Style != "" {
		prompt = fmt.Sprintf("%s\n\nStyle: %s", prompt, req.Style)
	}

	imgReq := kimiImageRequest{
		Model:  "moonshot-v1-8k",
		Prompt: prompt,
		Size:   size,
		N:      1,
	}

	// 优先使用请求指定的图像模型版本
	if req.Model != "" {
		imgReq.Model = req.Model
	}

	resp, err := k.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+k.apiKey).
		SetBody(imgReq).
		Post(kimiImageEndpoint)

	if err != nil {
		return nil, fmt.Errorf("kimi image request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("kimi image API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	var imgResp kimiImageResponse
	if err := json.Unmarshal(resp.Body(), &imgResp); err != nil {
		return nil, fmt.Errorf("kimi image response parse failed: %w", err)
	}

	if imgResp.Error != nil {
		return nil, fmt.Errorf("kimi image API error: %s", imgResp.Error.Message)
	}

	if len(imgResp.Data) == 0 {
		return nil, errors.New("kimi image API returned empty data")
	}

	return &ImageResponse{
		ImageURL: imgResp.Data[0].URL,
	}, nil
}

// AdjustCharacter 角色调整：组装角色约束提示词 + 调用图像生成
func (k *KimiProvider) AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error) {
	// 组装角色约束提示词
	prompt := req.Prompt
	if len(req.Attributes) > 0 {
		prompt += "\n\nCharacter attributes:"
		for key, value := range req.Attributes {
			prompt += fmt.Sprintf("\n- %s: %s", key, value)
		}
	}
	prompt += "\n\nPlease maintain character consistency with the reference images."

	// 复用图像生成能力
	imgReq := &ImageRequest{
		Prompt: prompt,
		Width:  1024,
		Height: 1024,
	}

	return k.GenerateImage(ctx, imgReq)
}

// Name 返回 Provider 名称
func (k *KimiProvider) Name() string {
	return "kimi"
}

// FallbackModels 返回同 Provider 内的降级模型列表
func (k *KimiProvider) FallbackModels() []string {
	return nil // 降级链已由 DB 驱动，不再硬编码
}

// Capabilities 返回 Kimi 支持的能力列表
func (k *KimiProvider) Capabilities() []string {
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
		"overview_extract",
		"outline_generate_characters",
		"butler_generate_topic",
		"butler_generate_storyline",
		"butler_generate_characters",
	}
}

// Embedding 不支持
func (k *KimiProvider) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("kimi does not support embedding")
}
