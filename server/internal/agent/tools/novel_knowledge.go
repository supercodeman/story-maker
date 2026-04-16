// server/internal/agent/tools/novel_knowledge.go
package tools

import (
	"context"
	"fmt"
	"strings"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// NewNovelKnowledgeTools 创建小说知识库工具集（按 novelID 绑定）
// 返回的 ToolRegistry 包含 4 个工具：query_characters, query_worldview, query_foreshadow, query_plotline
func NewNovelKnowledgeTools(novelID uint, knowledgeDAO *dao.KnowledgeDAO) *agent.ToolRegistry {
	registry := agent.NewToolRegistry()

	registry.Register(newQueryTool(
		"query_characters",
		"查询人物档案：角色的性格、语言特征、外貌、关系等设定。当对话涉及特定角色时使用。",
		model.KnowledgeCategoryCharacter,
		novelID, knowledgeDAO,
	))

	registry.Register(newQueryTool(
		"query_worldview",
		"查询世界观设定：地理、势力、规则、历史等。当涉及设定细节时使用。",
		model.KnowledgeCategoryWorldview,
		novelID, knowledgeDAO,
	))

	registry.Register(newQueryTool(
		"query_foreshadow",
		"查询伏笔追踪记录：哪些伏笔已埋下、哪些已揭示。当需要呼应或延续伏笔时使用。",
		model.KnowledgeCategoryForeshadow,
		novelID, knowledgeDAO,
	))

	registry.Register(newQueryTool(
		"query_plotline",
		"查询主线/支线剧情走向。当需要确认情节方向或避免偏离主线时使用。",
		model.KnowledgeCategoryPlotline,
		novelID, knowledgeDAO,
	))

	return registry
}

// newQueryTool 创建单个知识库查询工具
func newQueryTool(name, description, category string, novelID uint, knowledgeDAO *dao.KnowledgeDAO) *agent.Tool {
	return &agent.Tool{
		Name:        name,
		Description: description,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keyword": map[string]any{
					"type":        "string",
					"description": "搜索关键词，如角色名、地名、事件名等",
				},
			},
			"required": []string{"keyword"},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			keyword, _ := args["keyword"].(string)
			keyword = strings.TrimSpace(keyword)

			var items []model.NovelKnowledge
			var err error

			if keyword == "" {
				// 无关键词：返回该类别的全部条目
				items, err = knowledgeDAO.ListByNovelAndCategory(novelID, category)
			} else {
				// 有关键词：在该类别内搜索
				allItems, searchErr := knowledgeDAO.SearchByTags(novelID, keyword)
				if searchErr != nil {
					return "", searchErr
				}
				// 过滤出目标类别
				for _, item := range allItems {
					if item.Category == category {
						items = append(items, item)
					}
				}
				err = nil
			}
			if err != nil {
				return "", fmt.Errorf("query %s failed: %w", category, err)
			}

			if len(items) == 0 {
				return fmt.Sprintf("未找到与「%s」相关的%s记录", keyword, categoryLabel(category)), nil
			}

			var sb strings.Builder
			for _, item := range items {
				if category == model.KnowledgeCategoryForeshadow {
					status := "未揭示"
					if item.Resolved {
						status = "已揭示"
					}
					sb.WriteString(fmt.Sprintf("【%s】（%s）\n%s\n\n", item.Title, status, item.Content))
				} else {
					sb.WriteString(fmt.Sprintf("【%s】\n%s\n\n", item.Title, item.Content))
				}
			}
			return strings.TrimSpace(sb.String()), nil
		},
	}
}

func categoryLabel(category string) string {
	labels := map[string]string{
		model.KnowledgeCategoryCharacter:  "人物档案",
		model.KnowledgeCategoryWorldview:  "世界观",
		model.KnowledgeCategoryForeshadow: "伏笔",
		model.KnowledgeCategoryPlotline:   "剧情线索",
	}
	if l, ok := labels[category]; ok {
		return l
	}
	return category
}
