# Ai-Curton 向量数据库 RAG 方案设计

> Phase 2 技术方案 | 版本：v1.0 | 日期：2026-04-07
> 前置依赖：Phase 1 结构化知识库已完成

---

## 一、方案概述

在 Phase 1 结构化知识库的基础上，引入向量数据库和 Embedding 模型，实现语义级别的知识检索。核心目标：**写到哪里，自动找到相关设定**，不再依赖用户手动勾选知识条目。

### 1.1 核心能力

| 能力 | Phase 1（已实现） | Phase 2（本方案） |
|---|---|---|
| 知识存储 | MySQL 结构化表 | MySQL + 向量数据库 |
| 检索方式 | 关键词匹配 + 优先级排序 | 语义向量检索 + 关键词混合 |
| 上下文注入 | 手动管理 + 全量注入 | 自动检索 + 相关性排序 |
| 索引更新 | 手动 CRUD | 章节保存时自动异步索引 |

---

## 二、技术选型

### 2.1 向量数据库

| 方案 | 优势 | 劣势 | 推荐度 |
|---|---|---|---|
| **Qdrant** | Go SDK 成熟、轻量部署、过滤能力强、gRPC + REST 双协议 | 社区规模中等 | **首选** |
| Milvus Lite | 可嵌入进程、功能全面 | Go 嵌入模式不成熟、资源占用较大 | 备选 |
| Weaviate | GraphQL 接口、内置向量化 | 部署较重、Go SDK 偏弱 | 不推荐 |
| ChromaDB | Python 生态好 | 无官方 Go SDK | 不适用 |

**选定方案：Qdrant**

理由：
1. 官方 Go SDK（`github.com/qdrant/go-client`）质量高，gRPC 协议性能好
2. Docker 一键部署，资源占用低（单节点 < 200MB 内存）
3. 支持 Payload 过滤（按 `novel_id`、`category` 过滤），天然适配多小说隔离
4. 支持 HNSW 索引，百万级向量毫秒级检索

### 2.2 Embedding 模型

| 模型 | 维度 | 最大 Token | 中文效果 | 价格 | 推荐度 |
|---|---|---|---|---|---|
| **智谱 embedding-3** | 2048 | 8192 | 优秀 | 0.0005元/千token | **首选** |
| 通义 text-embedding-v3 | 1024 | 8192 | 优秀 | 0.0007元/千token | 备选 |
| BGE-M3 (本地) | 1024 | 8192 | 优秀 | 免费（需 GPU） | 自部署备选 |

**选定方案：智谱 embedding-3**

理由：
1. 项目已有智谱 Provider 适配层，接入成本最低
2. 2048 维度在中文文学文本上表现优异
3. 价格低廉，10 万字小说全量索引成本 < 0.5 元

### 2.3 文本分块策略

| 策略 | 适用场景 | 参数 |
|---|---|---|
| **滑动窗口** | 章节正文 | 窗口 500 字，步长 400 字（重叠 100 字） |
| 整条入库 | 知识条目（Phase 1） | 每条知识条目作为独立 chunk |
| 段落分割 | 长段落文本 | 按 `\n\n` 分段，超过 500 字的段落再滑动窗口 |

**重叠设计理由**：小说文本连续性强，100 字重叠确保跨窗口的语义不被截断（如对话跨段落）。

---

## 三、系统架构

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Service Layer                            │
│                                                              │
│  NovelService ──→ RAGRetriever ──→ PromptTemplateData       │
│       │                │                                     │
│       │          ┌─────┴──────┐                              │
│       │          │            │                              │
│       ▼          ▼            ▼                              │
│  Dispatcher   VectorStore  KnowledgeDAO                     │
│       │          │            │                              │
│       │          ▼            ▼                              │
│       │       Qdrant       MySQL                            │
│       │                                                      │
│       ▼                                                      │
│  RAGIndexer ──→ Chunker ──→ EmbeddingProvider ──→ Qdrant    │
│  (异步索引)                                                   │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 数据流

**写入流（索引）**：
```
章节保存/AI结果采纳
  └─ 异步 goroutine
       ├─ Chunker.Split(chapter.Content)  → []Chunk
       ├─ EmbeddingProvider.Embed(chunks) → [][]float32
       └─ VectorStore.Upsert(novelID, chunks, vectors)
```

**读取流（检索）**：
```
ChapterAIAction / buildTemplateData
  └─ RAGRetriever.Retrieve(novelID, query, topK)
       ├─ EmbeddingProvider.Embed(query)     → []float32
       ├─ VectorStore.Search(novelID, vector) → semanticResults
       ├─ KnowledgeDAO.SearchByTags(keyword)  → structuredResults
       └─ RRFMerge(semantic, structured)      → []Chunk (ranked)
```

---

## 四、模块设计

### 4.1 目录结构

```
server/internal/rag/
├── retriever.go      // 检索器：混合检索入口
├── indexer.go        // 索引器：异步分块+Embedding+入库
├── chunker.go        // 分块器：滑动窗口+段落分割
├── embedding.go      // Embedding 适配层
├── store.go          // 向量存储抽象接口
├── store_qdrant.go   // Qdrant 实现
└── merger.go         // RRF 融合排序
```

### 4.2 核心接口定义

```go
// rag/store.go

// Chunk 文本块
type Chunk struct {
    ID         string            // 唯一标识：{novel_id}_{chapter_id}_{chunk_idx}
    NovelID    uint              // 所属小说
    ChapterID  uint              // 所属章节（知识条目为 0）
    SourceType string            // "chapter" | "knowledge"
    Category   string            // 知识条目类别（chapter 类型为空）
    Title      string            // 章节标题或知识条目标题
    Content    string            // 文本内容
    Score      float32           // 检索得分（仅检索结果中有值）
    Metadata   map[string]string // 扩展元数据
}

// VectorStore 向量存储抽象接口
type VectorStore interface {
    // Init 初始化 Collection（幂等）
    Init(ctx context.Context) error
    // Upsert 写入或更新向量
    Upsert(ctx context.Context, chunks []Chunk, vectors [][]float32) error
    // Search 向量检索
    Search(ctx context.Context, novelID uint, vector []float32, topK int, filter map[string]string) ([]Chunk, error)
    // DeleteByChapter 删除章节的所有向量（章节更新时先删后插）
    DeleteByChapter(ctx context.Context, novelID, chapterID uint) error
    // DeleteByNovel 删除小说的所有向量（级联删除）
    DeleteByNovel(ctx context.Context, novelID uint) error
}

// EmbeddingProvider Embedding 模型抽象接口
type EmbeddingProvider interface {
    // Embed 批量文本向量化
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    // Dimension 返回向量维度
    Dimension() int
}

// Retriever 检索器
type Retriever struct {
    store        VectorStore
    embedding    EmbeddingProvider
    knowledgeDAO *dao.KnowledgeDAO
}
```

### 4.3 Qdrant Collection 设计

```go
// store_qdrant.go

// Collection 名称：ai_curton_chunks
// 向量维度：2048（智谱 embedding-3）
// 距离度量：Cosine

// Payload 字段（用于过滤）：
// - novel_id:    uint64   （必须，按小说隔离）
// - chapter_id:  uint64   （章节 ID，知识条目为 0）
// - source_type: string   （"chapter" | "knowledge"）
// - category:    string   （知识类别，章节为空）
// - title:       string   （标题）
// - content:     string   （原文，用于返回结果展示）

// 索引配置：
// - novel_id:    Payload Index（Integer，用于过滤）
// - source_type: Payload Index（Keyword，用于过滤）
// - category:    Payload Index（Keyword，用于过滤）
```

### 4.4 Chunker 分块器

```go
// chunker.go

type ChunkerConfig struct {
    WindowSize int // 滑动窗口大小（字符数），默认 500
    StepSize   int // 步长，默认 400（重叠 = WindowSize - StepSize）
    MinSize    int // 最小块大小，默认 50（过短的块丢弃）
}

type Chunker struct {
    config ChunkerConfig
}

// SplitChapter 对章节正文进行分块
func (c *Chunker) SplitChapter(novelID, chapterID uint, title, content string) []Chunk {
    // 1. 按段落分割（\n\n）
    // 2. 短段落合并，长段落滑动窗口
    // 3. 生成 Chunk ID：{novelID}_{chapterID}_{idx}
}

// SplitKnowledge 对知识条目进行分块（通常整条入库）
func (c *Chunker) SplitKnowledge(item *model.NovelKnowledge) []Chunk {
    // 知识条目通常较短，整条作为一个 Chunk
    // 超过 WindowSize 的条目才分块
}
```

### 4.5 混合检索与 RRF 融合

```go
// retriever.go

func (r *Retriever) Retrieve(ctx context.Context, novelID uint, query string, topK int) ([]Chunk, error) {
    // 1. 语义检索
    queryVec, err := r.embedding.Embed(ctx, []string{query})
    if err != nil {
        return nil, err
    }
    semanticResults, err := r.store.Search(ctx, novelID, queryVec[0], topK*2, nil)
    if err != nil {
        return nil, err
    }

    // 2. 结构化检索（复用 Phase 1 的 KnowledgeDAO）
    structuredResults := r.searchKnowledge(ctx, novelID, query)

    // 3. RRF 融合排序
    return r.rrfMerge(semanticResults, structuredResults, topK), nil
}

// merger.go

// RRF (Reciprocal Rank Fusion) 融合排序
// 公式：score = Σ 1/(k + rank_i)，k=60（标准值）
func rrfMerge(lists ...[]Chunk) []Chunk {
    scores := make(map[string]float64) // chunk ID → RRF score
    chunks := make(map[string]Chunk)
    k := 60.0

    for _, list := range lists {
        for rank, chunk := range list {
            scores[chunk.ID] += 1.0 / (k + float64(rank+1))
            chunks[chunk.ID] = chunk
        }
    }

    // 按 RRF score 降序排列
    // ...
}
```

### 4.6 异步索引器

```go
// indexer.go

type Indexer struct {
    store     VectorStore
    embedding EmbeddingProvider
    chunker   *Chunker
    queue     chan indexTask // 异步队列
}

type indexTask struct {
    NovelID   uint
    ChapterID uint
    Title     string
    Content   string
    TaskType  string // "chapter" | "knowledge"
}

// Start 启动索引 worker（建议 2-4 个 goroutine）
func (idx *Indexer) Start(workers int) {
    for i := 0; i < workers; i++ {
        go idx.worker()
    }
}

func (idx *Indexer) worker() {
    for task := range idx.queue {
        ctx := context.Background()
        // 1. 删除旧向量
        idx.store.DeleteByChapter(ctx, task.NovelID, task.ChapterID)
        // 2. 分块
        chunks := idx.chunker.SplitChapter(task.NovelID, task.ChapterID, task.Title, task.Content)
        // 3. Embedding
        texts := make([]string, len(chunks))
        for i, c := range chunks { texts[i] = c.Content }
        vectors, err := idx.embedding.Embed(ctx, texts)
        if err != nil { continue }
        // 4. 写入向量库
        idx.store.Upsert(ctx, chunks, vectors)
    }
}

// IndexChapter 提交章节索引任务（非阻塞）
func (idx *Indexer) IndexChapter(novelID, chapterID uint, title, content string) {
    idx.queue <- indexTask{
        NovelID: novelID, ChapterID: chapterID,
        Title: title, Content: content, TaskType: "chapter",
    }
}
```

---

## 五、集成改造点

### 5.1 NovelService 改造

```go
// service/novel.go 改造点

type NovelService struct {
    // ... 现有字段
    knowledgeSvc *KnowledgeService
    ragRetriever *rag.Retriever  // 新增
    ragIndexer   *rag.Indexer    // 新增
}

// buildTemplateData 改造：用 RAG 检索替代全量知识注入
func (s *NovelService) buildTemplateData(...) *model.PromptTemplateData {
    // ...

    // Phase 2：语义检索替代 Phase 1 的全量注入
    var knowledgeContext, characters, worldviewNotes string
    if s.ragRetriever != nil {
        // 用当前章节概要+标题作为查询
        query := chapter.Title + " " + chapter.Summary
        chunks, _ := s.ragRetriever.Retrieve(ctx, novel.ID, query, 10)
        knowledgeContext, characters, worldviewNotes = formatRAGChunks(chunks)
    } else if s.knowledgeSvc != nil {
        // fallback 到 Phase 1
        knowledgeContext, characters, worldviewNotes = s.knowledgeSvc.BuildKnowledgeContext(novel.ID, 3000)
    }

    // ...
}

// UpdateChapter / AcceptAIResult 改造：保存后触发异步索引
func (s *NovelService) UpdateChapter(...) {
    // ... 现有逻辑
    // 触发异步索引
    if s.ragIndexer != nil {
        s.ragIndexer.IndexChapter(chapter.NovelID, chapter.ID, chapter.Title, chapter.Content)
    }
}
```

### 5.2 Dispatcher 改造

在 `agent/dispatcher.go` 中，任务完成回调时触发索引：

```go
// 章节类任务完成后，触发 RAG 索引更新
if task.TaskType == "chapter_polish" || task.TaskType == "chapter_expand" || task.TaskType == "chapter_continue" {
    if indexer != nil {
        indexer.IndexChapter(...)
    }
}
```

### 5.3 docker-compose.yml 扩展

```yaml
services:
  # ... 现有服务

  qdrant:
    image: qdrant/qdrant:v1.12.1
    ports:
      - "6333:6333"   # REST API
      - "6334:6334"   # gRPC
    volumes:
      - qdrant_data:/qdrant/storage
    environment:
      - QDRANT__SERVICE__GRPC_PORT=6334
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 512M

volumes:
  qdrant_data:
```

### 5.4 go.mod 新增依赖

```
github.com/qdrant/go-client v1.12.1
google.golang.org/grpc v1.65.0
```

### 5.5 配置文件扩展

```yaml
# config.yaml
rag:
  enabled: true
  qdrant:
    host: "localhost"
    grpc_port: 6334
    collection: "ai_curton_chunks"
  embedding:
    provider: "zhipu"          # zhipu | qwen | local
    model: "embedding-3"
    dimension: 2048
    batch_size: 16             # 批量 Embedding 大小
  chunker:
    window_size: 500
    step_size: 400
    min_size: 50
  indexer:
    workers: 2                 # 异步索引 worker 数
    queue_size: 100            # 索引队列大小
  retriever:
    top_k: 10                  # 默认检索 Top-K
    rrf_k: 60                  # RRF 融合参数
```

---

## 六、Embedding 适配层

### 6.1 智谱 Embedding Provider

```go
// rag/embedding_zhipu.go

type ZhipuEmbeddingProvider struct {
    apiKey string
    model  string
    dim    int
}

func NewZhipuEmbeddingProvider(apiKey, model string) *ZhipuEmbeddingProvider {
    return &ZhipuEmbeddingProvider{
        apiKey: apiKey,
        model:  model, // "embedding-3"
        dim:    2048,
    }
}

func (p *ZhipuEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // POST https://open.bigmodel.cn/api/paas/v4/embeddings
    // Body: { "model": "embedding-3", "input": texts }
    // Response: { "data": [{ "embedding": [...], "index": 0 }] }
}

func (p *ZhipuEmbeddingProvider) Dimension() int { return p.dim }
```

### 6.2 通义 Embedding Provider（备选）

```go
// rag/embedding_qwen.go

type QwenEmbeddingProvider struct {
    apiKey string
    model  string
    dim    int
}

// POST https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding
// Body: { "model": "text-embedding-v3", "input": { "texts": [...] } }
```

---

## 七、性能与成本估算

### 7.1 索引性能

| 指标 | 估算值 |
|---|---|
| 单章节（3000 字）分块数 | ~7 个 chunk |
| 单章节 Embedding 耗时 | ~200ms（智谱 API） |
| 单章节索引总耗时 | ~300ms（含 Qdrant 写入） |
| 10 万字小说全量索引 | ~30 秒 |

### 7.2 检索性能

| 指标 | 估算值 |
|---|---|
| 单次向量检索（Qdrant） | < 5ms（万级向量） |
| 单次 Embedding（查询） | ~50ms |
| 混合检索总耗时 | ~100ms |

### 7.3 成本估算

| 场景 | Token 消耗 | 费用 |
|---|---|---|
| 10 万字小说全量索引 | ~15 万 token | ~0.075 元 |
| 单次检索查询 Embedding | ~100 token | ~0.00005 元 |
| 每日活跃写作（50 次检索） | ~5000 token | ~0.0025 元 |

### 7.4 存储估算

| 组件 | 10 万字小说 | 100 万字小说 |
|---|---|---|
| Qdrant 向量存储 | ~2MB | ~20MB |
| Qdrant 内存占用 | ~10MB | ~100MB |
| MySQL 知识条目 | ~100KB | ~1MB |

---

## 八、实施计划

### 8.1 阶段拆分

```
Week 1: 基础设施
  ├─ Qdrant Docker 部署 + 配置
  ├─ rag/ 模块骨架（接口定义）
  ├─ Embedding 适配层（智谱）
  └─ Chunker 实现 + 单元测试

Week 2: 索引管道
  ├─ VectorStore Qdrant 实现
  ├─ Indexer 异步索引器
  ├─ NovelService 集成（保存触发索引）
  └─ 全量索引迁移工具（已有章节）

Week 3: 检索管道
  ├─ Retriever 混合检索
  ├─ RRF 融合排序
  ├─ buildTemplateData 改造
  └─ 集成测试

Week 4: 优化与上线
  ├─ 检索质量评估（人工标注 + 自动评测）
  ├─ 性能调优（批量 Embedding、缓存）
  ├─ 前端检索结果可视化（可选）
  └─ 文档 + 部署
```

### 8.2 涉及文件清单

| 文件 | 操作 | 说明 |
|---|---|---|
| `server/internal/rag/` | 新增目录 | RAG 模块 |
| `server/internal/rag/store.go` | 新增 | 向量存储接口 |
| `server/internal/rag/store_qdrant.go` | 新增 | Qdrant 实现 |
| `server/internal/rag/embedding.go` | 新增 | Embedding 接口 |
| `server/internal/rag/embedding_zhipu.go` | 新增 | 智谱 Embedding |
| `server/internal/rag/chunker.go` | 新增 | 文本分块器 |
| `server/internal/rag/retriever.go` | 新增 | 混合检索器 |
| `server/internal/rag/merger.go` | 新增 | RRF 融合排序 |
| `server/internal/rag/indexer.go` | 新增 | 异步索引器 |
| `server/internal/service/novel.go` | 修改 | 集成 RAG 检索+索引 |
| `server/internal/router/router.go` | 修改 | 初始化 RAG 组件 |
| `server/config/config.go` | 修改 | 新增 RAG 配置 |
| `docker-compose.yml` | 修改 | 新增 Qdrant 服务 |
| `server/go.mod` | 修改 | 新增 Qdrant SDK 依赖 |

---

## 九、风险与应对

| 风险 | 影响 | 应对策略 |
|---|---|---|
| Qdrant 服务不可用 | 检索降级 | fallback 到 Phase 1 结构化检索，RAGRetriever 内置降级逻辑 |
| Embedding API 限流 | 索引延迟 | 批量请求 + 指数退避重试 + 本地队列缓冲 |
| 文学文本语义检索准确性 | 检索质量 | 引入 Reranker 二次排序（Phase 3）；人工评测调优 |
| 向量维度不匹配 | 数据迁移 | Collection 按维度命名，切换模型时新建 Collection + 全量重建 |
| 长篇小说向量膨胀 | 存储成本 | 按小说 ID 分 Collection（可选）；定期清理已删除章节的向量 |

---

## 十、与 Phase 3 的衔接

Phase 2 完成后，Phase 3（智能上下文编排）的主要扩展点：

1. **Token 预算管理器**：在 `Retriever.Retrieve` 返回结果后，按 Token 预算裁剪
2. **DAG 集成**：将 `RAGRetriever` 封装为 `RAGNode`，在 `orchestrator` 中并行执行
3. **反馈闭环**：用户采纳/拒绝 AI 结果时，调整知识条目权重和向量得分
4. **Reranker**：在 RRF 融合后增加 Cross-Encoder 重排序，提升 Top-K 精度

---

*文档生成时间：2026-04-07*
