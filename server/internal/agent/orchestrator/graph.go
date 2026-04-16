// server/internal/agent/orchestrator/graph.go
package orchestrator

import (
	"errors"
	"fmt"
)

// EdgeType 边类型
type EdgeType int

const (
	EdgeNormal      EdgeType = iota // 普通依赖边
	EdgeConditional                 // 条件边（运行时判断是否跳过）
)

// Node DAG 节点，对应一个 AI 子任务
type Node struct {
	ID             string            // 节点唯一标识，如 "outline", "character_desc"
	TaskType       string            // 对应 model.TaskType* 常量
	ModelName      string            // 使用的模型名
	FallbackModels []string          // 降级备选模型列表，主模型限流/失败时按顺序尝试
	Prompt         string            // 提示词模板，支持 {{.key}} 从 SharedState 注入
	OutputKey      string            // 执行结果写入 SharedState 的 key
	InputMap       map[string]string // 从 SharedState 读取的 key 映射（用于 Prompt 渲染）
	DependsOn      []string          // 依赖的节点 ID 列表
	MaxTokens      int               // 最大输出 token 数，0 表示使用 executor 默认值
	SkipOnAllFail  bool              // 所有模型失败时跳过节点而非报错
	SystemPrompt   string            // 系统提示词，通过 History JSON 传递给 executor
	EnableTools    bool              // 启用 tool calling（由 makeNodeExecutor 注入节点级工具）
}

// Edge DAG 边
type Edge struct {
	From      string
	To        string
	Type      EdgeType
	Condition func(state *SharedState) bool // 仅 EdgeConditional 时有效
}

// Graph DAG 图
type Graph struct {
	Nodes     map[string]*Node     // 普通节点
	LoopNodes map[string]*LoopNode // 循环复合节点
	Edges     []*Edge
}

// NewGraph 创建空 DAG 图
func NewGraph() *Graph {
	return &Graph{
		Nodes:     make(map[string]*Node),
		LoopNodes: make(map[string]*LoopNode),
		Edges:     make([]*Edge, 0),
	}
}

// AddNode 添加节点
func (g *Graph) AddNode(node *Node) {
	g.Nodes[node.ID] = node
}

// AddEdge 添加普通依赖边
func (g *Graph) AddEdge(from, to string) {
	g.Edges = append(g.Edges, &Edge{
		From: from,
		To:   to,
		Type: EdgeNormal,
	})
}

// AddConditionalEdge 添加条件边
func (g *Graph) AddConditionalEdge(from, to string, cond func(*SharedState) bool) {
	g.Edges = append(g.Edges, &Edge{
		From:      from,
		To:        to,
		Type:      EdgeConditional,
		Condition: cond,
	})
}

// AddLoopNode 添加循环复合节点
func (g *Graph) AddLoopNode(loop *LoopNode) {
	if g.LoopNodes == nil {
		g.LoopNodes = make(map[string]*LoopNode)
	}
	g.LoopNodes[loop.ID] = loop
}

// Validate 校验 DAG：检测环、孤立节点、缺失依赖
func (g *Graph) Validate() error {
	if len(g.Nodes) == 0 && len(g.LoopNodes) == 0 {
		return errors.New("graph has no nodes")
	}

	// 收集所有节点 ID（普通 + 循环），用于边引用检查
	allIDs := make(map[string]bool)
	for id := range g.Nodes {
		allIDs[id] = true
	}
	for id := range g.LoopNodes {
		if allIDs[id] {
			return fmt.Errorf("loop node ID %s conflicts with regular node", id)
		}
		allIDs[id] = true
	}

	// 检查边引用的节点是否存在
	for _, edge := range g.Edges {
		if !allIDs[edge.From] {
			return fmt.Errorf("edge references non-existent node: %s", edge.From)
		}
		if !allIDs[edge.To] {
			return fmt.Errorf("edge references non-existent node: %s", edge.To)
		}
	}

	// 检查普通节点 DependsOn
	for _, node := range g.Nodes {
		for _, dep := range node.DependsOn {
			if !allIDs[dep] {
				return fmt.Errorf("node %s depends on non-existent node: %s", node.ID, dep)
			}
		}
	}

	// 检查 LoopNode DependsOn
	for _, loop := range g.LoopNodes {
		for _, dep := range loop.DependsOn {
			if !allIDs[dep] {
				return fmt.Errorf("loop node %s depends on non-existent node: %s", loop.ID, dep)
			}
		}
	}

	// LoopNode 自身校验
	for _, loop := range g.LoopNodes {
		if loop.Config.MaxRounds <= 0 {
			return fmt.Errorf("loop node %s: MaxRounds must be > 0", loop.ID)
		}
		if loop.Config.SubGraphBuilder == nil {
			return fmt.Errorf("loop node %s: SubGraphBuilder is nil", loop.ID)
		}
		if loop.Config.ExitCondition == nil {
			return fmt.Errorf("loop node %s: ExitCondition is nil", loop.ID)
		}
		// 递归校验第 1 轮子 DAG
		subGraph := loop.Config.SubGraphBuilder(1)
		if err := subGraph.Validate(); err != nil {
			return fmt.Errorf("loop node %s: sub-graph validation failed: %w", loop.ID, err)
		}
	}

	// 环检测：通过拓扑排序
	_, err := g.TopologicalSort()
	return err
}

// TopologicalSort 拓扑排序，返回分层结果 [[layer0_ids], [layer1_ids], ...]
// 同一层内的节点可以并行执行
func (g *Graph) TopologicalSort() ([][]string, error) {
	// 构建入度表和邻接表（合并 Edges + DependsOn）
	inDegree := make(map[string]int)
	successors := make(map[string][]string) // from -> [to...]

	// 普通节点初始化
	for id := range g.Nodes {
		inDegree[id] = 0
	}
	// LoopNode 也参与拓扑排序
	for id := range g.LoopNodes {
		inDegree[id] = 0
	}

	// 从 Edges 构建
	for _, edge := range g.Edges {
		inDegree[edge.To]++
		successors[edge.From] = append(successors[edge.From], edge.To)
	}

	// 从普通节点 DependsOn 构建（DependsOn 表示"我依赖谁"，即 dep -> node 的边）
	for _, node := range g.Nodes {
		for _, dep := range node.DependsOn {
			if !g.hasEdge(dep, node.ID) {
				inDegree[node.ID]++
				successors[dep] = append(successors[dep], node.ID)
			}
		}
	}

	// 从 LoopNode DependsOn 构建
	for _, loop := range g.LoopNodes {
		for _, dep := range loop.DependsOn {
			if !g.hasEdge(dep, loop.ID) {
				inDegree[loop.ID]++
				successors[dep] = append(successors[dep], loop.ID)
			}
		}
	}

	// BFS 分层拓扑排序
	var layers [][]string
	processed := 0

	// 找出入度为 0 的节点作为第一层
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	for len(queue) > 0 {
		layers = append(layers, queue)
		processed += len(queue)

		var nextQueue []string
		for _, id := range queue {
			for _, succ := range successors[id] {
				inDegree[succ]--
				if inDegree[succ] == 0 {
					nextQueue = append(nextQueue, succ)
				}
			}
		}
		queue = nextQueue
	}

	totalNodes := len(g.Nodes) + len(g.LoopNodes)
	if processed != totalNodes {
		return nil, errors.New("graph contains a cycle")
	}

	return layers, nil
}

// hasEdge 检查是否已存在从 from 到 to 的边
func (g *Graph) hasEdge(from, to string) bool {
	for _, edge := range g.Edges {
		if edge.From == from && edge.To == to {
			return true
		}
	}
	return false
}

// NodeCount 返回顶层节点数量（LoopNode 算 1 个，不展开子节点）
// loop 内部子节点的进度通过 loop_round 状态展示，不参与整体百分比计算
func (g *Graph) NodeCount() int {
	return len(g.Nodes) + len(g.LoopNodes)
}
