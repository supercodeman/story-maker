# AI 漫画创作工坊（Ai-Curton）设计文档

## 概述

Ai-Curton 是一个多 Agent 协作的 AI 漫画创作平台，支持接入 Kimi、Kiro、Copilot 等大模型，提供工作空间管理、作品集管理、角色一致性管理、AI 辅助创作等能力。

**MVP 目标**：跑通"工作空间 → 作品集 → AI 生成"核心流程，先接入 Kimi 单模型。

---

## 一、系统架构

### 架构选型：模块化单体

Go 后端作为单体服务，内部按模块划分（auth、workspace、portfolio、character、ai-agent、asset），通过接口隔离。前端 Vue SPA 直连后端 REST API。

**选型理由**：
- MVP 阶段最重要的是跑通核心流程
- Go 单体性能足够，模块化设计保留后续拆分可能
- AI 调用通过 goroutine + channel 做轻量异步
- 部署一个二进制 + 一个前端静态资源，运维成本最低

### 架构图

```
┌─────────────────────────────────────────────────────┐
│                   Vue 3 前端                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │工作空间  │  │作品集    │  │AI 创作   │          │
│  │管理      │  │管理      │  │工坊      │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
                      ↓ REST API + WebSocket
┌─────────────────────────────────────────────────────┐
│              Go 后端（单体服务）                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │Auth      │  │Workspace │  │Portfolio │          │
│  │Module    │  │Module    │  │Module    │          │
│  └──────────┘  └──────────┘  └──────────┘          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │AI Agent  │  │Character │  │Asset     │          │
│  │Module    │  │Module    │  │Module    │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
         ↓                    ↓                ↓
      MySQL 8.0          本地文件存储        Redis 7
     (业务数据)       (预留 OSS 接口)      (缓存/队列)
```

### 技术栈

**前端**：
- Vue 3 + TypeScript + Vite
- UI 框架：Element Plus（淡雅主题定制）
- 状态管理：Pinia
- 路由：Vue Router
- HTTP 客户端：Axios
- 样式：Tailwind CSS + 自定义主题（未来科技风）

**后端**：
- Go 1.21+
- Web 框架：Gin
- ORM：GORM
- 数据库：MySQL 8.0
- 缓存：Redis 7
- 文件存储：本地存储（预留 OSS 接口）
- AI SDK：各家官方 SDK

---

## 二、核心数据模型

### 用户与工作空间

```go
// 用户表
type User struct {
    ID           uint      `gorm:"primaryKey"`
    Username     string    `gorm:"uniqueIndex;size:50"`
    Email        string    `gorm:"uniqueIndex;size:100"`
    PasswordHash string    `gorm:"size:255"`
    Role         string    `gorm:"size:20"` // admin, creator, viewer
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// 工作空间表（支持个人和团队）
type Workspace struct {
    ID          uint      `gorm:"primaryKey"`
    Name        string    `gorm:"size:100"`
    Type        string    `gorm:"size:20"` // personal, team
    OwnerID     uint      `gorm:"index"`
    Description string    `gorm:"type:text"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// 工作空间成员表
type WorkspaceMember struct {
    ID          uint      `gorm:"primaryKey"`
    WorkspaceID uint      `gorm:"index"`
    UserID      uint      `gorm:"index"`
    Role        string    `gorm:"size:20"` // owner, editor, viewer
    CreatedAt   time.Time
}
```

### 作品集与资源

```go
// 作品集表
type Portfolio struct {
    ID          uint      `gorm:"primaryKey"`
    WorkspaceID uint      `gorm:"index"`
    Name        string    `gorm:"size:100"`
    Description string    `gorm:"type:text"`
    CoverImage  string    `gorm:"size:500"` // 本地路径
    Status      string    `gorm:"size:20"`  // draft, published
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// 角色模型表（人物一致性管理）
type Character struct {
    ID          uint      `gorm:"primaryKey"`
    PortfolioID uint      `gorm:"index"`
    Name        string    `gorm:"size:100"`
    Description string    `gorm:"type:text"`
    ReferenceImages string `gorm:"type:json"` // 参考图路径数组
    LoraPath    string    `gorm:"size:500"`  // LoRA 模型路径（预留）
    Attributes  string    `gorm:"type:json"` // 角色属性（发型、服装等）
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// 资源表（生成的图片、文本等）
type Asset struct {
    ID          uint      `gorm:"primaryKey"`
    PortfolioID uint      `gorm:"index"`
    Type        string    `gorm:"size:20"`  // image, text, script
    FilePath    string    `gorm:"size:500"` // 本地存储路径
    Metadata    string    `gorm:"type:json"` // 生成参数、提示词等
    CreatedBy   uint      `gorm:"index"`
    CreatedAt   time.Time
}
```

### AI 任务管理

```go
// AI 任务表
type AITask struct {
    ID          uint      `gorm:"primaryKey"`
    UserID      uint      `gorm:"index"`
    PortfolioID uint      `gorm:"index"`
    TaskType    string    `gorm:"size:50"`  // text_gen, image_gen, character_adjust
    ModelName   string    `gorm:"size:50"`  // kimi, claude, copilot
    Prompt      string    `gorm:"type:text"`
    Status      string    `gorm:"size:20"`  // pending, running, completed, failed
    Result      string    `gorm:"type:json"` // 结果数据
    ErrorMsg    string    `gorm:"type:text"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// API Key 管理表
type APIKey struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"index"`
    Provider  string    `gorm:"size:50"` // kimi, claude, copilot
    KeyValue  string    `gorm:"size:500"` // 加密存储
    IsDefault bool      // 是否使用平台默认 Key
    CreatedAt time.Time
}
```

---

## 三、API 设计

### 认证与用户

```
POST   /api/v1/auth/register          # 用户注册
POST   /api/v1/auth/login             # 用户登录
POST   /api/v1/auth/logout            # 用户登出
GET    /api/v1/user/profile           # 获取用户信息
PUT    /api/v1/user/profile           # 更新用户信息
```

### 工作空间

```
GET    /api/v1/workspaces             # 获取工作空间列表
POST   /api/v1/workspaces             # 创建工作空间
GET    /api/v1/workspaces/:id         # 获取工作空间详情
PUT    /api/v1/workspaces/:id         # 更新工作空间
DELETE /api/v1/workspaces/:id         # 删除工作空间
GET    /api/v1/workspaces/:id/members # 获取成员列表
POST   /api/v1/workspaces/:id/members # 添加成员
DELETE /api/v1/workspaces/:id/members/:user_id # 移除成员
```

### 作品集

```
GET    /api/v1/portfolios             # 获取作品集列表（按工作空间过滤）
POST   /api/v1/portfolios             # 创建作品集
GET    /api/v1/portfolios/:id         # 获取作品集详情
PUT    /api/v1/portfolios/:id         # 更新作品集
DELETE /api/v1/portfolios/:id         # 删除作品集
```

### 角色管理

```
GET    /api/v1/portfolios/:id/characters      # 获取角色列表
POST   /api/v1/portfolios/:id/characters      # 创建角色
GET    /api/v1/characters/:id                 # 获取角色详情
PUT    /api/v1/characters/:id                 # 更新角色属性
DELETE /api/v1/characters/:id                 # 删除角色
POST   /api/v1/characters/:id/reference       # 上传参考图
```

### AI 能力

```
POST   /api/v1/ai/text/generate       # 文本生成（剧本、对话）
POST   /api/v1/ai/image/generate      # 图像生成
POST   /api/v1/ai/character/adjust    # 角色调整
GET    /api/v1/ai/tasks               # 获取任务列表
GET    /api/v1/ai/tasks/:id           # 获取任务详情
DELETE /api/v1/ai/tasks/:id           # 取消任务
```

### 资源管理

```
GET    /api/v1/portfolios/:id/assets  # 获取资源列表
POST   /api/v1/assets/upload          # 上传文件
GET    /api/v1/assets/:id             # 获取资源详情
DELETE /api/v1/assets/:id             # 删除资源
```

### API Key 管理

```
GET    /api/v1/apikeys                # 获取用户的 API Key 列表
POST   /api/v1/apikeys                # 添加 API Key
PUT    /api/v1/apikeys/:id            # 更新 API Key
DELETE /api/v1/apikeys/:id            # 删除 API Key
```

---

## 四、后端模块设计

### 目录结构

```
Ai-curton/
├── server/
│   ├── cmd/
│   │   └── main.go                  # 入口
│   ├── config/
│   │   └── config.go                # 配置加载
│   ├── internal/
│   │   ├── middleware/
│   │   │   ├── auth.go              # JWT 认证
│   │   │   ├── cors.go              # 跨域
│   │   │   └── permission.go        # 权限校验
│   │   ├── model/                   # 数据模型
│   │   │   ├── user.go
│   │   │   ├── workspace.go
│   │   │   ├── portfolio.go
│   │   │   ├── character.go
│   │   │   ├── asset.go
│   │   │   └── ai_task.go
│   │   ├── handler/                 # 请求处理
│   │   │   ├── auth.go
│   │   │   ├── workspace.go
│   │   │   ├── portfolio.go
│   │   │   ├── character.go
│   │   │   ├── asset.go
│   │   │   └── ai.go
│   │   ├── service/                 # 业务逻辑
│   │   │   ├── auth.go
│   │   │   ├── workspace.go
│   │   │   ├── portfolio.go
│   │   │   ├── character.go
│   │   │   ├── asset.go
│   │   │   └── ai.go
│   │   ├── dao/                     # 数据访问
│   │   │   ├── user.go
│   │   │   ├── workspace.go
│   │   │   ├── portfolio.go
│   │   │   ├── character.go
│   │   │   ├── asset.go
│   │   │   └── ai_task.go
│   │   ├── agent/                   # AI Agent 适配层
│   │   │   ├── provider.go          # 统一接口定义
│   │   │   ├── kimi.go              # Kimi 适配器
│   │   │   ├── claude.go            # Claude/Kiro 适配器
│   │   │   ├── copilot.go           # Copilot 适配器
│   │   │   └── dispatcher.go        # 任务分发与异步执行
│   │   └── storage/                 # 文件存储抽象
│   │       ├── storage.go           # 接口定义
│   │       ├── local.go             # 本地存储实现
│   │       └── oss.go               # OSS 实现（预留）
│   ├── config.yaml
│   ├── go.mod
│   └── go.sum
└── web/                             # Vue 前端
```

### AI Agent 统一接口

```go
type AIProvider interface {
    GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error)
    GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error)
    AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error)
    Name() string
}
```

Dispatcher 负责：根据用户选择的模型路由到对应 Provider；优先使用用户自己的 API Key，没有则使用平台默认 Key；通过 goroutine 异步执行任务，状态写入 AITask 表；结果通过 WebSocket 推送给前端。

### 文件存储抽象

```go
type Storage interface {
    Upload(ctx context.Context, file io.Reader, path string) (string, error)
    Download(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    GetURL(ctx context.Context, path string) (string, error)
}
```

本期实现 LocalStorage，后续切换 OSS 只需替换实现。

---

## 五、前端设计

### 目录结构

```
web/
├── src/
│   ├── api/                         # 接口层
│   │   ├── request.ts               # Axios 封装
│   │   ├── auth.ts
│   │   ├── workspace.ts
│   │   ├── portfolio.ts
│   │   ├── character.ts
│   │   ├── asset.ts
│   │   └── ai.ts
│   ├── assets/styles/
│   │   ├── theme.scss               # 主题变量
│   │   ├── global.scss              # 全局样式
│   │   └── animation.scss           # 科技感动效
│   ├── components/
│   │   ├── layout/
│   │   │   ├── AppHeader.vue
│   │   │   ├── AppSidebar.vue
│   │   │   └── AppLayout.vue
│   │   ├── common/
│   │   │   ├── GlowCard.vue         # 发光卡片
│   │   │   ├── NeonButton.vue       # 霓虹按钮
│   │   │   └── FadePanel.vue        # 渐变面板
│   │   └── ai/
│   │       ├── ModelSelector.vue
│   │       ├── PromptEditor.vue
│   │       └── TaskProgress.vue
│   ├── views/
│   │   ├── auth/                    # 登录/注册
│   │   ├── workspace/               # 工作空间
│   │   ├── portfolio/               # 作品集
│   │   ├── character/               # 角色管理
│   │   ├── studio/                  # AI 创作工坊
│   │   └── settings/                # API Key 管理
│   ├── store/                       # Pinia
│   ├── router/
│   ├── utils/
│   │   ├── websocket.ts
│   │   └── storage.ts
│   ├── App.vue
│   └── main.ts
├── vite.config.ts
├── tailwind.config.js
└── package.json
```

### 视觉风格（淡雅 + 未来科技感）

```scss
$colors: (
  primary:        #7C8CF8,    // 主色：低饱和度蓝紫
  primary-light:  #A5B4FC,
  primary-dark:   #5B6AE0,
  bg-deep:        #0F1117,    // 背景：深色系
  bg-surface:     #1A1D2E,
  bg-card:        #232640,
  bg-hover:       #2A2E4A,
  text-primary:   #E8EAF6,    // 文字
  text-secondary: #9CA3C0,
  text-muted:     #5C6280,
  accent-cyan:    #67E8F9,    // 强调色
  accent-green:   #6EE7B7,
  accent-amber:   #FCD34D,
  border-glow:    rgba(124, 140, 248, 0.2),
);
```

核心视觉特征：深色背景 + 低饱和度色彩；卡片带微弱发光边框；按钮带呼吸光效动画；大量使用 `backdrop-filter: blur()` 毛玻璃效果；等宽 + 无衬线字体混排。

### 页面布局

```
┌──────────────────────────────────────────────────┐
│  AppHeader（Logo + 工作空间切换 + 用户头像）       │
├────────┬─────────────────────────────────────────┤
│        │                                         │
│  侧边栏 │          主内容区                       │
│ 作品集  │  ┌─────────────────────────────────┐   │
│ 角色库  │  │  AI 创作工坊 / 作品集详情        │   │
│ AI工具  │  └─────────────────────────────────┘   │
│ 设置    │                                         │
├────────┴─────────────────────────────────────────┤
│  底部状态栏（AI 任务队列状态 + 存储用量）          │
└──────────────────────────────────────────────────┘
```

---

## 六、AI Agent 协作设计

### 多模型调度架构

```
用户发起请求 → AI Handler → AI Service → Dispatcher → Provider 适配层
                                                        ├── Kimi
                                                        ├── Claude/Kiro
                                                        └── Copilot
                                                             ↓
                                              更新 AITask 状态 → WebSocket 推送
```

### API Key 选择策略

1. 优先查找用户自己的 Key
2. 回退到平台默认 Key
3. 都没有则返回错误

### 任务类型与模型能力映射

| 任务类型 | 说明 | Kimi | Claude/Kiro | Copilot |
|---------|------|------|-------------|---------|
| `text_gen` | 剧本/对话生成 | ✅ | ✅ | ✅ |
| `text_polish` | 文本润色 | ✅ | ✅ | ✅ |
| `storyboard` | 分镜建议 | ✅ | ✅ | ✅ |
| `image_gen` | 文生图 | ✅ | ❌ | ✅ |
| `image_edit` | 局部重绘 | ✅ | ❌ | ✅ |
| `character_adjust` | 角色调整 | ✅ | ❌ | ✅ |

不支持的能力在前端模型选择器中自动置灰。Claude/Kiro 主要用于文本类任务。

### 角色一致性方案（MVP）

MVP 阶段采用**提示词约束 + 参考图**方案，不做 LoRA 训练：

1. 用户上传角色参考图（1-5 张），填写角色属性描述
2. 系统将参考图 + 属性描述组装成结构化提示词
3. 调用 AI 模型生成时，自动注入角色约束提示词
4. 生成结果关联到角色，积累参考图库
5. 后续版本引入 LoRA 微调能力

---

## 七、安全与部署

### 安全策略

**认证**：JWT Token，access_token（2h）+ refresh_token（7d）

**API Key 加密**：AES-256-GCM 加密存入 MySQL，密钥通过环境变量注入

**权限模型**：

| 操作 | owner | editor | viewer |
|------|-------|--------|--------|
| 查看工作空间/作品集 | ✅ | ✅ | ✅ |
| 创建/编辑作品集 | ✅ | ✅ | ❌ |
| 删除作品集 | ✅ | ❌ | ❌ |
| 管理成员 | ✅ | ❌ | ❌ |
| 使用 AI 能力 | ✅ | ✅ | ❌ |
| 删除工作空间 | ✅ | ❌ | ❌ |

**文件上传**：限制单文件 20MB，仅允许 jpg/png/webp/gif，服务端校验 MIME 类型

### 本地部署

```yaml
# docker-compose.yml
services:
  server:
    build: ./server
    ports: ["8080:8080"]
    volumes: ["./data/uploads:/app/uploads"]
    depends_on: [mysql, redis]
    environment:
      - DB_DSN=root:password@tcp(mysql:3306)/ai_curton?charset=utf8mb4
      - REDIS_ADDR=redis:6379
      - ENCRYPT_KEY=${ENCRYPT_KEY}
      - KIMI_API_KEY=${KIMI_API_KEY}
  web:
    build: ./web
    ports: ["3000:80"]
  mysql:
    image: mysql:8.0
    volumes: ["./data/mysql:/var/lib/mysql"]
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=ai_curton
  redis:
    image: redis:7-alpine
    volumes: ["./data/redis:/data"]
```

启动命令：`docker-compose up -d`

---

## 八、MVP 范围

### 本期交付

- 用户注册/登录
- 个人 + 团队工作空间（CRUD + 成员管理）
- 作品集管理（CRUD）
- 角色管理（属性编辑 + 参考图上传）
- 接入 Kimi（文本 + 图像生成）
- AI 任务异步执行 + WebSocket 推送
- API Key 管理（用户自有 + 平台默认）
- 本地文件存储（预留 OSS 接口）
- Docker Compose 本地部署

### 后续迭代

- 接入 Claude/Kiro、Copilot
- LoRA 角色微调
- OSS 文件存储
- 分镜编辑器
- 作品发布与分享
