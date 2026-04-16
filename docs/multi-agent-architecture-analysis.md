# Ai-Curton 多 Agent 协作架构现状分析（对比 LangChain）

> 分析日期：2026-04-07

---

## 一、当前架构：四层自研 Agent 体系

项目采用纯 Go 自研的四层 Agent 架构：

```
Provider 层 → Executor 层 → Dispatcher 层 → Orchestrator 层
```

### 1.1 Provider 适配器层 — 统一多模型调用接口

- `AIProvider` 接口抽象了 Zhipu、Qwen、Kimi 三家模型
- 每个 Provider 实现 `GenerateText` / `GenerateImage` / `AdjustCharacter` 三个能力
- 模型间通过 `FallbackModels` 实现自动降级（主模型失败 → 按优先级尝试备选）

**核心代码**：`server/internal/agent/provider.go`

### 1.2 Executor 策略层 — 按任务类型分派执行策略

- `TaskExecutor` 接口 + 注册表模式，按 `TaskType` 路由到具体执行器
- 已实现：TextTaskExecutor、ImageTaskExecutor、CharacterTaskExecutor、OutlineTaskExecutor、ChapterTaskExecutor
- 每个 Executor 封装了特定任务的 prompt 构造、工具调用、结果解析逻辑

**核心代码**：`server/internal/agent/executor.go`、`executor_text.go`、`executor_image.go` 等

### 1.3 Dispatcher 分发层 — 任务生命周期管理

- 接收 `AITask`，解析参数，选择 Provider + Executor
- goroutine 异步执行，通过 `Notifier` 接口做 WebSocket 实时推送
- 支持 Function Calling（ToolRegistry → ToolCall → 多轮对话循环）

**核心代码**：`server/internal/agent/dispatcher.go`

### 1.4 Orchestrator 编排层 — DAG 工作流引擎

- `Graph`（DAG 图）+ `Engine`（执行引擎）+ `SharedState`（共享状态池）
- 拓扑排序分层，同层节点通过 `errgroup` 并行执行
- 节点间通过 `SharedState` 传递数据（OutputKey → InputMap 映射）
- 支持条件边（`EdgeConditional`），运行时动态跳过节点
- Prompt 模板渲染：`{{.key}}` 语法从 SharedState 注入上游结果

**核心代码**：`server/internal/agent/orchestrator/` 目录

典型工作流示例（`BuildFullChapterGraph`）：

```
Layer 0: outline（大纲生成）
Layer 1: character_desc | scene_desc | dialogue（并行）
Layer 2: merge_polish（整合润色）
```

---

## 二、对比 LangChain 的详细分析

| 维度 | Ai-Curton 现状 | LangChain (LangGraph) |
|------|---------------|----------------------|
| **编排模型** | 静态 DAG，编译时确定拓扑 | StateGraph 支持动态路由、循环、条件分支 |
| **Agent 自主性** | 节点是"被编排的任务"，不具备自主决策能力 | Agent 可自主选择工具、决定下一步行动（ReAct 模式） |
| **节点间通信** | SharedState 键值池，单向写入/读取 | State channels + reducers，支持复杂状态合并 |
| **工具调用** | 有 ToolRegistry + Function Calling 循环，但仅在单节点内 | 工具调用是 Agent 核心能力，支持多轮推理-行动循环 |
| **记忆/上下文** | TokenManager 做窗口截断，无持久化记忆 | 内置 Memory 模块（短期/长期），支持 checkpointing |
| **人机交互** | 无 human-in-the-loop 机制 | 原生支持 interrupt/approve/reject 断点 |
| **错误恢复** | 模型降级（FallbackModels），节点失败则整个工作流失败 | 支持节点级重试、状态回滚、从 checkpoint 恢复 |
| **动态性** | 条件边是唯一的动态能力，且条件函数编译时写死 | 运行时动态路由，Agent 可根据中间结果改变执行路径 |

---

## 三、核心差距分析

### 3.1 Agent 不是真正的 Agent

当前的"Agent"本质上是"被编排的任务节点"。每个节点接收固定 prompt，调用 LLM，返回结果。它不具备 LangChain Agent 的核心特征：**自主推理 → 选择行动 → 观察结果 → 决定下一步**。Function Calling 循环虽然在 Dispatcher 层实现了，但没有接入 Orchestrator 的 DAG 流程中。

### 3.2 编排是静态的，缺乏运行时适应性

`BuildFullChapterGraph` 这类模板在编译时就固定了拓扑结构。LangGraph 的 StateGraph 允许节点根据运行时状态动态决定下一个节点（conditional edges 是函数而非静态配置），甚至支持循环（Agent 反复调用工具直到满意）。

### 3.3 缺少 Agent 间的"对话"能力

当前节点间只能通过 SharedState 传递文本结果，没有 Agent 间的协商、反馈、修正机制。比如 `merge_polish` 节点如果发现角色描写和对话不一致，无法回退要求 `character_desc` 重新生成。

---

## 四、当前架构的优势

相比 LangChain，现有架构也有明显优势：

- **性能**：Go + goroutine + errgroup 的并发模型比 Python asyncio 快得多，适合高吞吐场景
- **可控性**：静态 DAG 的执行路径完全可预测，调试和监控更简单
- **轻量**：没有 LangChain 那套庞大的抽象层，代码直接、依赖少
- **降级机制**：FallbackModels 的多模型自动降级是实用的生产级特性，LangChain 需要自己实现

---

## 五、建议演进方向

按优先级排列：

### P0：让节点具备 ReAct 能力

在 Engine 层支持"节点内多轮推理"，让单个节点可以自主调用工具、观察结果、决定是否继续。

### P1：支持动态路由

条件边的 Condition 函数改为可以调用 LLM 判断，而非硬编码逻辑。

### P2：增加反馈循环

允许下游节点对上游结果进行评估，不满意时触发重新执行（DAG 中引入有限循环）。

### P3：Checkpoint 机制

SharedState 支持持久化快照，工作流中断后可从断点恢复。

---

## 六、总结

当前架构作为 MVP 是合理的选择——静态 DAG + 多模型降级已经能覆盖"大纲→并行生成→整合"这类确定性工作流。但如果后续要做更复杂的创作场景（比如 Agent 自主探索剧情分支、多 Agent 角色扮演对话），就需要往动态编排和 Agent 自主性方向演进。
