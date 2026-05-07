// server/internal/agent/executor.go
package agent

import (
	"context"
	"fmt"
	"strings"

	"story-maker/server/internal/model"
)

// ExecContext 封装 Executor 需要的所有依赖
type ExecContext struct {
	Provider     AIProvider
	Task         *model.AITask
	TokenMgr     *TokenManager
	ToolRegistry *ToolRegistry
	ModelVersion string // 从 task.ModelName 解析出的具体模型版本，空则用 Provider 默认值
	GetProvider  func(name string) (AIProvider, error) // 可选：按名称获取其他 Provider（用于审核 agent 等跨模型场景）
}

// ParseModelName 解析 "provider" 或 "provider/model" 格式
// 例如 "qwen" -> ("qwen", ""), "qwen/qwen-max" -> ("qwen", "qwen-max")
func ParseModelName(modelName string) (providerName, modelVersion string) {
	parts := strings.SplitN(modelName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// TaskExecutor 任务执行策略接口
type TaskExecutor interface {
	Execute(ctx context.Context, ec *ExecContext) (interface{}, error)
}

// TaskExecutorRegistry 任务执行器注册表
type TaskExecutorRegistry struct {
	executors map[string]TaskExecutor
}

// NewTaskExecutorRegistry 创建注册表
func NewTaskExecutorRegistry() *TaskExecutorRegistry {
	return &TaskExecutorRegistry{executors: make(map[string]TaskExecutor)}
}

// Register 注册执行器
func (r *TaskExecutorRegistry) Register(taskType string, executor TaskExecutor) {
	r.executors[taskType] = executor
}

// Get 获取执行器
func (r *TaskExecutorRegistry) Get(taskType string) (TaskExecutor, error) {
	executor, ok := r.executors[taskType]
	if !ok {
		return nil, fmt.Errorf("unsupported task type: %s", taskType)
	}
	return executor, nil
}

// executeToolCall 执行单个工具调用（从 dispatcher.executeTool 迁移）
func executeToolCall(ctx context.Context, registry *ToolRegistry, call ToolCall) string {
	if registry == nil {
		return fmt.Sprintf("tool %s not found", call.Name)
	}

	tool, ok := registry.Get(call.Name)
	if !ok {
		return fmt.Sprintf("tool %s not found", call.Name)
	}

	result, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		return fmt.Sprintf("tool %s execution failed: %s", call.Name, err.Error())
	}
	return result
}

// extractJSONFromMarkdown 从 markdown 代码块中提取 JSON 内容
func extractJSONFromMarkdown(content string) string {
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "```json"); idx != -1 {
		content = content[idx+7:]
		if endIdx := strings.Index(content, "```"); endIdx != -1 {
			content = content[:endIdx]
		}
	} else if idx := strings.Index(content, "```"); idx != -1 {
		content = content[idx+3:]
		if endIdx := strings.Index(content, "```"); endIdx != -1 {
			content = content[:endIdx]
		}
	}
	return strings.TrimSpace(content)
}
