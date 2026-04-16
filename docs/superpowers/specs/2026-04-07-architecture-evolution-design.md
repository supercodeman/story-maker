# Ai-Curton 多 Agent 架构演进完整技术方案

> 日期：2026-04-07
> 状态：设计评审
> 前置文档：`docs/multi-agent-architecture-analysis.md`、`docs/superpowers/specs/2026-04-07-react-capability-design.md`

---

## 一、演进总览

### 1.1 问题本质（第一性原理）

当前 Orchestrator 是一个**静态 DAG 执行引擎**：编译时确定拓扑、节点单次调用 LLM、单向数据流、一次性执行。这在确定性工作流（大纲→并行生成→整合）中足够，但无法支撑更复杂的创作场景：

- Agent 无法自主获取信息和推理（→ P0 ReAct）
- 执行路径无法根据中间结果动态调整（→ P1 动态路由）
- 下游无法对上游质量进行评估和修正（→ P2 反馈循环）
- 工作流中断后无法恢复，长任务不可靠（→ P3 Checkpoint）

### 1.2 四阶段演进蓝图

```
现状（静态 DAG + 多模型降级）
    │
    ▼
P0: ReAct 能力 ── 节点从"单次调用"升级为"多轮推理循环"
    │  改动范围：orchestrator 包内部
    │  新增文件：react.go, tools/internal.go
    │  侵入度：零侵入 Dispatcher/Service
    │
    ▼
P1: 动态路由 ── DAG 从"编译时确定"升级为"运行时决策"
    │  改动范围：orchestrator + graph
    │  新增文件：router.go, llm_condition.go
    │  侵入度：轻微影响 templates
    │
    ▼
P2: 反馈循环 ── DAG 从"单向流"升级为"有限循环"
    │  改动范围：orchestrator + engine + graph
    │  新增文件：evaluator.go, feedback.go
    │  侵入度：影响 graph 结构和 engine 执行逻辑
    │
    ▼
P3: Checkpoint ── 工作流从"一次性执行"升级为"可中断可恢复"
       改动范围：新增 checkpoint 包 + engine + workflow service
       新增文件：checkpoint/, 数据库迁移
       侵入度：影响 engine + workflow service + 数据库
```

### 1.3 设计原则

- **每阶段独立可交付**：后一阶段建立在前一阶段基础上，但不要求前一阶段完美
- **侵入度递增**：P0 零侵入，P3 影响最大，风险逐步可控
- **向后兼容**：非 ReAct 节点、非动态路由边、非反馈循环的工作流，行为完全不变
- **接口预留**：每个阶段在设计时为下一阶段预留扩展点

---

## 二、P0：ReAct 能力（节点级多轮推理）

> 详细设计见 `docs/superpowers/specs/2026-04-07-react-capability-design.md`，此处仅做摘要和补充。

### 2.1 核心变更

| 组件 | 变更 | 文件 |
|------|------|------|
| Node | 新增 `ReactEnabled`、`Tools`、`ReactConfig` 字段 | `orchestrator/graph.go` |
| ReactExecutor | 新增 ReAct 推理循环（Thought→Action→Observation） | `orchestrator/react.go`（新增） |
| Engine | 识别 ReAct 节点，分流到 ReactExecutor | `orchestrator/engine.go` |
| InternalTools | 新增 5 个项目内部查询工具 | `agent/tools/internal.go`（新增） |
| ToolProvider | 新增工具提供接口，桥接 ToolRegistry 和 Engine | `orchestrator/react.go` |

### 2.2 执行流程

```
Engine.Run() → 拓扑排序 → 逐层执行
    │
    ├─ node.ReactEnabled == false → 原有路径：executor(ctx, node, state)
    │
    └─ node.ReactEnabled == true → ReactExecutor.Execute(ctx, node, state)
        │
        ├─ 构造 ReAct system prompt（任务 + 工具描述）
        ├─ 循环（最多 MaxRounds 轮）：
        │   ├─ 调用 LLM（Function Calling 模式）
        │   ├─ 有 ToolCalls → 执行工具 → Observation 追加 history
        │   ├─ 无 ToolCalls → FinalAnswer → 跳出
        │   └─ 熔断检查（轮次/时间/token）→ 强制收敛
        └─ 结果写入 SharedState
```

### 2.3 多维度熔断

```go
type ReactConfig struct {
    MaxRounds  int           // 默认 5
    MaxTimeout time.Duration // 默认 120s
    MaxTokens  int           // 默认 16384
}
```

三个维度任一触发 → 追加 system 消息要求立即输出最终答案 → 最后一次 LLM 调用不传 tools 定义。

### 2.4 内部工具

| 工具名 | 功能 | 入参 |
|--------|------|------|
| `get_characters` | 查询角色列表及属性 | `portfolio_id` |
| `get_world_setting` | 获取世界观设定 | `portfolio_id` |
| `get_chapter_summary` | 获取章节摘要 | `chapter_id` |
| `get_outline` | 获取大纲结构 | `portfolio_id` |
| `search_content` | 作品内容关键词搜索 | `portfolio_id`, `keyword` |

### 2.5 为 P1 预留的扩展点

- `ReactExecutor` 的 `ToolProvider` 接口可扩展为支持动态工具注入
- `ReactConfig` 可扩展为支持"推理策略"配置（如 ReAct / CoT / Plan-and-Execute）
