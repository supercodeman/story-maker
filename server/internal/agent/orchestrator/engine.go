// server/internal/agent/orchestrator/engine.go
package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"text/template"

	"golang.org/x/sync/errgroup"
)

// NodeExecutor 节点执行回调（由 Dispatcher.ExecuteSingle 提供）
type NodeExecutor func(ctx context.Context, node *Node, state *SharedState) (interface{}, error)

// ProgressCallback 节点级进度回调（WebSocket 通知）
type ProgressCallback func(workflowID uint, nodeID string, status string, result interface{})

// Engine 编排引擎，按 DAG 拓扑分层并发执行节点
type Engine struct {
	executor NodeExecutor
	callback ProgressCallback
}

// NewEngine 创建编排引擎
func NewEngine(executor NodeExecutor, callback ProgressCallback) *Engine {
	return &Engine{
		executor: executor,
		callback: callback,
	}
}

// Run 执行整个 DAG
// 1. 拓扑排序得到分层
// 2. 逐层遍历，层内 errgroup 并行
// 3. 每个节点: 判断类型（普通/循环）→ 分别执行
// 4. 任一节点失败则 ctx cancel，整个工作流失败
func (e *Engine) Run(ctx context.Context, workflowID uint, graph *Graph, state *SharedState) error {
	layers, err := graph.TopologicalSort()
	if err != nil {
		return fmt.Errorf("topological sort failed: %w", err)
	}

	for layerIdx, layer := range layers {
		log.Printf("[orchestrator] executing layer %d: %v", layerIdx, layer)

		g, gCtx := errgroup.WithContext(ctx)

		for _, nodeID := range layer {
			nodeID := nodeID // 闭包捕获

			// 判断是 LoopNode 还是普通 Node
			if loopNode, ok := graph.LoopNodes[nodeID]; ok {
				g.Go(func() error {
					return e.runLoop(gCtx, workflowID, loopNode, state)
				})
			} else {
				node := graph.Nodes[nodeID]
				g.Go(func() error {
					return e.runNode(gCtx, workflowID, graph, node, state)
				})
			}
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	return nil
}

// runNode 执行普通节点：条件边检查 → 渲染 Prompt → executor 执行 → 结果写入 state → callback 通知
func (e *Engine) runNode(ctx context.Context, workflowID uint, graph *Graph, node *Node, state *SharedState) error {
	// 条件边检查：如果所有指向该节点的条件边都不满足，则跳过
	if e.shouldSkip(graph, node, state) {
		log.Printf("[orchestrator] skipping node %s (condition not met)", node.ID)
		if e.callback != nil {
			e.callback(workflowID, node.ID, "skipped", nil)
		}
		return nil
	}

	// 渲染 Prompt 模板
	renderedPrompt, err := e.renderPrompt(node, state)
	if err != nil {
		return fmt.Errorf("render prompt for node %s: %w", node.ID, err)
	}
	node.Prompt = renderedPrompt

	// 通知开始执行
	if e.callback != nil {
		e.callback(workflowID, node.ID, "running", nil)
	}

	// 执行节点
	result, err := e.executor(ctx, node, state)
	if err != nil {
		if e.callback != nil {
			e.callback(workflowID, node.ID, "failed", nil)
		}
		return fmt.Errorf("execute node %s: %w", node.ID, err)
	}

	// executor 返回 (nil, nil) 表示节点被跳过（SkipOnAllFail）
	if result == nil {
		log.Printf("[orchestrator] node %s skipped (all models failed, SkipOnAllFail)", node.ID)
		if e.callback != nil {
			e.callback(workflowID, node.ID, "skipped", nil)
		}
		return nil
	}

	// 结果写入共享状态
	if node.OutputKey != "" {
		state.Set(node.OutputKey, extractContent(result))
	}

	// 通知完成
	if e.callback != nil {
		e.callback(workflowID, node.ID, "completed", result)
	}

	log.Printf("[orchestrator] node %s completed", node.ID)
	return nil
}

// runLoop 执行循环复合节点
// 每轮：构建子 DAG → 创建隔离 state → 递归 Run → 回写结果 → 检查退出条件
func (e *Engine) runLoop(ctx context.Context, workflowID uint, loop *LoopNode, state *SharedState) error {
	log.Printf("[orchestrator] entering loop %s (max_rounds=%d)", loop.ID, loop.Config.MaxRounds)

	if e.callback != nil {
		e.callback(workflowID, loop.ID, "running", map[string]interface{}{
			"type": "loop_start", "max_rounds": loop.Config.MaxRounds,
		})
	}

	var lastRound int
	for round := 1; round <= loop.Config.MaxRounds; round++ {
		lastRound = round
		log.Printf("[orchestrator] loop %s: starting round %d/%d", loop.ID, round, loop.Config.MaxRounds)

		// 通知前端当前轮次
		if e.callback != nil {
			e.callback(workflowID, loop.ID, "loop_round", map[string]interface{}{
				"round": round, "max_rounds": loop.Config.MaxRounds,
			})
		}

		// 1. 构建本轮子 DAG
		subGraph := loop.Config.SubGraphBuilder(round)
		if err := subGraph.Validate(); err != nil {
			return fmt.Errorf("loop %s round %d: invalid sub-graph: %w", loop.ID, round, err)
		}

		// 2. 创建子 DAG 的隔离 state 视图
		subState := newLoopRoundState(state, loop.ID, round)

		// 3. 递归调用 Engine.Run 执行子 DAG（天然支持嵌套 LoopNode）
		if err := e.Run(ctx, workflowID, subGraph, subState); err != nil {
			return fmt.Errorf("loop %s round %d failed: %w", loop.ID, round, err)
		}

		// 4. 将子 DAG 结果写回主 state（按轮次命名 + latest 指针）
		flushLoopRoundResults(state, subState, loop.ID, round)

		// 5. 检查退出条件
		if loop.Config.ExitCondition(state, round) {
			log.Printf("[orchestrator] loop %s: exit condition met at round %d", loop.ID, round)
			state.Set(loop.ID+".meta", LoopResult{
				TotalRounds: round, ExitReason: "condition_met", ExitRound: round,
			})
			if e.callback != nil {
				e.callback(workflowID, loop.ID, "completed", map[string]interface{}{
					"rounds": round, "exit_reason": "condition_met",
				})
			}
			return nil
		}
	}

	// 循环耗尽
	log.Printf("[orchestrator] loop %s: exhausted after %d rounds", loop.ID, lastRound)
	state.Set(loop.ID+".meta", LoopResult{
		TotalRounds: lastRound, ExitReason: "exhausted", ExitRound: lastRound,
	})

	switch loop.Config.OnExhausted {
	case ExhaustedFail:
		if e.callback != nil {
			e.callback(workflowID, loop.ID, "failed", map[string]interface{}{
				"rounds": lastRound, "exit_reason": "exhausted",
			})
		}
		return fmt.Errorf("loop %s: max rounds (%d) exhausted without meeting exit condition", loop.ID, lastRound)

	default: // ExhaustedContinue
		if e.callback != nil {
			e.callback(workflowID, loop.ID, "completed_with_warning", map[string]interface{}{
				"rounds": lastRound, "exit_reason": "exhausted",
			})
		}
		return nil
	}
}

// shouldSkip 检查节点是否应被跳过（所有指向它的条件边都不满足时跳过）
func (e *Engine) shouldSkip(graph *Graph, node *Node, state *SharedState) bool {
	hasConditional := false
	for _, edge := range graph.Edges {
		if edge.To != node.ID || edge.Type != EdgeConditional {
			continue
		}
		hasConditional = true
		if edge.Condition != nil && edge.Condition(state) {
			return false // 至少一个条件满足，不跳过
		}
	}
	// 没有条件边则不跳过；有条件边但全不满足则跳过
	return hasConditional
}

// renderPrompt 使用 SharedState 渲染节点的 Prompt 模板
func (e *Engine) renderPrompt(node *Node, state *SharedState) (string, error) {
	if node.Prompt == "" {
		return "", nil
	}

	// 构建模板数据：优先使用 InputMap 映射，否则直接用 Snapshot
	data := state.Snapshot()
	if len(node.InputMap) > 0 {
		mapped := make(map[string]interface{}, len(node.InputMap))
		for tplKey, stateKey := range node.InputMap {
			mapped[tplKey] = data[stateKey]
		}
		data = mapped
	}

	tmpl, err := template.New(node.ID).Parse(node.Prompt)
	if err != nil {
		return node.Prompt, nil // 解析失败则返回原始 Prompt
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return node.Prompt, nil
	}

	return buf.String(), nil
}

// extractContent 从 executor 返回的结果中提取纯文本
// executor 返回 map[string]interface{}{"content": "文本"} 结构，
// 直接存入 state 会导致下游模板渲染出 map[content:...] 乱码
func extractContent(result interface{}) interface{} {
	if m, ok := result.(map[string]interface{}); ok {
		if content, exists := m["content"]; exists {
			return content
		}
	}
	return result
}
