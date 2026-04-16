// server/internal/agent/orchestrator/revision_template.go
package orchestrator

import "fmt"

// BuildNovelRevisionGraph 构建小说变更分析工作流 DAG（两阶段方案的第一阶段）
// Stage 1: 影响分析 — 输入 Diff + 章节索引，输出受影响章节列表
// Stage 2: 调整规划 — 输入受影响章节概要 + Diff，输出每章修改方案
func BuildNovelRevisionGraph(modelName, diffText, chapterIndex string) *Graph {
	g := NewGraph()
	fb := FallbackModels(modelName)

	// Stage 1: 影响分析
	g.AddNode(&Node{
		ID:             "impact_analysis",
		TaskType:       "revision_analysis",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: fmt.Sprintf(`你是一个小说修订助手。以下是小说的章节索引和一组元数据变更。
请分析哪些章节会受到这些变更的影响。

【章节索引】
%s

%s

请以 JSON 格式返回受影响的章节列表：
{
  "affected_chapters": [
    {"chapter_order": 1, "impact_level": "content", "reason": "..."},
    {"chapter_order": 3, "impact_level": "summary", "reason": "..."}
  ]
}

impact_level 说明：
- "content": 需要修改正文内容
- "summary": 只需调整概要
- "none": 不受影响（不要列出）

仅列出确实受影响的章节，不要过度扩大范围。`, chapterIndex, diffText),
		OutputKey: "impact_result",
		MaxTokens: 4096,
	})

	// Stage 2: 调整规划
	g.AddNode(&Node{
		ID:             "revision_planning",
		TaskType:       "revision_planning",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: fmt.Sprintf(`你是一个小说修订规划助手。根据影响分析结果，为每个受影响的章节制定具体的修改方案。

【章节索引】
%s

%s

【影响分析结果】
{{.impact}}

请为每个受影响的章节制定修改方案，以 JSON 格式返回：
{
  "revision_plans": [
    {
      "chapter_order": 1,
      "impact_level": "content",
      "instructions": "具体修改指令...",
      "key_changes": ["变更点1", "变更点2"]
    }
  ]
}

修改指令要具体、可执行，避免模糊描述。`, chapterIndex, diffText),
		OutputKey: "planning_result",
		InputMap:  map[string]string{"impact": "impact_result"},
		DependsOn: []string{"impact_analysis"},
		MaxTokens: 8192,
	})

	// 添加边
	g.AddEdge("impact_analysis", "revision_planning")

	return g
}

// BuildNovelRevisionExecuteGraph 构建小说变更执行工作流 DAG（两阶段方案的第二阶段）
// 根据修改计划动态构建并行章节修改节点 + 一致性校验
// 调用方需要在 SharedState 中预填充 chapter_content_0, chapter_content_1, ... 等章节内容
func BuildNovelRevisionExecuteGraph(modelName, revisionPlan, chapterIndex string, chapterCount int) *Graph {
	g := NewGraph()
	fb := FallbackModels(modelName)

	// 并行章节修改节点（根据计划中的章节数量动态创建）
	reviseNodeIDs := make([]string, 0, chapterCount)
	for i := 0; i < chapterCount; i++ {
		nodeID := fmt.Sprintf("chapter_revise_%d", i)
		reviseNodeIDs = append(reviseNodeIDs, nodeID)

		contentKey := fmt.Sprintf("chapter_content_%d", i)

		g.AddNode(&Node{
			ID:             nodeID,
			TaskType:       "chapter_revise",
			ModelName:      modelName,
			FallbackModels: fb,
			Prompt: fmt.Sprintf(`你是一个小说章节修订助手。请根据修改方案修订以下章节。

【修改计划】
%s

【当前处理】第 %d 个修改任务

【章节内容】
{{.chapter_content}}

请直接输出修改后的完整章节内容，不要添加任何解释或标注。`, revisionPlan, i+1),
			OutputKey: fmt.Sprintf("revise_result_%d", i),
			InputMap:  map[string]string{"chapter_content": contentKey},
			MaxTokens: 16384,
		})
	}

	// 一致性校验节点 — 将所有修改结果拼接为 Prompt 变量
	consistencyInputMap := make(map[string]string)
	var revisedVarRefs string
	for i := 0; i < chapterCount; i++ {
		key := fmt.Sprintf("revised_%d", i)
		consistencyInputMap[key] = fmt.Sprintf("revise_result_%d", i)
		revisedVarRefs += fmt.Sprintf("\n--- 章节修改 %d ---\n{{.revised_%d}}\n", i+1, i)
	}

	g.AddNode(&Node{
		ID:             "consistency_check",
		TaskType:       "revision_analysis",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: fmt.Sprintf(`你是一个小说一致性校验助手。请检查以下修改后的章节内容是否存在前后矛盾。

【章节索引】
%s

【修改计划】
%s

【修改后的章节内容】
%s

请检查：
1. 人物名称、设定是否前后一致
2. 时间线是否合理
3. 情节是否连贯

以 JSON 格式返回：
{
  "consistent": true/false,
  "issues": [{"chapter_order": 1, "issue": "..."}]
}`, chapterIndex, revisionPlan, revisedVarRefs),
		OutputKey: "consistency_result",
		InputMap:  consistencyInputMap,
		DependsOn: reviseNodeIDs,
		MaxTokens: 4096,
	})

	// 添加边：所有修改节点 → 一致性校验
	for _, nodeID := range reviseNodeIDs {
		g.AddEdge(nodeID, "consistency_check")
	}

	return g
}
