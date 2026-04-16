// server/internal/service/search_ai_web.go
// AI 联网搜索 Provider — 利用模型原生联网搜索能力获取真实小说信息
// 内置多模型降级：主力模型失败后自动尝试备选模型
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/model"
)

// AIWebSearchProvider 利用 AI 模型的联网搜索能力搜索小说
type AIWebSearchProvider struct {
	dispatcher *agent.Dispatcher
	// 模型降级链：按优先级排列，第一个为主力模型
	modelChain    []string
	modelRegistry *ModelRegistryService
}

// NewAIWebSearchProvider 创建 AI 联网搜索 Provider
// primaryModel 为主力模型，自动构建降级链
func NewAIWebSearchProvider(dispatcher *agent.Dispatcher, primaryModel string) *AIWebSearchProvider {
	// 所有支持联网搜索的模型
	allModels := []string{"qwen", "zhipu", "kimi", "deepseek"}

	// 构建降级链：主力模型在前，其余按默认顺序追加
	chain := []string{primaryModel}
	for _, m := range allModels {
		if m != primaryModel {
			chain = append(chain, m)
		}
	}

	return &AIWebSearchProvider{dispatcher: dispatcher, modelChain: chain}
}

// SetModelRegistry 注入模型注册服务
func (p *AIWebSearchProvider) SetModelRegistry(mr *ModelRegistryService) {
	p.modelRegistry = mr
}

func (p *AIWebSearchProvider) Name() string { return "ai_web" }

func (p *AIWebSearchProvider) Available() bool {
	// 只要降级链中有任一模型可用即可
	for _, model := range p.modelChain {
		if _, err := p.dispatcher.GetProvider(model); err == nil {
			return true
		}
	}
	return false
}

func (p *AIWebSearchProvider) Search(ctx context.Context, keyword string, limit int) ([]NovelSearchResult, error) {
	// 动态获取降级链：优先使用 modelRegistry，兜底用构造时的 modelChain
	chain := p.modelChain
	if p.modelRegistry != nil && len(chain) > 0 {
		primary := chain[0]
		fallbacks := p.modelRegistry.GetFallbackChain(0, primary, model.CapTextGen)
		chain = append([]string{primary}, fallbacks...)
	}

	var lastErr error

	// 沿降级链逐个尝试
	for _, modelName := range chain {
		provider, err := p.dispatcher.GetProviderWithKey(ctx, modelName)
		if err != nil {
			log.Printf("[AIWebSearch] %s 不可用: %v, 尝试下一个模型...", modelName, err)
			lastErr = err
			continue
		}

		results, err := p.searchWithModel(ctx, provider, modelName, keyword, limit)
		if err != nil {
			log.Printf("[AIWebSearch] %s 搜索失败: %v, 尝试下一个模型...", modelName, err)
			lastErr = err
			continue
		}

		log.Printf("[AIWebSearch] %s 搜索成功, 关键词=%q, 结果数=%d", modelName, keyword, len(results))
		return results, nil
	}

	return nil, fmt.Errorf("所有 AI 模型均搜索失败, 最后错误: %w", lastErr)
}

// searchWithModel 使用指定模型执行联网搜索
func (p *AIWebSearchProvider) searchWithModel(
	ctx context.Context, provider agent.AIProvider, modelName string,
	keyword string, limit int,
) ([]NovelSearchResult, error) {
	prompt := fmt.Sprintf(
		`请联网搜索关键词"%s"相关的热门中文小说，找到%d本真实存在的小说。
要求：
1. 必须是真实存在、可以在网上搜到的小说
2. 提供准确的书名、作者、分类和详细简介
3. 简介需要包含世界观设定和主要剧情走向，至少100字

请严格按以下 JSON 数组格式返回，不要包含其他文字：
[{"title":"书名","author":"作者","category":"分类","intro":"详细简介（含世界观和剧情）"}]`,
		keyword, limit,
	)

	req := &agent.TextRequest{
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.3,
		Tools:       buildWebSearchTools(modelName),
	}

	// 通义千问通过 Extra 字段传递 enable_search
	if strings.Contains(modelName, "qwen") {
		req.Extra = map[string]string{"enable_search": "true"}
	}

	resp, err := provider.GenerateText(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI 联网搜索失败: %w", err)
	}

	return parseWebSearchResults(resp.Content)
}

// buildWebSearchTools 根据模型名称生成对应的联网搜索 Tools 参数
func buildWebSearchTools(modelName string) []map[string]any {
	switch {
	case strings.Contains(modelName, "zhipu"):
		return []map[string]any{
			{"type": "web_search", "web_search": map[string]any{"enable": true}},
		}
	case strings.Contains(modelName, "kimi"):
		return []map[string]any{
			{"type": "builtin_function", "function": map[string]any{"name": "web_search"}},
		}
	default:
		// 通义千问通过 Extra 传递，不需要 Tools
		return nil
	}
}

// parseWebSearchResults 从 AI 响应中解析结构化小说列表
func parseWebSearchResults(content string) ([]NovelSearchResult, error) {
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("AI 返回格式异常，无法解析 JSON")
	}
	jsonStr := content[start : end+1]

	var items []aiNovelItem
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		return nil, fmt.Errorf("解析 AI 联网搜索结果失败: %w", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("AI 联网搜索未返回有效结果")
	}

	results := make([]NovelSearchResult, 0, len(items))
	for _, item := range items {
		result := NovelSearchResult{
			Title:    item.Title,
			Author:   item.Author,
			Category: item.Category,
			Intro:    item.Intro,
		}
		ExtractNovelInfo(&result)
		results = append(results, result)
	}
	return results, nil
}
