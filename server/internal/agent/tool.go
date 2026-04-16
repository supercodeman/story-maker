// server/internal/agent/tool.go
package agent

import "context"

// ToolCall LLM 返回的工具调用请求
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Tool 工具定义
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
	Execute     func(ctx context.Context, args map[string]any) (string, error)
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]*Tool
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]*Tool)}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool *Tool) {
	r.tools[tool.Name] = tool
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// HasTools 是否有注册的工具
func (r *ToolRegistry) HasTools() bool {
	return len(r.tools) > 0
}

// Merge 合并另一个 ToolRegistry 的工具到当前 registry（同名覆盖）
// 返回新的 ToolRegistry，不修改原有 registry
func (r *ToolRegistry) Merge(other *ToolRegistry) *ToolRegistry {
	merged := NewToolRegistry()
	if r != nil {
		for name, tool := range r.tools {
			merged.tools[name] = tool
		}
	}
	if other != nil {
		for name, tool := range other.tools {
			merged.tools[name] = tool
		}
	}
	return merged
}

// ToFunctionDefs 转换为 LLM API 需要的 tools 格式
func (r *ToolRegistry) ToFunctionDefs() []map[string]any {
	defs := make([]map[string]any, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	return defs
}
