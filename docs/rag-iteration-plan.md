# Ai-Curton RAG 功能分析与迭代方案

> 文档版本：v1.0 | 日期：2026-04-07

---

## 一、现状诊断：当前上下文机制的局限性

经过完整代码审查，项目当前**没有任何 RAG 实现**。AI 生成的上下文来源仅有两条路径：

| 上下文来源 | 实现位置 | 局限性 |
|---|---|---|
| 前 5 章概要 + 前 1 章正文末尾 2000 字 | `service/novel.go:315, 529-568` | 长篇小说（50+ 章）时，LLM 对早期人物、伏笔、世界观完全失忆 |
| 番茄小说爬虫获取的简介/设定 | `service/crawler.go` | 仅用于创建阶段的参考，不参与后续章节生成的上下文 |
| Prompt 模板静态注入 | `model/prompt_template.go` | 模板数据结构 `PromptTemplateData` 中无知识库字段，无法注入检索结果 |

### 核心问题

**上下文窗口是"近视"的** — 只能看到最近 5 章，对全局设定、人物关系、前期伏笔完全无感知。

当前上下文构建流程：

```
ChapterAIAction()
  ├─ GetPreviousChapters(novelID, sortOrder, 5)   // 仅取前 5 章
  ├─ buildTemplateData()
  │     ├─ 前文概要：各章 Summary 拼接
  │     └─ 前一章正文：末尾截取 2000 字
  ├─ tplSvc.RenderPrompt()                         // 模板渲染
  └─ buildChapterContext() → History JSON           // 传递给 Dispatcher
```

---

## 二、产品形态分析：为什么 Ai-Curton 需要 RAG

Ai-Curton 是一个 AI 小说创作工坊，核心操作链路：

```
大纲生成 → 章节创作 → 润色/扩写/续写 → 版本管理
```

小说创作的本质需求是**长程一致性**：

- **人物一致性**：性格、外貌、关系在 100 章后不能矛盾
- **世界观一致性**：魔法体系、地理、历史、社会制度需全局统一
- **伏笔连贯性**：伏笔需要在几十章后被正确回收
- **文风统一性**：叙事节奏、语言风格需保持一致

当前的"前 5 章滑动窗口"方案在短篇（< 10 章）尚可，但对中长篇完全不够用。

---

## 三、RAG 迭代方案

### Phase 1：结构化知识库（无向量数据库）

> 预估周期：2-3 周 | 基础设施依赖：无（仅 MySQL）

#### 1.1 核心思路

先不引入向量数据库，利用已有的 MySQL + 结构化数据解决 80% 的问题。

#### 1.2 新增数据模型

```go
// server/internal/model/knowledge.go

// NovelKnowledge 小说知识条目
type NovelKnowledge struct {
    ID         uint      `gorm:"primaryKey" json:"id"`
    NovelID    uint      `gorm:"not null;index" json:"novel_id"`
    Category   string    `gorm:"size:30;not null;index" json:"category"`
    // category 枚举: character | worldview | plotline | foreshadow | style | custom
    Title      string    `gorm:"size:200;not null" json:"title"`
    Content    string    `gorm:"type:text;not null" json:"content"`
    Tags       string    `gorm:"size:500" json:"tags"`           // 逗号分隔的标签
    ChapterRef string    `gorm:"size:200" json:"chapter_ref"`    // 关联章节 ID 列表
    Priority   int       `gorm:"default:0" json:"priority"`      // 权重，越高越优先注入
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

#### 1.3 知识条目类型设计

| Category | 用途 | 示例内容 |
|---|---|---|
| `character` | 人物档案 | 姓名、外貌、性格、能力、关系网 |
| `worldview` | 世界观设定 | 魔法体系、地理、历史、社会制度 |
| `plotline` | 主线/支线剧情 | 剧情走向、关键事件时间线 |
| `foreshadow` | 伏笔追踪 | 伏笔内容、埋设章节、预期回收章节 |
| `style` | 文风规范 | 叙事视角、语言风格、禁用词 |
| `custom` | 自定义 | 用户自由添加的任何设定 |

#### 1.4 上下文注入改造

**扩展 `PromptTemplateData`**：

```go
// model/prompt_template.go

type PromptTemplateData struct {
    // --- 现有字段保持不变 ---
    NovelTitle       string
    NovelDescription string
    ChapterTitle     string
    ChapterSummary   string
    ChapterContent   string
    PrevSummaries    string
    PrevContent      string
    WordCount        int
    TargetWords      int
    SelectedText     string

    // --- 新增：知识库上下文 ---
    KnowledgeContext string // 根据当前章节相关性筛选后的知识条目（已格式化）
    Characters       string // 本章涉及的人物档案
    WorldviewNotes   string // 相关世界观设定
}
```

**改造 `buildTemplateData`**（`service/novel.go`）：

在构建上下文时，从 `NovelKnowledge` 表按 `novel_id` + `category` + `priority` 查询，拼接注入。知识库注入总字数上限建议 3000 字，避免 Token 浪费。

#### 1.5 知识自动提取

在 `AcceptAIResult` 和 `UpdateChapter` 流程中，增加异步任务：

```
章节保存/AI 结果采纳
  └─ 异步 goroutine
       ├─ 调用 LLM 提取知识条目（人物、设定、伏笔）
       ├─ 与已有条目去重/合并
       └─ 写入 NovelKnowledge 表（status=pending，待用户审核）
```

#### 1.6 前端改造

在 `NovelWorkshop.vue` 侧边栏新增"知识库"面板：

- 按 category 分 Tab 展示（人物 | 世界观 | 剧情线 | 伏笔 | 文风 | 自定义）
- 支持手动 CRUD
- 支持"从章节提取"按钮触发 AI 自动提取
- 生成时可勾选哪些知识条目参与上下文
- 待审核条目高亮提示

#### 1.7 涉及文件清单

| 文件 | 改动内容 |
|---|---|
| `model/knowledge.go` | 新增 |
| `dao/knowledge.go` | 新增，CRUD + 按优先级查询 |
| `service/knowledge.go` | 新增，知识管理 + AI 提取 |
| `handler/knowledge.go` | 新增，REST API |
| `router/router.go` | 新增知识库路由组 |
| `model/prompt_template.go` | 扩展 `PromptTemplateData` |
| `service/novel.go` | 改造 `buildTemplateData`，注入知识上下文 |
| `web/src/api/knowledge.ts` | 新增 |
| `web/src/store/knowledge.ts` | 新增 |
| `web/src/views/novel/NovelWorkshop.vue` | 新增知识库侧边栏面板 |

#### 1.8 Phase 1 价值

零基础设施成本，解决人物一致性和世界观遗忘问题，用户可控。

---

### Phase 2：语义检索（引入向量数据库）

> 预估周期：3-4 周 | 基础设施依赖：向量数据库 + Embedding API

#### 2.1 核心思路

Phase 1 的结构化知识库依赖用户手动管理和关键词匹配。Phase 2 引入 Embedding + 向量检索，实现"写到哪里，自动找到相关设定"。

#### 2.2 技术选型

| 组件 | 推荐方案 | 理由 |
|---|---|---|
| 向量数据库 | Qdrant（首选）或 Milvus Lite | Qdrant 轻量、Go SDK 成熟、Docker 一键部署 |
| Embedding 模型 | 智谱 `embedding-3` 或通义 `text-embedding-v3` | 已有 Provider 适配层（`agent/provider/`），接入成本低 |
| 分块策略 | 按段落 + 滑动窗口（500 字，重叠 100 字） | 小说文本连续性强，需要重叠保证语义完整 |

#### 2.3 架构扩展

```
现有架构:
  Service → Dispatcher → Executor → AIProvider

扩展后:
  Service → KnowledgeRetriever → Dispatcher → Executor → AIProvider
                  ↓
            VectorStore (Qdrant)
                  ↓
            EmbeddingProvider (智谱/通义)
```

新增模块：

```
server/internal/rag/
├── retriever.go    // 检索器：接收查询文本，返回 Top-K 相关知识片段
├── indexer.go      // 索引器：章节保存时异步分块、Embedding、入库
├── chunker.go      // 分块器：按段落/滑动窗口/语义边界分块
├── embedding.go    // Embedding 适配层：对接智谱/通义 Embedding API
└── store.go        // 向量存储抽象层：屏蔽底层向量数据库差异
```

#### 2.4 混合检索策略

```go
// rag/retriever.go

// Retrieve 混合检索：结构化 + 语义
func (r *Retriever) Retrieve(ctx context.Context, novelID uint, query string, topK int) ([]Chunk, error) {
    // 1. 向量语义检索 — 找到语义最相关的片段
    semanticResults := r.vectorSearch(ctx, novelID, query, topK*2)

    // 2. 结构化知识库检索 — Phase 1 的知识条目（关键词匹配 + 优先级）
    knowledgeResults := r.knowledgeSearch(ctx, novelID, query)

    // 3. 融合排序 — RRF (Reciprocal Rank Fusion)
    merged := r.rrfMerge(semanticResults, knowledgeResults, topK)

    return merged, nil
}
```

#### 2.5 索引时机

| 事件 | 动作 |
|---|---|
| 章节保存 / AI 结果采纳 | 异步分块 → Embedding → 写入向量库 |
| 知识条目 CRUD | 同步更新向量索引 |
| 小说删除 | 级联清理向量数据（按 `novel_id` 过滤删除） |

#### 2.6 docker-compose 扩展

```yaml
# docker-compose.yml 新增
services:
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant_data:/qdrant/storage
    restart: unless-stopped

volumes:
  qdrant_data:
```

#### 2.7 Phase 2 价值

写作时自动召回相关设定，不再依赖用户手动勾选，大幅降低长篇创作的认知负担。

---

### Phase 3：智能上下文编排

> 预估周期：4-6 周 | 依赖：Phase 2 完成

#### 3.1 核心思路

Phase 2 解决了"找到相关内容"，Phase 3 解决"如何在有限 Token 预算内最优编排上下文"。

#### 3.2 Token 预算管理器

```go
// rag/budget.go

type ContextBudget struct {
    TotalTokens     int // 模型上下文窗口（如 128K）
    SystemPrompt    int // 系统提示词预留
    OutputReserve   int // 输出预留
    UserInput       int // 用户输入/当前章节
    KnowledgeBudget int // 知识库上下文可用额度（动态计算）
}
```

根据不同操作类型动态分配 Token 预算：

| 操作类型 | 预算倾斜方向 |
|---|---|
| 续写 `continue` | 更多预算给前文上下文和伏笔 |
| 润色 `polish` | 更多预算给文风规范和当前章节 |
| 扩写 `expand` | 均衡分配，侧重人物和世界观 |
| 大纲生成 `outline` | 更多预算给全局剧情线和世界观 |

#### 3.3 上下文优先级排序

```
优先级从高到低：
  1. 当前章节内容（必须）
  2. 直接前一章末尾（必须）
  3. 本章涉及人物的档案（高优先）
  4. 语义检索 Top-K 结果（中优先）
  5. 前文概要（中优先）
  6. 世界观设定（按相关性）
  7. 伏笔追踪（按相关性）
  8. 文风规范（低优先，但始终包含摘要）
```

在 Token 预算内，按优先级从高到低填充，超出预算的低优先级内容自动截断或摘要化。

#### 3.4 与 DAG 工作流集成

在现有的 `orchestrator` 引擎中新增 `RAGNode` 节点类型：

```go
// agent/orchestrator/nodes/rag_node.go

type RAGNode struct {
    retriever *rag.Retriever
    budget    *ContextBudget
}

func (n *RAGNode) Execute(ctx context.Context, state *SharedState) error {
    query := state.GetString("current_context")
    novelID := state.Get("novel_id").(uint)

    // 检索相关知识片段
    chunks, err := n.retriever.Retrieve(ctx, novelID, query, 20)
    if err != nil {
        return err
    }

    // 按预算编排上下文
    compiled := n.budget.Compile(chunks)

    state.Set("rag_context", compiled)
    return nil
}
```

这样在 DAG 工作流中，RAG 检索可以作为独立节点，与其他节点并行执行，充分利用现有编排引擎的分层并行能力。

#### 3.5 反馈闭环

| 用户行为 | 系统响应 |
|---|---|
| 采纳 AI 结果 | 正反馈，提升相关知识条目权重 |
| 拒绝 / 大幅修改 | 负反馈，降低权重或标记需要更新 |
| 手动修正知识条目 | 更新向量索引，修正后续检索 |
| 定期自动检查 | LLM 对知识库做一致性检查（检测矛盾条目） |

#### 3.6 Phase 3 价值

从"能用"到"好用"，让 AI 在长篇创作中表现接近人类编辑的上下文理解能力。

---

## 四、实施优先级总览

```
Phase 1（结构化知识库）
  ├─ 投入：2-3 周，无新基础设施
  ├─ 收益：★★★★☆ 解决 80% 的一致性问题
  ├─ 风险：低
  └─ 建议：立即启动

Phase 2（语义检索）
  ├─ 投入：3-4 周，需引入向量数据库 + Embedding API
  ├─ 收益：★★★★★ 质的飞跃，自动化程度大幅提升
  ├─ 风险：中（Embedding 质量需评估）
  └─ 建议：Phase 1 上线验证后启动

Phase 3（智能编排）
  ├─ 投入：4-6 周，需要较多调优
  ├─ 收益：★★★☆☆ 锦上添花，长篇场景体验显著提升
  ├─ 风险：中（Token 预算策略需反复调优）
  └─ 建议：Phase 2 稳定后按需启动
```

---

## 五、关键风险与应对

| 风险 | 影响 | 应对策略 |
|---|---|---|
| **Token 成本增加** | RAG 注入上下文增加每次 API 调用的 Token 消耗 | Phase 1 设置知识库注入字数上限（3000 字）；Phase 3 引入 Token 预算管理器 |
| **Embedding 延迟** | 章节保存时索引阻塞用户操作 | 必须走异步队列，复用现有 Dispatcher 异步机制 |
| **知识库膨胀** | 长篇小说产生大量知识条目 | 设计归档/合并机制；按 category 设置条目数上限 |
| **检索准确性** | 小说文本语义检索比技术文档更难 | 针对文学文本做 Embedding 模型评估（智谱 vs 通义 vs BGE）；引入 Reranker 二次排序 |
| **知识冲突** | 自动提取的条目与用户手动维护的条目矛盾 | 自动提取条目默认 `pending` 状态，需用户审核确认 |

---

## 六、技术架构演进图

```
                        Phase 1                    Phase 2                    Phase 3
                    ┌──────────────┐          ┌──────────────┐          ┌──────────────┐
                    │  结构化知识库  │          │  语义检索     │          │  智能编排     │
                    │              │          │              │          │              │
  上下文来源        │ MySQL 知识表  │    +     │ 向量数据库    │    +     │ Token 预算    │
                    │ 手动+AI提取   │          │ Embedding    │          │ 优先级排序    │
                    │ 关键词匹配    │          │ 混合检索      │          │ DAG 集成     │
                    │              │          │              │          │ 反馈闭环      │
                    └──────┬───────┘          └──────┬───────┘          └──────┬───────┘
                           │                         │                         │
  现有架构集成点    service/novel.go          rag/retriever.go         orchestrator/nodes/
                    buildTemplateData()       Retrieve()               RAGNode.Execute()
                           │                         │                         │
                           ▼                         ▼                         ▼
                    PromptTemplateData        SharedState                SharedState
                    + KnowledgeContext        + rag_context             + compiled_context
```

---

*文档生成时间：2026-04-07*
