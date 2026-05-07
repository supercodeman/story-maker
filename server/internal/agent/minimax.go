// server/internal/agent/minimax.go
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
	minimaxDefaultModel = "MiniMax-M2.7"
	minimaxTimeout      = 300 * time.Second
)

// MinimaxProvider Minimax 模型适配器
type MinimaxProvider struct {
	apiKey  string
	baseURL string
	client *resty.Client
}

// NewMinimaxProvider 创建 Minimax Provider 实例
func NewMinimaxProvider(apiKey string) *MinimaxProvider {
	return NewMinimaxProviderWithBaseURL(apiKey, "https://api.minimaxi.com/v1")
}

// NewMinimaxProviderWithBaseURL 使用自定义 base URL 创建 Minimax Provider 实例
func NewMinimaxProviderWithBaseURL(apiKey, baseURL string) *MinimaxProvider {
	client := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(minimaxTimeout).
		SetHeader("Content-Type", "application/json")

	return &MinimaxProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  client,
	}
}

// SetAPIKey 动态设置 API Key
func (m *MinimaxProvider) SetAPIKey(apiKey string) {
	m.apiKey = apiKey
}

// SetBaseURL 动态设置 Base URL
func (m *MinimaxProvider) SetBaseURL(baseURL string) {
	m.baseURL = baseURL
	m.client.SetBaseURL(baseURL)
}

// minimaxChatRequest Minimax Chat API 请求体
type minimaxChatRequest struct {
	Model      string                  `json:"model"`
	Messages   []minimaxChatMessage   `json:"messages"`
	MaxTokens  int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Tools      []map[string]any       `json:"tools,omitempty"`
}

type minimaxChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// minimaxChatResponse Minimax Chat API 响应体
type minimaxChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Content          string `json:"content"`
			Role             string `json:"role"`
			Name             string `json:"name"`
			AudioContent     string `json:"audio_content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
	Created int `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   *struct {
		TotalTokens      int `json:"total_tokens"`
		TotalCharacters  int `json:"total_characters"`
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage,omitempty"`
	InputSensitive  bool `json:"input_sensitive"`
	OutputSensitive bool `json:"output_sensitive"`
	BaseResp        *struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateText 生成文本
func (m *MinimaxProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	// 构建消息列表
	messages := make([]minimaxChatMessage, 0, len(req.History)+2)

	// 系统提示
	if req.CharacterCtx != "" {
		messages = append(messages, minimaxChatMessage{
			Role:    "system",
			Content: req.CharacterCtx,
			Name:    "MiniMax AI",
		})
	}

	// 历史消息
	for _, h := range req.History {
		msg := minimaxChatMessage{
			Role:    h.Role,
			Content: h.Content,
		}
		if h.Name != "" {
			msg.Name = h.Name
		}
		messages = append(messages, msg)
	}

	// 用户提示
	if req.Prompt != "" {
		messages = append(messages, minimaxChatMessage{
			Role:    "user",
			Content: req.Prompt,
			Name:    "用户",
		})
	}

	// 设置 max_tokens
	maxTokens := req.MaxTokens
	if maxTokens > 8192 {
		maxTokens = 8192
	}

	chatReq := minimaxChatRequest{
		Model:       minimaxDefaultModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		Tools:       req.Tools,
	}

	// 优先使用请求指定的模型版本
	if req.Model != "" {
		chatReq.Model = req.Model
	}

	body, _ := json.Marshal(chatReq)

	resp, err := m.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+m.apiKey).
		SetBody(body).
		Post("/text/chatcompletion_v2")

	if err != nil {
		return nil, fmt.Errorf("minimax API request failed: %w", err)
	}

	var chatResp minimaxChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("minimax API response parse failed: %w, raw: %s", err, string(resp.Body()))
	}

	// 检查 base_resp 错误
	if chatResp.BaseResp != nil && chatResp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("minimax API error [%d]: %s", chatResp.BaseResp.StatusCode, chatResp.BaseResp.StatusMsg)
	}

	// 检查 error 字段
	if chatResp.Error != nil {
		return nil, fmt.Errorf("minimax API error [%d]: %s", chatResp.Error.Code, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("minimax API returned empty choices")
	}

	result := &TextResponse{
		Content: chatResp.Choices[0].Message.Content,
	}

	// 提取 token 消耗统计
	if chatResp.Usage != nil {
		result.Usage = &TokenUsage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		}
	}

	return result, nil
}

// GenerateImage 生成图像（Minimax 暂不支持，返回错误）
func (m *MinimaxProvider) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	return nil, errors.New("minimax does not support image generation")
}

// AdjustCharacter 角色调整（Minimax 暂不支持，返回错误）
func (m *MinimaxProvider) AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error) {
	return nil, errors.New("minimax does not support character adjustment")
}

// Embedding 文本向量化（Minimax 暂不支持，返回错误）
func (m *MinimaxProvider) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, errors.New("minimax does not support embedding")
}

// Name 返回 Provider 名称
func (m *MinimaxProvider) Name() string {
	return "minimax"
}

// FallbackModels 返回同 Provider 内的降级模型列表
func (m *MinimaxProvider) FallbackModels() []string {
	return []string{"MiniMax-M2.7", "abab6.5-chat", "abab6-chat"}
}

// Capabilities 返回 Minimax 支持的能力列表
func (m *MinimaxProvider) Capabilities() []string {
	return []string{
		"text_gen",
		"text_polish",
		"storyboard",
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
