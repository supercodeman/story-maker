// server/internal/service/chapter_summary.go
package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// 摘要树分组大小：每 groupSize 个节点聚合为上层节点
const summaryGroupSize = 5

// ChapterSummaryService 递归摘要树服务
type ChapterSummaryService struct {
	summaryDAO    *dao.ChapterSummaryDAO
	novelDAO      *dao.NovelDAO
	dispatcher    *agent.Dispatcher
	modelRegistry *ModelRegistryService
}

// NewChapterSummaryService 创建 ChapterSummaryService 实例
func NewChapterSummaryService(dispatcher *agent.Dispatcher) *ChapterSummaryService {
	return &ChapterSummaryService{
		summaryDAO: dao.NewChapterSummaryDAO(),
		novelDAO:   dao.NewNovelDAO(),
		dispatcher: dispatcher,
	}
}

// SetModelRegistry 注入模型注册服务
func (s *ChapterSummaryService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// RebuildTree 全量重建摘要树（小说初始化或手动触发）
func (s *ChapterSummaryService) RebuildTree(ctx context.Context, novelID uint) error {
	// 删除旧数据
	if err := s.summaryDAO.DeleteByNovel(novelID); err != nil {
		return fmt.Errorf("清除旧摘要树失败: %w", err)
	}

	// 获取所有章节
	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return fmt.Errorf("获取章节列表失败: %w", err)
	}
	if len(chapters) == 0 {
		return nil
	}

	// Level 0：每章一个摘要节点
	var level0Nodes []model.ChapterSummaryNode
	for _, ch := range chapters {
		summary := ch.Summary
		if summary == "" {
			// 无摘要则截取正文前 200 字
			runes := []rune(ch.Content)
			if len(runes) > 200 {
				summary = string(runes[:200]) + "..."
			} else {
				summary = ch.Content
			}
		}
		node := model.ChapterSummaryNode{
			NovelID:      novelID,
			Level:        0,
			StartChapter: ch.SortOrder,
			EndChapter:   ch.SortOrder,
			Summary:      summary,
		}
		if err := s.summaryDAO.Create(&node); err != nil {
			return fmt.Errorf("创建 Level 0 节点失败: %w", err)
		}
		level0Nodes = append(level0Nodes, node)
	}

	// 逐层聚合
	currentLevel := level0Nodes
	level := 0
	for len(currentLevel) > 1 {
		level++
		var nextLevel []model.ChapterSummaryNode
		for i := 0; i < len(currentLevel); i += summaryGroupSize {
			end := i + summaryGroupSize
			if end > len(currentLevel) {
				end = len(currentLevel)
			}
			group := currentLevel[i:end]

			// AI 聚合摘要
			mergedSummary, err := s.mergeSummaries(ctx, group)
			if err != nil {
				log.Printf("[chapter-summary] Level %d 聚合失败: %v", level, err)
				// 降级：直接拼接
				var parts []string
				for _, n := range group {
					parts = append(parts, n.Summary)
				}
				mergedSummary = strings.Join(parts, "\n")
			}

			node := model.ChapterSummaryNode{
				NovelID:      novelID,
				Level:        level,
				StartChapter: group[0].StartChapter,
				EndChapter:   group[len(group)-1].EndChapter,
				Summary:      mergedSummary,
			}
			if err := s.summaryDAO.Create(&node); err != nil {
				return fmt.Errorf("创建 Level %d 节点失败: %w", level, err)
			}

			// 回写 ParentID
			for _, child := range group {
				child.ParentID = &node.ID
				_ = s.summaryDAO.Update(&child)
			}

			nextLevel = append(nextLevel, node)
		}
		currentLevel = nextLevel
	}

	return nil
}

// UpdateIncremental 增量更新（新章节保存时调用）
// 只更新受影响的 Level 1 节点及其祖先
func (s *ChapterSummaryService) UpdateIncremental(ctx context.Context, novelID uint, chapterSortOrder int) error {
	// 获取章节信息
	chapters, err := s.novelDAO.ListChaptersByNovel(novelID)
	if err != nil {
		return err
	}
	if len(chapters) == 0 {
		return nil
	}

	// 查找或创建 Level 0 节点
	var chapter *model.Chapter
	for i := range chapters {
		if chapters[i].SortOrder == chapterSortOrder {
			chapter = &chapters[i]
			break
		}
	}
	if chapter == nil {
		return nil
	}

	summary := chapter.Summary
	if summary == "" {
		runes := []rune(chapter.Content)
		if len(runes) > 200 {
			summary = string(runes[:200]) + "..."
		} else {
			summary = chapter.Content
		}
	}

	// 更新或创建 Level 0 节点
	existing, err := s.summaryDAO.GetByRange(novelID, 0, chapterSortOrder, chapterSortOrder)
	if err != nil {
		// 不存在，创建
		node := &model.ChapterSummaryNode{
			NovelID:      novelID,
			Level:        0,
			StartChapter: chapterSortOrder,
			EndChapter:   chapterSortOrder,
			Summary:      summary,
		}
		return s.summaryDAO.Create(node)
	}
	existing.Summary = summary
	if err := s.summaryDAO.Update(existing); err != nil {
		return err
	}

	// 向上更新受影响的聚合节点
	return s.updateAncestors(ctx, novelID, chapterSortOrder)
}

// updateAncestors 更新受影响的祖先聚合节点
func (s *ChapterSummaryService) updateAncestors(ctx context.Context, novelID uint, chapterSortOrder int) error {
	// 获取所有层级的节点
	allNodes, err := s.summaryDAO.ListByNovel(novelID)
	if err != nil {
		return err
	}

	// 按层级分组
	levelMap := make(map[int][]model.ChapterSummaryNode)
	for _, n := range allNodes {
		levelMap[n.Level] = append(levelMap[n.Level], n)
	}

	// 从 Level 1 开始，找到包含该章节的聚合节点并更新
	for level := 1; ; level++ {
		nodes, ok := levelMap[level]
		if !ok || len(nodes) == 0 {
			break
		}

		for _, node := range nodes {
			if chapterSortOrder >= node.StartChapter && chapterSortOrder <= node.EndChapter {
				// 找到受影响的节点，重新聚合其子节点
				var children []model.ChapterSummaryNode
				childLevel := level - 1
				for _, child := range levelMap[childLevel] {
					if child.StartChapter >= node.StartChapter && child.EndChapter <= node.EndChapter {
						children = append(children, child)
					}
				}

				if len(children) > 0 {
					merged, err := s.mergeSummaries(ctx, children)
					if err != nil {
						log.Printf("[chapter-summary] 更新 Level %d 节点失败: %v", level, err)
						continue
					}
					node.Summary = merged
					_ = s.summaryDAO.Update(&node)
				}
				break
			}
		}
	}

	return nil
}

// GetRelevantContext 获取当前章节的层级摘要上下文
// 策略：当前所在 Level 1 分组的详细摘要 + 其他 Level 1 分组的概要 + Level 2 全局概要
func (s *ChapterSummaryService) GetRelevantContext(novelID uint, currentChapterOrder int, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 3000
	}

	var result strings.Builder
	totalChars := 0

	// 获取 Level 0 节点（当前分组内的详细摘要）
	level0Nodes, _ := s.summaryDAO.ListByNovelAndLevel(novelID, 0)
	level1Nodes, _ := s.summaryDAO.ListByNovelAndLevel(novelID, 1)
	level2Nodes, _ := s.summaryDAO.ListByNovelAndLevel(novelID, 2)

	// 确定当前章节所在的 Level 1 分组
	var currentGroup *model.ChapterSummaryNode
	for i := range level1Nodes {
		if currentChapterOrder >= level1Nodes[i].StartChapter && currentChapterOrder <= level1Nodes[i].EndChapter {
			currentGroup = &level1Nodes[i]
			break
		}
	}

	// 1. 写入 Level 2 全局概要（如果有）
	if len(level2Nodes) > 0 {
		for _, n := range level2Nodes {
			entry := fmt.Sprintf("【全局概要 第%d-%d章】%s\n", n.StartChapter, n.EndChapter, n.Summary)
			entryChars := len([]rune(entry))
			if totalChars+entryChars > maxChars {
				break
			}
			result.WriteString(entry)
			totalChars += entryChars
		}
		result.WriteString("\n")
	}

	// 2. 写入其他 Level 1 分组的概要
	for _, n := range level1Nodes {
		if currentGroup != nil && n.StartChapter == currentGroup.StartChapter {
			continue // 跳过当前分组，后面用详细摘要
		}
		entry := fmt.Sprintf("【第%d-%d章概要】%s\n", n.StartChapter, n.EndChapter, n.Summary)
		entryChars := len([]rune(entry))
		if totalChars+entryChars > maxChars {
			break
		}
		result.WriteString(entry)
		totalChars += entryChars
	}

	// 3. 写入当前分组内的 Level 0 详细摘要
	if currentGroup != nil {
		result.WriteString(fmt.Sprintf("\n【当前分组详细（第%d-%d章）】\n", currentGroup.StartChapter, currentGroup.EndChapter))
		for _, n := range level0Nodes {
			if n.StartChapter >= currentGroup.StartChapter && n.EndChapter <= currentGroup.EndChapter {
				entry := fmt.Sprintf("- 第%d章：%s\n", n.StartChapter, n.Summary)
				entryChars := len([]rune(entry))
				if totalChars+entryChars > maxChars {
					break
				}
				result.WriteString(entry)
				totalChars += entryChars
			}
		}
	}

	return result.String()
}

// mergeSummaries 调用 AI 将多个摘要合并为一个精炼摘要（含降级链）
func (s *ChapterSummaryService) mergeSummaries(ctx context.Context, nodes []model.ChapterSummaryNode) (string, error) {
	defaultModel := "qwen"
	if s.modelRegistry != nil {
		defaultModel = s.modelRegistry.GetDefaultModel(model.CapTextGen)
	}

	// 构建降级链
	chain := []string{defaultModel}
	if s.modelRegistry != nil {
		chain = append(chain, s.modelRegistry.GetFallbackChain(0, defaultModel, model.CapTextGen)...)
	}

	var parts []string
	for _, n := range nodes {
		parts = append(parts, fmt.Sprintf("第%d-%d章：%s", n.StartChapter, n.EndChapter, n.Summary))
	}

	prompt := fmt.Sprintf(`请将以下 %d 个章节摘要合并为一个精炼的综合摘要，保留：
1. 关键情节转折和事件
2. 人物关系变化
3. 未解决的伏笔和悬念
4. 重要的世界观信息
控制在 300 字以内。

%s`, len(nodes), strings.Join(parts, "\n\n"))

	var lastErr error
	for _, modelName := range chain {
		provider, err := s.dispatcher.GetProviderWithKey(ctx, modelName)
		if err != nil {
			log.Printf("[chapter-summary] 获取 %s Provider 失败: %v", modelName, err)
			lastErr = err
			continue
		}
		resp, err := provider.GenerateText(ctx, &agent.TextRequest{
			Prompt:      prompt,
			MaxTokens:   1024,
			Temperature: 0.3,
		})
		if err != nil {
			log.Printf("[chapter-summary] %s GenerateText 失败: %v", modelName, err)
			lastErr = err
			continue
		}
		return resp.Content, nil
	}

	return "", fmt.Errorf("合并摘要所有模型均失败: %w", lastErr)
}
