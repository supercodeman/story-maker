// server/internal/service/butler_iterative_prompt.go
// 管家多轮迭代 Prompt 模板
package service

import (
	"fmt"
	"strings"
)

// buildStorylineDraftPrompt 构建故事线草稿生成 Prompt（四幕结构化输出）
func buildStorylineDraftPrompt(setting, topic, userPrompt string, enableBeats, enableSubplots bool, chapterNum int, conversationHistory []ConversationMessage) (systemPrompt, userContent string) {
	systemPrompt = `你是资深小说策划师。根据选题构思一个结构紧凑、张力十足的故事线。

【输出格式】
先写一段全局故事概述（200-300字），概括核心冲突、主要人物、故事走向。

然后按四幕结构展开：

## 第一幕·开局铺垫（约占全书20%）
核心任务：（一句话说明本幕要完成什么）
关键事件：
1. [事件名]：[具体描述，谁做了什么导致什么结果]
2. ...
转折点：[本幕末尾的转折，推动进入下一幕]
悬念钩子：[留给读者的悬念]

## 第二幕·冲突升级（约占全书35%）
（同上格式）

## 第三幕·高潮转折（约占全书30%）
（同上格式）

## 第四幕·结局收束（约占全书15%）
（同上格式）

---STORY_STRUCTURE---
{
  "synopsis": "全局概述文本",
  "acts": [
    {
      "name": "开局铺垫",
      "ratio": 20,
      "core_mission": "...",
      "key_events": ["事件1", "事件2"],
      "turning_point": "...",
      "hook": "...",
      "beats": []
    },
    {
      "name": "冲突升级",
      "ratio": 35,
      "core_mission": "...",
      "key_events": ["事件1", "事件2"],
      "turning_point": "...",
      "hook": "...",
      "beats": []
    },
    {
      "name": "高潮转折",
      "ratio": 30,
      "core_mission": "...",
      "key_events": ["事件1", "事件2"],
      "turning_point": "...",
      "hook": "...",
      "beats": []
    },
    {
      "name": "结局收束",
      "ratio": 15,
      "core_mission": "...",
      "key_events": ["事件1", "事件2"],
      "turning_point": "...",
      "hook": "...",
      "beats": []
    }
  ]
}
---END_STRUCTURE---

要求：
- 全局概述要让人一眼看懂"这个故事讲什么、为什么好看"
- 四幕之间必须有因果递进，不能割裂
- 每幕的转折点必须自然且出人意料
- 悬念钩子要让读者想继续看下一幕
- beats 数组默认为空（短篇不需要段落细化）
- 避免雷同老套的剧情套路（如"废柴逆袭"、"退婚流"等常见套路需有创新变化）
- 语言风格要自然，不要用"引人入胜"、"扣人心弦"、"波澜壮阔"等空洞形容词`

	// 段落细化（中长篇可选）
	if enableBeats && chapterNum > 30 {
		systemPrompt += fmt.Sprintf(`

【段落细化】
本作品为中长篇（约%d章），请对每幕进行段落细化。
每幕拆分为2-4个段落，每个段落：
- 有独立的小目标和小高潮
- 段落末尾有钩子衔接下一段落
- 标注该段落大约覆盖几章

在 JSON 的 beats 数组中填入：
"beats": [
  {"name": "段落名", "goal": "小目标", "climax": "小高潮", "hook": "段落钩子", "chapter_count": 5}
]`, chapterNum)
	}

	// 支线交织（可选）
	if enableSubplots {
		systemPrompt += `

【支线交织】
除主线外，设计1-2条支线：
- 每条支线有独立的起因和走向
- 标注支线在哪些幕/段落与主线交汇
- 支线最终要汇入主线或独立收束

在 JSON 中增加 subplots 字段：
"subplots": [
  {"name": "支线名", "description": "...", "intersect_acts": [1, 3]}
]`
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("【创作方向】\n%s", setting))
	parts = append(parts, fmt.Sprintf("【选题结果】\n%s", topic))
	if userPrompt != "" {
		parts = append(parts, fmt.Sprintf("【用户补充要求】\n%s", userPrompt))
	}
	if historyText := formatConversationHistory(conversationHistory); historyText != "" {
		parts = append(parts, historyText)
	}
	userContent = strings.Join(parts, "\n\n")
	return
}

// buildStorylineReviewPrompt 构建故事线 Review Prompt（含结构完整性审查）
func buildStorylineReviewPrompt(content string) (systemPrompt, userContent string) {
	systemPrompt = `你是一位严格的小说编辑，专门审查故事线质量。你的核心职责是发现问题并直接修改。

## 审查维度（每项1-10分）
1. 结构完整性（权重25%）：四幕是否完整、转折点是否自然、钩子是否有效、幕间因果递进
2. 原创性（权重25%）：是否有雷同老套的剧情？是否有创新的冲突设计？
3. 故事线清晰度（权重20%）：主线是否清晰？暗线是否合理？明暗关系是否有机交织？
4. 自然度（权重15%）：是否有AI味？是否有空洞的形容词堆砌？语言是否像人类作者写的？
5. 吸引力（权重15%）：是否有足够的悬念和张力？读者是否会想继续看下去？

## 输出格式（严格JSON，不要加 markdown 代码块标记）
{
  "score": 7.5,
  "issues": [
    {"dimension": "结构完整性", "problem": "具体问题", "suggestion": "具体修改建议"}
  ],
  "revised_content": "修改后的完整故事线文本（包含四幕结构和 ---STORY_STRUCTURE--- 标记的 JSON）"
}

## 修改原则
- 直接修改内容，不要只给建议
- 保持原有的好的部分，只改有问题的部分
- 减少AI味：删除"引人入胜"、"扣人心弦"、"波澜壮阔"等空洞词汇
- 减少雷同：如果发现常见套路，替换为更有创意的设计
- 强化四幕结构：确保每幕有明确的核心任务、转折点和钩子
- revised_content 必须保留完整的四幕文本和 ---STORY_STRUCTURE--- ... ---END_STRUCTURE--- 标记包裹的 JSON`

	userContent = fmt.Sprintf("【待审查的故事线】\n%s", content)
	return
}

// buildCharactersDraftPrompt 构建人物设计草稿生成 Prompt（含出场规划）
func buildCharactersDraftPrompt(setting, storyline, userPrompt string, conversationHistory []ConversationMessage) (systemPrompt, userContent string) {
	systemPrompt = `你是一位资深小说策划师，擅长设计立体丰满的人物群像。

## 人物数量要求（硬性）
- 必须设计不少于 10 个有名有姓的关键出场人物
- 按角色层级分配：主角 2-3 人、核心配角 3-4 人、重要配角 4-5 人
- 每个层级的人物都要有完整设定，不可敷衍

## 每个人物必须包含以下维度
1. 姓名、身份/职业
2. 外在特征：1-2个标志性外貌或习惯动作（如总是摩挲拇指上的旧戒指）
3. 性格内核：表层性格 + 深层性格（如表面冷漠实则极度缺乏安全感）
4. 语言特征：说话的语气节奏、用词习惯、口头禅或标志性表达方式
5. 核心动机：驱动人物行动的内在欲望或恐惧
6. 人物弧光：故事中的成长或转变方向
7. 角色定位：在故事结构中的功能（主角/对手/导师/催化剂/镜像/门槛守卫等）

## 出场与关系要求
1. 出场规划：首次出场的幕/段落、出场方式（正面登场/侧面提及/悬念引入）
2. 阶段作用：在每幕中扮演什么角色（推动者/阻碍者/旁观者/缺席）
3. 关键场景：该人物最重要的2-3个场景（标注在哪一幕）
4. 人物间的关系羁绊：不只是简单的"朋友/敌人"，要有深层的情感纠葛和利益关系
5. 人物性格要有层次感，避免脸谱化
6. 语言风格要自然，避免AI味

## 输出格式
先输出每个人物的详细设定文本（包含出场规划和阶段作用），然后依次用标记输出三段结构化数据：

---CHARACTER_CARDS---
[
  {
    "name": "人物名",
    "identity": "身份/职业",
    "role_type": "主角/核心配角/重要配角",
    "appearance": "标志性外貌或习惯动作",
    "personality_surface": "表层性格",
    "personality_deep": "深层性格",
    "speech_style": "语言特征描述",
    "motivation": "核心动机",
    "arc": "人物弧光"
  }
]
---END_CARDS---

---APPEARANCE_PLAN---
[
  {
    "character": "人物名",
    "first_appear": {"act": 1, "beat": "", "method": "正面登场"},
    "act_roles": {"act1": "推动者", "act2": "阻碍者", "act3": "转变者", "act4": "缺席"},
    "key_scenes": [
      {"act": 1, "scene": "场景描述"},
      {"act": 3, "scene": "场景描述"}
    ]
  }
]
---END_APPEARANCE---

---RELATION_MATRIX---
[
  {"from": "人物A", "to": "人物B", "relation": "关系类型", "detail": "具体描述", "tension": "high/medium/low"}
]
---END_MATRIX---

注意：三段标记按顺序输出：CHARACTER_CARDS → APPEARANCE_PLAN → RELATION_MATRIX。`

	var parts []string
	parts = append(parts, fmt.Sprintf("【创作方向】\n%s", setting))
	parts = append(parts, fmt.Sprintf("【故事线】\n%s", storyline))

	// 从故事线中提取结构化数据注入
	if structJSON := extractTagContent(storyline, "---STORY_STRUCTURE---", "---END_STRUCTURE---"); structJSON != "" {
		parts = append(parts, fmt.Sprintf("【故事线结构】\n%s", structJSON))
	}

	if userPrompt != "" {
		parts = append(parts, fmt.Sprintf("【用户补充要求】\n%s", userPrompt))
	}
	if historyText := formatConversationHistory(conversationHistory); historyText != "" {
		parts = append(parts, historyText)
	}
	userContent = strings.Join(parts, "\n\n")
	return
}

// buildCharactersReviewPrompt 构建人物设计 Review Prompt（含出场规划审查）
func buildCharactersReviewPrompt(content string) (systemPrompt, userContent string) {
	systemPrompt = `你是一位严格的小说编辑，专门审查人物设计质量。你的核心职责是发现问题并直接修改。

## 审查维度（每项1-10分）
1. 人物数量（权重15%）：是否有不少于 10 个有名有姓的关键人物？不足 10 个直接扣至 5 分以下，必须补充至 10 个以上
2. 出场节奏（权重15%）：出场是否分散合理、是否有角色扎堆出场、首次出场是否有记忆点
3. 人物立体度（权重15%）：性格是否有层次？是否脸谱化？人物弧光是否合理？
4. 语言辨识度（权重15%）：每个人物是否有独特的说话方式？语言特征是否具体可执行（而非"说话温柔"这类空泛描述）？
5. 关系羁绊深度（权重15%）：人物间关系是否有深层纠葛？是否只是简单标签？
6. 自然度（权重10%）：是否有AI味？描述是否像人类作者写的？
7. 与故事线契合度（权重15%）：人物设定是否服务于故事线？角色定位是否清晰？

## 输出格式（严格JSON，不要加 markdown 代码块标记）
{
  "score": 7.5,
  "issues": [
    {"dimension": "人物数量", "problem": "具体问题", "suggestion": "具体修改建议"}
  ],
  "revised_content": "修改后的完整人物设计文本（包含 ---CHARACTER_CARDS---、---APPEARANCE_PLAN--- 和 ---RELATION_MATRIX--- 三段标记）"
}

## 修改原则
- 直接修改内容，不要只给建议
- 保持原有的好的部分，只改有问题的部分
- 人物不足 10 个时，必须根据故事线需要补充新人物
- 强化关系羁绊：如果关系太浅，补充深层情感纠葛和利益冲突
- 强化语言特征：如果人物缺少说话方式描述，补充具体的语气、用词习惯、口头禅等
- 优化出场节奏：避免多个重要角色在同一幕扎堆出场
- 减少AI味：删除空洞的形容词，用具体细节替代
- revised_content 必须包含完整的人物设定文本、---CHARACTER_CARDS--- 人物卡片、---APPEARANCE_PLAN--- 出场规划和 ---RELATION_MATRIX--- 关系矩阵`

	userContent = fmt.Sprintf("【待审查的人物设计】\n%s", content)
	return
}

// buildOpeningSummaryPolishPrompt 前5章概要精细化 Prompt
func buildOpeningSummaryPolishPrompt(chapters, storyline, characters string) (systemPrompt, userContent string) {
	systemPrompt = `你是资深小说编辑，专精开篇打磨。前5章是读者留存的关键窗口，需要将粗略的章节概要升级为精细化的创作蓝图。

【任务】
对每一章的概要进行二次迭代，输出精细化版本。

【每章精细化要求】
1. 场景拆分：拆为3-5个微场景，每个标注地点/时间/氛围关键词
2. 人物亮相方式：首次出场的角色需设计外貌特写、标志性动作、第一句台词
3. 情绪节奏曲线：标注开头/中段/结尾的情绪走向（如"平→升→悬"）
4. 章末钩子：具体描述如何制造悬念或期待感
5. 衔接点：与下一章的过渡设计

【输出格式】
先输出分析思路，然后输出精细化结果。
结果用标记包裹：

---OPENING_CHAPTERS---
[
  {
    "chapter_index": 1,
    "title": "章节标题",
    "enhanced_summary": "精细化后的完整概要（300-500字）",
    "micro_scenes": [
      {"location": "地点", "time": "时间", "mood": "氛围", "brief": "场景简述"}
    ],
    "character_debuts": [
      {"name": "角色名", "appearance": "外貌特写", "signature_action": "标志性动作", "first_line": "第一句台词"}
    ],
    "emotion_curve": "平→升→悬",
    "chapter_hook": "章末钩子描述",
    "bridge_to_next": "与下一章衔接点"
  }
]
---END_OPENING---

要求：
- 精细化概要必须与故事线和人物设定保持一致
- 人物亮相要有记忆点，避免流水账式介绍
- 每章的钩子要具体可执行，不要空泛的"留下悬念"
- 情绪节奏要有起伏，避免5章全是同一节奏`

	userContent = fmt.Sprintf("【前5章原始概要】\n%s\n\n【故事线结构】\n%s\n\n【人物设定】\n%s", chapters, storyline, characters)
	return
}

// buildOpeningSummaryReviewPrompt 前5章概要审查 Prompt
func buildOpeningSummaryReviewPrompt(content string) (systemPrompt, userContent string) {
	systemPrompt = `你是资深小说编辑，审查前5章精细化概要的质量。

## 审查维度（每项1-10分）
1. 人物亮相记忆点（权重25%）：首次出场是否有鲜明的视觉/行为/语言印象？
2. 基调一致性（权重20%）：5章整体基调是否统一？是否符合故事类型？
3. 钩子有效性（权重20%）：每章末尾的钩子是否能驱动读者继续阅读？
4. 节奏张力（权重20%）：5章的情绪曲线是否有起伏？是否避免了单调？
5. 衔接流畅度（权重15%）：章与章之间的过渡是否自然？

## 输出格式（严格JSON，不要加 markdown 代码块标记）
{
  "score": 7.5,
  "issues": [
    {"dimension": "人物亮相记忆点", "problem": "具体问题", "suggestion": "具体修改建议"}
  ],
  "revised_content": "修改后的完整内容（包含 ---OPENING_CHAPTERS--- 和 ---END_OPENING--- 标记）"
}

## 修改原则
- 直接修改内容，不要只给建议
- 保持原有的好的部分，只改有问题的部分
- 强化人物亮相：如果出场太平淡，补充具体的外貌特写、标志性动作
- 强化钩子：如果钩子空泛，改为具体的悬念场景
- 优化节奏：如果5章节奏单调，调整情绪曲线分布
- revised_content 必须包含完整的精细化概要`

	userContent = fmt.Sprintf("【待审查的前5章精细化概要】\n%s", content)
	return
}

// formatConversationHistory 将对话历史格式化为 prompt 文本
// 当 history 非空时返回格式化文本，空时返回空字符串
func formatConversationHistory(history []ConversationMessage) string {
	if len(history) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("【调整历史】\n")
	for _, msg := range history {
		if msg.Role == "user" {
			sb.WriteString(fmt.Sprintf("[用户] %s\n", msg.Content))
		} else {
			// assistant 消息截取前200字作为摘要
			content := msg.Content
			if len([]rune(content)) > 200 {
				content = string([]rune(content)[:200]) + "..."
			}
			sb.WriteString(fmt.Sprintf("[AI] %s\n", content))
		}
	}
	sb.WriteString("---\n请基于上述调整历史，重点针对用户最新反馈进行修改。保留之前已确认的部分。")
	return sb.String()
}

// extractTagContent 从文本中提取指定标记之间的内容
func extractTagContent(text, startTag, endTag string) string {
	startIdx := strings.Index(text, startTag)
	if startIdx < 0 {
		return ""
	}
	startIdx += len(startTag)
	endIdx := strings.Index(text[startIdx:], endTag)
	if endIdx < 0 {
		return ""
	}
	return strings.TrimSpace(text[startIdx : startIdx+endIdx])
}
