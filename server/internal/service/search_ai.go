// server/internal/service/search_ai.go
// AI 推荐 Provider — 兜底方案，复用已有 AI 模型能力生成小说推荐
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

// AISearchProvider 利用 AI 模型生成小说推荐（兜底 Provider）
type AISearchProvider struct {
	dispatcher    *agent.Dispatcher
	modelName     string
	modelRegistry *ModelRegistryService
}

// SetModelRegistry 注入模型注册中心，用于降级链
func (p *AISearchProvider) SetModelRegistry(mr *ModelRegistryService) {
	p.modelRegistry = mr
}

// NewAISearchProvider 创建 AI 搜索 Provider
func NewAISearchProvider(dispatcher *agent.Dispatcher, modelName string) *AISearchProvider {
	return &AISearchProvider{dispatcher: dispatcher, modelName: modelName}
}

func (p *AISearchProvider) Name() string { return "ai" }

func (p *AISearchProvider) Available() bool {
	_, err := p.dispatcher.GetProvider(p.modelName)
	return err == nil
}

// aiNovelItem AI 返回的单本小说结构
type aiNovelItem struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Category string `json:"category"`
	Intro    string `json:"intro"`
}

func (p *AISearchProvider) Search(ctx context.Context, keyword string, limit int) ([]NovelSearchResult, error) {
	// 构建降级链：主模型 + fallback
	chain := []string{p.modelName}
	if p.modelRegistry != nil {
		chain = append(chain, p.modelRegistry.GetFallbackProviders(0, p.modelName, model.CapTextGen)...)
	}

	prompt := fmt.Sprintf(
		`请根据关键词"%s"，推荐%d本相关的中文小说。
请严格按以下 JSON 数组格式返回，不要包含其他文字：
[{"title":"书名","author":"作者","category":"分类","intro":"100字以内简介"}]`,
		keyword, limit,
	)

	var lastErr error
	for _, modelName := range chain {
		provider, err := p.dispatcher.GetProviderWithKey(ctx, modelName)
		if err != nil {
			log.Printf("[AISearch] %s 不可用: %v", modelName, err)
			lastErr = err
			continue
		}

		resp, err := provider.GenerateText(ctx, &agent.TextRequest{
			Prompt:      prompt,
			MaxTokens:   2048,
			Temperature: 0.7,
		})
		if err != nil {
			log.Printf("[AISearch] %s 生成失败: %v", modelName, err)
			lastErr = err
			continue
		}

		results, err := parseAISearchResults(resp.Content)
		if err != nil {
			log.Printf("[AISearch] %s 解析失败: %v", modelName, err)
			lastErr = err
			continue
		}
		return results, nil
	}

	return nil, fmt.Errorf("AI 搜索所有模型均失败: %w", lastErr)
}

// parseAISearchResults 从 AI 响应内容中提取并解析小说推荐结果
func parseAISearchResults(content string) ([]NovelSearchResult, error) {
	// 尝试找到 JSON 数组的起止位置
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("AI 返回格式异常，无法解析 JSON")
	}
	jsonStr := content[start : end+1]

	var items []aiNovelItem
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		return nil, fmt.Errorf("解析 AI 推荐结果失败: %w", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("AI 未返回有效推荐")
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
