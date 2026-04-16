// server/internal/agent/orchestrator/hit_analysis_template.go
package orchestrator

// BuildHitAnalysisGraph 构建爆款拆解 DAG
// Layer 0（并行）: structure_analysis | rhythm_analysis | character_analysis
// Layer 1: synthesis（汇总三路结果，生成综合报告）
func BuildHitAnalysisGraph(modelName, sourceText string) *Graph {
	g := NewGraph()
	fb := FallbackModels(modelName)

	// 版权合规：所有 prompt 都要求不引用原文
	copyrightNotice := "\n\n重要：只分析叙事技法和结构模式，不要引用或复述原文内容。输出为抽象的技法分析。"

	// Layer 0: 三路并行分析
	g.AddNode(&Node{
		ID:             "structure_analysis",
		TaskType:       "text_gen",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: `分析以下小说素材的剧情结构，输出结构化分析：

【分析素材】
{{.source_text}}

请从以下维度分析：
1. 整体结构类型（三幕式/英雄之旅/起承转合/其他）
2. 各阶段划分及占比
3. 关键转折点的位置和类型
4. 高潮的设置方式
5. 开篇钩子和结尾处理

输出 JSON 格式：
{
  "structure_type": "结构类型",
  "phases": [{"name":"阶段名","ratio":"占比","description":"描述"}],
  "turning_points": [{"position":"位置","type":"类型","description":"描述"}],
  "climax_design": "高潮设计分析",
  "opening_hook": "开篇钩子分析",
  "ending_style": "结尾风格分析"
}` + copyrightNotice,
		OutputKey: "structure_result",
		InputMap:  map[string]string{"source_text": "source_text"},
		MaxTokens: 4096,
	})

	g.AddNode(&Node{
		ID:             "rhythm_analysis",
		TaskType:       "text_gen",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: `分析以下小说素材的叙事节奏，输出结构化分析：

【分析素材】
{{.source_text}}

请从以下维度分析：
1. 张弛节奏模式（紧张-舒缓的交替规律）
2. 钩子密度（悬念设置的频率和类型）
3. 爆点分布规律（高潮点的间隔和递进）
4. 叙事速度变化（快节奏/慢节奏的切换时机）
5. 信息释放节奏（伏笔和揭示的时机）

输出 JSON 格式：
{
  "tension_pattern": "张弛节奏模式描述",
  "hook_density": "钩子密度分析",
  "hook_types": ["悬念类型1","悬念类型2"],
  "climax_distribution": "爆点分布规律",
  "pacing_changes": "叙事速度变化分析",
  "info_release": "信息释放节奏分析"
}` + copyrightNotice,
		OutputKey: "rhythm_result",
		InputMap:  map[string]string{"source_text": "source_text"},
		MaxTokens: 4096,
	})

	g.AddNode(&Node{
		ID:             "character_analysis",
		TaskType:       "text_gen",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: `分析以下小说素材的人物弧线和角色设计，输出结构化分析：

【分析素材】
{{.source_text}}

请从以下维度分析：
1. 主角成长模式（成长弧线类型和关键转变节点）
2. 配角功能分类（导师/对手/盟友/催化剂等）
3. 人物关系动力学（核心关系的变化驱动力）
4. 角色塑造技法（如何让角色立体可信）
5. 群像处理方式（多角色的平衡和交织）

输出 JSON 格式：
{
  "protagonist_arc": "主角成长模式分析",
  "supporting_roles": [{"type":"角色类型","function":"叙事功能","technique":"塑造技法"}],
  "relationship_dynamics": "人物关系动力学分析",
  "characterization_techniques": ["技法1","技法2"],
  "ensemble_handling": "群像处理方式分析"
}` + copyrightNotice,
		OutputKey: "character_result",
		InputMap:  map[string]string{"source_text": "source_text"},
		MaxTokens: 4096,
	})

	// Layer 1: 综合汇总
	g.AddNode(&Node{
		ID:             "synthesis",
		TaskType:       "text_gen",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: `根据以下三个维度的分析结果，生成一份综合拆解报告：

【结构分析】
{{.structure}}

【节奏分析】
{{.rhythm}}

【人物分析】
{{.character}}

请综合以上分析，输出结构化的综合报告 JSON：
{
  "structure_analysis": "剧情结构综合分析（200字以内）",
  "rhythm_analysis": "节奏规律综合分析（200字以内）",
  "character_arcs": "人物弧线综合分析（200字以内）",
  "hook_points": [{"position":"位置描述","type":"钩子类型","technique":"具体技法"}],
  "style_features": "文风与叙事特征总结（200字以内）",
  "summary": "综合技法总结与创作建议（300字以内）"
}

要求：
- 提炼可复用的创作技法和模式
- 给出具体可操作的创作建议
- 不要引用或复述原文内容`,
		OutputKey: "synthesis_result",
		InputMap: map[string]string{
			"structure": "structure_result",
			"rhythm":    "rhythm_result",
			"character": "character_result",
		},
		DependsOn: []string{"structure_analysis", "rhythm_analysis", "character_analysis"},
		MaxTokens: 8192,
	})

	// 添加边
	g.AddEdge("structure_analysis", "synthesis")
	g.AddEdge("rhythm_analysis", "synthesis")
	g.AddEdge("character_analysis", "synthesis")

	return g
}
