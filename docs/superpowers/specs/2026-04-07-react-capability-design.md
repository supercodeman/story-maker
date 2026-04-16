# P0 ReAct 能力技术方案设计

> 日期：2026-04-07
> 状态：设计评审
> 前置文档：`docs/multi-agent-architecture-analysis.md`

---

## 一、问题本质分析（第一性原理）

当前 Orchestrator 的 DAG 节点是"单次调用"模型：接收 prompt → 调用 LLM → 返回结果。节点无法在执行过程中主动获取额外信息、验证中间结果、或根据观察调整策略。

这导致两个核心问题：

1. **信息不足**：节点只能依赖 SharedState 中上游写入的数据，无法主动查询项目内部数据（角色库、世界观、前章摘要等）
2. **质量不可控**：节点无法对自己的输出进行自我评估和修正，一次生成定终局

ReAct（Reasoning + Acting）模式的本质是让节点具备"思考 → 行动 → 观察 → 再思考"的闭环能力，从被动执行者变为主动推理者。

---

## 二、架构选型

### 选定方案：ReactExecutor 作为 Engine 内部组件

在 Orchestrator Engine 内部新增 ReactExecutor，作为 ReAct 节点的专用执行路径。非 ReAct 节点完全走原有路径，零侵入。

**备选方案对比：**

| 方案 | 优势 | 劣势 | 结论 |
|------|------|------|------|
| ReactExecutor 内嵌 Engine | 改动最小，Engine 仍是唯一编排入口 | 与 Engine 耦合较紧 | **选定** |
| ReactExecutor 作为独立 TaskExecutor | 解耦彻底，复用注册机制 | 需要 Dispatcher 层也支持，改动面大 | P1 考虑 |
| Node 自带 ReactLoop | 节点自治，最灵活 | 破坏 Node 纯数据结构设计 | 不采用 |

**选型理由：**
- P0 阶段优先最小改动、最低风险
- ReAct 逻辑集中在 Engine 内部，不扩散到 Dispatcher 层
- 后续 P1 做独立 ReAct Agent 时再考虑更大的架构调整

---

## 三、功能模块设计

### 3.1 Node 结构扩展

**文件**：`server/internal/agent/orchestrator/graph.go`

在现有 `Node` 结构体上新增三个字段：

```go
// Node DAG 节点，对应一个 AI 子任务
type Node struct {
    ID             string            // 节点唯一标识
    TaskType       string            // 对应 model.TaskType* 常量
    ModelName      string            // 使用的模型名
    FallbackModels []string          // 降级备选模型列表
    Prompt         string            // 提示词模板，支持 {{.key}} 从 SharedState 注入
    OutputKey      string            // 执行结果写入 SharedState 的 key
    InputMap       map[string]string // 从 SharedState 读取的 key 映射
    DependsOn      []string          // 依赖的节点 ID 列表

    // ---- 新增 ReAct 相关字段 ----
    ReactEnabled bool          // 是否启用 ReAct 模式，默认 false
    Tools        []string      // 该节点可用的工具名列表（从 ToolRegistry 筛选）
    ReactConfig  *ReactConfig  // ReAct 熔断配置，nil 则用全局默认值
}
```

**设计原则**：
- `ReactEnabled=false` 时，所有新增字段被忽略，现有行为完全不变
- `Tools` 是白名单机制，节点只能调用显式声明的工具，防止越权
- `ReactConfig` 为 nil 时使用全局默认配置，减少模板工厂的配置负担

### 3.2 ReactConfig 熔断配置

**文件**：`server/internal/agent/orchestrator/react.go`（新增）

```go
// ReactConfig ReAct 循环的多维度熔断配置
type ReactConfig struct {
    MaxRounds  int           // 最大推理轮次，默认 5
    MaxTimeout time.Duration // 单节点最大执行时间，默认 120s
    MaxTokens  int           // 累计 token 消耗上限，默认 16384
}

// DefaultReactConfig 全局默认配置
var DefaultReactConfig = &ReactConfig{
    MaxRounds:  5,
    MaxTimeout: 120 * time.Second,
    MaxTokens:  16384,
}
```

**熔断策略**：
- 三个维度任一触发即进入"强制收敛"模式
- 强制收敛不是直接失败，而是追加一条 system 消息要求 LLM 立即输出最终答案，再调一次 LLM
- 如果强制收敛后 LLM 仍返回 ToolCall，则报错终止

### 3.3 ReactExecutor 核心循环

**文件**：`server/internal/agent/orchestrator/react.go`（新增）

#### 3.3.1 结构定义

```go
// ToolProvider 工具提供接口（由 Dispatcher 层注入）
type ToolProvider interface {
    GetTool(name string) (ToolExecuteFunc, bool)
    GetToolDefs(names []string) []map[string]any
}

// ToolExecuteFunc 工具执行函数签名
type ToolExecuteFunc func(ctx context.Context, args map[string]any) (string, error)

// ReactExecutor ReAct 推理循环执行器
type ReactExecutor struct {
    nodeExecutor NodeExecutor   // 复用现有的 LLM 调用回调
    toolProvider ToolProvider   // 工具提供者
    callback     ProgressCallback
}
```

#### 3.3.2 核心执行流程

```
Execute(ctx, node, state):
  1. 获取 ReactConfig（node.ReactConfig ?? DefaultReactConfig）
  2. 构造 ReAct system prompt：
     - 注入节点原始 Prompt（已经过模板渲染）
     - 注入可用工具描述（从 ToolProvider 获取）
     - 注入输出格式约束（要求 LLM 在完成推理后输出 FINAL_ANSWER 标记）
  3. 初始化：history=[], tokenCount=0, startTime=now()
  4. 循环（round = 0; round < config.MaxRounds; round++）：
     a. 检查时间熔断：time.Since(startTime) > config.MaxTimeout
     b. 检查 token 熔断：tokenCount > config.MaxTokens
     c. 如果任一熔断触发 → 进入强制收敛（见 3.3.3）
     d. 调用 LLM（通过 nodeExecutor）
     e. 累加 tokenCount
     f. 解析返回：
        - 有 ToolCalls → 逐个执行工具 → Observation 追加到 history → callback 通知
        - 无 ToolCalls（纯文本）→ 视为 FinalAnswer → 跳出循环
     g. callback 通知当前轮次状态（thought + action）
  5. 返回最终结果
```

#### 3.3.3 强制收敛机制

```go
// forceConverge 熔断触发时，强制要求 LLM 输出最终答案
func (r *ReactExecutor) forceConverge(ctx context.Context, node *Node,
    history []ChatMessage, state *SharedState) (interface{}, error) {

    convergeMsg := ChatMessage{
        Role:    "system",
        Content: "你已达到推理轮次/时间/token上限。请立即基于已有信息给出最终答案，不要再调用任何工具。",
    }
    history = append(history, convergeMsg)

    // 最后一次 LLM 调用，不传 tools 定义，阻止工具调用
    result, err := r.nodeExecutor(ctx, node, state)
    if err != nil {
        return nil, fmt.Errorf("force converge failed: %w", err)
    }
    return result, nil
}
```

#### 3.3.4 ReAct System Prompt 模板

```go
const reactSystemPromptTemplate = `你是一个具备工具调用能力的 AI 助手。

## 你的任务
{{.TaskPrompt}}

## 可用工具
你可以通过 function calling 调用以下工具来获取信息：
{{.ToolDescriptions}}

## 工作方式
1. 先思考：分析任务需要哪些信息
2. 再行动：调用工具获取所需信息
3. 观察结果：分析工具返回的数据
4. 重复以上步骤直到信息充足
5. 给出最终答案

## 重要约束
- 每次只调用必要的工具，避免冗余调用
- 当信息足够时，直接给出最终答案，不要继续调用工具
- 最终答案应直接回复文本内容，不要包含工具调用`
```

**设计决策**：采用 Function Calling 而非纯文本 Thought/Action 解析。理由：
- 现有 Provider 层（Zhipu/Qwen/Kimi）已支持 Function Calling
- JSON 结构化输出比正则解析 `Thought:` / `Action:` 更可靠
- 复用现有 `ToolCall` / `ChatMessage` 数据结构，无需新增解析逻辑

### 3.4 内部工具（InternalTools）

**文件**：`server/internal/agent/tools/internal.go`（新增）

为 ReAct 节点提供项目内部数据查询能力：

| 工具名 | 功能 | 入参 | 数据来源 |
|--------|------|------|---------|
| `get_characters` | 查询作品集下的角色列表及属性 | `portfolio_id` | Character DAO |
| `get_world_setting` | 获取作品集的世界观设定 | `portfolio_id` | Portfolio DAO |
| `get_chapter_summary` | 获取指定章节的摘要 | `chapter_id` | Chapter DAO |
| `get_outline` | 获取当前大纲结构 | `portfolio_id` | Outline DAO |
| `search_content` | 在作品内容中搜索关键词 | `portfolio_id`, `keyword` | Chapter DAO |

#### 工具注册方式

```go
// NewInternalTools 创建内部工具集
// 接收 DAO 接口而非具体实现，保持依赖倒置
func NewInternalTools(charDAO CharacterQuerier, portfolioDAO PortfolioQuerier,
    chapterDAO ChapterQuerier) []*agent.Tool {

    return []*agent.Tool{
        newGetCharactersTool(charDAO),
        newGetWorldSettingTool(portfolioDAO),
        newGetChapterSummaryTool(chapterDAO),
        newGetOutlineTool(portfolioDAO),
        newSearchContentTool(chapterDAO),
    }
}
```

**接口定义**（查询专用，只读）：

```go
// CharacterQuerier 角色查询接口（内部工具专用）
type CharacterQuerier interface {
    ListByPortfolio(ctx context.Context, portfolioID uint) ([]model.Character, error)
}

// PortfolioQuerier 作品集查询接口
type PortfolioQuerier interface {
    GetWorldSetting(ctx context.Context, portfolioID uint) (string, error)
    GetOutline(ctx context.Context, portfolioID uint) ([]model.OutlineChapter, error)
}

// ChapterQuerier 章节查询接口
type ChapterQuerier interface {
    GetSummary(ctx context.Context, chapterID uint) (string, error)
    SearchContent(ctx context.Context, portfolioID uint, keyword string) ([]model.SearchResult, error)
}
```

**设计原则**：
- 内部工具只做查询，不做写入，保证 ReAct 循环的副作用可控
- 通过接口隔离，工具层不直接依赖 DAO 实现
- 工具返回纯文本（JSON 格式化的字符串），与外部工具（如 weather）保持一致

### 3.5 Engine 适配

**文件**：`server/internal/agent/orchestrator/engine.go`

#### 3.5.1 Engine 结构扩展

```go
type Engine struct {
    executor      NodeExecutor
    callback      ProgressCallback
    reactExecutor *ReactExecutor    // 新增：ReAct 执行器
}

func NewEngine(executor NodeExecutor, callback ProgressCallback,
    toolProvider ToolProvider) *Engine {

    return &Engine{
        executor: executor,
        callback: callback,
        reactExecutor: &ReactExecutor{
            nodeExecutor: executor,
            toolProvider: toolProvider,
            callback:     callback,
        },
    }
}
```

#### 3.5.2 Run 方法适配

在现有节点执行逻辑中增加 ReAct 分支：

```go
// 在 g.Go 闭包内，渲染 Prompt 之后：

var result interface{}
var err error

if node.ReactEnabled {
    // ReAct 模式：多轮推理循环
    result, err = e.reactExecutor.Execute(gCtx, node, state)
} else {
    // 原有模式：单次 LLM 调用
    result, err = e.executor(gCtx, node, state)
}
```

**改动范围**：仅 `Run` 方法内增加一个 if 分支，不改变拓扑排序、条件边检查、Prompt 渲染等现有逻辑。

### 3.6 模板工厂适配

**文件**：`server/internal/agent/orchestrator/templates.go`

以 `BuildFullChapterGraph` 为例，将 `outline` 节点升级为 ReAct 模式：

```go
// Layer 0: 大纲生成（ReAct 模式，可查询角色和世界观）
g.AddNode(&Node{
    ID:             "outline",
    TaskType:       "text_gen",
    ModelName:      modelName,
    FallbackModels: fb,
    Prompt:         "请根据以下信息生成章节大纲：\n标题：{{.title}}\n背景：{{.background}}",
    OutputKey:      "outline_result",
    InputMap:       map[string]string{"title": "title", "background": "background"},
    ReactEnabled:   true,  // 启用 ReAct
    Tools:          []string{"get_characters", "get_world_setting"},  // 可用工具
    ReactConfig:    &ReactConfig{MaxRounds: 3, MaxTimeout: 90 * time.Second, MaxTokens: 8192},
})
```

其他节点（character_desc、scene_desc、dialogue、merge_polish）保持 `ReactEnabled=false`，行为不变。

---

## 四、数据流示意

### 4.1 ReAct 节点执行时序

```
Engine.Run
  │
  ├─ TopologicalSort → layers
  │
  ├─ Layer 0: [outline(ReAct)]
  │   │
  │   └─ reactExecutor.Execute(outline, state)
  │       │
  │       ├─ Round 1: LLM 思考 → 调用 get_characters(portfolio_id=1)
  │       │           → 返回角色列表 → Observation 追加到 history
  │       │
  │       ├─ Round 2: LLM 思考 → 调用 get_world_setting(portfolio_id=1)
  │       │           → 返回世界观设定 → Observation 追加到 history
  │       │
  │       ├─ Round 3: LLM 信息充足 → 输出大纲（FinalAnswer）
  │       │
  │       └─ state.Set("outline_result", 大纲内容)
  │
  ├─ Layer 1: [character_desc, scene_desc, dialogue] (并行，非 ReAct)
  │   └─ 原有逻辑不变
  │
  └─ Layer 2: [merge_polish] (非 ReAct)
      └─ 原有逻辑不变
```

### 4.2 熔断时序

```
reactExecutor.Execute
  │
  ├─ Round 1: LLM → ToolCall → execute → Observation  (tokenCount += 2000)
  ├─ Round 2: LLM → ToolCall → execute → Observation  (tokenCount += 3000)
  ├─ Round 3: LLM → ToolCall → execute → Observation  (tokenCount += 4000)
  ├─ Round 4: LLM → ToolCall → execute → Observation  (tokenCount += 4500)
  │
  ├─ 检查：tokenCount(13500) + 预估下轮(~3000) > MaxTokens(16384)
  │   → 触发 token 熔断
  │
  ├─ forceConverge: 追加 system 消息 → 调用 LLM（不传 tools）
  │   → LLM 输出最终答案
  │
  └─ 返回结果
```

---

## 五、WebSocket 推送扩展

ReAct 节点的每一轮推理都通过 ProgressCallback 推送到前端：

```go
// ReAct 节点的进度消息格式
type ReactProgress struct {
    WorkflowID uint   `json:"workflow_id"`
    NodeID     string `json:"node_id"`
    Round      int    `json:"round"`       // 当前轮次
    MaxRounds  int    `json:"max_rounds"`  // 最大轮次
    Phase      string `json:"phase"`       // "thinking" | "acting" | "observing" | "final"
    ToolName   string `json:"tool_name,omitempty"`   // Phase=acting 时的工具名
    Summary    string `json:"summary,omitempty"`     // 当前轮次摘要
}
```

前端可据此展示 ReAct 推理过程的实时进度（如"正在查询角色库..."、"正在分析世界观设定..."）。

---

## 六、新增文件清单

| 文件路径 | 职责 | 预估行数 |
|---------|------|---------|
| `server/internal/agent/orchestrator/react.go` | ReactExecutor + ReactConfig + 核心循环 | ~180 |
| `server/internal/agent/tools/internal.go` | 内部工具集（角色/世界观/章节/大纲/搜索） | ~200 |
| `server/internal/agent/tools/internal_test.go` | 内部工具单元测试 | ~120 |
| `server/internal/agent/orchestrator/react_test.go` | ReactExecutor 单元测试 | ~150 |

### 修改文件清单

| 文件路径 | 改动内容 | 改动量 |
|---------|---------|--------|
| `orchestrator/graph.go` | Node 新增 3 个字段 + ReactConfig 类型 | ~15 行 |
| `orchestrator/engine.go` | Engine 新增 reactExecutor 字段 + Run 方法增加 if 分支 | ~20 行 |
| `orchestrator/templates.go` | outline 节点启用 ReactEnabled + Tools | ~5 行 |

---

## 七、测试策略

### 7.1 单元测试

- **ReactExecutor 测试**：Mock NodeExecutor + Mock ToolProvider，验证：
  - 正常 ReAct 循环（2 轮工具调用 + 最终答案）
  - 轮次熔断触发强制收敛
  - 时间熔断触发强制收敛
  - Token 熔断触发强制收敛
  - 工具执行失败的错误处理
  - 强制收敛后 LLM 仍返回 ToolCall 的错误处理

- **内部工具测试**：Mock DAO 接口，验证各工具的入参解析和返回格式

### 7.2 集成测试

- 使用 MockProvider 构建包含 ReAct 节点的 DAG，验证：
  - ReAct 节点与非 ReAct 节点混合执行
  - ReAct 节点的结果正确写入 SharedState
  - 下游节点能正确读取 ReAct 节点的输出
  - 并行层中 ReAct 节点与普通节点并行执行

---

## 八、风险评估与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| ReAct 循环导致延迟增加 | 用户等待时间变长 | 多维度熔断 + WebSocket 实时进度推送 |
| LLM 不遵循 ReAct 格式 | 解析失败 | 采用 Function Calling 而非文本解析；解析失败视为 FinalAnswer |
| 内部工具查询慢 | 拖慢 ReAct 循环 | 工具执行加 context timeout（单次 10s） |
| Token 消耗增加 | 成本上升 | MaxTokens 熔断 + 节点级工具白名单限制调用范围 |
| 非 ReAct 节点受影响 | 回归风险 | ReactEnabled=false 走原有路径，零侵入；集成测试覆盖 |

---

## 九、后续演进（P1+）

本方案为 P0，聚焦"节点级 ReAct"。后续演进方向：

- **P1：独立 ReAct Agent** — 在 Orchestrator 之上新增 ReActAgent 类型，不走 DAG 编排，自主决定调用哪些 Executor/Tool
- **P2：Agent 间反馈循环** — 下游节点可评估上游结果，不满意时触发重新执行
- **P3：Checkpoint 持久化** — SharedState 支持快照存储，工作流中断后从断点恢复
