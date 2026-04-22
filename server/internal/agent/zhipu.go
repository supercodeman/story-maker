// server/internal/agent/zhipu.go
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
	zhipuBaseURL       = "https://open.bigmodel.cn/api/paas/v4"
	zhipuChatEndpoint  = "/chat/completions"
	zhipuImageEndpoint = "/images/generations"
	zhipuDefaultModel  = "glm-4.5-flash"
	zhipuImageModel    = "cogview-3"
	zhipuTimeout       = 600 * time.Second
)

// ZhipuProvider 智谱 AI 模型适配器
type ZhipuProvider struct {
	apiKey string
	client *resty.Client
}

// NewZhipuProvider 创建智谱 Provider 实例
func NewZhipuProvider(apiKey string) *ZhipuProvider {
	client := resty.New().
		SetBaseURL(zhipuBaseURL).
		SetTimeout(zhipuTimeout).
		SetHeader("Content-Type", "application/json")

	return &ZhipuProvider{
		apiKey: apiKey,
		client: client,
	}
}

// SetAPIKey 动态设置 API Key
func (z *ZhipuProvider) SetAPIKey(apiKey string) {
	z.apiKey = apiKey
}

// zhipuChatRequest 智谱 Chat API 请求体
type zhipuChatRequest struct {
	Model       string             `json:"model"`
	Messages    []zhipuChatMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	Tools       []map[string]any   `json:"tools,omitempty"`
}

type zhipuChatMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []zhipuToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// zhipuToolCall 智谱工具调用结构
type zhipuToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON 字符串
	} `json:"function"`
}

// zhipuChatResponse 智谱 Chat API 响应体
type zhipuChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string          `json:"content"`
			ToolCalls []zhipuToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// zhipuImageRequest 智谱图像生成请求体
type zhipuImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
}

// zhipuImageResponse 智谱图像生成响应体
type zhipuImageResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateText 调用智谱 GLM-4 生成文本（支持 Function Calling）
func (z *ZhipuProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	if z.apiKey == "" {
		return nil, errors.New("zhipu api key is not set")
	}

	messages := make([]zhipuChatMessage, 0, len(req.History)+2)

	if req.CharacterCtx != "" {
		messages = append(messages, zhipuChatMessage{
			Role:    "system",
			Content: req.CharacterCtx,
		})
	}

	// 追加历史对话（包含 tool 相关消息）
	for _, h := range req.History {
		msg := zhipuChatMessage{
			Role:       h.Role,
			Content:    h.Content,
			ToolCallID: h.ToolCallID,
		}
		// 转换 assistant 消息中的 ToolCalls
		for _, tc := range h.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, zhipuToolCall{
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

	// 追加当前用户输入（仅当 Prompt 非空时，tool 循环中可能为空）
	if req.Prompt != "" {
		messages = append(messages, zhipuChatMessage{
			Role:    "user",
			Content: req.Prompt,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 16384
	}

	temperature := req.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}

	chatReq := zhipuChatRequest{
		Model:       zhipuDefaultModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Tools:       req.Tools,
	}

	// 优先使用请求指定的模型版本
	if req.Model != "" {
		chatReq.Model = req.Model
	}

	resp, err := z.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+z.apiKey).
		SetBody(chatReq).
		Post(zhipuChatEndpoint)

	if err != nil {
		return nil, fmt.Errorf("zhipu text request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("zhipu text API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	var chatResp zhipuChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("zhipu text response parse failed: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("zhipu text API error [%s]: %s", chatResp.Error.Code, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("zhipu text API returned empty choices")
	}

	choice := chatResp.Choices[0].Message
	textResp := &TextResponse{
		Content: choice.Content,
	}

	// 提取 token 消耗统计
	if chatResp.Usage != nil {
		textResp.Usage = &TokenUsage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		}
	}

	// 解析 tool_calls
	for _, tc := range choice.ToolCalls {
		var args map[string]any
		if tc.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		textResp.ToolCalls = append(textResp.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return textResp, nil
}

// marshalArgs 将 map 序列化为 JSON 字符串
func marshalArgs(args map[string]any) string {
	if args == nil {
		return "{}"
	}
	b, _ := json.Marshal(args)
	return string(b)
}

// GenerateImage 调用智谱 CogView 生成图像
func (z *ZhipuProvider) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	if z.apiKey == "" {
		return nil, errors.New("zhipu api key is not set")
	}

	imgReq := zhipuImageRequest{
		Model:  zhipuImageModel,
		Prompt: req.Prompt,
	}

	// 优先使用请求指定的图像模型版本
	if req.Model != "" {
		imgReq.Model = req.Model
	}

	// CogView 支持的尺寸：1024x1024, 768x1344, 864x1152, 1344x768, 1152x864, 1440x720, 720x1440
	if req.Width > 0 && req.Height > 0 {
		imgReq.Size = fmt.Sprintf("%dx%d", req.Width, req.Height)
	}

	resp, err := z.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+z.apiKey).
		SetBody(imgReq).
		Post(zhipuImageEndpoint)

	if err != nil {
		return nil, fmt.Errorf("zhipu image request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("zhipu image API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	var imgResp zhipuImageResponse
	if err := json.Unmarshal(resp.Body(), &imgResp); err != nil {
		return nil, fmt.Errorf("zhipu image response parse failed: %w", err)
	}

	if imgResp.Error != nil {
		return nil, fmt.Errorf("zhipu image API error [%s]: %s", imgResp.Error.Code, imgResp.Error.Message)
	}

	if len(imgResp.Data) == 0 || imgResp.Data[0].URL == "" {
		return nil, errors.New("zhipu image API returned empty result")
	}

	return &ImageResponse{
		ImageURL: imgResp.Data[0].URL,
	}, nil
}

// AdjustCharacter 角色调整：组装角色约束提示词 + 调用图像生成
func (z *ZhipuProvider) AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error) {
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

	return z.GenerateImage(ctx, imgReq)
}

// Name 返回 Provider 名称
func (z *ZhipuProvider) Name() string {
	return "zhipu"
}

// FallbackModels 返回同 Provider 内的降级模型列表
func (z *ZhipuProvider) FallbackModels() []string {
	return nil // 降级链已由 DB 驱动，不再硬编码
}

// Capabilities 返回智谱支持的能力列表
func (z *ZhipuProvider) Capabilities() []string {
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

// zhipuEmbeddingRequest 智谱 Embedding API 请求体
type zhipuEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// zhipuEmbeddingResponse 智谱 Embedding API 响应体
type zhipuEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Embedding 文本向量化
func (z *ZhipuProvider) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	embModel := "embedding-3"
	if req.Model != "" {
		embModel = req.Model
	}

	var allVectors [][]float64
	dimension := 0

	// 智谱 Embedding API 单次只支持一个文本，逐个调用
	for _, text := range req.Texts {
		body := zhipuEmbeddingRequest{
			Model: embModel,
			Input: text,
		}

		resp, err := z.client.R().
			SetContext(ctx).
			SetHeader("Authorization", "Bearer "+z.apiKey).
			SetBody(body).
			Post("/embeddings")
		if err != nil {
			return nil, fmt.Errorf("zhipu embedding request failed: %w", err)
		}

		var result zhipuEmbeddingResponse
		if err := json.Unmarshal(resp.Body(), &result); err != nil {
			return nil, fmt.Errorf("zhipu embedding parse failed: %w", err)
		}
		if result.Error != nil {
			return nil, fmt.Errorf("zhipu embedding error: %s", result.Error.Message)
		}
		if len(result.Data) == 0 {
			return nil, errors.New("zhipu embedding returned empty data")
		}

		vec := result.Data[0].Embedding
		allVectors = append(allVectors, vec)
		if dimension == 0 {
			dimension = len(vec)
		}
	}

	return &EmbeddingResponse{
		Vectors:   allVectors,
		Dimension: dimension,
	}, nil
}
