// server/internal/agent/orchestrator/templates.go
package orchestrator

import (
	"fmt"
)

// FallbackProvider 降级模型列表提供函数
// primary 为当前主模型名称，返回排除主模型后的降级模型列表
type FallbackProvider func(primary string) []string

// ChapterInput 批量扩写的章节输入
type ChapterInput struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// FallbackModels 根据主模型生成降级列表（默认实现）
// 排除主模型自身，按优先级排列备选模型
func FallbackModels(primary string) []string {
	all := []string{"zhipu", "qwen", "kimi", "deepseek"}
	var fallbacks []string
	for _, m := range all {
		if m != primary {
			fallbacks = append(fallbacks, m)
		}
	}
	return fallbacks
}

// filterProviders 从降级链中过滤出允许的模型
func filterProviders(chain []string, allowed ...string) []string {
	set := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		set[a] = true
	}
	var result []string
	for _, p := range chain {
		if set[p] {
			result = append(result, p)
		}
	}
	return result
}

// novelWritingSystemPrompt 小说正文写作的叙事风格规范
// 用于 draft、revision、expand 等正文生成节点的 system prompt
// 精简版本，节省 token
const novelWritingSystemPrompt = `【叙事风格规范】
- 感官细节每段2-3处（视觉/听觉/触觉），其余概括叙述
- 情感心理可直写（"他感到一阵寒意"），不必全靠动作暗示
- 动作连贯流畅，不拆解过细时值；场景切换用明确过渡词
- 主要角色各有1-2个辨识特征（口头禅/小动作）
- 关键转折处直写内心感受，不全部留白
- 避免连续超25字长句；段落3-8行`

// toolCallingHint 提示模型使用知识库工具的附加说明
const toolCallingHint = `

【知识库工具】
你可以使用以下工具按需查询小说设定，确保内容与已有设定一致：
- query_characters：查询角色档案（性格、语言特征、外貌、关系）
- query_worldview：查询世界观设定（地理、势力、规则）
- query_foreshadow：查询伏笔记录（哪些需要呼应或延续）
- query_plotline：查询剧情线索（主线/支线走向）
使用策略：先查询本章涉及的主要角色和关键设定，再开始写作。不确定的设定一定要查。
重要：查询完成后直接输出小说正文，不要输出任何过渡语句、查询总结或写作计划。`

// BuildFullChapterGraph 一键生成完整章节
// Layer 0: outline（大纲生成，注入知识库）
// Layer 1: draft（完整章节正文生成，融合角色/场景/对话/排版要求）
// Layer 2: review_loop（审核-修订循环，最多 3 轮，审核通过即退出）
// writingStyle 为格式化后的写作风格文本，为空则不注入
// knowledgeContext 为知识图谱上下文文本，为空则不注入
func BuildFullChapterGraph(modelName string, writingStyle string, knowledgeContext interface{}, fallbackFn FallbackProvider) *Graph {
	g := NewGraph()
	fb := fallbackFn(modelName)
	contentFb := filterProviders(fb, "qwen", "zhipu") // 内容生成节点仅降级到 qwen 和 zhipu

	// 排版规范（仅 draft 和 revision 使用）
	formatSpec := `【排版规范】
- 段首无空格无缩进，段落间空1行，段内不换行
- 对话用""包裹，后接动作/心理不换行
- 章节标题仅"第X章 标题"格式，标题下空1行接正文
- 每段聚焦一个场景/动作/心理/对话，单段不超5-6行
- 禁止：加粗、斜体、多余空行空格、agent标识、格式备注`

	// 文风禁忌（仅 draft 使用，review 用关键词引用）
	antiAISpec := `
【文风禁忌】
1. 禁止形容词堆砌（"深邃的眼眸""凌厉的气势"），用具体动作替代
2. 禁止煽情总结句（"他知道，这一切才刚刚开始""命运的齿轮开始转动"）
3. 禁止连续相同句式（连续"他……"开头），变换主语和节奏
4. 禁止万能过渡词（"然而""与此同时""就在这时"），用动作/场景切换过渡
5. 禁止直白心理旁白（"他心中一震""他暗暗想到"），用生理反应间接表现
6. 禁止套路化反应（"众人皆惊""全场哗然"），写具体人物的具体反应
7. 禁止同义反复（"他很愤怒，怒火在胸中燃烧"），只保留更强的表达
8. 禁止陈词滥调比喻（"如同暴风雨般"），一段最多一个且必须贴合场景`

	// 写作规范后缀
	styleSuffix := ""
	if writingStyle != "" {
		styleSuffix = "\n\n【写作规范】\n" + writingStyle
	}

	// 知识图谱上下文（仅 outline 注入，draft 通过大纲间接获得）
	knowledgeSuffix := ""
	if kc, ok := knowledgeContext.(string); ok && kc != "" {
		knowledgeSuffix = "\n\n【知识图谱（人物/情节/伏笔/世界观）】\n" + kc
	}

	// Layer 0: 大纲生成（注入知识图谱+前文+写作规范，为 draft 提供充分信息）
	g.AddNode(&Node{
		ID:             "outline",
		TaskType:       "text_gen",
		ModelName:      modelName,
		FallbackModels: contentFb,
		Prompt: `为第{{.chapter_order}}章「{{.title}}」设计大纲。禁止输出其他章节或全书规划。

章节编号：第{{.chapter_order}}章
章节标题：{{.title}}
章节概要：{{.background}}
{{if .prev_context}}
【前文回顾】
{{.prev_context}}
{{end}}` + knowledgeSuffix + `

要求：
1. 列出3-5个场景，每个写明：地点、核心事件（谁做了什么导致什么结果）、出场人物
2. 与前文衔接但不重复，人物行为符合知识图谱设定
3. 情节要有意外感，避免"被羞辱→获宝→打脸"等套路模板
4. 大纲中直接写出关键角色的外貌特征和说话风格要点，供正文写作参考` + styleSuffix + `

直接输出大纲，不要解释。`,
		OutputKey: "outline_result",
		InputMap:  map[string]string{"title": "title", "background": "background", "prev_context": "prev_context", "chapter_order": "chapter_sort_order"},
		MaxTokens: 4096,
	})

	// Layer 1: 骨架初稿生成（~1000字，后续用户手动扩写）
	// 启用 tool calling：模型可按需查询知识库（人物、世界观、伏笔、剧情线）
	g.AddNode(&Node{
		ID:             "draft",
		TaskType:       "text_gen",
		ModelName:      modelName,
		FallbackModels: contentFb,
		EnableTools:    true,
		SystemPrompt:   novelWritingSystemPrompt + toolCallingHint,
		Prompt: `根据大纲撰写第{{.chapter_order}}章「{{.title}}」的骨架初稿，约1000字。
只写第{{.chapter_order}}章，这是骨架初稿，后续用户会手动扩写充实。

在动笔前，请先使用工具查询本章涉及的角色档案和相关设定，确保人物言行、世界观细节与已有设定一致。

【章节大纲】
{{.outline}}

【骨架写作要求】
1. 覆盖大纲中所有场景，按场景顺序推进，不遗漏关键事件
2. 每个场景用2-3句建立感官锚点（一个视觉 + 一个听觉或触觉）
3. 关键对话直接写出完整台词（每场景2-3轮核心对话），非关键对话用一句话概括
4. 情节转折和人物情感变化必须体现，用动作或生理反应表现
5. 场景之间用一句过渡句衔接
6. 聚焦结构完整和节奏紧凑，不要注水
` + antiAISpec + `

` + formatSpec + styleSuffix + `

直接输出骨架初稿正文，不要解释或标注。`,
		OutputKey: "final_result",
		InputMap:  map[string]string{"outline": "outline_result", "title": "title", "chapter_order": "chapter_sort_order"},
		DependsOn: []string{"outline"},
		MaxTokens: 8192,
	})

	// 审核 prompt（精简：不注入知识图谱、前文、写作规范）
	// issues 使用三元组格式：quote（原文引用）+ problem（问题类型）+ fix（修改建议）
	reviewPrompt := `审核以下章节骨架初稿（约1000字），重点关注结构完整性和是否像人写的。

【大纲】
{{.outline}}

【正文】
{{.draft}}

以JSON输出审核结果：
{"passed":bool,"overall_score":0-100,"dimensions":{"plot_coherence":{"score":0-100,"issues":[{"quote":"原文片段","problem":"问题描述","fix":"修改建议"}]},"narrative_quality":{"score":0-100,"issues":[...]},"formatting":{"score":0-100,"issues":[...]},"ai_artifacts":{"score":0-100,"issues":[...]}},"revision_instructions":"修改方向"}

issues 格式要求：
- quote：精确引用原文中的句子或片段（10-50字），不要概括
- problem：一句话说明问题类型（如"形容词堆砌""煽情总结句""排版错误"）
- fix：给出具体的替换文本或修改方向

维度说明：
- plot_coherence：情节与大纲是否一致，有无逻辑漏洞，角色言行是否与大纲设定吻合
- narrative_quality：文笔节奏、对话自然度、描写层次、与前文衔接
- formatting：排版规范（段首无空格、段间空1行、对话格式）、字数在800-1500字区间
- ai_artifacts（权重最高）：形容词堆砌、煽情总结句、重复句式、万能过渡词（然而/就在这时）、直白心理旁白（他心中一震）、套路化反应（众人皆惊）、同义反复、陈词滥调比喻

判定：overall_score>=75则passed=true。ai_artifacts<70则直接passed=false。字数不足800直接passed=false。
仅输出JSON。`

	// 修订 prompt（精准逐条执行：基于三元组 issues 定位原文并修改）
	revisionPrompt := `根据审核意见修订以下章节。修订后骨架不少于800字。

【审核意见】
{{.review}}

【原始章节】
{{.draft}}

修订方法：
1. 逐条处理审核意见中 issues 列表的每个问题
2. 在原文中找到 quote 引用的位置
3. 按 fix 建议修改该处，保持上下文连贯
4. 未被 issues 提及的段落保持原样

修订原则：
- AI痕迹：堆砌→精准动词；煽情总结→删除；心理旁白→生理反应；套路反应→具体人物反应
- 排版：段首无空格，段间空1行，段内不换行，对话用""包裹且后接动作不换行
- 不要给修订说明

直接输出修订后的完整章节正文。不要给修订说明。`

	// Layer 2: 审核-修订循环（LoopNode 替代原来的 6 个静态节点）
	g.AddLoopNode(&LoopNode{
		ID:        "review_loop",
		DependsOn: []string{"draft"},
		Config: LoopConfig{
			MaxRounds:   2,
			OnExhausted: ExhaustedContinue, // 2 轮后仍不通过，使用最后一轮结果，标记 warning

			ExitCondition: func(state *SharedState, round int) bool {
				// review 被跳过（review_result 不存在），直接退出循环输出 content
				result, ok := parseReviewResult(state, "review_loop.latest.review_result")
				if !ok {
					return true // review 跳过或解析失败，退出循环
				}
				if !result.Passed {
					return false
				}
				// 硬校验：骨架初稿不足 1000 字不放行
				if contentTooShort(state, "final_result", 1000) {
					return false
				}
				return true
			},

			SubGraphBuilder: func(round int) *Graph {
				sub := NewGraph()

				// 审核节点（默认 deepseek，降级到用户模型，全部失败则跳过）
				sub.AddNode(&Node{
					ID:             "review",
					TaskType:       "text_gen",
					ModelName:      "deepseek",
					FallbackModels: []string{modelName},
					SkipOnAllFail:  true,
					Prompt:         reviewPrompt,
					OutputKey:      "review_result",
					InputMap: map[string]string{
						"outline": "outline_result",
						"draft":   "final_result",
					},
					MaxTokens: 2048,
				})

				// 修订节点（条件执行：审核不通过时才修订）
				sub.AddNode(&Node{
					ID:             "revision",
					TaskType:       "text_polish",
					ModelName:      modelName,
					FallbackModels: contentFb,
					SystemPrompt:   novelWritingSystemPrompt,
					Prompt:         revisionPrompt,
					OutputKey:      "final_result",
					InputMap: map[string]string{
						"review": "review_result",
						"draft":  "final_result",
					},
					DependsOn: []string{"review"},
					MaxTokens: 8192,
				})

				sub.AddConditionalEdge("review", "revision", func(state *SharedState) bool {
					return reviewNotPassed(state, "review_result") || contentTooShort(state, "final_result", 1000)
				})

				return sub
			},
		},
	})

	// 添加边
	g.AddEdge("outline", "draft")

	return g
}

// BuildBatchExpandGraph 批量章节扩写
// 所有章节并行扩写
// writingStyle 为格式化后的写作风格文本，为空则不注入
func BuildBatchExpandGraph(modelName string, chapters []ChapterInput, writingStyle string, fallbackFn FallbackProvider) *Graph {
	g := NewGraph()
	fb := fallbackFn(modelName)

	styleSuffix := ""
	if writingStyle != "" {
		styleSuffix = "\n\n【写作规范】\n" + writingStyle
	}

	for i, ch := range chapters {
		nodeID := fmt.Sprintf("expand_ch_%d", i)
		g.AddNode(&Node{
			ID:             nodeID,
			TaskType:       "chapter_expand",
			ModelName:      modelName,
			FallbackModels: fb,
			SystemPrompt:   novelWritingSystemPrompt,
			Prompt:         fmt.Sprintf("请扩写以下章节：\n标题：%s\n概要：%s\n要求：丰富细节，扩展情节，保持原有风格。", ch.Title, ch.Summary) + styleSuffix,
			OutputKey:      fmt.Sprintf("expand_result_%d", i),
			MaxTokens:      16384,
		})
	}

	return g
}
