// server/internal/agent/orchestrator/loop_test.go
package orchestrator

import (
	"context"
	"fmt"
	"testing"
)

// mockExecutor 返回一个简单的 NodeExecutor，将 node.Prompt 作为结果返回
func mockExecutor() NodeExecutor {
	return func(ctx context.Context, node *Node, state *SharedState) (interface{}, error) {
		return map[string]interface{}{"content": node.Prompt}, nil
	}
}

// mockCallbackCollector 收集所有 callback 调用
type callbackRecord struct {
	NodeID string
	Status string
}

func mockCallbackCollector(records *[]callbackRecord) ProgressCallback {
	return func(workflowID uint, nodeID string, status string, result interface{}) {
		*records = append(*records, callbackRecord{NodeID: nodeID, Status: status})
	}
}

// TestLoopBasicExitOnFirstRound 测试第 1 轮即满足退出条件
func TestLoopBasicExitOnFirstRound(t *testing.T) {
	g := NewGraph()

	g.AddLoopNode(&LoopNode{
		ID: "test_loop",
		Config: LoopConfig{
			MaxRounds:   3,
			OnExhausted: ExhaustedContinue,
			ExitCondition: func(state *SharedState, round int) bool {
				// 第 1 轮就退出
				return true
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{
					ID:        "step",
					TaskType:  "text_gen",
					Prompt:    fmt.Sprintf("round_%d", round),
					OutputKey: "step_result",
				})
				return sub
			},
		},
	})

	state := NewSharedState()
	engine := NewEngine(mockExecutor(), nil)

	err := engine.Run(context.Background(), 1, g, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证只执行了 1 轮
	meta, ok := state.Get("test_loop.meta")
	if !ok {
		t.Fatal("missing loop meta")
	}
	lr := meta.(LoopResult)
	if lr.TotalRounds != 1 {
		t.Errorf("expected 1 round, got %d", lr.TotalRounds)
	}
	if lr.ExitReason != "condition_met" {
		t.Errorf("expected condition_met, got %s", lr.ExitReason)
	}
}

// TestLoopExitOnSecondRound 测试第 2 轮满足退出条件
func TestLoopExitOnSecondRound(t *testing.T) {
	g := NewGraph()

	g.AddLoopNode(&LoopNode{
		ID: "test_loop",
		Config: LoopConfig{
			MaxRounds:   5,
			OnExhausted: ExhaustedContinue,
			ExitCondition: func(state *SharedState, round int) bool {
				return round >= 2
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{
					ID:        "step",
					TaskType:  "text_gen",
					Prompt:    fmt.Sprintf("round_%d", round),
					OutputKey: "step_result",
				})
				return sub
			},
		},
	})

	state := NewSharedState()
	engine := NewEngine(mockExecutor(), nil)

	err := engine.Run(context.Background(), 1, g, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta, _ := state.Get("test_loop.meta")
	lr := meta.(LoopResult)
	if lr.TotalRounds != 2 {
		t.Errorf("expected 2 rounds, got %d", lr.TotalRounds)
	}
}

// TestLoopExhaustedContinue 测试循环耗尽 + ExhaustedContinue 策略
func TestLoopExhaustedContinue(t *testing.T) {
	g := NewGraph()

	g.AddLoopNode(&LoopNode{
		ID: "test_loop",
		Config: LoopConfig{
			MaxRounds:   2,
			OnExhausted: ExhaustedContinue,
			ExitCondition: func(state *SharedState, round int) bool {
				return false // 永远不满足
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{
					ID:        "step",
					TaskType:  "text_gen",
					Prompt:    fmt.Sprintf("round_%d", round),
					OutputKey: "step_result",
				})
				return sub
			},
		},
	})

	state := NewSharedState()
	engine := NewEngine(mockExecutor(), nil)

	err := engine.Run(context.Background(), 1, g, state)
	if err != nil {
		t.Fatalf("ExhaustedContinue should not return error, got: %v", err)
	}

	meta, _ := state.Get("test_loop.meta")
	lr := meta.(LoopResult)
	if lr.ExitReason != "exhausted" {
		t.Errorf("expected exhausted, got %s", lr.ExitReason)
	}
	if lr.TotalRounds != 2 {
		t.Errorf("expected 2 rounds, got %d", lr.TotalRounds)
	}
}

// TestLoopExhaustedFail 测试循环耗尽 + ExhaustedFail 策略
func TestLoopExhaustedFail(t *testing.T) {
	g := NewGraph()

	g.AddLoopNode(&LoopNode{
		ID: "test_loop",
		Config: LoopConfig{
			MaxRounds:   2,
			OnExhausted: ExhaustedFail,
			ExitCondition: func(state *SharedState, round int) bool {
				return false
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{
					ID:        "step",
					TaskType:  "text_gen",
					Prompt:    "test",
					OutputKey: "step_result",
				})
				return sub
			},
		},
	})

	state := NewSharedState()
	engine := NewEngine(mockExecutor(), nil)

	err := engine.Run(context.Background(), 1, g, state)
	if err == nil {
		t.Fatal("ExhaustedFail should return error")
	}
}

// TestLoopStateHistoryPreserved 测试每轮结果按轮次保留
func TestLoopStateHistoryPreserved(t *testing.T) {
	g := NewGraph()

	g.AddLoopNode(&LoopNode{
		ID: "test_loop",
		Config: LoopConfig{
			MaxRounds:   3,
			OnExhausted: ExhaustedContinue,
			ExitCondition: func(state *SharedState, round int) bool {
				return round >= 3
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{
					ID:        "step",
					TaskType:  "text_gen",
					Prompt:    fmt.Sprintf("content_round_%d", round),
					OutputKey: "step_result",
				})
				return sub
			},
		},
	})

	state := NewSharedState()
	engine := NewEngine(mockExecutor(), nil)

	err := engine.Run(context.Background(), 1, g, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证每轮历史都保留
	for round := 1; round <= 3; round++ {
		key := fmt.Sprintf("test_loop.round_%d.step_result", round)
		val, ok := state.Get(key)
		if !ok {
			t.Errorf("missing history key: %s", key)
			continue
		}
		expected := fmt.Sprintf("content_round_%d", round)
		if val != expected {
			t.Errorf("round %d: expected %q, got %q", round, expected, val)
		}
	}

	// 验证 latest 指向最后一轮
	latestVal, ok := state.Get("test_loop.latest.step_result")
	if !ok {
		t.Fatal("missing latest key")
	}
	if latestVal != "content_round_3" {
		t.Errorf("latest: expected content_round_3, got %v", latestVal)
	}

	// 验证原始 key 也被回写
	origVal, ok := state.Get("step_result")
	if !ok {
		t.Fatal("missing original key")
	}
	if origVal != "content_round_3" {
		t.Errorf("original key: expected content_round_3, got %v", origVal)
	}
}

// TestLoopWithDependency 测试 LoopNode 依赖前置普通节点
func TestLoopWithDependency(t *testing.T) {
	g := NewGraph()

	// 前置节点
	g.AddNode(&Node{
		ID:        "prepare",
		TaskType:  "text_gen",
		Prompt:    "prepared_data",
		OutputKey: "input_data",
	})

	// 循环节点依赖 prepare
	g.AddLoopNode(&LoopNode{
		ID:        "process_loop",
		DependsOn: []string{"prepare"},
		Config: LoopConfig{
			MaxRounds:   2,
			OnExhausted: ExhaustedContinue,
			ExitCondition: func(state *SharedState, round int) bool {
				return round >= 1
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{
					ID:        "process",
					TaskType:  "text_gen",
					Prompt:    "processing",
					OutputKey: "process_result",
					InputMap:  map[string]string{"input": "input_data"},
				})
				return sub
			},
		},
	})

	g.AddEdge("prepare", "process_loop")

	state := NewSharedState()
	engine := NewEngine(mockExecutor(), nil)

	err := engine.Run(context.Background(), 1, g, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证前置节点结果存在
	if _, ok := state.Get("input_data"); !ok {
		t.Error("missing input_data from prepare node")
	}

	// 验证循环节点结果存在
	if _, ok := state.Get("process_result"); !ok {
		t.Error("missing process_result from loop")
	}
}

// TestNestedLoop 测试嵌套循环
func TestNestedLoop(t *testing.T) {
	g := NewGraph()

	// 用 executor 调用次数来统计实际执行轮次，避免 Validate 调用 SubGraphBuilder 干扰计数
	outerExecutions := 0
	innerExecutions := 0

	countingExecutor := func(ctx context.Context, node *Node, state *SharedState) (interface{}, error) {
		if node.ID == "outer_marker" {
			outerExecutions++
		}
		if node.ID == "inner_step" {
			innerExecutions++
		}
		return map[string]interface{}{"content": node.Prompt}, nil
	}

	g.AddLoopNode(&LoopNode{
		ID: "outer_loop",
		Config: LoopConfig{
			MaxRounds:   2,
			OnExhausted: ExhaustedContinue,
			ExitCondition: func(state *SharedState, round int) bool {
				return round >= 2
			},
			SubGraphBuilder: func(outerRound int) *Graph {
				sub := NewGraph()

				// 外层标记节点
				sub.AddNode(&Node{
					ID: "outer_marker", TaskType: "text_gen",
					Prompt: fmt.Sprintf("outer_%d", outerRound), OutputKey: "outer_result",
				})

				// 内层循环依赖外层标记
				sub.AddLoopNode(&LoopNode{
					ID:        "inner_loop",
					DependsOn: []string{"outer_marker"},
					Config: LoopConfig{
						MaxRounds:   2,
						OnExhausted: ExhaustedContinue,
						ExitCondition: func(state *SharedState, round int) bool {
							return round >= 2
						},
						SubGraphBuilder: func(innerRound int) *Graph {
							innerSub := NewGraph()
							innerSub.AddNode(&Node{
								ID: "inner_step", TaskType: "text_gen",
								Prompt:    fmt.Sprintf("outer_%d_inner_%d", outerRound, innerRound),
								OutputKey: "inner_result",
							})
							return innerSub
						},
					},
				})

				sub.AddEdge("outer_marker", "inner_loop")
				return sub
			},
		},
	})

	state := NewSharedState()
	engine := NewEngine(countingExecutor, nil)

	err := engine.Run(context.Background(), 1, g, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 外层 2 轮，每轮内层 2 轮 = 总共 4 次内层执行
	if outerExecutions != 2 {
		t.Errorf("expected 2 outer executions, got %d", outerExecutions)
	}
	if innerExecutions != 4 {
		t.Errorf("expected 4 inner executions, got %d", innerExecutions)
	}
}

// TestLoopCallbackEvents 测试循环的 callback 事件序列
func TestLoopCallbackEvents(t *testing.T) {
	g := NewGraph()

	g.AddLoopNode(&LoopNode{
		ID: "test_loop",
		Config: LoopConfig{
			MaxRounds:   2,
			OnExhausted: ExhaustedContinue,
			ExitCondition: func(state *SharedState, round int) bool {
				return round >= 2
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{
					ID:        "step",
					TaskType:  "text_gen",
					Prompt:    "test",
					OutputKey: "result",
				})
				return sub
			},
		},
	})

	var records []callbackRecord
	state := NewSharedState()
	engine := NewEngine(mockExecutor(), mockCallbackCollector(&records))

	err := engine.Run(context.Background(), 1, g, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证事件序列：running → loop_round(1) → step running → step completed → loop_round(2) → step running → step completed → completed
	expectedStatuses := []string{"running", "loop_round", "running", "completed", "loop_round", "running", "completed", "completed"}
	if len(records) != len(expectedStatuses) {
		t.Fatalf("expected %d callbacks, got %d: %+v", len(expectedStatuses), len(records), records)
	}
	for i, expected := range expectedStatuses {
		if records[i].Status != expected {
			t.Errorf("callback[%d]: expected status %q, got %q", i, expected, records[i].Status)
		}
	}
}

// TestReviewNotPassed 测试修复后的 reviewNotPassed
func TestReviewNotPassed(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"passed true", `{"passed": true, "overall_score": 80}`, false},
		{"passed false", `{"passed": false, "overall_score": 60}`, true},
		{"passed false no space", `{"passed":false,"overall_score":50}`, true},
		{"empty string", "", true},
		{"invalid json", "not json", true},
		{"missing key", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewSharedState()
			if tt.value != "" {
				state.Set("review", tt.value)
			}
			result := reviewNotPassed(state, "review")
			if result != tt.expected {
				t.Errorf("reviewNotPassed(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestGraphValidateLoopNode 测试 Graph.Validate 对 LoopNode 的校验
func TestGraphValidateLoopNode(t *testing.T) {
	// 正常情况
	g := NewGraph()
	g.AddNode(&Node{ID: "start", TaskType: "text_gen", Prompt: "test", OutputKey: "out"})
	g.AddLoopNode(&LoopNode{
		ID:        "loop",
		DependsOn: []string{"start"},
		Config: LoopConfig{
			MaxRounds: 2,
			ExitCondition: func(state *SharedState, round int) bool {
				return true
			},
			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()
				sub.AddNode(&Node{ID: "sub_step", TaskType: "text_gen", Prompt: "test", OutputKey: "sub_out"})
				return sub
			},
		},
	})
	g.AddEdge("start", "loop")

	if err := g.Validate(); err != nil {
		t.Fatalf("valid graph should pass: %v", err)
	}

	// MaxRounds <= 0
	g2 := NewGraph()
	g2.AddLoopNode(&LoopNode{
		ID: "bad_loop",
		Config: LoopConfig{
			MaxRounds:       0,
			ExitCondition:   func(state *SharedState, round int) bool { return true },
			SubGraphBuilder: func(round int) *Graph { return NewGraph() },
		},
	})
	if err := g2.Validate(); err == nil {
		t.Error("MaxRounds=0 should fail validation")
	}

	// ID 冲突
	g3 := NewGraph()
	g3.AddNode(&Node{ID: "conflict", TaskType: "text_gen", Prompt: "test", OutputKey: "out"})
	g3.AddLoopNode(&LoopNode{
		ID: "conflict",
		Config: LoopConfig{
			MaxRounds:       1,
			ExitCondition:   func(state *SharedState, round int) bool { return true },
			SubGraphBuilder: func(round int) *Graph { sub := NewGraph(); sub.AddNode(&Node{ID: "s", TaskType: "text_gen", Prompt: "t", OutputKey: "o"}); return sub },
		},
	})
	if err := g3.Validate(); err == nil {
		t.Error("conflicting IDs should fail validation")
	}
}
