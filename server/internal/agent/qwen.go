// server/internal/agent/qwen.go
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
	qwenBaseURL           = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	qwenChatEndpoint      = "/chat/completions"
	qwenImageEndpoint     = "/images/generations"
	qwenDefaultModel      = "deepseek-v3"
	qwenDefaultImageModel = "wanx-2.1"
	qwenTimeout           = 300 * time.Second
)

// QwenProvider 通义千问模型适配器
type QwenProvider struct {
	apiKey string
	client *resty.Client
}

// NewQwenProvider 创建千问 Provider 实例
func NewQwenProvider(apiKey string) *QwenProvider {
	client := resty.New().
		SetBaseURL(qwenBaseURL).
		SetTimeout(qwenTimeout).
		SetHeader("Content-Type", "application/json")

	return &QwenProvider{
		apiKey: apiKey,
		client: client,
	}
}

// SetAPIKey 动态设置 API Key
func (q *QwenProvider) SetAPIKey(apiKey string) {
	q.apiKey = apiKey
}

// qwenChatRequest 千问 Chat API 请求体（OpenAI 兼容格式）
type qwenChatRequest struct {
	Model        string            `json:"model"`
	Messages     []qwenChatMessage `json:"messages"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
	Temperature  float64           `json:"temperature,omitempty"`
	Tools        []map[string]any  `json:"tools,omitempty"`
	EnableSearch bool              `json:"enable_search,omitempty"`
}

type qwenChatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []qwenToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// qwenToolCall 千问工具调用结构（OpenAI 兼容）
type qwenToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// qwenChatResponse 千问 Chat API 响应体
type qwenChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string         `json:"content"`
			ToolCalls []qwenToolCall `json:"tool_calls,omitempty"`
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
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateText 生成文本
func (q *QwenProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	// 构建消息列表
	messages := make([]qwenChatMessage, 0, len(req.History)+2)

	// 系统提示
	if req.CharacterCtx != "" {
		messages = append(messages, qwenChatMessage{Role: "system", Content: req.CharacterCtx})
	}

	// 历史消息
	for _, h := range req.History {
		msg := qwenChatMessage{Role: h.Role, Content: h.Content}
		if h.ToolCallID != "" {
			msg.ToolCallID = h.ToolCallID
		}
		for _, tc := range h.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, qwenToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{Name: tc.Name, Arguments: marshalArgs(tc.Arguments)},
			})
		}
		messages = append(messages, msg)
	}

	// 用户提示（tool call 后续轮次 Prompt 为空，由 history 驱动，不追加空 user 消息）
	if req.Prompt != "" {
		messages = append(messages, qwenChatMessage{Role: "user", Content: req.Prompt})
	}

	// qwen-max max_tokens 上限 8192
	maxTokens := req.MaxTokens
	if maxTokens > 8192 {
		maxTokens = 8192
	}

	chatReq := qwenChatRequest{
		Model:       qwenDefaultModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		Tools:       req.Tools,
	}

	// 优先使用请求指定的模型版本
	if req.Model != "" {
		chatReq.Model = req.Model
	}

	// 通义千问联网搜索通过请求体顶层 enable_search 字段控制
	if req.Extra["enable_search"] == "true" {
		chatReq.EnableSearch = true
	}

	body, _ := json.Marshal(chatReq)

	resp, err := q.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+q.apiKey).
		SetBody(body).
		Post(qwenChatEndpoint)

	if err != nil {
		return nil, fmt.Errorf("qwen API request failed: %w", err)
	}

	var chatResp qwenChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("qwen API response parse failed: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("qwen API error [%s]: %s", chatResp.Error.Code, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("qwen API returned empty choices")
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

	// 转换 tool_calls
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

// qwenImageRequest 千问图像生成请求体（Dashscope OpenAI 兼容格式）
type qwenImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
	N      int    `json:"n"`
}

// qwenImageResponse 千问图像生成响应体
type qwenImageResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateImage 千问文生图（通过 Dashscope wanx 系列模型）
func (q *QwenProvider) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	imageModel := qwenDefaultImageModel
	if req.Model != "" {
		imageModel = req.Model
	}

	size := "1024*1024"
	if req.Width > 0 && req.Height > 0 {
		size = fmt.Sprintf("%d*%d", req.Width, req.Height)
	}

	imgReq := qwenImageRequest{
		Model:  imageModel,
		Prompt: req.Prompt,
		Size:   size,
		N:      1,
	}

	body, _ := json.Marshal(imgReq)

	resp, err := q.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+q.apiKey).
		SetBody(body).
		Post(qwenImageEndpoint)

	if err != nil {
		return nil, fmt.Errorf("qwen image API request failed: %w", err)
	}

	var imgResp qwenImageResponse
	if err := json.Unmarshal(resp.Body(), &imgResp); err != nil {
		return nil, fmt.Errorf("qwen image API response parse failed: %w", err)
	}

	if imgResp.Error != nil {
		return nil, fmt.Errorf("qwen image API error [%s]: %s", imgResp.Error.Code, imgResp.Error.Message)
	}

	if len(imgResp.Data) == 0 {
		return nil, errors.New("qwen image API returned empty data")
	}

	return &ImageResponse{
		ImageURL: imgResp.Data[0].URL,
	}, nil
}

// AdjustCharacter 千问角色调整（基于文生图能力）
func (q *QwenProvider) AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error) {
	prompt := req.Prompt
	if len(req.Attributes) > 0 {
		prompt += "\n\n角色属性:"
		for key, value := range req.Attributes {
			prompt += fmt.Sprintf("\n- %s: %s", key, value)
		}
	}
	prompt += "\n\n请保持角色与参考图的一致性。"

	imgReq := &ImageRequest{
		Prompt: prompt,
		Width:  1024,
		Height: 1024,
	}

	return q.GenerateImage(ctx, imgReq)
}

// Name 返回 Provider 名称
func (q *QwenProvider) Name() string {
	return "qwen"
}

// FallbackModels 返回同 Provider 内的降级模型列表
func (q *QwenProvider) FallbackModels() []string {
	return []string{"deepseek-v3.2", "deepseek-v3", "MiniMax-M2.5"}
}

// Capabilities 返回千问支持的能力列表
func (q *QwenProvider) Capabilities() []string {
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
		"embedding",
		"outline_generate_characters",
		"butler_generate_topic",
		"butler_generate_storyline",
		"butler_generate_characters",
	}
}

// qwenEmbeddingRequest 通义 Embedding API 请求体（OpenAI 兼容模式）
type qwenEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// qwenEmbeddingResponse 通义 Embedding API 响应体
type qwenEmbeddingResponse struct {
	Output struct {
		Embeddings []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"embeddings"`
	} `json:"output"`
	// 兼容 OpenAI 格式
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Embedding 文本向量化
func (q *QwenProvider) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	embModel := "text-embedding-v3"
	if req.Model != "" {
		embModel = req.Model
	}

	body := qwenEmbeddingRequest{
		Model: embModel,
		Input: req.Texts,
	}

	resp, err := q.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+q.apiKey).
		SetBody(body).
		Post("/embeddings")
	if err != nil {
		return nil, fmt.Errorf("qwen embedding request failed: %w", err)
	}

	var result qwenEmbeddingResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("qwen embedding parse failed: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("qwen embedding error: %s", result.Error.Message)
	}

	var allVectors [][]float64
	// 优先使用 OpenAI 兼容格式
	if len(result.Data) > 0 {
		for _, d := range result.Data {
			allVectors = append(allVectors, d.Embedding)
		}
	} else if len(result.Output.Embeddings) > 0 {
		for _, e := range result.Output.Embeddings {
			allVectors = append(allVectors, e.Embedding)
		}
	} else {
		return nil, errors.New("qwen embedding returned empty data")
	}

	dimension := 0
	if len(allVectors) > 0 {
		dimension = len(allVectors[0])
	}

	return &EmbeddingResponse{
		Vectors:   allVectors,
		Dimension: dimension,
	}, nil
}
