// server/internal/service/model_registry.go
package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// ModelInfo 模型信息（API 返回结构）
type ModelInfo struct {
	Value     string      `json:"value"`
	Label     string      `json:"label"`
	Provider  string      `json:"provider"`
	Available bool        `json:"available"`
	KeySource string      `json:"key_source"`
	LatencyMs int         `json:"latency_ms"`
	SubModels []ModelInfo `json:"sub_models,omitempty"`
}

// AddModelRequest 新增模型请求
type AddModelRequest struct {
	Provider   string `json:"provider"`
	ModelName  string `json:"model_name"`
	Capability string `json:"capability"`
}

// TestModelResult 单模型测试结果
type TestModelResult struct {
	Available bool   `json:"available"`
	LatencyMs int    `json:"latency_ms"`
	Error     string `json:"error"`
}

// ModelStatusDetail 模型状态详情（调试面板用）
type ModelStatusDetail struct {
	ID         uint    `json:"id"`
	Provider   string  `json:"provider"`
	ModelName  string  `json:"model_name"`
	Capability string  `json:"capability"`
	Available  bool    `json:"available"`
	LatencyMs  int     `json:"latency_ms"`
	Priority   int     `json:"priority"`
	LastCheck  *string `json:"last_check"`
	LastError  string  `json:"last_error"`
}

// ModelRegistryService 模型注册中心服务
type ModelRegistryService struct {
	dao        *dao.AIModelStatusDAO
	keyService *APIKeyService
	mu         sync.RWMutex
	cache      map[string]*model.AIModelStatus // key: "provider:model_name:capability"
	stopCh     chan struct{}
}

// NewModelRegistryService 创建 ModelRegistryService 实例
func NewModelRegistryService(dao *dao.AIModelStatusDAO, keyService *APIKeyService) *ModelRegistryService {
	return &ModelRegistryService{
		dao:        dao,
		keyService: keyService,
		cache:      make(map[string]*model.AIModelStatus),
		stopCh:     make(chan struct{}),
	}
}

// cacheKey 生成缓存键
func cacheKey(provider, modelName, capability string) string {
	return provider + ":" + modelName + ":" + capability
}

// SeedFromMeta 启动时从 DefaultProviders 静态元数据 seed 到 DB（仅插入不存在的记录）
func (s *ModelRegistryService) SeedFromMeta(ctx context.Context) error {
	for _, pm := range model.DefaultProviders {
		for _, cap := range pm.Capabilities {
			// seed provider 级别的默认模型
			status := &model.AIModelStatus{
				Provider:    pm.Provider,
				ModelName:   "",
				Capability:  cap,
				IsAvailable: true,
			}
			if err := s.dao.Upsert(ctx, status); err != nil {
				return fmt.Errorf("seed %s/%s failed: %w", pm.Provider, cap, err)
			}
			// seed 具体子模型
			for _, m := range pm.Models {
				if m.ModelName == "" {
					continue // 默认模型已在上面 seed
				}
				sub := &model.AIModelStatus{
					Provider:    pm.Provider,
					ModelName:   m.ModelName,
					Capability:  cap,
					IsAvailable: true,
				}
				if err := s.dao.Upsert(ctx, sub); err != nil {
					return fmt.Errorf("seed %s/%s/%s failed: %w", pm.Provider, m.ModelName, cap, err)
				}
			}
		}
	}
	log.Println("[ModelRegistry] seed from meta completed")
	return nil
}

// LoadCache 从 DB 加载到内存缓存
func (s *ModelRegistryService) LoadCache(ctx context.Context) error {
	all, err := s.dao.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("load cache failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*model.AIModelStatus, len(all))
	for _, item := range all {
		s.cache[cacheKey(item.Provider, item.ModelName, item.Capability)] = item
	}
	log.Printf("[ModelRegistry] cache loaded: %d entries", len(s.cache))
	return nil
}

// GetAvailableModels 返回用户可用的模型列表
func (s *ModelRegistryService) GetAvailableModels(ctx context.Context, userID uint, capability string) ([]ModelInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 按 Provider 分组
	type providerGroup struct {
		meta      model.ProviderMeta
		available bool
		latencyMs int
		keySource string
		subModels []ModelInfo
	}
	groups := make(map[string]*providerGroup)

	// 初始化分组（按 DefaultProviders 顺序）
	for _, pm := range model.DefaultProviders {
		groups[pm.Provider] = &providerGroup{meta: pm}
	}

	// 填充缓存数据
	for _, item := range s.cache {
		if capability != "" && item.Capability != capability {
			continue
		}
		g, ok := groups[item.Provider]
		if !ok {
			continue
		}
		if item.ModelName == "" {
			// provider 级别状态
			g.available = item.IsAvailable
			g.latencyMs = item.LatencyMs
		} else {
			// 子模型：始终加入，Available 取实际探测结果
			g.subModels = append(g.subModels, ModelInfo{
				Value:     item.Provider + "/" + item.ModelName,
				Label:     s.getModelDisplayName(item.Provider, item.ModelName),
				Provider:  item.Provider,
				Available: item.IsAvailable,
				LatencyMs: item.LatencyMs,
			})
		}
	}

	// 检查用户是否有自有 Key
	for provider, g := range groups {
		if userID > 0 {
			key, err := s.keyService.GetUserKey(ctx, userID, provider)
			if err == nil && key != "" {
				g.keySource = "user"
			}
		}
		if g.keySource == "" {
			key, err := s.keyService.GetDefaultKey(ctx, provider)
			if err == nil && key != "" {
				g.keySource = "platform"
			}
		}
	}

	// 按 Priority 排序输出，始终返回所有 Provider（不可用的 Available=false）
	var result []ModelInfo
	for _, pm := range model.DefaultProviders {
		g := groups[pm.Provider]
		if g == nil {
			continue
		}
		info := ModelInfo{
			Value:     pm.Provider,
			Label:     pm.DisplayName,
			Provider:  pm.Provider,
			Available: g.available,
			KeySource: g.keySource,
			LatencyMs: g.latencyMs,
			SubModels: g.subModels,
		}
		result = append(result, info)
	}

	return result, nil
}

// GetDefaultModel 返回指定能力的默认模型（按 Priority 排序，取第一个可用的）
func (s *ModelRegistryService) GetDefaultModel(capability string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, pm := range model.DefaultProviders {
		key := cacheKey(pm.Provider, "", capability)
		if status, ok := s.cache[key]; ok && status.IsAvailable {
			return pm.Provider
		}
	}
	// 全部不可用时返回第一个 provider 作为兜底
	if len(model.DefaultProviders) > 0 {
		return model.DefaultProviders[0].Provider
	}
	return "qwen"
}

// IsModelAvailable 校验指定 Provider 是否可用（用于用户指定模型时的前置校验）
// 不可用时返回 false，调用方可据此降级到默认模型或拒绝请求
func (s *ModelRegistryService) IsModelAvailable(provider, capability string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := cacheKey(provider, "", capability)
	if status, ok := s.cache[key]; ok {
		return status.IsAvailable
	}
	// 缓存中无记录，视为不可用
	return false
}

// IsSubModelAvailable 检查指定 Provider 下的子模型是否可用
func (s *ModelRegistryService) IsSubModelAvailable(provider, modelName, capability string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := cacheKey(provider, modelName, capability)
	if status, ok := s.cache[key]; ok {
		return status.IsAvailable
	}
	// 缓存中无记录，回退检查 provider 级别状态
	providerKey := cacheKey(provider, "", capability)
	if status, ok := s.cache[providerKey]; ok {
		return status.IsAvailable
	}
	return false
}

// ResolveModel 解析最终使用的模型：用户指定的模型可用则用，否则降级到默认模型
func (s *ModelRegistryService) ResolveModel(userModel, capability string) string {
	if userModel != "" {
		// 提取 provider（支持 "provider" 和 "provider/model" 两种格式）
		provider := userModel
		if idx := strings.Index(userModel, "/"); idx > 0 {
			provider = userModel[:idx]
		}
		if s.IsModelAvailable(provider, capability) {
			return userModel
		}
		log.Printf("[ModelRegistry] 用户指定模型 %s 不可用，降级到默认模型", userModel)
	}
	return s.GetDefaultModel(capability)
}

// GetFallbackProviders 返回降级 Provider 列表（排除 primary 自身）
// 有个人 Key 的 provider 优先排前面（组内按 Priority），无个人 Key 的排后面
func (s *ModelRegistryService) GetFallbackProviders(userID uint, primary string, capability string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 分两组：有个人 Key 的和没有的
	var withUserKey, withoutUserKey []string
	for _, pm := range model.DefaultProviders {
		if pm.Provider == primary {
			continue
		}
		key := cacheKey(pm.Provider, "", capability)
		if status, ok := s.cache[key]; ok && !status.IsAvailable {
			continue
		}

		// 检查用户是否有该 provider 的个人 Key
		hasUserKey := false
		if userID > 0 && s.keyService != nil {
			ctx := context.Background()
			uk, err := s.keyService.GetUserKey(ctx, userID, pm.Provider)
			if err == nil && uk != "" {
				hasUserKey = true
			}
		}

		if hasUserKey {
			withUserKey = append(withUserKey, pm.Provider)
		} else {
			withoutUserKey = append(withoutUserKey, pm.Provider)
		}
	}

	// 有个人 Key 的排前面，组内已按 DefaultProviders 的 Priority 顺序
	return append(withUserKey, withoutUserKey...)
}

// GetFallbackChain 实现 agent.ModelChecker 接口
// 从 DB 缓存获取指定能力的降级链，按优先级排序，排除当前模型
func (s *ModelRegistryService) GetFallbackChain(capability, currentProvider, currentModel string) []agent.FallbackEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 收集所有可用的子模型，按 Priority、LatencyMs 排序
	type candidate struct {
		provider  string
		modelName string
		priority  int
		latencyMs int
	}
	var candidates []candidate

	for _, item := range s.cache {
		if item.Capability != capability || !item.IsAvailable {
			continue
		}
		if item.ModelName == "" {
			continue // 跳过 provider 级别记录，只取具体子模型
		}
		// 排除当前正在使用的模型
		if item.Provider == currentProvider && item.ModelName == currentModel {
			continue
		}
		candidates = append(candidates, candidate{
			provider:  item.Provider,
			modelName: item.ModelName,
			priority:  item.Priority,
			latencyMs: item.LatencyMs,
		})
	}

	// 按 priority ASC, latency_ms ASC 排序
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].priority < candidates[i].priority ||
				(candidates[j].priority == candidates[i].priority && candidates[j].latencyMs < candidates[i].latencyMs) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	result := make([]agent.FallbackEntry, 0, len(candidates))
	for _, c := range candidates {
		result = append(result, agent.FallbackEntry{
			Provider:  c.provider,
			ModelName: c.modelName,
		})
	}
	return result
}

// HealthCheckAll 探测所有 Provider 及其子模型的可用性
func (s *ModelRegistryService) HealthCheckAll(ctx context.Context, providers map[string]agent.AIProvider) error {
	type probeJob struct {
		providerName string
		provider     agent.AIProvider
		apiKey       string
		modelName    string
		capability   string
	}

	// 收集所有探测任务
	var jobs []probeJob
	for _, pm := range model.DefaultProviders {
		provider, ok := providers[pm.Provider]
		if !ok {
			continue
		}
		apiKey := s.resolveKeyForProbe(ctx, pm.Provider)

		for _, cap := range pm.Capabilities {
			// Provider 级别（默认模型）
			jobs = append(jobs, probeJob{pm.Provider, provider, apiKey, "", cap})
			// 子模型
			for _, m := range pm.Models {
				if m.ModelName == "" {
					continue
				}
				jobs = append(jobs, probeJob{pm.Provider, provider, apiKey, m.ModelName, cap})
			}
		}
	}

	// 并发探测，同一 Provider 内串行（共用 API Key，避免并发竞争）
	// 按 Provider 分组
	grouped := make(map[string][]probeJob)
	for _, j := range jobs {
		grouped[j.providerName] = append(grouped[j.providerName], j)
	}

	var wg sync.WaitGroup
	for _, providerJobs := range grouped {
		wg.Add(1)
		go func(pJobs []probeJob) {
			defer wg.Done()
			for _, j := range pJobs {
				available, latencyMs, lastError := s.probeModel(ctx, j.provider, j.apiKey, j.modelName, j.capability)
				if err := s.dao.UpdateAvailability(ctx, j.providerName, j.modelName, j.capability, available, latencyMs, lastError); err != nil {
					log.Printf("[ModelRegistry] update availability failed: %s/%s/%s: %v", j.providerName, j.modelName, j.capability, err)
				}
			}
		}(providerJobs)
	}
	wg.Wait()

	// 刷新缓存
	return s.LoadCache(ctx)
}

// resolveKeyForProbe 获取探测用的 API Key（平台默认 Key）
func (s *ModelRegistryService) resolveKeyForProbe(ctx context.Context, provider string) string {
	if s.keyService == nil {
		return ""
	}
	key, err := s.keyService.GetDefaultKey(ctx, provider)
	if err != nil {
		log.Printf("[ModelRegistry] resolve key for probe %s failed: %v", provider, err)
		return ""
	}
	return key
}

// probeModel 探测单个模型的指定能力
// modelName 为空表示探测 Provider 默认模型
func (s *ModelRegistryService) probeModel(ctx context.Context, provider agent.AIProvider, apiKey, modelName, capability string) (available bool, latencyMs int, lastError string) {
	// 注入 API Key
	if apiKey != "" {
		s.setProviderKey(provider, apiKey)
	}

	start := time.Now()

	switch capability {
	case model.CapTextGen, model.CapTextPolish:
		_, err := provider.GenerateText(ctx, &agent.TextRequest{
			Model:     modelName,
			Prompt:    "hi",
			MaxTokens: 1,
		})
		latencyMs = int(time.Since(start).Milliseconds())
		if err != nil {
			return false, latencyMs, err.Error()
		}
		return true, latencyMs, ""

	case model.CapEmbedding:
		_, err := provider.Embedding(ctx, &agent.EmbeddingRequest{
			Texts: []string{"test"},
		})
		latencyMs = int(time.Since(start).Milliseconds())
		if err != nil {
			return false, latencyMs, err.Error()
		}
		return true, latencyMs, ""

	default:
		// image_gen 等高成本能力，跳过实际探测，标记为可用
		return true, 0, ""
	}
}

// setProviderKey 动态设置 Provider 的 API Key
func (s *ModelRegistryService) setProviderKey(provider agent.AIProvider, apiKey string) {
	switch p := provider.(type) {
	case *agent.QwenProvider:
		p.SetAPIKey(apiKey)
	case *agent.ZhipuProvider:
		p.SetAPIKey(apiKey)
	case *agent.DeepSeekProvider:
		p.SetAPIKey(apiKey)
	case *agent.KimiProvider:
		p.SetAPIKey(apiKey)
	case *agent.MinimaxProvider:
		p.SetAPIKey(apiKey)
	}
}

// StartPeriodicCheck 启动定时探测 goroutine
func (s *ModelRegistryService) StartPeriodicCheck(ctx context.Context, interval time.Duration, providers map[string]agent.AIProvider) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("[ModelRegistry] periodic health check started")
				if err := s.HealthCheckAll(ctx, providers); err != nil {
					log.Printf("[ModelRegistry] periodic health check failed: %v", err)
				}
				log.Println("[ModelRegistry] periodic health check completed")
			case <-s.stopCh:
				log.Println("[ModelRegistry] periodic check stopped")
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	log.Printf("[ModelRegistry] periodic check started, interval=%v", interval)
}

// GetAllModelStatus 返回所有模型的完整状态详情（调试面板用）
func (s *ModelRegistryService) GetAllModelStatus(ctx context.Context) ([]ModelStatusDetail, error) {
	all, err := s.dao.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all model status failed: %w", err)
	}

	result := make([]ModelStatusDetail, 0, len(all))
	for _, item := range all {
		detail := ModelStatusDetail{
			ID:         item.ID,
			Provider:   item.Provider,
			ModelName:  item.ModelName,
			Capability: item.Capability,
			Available:  item.IsAvailable,
			LatencyMs:  item.LatencyMs,
			Priority:   item.Priority,
			LastError:  item.LastError,
		}
		if item.LastCheck != nil {
			t := item.LastCheck.Format("2006-01-02 15:04:05")
			detail.LastCheck = &t
		}
		result = append(result, detail)
	}
	return result, nil
}

// AddModel 新增模型记录到 DB 并刷新缓存
func (s *ModelRegistryService) AddModel(ctx context.Context, req *AddModelRequest) error {
	status := &model.AIModelStatus{
		Provider:    req.Provider,
		ModelName:   req.ModelName,
		Capability:  req.Capability,
		IsAvailable: true,
	}
	if err := s.dao.Upsert(ctx, status); err != nil {
		return fmt.Errorf("add model failed: %w", err)
	}
	return s.LoadCache(ctx)
}

// DeleteModel 按 ID 删除模型记录并刷新缓存
func (s *ModelRegistryService) DeleteModel(ctx context.Context, id uint) error {
	if err := s.dao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete model failed: %w", err)
	}
	return s.LoadCache(ctx)
}

// UpdateModelPriority 更新模型优先级并刷新缓存
func (s *ModelRegistryService) UpdateModelPriority(ctx context.Context, id uint, priority int) error {
	if err := s.dao.UpdatePriority(ctx, id, priority); err != nil {
		return fmt.Errorf("update priority failed: %w", err)
	}
	return s.LoadCache(ctx)
}

// TestSingleModel 测试单个模型的可用性，结果写入 DB 并刷新缓存
func (s *ModelRegistryService) TestSingleModel(ctx context.Context, provider, modelName, capability string, providers map[string]agent.AIProvider) (*TestModelResult, error) {
	p, ok := providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	apiKey := s.resolveKeyForProbe(ctx, provider)
	available, latencyMs, lastError := s.probeModel(ctx, p, apiKey, modelName, capability)

	// 写入 DB
	if err := s.dao.UpdateAvailability(ctx, provider, modelName, capability, available, latencyMs, lastError); err != nil {
		log.Printf("[ModelRegistry] update availability failed: %s/%s/%s: %v", provider, modelName, capability, err)
	}

	// 刷新缓存
	_ = s.LoadCache(ctx)

	return &TestModelResult{
		Available: available,
		LatencyMs: latencyMs,
		Error:     lastError,
	}, nil
}

// Stop 停止定时探测
func (s *ModelRegistryService) Stop() {
	close(s.stopCh)
}

// getModelDisplayName 获取模型显示名称
func (s *ModelRegistryService) getModelDisplayName(provider, modelName string) string {
	for _, pm := range model.DefaultProviders {
		if pm.Provider == provider {
			for _, m := range pm.Models {
				if m.ModelName == modelName {
					return m.DisplayName
				}
			}
		}
	}
	return modelName
}
