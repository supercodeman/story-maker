# Ai-Curton

AI 驱动的创意工坊平台 —— 集成多 Agent 协作、工作流编排、智能联想、RAG 记忆系统于一体的小说辅助创作全栈应用。

## 功能特性

**AI 多 Agent 体系**
- 四层架构：Provider → Executor → Dispatcher → Orchestrator
- 多模型适配：通义千问（默认）、智谱 GLM、DeepSeek、Kimi Moonshot，统一接口无缝切换
- DAG 编排引擎：支持拓扑分层并行执行、节点间数据传递（SharedState）、条件边跳过
- 两层自动降级：同 Provider 内模型降级 + 跨 Provider 降级，IsRetryableError 智能判定
- Function Calling：工具注册表模式，LLM 可调用外部工具（web_search、knowledge_query）

**智能联想系统**
- 实时续写联想：编辑器输入停顿 800ms 自动触发，生成 50-150 字续写建议
- 意图推断：IntentService 分析前文语境（对话/描写/转场/高潮等）
- 写作风格注入：WritingStyleService 提供风格规范约束
- 用户偏好学习：UserBehaviorService 追踪采纳/修改/拒绝行为，动态优化 PromptSummary
- Tab 采纳 / Esc 取消：InlineSuggestionEditor 灰色叠加文本交互

**小说工坊**
- 章节管理：创建、编辑、排序、版本历史（ChapterVersion 版本链）
- AI 辅助写作：概要润色、正文润色、扩写、续写
- 划词操作：选中文本局部润色/扩写，精确替换
- 一键生成：工作流编排自动生成完整章节（大纲 → 角色/场景/对话并行 → 整合润色）
- 大纲工坊：AI 驱动的小说大纲自动生成与采纳
- Prompt 模板：Go text/template 语法，支持系统默认和小说级自定义
- AI 管家对话：侧边栏对话式 AI 助手（ConversationService）
- 爆款分析：HitAnalysisService 分析热门作品特征

**记忆与上下文（RAG）**
- 短期记忆：编辑器实时状态（光标前文 500 字、划词文本）
- 中期记忆：会话级对话历史持久化、SharedState DAG 节点间传递
- 长期记忆：Memory 记忆库（文档上传 → 分块 → Embedding → 语义检索）
- 记忆绑定：将记忆库绑定到小说，章节生成时自动检索 Top-K 相关 chunk 注入 Prompt
- 记忆市场：用户可发布/购买记忆库（世界观、角色设定等）
- 知识库：KnowledgeService 提供角色设定 + 世界观笔记检索

**平台能力**
- 工作空间：个人/团队协作，角色权限（owner/editor/viewer）
- 作品集管理：归属工作空间的作品集组织
- 角色管理：人物一致性管理，支持参考图
- AI Studio：文本/图像生成，多模型对比
- API Key 管理：用户自有 Key + 平台默认 Key，AES-256 加密存储
- 钱包系统：Token 消耗计量与余额管理
- WebSocket 实时推送：任务状态、工作流进度实时通知
- 情节结构：PlotStructureService 时间线管理

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 后端 | Go + Gin | 1.22+ |
| ORM | GORM + MySQL | 8.0 |
| 缓存 | Redis | 7 |
| 前端 | Vue 3 + TypeScript | 3.4+ |
| UI | Element Plus (暗色主题) | 2.5+ |
| 状态管理 | Pinia | 2.x |
| 构建 | Vite | 5.x |
| 实时通信 | gorilla/websocket | 1.5 |
| 认证 | JWT (access 2h + refresh 7d) | - |

## 项目结构

```
Ai-curton/
├── server/                          # Go 后端
│   ├── cmd/main.go                  # 入口
│   ├── config/                      # 配置文件
│   ├── scripts/schema.sql           # 数据库初始化
│   └── internal/
│       ├── model/        (23)       # 数据模型
│       ├── dao/          (23)       # 数据访问层
│       ├── service/      (30)       # 业务逻辑层
│       ├── handler/      (24)       # HTTP 处理器
│       ├── router/                  # 路由配置 + 依赖注入
│       ├── middleware/              # 中间件（JWT、CORS）
│       ├── agent/        (15)       # AI Agent 子系统
│       │   ├── dispatcher.go        # 任务分发 + 两层降级
│       │   ├── provider.go          # AIProvider 接口定义
│       │   ├── qwen.go             # 通义千问（默认 Provider）
│       │   ├── zhipu.go            # 智谱适配
│       │   ├── deepseek.go         # DeepSeek 适配
│       │   ├── kimi.go             # Kimi 适配
│       │   ├── executor*.go        # 任务执行器（文本/章节/大纲/图像）
│       │   ├── tool.go             # 工具注册表 + Function Calling
│       │   ├── token.go            # Token 计量
│       │   └── content_cleaner.go  # 内容清洗
│       └── storage/                 # 文件存储
├── web/                             # Vue 3 前端
│   └── src/
│       ├── api/          (21)       # HTTP 接口层
│       ├── store/        (15)       # Pinia 状态管理
│       ├── views/        (31)       # 页面视图
│       ├── components/              # 组件
│       │   ├── layout/             # AppLayout / Header / Sidebar
│       │   ├── ai/                 # TaskResult
│       │   ├── common/             # GlowCard / NeonButton / FadePanel
│       │   └── editor/             # InlineSuggestionEditor
│       └── assets/styles/           # theme.scss / global.scss
├── docker-compose.yml               # Docker 编排
└── architecture.html                # 架构设计文档（可视化，暗色/亮色双主题）
```

## 快速开始

### 环境要求

- Go 1.22+
- Node.js 16+
- MySQL 8.0
- Redis 7

### 数据库初始化

```bash
mysql -u root -p < server/scripts/schema.sql
```

### 后端启动

```bash
cd server
cp config/config.yaml config/config.local.yaml  # 按需修改配置
go mod tidy
go run cmd/main.go
```

后端默认监听 `:8080`。

### 前端启动

```bash
cd web
npm install
npm run dev
```

前端开发服务器 `http://localhost:3000`，自动代理 `/api` → `http://localhost:8080`。

### Docker 启动

```bash
docker-compose up -d
```

一键启动 MySQL + Redis + Server，访问 `http://localhost:8080`。

## 配置说明

后端配置文件 `server/config/config.yaml`：

```yaml
server:
  port: 8080

database:
  host: 127.0.0.1
  port: 3306
  name: ai_curton
  user: root
  password: your_password

redis:
  addr: 127.0.0.1:6379

jwt:
  secret: your_jwt_secret

ai:
  qwen:
    api_key: your_qwen_key       # 通义千问（默认 Provider）
  zhipu:
    api_key: your_zhipu_key      # 智谱 API Key
  deepseek:
    api_key: your_deepseek_key   # DeepSeek API Key
  kimi:
    api_key: your_kimi_key       # Kimi API Key
```

> 用户也可在平台设置页面配置自己的 API Key，优先级高于平台默认 Key。

## API 概览

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | `POST /api/v1/auth/register` | 注册 |
| | `POST /api/v1/auth/login` | 登录 |
| 工作空间 | `GET/POST /api/v1/workspaces` | 列表/创建 |
| 作品集 | `GET/POST /api/v1/workspaces/:id/portfolios` | 列表/创建 |
| 小说 | `GET/POST /api/v1/novels` | 列表/创建 |
| 章节 | `PUT /api/v1/chapters/:id` | 更新章节 |
| | `POST /api/v1/chapters/:id/ai` | AI 辅助写作 |
| 大纲 | `POST /api/v1/outline/generate` | 生成大纲 |
| AI | `POST /api/v1/ai/text/generate` | 文本生成 |
| | `POST /api/v1/ai/image/generate` | 图像生成 |
| 联想 | `POST /api/v1/suggestion` | 实时联想 |
| 记忆 | `GET/POST /api/v1/memories` | 记忆库管理 |
| | `POST /api/v1/memories/:id/bind` | 绑定到小说 |
| 工作流 | `POST /api/v1/ai/workflows/submit` | 提交工作流 |
| 钱包 | `GET /api/v1/wallet` | 余额查询 |
| WebSocket | `GET /ws` | 实时推送 |

## 架构设计

打开 `architecture.html` 查看完整的可视化架构文档（支持暗色/亮色主题切换），涵盖：

1. 系统架构总览（三层架构图）
2. 前端架构（技术栈 + 15 个 Store 模块）
3. 页面路由布局
4. 后端分层架构（Handler/Service/DAO/Model/Agent）
5. AI Agent 体系（4 Provider + Dispatcher + Executor）
6. 编排引擎（DAG 工作流 + SharedState）
7. 智能联想系统（Suggestion + Intent + UserBehavior）
8. 记忆与上下文（三层记忆 + RAG 语义检索）
9. 小说工坊（功能矩阵 + 页面布局）
10. 数据模型（实体关系 + 状态机）
11. 降级与容错（两层 Fallback + IsRetryableError）
12. 工具与 Function Calling
13. 设计系统（色彩 + 组件库 + SCSS 变量）

## License

MIT
