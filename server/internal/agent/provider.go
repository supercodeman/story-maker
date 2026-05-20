// server/internal/agent/provider.go
package agent

import "context"

// ChatMessage 对话历史消息
type ChatMessage struct {
	Role      string     `json:"role"`                 // user, assistant, system, tool
	Content   string     `json:"content"`
	Name      string     `json:"name,omitempty"`       // tool 角色时的工具名
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // assistant 角色时的工具调用
	ToolCallID string    `json:"tool_call_id,omitempty"` // tool 角色时对应的 call ID
}

// TextRequest 文本生成请求
type TextRequest struct {
	Model        string            `json:"model"`         // 指定模型版本，空则用 Provider 默认值
	Prompt       string            `json:"prompt"`
	CharacterCtx string            `json:"character_ctx"` // 角色上下文约束
	History      []ChatMessage     `json:"history"`       // 多轮对话历史
	MaxTokens    int               `json:"max_tokens"`
	Temperature  float64           `json:"temperature"`
	Extra        map[string]string `json:"extra"`          // 扩展参数
	Tools        []map[string]any  `json:"tools,omitempty"` // Function Calling 工具定义
}

// TokenUsage API 返回的 token 消耗统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// TextResponse 文本生成响应
type TextResponse struct {
	Content   string      `json:"content"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"` // LLM 返回的工具调用请求
	Usage     *TokenUsage `json:"usage,omitempty"`      // token 消耗统计
}

// ImageRequest 图像生成请求
type ImageRequest struct {
	Model        string `json:"model"`         // 指定图像模型版本，空则用 Provider 默认值
	Prompt       string `json:"prompt"`
	ReferenceURL string `json:"reference_url"` // 参考图 URL
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Style        string `json:"style"` // 风格参数
}

// ImageResponse 图像生成响应
type ImageResponse struct {
	ImageURL string `json:"image_url"` // 生成的图片 URL
	FilePath string `json:"file_path"` // 本地存储路径
}

// CharacterAdjustRequest 角色调整请求
type CharacterAdjustRequest struct {
	CharacterID  uint              `json:"character_id"`
	ReferenceIDs []uint            `json:"reference_ids"` // 参考图 ID 列表
	Prompt       string            `json:"prompt"`
	Attributes   map[string]string `json:"attributes"` // 角色属性（发型、服装等）
}

// EmbeddingRequest Embedding 请求
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Texts []string `json:"texts"`
}

// EmbeddingResponse Embedding 响应
type EmbeddingResponse struct {
	Vectors   [][]float64 `json:"vectors"`
	Dimension int         `json:"dimension"`
}

// TTSRequest 文本转语音请求
type TTSRequest struct {
	Text    string  `json:"text"`     // 待转换文本
	VoiceID string  `json:"voice_id"` // 音色ID
	Speed   float64 `json:"speed"`    // 语速（0.5-2.0）
	Emotion string  `json:"emotion"`  // 情感标签（neutral, happy, sad 等）
}

// TTSResponse 文本转语音响应
type TTSResponse struct {
	AudioURL string  `json:"audio_url"` // 音频文件 URL
	FilePath string  `json:"file_path"` // 本地存储路径
	Duration float64 `json:"duration"`  // 音频时长（秒）
}

// VideoGenRequest 视频生成请求
type VideoGenRequest struct {
	Prompt             string  `json:"prompt"`               // 场景描述文本
	Model              string  `json:"model"`                // 模型版本
	ReferenceImagePath string  `json:"reference_image_path"` // 参考图本地路径
	Duration           float64 `json:"duration"`             // 期望时长（秒）
}

// VideoGenResponse 视频生成响应
type VideoGenResponse struct {
	VideoURL string  `json:"video_url"` // 视频文件 URL
	FilePath string  `json:"file_path"` // 本地存储路径
	Duration float64 `json:"duration"`  // 视频时长（秒）
}

// AIProvider AI 模型提供商统一接口
type AIProvider interface {
	// GenerateText 生成文本（剧本、对话、润色等）
	GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error)

	// GenerateImage 生成图像
	GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error)

	// AdjustCharacter 角色调整（基于参考图和属性生成一致性角色图）
	AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error)

	// Embedding 文本向量化
	Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// Name 返回 Provider 名称
	Name() string

	// Capabilities 返回支持的能力列表
	Capabilities() []string

	// FallbackModels 返回同 Provider 内的降级模型列表（按优先级排列）
	FallbackModels() []string
}

// TTSProvider 文本转语音 Provider 接口
type TTSProvider interface {
	// GenerateSpeech 将文本转换为语音
	GenerateSpeech(ctx context.Context, req *TTSRequest) (*TTSResponse, error)
	// Name 返回 Provider 名称
	Name() string
}

// VideoProvider 视频生成 Provider 接口
type VideoProvider interface {
	// GenerateVideo 根据文本描述生成视频
	GenerateVideo(ctx context.Context, req *VideoGenRequest) (*VideoGenResponse, error)
	// Name 返回 Provider 名称
	Name() string
}

// T2IRequest 文生图请求（独立于 AIProvider 的 ImageRequest）
type T2IRequest struct {
	Prompt            string   `json:"prompt"`
	AspectRatio       string   `json:"aspect_ratio"`        // "1:1", "16:9", "9:16" 等
	N                 int      `json:"n"`                   // 生成数量 1-4
	CharacterRefPaths []string `json:"character_ref_paths"` // 角色参考图本地路径（可选）
}

// T2IResponse 文生图响应
type T2IResponse struct {
	Images []ImageResult `json:"images"`
}

// ImageResult 单张图片结果
type ImageResult struct {
	URL      string `json:"url"`
	FilePath string `json:"file_path"`
}

// ImageGenProvider 文生图 Provider 接口（独立于 AIProvider，专用于 T2I 任务）
type ImageGenProvider interface {
	// GenerateImages 根据文本描述生成图片
	GenerateImages(ctx context.Context, req *T2IRequest) (*T2IResponse, error)
	// Name 返回 Provider 名称
	Name() string
}
