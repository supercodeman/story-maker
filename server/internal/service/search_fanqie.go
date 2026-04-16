// server/internal/service/search_fanqie.go
// 番茄小说搜索 Provider — 从 crawler.go 迁移重构
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

// FanqieSearchProvider 番茄小说搜索引擎
type FanqieSearchProvider struct {
	client *resty.Client
	uas    []string
}

// NewFanqieSearchProvider 创建番茄小说搜索 Provider
func NewFanqieSearchProvider() *FanqieSearchProvider {
	client := resty.New().
		SetTimeout(5 * time.Second).
		SetRetryCount(1).
		SetRetryWaitTime(500 * time.Millisecond)

	// 读取系统代理配置
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			client.SetTransport(&http.Transport{Proxy: http.ProxyURL(proxyURL)})
		}
	} else if proxy := os.Getenv("https_proxy"); proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			client.SetTransport(&http.Transport{Proxy: http.ProxyURL(proxyURL)})
		}
	}

	return &FanqieSearchProvider{
		client: client,
		uas: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
	}
}

func (p *FanqieSearchProvider) Name() string    { return "fanqie" }
func (p *FanqieSearchProvider) Available() bool { return true }

func (p *FanqieSearchProvider) randomUA() string {
	return p.uas[rand.Intn(len(p.uas))]
}

func (p *FanqieSearchProvider) randomDelay() {
	delay := time.Duration(200+rand.Intn(300)) * time.Millisecond
	time.Sleep(delay)
}

// fanqieSearchResp 番茄小说搜索 API 响应结构
type fanqieSearchResp struct {
	Code int `json:"code"`
	Data struct {
		SearchBookData []struct {
			BookID   string `json:"book_id"`
			BookName string `json:"book_name"`
			Author   string `json:"author"`
			Abstract string `json:"abstract"`
			Category string `json:"category"`
			Cover    string `json:"thumb_url"`
		} `json:"search_book_data"`
	} `json:"data"`
}

func (p *FanqieSearchProvider) Search(ctx context.Context, keyword string, limit int) ([]NovelSearchResult, error) {
	// 请求番茄小说搜索 API
	searchURL := fmt.Sprintf(
		"https://fanqienovel.com/api/author/search/search_book/v1?filter=127,127,127,127&page_count=%d&page_index=0&query_type=0&query_word=%s",
		limit, keyword,
	)

	resp, err := p.client.R().
		SetContext(ctx).
		SetHeader("User-Agent", p.randomUA()).
		SetHeader("Accept", "application/json").
		Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("番茄搜索请求失败: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("番茄搜索返回非200状态: %d", resp.StatusCode())
	}

	var searchResp fanqieSearchResp
	if err := json.Unmarshal(resp.Body(), &searchResp); err != nil {
		return nil, fmt.Errorf("解析番茄搜索结果失败: %w", err)
	}

	books := searchResp.Data.SearchBookData
	if len(books) == 0 {
		return nil, fmt.Errorf("番茄搜索无结果")
	}
	if len(books) > limit {
		books = books[:limit]
	}

	// 逐本获取详情页，提取更多信息
	results := make([]NovelSearchResult, 0, len(books))
	for _, book := range books {
		result := NovelSearchResult{
			Title:     book.BookName,
			Author:    book.Author,
			Category:  book.Category,
			Cover:     book.Cover,
			Intro:     book.Abstract,
			SourceURL: fmt.Sprintf("https://fanqienovel.com/page/%s", book.BookID),
		}

		// 尝试获取详情页丰富简介
		p.randomDelay()
		if detail := p.fetchBookDetail(ctx, book.BookID); detail != "" {
			result.Intro = detail
		}

		ExtractNovelInfo(&result)
		results = append(results, result)
	}

	return results, nil
}

// fetchBookDetail 获取书籍详情页的完整简介
func (p *FanqieSearchProvider) fetchBookDetail(ctx context.Context, bookID string) string {
	detailURL := fmt.Sprintf("https://fanqienovel.com/page/%s", bookID)

	resp, err := p.client.R().
		SetContext(ctx).
		SetHeader("User-Agent", p.randomUA()).
		SetHeader("Accept", "text/html").
		Get(detailURL)
	if err != nil || resp.StatusCode() != http.StatusOK {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
	if err != nil {
		return ""
	}

	intro := doc.Find(".page-abstract-content").Text()
	if intro == "" {
		intro = doc.Find("[class*='abstract']").First().Text()
	}
	if intro == "" {
		intro = doc.Find("meta[name='description']").AttrOr("content", "")
	}

	return strings.TrimSpace(intro)
}
