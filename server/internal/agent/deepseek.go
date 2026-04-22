// server/internal/agent/deepseek.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	deepseekBaseURL      = "http://127.0.0.1:11434/v1"
	deepseekChatEndpoint = "/chat/completions"
	deepseekDefaultModel = "deepseek-r1:latest"
	deepseekTimeout      = 120 * time.Second
)

// DeepSeekProvider 深度求索模型适配器
type DeepSeekProvider struct {
	apiKey string
	client *resty.Client
}

// NewDeepSeekProvider 创建 DeepSeek Provider 实例，apiKey 应为 sk-xxx 格式
func NewDeepSeekProvider(apiKey string) *DeepSeekProvider {
	// 清理 API Key：去除首尾空白字符
	cleanKey := strings.TrimSpace(apiKey)

	client := resty.New().
		SetBaseURL(deepseekBaseURL).
		SetTimeout(deepseekTimeout).
		SetHeader("Content-Type", "application/json").
		// 关键修复：在客户端层面统一设置 Authorization 头
		SetHeader("Authorization", "Bearer "+cleanKey)

	return &DeepSeekProvider{
		apiKey: cleanKey,
		client: client,
	}
}

// SetAPIKey 动态更新 API Key（自动清理空白）
func (d *DeepSeekProvider) SetAPIKey(apiKey string) {
	cleanKey := strings.TrimSpace(apiKey)
	d.apiKey = cleanKey
	// 更新 client 的默认 Authorization 头
	d.client.SetHeader("Authorization", "Bearer "+cleanKey)
}

// deepseekChatRequest 请求体（OpenAI 兼容格式）
type deepseekChatRequest struct {
	Model       string                `json:"model"`
	Messages    []deepseekChatMessage `json:"messages"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Temperature float64               `json:"temperature,omitempty"`
	TopP        float64               `json:"top_p,omitempty"`
	Tools       []map[string]any      `json:"tools,omitempty"`
	Stream      bool                  `json:"stream,omitempty"`
}

type deepseekChatMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	Name       string             `json:"name,omitempty"`
	ToolCalls  []deepseekToolCall `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
}

type deepseekToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// deepseekChatResponse 成功响应
type deepseekChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string             `json:"content"`
			ToolCalls []deepseekToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// deepseekErrorResponse 错误响应
type deepseekErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// GenerateText 生成文本
func (d *DeepSeekProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	if req == nil {
		return nil, errors.New("text request is nil")
	}
	if d.apiKey == "" {
		return nil, errors.New("deepseek api key is not set")
	}

	// 构建消息列表
	messages := make([]deepseekChatMessage, 0, len(req.History)+2)

	if req.CharacterCtx != "" {
		messages = append(messages, deepseekChatMessage{Role: "system", Content: req.CharacterCtx})
	}

	for _, h := range req.History {
		msg := deepseekChatMessage{
			Role:       h.Role,
			Content:    h.Content,
			Name:       h.Name,
			ToolCallID: h.ToolCallID,
		}
		for _, tc := range h.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, deepseekToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Name,
					Arguments: marshalArgs(tc.Arguments),
				},
			})
		}
		messages = append(messages, msg)
	}

	if req.Prompt != "" {
		messages = append(messages, deepseekChatMessage{Role: "user", Content: req.Prompt})
	}

	chatReq := deepseekChatRequest{
		Model:       deepseekDefaultModel,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Tools:       req.Tools,
	}

	//if req.Model != "" {
	//	chatReq.Model = req.Model
	//}
	if req.Extra != nil {
		if topP, ok := req.Extra["top_p"]; ok {
			var tp float64
			if _, err := fmt.Sscanf(topP, "%f", &tp); err == nil {
				chatReq.TopP = tp
			}
		}
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 发起请求（注意：Authorization 头已在 client 层面设置，无需每次重复）
	resp, err := d.client.R().
		SetContext(ctx).
		SetBody(body).
		Post(deepseekChatEndpoint)

	if err != nil {
		return nil, fmt.Errorf("deepseek API request failed: %w", err)
	}

	// 处理非 200 状态码
	if resp.StatusCode() != 200 {
		rawBody := resp.String()
		// 尝试解析标准错误格式
		var errResp deepseekErrorResponse
		if unmarshalErr := json.Unmarshal(resp.Body(), &errResp); unmarshalErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("deepseek API error (status %d): %s [%s]",
				resp.StatusCode(), errResp.Error.Message, errResp.Error.Code)
		}
		return nil, fmt.Errorf("deepseek API returned %d, body: %s", resp.StatusCode(), rawBody)
	}

	var chatResp deepseekChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w, raw: %s", err, resp.String())
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("empty choices in response")
	}

	result := &TextResponse{
		Content: chatResp.Choices[0].Message.Content,
	}
	if chatResp.Usage != nil {
		result.Usage = &TokenUsage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		}
	}
	for _, tc := range chatResp.Choices[0].Message.ToolCalls {
		var args map[string]any
		if tc.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return result, nil
}

// GenerateImage 不支持
func (d *DeepSeekProvider) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	return nil, fmt.Errorf("deepseek does not support image generation")
}

// AdjustCharacter 不支持
func (d *DeepSeekProvider) AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error) {
	return nil, fmt.Errorf("deepseek does not support character adjustment")
}

// Name 返回名称
func (d *DeepSeekProvider) Name() string {
	return "deepseek"
}

// FallbackModels 降级模型
func (d *DeepSeekProvider) FallbackModels() []string {
	return nil // 降级链已由 DB 驱动，不再硬编码
}

// Capabilities 能力列表
func (d *DeepSeekProvider) Capabilities() []string {
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

// Embedding 不支持
func (d *DeepSeekProvider) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("deepseek does not support embedding")
}
