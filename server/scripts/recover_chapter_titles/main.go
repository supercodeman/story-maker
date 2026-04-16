// 章节标题恢复工具
// 从 ai_tasks 表的 outline_generate 任务结果中提取原始章节标题，
// 与当前 chapters 表对比，生成恢复 SQL 并可选择直接执行。
//
// 用法：
//   cd server && go run scripts/recover_chapter_titles/main.go
//   cd server && go run scripts/recover_chapter_titles/main.go --dry-run   # 仅预览，不执行
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"ai-curton/server/config"
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// outlineChapter 大纲生成结果中的章节结构
type outlineChapter struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// recoveryItem 单条恢复记录
type recoveryItem struct {
	ChapterID    uint
	SortOrder    int
	CurrentTitle string
	OriginalTitle string
}

func main() {
	dryRun := false
	for _, arg := range os.Args[1:] {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	// 加载配置并初始化数据库
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := model.InitDB(&cfg.Database); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	db := model.DB

	// 第一步：查找"心跳备忘录"小说
	var novel model.Novel
	if err := db.Where("title LIKE ?", "%心跳备忘录%").First(&novel).Error; err != nil {
		log.Fatalf("未找到'心跳备忘录'小说: %v", err)
	}
	fmt.Printf("找到小说: [ID=%d] %s (章节数: %d)\n\n", novel.ID, novel.Title, novel.ChapterCount)

	// 第二步：获取当前所有章节
	var chapters []model.Chapter
	if err := db.Where("novel_id = ?", novel.ID).Order("sort_order ASC").Find(&chapters).Error; err != nil {
		log.Fatalf("获取章节列表失败: %v", err)
	}
	fmt.Printf("当前章节列表 (%d 章):\n", len(chapters))
	for _, ch := range chapters {
		fmt.Printf("  [%d] 排序=%d 标题=\"%s\"\n", ch.ID, ch.SortOrder, ch.Title)
	}
	fmt.Println()

	// 第三步：查找相关的 outline_generate 任务
	var tasks []model.AITask
	query := db.Where("task_type = ? AND status = ?", model.TaskTypeOutlineGenerate, model.TaskStatusCompleted)
	// 优先按 novel_id 精确匹配
	if err := query.Where("novel_id = ?", novel.ID).Order("created_at DESC").Find(&tasks).Error; err != nil || len(tasks) == 0 {
		// 如果 novel_id 没匹配到（可能大纲生成时还没创建小说），按 portfolio_id 查
		db.Where("task_type = ? AND status = ? AND portfolio_id = ?",
			model.TaskTypeOutlineGenerate, model.TaskStatusCompleted, novel.PortfolioID).
			Order("created_at DESC").Find(&tasks)
	}

	if len(tasks) == 0 {
		// 最后兜底：查所有 outline_generate 任务，在 result 中搜索关键词
		db.Where("task_type = ? AND status = ?",
			model.TaskTypeOutlineGenerate, model.TaskStatusCompleted).
			Order("created_at DESC").Find(&tasks)
	}

	fmt.Printf("找到 %d 个大纲生成任务\n\n", len(tasks))

	// 第四步：解析每个任务的 result，找到最匹配的大纲
	type candidate struct {
		taskID     uint
		taskTime   string
		chapters   []outlineChapter
		matchScore int // 与当前章节数的匹配度
	}
	var candidates []candidate

	for _, task := range tasks {
		if task.Result == "" {
			continue
		}

		// 尝试解析 result JSON
		parsed := parseOutlineResult(task.Result)
		if len(parsed) == 0 {
			continue
		}

		// 计算匹配度：章节数一致得高分，summary 匹配得额外分
		score := 0
		if len(parsed) == len(chapters) {
			score += 100
		}
		// 检查 summary 匹配度
		for i, ch := range parsed {
			if i < len(chapters) && ch.Summary != "" && chapters[i].Summary != "" {
				if ch.Summary == chapters[i].Summary || strings.Contains(chapters[i].Summary, ch.Summary[:min(20, len(ch.Summary))]) {
					score += 10
				}
			}
		}

		candidates = append(candidates, candidate{
			taskID:     task.ID,
			taskTime:   task.CreatedAt.Format("2006-01-02 15:04:05"),
			chapters:   parsed,
			matchScore: score,
		})
	}

	if len(candidates) == 0 {
		fmt.Println("未找到可用的大纲生成记录。")
		fmt.Println("\n其他恢复建议：")
		fmt.Println("  1. 检查数据库是否有备份")
		fmt.Println("  2. 检查 MySQL binlog 是否开启")
		fmt.Println("  3. 查看 ai_tasks 表中 outline_title_polish 类型的任务")
		os.Exit(1)
	}

	// 按匹配度排序，展示所有候选
	fmt.Printf("找到 %d 个候选大纲:\n\n", len(candidates))
	bestIdx := 0
	bestScore := 0
	for i, c := range candidates {
		marker := "  "
		if c.matchScore > bestScore {
			bestScore = c.matchScore
			bestIdx = i
		}
		if c.matchScore == bestScore && i == bestIdx {
			marker = "★ "
		}
		fmt.Printf("%s[任务ID=%d] 时间=%s 章节数=%d 匹配度=%d\n", marker, c.taskID, c.taskTime, len(c.chapters), c.matchScore)
		for j, ch := range c.chapters {
			fmt.Printf("    第%d章: %s\n", j+1, ch.Title)
		}
		fmt.Println()
	}

	// 选择最佳候选
	best := candidates[bestIdx]
	fmt.Printf("选择最佳匹配: 任务ID=%d (匹配度=%d)\n\n", best.taskID, best.matchScore)

	// 第五步：生成恢复方案
	var items []recoveryItem
	for i, ch := range chapters {
		if i >= len(best.chapters) {
			fmt.Printf("⚠ 章节 [ID=%d] 排序=%d 在大纲中无对应项，跳过\n", ch.ID, ch.SortOrder)
			continue
		}
		original := best.chapters[i]
		if ch.Title != original.Title {
			items = append(items, recoveryItem{
				ChapterID:    ch.ID,
				SortOrder:    ch.SortOrder,
				CurrentTitle: ch.Title,
				OriginalTitle: original.Title,
			})
		}
	}

	if len(items) == 0 {
		fmt.Println("\n所有章节标题与大纲一致，无需恢复。")
		return
	}

	// 展示恢复方案
	fmt.Printf("\n需要恢复 %d 个章节标题:\n", len(items))
	fmt.Println(strings.Repeat("-", 80))
	for _, item := range items {
		fmt.Printf("  章节ID=%-4d 排序=%-3d  \"%s\"  →  \"%s\"\n",
			item.ChapterID, item.SortOrder, item.CurrentTitle, item.OriginalTitle)
	}
	fmt.Println(strings.Repeat("-", 80))

	// 生成恢复 SQL（无论是否 dry-run 都输出）
	fmt.Println("\n恢复 SQL:")
	fmt.Println("BEGIN;")
	for _, item := range items {
		escaped := strings.ReplaceAll(item.OriginalTitle, "'", "\\'")
		fmt.Printf("UPDATE chapters SET title = '%s' WHERE id = %d;\n", escaped, item.ChapterID)
	}
	fmt.Println("COMMIT;")

	// 执行恢复
	if dryRun {
		fmt.Println("\n[DRY-RUN 模式] 未执行任何修改。去掉 --dry-run 参数即可执行恢复。")
		return
	}

	fmt.Print("\n即将执行恢复，是否继续？(y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("已取消。")
		return
	}

	// 事务内批量更新
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&model.Chapter{}).Where("id = ?", item.ChapterID).
				Update("title", item.OriginalTitle).Error; err != nil {
				return fmt.Errorf("更新章节 %d 失败: %w", item.ChapterID, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("恢复失败: %v", err)
	}

	fmt.Printf("\n恢复完成！已更新 %d 个章节标题。\n", len(items))
}

// parseOutlineResult 解析大纲任务的 result 字段
// AI 返回的格式可能是纯 JSON 数组，也可能包裹在 markdown 代码块中
func parseOutlineResult(result string) []outlineChapter {
	result = strings.TrimSpace(result)

	// 尝试去除 markdown 代码块包裹
	if strings.HasPrefix(result, "```") {
		lines := strings.Split(result, "\n")
		// 去掉首尾的 ``` 行
		if len(lines) >= 3 {
			start := 1
			end := len(lines) - 1
			if strings.HasPrefix(lines[0], "```") {
				start = 1
			}
			if strings.TrimSpace(lines[end]) == "```" || strings.HasPrefix(strings.TrimSpace(lines[end]), "```") {
				end = end
			}
			result = strings.Join(lines[start:end], "\n")
		}
	}

	result = strings.TrimSpace(result)

	// 直接尝试解析 JSON 数组
	var chapters []outlineChapter
	if err := json.Unmarshal([]byte(result), &chapters); err == nil && len(chapters) > 0 {
		return chapters
	}

	// 尝试找到 JSON 数组的起止位置
	startIdx := strings.Index(result, "[")
	endIdx := strings.LastIndex(result, "]")
	if startIdx >= 0 && endIdx > startIdx {
		jsonStr := result[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &chapters); err == nil && len(chapters) > 0 {
			return chapters
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
