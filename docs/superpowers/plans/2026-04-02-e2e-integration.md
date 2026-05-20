# 端到端完整链路打通 - 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 打通从注册登录到 AI Mock 生成的完整链路，前后端联调可用

**Architecture:** 后端新增 Mock Provider + 修复编译问题 + 修复 auth 中间件硬编码 token；前端补全所有 placeholder 页面为可交互页面；前端 API 路径对齐后端实际路由

**Tech Stack:** Go 1.22/Gin/GORM/WebSocket (后端), Vue 3/TypeScript/Element Plus/Pinia (前端)

---

## 关键发现（影响计划的问题）

1. **auth.go 中间件硬编码了 token**：`middleware/auth.go:27` 将 `authHeader` 覆盖为一个硬编码的 JWT 字符串，必须修复
2. **前端 API 路径与后端路由不匹配**：前端 `portfolio.ts` 和 `character.ts` 使用 `/workspaces/:id/portfolios/...` 路径，但后端路由是 `/portfolios?workspace_id=xxx` 和 `/portfolios/:id/characters`
3. **后端 handler 响应格式不一致**：`ai.go` 直接用 `c.JSON` 而非统一的 `Success()/Error()` 函数
4. **config.yaml 结构不匹配**：`main.go` 加载 `./config/config.yaml`（即 `server/config/config.yaml`），但该文件的结构（如 `encrypt_key`、`ai.kimi.api_key`）与 `config.go` 中定义的结构体（`encrypt.key`、`kimi.api_key`）不一致。需要统一为 `config.go` 定义的格式
5. **新增 TaskResult 组件**：`web/src/components/ai/TaskResult.vue` — AI 任务结果展示组件

## 文件结构

### 新建文件
- `server/internal/agent/mock.go` — Mock AI Provider
- `server/internal/middleware/logger.go` — 请求日志中间件
- `server/internal/middleware/recovery.go` — panic 恢复中间件
- `web/src/store/portfolio.ts` — 作品集状态管理
- `web/src/store/character.ts` — 角色状态管理
- `web/src/views/NotFound.vue` — 404 页面
- `web/src/components/ai/TaskResult.vue` — AI 任务结果展示组件

### 修改文件
- `server/go.mod` — 更新 toolchain 版本
- `server/config.yaml` — 添加 ai.default_provider 配置
- `server/config/config.go` — 添加 AI 配置结构
- `server/internal/agent/dispatcher.go` — 支持 Mock Provider 动态 Key 设置
- `server/internal/router/router.go` — 注册 Mock Provider + 新中间件
- `server/internal/middleware/auth.go` — 移除硬编码 token
- `server/internal/handler/ai.go` — 使用统一响应格式
- `web/src/api/portfolio.ts` — 对齐后端路由
- `web/src/api/character.ts` — 对齐后端路由
- `web/src/api/ai.ts` — 对齐后端路由，完善接口
- `web/src/utils/websocket.ts` — 完善自动重连
- `web/src/store/ai.ts` — 完善 WebSocket 消息处理
- `web/src/router/index.ts` — 添加 404 路由
- `web/src/views/workspace/WorkspaceDetail.vue` — 完整实现
- `web/src/views/portfolio/PortfolioDetail.vue` — 完整实现
- `web/src/views/character/CharacterList.vue` — 完整实现
- `web/src/views/studio/AIStudio.vue` — 完整实现
- `web/src/views/settings/APIKeyManage.vue` — 完整实现

---

### Task 1: 修复 Go 版本 + 编译问题

**Files:**
- Modify: `server/go.mod:1-5`

- [ ] **Step 1: 更新 go.mod toolchain 版本**

将 `server/go.mod` 中的 toolchain 从 `go1.22.12` 改为 `go1.23.10`：

```go
module story-maker/server

go 1.22

toolchain go1.23.10
```

- [ ] **Step 2: 清理编译缓存并验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go clean -cache
go build ./cmd/main.go
```

Expected: 编译成功，无版本不匹配错误

- [ ] **Step 3: Commit**

```bash
git add server/go.mod
git commit -m "fix: 更新 toolchain 版本为 go1.23.10 修复编译错误"
```

---

### Task 2: 修复 auth 中间件硬编码 token

**Files:**
- Modify: `server/internal/middleware/auth.go:27`

- [ ] **Step 1: 移除硬编码 token**

`auth.go:27` 有一行 `authHeader = "Bearer eyJ..."` 将请求头覆盖为硬编码值，必须删除这行。修改后的代码：

```go
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization header 中提取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
```

即删除第 27 行 `authHeader ="Bearer eyJhbGci..."` 这一行。

- [ ] **Step 2: 验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go build ./cmd/main.go
```

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add server/internal/middleware/auth.go
git commit -m "fix: 移除 auth 中间件中硬编码的 JWT token"
```

---

### Task 3: 添加 Mock Provider

**Files:**
- Create: `server/internal/agent/mock.go`
- Modify: `server/config.yaml`
- Modify: `server/config/config.go`

- [ ] **Step 1: 添加 AI 配置到 config.go**

在 `config/config.go` 的 `AppConfig` 结构体中添加 AI 配置：

```go
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Encrypt  EncryptConfig  `mapstructure:"encrypt"`
	Upload   UploadConfig   `mapstructure:"upload"`
	Kimi     KimiConfig     `mapstructure:"kimi"`
	AI       AIConfig       `mapstructure:"ai"`
}

type AIConfig struct {
	DefaultProvider string `mapstructure:"default_provider"` // mock, kimi
}
```

- [ ] **Step 2: 修复 config/config.yaml 结构对齐 config.go**

`server/config/config.yaml` 的结构与 `config.go` 不匹配，需要重写为正确格式：

```yaml
# server/config/config.yaml
server:
  port: 8080
  mode: debug

database:
  dsn: "root:Super!123@tcp(127.0.0.1:3306)/ai_curton?charset=utf8mb4&parseTime=True&loc=Local"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

jwt:
  secret: "htujhju7u"
  access_token_ttl: 172800
  refresh_token_ttl: 604800

encrypt:
  key: "678ijhgyhn-pad-to-32-bytes!!!!"

upload:
  path: "./uploads"
  max_size: 20971520

kimi:
  api_key: ""
  base_url: "https://api.moonshot.cn/v1"

ai:
  default_provider: "mock"
```

- [ ] **Step 3: 添加 ai 配置到 config.yaml（根目录备份也更新）**

在 `server/config.yaml`（根目录的那份）末尾添加：

```yaml
ai:
  default_provider: "mock"
```

- [ ] **Step 4: 创建 mock.go**

创建 `server/internal/agent/mock.go`：

```go
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

// Capabilities 返回 Mock 支持的所有能力
func (m *MockProvider) Capabilities() []string {
	return []string{
		"text_gen",
		"text_polish",
		"storyboard",
		"image_gen",
		"image_edit",
		"character_adjust",
	}
}
```

- [ ] **Step 5: 验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go build ./cmd/main.go
```

Expected: 编译成功

- [ ] **Step 6: Commit**

```bash
git add server/internal/agent/mock.go server/config/config.go server/config/config.yaml server/config.yaml
git commit -m "feat: 添加 Mock AI Provider 用于开发测试"
```

---

### Task 4: 注册 Mock Provider 到 Dispatcher + 添加中间件

**Files:**
- Create: `server/internal/middleware/logger.go`
- Create: `server/internal/middleware/recovery.go`
- Modify: `server/internal/router/router.go:46-48`
- Modify: `server/internal/agent/dispatcher.go:152-155`

- [ ] **Step 1: 创建请求日志中间件**

创建 `server/internal/middleware/logger.go`：

```go
// server/internal/middleware/logger.go
package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 请求日志中间件，记录请求路径、耗时、状态码
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[%s] %s %d %v", method, path, status, latency)
	}
}
```

- [ ] **Step 2: 创建 Recovery 中间件**

创建 `server/internal/middleware/recovery.go`：

```go
// server/internal/middleware/recovery.go
package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v\n%s", err, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
```

- [ ] **Step 3: 修改 dispatcher.go 支持 Mock Provider 动态 Key**

在 `dispatcher.go` 的 `executeTask` 方法中，第 152-155 行的类型断言只处理了 KimiProvider，需要扩展为跳过 MockProvider 的 Key 设置：

```go
	// 动态设置 API Key（Mock Provider 不需要 Key）
	if _, ok := provider.(*MockProvider); !ok {
		if kp, ok := provider.(*KimiProvider); ok {
			kp.SetAPIKey(apiKey)
		}
	}
```

同时修改 `resolveKey` 方法，Mock Provider 不需要 Key：

在 `resolveKey` 方法开头添加：

```go
func (d *Dispatcher) resolveKey(ctx context.Context, userID uint, provider string) (string, error) {
	// Mock Provider 不需要 API Key
	if provider == "mock" {
		return "mock-key", nil
	}

	// 优先查找用户自己的 Key
```

- [ ] **Step 4: 修改 router.go 注册 Mock Provider + 新中间件**

修改 `router.go` 的 `Setup` 函数：

1. 将 `r := gin.Default()` 改为 `r := gin.New()`（避免重复注册默认中间件）
2. 添加自定义中间件
3. 根据配置注册 Mock 或 Kimi Provider

```go
func Setup() *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
```

在注册 Provider 的部分（约第 46-48 行），改为根据配置选择：

```go
	// 根据配置注册 AI Provider
	if config.Global.AI.DefaultProvider == "mock" {
		mockProvider := agent.NewMockProvider()
		dispatcher.RegisterProvider("mock", mockProvider)
		// 同时注册为 kimi，这样前端选 kimi 也能走 mock
		dispatcher.RegisterProvider("kimi", mockProvider)
	} else {
		kimiProvider := agent.NewKimiProvider("")
		dispatcher.RegisterProvider("kimi", kimiProvider)
	}
```

需要在 import 中添加 `"story-maker/server/config"`。

- [ ] **Step 5: 验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go build ./cmd/main.go
```

Expected: 编译成功

- [ ] **Step 6: Commit**

```bash
git add server/internal/middleware/logger.go server/internal/middleware/recovery.go server/internal/router/router.go server/internal/agent/dispatcher.go
git commit -m "feat: 注册 Mock Provider + 添加日志和 Recovery 中间件"
```

---

### Task 5: 统一后端 AI Handler 响应格式

**Files:**
- Modify: `server/internal/handler/ai.go`

- [ ] **Step 1: 将 ai.go 中所有 c.JSON 调用改为统一响应函数**

将 `handler/ai.go` 中的所有响应改为使用 `Success()`/`BadRequest()`/`InternalError()`：

```go
// GenerateText 文本生成接口
func (h *AIHandler) GenerateText(c *gin.Context) {
	var req GenerateTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitTextTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{TaskID: taskID, Status: "pending"})
}

// GenerateImage 图像生成接口
func (h *AIHandler) GenerateImage(c *gin.Context) {
	var req GenerateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitImageTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{TaskID: taskID, Status: "pending"})
}

// AdjustCharacter 角色调整接口
func (h *AIHandler) AdjustCharacter(c *gin.Context) {
	var req AdjustCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	taskID, err := h.aiService.SubmitCharacterAdjustTask(c.Request.Context(), userID, req.PortfolioID, req.ModelName, req.Prompt)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, TaskResponse{TaskID: taskID, Status: "pending"})
}

// GetTask 获取任务详情
func (h *AIHandler) GetTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "invalid task id")
		return
	}

	userID := c.GetUint("user_id")

	task, err := h.aiService.GetTask(c.Request.Context(), uint(taskID), userID)
	if err != nil {
		Error(c, 404, err.Error())
		return
	}

	Success(c, task)
}

// ListTasks 获取任务列表
func (h *AIHandler) ListTasks(c *gin.Context) {
	userID := c.GetUint("user_id")

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
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"tasks":     tasks,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CancelTask 取消任务
func (h *AIHandler) CancelTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "invalid task id")
		return
	}

	userID := c.GetUint("user_id")

	if err := h.aiService.CancelTask(c.Request.Context(), uint(taskID), userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "task cancelled", nil)
}
```

同时移除未使用的 `"net/http"` import（因为不再直接用 `http.StatusXxx`）。

- [ ] **Step 2: 验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go build ./cmd/main.go
```

- [ ] **Step 3: Commit**

```bash
git add server/internal/handler/ai.go
git commit -m "refactor: AI handler 使用统一响应格式"
```

---

### Task 6: 前端 API 路径对齐后端路由

**Files:**
- Modify: `web/src/api/portfolio.ts`
- Modify: `web/src/api/character.ts`
- Modify: `web/src/api/ai.ts`

- [ ] **Step 1: 修复 portfolio.ts API 路径**

后端路由是 `GET /api/v1/portfolios?workspace_id=xxx`，不是 `/workspaces/:id/portfolios`。修改：

```typescript
// web/src/api/portfolio.ts
import request from './request'

export interface Portfolio {
  id: number
  workspace_id: number
  name: string
  description: string
  cover_image: string
  status: string
  created_at: string
  updated_at: string
}

export interface CreatePortfolioPayload {
  workspace_id: number
  name: string
  description?: string
}

export const portfolioApi = {
  list: (workspaceId: number) =>
    request.get('/portfolios', { params: { workspace_id: workspaceId } }),
  get: (id: number) => request.get(`/portfolios/${id}`),
  create: (data: CreatePortfolioPayload) => request.post('/portfolios', data),
  update: (id: number, data: Partial<CreatePortfolioPayload>) =>
    request.put(`/portfolios/${id}`, data),
  delete: (id: number) => request.delete(`/portfolios/${id}`),
}
```

- [ ] **Step 2: 修复 character.ts API 路径**

后端路由是 `GET /api/v1/portfolios/:id/characters` 和 `GET /api/v1/characters/:id`。修改：

```typescript
// web/src/api/character.ts
import request from './request'

export interface Character {
  id: number
  portfolio_id: number
  name: string
  description: string
  reference_images: string
  attributes: string
  created_at: string
  updated_at: string
}

export interface CreateCharacterPayload {
  name: string
  description?: string
  attributes?: Record<string, string>
}

export const characterApi = {
  list: (portfolioId: number) =>
    request.get(`/portfolios/${portfolioId}/characters`),
  get: (id: number) => request.get(`/characters/${id}`),
  create: (portfolioId: number, data: CreateCharacterPayload) =>
    request.post(`/portfolios/${portfolioId}/characters`, data),
  update: (id: number, data: Partial<CreateCharacterPayload>) =>
    request.put(`/characters/${id}`, data),
  delete: (id: number) => request.delete(`/characters/${id}`),
  uploadReference: (id: number, file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request.post(`/characters/${id}/reference`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}
```

- [ ] **Step 3: 修复 ai.ts API 路径**

后端路由是 `POST /api/v1/ai/text/generate` 等。修改：

```typescript
// web/src/api/ai.ts
import request from './request'

export interface TextGenRequest {
  portfolio_id: number
  model_name: string
  prompt: string
}

export interface ImageGenRequest {
  portfolio_id: number
  model_name: string
  prompt: string
}

export interface CharacterAdjustRequest {
  portfolio_id: number
  model_name: string
  prompt: string
}

export interface TaskResponse {
  task_id: number
  status: string
}

export interface AITask {
  id: number
  user_id: number
  portfolio_id: number
  task_type: string
  model_name: string
  prompt: string
  status: string
  result: string
  error_msg: string
  created_at: string
  updated_at: string
}

export const aiApi = {
  generateText: (data: TextGenRequest) =>
    request.post('/ai/text/generate', data),
  generateImage: (data: ImageGenRequest) =>
    request.post('/ai/image/generate', data),
  adjustCharacter: (data: CharacterAdjustRequest) =>
    request.post('/ai/character/adjust', data),
  getTask: (taskId: number) => request.get(`/ai/tasks/${taskId}`),
  listTasks: (params?: { portfolio_id?: number; page?: number; page_size?: number }) =>
    request.get('/ai/tasks', { params }),
  cancelTask: (taskId: number) => request.delete(`/ai/tasks/${taskId}`),
}
```

- [ ] **Step 4: Commit**

```bash
git add web/src/api/portfolio.ts web/src/api/character.ts web/src/api/ai.ts
git commit -m "fix: 前端 API 路径对齐后端实际路由"
```

---

### Task 7: 完善 WebSocket + AI Store

**Files:**
- Modify: `web/src/utils/websocket.ts`
- Modify: `web/src/store/ai.ts`

- [ ] **Step 1: 完善 websocket.ts 自动重连和心跳**

```typescript
// web/src/utils/websocket.ts
import { useAIStore } from '@/store/ai'

let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 10
const RECONNECT_INTERVAL = 3000

export function connectWebSocket() {
  const token = localStorage.getItem('access_token')
  if (!token) {
    console.warn('No access token, skipping WebSocket')
    return
  }

  // 清理旧连接
  if (ws) {
    ws.close()
    ws = null
  }

  const wsUrl = `ws://localhost:8080/ws?token=${token}`
  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('WebSocket connected')
    reconnectAttempts = 0
    const aiStore = useAIStore()
    aiStore.setWsConnected(true)
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'task_update' && msg.data) {
        const aiStore = useAIStore()
        aiStore.handleTaskUpdate(msg.data)
      }
    } catch (e) {
      console.error('Failed to parse WS message:', e)
    }
  }

  ws.onerror = () => {
    const aiStore = useAIStore()
    aiStore.setWsConnected(false)
  }

  ws.onclose = () => {
    console.log('WebSocket disconnected')
    const aiStore = useAIStore()
    aiStore.setWsConnected(false)
    ws = null

    // 自动重连
    if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
      reconnectAttempts++
      reconnectTimer = setTimeout(connectWebSocket, RECONNECT_INTERVAL)
    }
  }
}

export function disconnectWebSocket() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  reconnectAttempts = MAX_RECONNECT_ATTEMPTS // 阻止重连
  if (ws) {
    ws.close()
    ws = null
  }
}
```

- [ ] **Step 2: 完善 ai.ts store**

```typescript
// web/src/store/ai.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { aiApi } from '@/api/ai'
import type { AITask } from '@/api/ai'

export type { AITask } from '@/api/ai'

export const useAIStore = defineStore('ai', () => {
  const tasks = ref<AITask[]>([])
  const wsConnected = ref(false)
  const loading = ref(false)

  const pendingTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'pending' || t.status === 'running')
  )

  const completedTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'completed')
  )

  // 加载任务列表
  async function fetchTasks(portfolioId?: number) {
    loading.value = true
    try {
      const data: any = await aiApi.listTasks({
        portfolio_id: portfolioId,
        page: 1,
        page_size: 50,
      })
      tasks.value = data.tasks || []
    } finally {
      loading.value = false
    }
  }

  // 提交文本生成任务
  async function submitTextTask(portfolioId: number, modelName: string, prompt: string) {
    const data: any = await aiApi.generateText({
      portfolio_id: portfolioId,
      model_name: modelName,
      prompt,
    })
    // 立即添加到本地列表
    tasks.value.unshift({
      id: data.task_id,
      user_id: 0,
      portfolio_id: portfolioId,
      task_type: 'text_gen',
      model_name: modelName,
      prompt,
      status: 'pending',
      result: '',
      error_msg: '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    })
    return data.task_id
  }

  // 提交图像生成任务
  async function submitImageTask(portfolioId: number, modelName: string, prompt: string) {
    const data: any = await aiApi.generateImage({
      portfolio_id: portfolioId,
      model_name: modelName,
      prompt,
    })
    tasks.value.unshift({
      id: data.task_id,
      user_id: 0,
      portfolio_id: portfolioId,
      task_type: 'image_gen',
      model_name: modelName,
      prompt,
      status: 'pending',
      result: '',
      error_msg: '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    })
    return data.task_id
  }

  // 提交角色调整任务
  async function submitCharacterAdjustTask(portfolioId: number, modelName: string, prompt: string) {
    const data: any = await aiApi.adjustCharacter({
      portfolio_id: portfolioId,
      model_name: modelName,
      prompt,
    })
    tasks.value.unshift({
      id: data.task_id,
      user_id: 0,
      portfolio_id: portfolioId,
      task_type: 'character_adjust',
      model_name: modelName,
      prompt,
      status: 'pending',
      result: '',
      error_msg: '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    })
    return data.task_id
  }

  // 处理 WebSocket 任务更新
  function handleTaskUpdate(data: { task_id: number; status: string; result?: any; error?: string }) {
    const idx = tasks.value.findIndex((t) => t.id === data.task_id)
    if (idx !== -1) {
      tasks.value[idx].status = data.status
      if (data.result) {
        tasks.value[idx].result = JSON.stringify(data.result)
      }
      if (data.error) {
        tasks.value[idx].error_msg = data.error
      }
      tasks.value[idx].updated_at = new Date().toISOString()
    }
  }

  function setWsConnected(connected: boolean) {
    wsConnected.value = connected
  }

  function clearCompleted() {
    tasks.value = tasks.value.filter(
      (t) => t.status === 'pending' || t.status === 'running'
    )
  }

  return {
    tasks,
    wsConnected,
    loading,
    pendingTasks,
    completedTasks,
    fetchTasks,
    submitTextTask,
    submitImageTask,
    submitCharacterAdjustTask,
    handleTaskUpdate,
    setWsConnected,
    clearCompleted,
  }
})
```

- [ ] **Step 3: Commit**

```bash
git add web/src/utils/websocket.ts web/src/store/ai.ts
git commit -m "feat: 完善 WebSocket 自动重连 + AI Store 任务管理"
```

---

### Task 8: 补充 Portfolio Store + Character Store

**Files:**
- Create: `web/src/store/portfolio.ts`
- Create: `web/src/store/character.ts`

- [ ] **Step 1: 创建 portfolio store**

```typescript
// web/src/store/portfolio.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { portfolioApi } from '@/api/portfolio'
import type { Portfolio } from '@/api/portfolio'

export const usePortfolioStore = defineStore('portfolio', () => {
  const portfolios = ref<Portfolio[]>([])
  const currentPortfolio = ref<Portfolio | null>(null)
  const loading = ref(false)

  async function fetchPortfolios(workspaceId: number) {
    loading.value = true
    try {
      const data: any = await portfolioApi.list(workspaceId)
      portfolios.value = Array.isArray(data) ? data : data.items || []
    } finally {
      loading.value = false
    }
  }

  async function fetchPortfolio(id: number) {
    loading.value = true
    try {
      const data: any = await portfolioApi.get(id)
      currentPortfolio.value = data
    } finally {
      loading.value = false
    }
  }

  return {
    portfolios,
    currentPortfolio,
    loading,
    fetchPortfolios,
    fetchPortfolio,
  }
})
```

- [ ] **Step 2: 创建 character store**

```typescript
// web/src/store/character.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { characterApi } from '@/api/character'
import type { Character } from '@/api/character'

export const useCharacterStore = defineStore('character', () => {
  const characters = ref<Character[]>([])
  const loading = ref(false)

  async function fetchCharacters(portfolioId: number) {
    loading.value = true
    try {
      const data: any = await characterApi.list(portfolioId)
      characters.value = Array.isArray(data) ? data : data.items || []
    } finally {
      loading.value = false
    }
  }

  return {
    characters,
    loading,
    fetchCharacters,
  }
})
```

- [ ] **Step 3: Commit**

```bash
git add web/src/store/portfolio.ts web/src/store/character.ts
git commit -m "feat: 添加 Portfolio 和 Character 状态管理"
```

---

### Task 9: 实现 WorkspaceDetail 页面

**Files:**
- Modify: `web/src/views/workspace/WorkspaceDetail.vue`

- [ ] **Step 1: 实现完整的 WorkspaceDetail 页面**

替换 placeholder 为完整实现，包含：工作空间信息展示、作品集列表、创建作品集弹窗、成员管理入口。

```vue
<!-- web/src/views/workspace/WorkspaceDetail.vue -->
<template>
  <div class="workspace-detail">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ workspace?.name || 'Loading...' }}</h1>
        <p class="page-desc">{{ workspace?.description || '' }}</p>
      </div>
      <div class="header-actions">
        <NeonButton @click="showMemberDialog = true">Members</NeonButton>
        <NeonButton type="primary" @click="showCreateDialog = true">+ New Portfolio</NeonButton>
      </div>
    </div>

    <div v-loading="loading" class="portfolio-grid">
      <GlowCard
        v-for="p in portfolios"
        :key="p.id"
        hoverable
        class="portfolio-card"
        @click="goToPortfolio(p.id)"
      >
        <div class="portfolio-card__header">
          <h3>{{ p.name }}</h3>
          <el-tag :type="p.status === 'published' ? 'success' : 'info'" size="small">
            {{ p.status || 'draft' }}
          </el-tag>
        </div>
        <p class="portfolio-card__desc">{{ p.description || 'No description' }}</p>
        <div class="portfolio-card__footer">
          <span>{{ formatDate(p.created_at) }}</span>
          <el-dropdown trigger="click" @command="handlePortfolioAction($event, p)">
            <span class="el-dropdown-link" @click.stop>...</span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit">Edit</el-dropdown-item>
                <el-dropdown-item command="delete" divided>Delete</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </GlowCard>
    </div>

    <!-- 创建作品集弹窗 -->
    <el-dialog v-model="showCreateDialog" title="Create Portfolio" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Portfolio name" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleCreate">Create</NeonButton>
      </template>
    </el-dialog>

    <!-- 成员管理弹窗 -->
    <el-dialog v-model="showMemberDialog" title="Members" width="600px">
      <MemberManage v-if="showMemberDialog" :workspace-id="Number(id)" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import { workspaceApi } from '@/api/workspace'
import { portfolioApi } from '@/api/portfolio'
import type { Portfolio } from '@/api/portfolio'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import MemberManage from './MemberManage.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()

const workspace = ref<any>(null)
const portfolios = ref<Portfolio[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showMemberDialog = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({ name: '', description: '' })
const rules: FormRules = {
  name: [{ required: true, message: 'Please enter name', trigger: 'blur' }],
}

onMounted(async () => {
  await Promise.all([fetchWorkspace(), fetchPortfolios()])
})

async function fetchWorkspace() {
  try {
    workspace.value = await workspaceApi.get(Number(props.id))
  } catch (e: any) {
    ElMessage.error('Failed to load workspace')
  }
}

async function fetchPortfolios() {
  loading.value = true
  try {
    const data: any = await portfolioApi.list(Number(props.id))
    portfolios.value = Array.isArray(data) ? data : data.items || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await portfolioApi.create({
        workspace_id: Number(props.id),
        name: form.name,
        description: form.description,
      })
      ElMessage.success('Portfolio created')
      showCreateDialog.value = false
      Object.assign(form, { name: '', description: '' })
      await fetchPortfolios()
    } finally {
      submitting.value = false
    }
  })
}

function goToPortfolio(pid: number) {
  router.push(`/workspace/${props.id}/portfolio/${pid}`)
}

async function handlePortfolioAction(action: string, p: Portfolio) {
  if (action === 'delete') {
    await ElMessageBox.confirm('Delete this portfolio?', 'Confirm')
    await portfolioApi.delete(p.id)
    ElMessage.success('Deleted')
    await fetchPortfolios()
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString()
}
</script>

<style scoped lang="scss">
.workspace-detail { width: 100%; max-width: 1200px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 32px; }
.page-title { font-size: 28px; font-weight: 700; color: var(--color-text-primary); }
.page-desc { font-size: 14px; color: var(--color-text-secondary); margin-top: 4px; }
.header-actions { display: flex; gap: 12px; }
.portfolio-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 24px; }
.portfolio-card {
  cursor: pointer;
  &__header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;
    h3 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); }
  }
  &__desc { font-size: 13px; color: var(--color-text-secondary); min-height: 36px; margin-bottom: 12px; }
  &__footer { display: flex; justify-content: space-between; align-items: center; padding-top: 12px; border-top: 1px solid var(--border-glow); font-size: 12px; color: var(--color-text-muted); }
}
.el-dropdown-link { cursor: pointer; padding: 4px 8px; }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/views/workspace/WorkspaceDetail.vue
git commit -m "feat: 实现 WorkspaceDetail 页面（作品集列表+成员管理）"
```

---

### Task 10: 实现 PortfolioDetail 页面

**Files:**
- Modify: `web/src/views/portfolio/PortfolioDetail.vue`

- [ ] **Step 1: 实现完整的 PortfolioDetail 页面**

包含：作品集信息、角色列表入口、AI 工坊入口、资源列表。

```vue
<!-- web/src/views/portfolio/PortfolioDetail.vue -->
<template>
  <div class="portfolio-detail">
    <div class="page-header">
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">Workspace</el-breadcrumb-item>
          <el-breadcrumb-item>{{ portfolio?.name || '...' }}</el-breadcrumb-item>
        </el-breadcrumb>
        <h1 class="page-title">{{ portfolio?.name || 'Loading...' }}</h1>
        <p class="page-desc">{{ portfolio?.description || '' }}</p>
      </div>
    </div>

    <div class="action-cards">
      <GlowCard hoverable class="action-card" @click="goToCharacters">
        <div class="action-card__icon">🎭</div>
        <h3>Characters</h3>
        <p>Manage characters for this portfolio</p>
      </GlowCard>
      <GlowCard hoverable class="action-card" @click="goToStudio">
        <div class="action-card__icon">🤖</div>
        <h3>AI Studio</h3>
        <p>Generate text and images with AI</p>
      </GlowCard>
    </div>

    <div class="section">
      <h2 class="section-title">Recent AI Tasks</h2>
      <div v-loading="tasksLoading" class="task-list">
        <div v-if="tasks.length === 0" class="empty-state">No tasks yet. Go to AI Studio to create one.</div>
        <GlowCard v-for="task in tasks" :key="task.id" class="task-card">
          <div class="task-card__header">
            <el-tag :type="statusTagType(task.status)" size="small">{{ task.status }}</el-tag>
            <span class="task-card__type">{{ task.task_type }}</span>
          </div>
          <p class="task-card__prompt">{{ task.prompt }}</p>
          <div v-if="task.status === 'completed' && task.result" class="task-card__result">
            <TaskResult :task="task" />
          </div>
          <div class="task-card__footer">{{ formatDate(task.created_at) }}</div>
        </GlowCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, defineAsyncComponent } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { portfolioApi } from '@/api/portfolio'
import { aiApi } from '@/api/ai'
import type { Portfolio } from '@/api/portfolio'
import type { AITask } from '@/api/ai'
import GlowCard from '@/components/common/GlowCard.vue'

const TaskResult = defineAsyncComponent(() => import('@/components/ai/TaskResult.vue'))

const props = defineProps<{ id: string; pid: string }>()
const router = useRouter()

const portfolio = ref<Portfolio | null>(null)
const tasks = ref<AITask[]>([])
const tasksLoading = ref(false)

onMounted(async () => {
  try {
    portfolio.value = await portfolioApi.get(Number(props.pid)) as any
  } catch { ElMessage.error('Failed to load portfolio') }

  tasksLoading.value = true
  try {
    const data: any = await aiApi.listTasks({ portfolio_id: Number(props.pid) })
    tasks.value = data.tasks || []
  } finally { tasksLoading.value = false }
})

function goToCharacters() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/characters`)
}
function goToStudio() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/studio`)
}
function statusTagType(s: string) {
  const map: Record<string, string> = { completed: 'success', running: 'warning', failed: 'danger', pending: 'info' }
  return map[s] || 'info'
}
function formatDate(d: string) { return new Date(d).toLocaleString() }
</script>

<style scoped lang="scss">
.portfolio-detail { width: 100%; max-width: 1200px; margin: 0 auto; }
.page-header { margin-bottom: 32px; }
.page-title { font-size: 28px; font-weight: 700; color: var(--color-text-primary); margin-top: 12px; }
.page-desc { font-size: 14px; color: var(--color-text-secondary); margin-top: 4px; }
.action-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 20px; margin-bottom: 40px; }
.action-card { cursor: pointer; text-align: center; padding: 24px;
  &__icon { font-size: 36px; margin-bottom: 12px; }
  h3 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
  p { font-size: 13px; color: var(--color-text-secondary); }
}
.section { margin-bottom: 32px; }
.section-title { font-size: 20px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 16px; }
.task-list { display: flex; flex-direction: column; gap: 12px; }
.empty-state { text-align: center; padding: 40px; color: var(--color-text-muted); font-size: 14px; }
.task-card {
  &__header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
  &__type { font-size: 12px; color: var(--color-text-muted); }
  &__prompt { font-size: 14px; color: var(--color-text-secondary); margin-bottom: 8px; white-space: pre-wrap; max-height: 80px; overflow: hidden; }
  &__result { margin-bottom: 8px; }
  &__footer { font-size: 12px; color: var(--color-text-muted); }
}
</style>
```

- [ ] **Step 2: 创建 TaskResult 组件**

创建 `web/src/components/ai/TaskResult.vue`：

```vue
<!-- web/src/components/ai/TaskResult.vue -->
<template>
  <div class="task-result">
    <template v-if="parsedResult">
      <!-- 文本结果 -->
      <div v-if="parsedResult.content" class="result-text">
        <pre>{{ parsedResult.content }}</pre>
      </div>
      <!-- 图像结果 -->
      <div v-if="parsedResult.image_url" class="result-image">
        <img :src="parsedResult.image_url" alt="AI Generated" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AITask } from '@/api/ai'

const props = defineProps<{ task: AITask }>()

const parsedResult = computed(() => {
  if (!props.task.result) return null
  try { return JSON.parse(props.task.result) } catch { return null }
})
</script>

<style scoped lang="scss">
.result-text pre {
  background: var(--color-bg-deep);
  padding: 12px;
  border-radius: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
  max-height: 200px;
  overflow-y: auto;
}
.result-image img {
  max-width: 300px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
}
</style>
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/portfolio/PortfolioDetail.vue web/src/components/ai/TaskResult.vue
git commit -m "feat: 实现 PortfolioDetail 页面 + TaskResult 组件"
```

---

### Task 11: 实现 CharacterList 页面

**Files:**
- Modify: `web/src/views/character/CharacterList.vue`

- [ ] **Step 1: 实现完整的 CharacterList 页面**

```vue
<!-- web/src/views/character/CharacterList.vue -->
<template>
  <div class="character-list">
    <div class="page-header">
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">Workspace</el-breadcrumb-item>
          <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">Portfolio</el-breadcrumb-item>
          <el-breadcrumb-item>Characters</el-breadcrumb-item>
        </el-breadcrumb>
        <h1 class="page-title">Characters</h1>
      </div>
      <NeonButton type="primary" @click="showCreateDialog = true">+ New Character</NeonButton>
    </div>

    <div v-loading="loading" class="character-grid">
      <div v-if="characters.length === 0 && !loading" class="empty-state">
        No characters yet. Create one to get started.
      </div>
      <GlowCard v-for="ch in characters" :key="ch.id" hoverable class="character-card">
        <div class="character-card__avatar">🎭</div>
        <h3 class="character-card__name">{{ ch.name }}</h3>
        <p class="character-card__desc">{{ ch.description || 'No description' }}</p>
        <div class="character-card__actions">
          <el-button size="small" @click="editCharacter(ch)">Edit</el-button>
          <el-button size="small" type="danger" @click="deleteCharacter(ch.id)">Delete</el-button>
        </div>
      </GlowCard>
    </div>

    <!-- 创建/编辑角色弹窗 -->
    <el-dialog v-model="showCreateDialog" :title="editingId ? 'Edit Character' : 'Create Character'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Character name" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeDialog">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ editingId ? 'Save' : 'Create' }}
        </NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import { characterApi } from '@/api/character'
import type { Character } from '@/api/character'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const props = defineProps<{ id: string; pid: string }>()

const characters = ref<Character[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const form = reactive({ name: '', description: '' })
const rules: FormRules = {
  name: [{ required: true, message: 'Please enter name', trigger: 'blur' }],
}

onMounted(() => fetchCharacters())

async function fetchCharacters() {
  loading.value = true
  try {
    const data: any = await characterApi.list(Number(props.pid))
    characters.value = Array.isArray(data) ? data : data.items || []
  } finally { loading.value = false }
}

function editCharacter(ch: Character) {
  editingId.value = ch.id
  form.name = ch.name
  form.description = ch.description || ''
  showCreateDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingId.value = null
  Object.assign(form, { name: '', description: '' })
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await characterApi.update(editingId.value, form)
        ElMessage.success('Character updated')
      } else {
        await characterApi.create(Number(props.pid), form)
        ElMessage.success('Character created')
      }
      closeDialog()
      await fetchCharacters()
    } finally { submitting.value = false }
  })
}

async function deleteCharacter(id: number) {
  await ElMessageBox.confirm('Delete this character?', 'Confirm')
  await characterApi.delete(id)
  ElMessage.success('Deleted')
  await fetchCharacters()
}
</script>

<style scoped lang="scss">
.character-list { width: 100%; max-width: 1200px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 32px; }
.page-title { font-size: 28px; font-weight: 700; color: var(--color-text-primary); margin-top: 12px; }
.character-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 20px; }
.empty-state { grid-column: 1 / -1; text-align: center; padding: 60px; color: var(--color-text-muted); }
.character-card {
  text-align: center; padding: 24px;
  &__avatar { font-size: 48px; margin-bottom: 12px; }
  &__name { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
  &__desc { font-size: 13px; color: var(--color-text-secondary); min-height: 36px; margin-bottom: 12px; }
  &__actions { display: flex; justify-content: center; gap: 8px; }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/views/character/CharacterList.vue
git commit -m "feat: 实现 CharacterList 页面"
```

---

### Task 12: 实现 AI Studio 页面

**Files:**
- Modify: `web/src/views/studio/AIStudio.vue`

- [ ] **Step 1: 实现完整的 AI Studio 页面**

核心页面：任务提交表单（文本/图像/角色调整）+ 任务列表 + 实时状态 + 结果展示。

```vue
<!-- web/src/views/studio/AIStudio.vue -->
<template>
  <div class="ai-studio">
    <div class="page-header">
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">Workspace</el-breadcrumb-item>
          <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">Portfolio</el-breadcrumb-item>
          <el-breadcrumb-item>AI Studio</el-breadcrumb-item>
        </el-breadcrumb>
        <h1 class="page-title">AI Studio</h1>
      </div>
      <div class="ws-status">
        <span :class="['ws-dot', { connected: aiStore.wsConnected }]"></span>
        {{ aiStore.wsConnected ? 'Connected' : 'Disconnected' }}
      </div>
    </div>

    <div class="studio-layout">
      <!-- 左侧：任务提交 -->
      <GlowCard class="submit-panel">
        <h3 class="panel-title">Create Task</h3>

        <el-form :model="form" label-position="top">
          <el-form-item label="Task Type">
            <el-radio-group v-model="form.taskType">
              <el-radio-button value="text_gen">Text</el-radio-button>
              <el-radio-button value="image_gen">Image</el-radio-button>
              <el-radio-button value="character_adjust">Character</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="Model">
            <el-select v-model="form.modelName" style="width: 100%">
              <el-option label="Mock (Dev)" value="mock" />
              <el-option label="Kimi" value="kimi" />
            </el-select>
          </el-form-item>

          <el-form-item label="Prompt">
            <el-input
              v-model="form.prompt"
              type="textarea"
              :rows="6"
              placeholder="Describe what you want to generate..."
            />
          </el-form-item>

          <NeonButton
            type="primary"
            :loading="submitting"
            :disabled="!form.prompt.trim()"
            style="width: 100%"
            @click="handleSubmit"
          >
            Generate
          </NeonButton>
        </el-form>
      </GlowCard>

      <!-- 右侧：任务列表 -->
      <div class="task-panel">
        <div class="panel-header">
          <h3 class="panel-title">Tasks</h3>
          <el-button v-if="aiStore.completedTasks.length > 0" size="small" @click="aiStore.clearCompleted">
            Clear Completed
          </el-button>
        </div>

        <div v-loading="aiStore.loading" class="task-list">
          <div v-if="aiStore.tasks.length === 0" class="empty-state">
            No tasks yet. Submit a prompt to get started.
          </div>

          <GlowCard v-for="task in aiStore.tasks" :key="task.id" class="task-item">
            <div class="task-item__header">
              <el-tag :type="statusTagType(task.status)" size="small">{{ task.status }}</el-tag>
              <span class="task-item__type">{{ task.task_type }}</span>
              <span class="task-item__time">{{ formatTime(task.created_at) }}</span>
            </div>
            <p class="task-item__prompt">{{ task.prompt }}</p>

            <!-- 运行中动画 -->
            <div v-if="task.status === 'running'" class="task-item__loading">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>Generating...</span>
            </div>

            <!-- 结果展示 -->
            <div v-if="task.status === 'completed' && task.result" class="task-item__result">
              <TaskResult :task="task" />
            </div>

            <!-- 错误信息 -->
            <div v-if="task.status === 'failed'" class="task-item__error">
              {{ task.error_msg || 'Task failed' }}
            </div>
          </GlowCard>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { useAIStore } from '@/store/ai'
import { connectWebSocket, disconnectWebSocket } from '@/utils/websocket'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import TaskResult from '@/components/ai/TaskResult.vue'

const props = defineProps<{ id: string; pid: string }>()
const aiStore = useAIStore()
const submitting = ref(false)

const form = reactive({
  taskType: 'text_gen',
  modelName: 'mock',
  prompt: '',
})

onMounted(() => {
  connectWebSocket()
  aiStore.fetchTasks(Number(props.pid))
})

onUnmounted(() => {
  disconnectWebSocket()
})

async function handleSubmit() {
  if (!form.prompt.trim()) return
  submitting.value = true
  try {
    const portfolioId = Number(props.pid)
    switch (form.taskType) {
      case 'text_gen':
        await aiStore.submitTextTask(portfolioId, form.modelName, form.prompt)
        break
      case 'image_gen':
        await aiStore.submitImageTask(portfolioId, form.modelName, form.prompt)
        break
      case 'character_adjust':
        await aiStore.submitCharacterAdjustTask(portfolioId, form.modelName, form.prompt)
        break
    }
    ElMessage.success('Task submitted')
    form.prompt = ''
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to submit task')
  } finally {
    submitting.value = false
  }
}

function statusTagType(s: string) {
  const map: Record<string, string> = { completed: 'success', running: 'warning', failed: 'danger', pending: 'info' }
  return map[s] || 'info'
}

function formatTime(d: string) {
  return new Date(d).toLocaleTimeString()
}
</script>

<style scoped lang="scss">
.ai-studio { width: 100%; max-width: 1400px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-title { font-size: 28px; font-weight: 700; color: var(--color-text-primary); margin-top: 12px; }
.ws-status { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--color-text-muted); }
.ws-dot { width: 8px; height: 8px; border-radius: 50%; background: #ef4444;
  &.connected { background: #22c55e; }
}
.studio-layout { display: grid; grid-template-columns: 380px 1fr; gap: 24px; }
.submit-panel { padding: 24px; }
.panel-title { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 16px; }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.task-list { display: flex; flex-direction: column; gap: 12px; }
.empty-state { text-align: center; padding: 40px; color: var(--color-text-muted); font-size: 14px; }
.task-item {
  &__header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
  &__type { font-size: 12px; color: var(--color-text-muted); }
  &__time { font-size: 12px; color: var(--color-text-muted); margin-left: auto; }
  &__prompt { font-size: 14px; color: var(--color-text-secondary); margin-bottom: 8px; white-space: pre-wrap; max-height: 60px; overflow: hidden; }
  &__loading { display: flex; align-items: center; gap: 8px; color: var(--color-primary); font-size: 13px; padding: 8px 0; }
  &__result { margin-top: 8px; }
  &__error { color: #ef4444; font-size: 13px; padding: 8px; background: rgba(239, 68, 68, 0.1); border-radius: 6px; }
}

@media (max-width: 768px) {
  .studio-layout { grid-template-columns: 1fr; }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/views/studio/AIStudio.vue
git commit -m "feat: 实现 AI Studio 页面（任务提交+实时状态+结果展示）"
```

---

### Task 13: 实现 API Key 管理页面 + MemberManage 完善

**Files:**
- Modify: `web/src/views/settings/APIKeyManage.vue`
- Modify: `web/src/views/workspace/MemberManage.vue`

- [ ] **Step 1: 实现 APIKeyManage 页面**

```vue
<!-- web/src/views/settings/APIKeyManage.vue -->
<template>
  <div class="apikey-manage">
    <div class="page-header">
      <h1 class="page-title">API Key Management</h1>
      <NeonButton type="primary" @click="showAddDialog = true">+ Add Key</NeonButton>
    </div>

    <div v-loading="loading" class="key-list">
      <div v-if="keys.length === 0 && !loading" class="empty-state">
        No API keys configured. Add one to use real AI models.
      </div>
      <GlowCard v-for="key in keys" :key="key.id" class="key-card">
        <div class="key-card__header">
          <el-tag size="small">{{ key.provider }}</el-tag>
          <span class="key-card__mask">{{ key.key_mask }}</span>
        </div>
        <div class="key-card__actions">
          <el-button size="small" type="danger" @click="handleDelete(key.id)">Delete</el-button>
        </div>
      </GlowCard>
    </div>

    <el-dialog v-model="showAddDialog" title="Add API Key" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Provider" prop="provider">
          <el-select v-model="form.provider" style="width: 100%">
            <el-option label="Kimi" value="kimi" />
            <el-option label="Claude" value="claude" />
            <el-option label="Copilot" value="copilot" />
          </el-select>
        </el-form-item>
        <el-form-item label="API Key" prop="key_value">
          <el-input v-model="form.key_value" type="password" show-password placeholder="Enter your API key" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleAdd">Add</NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import request from '@/api/request'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

interface APIKeyItem { id: number; provider: string; key_mask: string; is_default: boolean }

const keys = ref<APIKeyItem[]>([])
const loading = ref(false)
const showAddDialog = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({ provider: 'kimi', key_value: '' })
const rules: FormRules = {
  provider: [{ required: true, message: 'Select provider', trigger: 'change' }],
  key_value: [{ required: true, message: 'Enter API key', trigger: 'blur' }],
}

onMounted(() => fetchKeys())

async function fetchKeys() {
  loading.value = true
  try {
    const data: any = await request.get('/apikeys')
    keys.value = Array.isArray(data) ? data : []
  } finally { loading.value = false }
}

async function handleAdd() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await request.post('/apikeys', form)
      ElMessage.success('API Key added')
      showAddDialog.value = false
      form.key_value = ''
      await fetchKeys()
    } finally { submitting.value = false }
  })
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm('Delete this API key?', 'Confirm')
  await request.delete(`/apikeys/${id}`)
  ElMessage.success('Deleted')
  await fetchKeys()
}
</script>

<style scoped lang="scss">
.apikey-manage { width: 100%; max-width: 800px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.page-title { font-size: 28px; font-weight: 700; color: var(--color-text-primary); }
.key-list { display: flex; flex-direction: column; gap: 12px; }
.empty-state { text-align: center; padding: 40px; color: var(--color-text-muted); }
.key-card {
  &__header { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
  &__mask { font-family: monospace; font-size: 14px; color: var(--color-text-secondary); }
  &__actions { display: flex; justify-content: flex-end; }
}
</style>
```

- [ ] **Step 2: 完善 MemberManage 组件**

检查 `web/src/views/workspace/MemberManage.vue` 是否已有实现。如果是 placeholder，替换为：

```vue
<!-- web/src/views/workspace/MemberManage.vue -->
<template>
  <div class="member-manage">
    <div v-loading="loading" class="member-list">
      <div v-for="m in members" :key="m.id" class="member-item">
        <div class="member-info">
          <span class="member-name">{{ m.username || `User #${m.user_id}` }}</span>
          <el-tag :type="m.role === 'owner' ? 'warning' : 'info'" size="small">{{ m.role }}</el-tag>
        </div>
        <el-button
          v-if="m.role !== 'owner'"
          size="small"
          type="danger"
          @click="handleRemove(m.user_id)"
        >Remove</el-button>
      </div>
    </div>

    <el-divider />

    <div class="add-member">
      <el-input v-model="newUserId" placeholder="User ID" style="width: 200px" />
      <el-select v-model="newRole" style="width: 120px">
        <el-option label="Editor" value="editor" />
        <el-option label="Viewer" value="viewer" />
      </el-select>
      <el-button type="primary" @click="handleAdd">Add</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { workspaceApi } from '@/api/workspace'
import type { WorkspaceMember } from '@/api/workspace'

const props = defineProps<{ workspaceId: number }>()

const members = ref<WorkspaceMember[]>([])
const loading = ref(false)
const newUserId = ref('')
const newRole = ref('editor')

onMounted(() => fetchMembers())

async function fetchMembers() {
  loading.value = true
  try {
    const data: any = await workspaceApi.getMembers(props.workspaceId)
    members.value = Array.isArray(data) ? data : data.items || []
  } finally { loading.value = false }
}

async function handleAdd() {
  const uid = Number(newUserId.value)
  if (!uid) { ElMessage.warning('Enter a valid user ID'); return }
  await workspaceApi.addMember(props.workspaceId, { user_id: uid, role: newRole.value })
  ElMessage.success('Member added')
  newUserId.value = ''
  await fetchMembers()
}

async function handleRemove(userId: number) {
  await workspaceApi.removeMember(props.workspaceId, userId)
  ElMessage.success('Member removed')
  await fetchMembers()
}
</script>

<style scoped lang="scss">
.member-list { display: flex; flex-direction: column; gap: 8px; }
.member-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; }
.member-info { display: flex; align-items: center; gap: 8px; }
.member-name { font-size: 14px; color: var(--color-text-primary); }
.add-member { display: flex; gap: 8px; align-items: center; }
</style>
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/settings/APIKeyManage.vue web/src/views/workspace/MemberManage.vue
git commit -m "feat: 实现 API Key 管理页面 + 完善成员管理组件"
```

---

### Task 14: 添加 404 页面 + 路由更新

**Files:**
- Create: `web/src/views/NotFound.vue`
- Modify: `web/src/router/index.ts`

- [ ] **Step 1: 创建 404 页面**

```vue
<!-- web/src/views/NotFound.vue -->
<template>
  <div class="not-found">
    <h1>404</h1>
    <p>Page not found</p>
    <router-link to="/workspaces" class="back-link">Back to Workspaces</router-link>
  </div>
</template>

<style scoped lang="scss">
.not-found {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  height: 100vh; background: var(--color-bg-deep);
  h1 { font-size: 72px; font-weight: 700; color: var(--color-primary); margin-bottom: 8px; }
  p { font-size: 18px; color: var(--color-text-secondary); margin-bottom: 24px; }
  .back-link { color: var(--color-primary); text-decoration: none; font-size: 14px;
    &:hover { color: var(--color-primary-light); }
  }
}
</style>
```

- [ ] **Step 2: 更新路由 catch-all 指向 404 页面**

修改 `router/index.ts` 中的 catch-all 路由：

```typescript
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue'),
    meta: { requiresAuth: false, title: '404' },
  },
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/NotFound.vue web/src/router/index.ts
git commit -m "feat: 添加 404 页面"
```

---

### Task 15: 端到端验证

- [ ] **Step 1: 启动后端**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go build ./cmd/main.go && echo "Backend build OK"
```

Expected: `Backend build OK`

- [ ] **Step 2: 验证前端编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/web
npm run build 2>&1 | tail -5
```

Expected: 编译成功，无 TypeScript 错误

- [ ] **Step 3: 手动验证链路**

验证清单：
1. 后端启动：`go run ./cmd/main.go`（需要 MySQL + Redis）
2. 前端启动：`npm run dev`
3. 注册新用户 → 登录
4. 创建工作空间 → 进入详情
5. 创建作品集 → 进入详情
6. 创建角色
7. 进入 AI Studio → 选择 Mock 模型 → 提交文本生成任务
8. 观察 WebSocket 推送 → 查看 Mock 结果
9. 提交图像生成任务 → 查看占位图结果
10. API Key 管理页面可正常操作

- [ ] **Step 4: Final Commit**

```bash
git add -A
git commit -m "feat: 端到端完整链路打通 - Mock AI + 前端页面完善"
```




