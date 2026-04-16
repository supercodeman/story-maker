// server/internal/service/novel_search.go
package service

import (
	"context"
	"log"
	"regexp"
	"strings"
)

// NovelSearchProvider 小说搜索引擎接口
// 遵循 ISP 原则：仅定义搜索所需的最小方法集
type NovelSearchProvider interface {
	// Name 返回引擎名称，用于日志和降级提示
	Name() string
	// Search 执行搜索，返回结果列表
	Search(ctx context.Context, keyword string, limit int) ([]NovelSearchResult, error)
	// Available 检查该引擎是否可用（如 API Key 是否配置）
	Available() bool
}

// NovelSearchResult 热门小说搜索结果
type NovelSearchResult struct {
	Title      string `json:"title"`
	Author     string `json:"author"`
	Category   string `json:"category"`
	Cover      string `json:"cover"`
	Intro      string `json:"intro"`
	Setting    string `json:"setting"`
	Characters string `json:"characters"`
	Plot       string `json:"plot"`
	SourceURL  string `json:"source_url"`
}

// NovelSearchService 搜索服务 — 链式降级调度
type NovelSearchService struct {
	providers []NovelSearchProvider
}

// NewNovelSearchService 创建搜索服务，按优先级注册 Provider
func NewNovelSearchService(providers ...NovelSearchProvider) *NovelSearchService {
	return &NovelSearchService{providers: providers}
}

// Search 链式降级搜索：依次尝试每个 Provider，成功即返回
// 返回值：results 搜索结果, source 数据来源名称, warning 警告信息
func (s *NovelSearchService) Search(ctx context.Context, keyword string, limit int) (
	results []NovelSearchResult, source string, warning string,
) {
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	for _, p := range s.providers {
		if !p.Available() {
			log.Printf("[NovelSearch] %s is not available, skipping", p.Name())
			continue
		}
		res, err := p.Search(ctx, keyword, limit)
		if err == nil && len(res) > 0 {
			log.Printf("[NovelSearch] %s succeeded for keyword=%q, got %d results", p.Name(), keyword, len(res))
			return res, p.Name(), ""
		}
		log.Printf("[NovelSearch] %s failed for keyword=%q: %v, trying next provider...", p.Name(), keyword, err)
	}
	return []NovelSearchResult{}, "", "Search service temporarily unavailable, please try again later"
}

// ========== 公共工具函数，供各 Provider 复用 ==========

// ExtractNovelInfo 从简介中提取世界观、人物、剧情信息
func ExtractNovelInfo(result *NovelSearchResult) {
	intro := result.Intro
	if intro == "" {
		return
	}

	// 提取人物：匹配书名号《》和引号""中的内容
	var characters []string
	for _, pair := range []struct{ open, close string }{
		{"《", "》"},
		{"\u201c", "\u201d"},
		{"「", "」"},
	} {
		parts := strings.Split(intro, pair.open)
		for i := 1; i < len(parts); i++ {
			idx := strings.Index(parts[i], pair.close)
			if idx > 0 && idx <= 20 {
				characters = append(characters, parts[i][:idx])
			}
		}
	}
	if len(characters) > 0 {
		result.Characters = strings.Join(characters, "、")
	}

	// 按句号拆分，前半部分偏设定，后半部分偏剧情
	paragraphs := strings.FieldsFunc(intro, func(r rune) bool {
		return r == '\n' || r == '。' || r == '！' || r == '？'
	})
	var cleaned []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}

	if len(cleaned) == 0 {
		result.Setting = intro
		result.Plot = intro
		return
	}

	mid := len(cleaned) / 2
	if mid == 0 {
		mid = 1
	}
	result.Setting = strings.Join(cleaned[:mid], "。") + "。"
	result.Plot = strings.Join(cleaned[mid:], "。") + "。"
}

// 用于清理搜索结果标题的正则
var titleCleanRe = regexp.MustCompile(`[\s\-_|]+(?:小说|在线阅读|全文|最新章节|txt|下载|免费).*$`)

// ExtractNovelTitle 从搜索引擎标题中提取小说名
func ExtractNovelTitle(raw string) string {
	// 去除常见后缀噪音
	title := titleCleanRe.ReplaceAllString(raw, "")
	// 去除来源站点名（通常在 - 或 | 之后）
	if idx := strings.LastIndex(title, " - "); idx > 0 {
		title = title[:idx]
	}
	if idx := strings.LastIndex(title, "_"); idx > 0 && idx < len(title)-1 {
		title = title[:idx]
	}
	return strings.TrimSpace(title)
}

// ExtractMetaFromSnippet 从搜索摘要中尝试提取作者和分类
func ExtractMetaFromSnippet(result *NovelSearchResult, snippet string) {
	// 匹配 "作者：xxx" 或 "作者:xxx"
	authorRe := regexp.MustCompile(`作者[：:]\s*([^\s,，。;；]+)`)
	if m := authorRe.FindStringSubmatch(snippet); len(m) > 1 {
		result.Author = m[1]
	}
	// 匹配 "类型：xxx" 或 "分类：xxx" 或 "类别：xxx"
	categoryRe := regexp.MustCompile(`(?:类型|分类|类别)[：:]\s*([^\s,，。;；]+)`)
	if m := categoryRe.FindStringSubmatch(snippet); len(m) > 1 {
		result.Category = m[1]
	}
}
