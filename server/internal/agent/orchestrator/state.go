// server/internal/agent/orchestrator/state.go
package orchestrator

import "sync"

// SharedState 节点间共享的并发安全状态池
type SharedState struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewSharedState 创建共享状态实例
func NewSharedState() *SharedState {
	return &SharedState{
		data: make(map[string]interface{}),
	}
}

// Set 写入键值对
func (s *SharedState) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get 读取值
func (s *SharedState) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// GetString 读取字符串值，不存在返回空串
func (s *SharedState) GetString(key string) string {
	v, ok := s.Get(key)
	if !ok {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

// Snapshot 返回当前状态快照（用于模板渲染），非并发安全的副本
func (s *SharedState) Snapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		snap[k] = v
	}
	return snap
}
