// server/internal/agent/dispatcher.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"ai-curton/server/internal/model"
)

// FallbackEntry 降级链条目
type FallbackEntry struct {
	Provider  string
	ModelName string
}

// TaskResult 任务结果结构
type TaskResult struct {
	TaskID uint        `json:"task_id"`
	Status string      `json:"status"`
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

// Dispatcher AI 任务分发器，负责路由、异步执行、状态管理
type Dispatcher struct {
	providers        map[string]AIProvider    // model_name -> Provider
	ttsProvider      TTSProvider              // TTS Provider（MiniMax 等）
	videoProvider    VideoProvider            // Video Provider（CogVideo 等）
	keyStore         KeyStore                 // API Key 存储接口
	taskStore        TaskStore                // 任务存储接口
	notifier         Notifier                 // WebSocket 通知接口
	modelChecker     ModelChecker             // 模型可用性检查（可选，由 service 层注入）
	toolRegistry     *ToolRegistry            // 工具注册表
	tokenMgr         *TokenManager            // 上下文窗口管理器
	executorRegistry *TaskExecutorRegistry    // 任务执行器注册表
	OnTaskCompleted  func(task *model.AITask) // 任务完成回调，用于刷新 token 缓存等
	mu               sync.RWMutex
}

// KeyStore API Key 存储接口（由 Service 层实现）
type KeyStore interface {
	GetUserKey(ctx context.Context, userID uint, provider string) (string, error)
	GetDefaultKey(ctx context.Context, provider string) (string, error)
}

// TaskStore 任务存储接口（由 DAO 层实现）
type TaskStore interface {
	CreateTask(ctx context.Context, task *model.AITask) error
	UpdateTask(ctx context.Context, task *model.AITask) error
	GetTask(ctx context.Context, taskID uint) (*model.AITask, error)
}

// Notifier WebSocket 通知接口
type Notifier interface {
	NotifyUser(userID uint, message interface{}) error
	NotifyUserWithType(userID uint, msgType string, message interface{}) error
}

// ModelChecker 模型可用性检查接口（由 service 层实现，注入到 Dispatcher）
type ModelChecker interface {
	// IsModelAvailable 检查指定 Provider 的指定能力是否可用（provider 级别）
	IsModelAvailable(provider, capability string) bool
	// IsSubModelAvailable 检查指定 Provider 下的子模型是否可用
	IsSubModelAvailable(provider, modelName, capability string) bool
	// GetFallbackChain 获取指定能力的降级链（从 DB 获取，按优先级排序，排除当前模型）
	GetFallbackChain(capability, currentProvider, currentModel string) []FallbackEntry
}

// NewDispatcher 创建 Dispatcher 实例
func NewDispatcher(keyStore KeyStore, taskStore TaskStore, notifier Notifier) *Dispatcher {
	d := &Dispatcher{
		providers:        make(map[string]AIProvider),
		keyStore:         keyStore,
		taskStore:        taskStore,
		notifier:         notifier,
		toolRegistry:     NewToolRegistry(),
		tokenMgr:         NewTokenManager(32768, 16384), // 32K 窗口，预留 16K 回复空间以支持长文生成
		executorRegistry: NewTaskExecutorRegistry(),
	}
	d.registerDefaultExecutors()
	return d
}

// registerDefaultExecutors 注册所有内置任务执行器
func (d *Dispatcher) registerDefaultExecutors() {
	text := &TextTaskExecutor{}
	d.executorRegistry.Register(model.TaskTypeTextGen, text)
	d.executorRegistry.Register(model.TaskTypeTextPolish, text)
	d.executorRegistry.Register(model.TaskTypeStoryboard, text)

	image := &ImageTaskExecutor{}
	d.executorRegistry.Register(model.TaskTypeImageGen, image)
	d.executorRegistry.Register(model.TaskTypeImageEdit, image)

	d.executorRegistry.Register(model.TaskTypeCharacterAdjust, &CharacterTaskExecutor{})

	chapter := &ChapterTaskExecutor{}
	d.executorRegistry.Register(model.TaskTypeChapterSummaryPolish, chapter)
	d.executorRegistry.Register(model.TaskTypeChapterPolish, chapter)
	d.executorRegistry.Register(model.TaskTypeChapterExpand, chapter)
	d.executorRegistry.Register(model.TaskTypeChapterContinue, chapter)

	d.executorRegistry.Register(model.TaskTypeOutlineGenerate, &OutlineTaskExecutor{})

	outlineChapter := &OutlineChapterExecutor{}
	d.executorRegistry.Register(model.TaskTypeOutlineTitlePolish, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeOutlineSummaryPolish, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeOutlineSummaryExpand, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeOutlineGenerateCharacters, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerGenerateTopic, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerGenerateStoryline, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerGenerateCharacters, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerStorylineDraft, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerStorylineReview, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerCharactersDraft, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerCharactersReview, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerOpeningDraft, outlineChapter)
	d.executorRegistry.Register(model.TaskTypeButlerOpeningReview, outlineChapter)

	d.executorRegistry.Register(model.TaskTypeKnowledgeExtract, text)
	d.executorRegistry.Register(model.TaskTypeOverviewExtract, text)

	// 记忆提取相关任务（复用 text executor）
	d.executorRegistry.Register(model.TaskTypeMemoryFeatureExtract, text)
	d.executorRegistry.Register(model.TaskTypeMemoryPromptCompile, text)
	d.executorRegistry.Register(model.TaskTypeMemoryQualityEval, text)
	d.executorRegistry.Register(model.TaskTypeMemoryReviewQuality, text)
	d.executorRegistry.Register(model.TaskTypeMemoryReviewCompliance, text)
	d.executorRegistry.Register(model.TaskTypeMemoryReviewDecision, text)

	// 小说修订工作流（复用 text executor）
	d.executorRegistry.Register(model.TaskTypeRevisionAnalysis, text)
	d.executorRegistry.Register(model.TaskTypeRevisionPlanning, text)

	// 注意：audio_gen 和 video_gen 的 executor 需要在 Provider 注册后通过
	// RegisterTTSProvider / RegisterVideoProvider 方法注册
}

// RegisterTTSProvider 注册 TTS Provider 并注册对应的 Executor
func (d *Dispatcher) RegisterTTSProvider(tts TTSProvider) {
	d.ttsProvider = tts
	d.executorRegistry.Register(model.TaskTypeAudioGen, NewAudioTaskExecutor(tts))
}

// RegisterVideoProvider 注册 Video Provider 并注册对应的 Executor
func (d *Dispatcher) RegisterVideoProvider(vp VideoProvider) {
	d.videoProvider = vp
	d.executorRegistry.Register(model.TaskTypeVideoGen, NewVideoTaskExecutor(vp))
}

// GetTTSProvider 获取 TTS Provider
func (d *Dispatcher) GetTTSProvider() TTSProvider {
	return d.ttsProvider
}

// GetVideoProvider 获取 Video Provider
func (d *Dispatcher) GetVideoProvider() VideoProvider {
	return d.videoProvider
}

// RegisterExecutor 注册自定义任务执行器（供外部扩展）
func (d *Dispatcher) RegisterExecutor(taskType string, executor TaskExecutor) {
	d.executorRegistry.Register(taskType, executor)
}

// SetToolRegistry 设置工具注册表
func (d *Dispatcher) SetToolRegistry(registry *ToolRegistry) {
	d.toolRegistry = registry
}

// SetModelChecker 注入模型可用性检查器（由 service 层在初始化后注入）
func (d *Dispatcher) SetModelChecker(mc ModelChecker) {
	d.modelChecker = mc
}

// RegisterProvider 注册 Provider
func (d *Dispatcher) RegisterProvider(modelName string, provider AIProvider) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.providers[modelName] = provider
}

// GetAllProviders 返回所有已注册的 Provider 副本（供 HealthCheck 使用）
func (d *Dispatcher) GetAllProviders() map[string]AIProvider {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string]AIProvider, len(d.providers))
	for k, v := range d.providers {
		result[k] = v
	}
	return result
}

// GetProvider 根据 model_name 获取对应 Provider
func (d *Dispatcher) GetProvider(modelName string) (AIProvider, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	provider, ok := d.providers[modelName]
	if !ok {
		return nil, fmt.Errorf("provider not found for model: %s", modelName)
	}
	return provider, nil
}

// GetProviderWithKey 获取 Provider 并注入平台默认 API Key
// 供搜索等非 Dispatch 流程使用，modelName 支持 "provider/model" 格式
func (d *Dispatcher) GetProviderWithKey(ctx context.Context, modelName string) (AIProvider, error) {
	providerName, _ := ParseModelName(modelName)
	provider, err := d.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Mock Provider 不需要 Key
	if _, ok := provider.(*MockProvider); ok {
		return provider, nil
	}

	// 获取平台默认 Key（使用 providerName）
	apiKey, err := d.keyStore.GetDefaultKey(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("no available API key for provider %s: %w", providerName, err)
	}

	// 动态注入 Key
	switch p := provider.(type) {
	case *KimiProvider:
		p.SetAPIKey(apiKey)
	case *ZhipuProvider:
		p.SetAPIKey(apiKey)
	case *QwenProvider:
		p.SetAPIKey(apiKey)
	case *DeepSeekProvider:
		p.SetAPIKey(apiKey)
	case *MinimaxProvider:
		p.SetAPIKey(apiKey)
	}

	return provider, nil
}

// CheckCapability 检查模型是否支持指定任务类型
// modelName 支持 "provider" 或 "provider/model" 格式
func (d *Dispatcher) CheckCapability(modelName string, taskType string) error {
	providerName, _ := ParseModelName(modelName)
	provider, err := d.GetProvider(providerName)
	if err != nil {
		return err
	}

	capabilities := provider.Capabilities()
	for _, cap := range capabilities {
		if cap == taskType {
			return nil
		}
	}

	return fmt.Errorf("model %s does not support task type %s", providerName, taskType)
}

// resolveKey 解析 API Key：优先用户 Key → 平台默认 Key
// providerName 为纯 provider 名称（不含模型版本）
func (d *Dispatcher) resolveKey(ctx context.Context, userID uint, providerName string) (string, error) {
	// Mock Provider 不需要 API Key
	if providerName == "mock" {
		return "mock-key", nil
	}

	// 优先查找用户自己的 Key
	userKey, err := d.keyStore.GetUserKey(ctx, userID, providerName)
	if err == nil && userKey != "" {
		return userKey, nil
	}

	// 回退到平台默认 Key
	defaultKey, err := d.keyStore.GetDefaultKey(ctx, providerName)
	if err != nil {
		return "", fmt.Errorf("no available API key for provider %s", providerName)
	}

	return defaultKey, nil
}

// Dispatch 分发任务：创建 AITask 记录 → goroutine 异步执行 → 更新状态 → 通知 WebSocket
func (d *Dispatcher) Dispatch(ctx context.Context, task *model.AITask) error {
	// 1. 解析 provider/model 格式，用 providerName 做能力检查
	providerName, _ := ParseModelName(task.ModelName)
	if err := d.CheckCapability(providerName, task.TaskType); err != nil {
		return err
	}

	// 2. 创建任务记录
	task.Status = model.TaskStatusPending
	if err := d.taskStore.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// 3. 异步执行任务
	go d.executeTask(context.Background(), task)

	return nil
}

// ExecuteSingle 同步执行单个任务（供 Orchestrator 编排引擎调用）
// 支持同 Provider 内版本降级；跨 Provider 降级由 makeNodeExecutor 处理
func (d *Dispatcher) ExecuteSingle(ctx context.Context, task *model.AITask) (interface{}, error) {
	return d.ExecuteSingleWithTools(ctx, task, nil)
}

// ExecuteSingleWithTools 同步执行单个任务，支持节点级工具注入
// overrideTools 不为 nil 时，与全局 toolRegistry 合并后传给 executor
func (d *Dispatcher) ExecuteSingleWithTools(ctx context.Context, task *model.AITask, overrideTools *ToolRegistry) (interface{}, error) {
	// 解析 provider/model 格式
	providerName, modelVersion := ParseModelName(task.ModelName)

	// 创建任务记录
	task.Status = model.TaskStatusRunning
	if err := d.taskStore.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 第一层：用原始模型尝试
	result, err := d.executeSingleAttempt(ctx, task, providerName, modelVersion, overrideTools)
	if err == nil {
		d.handleTaskSuccess(ctx, task, result)
		return result, nil
	}

	if !IsRetryableError(err) {
		d.handleTaskError(ctx, task, err)
		return nil, err
	}
	log.Printf("[dispatcher] ExecuteSingle task %d: %s/%s failed (%v), trying intra-provider fallback", task.ID, providerName, modelVersion, err)

	// 账号级错误，同 Provider 换模型无意义，直接返回让上层跨 Provider 降级
	if IsAccountLevelError(err) {
		log.Printf("[dispatcher] ExecuteSingle task %d: %s 账号级错误，跳过同 Provider 降级", task.ID, providerName)
		d.handleTaskError(ctx, task, err)
		return nil, err
	}

	// 第二层：同 Provider 内版本降级
	provider, providerErr := d.GetProvider(providerName)
	capForCheck := taskTypeToCapability(task.TaskType)
	if providerErr == nil {
		for _, fbModel := range provider.FallbackModels() {
			// 跳过已知不可用的子模型
			if d.modelChecker != nil && capForCheck != "" && !d.modelChecker.IsSubModelAvailable(providerName, fbModel, capForCheck) {
				log.Printf("[dispatcher] ExecuteSingle task %d: skip %s/%s (marked unavailable)", task.ID, providerName, fbModel)
				continue
			}
			log.Printf("[dispatcher] ExecuteSingle task %d: falling back to %s/%s", task.ID, providerName, fbModel)
			result, err = d.executeSingleAttempt(ctx, task, providerName, fbModel, overrideTools)
			if err == nil {
				d.handleTaskSuccess(ctx, task, result)
				return result, nil
			}
			if IsAccountLevelError(err) {
				log.Printf("[dispatcher] ExecuteSingle task %d: %s/%s 账号级错误，跳过剩余同 Provider 降级", task.ID, providerName, fbModel)
				break
			}
			if !IsRetryableError(err) {
				d.handleTaskError(ctx, task, err)
				return nil, err
			}
			log.Printf("[dispatcher] ExecuteSingle task %d: %s/%s failed (%v)", task.ID, providerName, fbModel, err)
		}
	}

	// 同 Provider 内所有模型均失败，返回错误（跨 Provider 降级由上层处理）
	d.handleTaskError(ctx, task, err)
	return nil, err
}

// executeSingleAttempt 执行单次尝试（指定 providerName 和 modelVersion）
// 返回 (result, error)，不处理任务状态更新
// overrideTools 不为 nil 时，与全局 toolRegistry 合并后传给 executor
func (d *Dispatcher) executeSingleAttempt(ctx context.Context, task *model.AITask, providerName, modelVersion string, overrideTools *ToolRegistry) (interface{}, error) {
	// 解析 API Key
	apiKey, err := d.resolveKey(ctx, task.UserID, providerName)
	if err != nil {
		return nil, err
	}

	// 获取 Provider
	provider, err := d.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// 动态设置 API Key（Mock Provider 不需要）
	if _, ok := provider.(*MockProvider); !ok {
		if kp, ok := provider.(*KimiProvider); ok {
			kp.SetAPIKey(apiKey)
		}
		if zp, ok := provider.(*ZhipuProvider); ok {
			zp.SetAPIKey(apiKey)
		}
		if qp, ok := provider.(*QwenProvider); ok {
			qp.SetAPIKey(apiKey)
		}
		if dp, ok := provider.(*DeepSeekProvider); ok {
			dp.SetAPIKey(apiKey)
		}
		if dp, ok := provider.(*MinimaxProvider); ok {
			dp.SetAPIKey(apiKey)
		}
	}

	// 查找并执行 Executor
	executor, err := d.executorRegistry.Get(task.TaskType)
	if err != nil {
		return nil, err
	}

	// 合并工具注册表：节点级覆盖 + 全局
	toolReg := d.toolRegistry
	if overrideTools != nil {
		toolReg = d.toolRegistry.Merge(overrideTools)
	}

	ec := &ExecContext{
		Provider:     provider,
		Task:         task,
		TokenMgr:     d.tokenMgr,
		ToolRegistry: toolReg,
		ModelVersion: modelVersion,
		GetProvider:  d.GetProvider,
	}

	return executor.Execute(ctx, ec)
}

// executeTask 异步执行任务，支持两层 fallback：
// 1. 同 Provider 内版本降级（FallbackModels）
// 2. 跨 Provider 降级（fallbackProviders）
func (d *Dispatcher) executeTask(ctx context.Context, task *model.AITask) {
	// 解析 provider/model 格式
	providerName, modelVersion := ParseModelName(task.ModelName)

	// 更新状态为 running
	task.Status = model.TaskStatusRunning
	_ = d.taskStore.UpdateTask(ctx, task)
	d.notifyTaskUpdate(task.UserID, task.ID, task.Status, nil, "")

	// 第一层：用原始模型尝试
	result, err := d.executeSingleAttempt(ctx, task, providerName, modelVersion, nil)
	if err == nil {
		d.handleTaskSuccess(ctx, task, result)
		return
	}

	if !IsRetryableError(err) {
		d.handleTaskError(ctx, task, err)
		return
	}
	log.Printf("[dispatcher] task %d: %s/%s failed (%v), trying intra-provider fallback", task.ID, providerName, modelVersion, err)

	// 第二层：同 Provider 内版本降级
	provider, providerErr := d.GetProvider(providerName)
	capability := taskTypeToCapability(task.TaskType)
	if providerErr == nil {
		for _, fbModel := range provider.FallbackModels() {
			// 跳过已知不可用的子模型
			if d.modelChecker != nil && capability != "" && !d.modelChecker.IsSubModelAvailable(providerName, fbModel, capability) {
				log.Printf("[dispatcher] task %d: skip %s/%s (marked unavailable)", task.ID, providerName, fbModel)
				continue
			}
			log.Printf("[dispatcher] task %d: falling back to %s/%s", task.ID, providerName, fbModel)
			result, err = d.executeSingleAttempt(ctx, task, providerName, fbModel, nil)
			if err == nil {
				d.handleTaskSuccess(ctx, task, result)
				return
			}
			if !IsRetryableError(err) {
				d.handleTaskError(ctx, task, err)
				return
			}
			log.Printf("[dispatcher] task %d: %s/%s failed (%v)", task.ID, providerName, fbModel, err)
		}
	}

	// 第三层：跨 Provider 降级（使用对方默认模型，跳过不可用的）
	for _, fbProvider := range d.fallbackProviders(providerName, task.TaskType) {
		log.Printf("[dispatcher] task %d: cross-provider fallback to %s", task.ID, fbProvider)
		result, err = d.executeSingleAttempt(ctx, task, fbProvider, "", nil)
		if err == nil {
			d.handleTaskSuccess(ctx, task, result)
			return
		}
		if !IsRetryableError(err) {
			d.handleTaskError(ctx, task, err)
			return
		}
		log.Printf("[dispatcher] task %d: %s failed (%v)", task.ID, fbProvider, err)
	}

	// 所有降级均失败
	d.handleTaskError(ctx, task, fmt.Errorf("all fallback models exhausted: %w", err))
}

// handleTaskSuccess 处理任务成功，提取 token 统计写入 AITask
func (d *Dispatcher) handleTaskSuccess(ctx context.Context, task *model.AITask, result interface{}) {
	task.Status = model.TaskStatusCompleted

	// 从 result map 中提取 token 统计
	if m, ok := result.(map[string]interface{}); ok {
		if u, exists := m["usage"]; exists {
			if usageBytes, err := json.Marshal(u); err == nil {
				var usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}
				if json.Unmarshal(usageBytes, &usage) == nil {
					task.PromptTokens = usage.PromptTokens
					task.CompletionTokens = usage.CompletionTokens
					task.TotalTokens = usage.TotalTokens
				}
			}
			delete(m, "usage") // 从 Result JSON 中移除 usage，避免冗余存储
		}
	}

	resultJSON, _ := json.Marshal(result)
	task.Result = string(resultJSON)
	_ = d.taskStore.UpdateTask(ctx, task)

	d.notifyTaskUpdate(task.UserID, task.ID, task.Status, result, "")

	// 任务完成回调（刷新 token 缓存等）
	if d.OnTaskCompleted != nil {
		d.OnTaskCompleted(task)
	}
}

// handleTaskError 处理任务失败
func (d *Dispatcher) handleTaskError(ctx context.Context, task *model.AITask, err error) {
	task.Status = model.TaskStatusFailed
	task.ErrorMsg = err.Error()
	_ = d.taskStore.UpdateTask(ctx, task)

	d.notifyTaskUpdate(task.UserID, task.ID, task.Status, nil, err.Error())
}

// notifyTaskUpdate 通知任务状态更新
func (d *Dispatcher) notifyTaskUpdate(userID uint, taskID uint, status string, result interface{}, errorMsg string) {
	if d.notifier == nil {
		return
	}

	message := TaskResult{
		TaskID: taskID,
		Status: status,
		Result: result,
		Error:  errorMsg,
	}

	_ = d.notifier.NotifyUser(userID, message)
}

// CancelTask 取消任务（仅更新状态，不中断执行）
func (d *Dispatcher) CancelTask(ctx context.Context, taskID uint) error {
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status == model.TaskStatusCompleted || task.Status == model.TaskStatusFailed {
		return errors.New("cannot cancel completed or failed task")
	}

	task.Status = model.TaskStatusCancelled
	return d.taskStore.UpdateTask(ctx, task)
}

// IsAccountLevelError 判断是否为账号级别错误（计费问题、账号封禁等）
// 此类错误同 Provider 内换模型无意义，应直接跨 Provider 降级
// 注意：FreeTierOnly / AllocationQuota 是模型级配额限制（同 Key 下不同模型有独立额度），
// 不属于账号级错误，应允许同 Provider 内降级到其他模型
func IsAccountLevelError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	patterns := []string{
		"billing",
		"insufficient_quota",
		"account",
	}
	for _, p := range patterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

// IsRetryableError 判断错误是否可重试（限流、服务端临时错误、额度耗尽等）
// 匹配后应继续尝试下一个降级模型
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"429",
		"rate limit",
		"速率限制",
		"500",
		"502",
		"503",
		"timeout",
		"connection refused",
		"no available api key",
		"provider not found",
		"allocationquota",
		"quota",
		"exhausted",
		"insufficient",
		"billing",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// fallbackProviders 返回跨 Provider 降级列表（排除当前 Provider）
// 从已注册的 providers map 中动态获取，不再硬编码
func (d *Dispatcher) fallbackProviders(current string, taskType string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 将 taskType 映射到 capability（用于可用性检查）
	capability := taskTypeToCapability(taskType)

	var result []string
	for name := range d.providers {
		if name == current {
			continue
		}
		// 如果注入了 modelChecker，跳过不可用的 Provider
		if d.modelChecker != nil && capability != "" && !d.modelChecker.IsModelAvailable(name, capability) {
			log.Printf("[dispatcher] fallback skip %s: not available for %s", name, capability)
			continue
		}
		result = append(result, name)
	}
	return result
}

// taskTypeToCapability 将任务类型映射到模型能力（用于可用性检查）
func taskTypeToCapability(taskType string) string {
	switch taskType {
	case "text_gen", "text_polish", "storyboard",
		"chapter_summary_polish", "chapter_polish", "chapter_expand", "chapter_continue",
		"outline_generate", "outline_title_polish", "outline_summary_polish", "outline_summary_expand",
		"outline_generate_characters",
		"butler_generate_topic", "butler_generate_storyline", "butler_generate_characters",
		"butler_storyline_draft", "butler_storyline_review",
		"butler_characters_draft", "butler_characters_review",
		"knowledge_extract", "overview_extract",
		"memory_feature_extract", "memory_prompt_compile", "memory_quality_eval",
		"memory_review_quality", "memory_review_compliance", "memory_review_decision",
		"revision_analysis", "revision_planning",
		"character_adjust":
		return "text_gen"
	case "image_gen":
		return "image_gen"
	case "image_edit":
		return "image_edit"
	default:
		return "text_gen"
	}
}

// GenerateTextWithFallback 同步调用 AI 生成文本，自带降级机制
// 供 service 层直接调用 AI 的场景使用（如事实采集、知识提取等）
func (d *Dispatcher) GenerateTextWithFallback(ctx context.Context, preferredModel string, req *TextRequest) (*TextResponse, error) {
	providerName, modelVersion := ParseModelName(preferredModel)

	// 第一层：用首选模型尝试
	resp, err := d.generateTextAttempt(ctx, providerName, modelVersion, req)
	if err == nil {
		return resp, nil
	}
	if !IsRetryableError(err) {
		return nil, err
	}
	log.Printf("[dispatcher] GenerateTextWithFallback: %s/%s failed (%v), trying fallback", providerName, modelVersion, err)

	// 如果有 modelChecker，使用 DB 驱动的降级链
	if d.modelChecker != nil {
		capability := taskTypeToCapability("text_gen")
		chain := d.modelChecker.GetFallbackChain(capability, providerName, modelVersion)
		for _, entry := range chain {
			log.Printf("[dispatcher] GenerateTextWithFallback: fallback to %s/%s", entry.Provider, entry.ModelName)
			resp, err = d.generateTextAttempt(ctx, entry.Provider, entry.ModelName, req)
			if err == nil {
				return resp, nil
			}
			if !IsRetryableError(err) {
				return nil, err
			}
		}
		return nil, fmt.Errorf("all fallback models exhausted: %w", err)
	}

	// 兜底：无 modelChecker 时使用旧逻辑
	if !IsAccountLevelError(err) {
		provider, providerErr := d.GetProvider(providerName)
		if providerErr == nil {
			for _, fbModel := range provider.FallbackModels() {
				log.Printf("[dispatcher] GenerateTextWithFallback: falling back to %s/%s", providerName, fbModel)
				resp, err = d.generateTextAttempt(ctx, providerName, fbModel, req)
				if err == nil {
					return resp, nil
				}
				if IsAccountLevelError(err) {
					break
				}
				if !IsRetryableError(err) {
					return nil, err
				}
			}
		}
	}

	for _, fbProvider := range d.fallbackProviders(providerName, "text_gen") {
		log.Printf("[dispatcher] GenerateTextWithFallback: cross-provider fallback to %s", fbProvider)
		resp, err = d.generateTextAttempt(ctx, fbProvider, "", req)
		if err == nil {
			return resp, nil
		}
		if !IsRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("all fallback models exhausted: %w", err)
}

// generateTextAttempt 单次 GenerateText 尝试（注入 key + 设置模型版本）
func (d *Dispatcher) generateTextAttempt(ctx context.Context, providerName, modelVersion string, req *TextRequest) (*TextResponse, error) {
	provider, err := d.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	if _, ok := provider.(*MockProvider); !ok {
		apiKey, keyErr := d.resolveKey(ctx, 0, providerName)
		if keyErr != nil {
			return nil, keyErr
		}
		switch p := provider.(type) {
		case *KimiProvider:
			p.SetAPIKey(apiKey)
		case *ZhipuProvider:
			p.SetAPIKey(apiKey)
		case *QwenProvider:
			p.SetAPIKey(apiKey)
		case *DeepSeekProvider:
			p.SetAPIKey(apiKey)
		case *MinimaxProvider:
			p.SetAPIKey(apiKey)

		}
	}
	if modelVersion != "" {
		req = &TextRequest{
			Prompt:       req.Prompt,
			CharacterCtx: req.CharacterCtx,
			MaxTokens:    req.MaxTokens,
			Temperature:  req.Temperature,
			History:      req.History,
			Tools:        req.Tools,
			Extra:        req.Extra,
			Model:        modelVersion,
		}
	}
	return provider.GenerateText(ctx, req)
}
