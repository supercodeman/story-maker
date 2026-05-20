# Plan 3: AI Agent 层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 AI 多模型调度框架，接入 Kimi，支持文本生成、图像生成、角色调整，异步任务执行 + WebSocket 推送。

**Architecture:** Provider 适配器模式统一多模型接口，Dispatcher 负责路由和异步执行，WebSocket Hub 负责实时推送。

**Tech Stack:** Go, Gin, gorilla/websocket, Kimi API, AES-256-GCM

---

### Task 1: AI 任务模型

**Files:**
- Create: `server/internal/model/ai_task.go`

- [ ] **Step 1: 创建 AITask 模型文件**

```go
// server/internal/model/ai_task.go
package model

import "time"

// AITask AI 任务表，记录每次 AI 调用的完整生命周期
type AITask struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	PortfolioID uint      `gorm:"index" json:"portfolio_id"`
	TaskType    string    `gorm:"size:50" json:"task_type"`     // text_gen, image_gen, character_adjust
	ModelName   string    `gorm:"size:50" json:"model_name"`    // kimi, claude, copilot
	Prompt      string    `gorm:"type:text" json:"prompt"`
	Status      string    `gorm:"size:20;default:pending" json:"status"` // pending, running, completed, failed, cancelled
	Result      string    `gorm:"type:json" json:"result"`
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AITask 状态常量
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

// AITask 任务类型常量
const (
	TaskTypeTextGen         = "text_gen"
	TaskTypeTextPolish      = "text_polish"
	TaskTypeStoryboard      = "storyboard"
	TaskTypeImageGen        = "image_gen"
	TaskTypeImageEdit       = "image_edit"
	TaskTypeCharacterAdjust = "character_adjust"
)
```


- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/model/ai_task.go
git commit -m "feat: add AITask model with status and type constants"
```

---

### Task 2: API Key 模型 + 加密工具

**Files:**
- Create: `server/internal/model/api_key.go`
- Create: `server/internal/util/crypto.go`

- [ ] **Step 1: 创建 APIKey 模型文件**

```go
// server/internal/model/api_key.go
package model

import "time"

// APIKey 用户 API Key 管理表，支持用户自有 Key 和平台默认 Key
type APIKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Provider  string    `gorm:"size:50" json:"provider"` // kimi, claude, copilot
	KeyValue  string    `gorm:"size:500" json:"-"`       // 加密存储，不返回给前端
	IsDefault bool      `json:"is_default"`              // 是否为该 Provider 的默认 Key
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Provider 常量
const (
	ProviderKimi    = "kimi"
	ProviderClaude  = "claude"
	ProviderCopilot = "copilot"
)
```

- [ ] **Step 2: 创建加密工具文件**

```go
// server/internal/util/crypto.go
package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptAES 使用 AES-256-GCM 加密数据
// key 必须是 32 字节（256 位）
func EncryptAES(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES 使用 AES-256-GCM 解密数据
func DecryptAES(ciphertext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("key must be 32 bytes for AES-256")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/model/api_key.go server/internal/util/crypto.go
git commit -m "feat: add APIKey model and AES-256-GCM encryption utilities"
```

---

### Task 3: AI Provider 接口定义

**Files:**
- Create: `server/internal/agent/provider.go`

- [ ] **Step 1: 创建 Provider 接口文件**

```go
// server/internal/agent/provider.go
package agent

import "context"

// TextRequest 文本生成请求
type TextRequest struct {
	Prompt       string            `json:"prompt"`
	CharacterCtx string            `json:"character_ctx"` // 角色上下文约束
	MaxTokens    int               `json:"max_tokens"`
	Temperature  float64           `json:"temperature"`
	Extra        map[string]string `json:"extra"` // 扩展参数
}

// TextResponse 文本生成响应
type TextResponse struct {
	Content string `json:"content"`
}

// ImageRequest 图像生成请求
type ImageRequest struct {
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

// AIProvider AI 模型提供商统一接口
type AIProvider interface {
	// GenerateText 生成文本（剧本、对话、润色等）
	GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error)

	// GenerateImage 生成图像
	GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error)

	// AdjustCharacter 角色调整（基于参考图和属性生成一致性角色图）
	AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error)

	// Name 返回 Provider 名称
	Name() string

	// Capabilities 返回支持的能力列表
	Capabilities() []string
}

// Capability 能力常量
const (
	CapTextGen         = "text_gen"
	CapTextPolish      = "text_polish"
	CapStoryboard      = "storyboard"
	CapImageGen        = "image_gen"
	CapImageEdit       = "image_edit"
	CapCharacterAdjust = "character_adjust"
)
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/agent/provider.go
git commit -m "feat: define AIProvider interface with text/image/character capabilities"
```

---

### Task 4: Kimi 适配器

**Files:**
- Create: `server/internal/agent/kimi.go`

- [ ] **Step 1: 安装 HTTP 依赖（如尚未安装）**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go get github.com/go-resty/resty/v2@v2.11.0
```

- [ ] **Step 2: 创建 Kimi 适配器文件**

```go
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
	kimiDefaultModel  = "moonshot-v1-8k"
	kimiTimeout       = 120 * time.Second
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
	messages := make([]kimiChatMessage, 0, 2)

	// 如果有角色上下文，作为 system 消息注入
	if req.CharacterCtx != "" {
		messages = append(messages, kimiChatMessage{
			Role:    "system",
			Content: req.CharacterCtx,
		})
	}

	messages = append(messages, kimiChatMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
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

	return &TextResponse{
		Content: chatResp.Choices[0].Message.Content,
	}, nil
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
	return ProviderKimi
}

// Capabilities 返回 Kimi 支持的能力列表
func (k *KimiProvider) Capabilities() []string {
	return []string{
		CapTextGen,
		CapTextPolish,
		CapStoryboard,
		CapImageGen,
		CapImageEdit,
		CapCharacterAdjust,
	}
}
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/agent/kimi.go
git commit -m "feat: implement Kimi provider adapter with text/image/character APIs"
```

---

### Task 5: Dispatcher（任务分发器）

**Files:**
- Create: `server/internal/agent/dispatcher.go`

- [ ] **Step 1: 创建 Dispatcher 文件**

```go
// server/internal/agent/dispatcher.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"story-maker/server/internal/model"
)

// TaskResult 任务结果结构
type TaskResult struct {
	TaskID uint        `json:"task_id"`
	Status string      `json:"status"`
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

// Dispatcher AI 任务分发器，负责路由、异步执行、状态管理
type Dispatcher struct {
	providers map[string]AIProvider // model_name -> Provider
	keyStore  KeyStore              // API Key 存储接口
	taskStore TaskStore             // 任务存储接口
	notifier  Notifier              // WebSocket 通知接口
	mu        sync.RWMutex
}

// KeyStore API Key 存储接口（由 Service 层实现）
type KeyStore interface {
	GetUserKey(ctx context.Context, userID uint, provider string) (string, error)
	GetDefaultKey(ctx context.Context, provider string) (string, error)
}

// TaskStore 任务存储接口（由 DAO 层实现）
type TaskStore interface {
	CreateTask(ctx context.Context, task *model.AITask) error
	UpdateTask(ctx context.Context, task *model.AITask) error
	GetTask(ctx context.Context, taskID uint) (*model.AITask, error)
}

// Notifier WebSocket 通知接口
type Notifier interface {
	NotifyUser(userID uint, message interface{}) error
}

// NewDispatcher 创建 Dispatcher 实例
func NewDispatcher(keyStore KeyStore, taskStore TaskStore, notifier Notifier) *Dispatcher {
	return &Dispatcher{
		providers: make(map[string]AIProvider),
		keyStore:  keyStore,
		taskStore: taskStore,
		notifier:  notifier,
	}
}

// RegisterProvider 注册 Provider
func (d *Dispatcher) RegisterProvider(modelName string, provider AIProvider) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.providers[modelName] = provider
}

// GetProvider 根据 model_name 获取对应 Provider
func (d *Dispatcher) GetProvider(modelName string) (AIProvider, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	provider, ok := d.providers[modelName]
	if !ok {
		return nil, fmt.Errorf("provider not found for model: %s", modelName)
	}
	return provider, nil
}

// CheckCapability 检查模型是否支持指定任务类型
func (d *Dispatcher) CheckCapability(modelName string, taskType string) error {
	provider, err := d.GetProvider(modelName)
	if err != nil {
		return err
	}

	capabilities := provider.Capabilities()
	for _, cap := range capabilities {
		if cap == taskType {
			return nil
		}
	}

	return fmt.Errorf("model %s does not support task type %s", modelName, taskType)
}

// resolveKey 解析 API Key：优先用户 Key → 平台默认 Key
func (d *Dispatcher) resolveKey(ctx context.Context, userID uint, provider string) (string, error) {
	// 优先查找用户自己的 Key
	userKey, err := d.keyStore.GetUserKey(ctx, userID, provider)
	if err == nil && userKey != "" {
		return userKey, nil
	}

	// 回退到平台默认 Key
	defaultKey, err := d.keyStore.GetDefaultKey(ctx, provider)
	if err != nil {
		return "", fmt.Errorf("no available API key for provider %s", provider)
	}

	return defaultKey, nil
}

// Dispatch 分发任务：创建 AITask 记录 → goroutine 异步执行 → 更新状态 → 通知 WebSocket
func (d *Dispatcher) Dispatch(ctx context.Context, task *model.AITask) error {
	// 1. 检查能力支持
	if err := d.CheckCapability(task.ModelName, task.TaskType); err != nil {
		return err
	}

	// 2. 创建任务记录
	task.Status = model.TaskStatusPending
	if err := d.taskStore.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// 3. 异步执行任务
	go d.executeTask(context.Background(), task)

	return nil
}

// executeTask 异步执行任务
func (d *Dispatcher) executeTask(ctx context.Context, task *model.AITask) {
	// 更新状态为 running
	task.Status = model.TaskStatusRunning
	_ = d.taskStore.UpdateTask(ctx, task)
	d.notifyTaskUpdate(task.UserID, task.ID, task.Status, nil, "")

	// 解析 API Key
	apiKey, err := d.resolveKey(ctx, task.UserID, task.ModelName)
	if err != nil {
		d.handleTaskError(ctx, task, err)
		return
	}

	// 获取 Provider
	provider, err := d.GetProvider(task.ModelName)
	if err != nil {
		d.handleTaskError(ctx, task, err)
		return
	}

	// 动态设置 API Key（如果 Provider 支持）
	if kp, ok := provider.(*KimiProvider); ok {
		kp.SetAPIKey(apiKey)
	}

	// 根据任务类型调用对应方法
	var result interface{}
	switch task.TaskType {
	case model.TaskTypeTextGen, model.TaskTypeTextPolish, model.TaskTypeStoryboard:
		result, err = d.executeTextTask(ctx, provider, task)
	case model.TaskTypeImageGen, model.TaskTypeImageEdit:
		result, err = d.executeImageTask(ctx, provider, task)
	case model.TaskTypeCharacterAdjust:
		result, err = d.executeCharacterTask(ctx, provider, task)
	default:
		err = fmt.Errorf("unsupported task type: %s", task.TaskType)
	}

	if err != nil {
		d.handleTaskError(ctx, task, err)
		return
	}

	// 任务成功
	d.handleTaskSuccess(ctx, task, result)
}

// executeTextTask 执行文本任务
func (d *Dispatcher) executeTextTask(ctx context.Context, provider AIProvider, task *model.AITask) (interface{}, error) {
	req := &TextRequest{
		Prompt:    task.Prompt,
		MaxTokens: 2048,
	}

	resp, err := provider.GenerateText(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content": resp.Content,
	}, nil
}

// executeImageTask 执行图像任务
func (d *Dispatcher) executeImageTask(ctx context.Context, provider AIProvider, task *model.AITask) (interface{}, error) {
	req := &ImageRequest{
		Prompt: task.Prompt,
		Width:  1024,
		Height: 1024,
	}

	resp, err := provider.GenerateImage(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image_url": resp.ImageURL,
		"file_path": resp.FilePath,
	}, nil
}

// executeCharacterTask 执行角色调整任务
func (d *Dispatcher) executeCharacterTask(ctx context.Context, provider AIProvider, task *model.AITask) (interface{}, error) {
	// 从 task.Prompt 解析 CharacterAdjustRequest（实际应从 task.Result 或单独字段存储）
	req := &CharacterAdjustRequest{
		Prompt: task.Prompt,
	}

	resp, err := provider.AdjustCharacter(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image_url": resp.ImageURL,
		"file_path": resp.FilePath,
	}, nil
}

// handleTaskSuccess 处理任务成功
func (d *Dispatcher) handleTaskSuccess(ctx context.Context, task *model.AITask, result interface{}) {
	task.Status = model.TaskStatusCompleted
	resultJSON, _ := json.Marshal(result)
	task.Result = string(resultJSON)
	_ = d.taskStore.UpdateTask(ctx, task)

	d.notifyTaskUpdate(task.UserID, task.ID, task.Status, result, "")
}

// handleTaskError 处理任务失败
func (d *Dispatcher) handleTaskError(ctx context.Context, task *model.AITask, err error) {
	task.Status = model.TaskStatusFailed
	task.ErrorMsg = err.Error()
	_ = d.taskStore.UpdateTask(ctx, task)

	d.notifyTaskUpdate(task.UserID, task.ID, task.Status, nil, err.Error())
}

// notifyTaskUpdate 通知任务状态更新
func (d *Dispatcher) notifyTaskUpdate(userID uint, taskID uint, status string, result interface{}, errorMsg string) {
	if d.notifier == nil {
		return
	}

	message := TaskResult{
		TaskID: taskID,
		Status: status,
		Result: result,
		Error:  errorMsg,
	}

	_ = d.notifier.NotifyUser(userID, message)
}

// CancelTask 取消任务（仅更新状态，不中断执行）
func (d *Dispatcher) CancelTask(ctx context.Context, taskID uint) error {
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status == model.TaskStatusCompleted || task.Status == model.TaskStatusFailed {
		return errors.New("cannot cancel completed or failed task")
	}

	task.Status = model.TaskStatusCancelled
	return d.taskStore.UpdateTask(ctx, task)
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/agent/dispatcher.go
git commit -m "feat: implement Dispatcher with async task execution and WebSocket notification"
```

---

### Task 6: WebSocket 推送

**Files:**
- Create: `server/internal/handler/ws.go`

- [ ] **Step 1: 安装 WebSocket 依赖**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go get github.com/gorilla/websocket@v1.5.1
```

- [ ] **Step 2: 创建 WebSocket Handler 文件**

```go
// server/internal/handler/ws.go
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境需严格校验 Origin
	},
}

// WSMessage WebSocket 消息结构
type WSMessage struct {
	Type string      `json:"type"` // task_update, system_notification
	Data interface{} `json:"data"`
}

// Client WebSocket 客户端连接
type Client struct {
	userID uint
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
}

// Hub WebSocket 连接管理中心
type Hub struct {
	clients    map[uint]map[*Client]bool // userID -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	UserID  uint
	Message []byte
}

// NewHub 创建 Hub 实例
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run 启动 Hub 主循环
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered: user_id=%d", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.userID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client unregistered: user_id=%d", client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.UserID]
			h.mu.RUnlock()

			for client := range clients {
				select {
				case client.send <- message.Message:
				default:
					close(client.send)
					h.mu.Lock()
					delete(h.clients[message.UserID], client)
					h.mu.Unlock()
				}
			}
		}
	}
}

// NotifyUser 向指定用户推送消息
func (h *Hub) NotifyUser(userID uint, message interface{}) error {
	wsMsg := WSMessage{
		Type: "task_update",
		Data: message,
	}

	msgBytes, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}

	h.broadcast <- &BroadcastMessage{
		UserID:  userID,
		Message: msgBytes,
	}

	return nil
}

// readPump 读取客户端消息（心跳检测）
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
	}
}

// writePump 向客户端写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量写入队列中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WSHandler WebSocket 连接处理器
type WSHandler struct {
	hub *Hub
}

// NewWSHandler 创建 WSHandler 实例
func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// HandleWebSocket 处理 WebSocket 连接请求
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// 从 JWT 中间件获取 user_id
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		userID: userID.(uint),
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    h.hub,
	}

	h.hub.register <- client

	go client.writePump()
	go client.readPump()
}
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/handler/ws.go
git commit -m "feat: implement WebSocket Hub for real-time task notifications"
```

---

### Task 7: AI Task DAO + Service

**Files:**
- Create: `server/internal/dao/ai_task.go`
- Create: `server/internal/service/ai.go`

- [ ] **Step 1: 创建 AI Task DAO 文件**

```go
// server/internal/dao/ai_task.go
package dao

import (
	"context"

	"story-maker/server/internal/model"
	"gorm.io/gorm"
)

// AITaskDAO AI 任务数据访问层
type AITaskDAO struct {
	db *gorm.DB
}

// NewAITaskDAO 创建 AITaskDAO 实例
func NewAITaskDAO(db *gorm.DB) *AITaskDAO {
	return &AITaskDAO{db: db}
}

// CreateTask 创建任务
func (d *AITaskDAO) CreateTask(ctx context.Context, task *model.AITask) error {
	return d.db.WithContext(ctx).Create(task).Error
}

// UpdateTask 更新任务
func (d *AITaskDAO) UpdateTask(ctx context.Context, task *model.AITask) error {
	return d.db.WithContext(ctx).Save(task).Error
}

// GetTask 根据 ID 获取任务
func (d *AITaskDAO) GetTask(ctx context.Context, taskID uint) (*model.AITask, error) {
	var task model.AITask
	err := d.db.WithContext(ctx).First(&task, taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasksByUser 获取用户的任务列表
func (d *AITaskDAO) ListTasksByUser(ctx context.Context, userID uint, limit, offset int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64

	query := d.db.WithContext(ctx).Where("user_id = ?", userID)

	if err := query.Model(&model.AITask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}

// ListTasksByPortfolio 获取作品集的任务列表
func (d *AITaskDAO) ListTasksByPortfolio(ctx context.Context, portfolioID uint, limit, offset int) ([]*model.AITask, int64, error) {
	var tasks []*model.AITask
	var total int64

	query := d.db.WithContext(ctx).Where("portfolio_id = ?", portfolioID)

	if err := query.Model(&model.AITask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}

// DeleteTask 删除任务
func (d *AITaskDAO) DeleteTask(ctx context.Context, taskID uint) error {
	return d.db.WithContext(ctx).Delete(&model.AITask{}, taskID).Error
}
```

- [ ] **Step 2: 创建 AI Service 文件**

```go
// server/internal/service/ai.go
package service

import (
	"context"
	"errors"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// AIService AI 服务层
type AIService struct {
	taskDAO    *dao.AITaskDAO
	dispatcher *agent.Dispatcher
}

// NewAIService 创建 AIService 实例
func NewAIService(taskDAO *dao.AITaskDAO, dispatcher *agent.Dispatcher) *AIService {
	return &AIService{
		taskDAO:    taskDAO,
		dispatcher: dispatcher,
	}
}

// SubmitTextTask 提交文本生成任务
func (s *AIService) SubmitTextTask(ctx context.Context, userID, portfolioID uint, modelName, prompt string) (uint, error) {
	// 校验输入
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}
	if modelName == "" {
		modelName = model.ProviderKimi
	}

	// 创建任务
	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeTextGen,
		ModelName:   modelName,
		Prompt:      prompt,
	}

	// 分发任务
	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// SubmitImageTask 提交图像生成任务
func (s *AIService) SubmitImageTask(ctx context.Context, userID, portfolioID uint, modelName, prompt string) (uint, error) {
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}
	if modelName == "" {
		modelName = model.ProviderKimi
	}

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeImageGen,
		ModelName:   modelName,
		Prompt:      prompt,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// SubmitCharacterAdjustTask 提交角色调整任务
func (s *AIService) SubmitCharacterAdjustTask(ctx context.Context, userID, portfolioID uint, modelName, prompt string) (uint, error) {
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}
	if modelName == "" {
		modelName = model.ProviderKimi
	}

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeCharacterAdjust,
		ModelName:   modelName,
		Prompt:      prompt,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// GetTask 获取任务详情
func (s *AIService) GetTask(ctx context.Context, taskID, userID uint) (*model.AITask, error) {
	task, err := s.taskDAO.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 权限校验：只能查看自己的任务
	if task.UserID != userID {
		return nil, errors.New("permission denied")
	}

	return task, nil
}

// ListTasks 获取任务列表
func (s *AIService) ListTasks(ctx context.Context, userID uint, portfolioID *uint, page, pageSize int) ([]*model.AITask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	if portfolioID != nil {
		return s.taskDAO.ListTasksByPortfolio(ctx, *portfolioID, pageSize, offset)
	}

	return s.taskDAO.ListTasksByUser(ctx, userID, pageSize, offset)
}

// CancelTask 取消任务
func (s *AIService) CancelTask(ctx context.Context, taskID, userID uint) error {
	task, err := s.taskDAO.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// 权限校验
	if task.UserID != userID {
		return errors.New("permission denied")
	}

	return s.dispatcher.CancelTask(ctx, taskID)
}
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/dao/ai_task.go server/internal/service/ai.go
git commit -m "feat: implement AI task DAO and service with validation and permission checks"
```

---

### Task 8: AI Handler

**Files:**
- Create: `server/internal/handler/ai.go`

- [ ] **Step 1: 创建 AI Handler 文件**

```go
// server/internal/handler/ai.go
package handler

import (
	"net/http"
	"strconv"

	"story-maker/server/internal/service"
	"github.com/gin-gonic/gin"
)

// AIHandler AI 请求处理器
type AIHandler struct {
	aiService *service.AIService
}

// NewAIHandler 创建 AIHandler 实例
func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// GenerateTextRequest 文本生成请求
type GenerateTextRequest struct {
	PortfolioID uint   `json:"portfolio_id"`
	ModelName   string `json:"model_name"`
	Prompt      string `json:"prompt" binding:"required"`
}

// GenerateImageRequest 图像生成请求
type GenerateImageRequest struct {
	PortfolioID uint   `json:"portfolio_id"`
	ModelName   string `json:"model_name"`
	Prompt      string `json:"prompt" binding:"required"`
}

// AdjustCharacterRequest 角色调整请求
type AdjustCharacterRequest struct {
	PortfolioID uint   `json:"portfolio_id"`
	ModelName   string `json:"model_name"`
	Prompt      string `json:"prompt" binding:"required"`
}

// TaskResponse 任务响应
type TaskResponse struct {
	TaskID uint   `json:"task_id"`
	Status string `json:"status"`
}

// GenerateText 文本生成接口
// POST /api/v1/ai/text/generate
func (h *AIHandler) GenerateText(c *gin.Context) {
	var req GenerateTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitTextTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// GenerateImage 图像生成接口
// POST /api/v1/ai/image/generate
func (h *AIHandler) GenerateImage(c *gin.Context) {
	var req GenerateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitImageTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// AdjustCharacter 角色调整接口
// POST /api/v1/ai/character/adjust
func (h *AIHandler) AdjustCharacter(c *gin.Context) {
	var req AdjustCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitCharacterAdjustTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TaskResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

// GetTask 获取任务详情
// GET /api/v1/ai/tasks/:id
func (h *AIHandler) GetTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	userID := c.GetUint("user_id")

	task, err := h.aiService.GetTask(c.Request.Context(), uint(taskID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// ListTasks 获取任务列表
// GET /api/v1/ai/tasks
func (h *AIHandler) ListTasks(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var portfolioID *uint
	if pidStr := c.Query("portfolio_id"); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 32)
		if err == nil {
			pidUint := uint(pid)
			portfolioID = &pidUint
		}
	}

	tasks, total, err := h.aiService.ListTasks(c.Request.Context(), userID, portfolioID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// CancelTask 取消任务
// DELETE /api/v1/ai/tasks/:id
func (h *AIHandler) CancelTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	userID := c.GetUint("user_id")

	if err := h.aiService.CancelTask(c.Request.Context(), uint(taskID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/handler/ai.go
git commit -m "feat: implement AI handler with text/image/character endpoints"
```

---

### Task 9: API Key DAO + Service + Handler

**Files:**
- Create: `server/internal/dao/api_key.go`
- Create: `server/internal/service/api_key.go`
- Create: `server/internal/handler/api_key.go`

- [ ] **Step 1: 创建 API Key DAO 文件**

```go
// server/internal/dao/api_key.go
package dao

import (
	"context"

	"story-maker/server/internal/model"
	"gorm.io/gorm"
)

// APIKeyDAO API Key 数据访问层
type APIKeyDAO struct {
	db *gorm.DB
}

// NewAPIKeyDAO 创建 APIKeyDAO 实例
func NewAPIKeyDAO(db *gorm.DB) *APIKeyDAO {
	return &APIKeyDAO{db: db}
}

// CreateKey 创建 API Key
func (d *APIKeyDAO) CreateKey(ctx context.Context, key *model.APIKey) error {
	return d.db.WithContext(ctx).Create(key).Error
}

// GetKeys 获取用户的所有 API Key
func (d *APIKeyDAO) GetKeys(ctx context.Context, userID uint) ([]*model.APIKey, error) {
	var keys []*model.APIKey
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

// GetKeyByID 根据 ID 获取 API Key
func (d *APIKeyDAO) GetKeyByID(ctx context.Context, keyID uint) (*model.APIKey, error) {
	var key model.APIKey
	err := d.db.WithContext(ctx).First(&key, keyID).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// UpdateKey 更新 API Key
func (d *APIKeyDAO) UpdateKey(ctx context.Context, key *model.APIKey) error {
	return d.db.WithContext(ctx).Save(key).Error
}

// DeleteKey 删除 API Key
func (d *APIKeyDAO) DeleteKey(ctx context.Context, keyID uint) error {
	return d.db.WithContext(ctx).Delete(&model.APIKey{}, keyID).Error
}

// GetUserKey 获取用户指定 Provider 的 API Key（优先 is_default=true）
func (d *APIKeyDAO) GetUserKey(ctx context.Context, userID uint, provider string) (*model.APIKey, error) {
	var key model.APIKey
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Order("is_default DESC").
		First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}
```

- [ ] **Step 2: 创建 API Key Service 文件**

```go
// server/internal/service/api_key.go
package service

import (
	"context"
	"errors"

	"story-maker/server/config"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
	"story-maker/server/internal/util"
)

// APIKeyService API Key 服务层，负责加密存储和解密读取
type APIKeyService struct {
	keyDAO    *dao.APIKeyDAO
	encryptKey []byte // AES-256 加密密钥（32 字节）
}

// NewAPIKeyService 创建 APIKeyService 实例
func NewAPIKeyService(keyDAO *dao.APIKeyDAO) *APIKeyService {
	// 从配置获取加密密钥
	encryptKey := []byte(config.Cfg.GetString("encrypt_key"))
	if len(encryptKey) < 32 {
		// 补齐到 32 字节
		padded := make([]byte, 32)
		copy(padded, encryptKey)
		encryptKey = padded
	}

	return &APIKeyService{
		keyDAO:     keyDAO,
		encryptKey: encryptKey[:32],
	}
}

// APIKeyResponse API Key 响应（脱敏）
type APIKeyResponse struct {
	ID        uint   `json:"id"`
	Provider  string `json:"provider"`
	KeyMask   string `json:"key_mask"` // 脱敏后的 Key（仅显示前4后4位）
	IsDefault bool   `json:"is_default"`
}

// CreateKey 创建 API Key（加密存储）
func (s *APIKeyService) CreateKey(ctx context.Context, userID uint, provider, keyValue string) (*APIKeyResponse, error) {
	if provider == "" || keyValue == "" {
		return nil, errors.New("provider and key_value are required")
	}

	// 校验 Provider 白名单
	if !isValidProvider(provider) {
		return nil, errors.New("invalid provider, supported: kimi, claude, copilot")
	}

	// 加密 Key
	encrypted, err := util.EncryptAES(keyValue, s.encryptKey)
	if err != nil {
		return nil, errors.New("failed to encrypt API key")
	}

	key := &model.APIKey{
		UserID:    userID,
		Provider:  provider,
		KeyValue:  encrypted,
		IsDefault: true,
	}

	if err := s.keyDAO.CreateKey(ctx, key); err != nil {
		return nil, err
	}

	return &APIKeyResponse{
		ID:        key.ID,
		Provider:  key.Provider,
		KeyMask:   maskKey(keyValue),
		IsDefault: key.IsDefault,
	}, nil
}

// GetKeys 获取用户的 API Key 列表（脱敏）
func (s *APIKeyService) GetKeys(ctx context.Context, userID uint) ([]*APIKeyResponse, error) {
	keys, err := s.keyDAO.GetKeys(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		// 解密后脱敏
		decrypted, err := util.DecryptAES(key.KeyValue, s.encryptKey)
		mask := "****"
		if err == nil {
			mask = maskKey(decrypted)
		}

		result = append(result, &APIKeyResponse{
			ID:        key.ID,
			Provider:  key.Provider,
			KeyMask:   mask,
			IsDefault: key.IsDefault,
		})
	}

	return result, nil
}

// UpdateKey 更新 API Key
func (s *APIKeyService) UpdateKey(ctx context.Context, keyID, userID uint, keyValue string, isDefault *bool) error {
	key, err := s.keyDAO.GetKeyByID(ctx, keyID)
	if err != nil {
		return err
	}

	// 权限校验
	if key.UserID != userID {
		return errors.New("permission denied")
	}

	// 更新 Key 值
	if keyValue != "" {
		encrypted, err := util.EncryptAES(keyValue, s.encryptKey)
		if err != nil {
			return errors.New("failed to encrypt API key")
		}
		key.KeyValue = encrypted
	}

	// 更新默认标记
	if isDefault != nil {
		key.IsDefault = *isDefault
	}

	return s.keyDAO.UpdateKey(ctx, key)
}

// DeleteKey 删除 API Key
func (s *APIKeyService) DeleteKey(ctx context.Context, keyID, userID uint) error {
	key, err := s.keyDAO.GetKeyByID(ctx, keyID)
	if err != nil {
		return err
	}

	if key.UserID != userID {
		return errors.New("permission denied")
	}

	return s.keyDAO.DeleteKey(ctx, keyID)
}

// GetUserKey 获取用户指定 Provider 的解密后 Key（供 Dispatcher 调用）
func (s *APIKeyService) GetUserKey(ctx context.Context, userID uint, provider string) (string, error) {
	key, err := s.keyDAO.GetUserKey(ctx, userID, provider)
	if err != nil {
		return "", err
	}

	decrypted, err := util.DecryptAES(key.KeyValue, s.encryptKey)
	if err != nil {
		return "", errors.New("failed to decrypt API key")
	}

	return decrypted, nil
}

// GetDefaultKey 获取平台默认 Key（从配置文件读取）
func (s *APIKeyService) GetDefaultKey(ctx context.Context, provider string) (string, error) {
	key := config.Cfg.GetString("ai." + provider + ".api_key")
	if key == "" {
		return "", errors.New("no default API key configured for " + provider)
	}
	return key, nil
}

// isValidProvider 校验 Provider 白名单
func isValidProvider(provider string) bool {
	switch provider {
	case model.ProviderKimi, model.ProviderClaude, model.ProviderCopilot:
		return true
	default:
		return false
	}
}

// maskKey 脱敏 Key：显示前4后4位，中间用 **** 替代
func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
```

- [ ] **Step 3: 创建 API Key Handler 文件**

```go
// server/internal/handler/api_key.go
package handler

import (
	"net/http"
	"strconv"

	"story-maker/server/internal/service"
	"github.com/gin-gonic/gin"
)

// APIKeyHandler API Key 请求处理器
type APIKeyHandler struct {
	keyService *service.APIKeyService
}

// NewAPIKeyHandler 创建 APIKeyHandler 实例
func NewAPIKeyHandler(keyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{keyService: keyService}
}

// CreateKeyRequest 创建 API Key 请求
type CreateKeyRequest struct {
	Provider string `json:"provider" binding:"required"`
	KeyValue string `json:"key_value" binding:"required"`
}

// UpdateKeyRequest 更新 API Key 请求
type UpdateKeyRequest struct {
	KeyValue  string `json:"key_value"`
	IsDefault *bool  `json:"is_default"`
}

// ListKeys 获取用户的 API Key 列表
// GET /api/v1/apikeys
func (h *APIKeyHandler) ListKeys(c *gin.Context) {
	userID := c.GetUint("user_id")

	keys, err := h.keyService.GetKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// CreateKey 创建 API Key
// POST /api/v1/apikeys
func (h *APIKeyHandler) CreateKey(c *gin.Context) {
	var req CreateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	key, err := h.keyService.CreateKey(c.Request.Context(), userID, req.Provider, req.KeyValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, key)
}

// UpdateKey 更新 API Key
// PUT /api/v1/apikeys/:id
func (h *APIKeyHandler) UpdateKey(c *gin.Context) {
	keyIDStr := c.Param("id")
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	var req UpdateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	if err := h.keyService.UpdateKey(c.Request.Context(), uint(keyID), userID, req.KeyValue, req.IsDefault); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key updated"})
}

// DeleteKey 删除 API Key
// DELETE /api/v1/apikeys/:id
func (h *APIKeyHandler) DeleteKey(c *gin.Context) {
	keyIDStr := c.Param("id")
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	userID := c.GetUint("user_id")

	if err := h.keyService.DeleteKey(c.Request.Context(), uint(keyID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/dao/api_key.go server/internal/service/api_key.go server/internal/handler/api_key.go
git commit -m "feat: implement API Key CRUD with AES encryption and masked responses"
```

---

### Task 10: 路由注册与集成

**Files:**
- Update: `server/cmd/main.go`
- Create: `server/config/config.yaml`

- [ ] **Step 1: 创建配置文件模板**

```yaml
# server/config/config.yaml
server:
  port: 8080
  mode: debug # debug, release

database:
  dsn: "root:password@tcp(localhost:3306)/ai_curton?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

jwt:
  secret: "your-jwt-secret-key-change-in-production"
  expire_hours: 48

# AES-256 加密密钥（32 字节）
encrypt_key: "your-32-byte-encryption-key-here"

# AI Provider 默认 API Key
ai:
  kimi:
    api_key: ""
  claude:
    api_key: ""
  copilot:
    api_key: ""

storage:
  type: local # local, oss
  local:
    base_path: "./uploads"
```

- [ ] **Step 2: 更新 main.go 集成所有模块**

```go
// server/cmd/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"story-maker/server/config"
	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/handler"
	"story-maker/server/internal/middleware"
	"story-maker/server/internal/model"
	"story-maker/server/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	if err := config.LoadConfig("./config/config.yaml"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化数据库
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	// 3. 初始化 Redis
	rdb := initRedis()

	// 4. 初始化 WebSocket Hub
	wsHub := handler.NewHub()
	go wsHub.Run()

	// 5. 初始化 DAO 层
	aiTaskDAO := dao.NewAITaskDAO(db)
	apiKeyDAO := dao.NewAPIKeyDAO(db)

	// 6. 初始化 Service 层
	apiKeyService := service.NewAPIKeyService(apiKeyDAO)

	// 7. 初始化 AI Dispatcher
	dispatcher := agent.NewDispatcher(apiKeyService, aiTaskDAO, wsHub)

	// 注册 Kimi Provider
	kimiProvider := agent.NewKimiProvider("")
	dispatcher.RegisterProvider(model.ProviderKimi, kimiProvider)

	aiService := service.NewAIService(aiTaskDAO, dispatcher)

	// 8. 初始化 Handler 层
	aiHandler := handler.NewAIHandler(aiService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	wsHandler := handler.NewWSHandler(wsHub)

	// 9. 初始化 Gin 路由
	r := setupRouter(aiHandler, apiKeyHandler, wsHandler, rdb)

	// 10. 启动服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Cfg.GetInt("server.port")),
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 11. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// initDB 初始化数据库连接并自动迁移
func initDB() (*gorm.DB, error) {
	dsn := config.Cfg.GetString("database.dsn")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&model.AITask{},
		&model.APIKey{},
	); err != nil {
		return nil, err
	}

	log.Println("Database connected and migrated")
	return db, nil
}

// initRedis 初始化 Redis 连接
func initRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Cfg.GetString("redis.addr"),
		Password: config.Cfg.GetString("redis.password"),
		DB:       config.Cfg.GetInt("redis.db"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis connected")
	return rdb
}

// setupRouter 配置路由
func setupRouter(
	aiHandler *handler.AIHandler,
	apiKeyHandler *handler.APIKeyHandler,
	wsHandler *handler.WSHandler,
	rdb *redis.Client,
) *gin.Engine {
	mode := config.Cfg.GetString("server.mode")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// WebSocket 路由（需要 JWT 认证）
	jwtSecret := config.Cfg.GetString("jwt.secret")
	authMiddleware := middleware.JWTAuth(jwtSecret)

	r.GET("/ws", authMiddleware, wsHandler.HandleWebSocket)

	// API 路由组
	v1 := r.Group("/api/v1")
	{
		// AI 能力接口（需要认证）
		ai := v1.Group("/ai", authMiddleware)
		{
			ai.POST("/text/generate", aiHandler.GenerateText)
			ai.POST("/image/generate", aiHandler.GenerateImage)
			ai.POST("/character/adjust", aiHandler.AdjustCharacter)
			ai.GET("/tasks", aiHandler.ListTasks)
			ai.GET("/tasks/:id", aiHandler.GetTask)
			ai.DELETE("/tasks/:id", aiHandler.CancelTask)
		}

		// API Key 管理接口（需要认证）
		apikeys := v1.Group("/apikeys", authMiddleware)
		{
			apikeys.GET("", apiKeyHandler.ListKeys)
			apikeys.POST("", apiKeyHandler.CreateKey)
			apikeys.PUT("/:id", apiKeyHandler.UpdateKey)
			apikeys.DELETE("/:id", apiKeyHandler.DeleteKey)
		}
	}

	return r
}
```

- [ ] **Step 3: 创建配置加载工具（如尚未存在）**

```go
// server/config/config.go
package config

import (
	"github.com/spf13/viper"
)

var Cfg *viper.Viper

// LoadConfig 加载配置文件
func LoadConfig(path string) error {
	Cfg = viper.New()
	Cfg.SetConfigFile(path)
	Cfg.SetConfigType("yaml")

	if err := Cfg.ReadInConfig(); err != nil {
		return err
	}

	return nil
}
```

- [ ] **Step 4: 创建 JWT 中间件（如尚未存在）**

```go
// server/internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth JWT 认证中间件
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in token"})
			c.Abort()
			return
		}

		c.Set("user_id", uint(userID))
		c.Next()
	}
}
```

- [ ] **Step 5: 创建 util 目录（如尚未存在）**

```bash
mkdir -p /Users/sangchenglong/go/src/story-maker/server/internal/util
```

- [ ] **Step 6: 验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go mod tidy
go build ./cmd/...
```

- [ ] **Step 7: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/cmd/main.go server/config/config.go server/config/config.yaml server/internal/middleware/auth.go
git commit -m "feat: integrate AI Agent layer with routing, WebSocket, and configuration"
```

---

### Task 11: 数据库迁移与测试

**Files:**
- Create: `server/scripts/migrate.sql`

- [ ] **Step 1: 创建数据库迁移脚本**

```sql
-- server/scripts/migrate.sql
-- AI 任务表
CREATE TABLE IF NOT EXISTS ai_tasks (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    portfolio_id BIGINT UNSIGNED NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    model_name VARCHAR(50) NOT NULL,
    prompt TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    result JSON,
    error_msg TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_portfolio_id (portfolio_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API Key 表
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    provider VARCHAR(50) NOT NULL,
    key_value VARCHAR(500) NOT NULL,
    is_default BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_provider (provider)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

- [ ] **Step 2: 创建测试脚本**

```bash
# server/scripts/test_api.sh
#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"
TOKEN="your-jwt-token-here"

echo "=== Testing AI Text Generation ==="
curl -X POST "$BASE_URL/ai/text/generate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "portfolio_id": 1,
    "model_name": "kimi",
    "prompt": "写一个科幻漫画的开场白"
  }'

echo -e "\n\n=== Testing AI Image Generation ==="
curl -X POST "$BASE_URL/ai/image/generate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "portfolio_id": 1,
    "model_name": "kimi",
    "prompt": "一个未来城市的夜景，赛博朋克风格"
  }'

echo -e "\n\n=== Testing Task List ==="
curl -X GET "$BASE_URL/ai/tasks?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"

echo -e "\n\n=== Testing API Key Creation ==="
curl -X POST "$BASE_URL/apikeys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "kimi",
    "key_value": "sk-test-key-123456"
  }'

echo -e "\n\n=== Testing API Key List ==="
curl -X GET "$BASE_URL/apikeys" \
  -H "Authorization: Bearer $TOKEN"
```

- [ ] **Step 3: 添加执行权限**

```bash
chmod +x /Users/sangchenglong/go/src/story-maker/server/scripts/test_api.sh
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/scripts/
git commit -m "feat: add database migration and API test scripts"
```

---

### Task 12: 文档与总结

**Files:**
- Create: `server/docs/ai-agent-integration.md`

- [ ] **Step 1: 创建集成文档**

```markdown
# AI Agent 层集成文档

## 架构概览

AI Agent 层采用 Provider 适配器模式，统一多模型接口，通过 Dispatcher 实现任务路由和异步执行，WebSocket Hub 负责实时推送。

### 核心组件

1. **AIProvider 接口**：统一 AI 模型调用接口
2. **KimiProvider**：Kimi 模型适配器
3. **Dispatcher**：任务分发器，负责路由、异步执行、状态管理
4. **WebSocket Hub**：连接管理和消息推送
5. **AIService**：业务逻辑层，任务提交和查询
6. **APIKeyService**：API Key 加密存储和解密读取

### 数据流

```
用户请求 → AI Handler → AI Service → Dispatcher → Provider 适配层
                                                      ├── Kimi
                                                      ├── Claude（预留）
                                                      └── Copilot（预留）
                                                           ↓
                                            更新 AITask 状态 → WebSocket 推送
```

## API 接口

### 文本生成

```bash
POST /api/v1/ai/text/generate
Authorization: Bearer <token>
Content-Type: application/json

{
  "portfolio_id": 1,
  "model_name": "kimi",
  "prompt": "写一个科幻漫画的开场白"
}
```

### 图像生成

```bash
POST /api/v1/ai/image/generate
Authorization: Bearer <token>
Content-Type: application/json

{
  "portfolio_id": 1,
  "model_name": "kimi",
  "prompt": "一个未来城市的夜景，赛博朋克风格"
}
```

### 角色调整

```bash
POST /api/v1/ai/character/adjust
Authorization: Bearer <token>
Content-Type: application/json

{
  "portfolio_id": 1,
  "model_name": "kimi",
  "prompt": "调整角色发型为短发，服装为科技战甲"
}
```

### 任务查询

```bash
GET /api/v1/ai/tasks?page=1&page_size=20&portfolio_id=1
Authorization: Bearer <token>
```

### 任务详情

```bash
GET /api/v1/ai/tasks/:id
Authorization: Bearer <token>
```

### 取消任务

```bash
DELETE /api/v1/ai/tasks/:id
Authorization: Bearer <token>
```

## API Key 管理

### 创建 API Key

```bash
POST /api/v1/apikeys
Authorization: Bearer <token>
Content-Type: application/json

{
  "provider": "kimi",
  "key_value": "sk-your-api-key"
}
```

### 查询 API Key 列表

```bash
GET /api/v1/apikeys
Authorization: Bearer <token>
```

响应示例：

```json
{
  "keys": [
    {
      "id": 1,
      "provider": "kimi",
      "key_mask": "sk-1****5678",
      "is_default": true
    }
  ]
}
```

### 更新 API Key

```bash
PUT /api/v1/apikeys/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "key_value": "sk-new-api-key",
  "is_default": true
}
```

### 删除 API Key

```bash
DELETE /api/v1/apikeys/:id
Authorization: Bearer <token>
```

## WebSocket 连接

### 连接

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=<jwt-token>');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Task update:', message);
  // { type: "task_update", data: { task_id, status, result, error } }
};
```

### 消息格式

```json
{
  "type": "task_update",
  "data": {
    "task_id": 123,
    "status": "completed",
    "result": {
      "content": "生成的文本内容..."
    }
  }
}
```

## 配置说明

### config.yaml

```yaml
# AES-256 加密密钥（32 字节）
encrypt_key: "your-32-byte-encryption-key-here"

# AI Provider 默认 API Key
ai:
  kimi:
    api_key: "sk-your-kimi-api-key"
  claude:
    api_key: ""
  copilot:
    api_key: ""
```

## 安全注意事项

1. **API Key 加密**：所有用户 API Key 使用 AES-256-GCM 加密存储
2. **权限校验**：任务查询和取消操作需校验 user_id
3. **Provider 白名单**：仅允许 kimi、claude、copilot
4. **WebSocket 认证**：连接时需携带有效 JWT Token

## 扩展新模型

1. 实现 `AIProvider` 接口
2. 在 `main.go` 中注册 Provider
3. 更新 `Capabilities()` 方法声明支持的能力
4. 配置默认 API Key

示例：

```go
// 实现 Claude Provider
type ClaudeProvider struct {
    apiKey string
}

func (c *ClaudeProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
    // 调用 Claude API
}

// 注册
claudeProvider := agent.NewClaudeProvider("")
dispatcher.RegisterProvider(model.ProviderClaude, claudeProvider)
```

## 测试

运行测试脚本：

```bash
cd /Users/sangchenglong/go/src/story-maker/server
./scripts/test_api.sh
```

## 已知限制

1. 当前仅接入 Kimi 模型
2. 任务取消仅更新状态，不中断执行
3. WebSocket 不支持断线重连（需前端实现）
4. 图像生成结果暂未下载到本地存储

## 后续优化

1. 接入 Claude/Kiro、Copilot
2. 实现任务队列和并发控制
3. 添加任务重试机制
4. 图像结果自动下载到本地/OSS
5. WebSocket 心跳和断线重连
6. 任务执行超时控制
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/docs/
git commit -m "docs: add AI Agent integration documentation"
```

---

## 完成检查清单

- [ ] Task 1: AI 任务模型创建完成
- [ ] Task 2: API Key 模型 + 加密工具创建完成
- [ ] Task 3: AI Provider 接口定义完成
- [ ] Task 4: Kimi 适配器实现完成
- [ ] Task 5: Dispatcher 实现完成
- [ ] Task 6: WebSocket 推送实现完成
- [ ] Task 7: AI Task DAO + Service 实现完成
- [ ] Task 8: AI Handler 实现完成
- [ ] Task 9: API Key DAO + Service + Handler 实现完成
- [ ] Task 10: 路由注册与集成完成
- [ ] Task 11: 数据库迁移与测试脚本创建完成
- [ ] Task 12: 集成文档编写完成

## 验证步骤

1. 启动 MySQL 和 Redis
2. 配置 `config.yaml` 中的数据库连接和 Kimi API Key
3. 运行 `go run ./cmd/main.go`
4. 使用 Postman 或 curl 测试 API 接口
5. 使用 WebSocket 客户端测试实时推送
6. 检查数据库中的任务记录和状态更新

## 注意事项

1. 所有代码遵循 DRY、KISS、SOLID、YAGNI 原则
2. 单个文件不超过 500 行，如超出需拆分
3. 所有注释使用中文，面向用户的错误信息使用英文
4. API Key 必须加密存储，不可明文
5. WebSocket 连接需 JWT 认证
6. 任务执行失败需记录详细错误信息

