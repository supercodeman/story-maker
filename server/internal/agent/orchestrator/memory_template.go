// server/internal/agent/orchestrator/memory_template.go
package orchestrator

import "fmt"

// 记忆类别对应的特征提取 Prompt
var memoryFeaturePrompts = map[string]string{
	"style": `分析以下小说样本的写作风格特征，按 4 个子维度提取为结构化 JSON。

每个子维度需包含：
- description：该维度的特征描述
- score：该维度的评分（0-100）
- examples：从样本中提取的 2-3 个典型句子

输出 JSON：
{
  "tone": {
    "description": "文风调性描述（如冷峻克制/温暖治愈/幽默讽刺等）",
    "score": 85,
    "examples": ["典型句子1", "典型句子2"]
  },
  "rhythm": {
    "description": "行文节奏描述（短句为主/长短交替/绵密长句等）",
    "score": 80,
    "examples": ["典型句子1", "典型句子2"]
  },
  "vocabulary": {
    "description": "用词特点描述（文雅/通俗/专业术语/修辞手法等）",
    "score": 75,
    "examples": ["典型句子1", "典型句子2"]
  },
  "dialogue_style": {
    "description": "对话风格描述（简洁利落/文艺腔/口语化/方言特色等）",
    "score": 80,
    "examples": ["典型对话1", "典型对话2"]
  },
  "forbidden_patterns": ["应避免的表达1", "应避免的表达2"],
  "reference_style": "最接近的知名作家风格"
}

样本文本：
{{.sample_text}}

请严格按照上述 JSON 格式输出，不要添加额外说明。`,

	"character": `分析以下小说样本中的人物特征，提取为结构化 JSON：
{
  "personality_traits": ["性格特征"],
  "speech_style": "说话方式描述",
  "catchphrases": ["口头禅"],
  "behavior_logic": "行为逻辑描述",
  "emotional_range": "情感表达范围"
}

样本文本：
{{.sample_text}}

请严格按照上述 JSON 格式输出，不要添加额外说明。`,

	"worldview": `分析以下小说样本中的世界观设定，提取为结构化 JSON：
{
  "power_system": "力量体系描述",
  "world_rules": ["世界规则"],
  "factions": ["势力/阵营"],
  "history_background": "历史背景",
  "technology_level": "科技/魔法水平"
}

样本文本：
{{.sample_text}}

请严格按照上述 JSON 格式输出，不要添加额外说明。`,

	"plot_preference": `分析以下小说样本的剧情偏好特征，提取为结构化 JSON：
{
  "tension_density": "爽点密度（高/中/低）",
  "twist_frequency": "反转频率",
  "pacing": "节奏偏好（快节奏/慢热/张弛有度）",
  "ending_preference": "结局偏好（HE/BE/开放式）",
  "conflict_style": "冲突风格（正面对抗/暗流涌动/心理博弈）"
}

样本文本：
{{.sample_text}}

请严格按照上述 JSON 格式输出，不要添加额外说明。`,
}

// promptCompileTemplate Prompt 编译模板（支持子维度 prompt_part 生成）
const promptCompileTemplate = `根据以下结构化特征，编写一段写作指令（Prompt 模板），用于指导 AI 在创作时遵循该风格。
要求：
1. 指令要具体、可执行，避免模糊描述
2. 包含正面指令（应该怎么写）和负面指令（不应该怎么写）
3. 从样本中提取 3-5 个最能代表风格的"锚定句"作为 few-shot 示例
4. 总长度控制在 500 字以内
5. 如果特征中包含 tone/rhythm/vocabulary/dialogue_style 子维度，请为每个子维度生成独立的 prompt_part 写作指令片段

特征 JSON：
{{.features}}

原始样本（用于提取锚定句）：
{{.sample_text}}

输出 JSON：
{
  "prompt_template": "完整写作指令...",
  "anchor_texts": ["锚定句1", "锚定句2", ...],
  "dimension_prompts": {
    "tone": "文风维度的写作指令片段（如特征中无此维度则留空）",
    "rhythm": "句式维度的写作指令片段",
    "vocabulary": "语感维度的写作指令片段",
    "dialogue_style": "对话维度的写作指令片段"
  }
}

请严格按照上述 JSON 格式输出，不要添加额外说明。`

// qualityEvalTemplate 质量评估模板（多维评分）
const qualityEvalTemplate = `使用以下写作指令生成 100 字小说片段，然后从 4 个维度评估该记忆的质量（每项 0-100 分）。

写作指令：
{{.prompt_tpl}}

参考锚定句：
{{.anchor_texts}}

评分维度说明：
- consistency（风格一致性）：生成片段与原始样本的风格吻合度
- reproducibility（可复现性）：该指令能否稳定复现目标风格
- uniqueness（独特性）：该风格是否有辨识度，区别于通用写法
- practicality（实用性）：该指令在实际创作中的可操作性

输出 JSON：
{
  "preview_text": "生成的100字片段",
  "consistency": 85,
  "reproducibility": 80,
  "uniqueness": 75,
  "practicality": 90,
  "evaluation": "评价说明"
}

请严格按照上述 JSON 格式输出，不要添加额外说明。`

// BuildMemoryExtractGraph 构建记忆提取 DAG
// Layer 0（并行）: feature_extract + embedding_gen
// Layer 1: prompt_compile（依赖 feature_extract）
// Layer 2: quality_eval（依赖 prompt_compile）
func BuildMemoryExtractGraph(modelName, category string) *Graph {
	g := NewGraph()
	fb := FallbackModels(modelName)

	featurePrompt, ok := memoryFeaturePrompts[category]
	if !ok {
		featurePrompt = memoryFeaturePrompts["style"]
	}

	// Layer 0: 特征提取
	g.AddNode(&Node{
		ID:             "feature_extract",
		TaskType:       "memory_feature_extract",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt:         featurePrompt,
		OutputKey:      "features",
		InputMap:       map[string]string{"sample_text": "sample_text"},
		MaxTokens:      4096,
	})

	// Layer 1: Prompt 编译（依赖特征提取结果）
	g.AddNode(&Node{
		ID:             "prompt_compile",
		TaskType:       "memory_prompt_compile",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt:         promptCompileTemplate,
		OutputKey:      "compiled",
		InputMap:       map[string]string{"features": "features", "sample_text": "sample_text"},
		DependsOn:      []string{"feature_extract"},
		MaxTokens:      4096,
	})

	// Layer 2: 质量评估（依赖 Prompt 编译结果）
	g.AddNode(&Node{
		ID:             "quality_eval",
		TaskType:       "memory_quality_eval",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt:         qualityEvalTemplate,
		OutputKey:      "quality",
		InputMap:       map[string]string{"prompt_tpl": "compiled", "anchor_texts": "compiled"},
		DependsOn:      []string{"prompt_compile"},
		MaxTokens:      2048,
	})

	// 添加边
	g.AddEdge("feature_extract", "prompt_compile")
	g.AddEdge("prompt_compile", "quality_eval")

	return g
}

// BuildMemoryReviewGraph 构建记忆审核 DAG
// Layer 0（并行）: quality_check + compliance_check
// Layer 1: review_decision
func BuildMemoryReviewGraph(modelName string) *Graph {
	g := NewGraph()
	fb := FallbackModels(modelName)

	// Layer 0: 质量检查（多维评分）
	g.AddNode(&Node{
		ID:             "quality_check",
		TaskType:       "memory_review_quality",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: `评估以下写作记忆的质量，生成一段 100 字样本并从 4 个维度打分（每项 0-100）。

特征：
{{.features}}

Prompt 模板：
{{.prompt_tpl}}

评分维度：
- consistency（风格一致性）
- reproducibility（可复现性）
- uniqueness（独特性）
- practicality（实用性）

输出 JSON：
{
  "sample_text": "生成的100字样本",
  "consistency": 85,
  "reproducibility": 80,
  "uniqueness": 75,
  "practicality": 90,
  "quality_issues": ["问题1", "问题2"]
}`,
		OutputKey: "quality_result",
		InputMap:  map[string]string{"features": "features", "prompt_tpl": "prompt_tpl"},
		MaxTokens: 2048,
	})

	// Layer 0: 合规检查（并行）
	g.AddNode(&Node{
		ID:             "compliance_check",
		TaskType:       "memory_review_compliance",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: `检查以下写作记忆内容是否合规（无色情、暴力、政治敏感等违规内容）。

特征：
{{.features}}

Prompt 模板：
{{.prompt_tpl}}

输出 JSON：
{
  "is_compliant": true,
  "violations": [],
  "risk_level": "low"
}`,
		OutputKey: "compliance_result",
		InputMap:  map[string]string{"features": "features", "prompt_tpl": "prompt_tpl"},
		MaxTokens: 1024,
	})

	// Layer 1: 审核决策
	g.AddNode(&Node{
		ID:             "review_decision",
		TaskType:       "memory_review_decision",
		ModelName:      modelName,
		FallbackModels: fb,
		Prompt: fmt.Sprintf(`根据质量检查和合规检查结果，做出审核决策。

质量检查结果：
{{.quality}}

合规检查结果：
{{.compliance}}

审核标准：
- 质量检查中 4 项维度（consistency/reproducibility/uniqueness/practicality）平均分 >= 70（即 B 级及以上）且合规通过 → approved
- 平均分 < 70 → rejected（说明原因及各维度得分）
- 合规不通过 → rejected（说明违规项）

输出 JSON：
{
  "decision": "approved 或 rejected",
  "reason": "决策原因",
  "suggestions": ["改进建议"]
}`),
		OutputKey: "decision",
		InputMap:  map[string]string{"quality": "quality_result", "compliance": "compliance_result"},
		DependsOn: []string{"quality_check", "compliance_check"},
		MaxTokens: 1024,
	})

	// 添加边
	g.AddEdge("quality_check", "review_decision")
	g.AddEdge("compliance_check", "review_decision")

	return g
}
