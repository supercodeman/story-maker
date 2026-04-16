# Ai-Curton 项目架构设计文档

> AI 驱动的创意工坊平台 — 整体架构、页面结构、组件层级、路由设计与设计系统

---

## 一、系统架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client (Browser)                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  Vue 3 + TypeScript + Vite + Element Plus + Pinia            │  │
│  │  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │  │
│  │  │ Router  │  │  Store   │  │   API    │  │  WebSocket   │  │  │
│  │  │ (Guard) │  │ (Pinia)  │  │ (Axios)  │  │  (Realtime)  │  │  │
│  │  └────┬────┘  └────┬─────┘  └────┬─────┘  └──────┬───────┘  │  │
│  │       │            │             │                │           │  │
│  │  ┌────┴────────────┴─────────────┴────────────────┴───────┐  │  │
│  │  │              Views / Components Layer                   │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ HTTP / WebSocket
┌──────────────────────────────┴──────────────────────────────────────┐
│                      Server (Go + Gin)                              │
│  ┌──────────┐  ┌───────────┐  ┌───────────┐  ┌──────────────────┐  │
│  │Middleware │  │  Handler  │  │  Service  │  │      DAO         │  │
│  │ JWT/CORS │→ │ HTTP+WS   │→ │  业务逻辑  │→ │  数据访问层      │  │
│  │ Logger   │  │ 请求处理   │  │  权限校验  │  │  GORM + MySQL   │  │
│  └──────────┘  └───────────┘  └─────┬─────┘  └──────────────────┘  │
│                                     │                               │
│  ┌──────────────────────────────────┴────────────────────────────┐  │
│  │                    Agent (AI Provider)                         │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │  │
│  │  │Dispatcher│→ │  Zhipu   │  │   Kimi   │  │    Mock      │  │  │
│  │  │ 任务分发  │  │ GLM-4    │  │ Moonshot │  │   (Dev)      │  │  │
│  │  │ 异步执行  │  │ CogView  │  │          │  │              │  │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────┴──────────────────────────────────────┐
│                     Infrastructure                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │  MySQL 8.0   │  │  Redis 7     │  │  Local File Storage      │  │
│  │  数据持久化   │  │  Token 缓存   │  │  uploads/                │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 二、前端架构

### 2.1 技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| 框架 | Vue 3 (Composition API) | ^3.4.0 |
| 语言 | TypeScript | ^5.3.0 |
| 构建 | Vite | ^5.0.0 |
| UI 库 | Element Plus | ^2.5.0 |
| 状态管理 | Pinia | ^2.1.7 |
| HTTP | Axios | ^1.6.0 |
| 样式 | SCSS + Tailwind CSS | sass ^1.69 / tw ^3.4 |
| 路由 | Vue Router | ^4.2.5 |

### 2.2 目录结构

```
web/src/
├── main.ts                     # 应用入口
├── App.vue                     # 根组件
├── router/index.ts             # 路由配置 + 导航守卫
├── store/                      # Pinia 状态管理
│   ├── user.ts                 #   用户认证
│   ├── ai.ts                   #   AI 任务 + WebSocket 状态
│   ├── workspace.ts            #   工作空间
│   ├── portfolio.ts            #   作品集
│   └── character.ts            #   角色
├── api/                        # API 接口层
│   ├── request.ts              #   Axios 实例 + 拦截器
│   ├── types.ts                #   通用类型
│   ├── ai.ts                   #   AI 接口
│   ├── workspace.ts            #   工作空间接口
│   ├── portfolio.ts            #   作品集接口
│   └── character.ts            #   角色接口
├── components/
│   ├── layout/                 # 布局组件
│   │   ├── AppLayout.vue       #   主布局（Header + Sidebar + Main + Footer）
│   │   ├── AppHeader.vue       #   顶部导航栏 (64px)
│   │   └── AppSidebar.vue      #   左侧菜单栏 (240px)
│   └── common/                 # 通用组件
│       ├── GlowCard.vue        #   发光卡片
│       ├── NeonButton.vue      #   霓虹按钮
│       └── FadePanel.vue       #   渐变面板
├── views/                      # 页面组件
│   ├── auth/                   #   认证模块
│   ├── workspace/              #   工作空间模块
│   ├── portfolio/              #   作品集模块
│   ├── character/              #   角色模块
│   ├── studio/                 #   AI Studio 模块（核心）
│   └── settings/               #   设置模块
├── utils/websocket.ts          # WebSocket 管理
└── assets/styles/
    ├── theme.scss              # 主题变量
    ├── global.scss             # 全局样式
    └── animation.scss          # 动画定义
```

### 2.3 路由结构

```
/login                                    # 登录页（无需认证）
/register                                 # 注册页（无需认证）
/                                         # AppLayout 主布局（需认证）
├── /workspaces                           #   工作空间列表
├── /workspace/:id                        #   工作空间详情
├── /workspace/:id/portfolio/:pid         #   作品集详情
├── /workspace/:id/portfolio/:pid/characters  # 角色列表
├── /workspace/:id/portfolio/:pid/studio  #   AI Studio（核心页面）
└── /settings/apikeys                     #   API Key 管理
```

### 2.4 页面布局结构

```
┌─────────────────────────────────────────────────────────────┐
│  AppHeader (64px)                                [User]     │
│  Logo  |  Space Selector                                    │
├────────┬────────────────────────────────────────────────────┤
│        │                                                    │
│  App   │  AppLayout__main (flex:1, overflow-y:auto)         │
│  Side  │  ┌──────────────────────────────────────────────┐  │
│  bar   │  │  <router-view />                             │  │
│        │  │  各页面内容在此渲染                            │  │
│ 240px  │  │                                              │  │
│        │  └──────────────────────────────────────────────┘  │
│ - Workspaces                                                │
│ - Portfolios                                                │
│ - Settings                                                  │
│        │                                                    │
├────────┴────────────────────────────────────────────────────┤
│  AppLayout__footer (32px)          Connected    Ai-Curton   │
└─────────────────────────────────────────────────────────────┘
```

### 2.5 AI Studio 页面布局（核心页面）

```
┌─────────────────────────────────────────────────────────────┐
│  Breadcrumb: Workspace / Portfolio / AI Studio   Connected  │
│  AI Studio                                                  │
├──────────────┬──────────────────────────────────────────────┤
│              │                                              │
│  Settings    │  Chat Messages Area (flex:1, scroll)         │
│  ┌────────┐  │  ┌────────────────────────────────────────┐  │
│  │TaskType│  │  │                                        │  │
│  │Text/Img│  │  │  👤 You          [用户消息气泡]  →     │  │
│  │        │  │  │                                        │  │
│  │ Model  │  │  │  ← [AI 回复气泡]          🤖 zhipu    │  │
│  │ Select │  │  │                                        │  │
│  │        │  │  │  👤 You          [用户消息气泡]  →     │  │
│  │ClearBtn│  │  │                                        │  │
│  └────────┘  │  │  ← [Loading...]           🤖 zhipu    │  │
│              │  │     ● ● ● Thinking...                  │  │
│  History     │  └────────────────────────────────────────┘  │
│  ┌────────┐  │                                              │
│  │+ New   │  ├──────────────────────────────────────────────┤
│  │        │  │  Chat Input Area                             │
│  │▶ task1 │  │  ┌──────────────────────────┐  ┌──────────┐ │
│  │  task2 │  │  │  Type your message...    │  │   Send   │ │
│  │  task3 │  │  └──────────────────────────┘  └──────────┘ │
│  │  ...   │  │                                              │
│  └────────┘  │                                              │
│   260px      │              flex: 1                         │
└──────────────┴──────────────────────────────────────────────┘
```

### 2.6 组件层级关系

```
App.vue
└── <router-view />
    ├── Login.vue                    # /login
    ├── Register.vue                 # /register
    └── AppLayout.vue                # / (认证后)
        ├── AppHeader.vue
        │   ├── Logo
        │   ├── WorkspaceSelector (el-select)
        │   └── UserMenu (el-dropdown)
        ├── AppSidebar.vue
        │   ├── NavItem: Workspaces
        │   ├── NavItem: Portfolios
        │   └── NavItem: Settings
        ├── <router-view />          # 内容区
        │   ├── WorkspaceList.vue
        │   │   └── GlowCard (×N)
        │   ├── WorkspaceDetail.vue
        │   │   ├── GlowCard (信息)
        │   │   ├── GlowCard (作品集列表)
        │   │   └── GlowCard (成员管理)
        │   ├── PortfolioDetail.vue
        │   │   ├── GlowCard (信息)
        │   │   ├── GlowCard (角色列表)
        │   │   └── NeonButton (操作)
        │   ├── CharacterList.vue
        │   │   └── GlowCard (×N 角色卡片)
        │   ├── AIStudio.vue         # ★ 核心页面
        │   │   ├── GlowCard (Settings)
        │   │   ├── GlowCard (History)
        │   │   ├── ChatMessages
        │   │   │   ├── ChatMsg--user (×N)
        │   │   │   ├── ChatMsg--assistant (×N)
        │   │   │   └── ChatMsg--loading
        │   │   └── ChatInput
        │   │       ├── el-input (textarea)
        │   │       └── NeonButton (Send)
        │   └── APIKeyManage.vue
        │       └── GlowCard (Key 列表)
        └── AppFooter (内联)
```

---

## 三、后端架构

### 3.1 技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.22 |
| Web 框架 | Gin | v1.9.1 |
| ORM | GORM | v1.25.7 |
| 数据库 | MySQL | 8.0 |
| 缓存 | Redis | 7 |
| 认证 | JWT (golang-jwt) | v5.2.1 |
| WebSocket | Gorilla WebSocket | v1.5.1 |
| HTTP 客户端 | Resty | v2.11.0 |
| 配置 | Viper | v1.18.2 |

### 3.2 分层架构

```
┌─────────────────────────────────────────────────────┐
│                   Router (路由层)                     │
│  定义 URL → Handler 映射，挂载中间件                   │
├─────────────────────────────────────────────────────┤
│                 Middleware (中间件层)                  │
│  ┌────────┐ ┌──────┐ ┌────────┐ ┌──────────────┐   │
│  │  JWT   │ │ CORS │ │ Logger │ │   Recovery    │   │
│  │  认证  │ │ 跨域  │ │  日志  │ │   异常恢复    │   │
│  └────────┘ └──────┘ └────────┘ └──────────────┘   │
├─────────────────────────────────────────────────────┤
│                  Handler (处理层)                     │
│  解析请求参数 → 调用 Service → 返回统一响应格式         │
│  auth / user / workspace / portfolio / character     │
│  ai / asset / api_key / ws                           │
├─────────────────────────────────────────────────────┤
│                  Service (服务层)                     │
│  业务逻辑 + 权限校验 + 输入验证                        │
│  AuthService / WorkspaceService / AIService ...      │
├─────────────────────────────────────────────────────┤
│                    DAO (数据访问层)                    │
│  GORM 操作封装，一个 Model 对应一个 DAO               │
│  UserDAO / WorkspaceDAO / AITaskDAO ...              │
├─────────────────────────────────────────────────────┤
│                   Model (模型层)                      │
│  GORM 结构体定义 + AutoMigrate                       │
│  User / Workspace / Portfolio / Character / AITask   │
└─────────────────────────────────────────────────────┘
```

### 3.3 AI 任务处理流程

```
用户发送 Prompt
       │
       ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Handler    │────▶│   Service    │────▶│  Dispatcher  │
│  解析请求     │     │  校验 + 创建  │     │  能力检查     │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                  │
                              ┌────────────────────┤
                              │ CreateTask (DB)    │
                              │ status: pending    │
                              ▼                    │
                     ┌──────────────┐              │
                     │  返回 taskId  │◀─────────────┘
                     │  给前端       │
                     └──────────────┘
                              │
                     go executeTask()  ← 异步 goroutine
                              │
                              ▼
                     ┌──────────────┐
                     │ status:running│──▶ WebSocket 推送
                     └──────┬───────┘
                            │
                   ┌────────┴────────┐
                   ▼                 ▼
            ┌────────────┐   ┌────────────┐
            │  Provider  │   │  Provider  │
            │  Zhipu     │   │  Kimi      │
            │  GLM-4     │   │  Moonshot  │
            │  CogView   │   │            │
            └─────┬──────┘   └─────┬──────┘
                  │                │
                  ▼                ▼
            ┌──────────────────────────┐
            │  handleTaskSuccess /     │
            │  handleTaskError         │
            │  status: completed/failed│
            └────────────┬─────────────┘
                         │
                         ▼
              ┌──────────────────┐
              │  WebSocket 推送   │──▶ 前端 watch 更新对话
              │  task_update      │
              └──────────────────┘
```

### 3.4 数据模型关系

```
User ─────────────────────────────────────────────┐
 │                                                 │
 ├──▶ Workspace (1:N)                              │
 │     │                                           │
 │     ├──▶ WorkspaceMember (1:N)                  │
 │     │     └── role: owner / editor / viewer     │
 │     │                                           │
 │     └──▶ Portfolio (1:N)                        │
 │           │                                     │
 │           ├──▶ Character (1:N)                  │
 │           │     └── reference_image             │
 │           │                                     │
 │           ├──▶ Asset (1:N)                      │
 │           │     └── file_path, file_type        │
 │           │                                     │
 │           └──▶ AITask (1:N)                     │
 │                 ├── task_type: text_gen /        │
 │                 │   image_gen / character_adjust │
 │                 ├── model_name: zhipu / kimi     │
 │                 ├── prompt + history (JSON)      │
 │                 ├── status: pending → running    │
 │                 │   → completed / failed         │
 │                 └── result (JSON)                │
 │                                                  │
 └──▶ APIKey (1:N) ────────────────────────────────┘
       ├── provider: zhipu / kimi
       └── key_encrypted (AES-256)
```

### 3.5 API 路由总览

```
/api/v1
├── /auth                          # 认证（公开）
│   ├── POST   /register
│   ├── POST   /login
│   └── POST   /logout
│
├── /user                          # 用户（需认证）
│   ├── GET    /profile
│   └── PUT    /profile
│
├── /workspaces                    # 工作空间
│   ├── GET    /                   # 列表
│   ├── POST   /                   # 创建
│   ├── GET    /:id                # 详情
│   ├── PUT    /:id                # 更新
│   ├── DELETE /:id                # 删除
│   ├── GET    /:id/members        # 成员列表
│   ├── POST   /:id/members        # 添加成员
│   └── DELETE /:id/members/:uid   # 移除成员
│
├── /portfolios                    # 作品集
│   ├── GET    /
│   ├── POST   /
│   ├── GET    /:id
│   ├── PUT    /:id
│   ├── DELETE /:id
│   ├── GET    /:id/characters     # 关联角色
│   ├── POST   /:id/characters
│   └── GET    /:id/assets         # 关联资产
│
├── /characters                    # 角色
│   ├── GET    /:id
│   ├── PUT    /:id
│   ├── DELETE /:id
│   └── POST   /:id/reference     # 上传参考图
│
├── /assets                        # 资产
│   ├── POST   /upload
│   ├── GET    /:id
│   └── DELETE /:id
│
├── /ai                            # AI 任务
│   ├── POST   /text/generate      # 文本生成
│   ├── POST   /image/generate     # 图像生成
│   ├── POST   /character/adjust   # 角色调整
│   ├── GET    /tasks              # 任务列表
│   ├── GET    /tasks/:id          # 任务详情
│   └── DELETE /tasks/:id          # 取消任务
│
├── /apikeys                       # API Key 管理
│   ├── GET    /
│   ├── POST   /
│   ├── PUT    /:id
│   └── DELETE /:id
│
└── GET /ws?token=xxx              # WebSocket 实时通信
```

---

## 四、设计系统 (Design System)

> 基于 UI/UX Pro Max 分析，结合项目现有风格

### 4.1 设计风格

| 属性 | 值 |
|------|-----|
| 风格 | Cyberpunk UI / Dark Mode (OLED) |
| 关键词 | Neon, dark mode, glow, sci-fi, futuristic |
| 适用场景 | AI 工具、创意平台、开发者工具 |
| 性能 | ⚠ Moderate（glow 效果需控制） |
| 无障碍 | ⚠ 需注意深色背景对比度 |

### 4.2 色彩体系

```
┌─────────────────────────────────────────────────────┐
│  Color Tokens (theme.scss)                          │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Primary        #7C8CF8  ████████  紫蓝（主色调）    │
│  Primary Light  #A5B4FC  ████████  浅紫蓝           │
│                                                     │
│  Background                                         │
│    Deep         #0F1117  ████████  最深背景          │
│    Base         #1A1D2E  ████████  卡片/面板背景     │
│    Elevated     #232640  ████████  悬浮/高亮背景     │
│                                                     │
│  Text                                               │
│    Primary      #E8EAF6  ████████  主要文字          │
│    Secondary    #9CA3C0  ████████  次要文字          │
│    Muted        #5C6280  ████████  辅助/禁用文字     │
│                                                     │
│  Accent                                             │
│    Cyan         #22D3EE  ████████  强调/链接         │
│    Green        #22C55E  ████████  成功/在线         │
│    Amber        #F59E0B  ████████  警告/进行中       │
│    Red          #EF4444  ████████  错误/失败         │
│                                                     │
│  Border                                             │
│    Default      #2A2D3E  ████████  默认边框          │
│    Glow         rgba(124,140,248,0.3)  发光边框      │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 4.3 字体体系

| 用途 | 字体 | 字重 | 大小 |
|------|------|------|------|
| 页面标题 | System / Inter | 700 | 28px |
| 面板标题 | System / Inter | 600 | 16px |
| 正文 | System / Inter | 400 | 14px |
| 辅助文字 | System / Inter | 400 | 13px |
| 标签/状态 | System / Inter | 500 | 11px |
| 代码/AI输出 | Monospace | 400 | 14px |

推荐升级方案（Cyberpunk 风格增强）：
- 标题字体：**Orbitron** (700/900)
- 正文字体：**JetBrains Mono** (400/500)

### 4.4 间距系统

```
基础单位: 4px

Spacing Scale:
  xs:   4px   (紧凑间距)
  sm:   8px   (元素内间距)
  md:  12px   (组件间距)
  lg:  16px   (区块间距)
  xl:  20px   (面板内边距)
  2xl: 24px   (区域间距)
  3xl: 32px   (大区块间距)
```

### 4.5 圆角系统

| 元素 | 圆角 |
|------|------|
| 按钮 | 8px |
| 卡片/面板 | 12px |
| 对话气泡 | 12px (角落 2px) |
| 输入框 | 8px |
| 头像 | 50% (圆形) |
| 状态标签 | 4px |

### 4.6 阴影与发光效果

```scss
// 卡片发光效果 (GlowCard)
box-shadow: 0 0 15px rgba(124, 140, 248, 0.1),
            0 0 30px rgba(124, 140, 248, 0.05);
border: 1px solid rgba(124, 140, 248, 0.2);

// 按钮霓虹效果 (NeonButton)
box-shadow: 0 0 10px rgba(124, 140, 248, 0.4),
            0 0 20px rgba(124, 140, 248, 0.2);

// 状态指示灯
.ws-dot.connected { background: #22c55e; }
.ws-dot.disconnected { background: #ef4444; }
```

### 4.7 自定义组件规范

| 组件 | 用途 | 特效 |
|------|------|------|
| GlowCard | 内容容器 | 边框发光 + 背景渐变 |
| NeonButton | 主要操作 | 霓虹发光 + hover 增强 |
| FadePanel | 渐变面板 | 渐入动画 |

### 4.8 状态颜色映射

```
Task Status:
  completed  → #22C55E (绿色)  + rgba(34,197,94,0.1) 背景
  failed     → #EF4444 (红色)  + rgba(239,68,68,0.1) 背景
  running    → #F59E0B (琥珀)  + rgba(245,158,11,0.1) 背景
  pending    → #F59E0B (琥珀)  + rgba(245,158,11,0.1) 背景

WebSocket:
  connected    → #22C55E 绿点
  disconnected → #EF4444 红点
```

---

## 五、数据流

### 5.1 认证流程

```
┌────────┐  POST /auth/login   ┌────────┐  JWT Token   ┌────────┐
│ Login  │ ──────────────────▶ │ Server │ ───────────▶ │ Store  │
│  Page  │                     │  Auth  │              │ (user) │
└────────┘                     └────────┘              └───┬────┘
                                                          │
                                              localStorage.setItem
                                              ('access_token', token)
                                                          │
                                                          ▼
                                              Axios 拦截器自动附加
                                              Authorization: Bearer xxx
```

### 5.2 AI 对话数据流

```
┌──────────┐  inputText   ┌──────────┐  submitTask   ┌──────────┐
│ ChatInput│ ───────────▶ │ AIStudio │ ────────────▶ │ AI Store │
│          │              │handleSend│               │          │
└──────────┘              └──────────┘               └────┬─────┘
                                                          │
                               POST /ai/text/generate     │
                               POST /ai/image/generate    │
                                                          ▼
┌──────────┐  task_update  ┌──────────┐  Dispatch    ┌──────────┐
│WebSocket │ ◀──────────── │  Server  │ ◀─────────── │ Provider │
│ Client   │               │   Hub    │              │ Zhipu/   │
└────┬─────┘               └──────────┘              │ Kimi     │
     │                                               └──────────┘
     │ handleTaskUpdate
     ▼
┌──────────┐  watch tasks  ┌──────────┐  push msg    ┌──────────┐
│ AI Store │ ────────────▶ │ AIStudio │ ───────────▶ │ Messages │
│  tasks[] │               │  watch() │              │  Array   │
└──────────┘               └──────────┘              └──────────┘
```

### 5.3 轮询兜底机制

```
handleSend() 成功
     │
     ├──▶ pendingTaskId = taskId
     ├──▶ activeTaskId = taskId
     ├──▶ generating = true
     └──▶ startPolling()  ← 每 3s fetchTasks
              │
              ▼
     WebSocket 推送 OR 轮询拉取
              │
              ▼
     watch(tasks) 检测到 completed/failed
              │
              ├──▶ generating = false
              ├──▶ push AI message
              └──▶ stopPolling()
```

---

## 六、部署架构

```
┌─────────────────────────────────────────────────┐
│              Docker Compose                      │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ MySQL 8  │  │ Redis 7  │  │  Go Server   │  │
│  │ :3306    │  │ :6379    │  │  :8080       │  │
│  │          │  │          │  │  Gin + WS    │  │
│  └──────────┘  └──────────┘  └──────┬───────┘  │
│                                      │          │
│                              ┌───────┴───────┐  │
│                              │  Vite (Dev)   │  │
│                              │  :3000        │  │
│                              │  Proxy → 8080 │  │
│                              └───────────────┘  │
└─────────────────────────────────────────────────┘

外部依赖:
  ├── 智谱 AI API  (https://open.bigmodel.cn)
  └── Kimi API     (https://api.moonshot.cn)
```

---

## 七、安全架构

```
┌─────────────────────────────────────────────────┐
│                  安全层级                         │
├─────────────────────────────────────────────────┤
│                                                  │
│  L1: 传输层                                      │
│      └── CORS 白名单 + HTTPS (生产环境)           │
│                                                  │
│  L2: 认证层                                      │
│      ├── JWT (access_token: 2h, refresh: 7d)    │
│      └── 密码 bcrypt 加密                        │
│                                                  │
│  L3: 授权层                                      │
│      ├── 工作空间角色: owner > editor > viewer    │
│      └── 资源归属校验 (user_id 匹配)              │
│                                                  │
│  L4: 数据层                                      │
│      ├── API Key AES-256 加密存储                 │
│      ├── SQL 注入防护 (GORM 参数化)               │
│      └── 文件上传大小限制 (20MB)                   │
│                                                  │
└─────────────────────────────────────────────────┘
```

---

*文档生成时间: 2026-04-02*
*基于 UI/UX Pro Max Design Intelligence 分析*
