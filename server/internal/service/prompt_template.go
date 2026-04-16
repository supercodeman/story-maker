// server/internal/service/prompt_template.go
package service

import (
	"bytes"
	"fmt"
	"text/template"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// PromptTemplateService Prompt 模板业务逻辑层
type PromptTemplateService struct {
	dao *dao.PromptTemplateDAO
}

// NewPromptTemplateService 创建 PromptTemplateService 实例
func NewPromptTemplateService() *PromptTemplateService {
	return &PromptTemplateService{
		dao: dao.NewPromptTemplateDAO(),
	}
}

// RenderPrompt 获取模板并渲染：系统默认为基础，小说级自定义作为补充追加
func (s *PromptTemplateService) RenderPrompt(novelID uint, action, promptType string, data *model.PromptTemplateData) (string, error) {
	// 始终渲染系统默认模板
	defaultTpl, err := s.dao.GetDefault(action, promptType)
	if err != nil {
		return "", fmt.Errorf("default template not found for action=%s type=%s: %w", action, promptType, err)
	}
	base, err := s.renderContent(defaultTpl.Content, data)
	if err != nil {
		return "", err
	}

	// 查找小说级自定义模板，作为补充追加
	if novelID > 0 {
		customTpl, err := s.dao.GetCustom(novelID, action, promptType)
		if err == nil && customTpl.Content != "" {
			extra, err := s.renderContent(customTpl.Content, data)
			if err == nil && extra != "" {
				base += "\n\n【用户补充指令】\n" + extra
			}
		}
	}

	return base, nil
}

// PreviewTemplate 用临时模板内容预览渲染结果
func (s *PromptTemplateService) PreviewTemplate(content string, data *model.PromptTemplateData) (string, error) {
	return s.renderContent(content, data)
}

// ListMerged 列出小说的模板（合并默认+自定义，自定义覆盖默认）
func (s *PromptTemplateService) ListMerged(novelID uint) ([]model.PromptTemplate, error) {
	defaults, err := s.dao.ListDefaults()
	if err != nil {
		return nil, err
	}

	if novelID == 0 {
		return defaults, nil
	}

	customs, err := s.dao.ListByNovel(novelID)
	if err != nil {
		return nil, err
	}

	// 用自定义覆盖默认
	customMap := make(map[string]*model.PromptTemplate)
	for i := range customs {
		key := customs[i].Action + ":" + customs[i].PromptType
		customMap[key] = &customs[i]
	}

	result := make([]model.PromptTemplate, 0, len(defaults))
	for _, d := range defaults {
		key := d.Action + ":" + d.PromptType
		if c, ok := customMap[key]; ok {
			result = append(result, *c)
		} else {
			result = append(result, d)
		}
	}
	return result, nil
}

// Upsert 创建或更新小说级自定义模板
func (s *PromptTemplateService) Upsert(tpl *model.PromptTemplate) error {
	return s.dao.Upsert(tpl)
}

// Delete 删除自定义模板（恢复使用默认）
func (s *PromptTemplateService) Delete(id uint) error {
	return s.dao.Delete(id)
}

// SeedDefaults 初始化默认模板
func (s *PromptTemplateService) SeedDefaults() error {
	templates := defaultTemplates()
	return s.dao.SeedDefaults(templates)
}

// renderContent 使用 text/template 渲染模板内容
func (s *PromptTemplateService) renderContent(content string, data *model.PromptTemplateData) (string, error) {
	t, err := template.New("prompt").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}
	return buf.String(), nil
}

// defaultTemplates 返回 16 条默认模板（8 action × 2 type）
func defaultTemplates() []model.PromptTemplate {
	return []model.PromptTemplate{
		// summary_polish
		{
			Action: "summary_polish", PromptType: "system",
			Name: "概要润色-系统提示",
			Content: `你是一位资深小说策划编辑。你的任务是对章节概要进行深度润色和扩写，使其成为后续章节正文写作的高质量参考蓝图。

核心要求：
1. 充分结合知识库中的角色档案、世界观设定、伏笔记录和剧情线索
2. 确保概要与前后章节的情节衔接自然，故事线连贯
3. 润色后的概要应包含：场景描述、关键事件、人物互动、情感变化、伏笔呼应
4. 保持原有情节方向不变，但要充实细节，使概要具备指导正文写作的充分信息量

直接输出润色后的概要文本，不要包含标题或额外说明。{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}`,
		},
		{
			Action: "summary_polish", PromptType: "user",
			Name: "概要润色-用户提示",
			Content: `【小说背景】
{{.NovelDescription}}
{{- if .Characters}}

【角色档案】
{{.Characters}}
{{- end}}
{{- if .WorldviewNotes}}

【世界观设定】
{{.WorldviewNotes}}
{{- end}}
{{- if .KnowledgeContext}}

【知识库参考（伏笔/剧情线索）】
{{.KnowledgeContext}}
{{- end}}

【前文各章概要】
{{.PrevSummaries}}
{{- if .NextChapters}}

【后续章节概要】
{{.NextChapters}}
{{- end}}

【当前章节】{{.ChapterTitle}}
【当前章节概要】
{{.ChapterSummary}}

请对以上概要进行深度润色和扩写，要求：
1. 结合角色档案，明确本章出场人物的行为动机和情感状态
2. 与前文概要衔接，交代因果关系；与后文概要呼应，埋设必要铺垫
3. 检查并融入知识库中与本章相关的伏笔和剧情线索
4. 补充关键场景的环境要素和氛围基调
5. 润色后概要应在300-400字之间，信息密度高，可直接指导正文写作

直接输出润色后的概要文本。`,
		},
		// polish
		{
			Action: "polish", PromptType: "system",
			Name: "正文润色-系统提示",
			Content: `你是一位经验丰富的文学编辑。对原文进行润色，改善表达质量，同时保持作者的个人风格和叙事节奏。

润色原则：
- 改善不等于堆砌：用更准确的词替换模糊的词，而非用更华丽的词替换普通的词
- 写得好的段落少改或不改，有明显问题的段落重点修改
- 保持原文的语言风格和叙事腔调，不要把朴素文风改成华丽文风
- 保持原有情节和人物关系不变

直接输出润色后的正文。{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}{{if .ReviewContext}}

【历史审核问题（请在本次操作中规避）】
{{.ReviewContext}}{{end}}

【叙事风格】
- 感官细节每段2-3处，其余概括叙述；情感心理可直写，不必全靠动作暗示
- 动作连贯流畅，不拆解过细时值；场景切换用明确过渡词
- 主要角色各有1-2个辨识特征；关键转折处直写内心感受
- 人物行为合理
- 成长线贴合现实
- 句子不超25字连续出现；段落3-8行`,
		},
		{
			Action: "polish", PromptType: "user",
			Name: "正文润色-用户提示",
			Content: `【小说背景】
{{.NovelDescription}}

【当前章节】{{.ChapterTitle}}
【章节概要】{{.ChapterSummary}}
{{- if .Characters}}

【人物档案】
{{.Characters}}
{{- end}}
{{- if .SelectedText}}
【选中片段】
{{.SelectedText}}

请仅对以上选中片段进行润色，保持上下文语境一致。只输出替换后的片段文本，不要输出完整章节。
{{- else}}
【原文】
{{.ChapterContent}}

请对以上原文进行润色。要求：
1. 用更准确的动词和名词替换模糊笼统的表达
2. {{- if .Characters}}参照人物档案中的性格和说话方式润色对话，确保每个角色的语气与档案设定一致{{- else}}改善对话自然度，让不同角色的语气有区分{{- end}}
3. 在描写薄弱处适当补充感官细节，已经写得好的地方保持原样
4. 优化句式节奏，减少连续相同句式
5. 保持原有情节走向、人物关系和作者文风不变
6. 行为处事方式要贴合世界观设定
{{- end}}
{{- if .PolishModeInstruction}}

【润色方向】
{{.PolishModeInstruction}}
{{- end}}`,
		},
		// expand
		{
			Action: "expand", PromptType: "system",
			Name: "章节扩写-系统提示",
			Content: `你是一位专业的小说作家。严格按照章节概要进行扩写，丰富细节描写和对话，不要偏离概要设定的情节方向。扩写后正文不少于3000字，这是硬性要求。直接输出扩写后的完整正文。{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}{{if .ReviewContext}}

【历史审核问题（请在本次操作中规避）】
{{.ReviewContext}}{{end}}

【叙事风格】
- 感官细节每段2-3处，其余概括叙述；情感心理可直写，不必全靠动作暗示
- 动作连贯流畅，不拆解过细时值；场景切换用明确过渡词
- 主要角色各有1-2个辨识特征；关键转折处直写内心感受
- 句子不超25字连续出现；段落3-8行`,
		},
		{
			Action: "expand", PromptType: "user",
			Name: "章节扩写-用户提示",
			Content: `【小说背景】
{{.NovelDescription}}

【前文各章概要】
{{.PrevSummaries}}
{{- if .PrevContent}}

【前一章末尾内容】
{{.PrevContent}}
{{- end}}
{{- if .KnowledgeContext}}

【知识库参考】
{{.KnowledgeContext}}
{{- end}}

【当前章节】{{.ChapterTitle}}
【章节概要】{{.ChapterSummary}}
{{- if .SelectedText}}
【选中片段】
{{.SelectedText}}

请仅对以上选中片段进行扩写，丰富细节描写和对话，保持上下文语境一致。只输出替换后的片段文本，不要输出完整章节。
{{- else}}
【当前内容（{{.WordCount}}字）】
{{.ChapterContent}}

请严格按照章节概要进行扩写，目标 {{.TargetWords}} 字，正文不少于3000字（硬性要求）。丰富场景描写和人物对话，确保与前文情节衔接自然，不要偏离概要设定的情节方向。
{{- end}}`,
		},
		// continue
		{
			Action: "continue", PromptType: "system",
			Name: "章节续写-系统提示",
			Content: `你是一位专业的小说作家。严格按照章节概要续写后续情节，不要偏离概要设定的情节方向。直接输出续写的内容。{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}{{if .ReviewContext}}

【历史审核问题（请在本次操作中规避）】
{{.ReviewContext}}{{end}}

【叙事风格】
- 感官细节每段2-3处，其余概括叙述；情感心理可直写，不必全靠动作暗示
- 动作连贯流畅，不拆解过细时值；场景切换用明确过渡词
- 主要角色各有1-2个辨识特征；关键转折处直写内心感受
- 句子不超25字连续出现；段落3-8行`,
		},
		{
			Action: "continue", PromptType: "user",
			Name: "章节续写-用户提示",
			Content: `【小说背景】
{{.NovelDescription}}

【前文各章概要】
{{.PrevSummaries}}
{{- if .PrevContent}}

【前一章末尾内容】
{{.PrevContent}}
{{- end}}
{{- if .KnowledgeContext}}

【知识库参考】
{{.KnowledgeContext}}
{{- end}}

【当前章节】{{.ChapterTitle}}
【章节概要】{{.ChapterSummary}}
【当前章节已有内容】
{{.ChapterContent}}

请严格按照章节概要续写约 500-1000 字，确保与前文情节自然衔接，不要偏离概要设定的情节方向。`,
		},

		// ========== 大纲模板（outline_generate / outline_title_polish / outline_summary_polish / outline_summary_expand） ==========

		// outline_generate
		{
			Action: "outline_generate", PromptType: "system",
			Name: "大纲生成-系统提示",
			Content: `你是一位专业的小说策划师。根据用户提供的设定、人物和剧情思路，生成一个完整的小说大纲。
每个章节的 title 必须是具体的、有吸引力的标题（如"暗夜追踪"、"命运的抉择"），绝对不能使用"章节标题"、"章节题目"等占位符。
你必须严格按照以下 JSON 格式输出，不要包含任何其他文字：
[
  {"title": "第一章 暗夜追踪", "summary": "100-200字的章节概要..."},
  {"title": "第二章 命运的抉择", "summary": "100-200字的章节概要..."}
]{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}`,
		},
		{
			Action: "outline_generate", PromptType: "user",
			Name: "大纲生成-用户提示",
			Content: `【世界观/设定】
{{.Setting}}
{{- if .Characters}}

【主要人物】
{{.Characters}}
{{- end}}
{{- if .Background}}

【背景信息】
{{.Background}}
{{- end}}

【剧情思路】
{{.Plot}}

请生成约 {{.ChapterNum}} 个章节的大纲，每个章节包含标题和 100-200 字的概要。`,
		},

		// outline_title_polish
		{
			Action: "outline_title_polish", PromptType: "system",
			Name: "大纲标题润色-系统提示",
			Content: `你是一位专业的小说策划师。对用户提供的章节标题进行润色，使其更加精炼、有吸引力，同时保持与章节内容的关联性。只输出润色后的标题文本，不要包含任何解释或额外内容。{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}`,
		},
		{
			Action: "outline_title_polish", PromptType: "user",
			Name: "大纲标题润色-用户提示",
			Content: `{{- if .Setting}}【故事设定】
{{.Setting}}

{{end -}}
{{- if .PrevChapters}}【前文章节】
{{.PrevChapters}}

{{end -}}
{{- if .NextChapters}}【后续章节】
{{.NextChapters}}

{{end -}}
【当前章节标题】
{{.ChapterTitle}}
{{- if .ChapterSummary}}

【章节概要】
{{.ChapterSummary}}
{{- end}}

请润色上述章节标题，使其更加精炼、有吸引力。`,
		},

		// outline_summary_polish
		{
			Action: "outline_summary_polish", PromptType: "system",
			Name: "大纲概要润色-系统提示",
			Content: `你是一位专业的小说策划师。对用户提供的章节概要进行润色，使其更加清晰、连贯、有吸引力，保持原有情节方向不变。只输出润色后的概要文本，不要包含标题或额外说明。{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}`,
		},
		{
			Action: "outline_summary_polish", PromptType: "user",
			Name: "大纲概要润色-用户提示",
			Content: `{{- if .Setting}}【故事设定】
{{.Setting}}

{{end -}}
{{- if .PrevChapters}}【前文章节】
{{.PrevChapters}}

{{end -}}
{{- if .NextChapters}}【后续章节】
{{.NextChapters}}

{{end -}}
{{- if .ChapterTitle}}【章节标题】
{{.ChapterTitle}}

{{end -}}
【当前章节概要】
{{.ChapterSummary}}

请润色上述章节概要，使其更加清晰、连贯、有吸引力。`,
		},

		// outline_summary_expand
		{
			Action: "outline_summary_expand", PromptType: "system",
			Name: "大纲概要扩写-系统提示",
			Content: `你是一位专业的小说策划师。对用户提供的章节概要进行扩写，丰富情节细节、人物动机和场景描写，使概要更加充实完整。只输出扩写后的概要文本，不要包含标题或额外说明。{{if .WritingStyle}}

【写作规范】
{{.WritingStyle}}{{end}}`,
		},
		{
			Action: "outline_summary_expand", PromptType: "user",
			Name: "大纲概要扩写-用户提示",
			Content: `{{- if .Setting}}【故事设定】
{{.Setting}}

{{end -}}
{{- if .PrevChapters}}【前文章节】
{{.PrevChapters}}

{{end -}}
{{- if .NextChapters}}【后续章节】
{{.NextChapters}}

{{end -}}
{{- if .ChapterTitle}}【章节标题】
{{.ChapterTitle}}

{{end -}}
【当前章节概要】
{{.ChapterSummary}}

请扩写上述章节概要，丰富情节细节、人物动机和场景描写。`,
		},
	}
}
