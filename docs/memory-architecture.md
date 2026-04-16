# 记忆模块应用层架构

## 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (Vue 3)                         │
│  ┌──────────┐  ┌──────────────┐  ┌──────────┐  ┌────────────┐  │
│  │MyMemories│  │MemoryMarket  │  │MemDetail │  │ WalletPage │  │
│  └────┬─────┘  └──────┬───────┘  └────┬─────┘  └─────┬──────┘  │
│       │               │               │              │          │
│  ┌────┴─────┐  ┌──────┴───────┐  ┌────┴─────┐  ┌────┴──────┐  │
│  │memoryStore│  │ marketStore  │  │marketStore│  │walletStore│  │
│  └────┬─────┘  └──────┬───────┘  └────┬─────┘  └─────┬─────┘  │
│       │               │               │              │          │
│  ┌────┴─────┐  ┌──────┴───────┐              ┌───────┴──────┐  │
│  │memory API│  │ market API   │              │ wallet API   │  │
│  └──────────┘  └──────────────┘              └──────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │ HTTP
┌─────────────────────────────────────────────────────────────────┐
│                      Backend (Go + Gin)                         │
│                                                                 │
│  ┌─────────────────── Router Layer ──────────────────────────┐  │
│  │ /memories/*    /market/memories/*    /wallet/*             │  │
│  └───────┬──────────────┬──────────────────┬─────────────────┘  │
│          │              │                  │                    │
│  ┌───────┴──────┐ ┌─────┴──────┐ ┌────────┴───────┐           │
│  │MemoryHandler │ │MarketHandler│ │ WalletHandler  │           │
│  └───────┬──────┘ └─────┬──────┘ └────────┬───────┘           │
│          │              │                  │                    │
│  ┌───────┴──────┐ ┌─────┴──────┐ ┌────────┴───────┐           │
│  │MemoryService │ │MarketService│ │ WalletService  │           │
│  └───┬──────┬───┘ └──┬─────┬───┘ └────────┬───────┘           │
│      │      │        │     │               │                    │
│  ┌───┴──┐ ┌─┴────┐ ┌┴───┐ │         ┌─────┴─────┐            │
│  │Memory│ │Agent  │ │Mkt │ │         │ WalletDAO │            │
│  │ DAO  │ │Disp.  │ │DAO │ │         └───────────┘            │
│  └──────┘ └──┬───┘ └────┘ │                                    │
│              │             │                                    │
│  ┌───────────┴─────────────┴──────────────────────┐            │
│  │          Orchestrator (DAG Engine)              │            │
│  │  ┌──────────────┐  ┌───────────────────┐       │            │
│  │  │MemoryExtract │  │  MemoryReview     │       │            │
│  │  │   Graph      │  │    Graph          │       │            │
│  │  └──────────────┘  └───────────────────┘       │            │
│  └────────────────────────────────────────────────┘            │
│                                                                 │
│  ┌──────────────────── Model Layer ──────────────────────────┐  │
│  │ WritingMemory │ MemoryVersion │ MemoryEmbedding           │  │
│  │ NovelBinding  │ MemoryOrder   │ MemoryLicense             │  │
│  │ MemoryReview  │ UserWallet    │ WalletTransaction         │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                  │
│                         MySQL / GORM                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## 各模块详解

### 1. Model 层（数据模型）

| 模型 | 文件 | 作用 |
|------|------|------|
| `WritingMemory` | `model/writing_memory.go` | 记忆主表，存储提取的写作特征、Prompt 模板、锚定句、预览文本、质量评分等 |
| `WritingMemoryVersion` | `model/writing_memory.go` | 记忆版本历史，每次追加样本重新提取时保存快照，支持回溯 |
| `MemoryEmbedding` | `model/writing_memory.go` | 记忆的向量化分块，将样本文本按 500 字切片后存储 Embedding 向量，用于语义检索 |
| `NovelMemoryBinding` | `model/writing_memory.go` | 小说-记忆绑定关系，一部小说可绑定多个类别的记忆（风格/人设/世界观/剧情偏好） |
| `MemoryOrder` | `model/market.go` | 交易订单，记录买家、卖家、金额、平台抽成、卖家收入 |
| `MemoryLicense` | `model/market.go` | 购买授权，标记用户对某个记忆的使用权 |
| `MemoryReview` | `model/market.go` | 用户评价，1-5 星评分 + 文字评论 |
| `UserWallet` | `model/wallet.go` | 用户钱包，余额/冻结/累计收入/累计支出，带乐观锁 version 字段 |
| `WalletTransaction` | `model/wallet.go` | 钱包流水，记录每笔充值/购买/收入/退款的金额变动 |

### 2. DAO 层（数据访问）

| DAO | 文件 | 作用 |
|-----|------|------|
| `WritingMemoryDAO` | `dao/writing_memory.go` | 记忆 CRUD、按用户/类别/状态查询、样本哈希去重、Embedding 批量操作、小说绑定管理、市场上架列表（含关键词搜索和排序）、可用记忆查询（自己的 + 已购买的） |
| `MarketDAO` | `dao/market.go` | 订单创建/查询、授权创建/检查、评价 CRUD、平均评分计算 |
| `WalletDAO` | `dao/wallet.go` | 钱包获取或创建、乐观锁扣减/增加积分、充值、流水记录 |

### 3. Service 层（业务逻辑）

#### MemoryService（`service/memory.go`）

核心业务服务，负责记忆的完整生命周期：

- **创建记忆**：校验类别、样本长度（≥200 字）、SHA256 去重，创建草稿状态记忆
- **提取结果更新**：接收 DAG 工作流回调，保存特征 JSON、Prompt 模板、锚定句、预览文本、质量评分，同时创建版本快照
- **追加样本**：将新文本追加到已有样本，重新计算哈希，版本号 +1
- **上架申请**：校验已完成提取，设置价格，状态变更为 `reviewing`
- **Embedding 生成**：将样本按 500 字分块，调用 AI Provider 的 Embedding 接口生成向量，存入 `MemoryEmbedding` 表
- **语义检索**：对查询文本生成 Embedding，与记忆的所有分块向量计算余弦相似度，返回 Top-K 最相关片段
- **小说绑定**：管理小说与记忆的绑定关系（一个类别只能绑一个记忆）

#### MarketService（`service/market.go`）

交易市场服务，处理购买和评价：

- **市场浏览**：按类别/关键词/排序方式分页查询已上架记忆
- **版权隔离**：创建者看全部字段，购买者看 features + prompt_tpl，未购买者只看基础信息和预览
- **购买流程**（单事务）：
  1. 校验记忆状态、不能买自己的、不能重复购买
  2. 乐观锁扣减买家积分
  3. 增加卖家积分（扣除 10% 平台抽成）
  4. 创建订单 + 授权 + 双方流水
  5. 更新销量
- **评价**：校验已购买、未重复评价，提交后自动更新记忆的平均评分

#### WalletService（`service/wallet.go`）

钱包服务：

- **获取钱包**：自动创建（首次访问时）
- **充值**：管理员操作，事务内增加余额 + 创建流水
- **流水查询**：分页返回用户的交易记录

### 4. Handler 层（API 接口）

| Handler | 路由前缀 | 接口 |
|---------|----------|------|
| `MemoryHandler` | `/api/v1/memories` | CRUD、追加样本、上架/下架、生成预览、可用记忆列表 |
| `MemoryHandler` | `/api/v1/novels/:id/memory-bindings` | 小说-记忆绑定的查询和设置 |
| `MarketHandler` | `/api/v1/market/memories` | 市场浏览、详情（版权隔离）、购买、评价 |
| `WalletHandler` | `/api/v1/wallet` | 钱包信息、流水列表、充值 |

### 5. Orchestrator 层（DAG 工作流）

#### 记忆提取工作流（`BuildMemoryExtractGraph`）

```
Layer 0: feature_extract（特征提取）
    ↓
Layer 1: prompt_compile（Prompt 编译）
    ↓
Layer 2: quality_eval（质量评估）
```

- **feature_extract**：根据记忆类别（style/character/worldview/plot_preference）使用对应的 Prompt 模板，从样本中提取结构化 JSON 特征
- **prompt_compile**：将特征 JSON 编译为可执行的写作指令，并从样本中提取 3-5 个锚定句作为 few-shot 示例
- **quality_eval**：使用编译后的 Prompt 生成 100 字样本，自评与原始样本的风格一致性（0-100 分）

#### 记忆审核工作流（`BuildMemoryReviewGraph`）

```
Layer 0: quality_check ─┐
                        ├→ Layer 1: review_decision
Layer 0: compliance_check┘
```

- **quality_check**：生成样本并打分，检查质量问题
- **compliance_check**：检查内容合规性（无违规内容）
- **review_decision**：综合两项检查结果，输出 approved/rejected 决策

### 6. AI Provider 层（Embedding 扩展）

在 `AIProvider` 接口新增 `Embedding` 方法：

| Provider | Embedding 支持 | 模型 |
|----------|---------------|------|
| 智谱 (ZhipuProvider) | ✅ | `embedding-3` |
| 通义 (QwenProvider) | ✅ | `text-embedding-v3` |
| DeepSeek | ❌ | - |
| Kimi | ❌ | - |
| Mock | ✅ (随机向量) | - |

### 7. 写作风格注入（`WritingStyleService`）

`FormatStyleForPrompt` 方法在生成写作规范时，自动查询小说绑定的记忆，将记忆的 Prompt 模板和锚定句注入到 AI 生成的上下文中：

```
【全局风格】叙事视角、文风调性、语言风格...
【写作风格记忆·我的风格】Prompt 模板内容...
【风格参考句】1. 锚定句1  2. 锚定句2 ...
【人设模板记忆·角色库】Prompt 模板内容...
【场景预设】...
```

### 8. Frontend 层

| 页面 | 文件 | 作用 |
|------|------|------|
| 我的记忆 | `views/memory/MyMemories.vue` | 记忆列表、类别筛选、上架/下架/删除、详情查看 |
| 提取对话框 | `views/memory/MemoryExtractDialog.vue` | 选择类别、输入样本文本、创建记忆并触发提取 |
| 记忆市场 | `views/market/MemoryMarket.vue` | 市场浏览、搜索、排序、分页 |
| 记忆详情 | `views/market/MemoryDetail.vue` | 版权隔离展示、购买、评价 |
| 钱包 | `views/wallet/WalletPage.vue` | 余额概览、交易流水 |

| Store | 文件 | 作用 |
|-------|------|------|
| `useMemoryStore` | `store/memory.ts` | 记忆 CRUD 状态管理、小说绑定、可用记忆列表 |
| `useMarketStore` | `store/market.ts` | 市场列表、详情、购买、评价状态管理 |
| `useWalletStore` | `store/wallet.ts` | 钱包余额、流水列表状态管理 |

---

## 数据流示意

### 记忆提取流程

```
用户粘贴样本 → MemoryExtractDialog → memoryApi.create()
    → MemoryHandler.Create → MemoryService.CreateMemory
        → WritingMemoryDAO.Create（状态: draft）
        → [异步] WorkflowService.Submit（type: memory_extract）
            → Orchestrator.BuildMemoryExtractGraph
                → feature_extract → prompt_compile → quality_eval
            → MemoryService.UpdateExtractResult（保存特征+Prompt+评分）
```

### 购买流程

```
用户点击购买 → marketApi.buy()
    → MarketHandler.Buy → MarketService.PurchaseMemory
        → 事务开始
            → WalletDAO.Deduct（买家扣款，乐观锁）
            → WalletDAO.Credit（卖家收款，扣除10%平台费）
            → MarketDAO.CreateOrder
            → MarketDAO.CreateLicense
            → WalletDAO.CreateTransaction × 2（双方流水）
            → WritingMemoryDAO.IncrSalesCount
        → 事务提交
```

### 记忆注入创作流程

```
AI 生成章节 → WritingStyleService.FormatStyleForPrompt(novelID)
    → WritingMemoryDAO.ListBindingsByNovel(novelID)
    → 遍历绑定的记忆，注入 PromptTpl + AnchorTexts
    → 拼接到写作规范中，传给 AI Provider
```
