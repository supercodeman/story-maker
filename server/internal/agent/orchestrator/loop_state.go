// server/internal/agent/orchestrator/loop_state.go
package orchestrator

import "fmt"

// newLoopRoundState 创建子 DAG 的 state 视图
// 将主 state 的快照注入子 state，子 DAG 可以通过 InputMap 引用主 state 的数据
// 同时注入轮次元信息（loop_round, loop_id），子 DAG 的 Prompt 可以用 {{.loop_round}}
func newLoopRoundState(parent *SharedState, loopID string, round int) *SharedState {
	sub := NewSharedState()
	// 将主 state 的快照注入子 state
	for k, v := range parent.Snapshot() {
		sub.Set(k, v)
	}
	// 注入轮次元信息
	sub.Set("loop_round", round)
	sub.Set("loop_id", loopID)
	return sub
}

// flushLoopRoundResults 将子 DAG 的输出写回主 state
// 命名规范：
//   - 历史保留：{loopID}.round_{round}.{key}
//   - latest 指针：{loopID}.latest.{key}（每轮覆盖）
//   - 原始 key 直接回写（让后续主 DAG 节点可以直接引用）
func flushLoopRoundResults(parent *SharedState, sub *SharedState, loopID string, round int) {
	parentSnapshot := parent.Snapshot()
	subSnapshot := sub.Snapshot()

	for k, v := range subSnapshot {
		// 跳过从 parent 继承且未被修改的 key
		if parentVal, exists := parentSnapshot[k]; exists {
			if fmt.Sprintf("%v", v) == fmt.Sprintf("%v", parentVal) {
				continue
			}
		}

		// 跳过注入的元信息 key
		if k == "loop_round" || k == "loop_id" {
			continue
		}

		// 按轮次命名保留历史
		roundKey := fmt.Sprintf("%s.round_%d.%s", loopID, round, k)
		parent.Set(roundKey, v)

		// 更新 latest 指针
		latestKey := fmt.Sprintf("%s.latest.%s", loopID, k)
		parent.Set(latestKey, v)

		// 直接写回原始 key
		parent.Set(k, v)
	}
}
