package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// comicScriptInput 漫剧剧本生成输入
type comicScriptInput struct {
	ChapterContent string              `json:"chapter_content"`
	Characters     []comicCharacterDef `json:"characters"`
	Worldview      string              `json:"worldview"`
	Style          string              `json:"style"`
	MaxScenes      int                 `json:"max_scenes"`
}

// comicCharacterDef 角色定义
type comicCharacterDef struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// ComicScriptOutput 剧本生成输出
type ComicScriptOutput struct {
	Scenes []ComicScriptScene `json:"scenes"`
}

// ComicScriptScene 单个场景
type ComicScriptScene struct {
	SeqNo     int             `json:"seq_no"`
	SceneDesc string          `json:"scene_desc"`
	Dialogue  []SceneDialogue `json:"dialogue"`
	Emotion   string          `json:"emotion"`
	MediaType string          `json:"media_type"`
	Duration  float64         `json:"duration"`
}

// SceneDialogue 场景对白
type SceneDialogue struct {
	Character string `json:"character"`
	Line      string `json:"line"`
	Emotion   string `json:"emotion"`
}

// ComicScriptExecutor 漫剧剧本生成执行器
type ComicScriptExecutor struct{}

func (e *ComicScriptExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	var input comicScriptInput
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &input); err != nil {
		return nil, fmt.Errorf("failed to parse comic script input: %w", err)
	}
	if input.ChapterContent == "" {
		return nil, fmt.Errorf("chapter_content is required")
	}
	if input.MaxScenes <= 0 {
		input.MaxScenes = 8
	}

	systemPrompt := e.buildSystemPrompt(input)
	userPrompt := fmt.Sprintf("请将以下小说章节内容改编为漫剧剧本：\n\n%s", input.ChapterContent)

	req := &TextRequest{
		Model:        ec.ModelVersion,
		Prompt:       userPrompt,
		CharacterCtx: systemPrompt,
		MaxTokens:    8192,
		Temperature:  0.7,
	}

	resp, err := ec.Provider.GenerateText(ctx, req)
	if err != nil {
		return nil, err
	}

	output, err := e.parseScriptOutput(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI script output: %w", err)
	}

	if len(output.Scenes) == 0 {
		return nil, fmt.Errorf("AI returned empty scenes list")
	}

	return map[string]interface{}{
		"scenes": output.Scenes,
		"usage":  resp.Usage,
	}, nil
}

// buildSystemPrompt 构建系统提示词
func (e *ComicScriptExecutor) buildSystemPrompt(input comicScriptInput) string {
	var sb strings.Builder
	sb.WriteString(`你是一个专业的漫剧编剧。你的任务是将小说章节改编为适合漫剧呈现的结构化剧本。

## 输出格式要求
你必须严格按照以下 JSON schema 输出，不要输出任何其他内容：

{
  "scenes": [
    {
      "seq_no": 1,
      "scene_desc": "场景的视觉描述，用于后续生成画面",
      "dialogue": [
        {"character": "角色名", "line": "台词内容", "emotion": "情绪标签"}
      ],
      "emotion": "场景整体情绪基调(如 warm/tense/melancholy/exciting/calm)",
      "media_type": "video 或 dynamic_image",
      "duration": 5.0
    }
  ]
}

## 规则
1. 每个场景的 scene_desc 要具体、有画面感，包含环境、光线、人物动作
2. media_type 选择标准：动作激烈/转场关键用 "video"，静态情绪/过渡用 "dynamic_image"
3. duration 范围 2-8 秒，根据场景复杂度和对白长度决定
4. dialogue 中的 emotion 使用英文标签：neutral/happy/sad/angry/surprised/fearful/disgusted
5. 场景数量控制在 3-`)
	sb.WriteString(fmt.Sprintf("%d 个之间", input.MaxScenes))

	if input.Worldview != "" {
		sb.WriteString(fmt.Sprintf("\n\n## 世界观设定\n%s", input.Worldview))
	}
	if len(input.Characters) > 0 {
		sb.WriteString("\n\n## 角色列表\n")
		for _, c := range input.Characters {
			sb.WriteString(fmt.Sprintf("- %s：%s\n", c.Name, c.Desc))
		}
	}
	if input.Style != "" {
		sb.WriteString(fmt.Sprintf("\n\n## 风格偏好\n%s", input.Style))
	}
	return sb.String()
}

// parseScriptOutput 解析 AI 返回的剧本 JSON
func (e *ComicScriptExecutor) parseScriptOutput(content string) (*ComicScriptOutput, error) {
	var output ComicScriptOutput
	if err := json.Unmarshal([]byte(content), &output); err == nil {
		return &output, nil
	}
	cleaned := extractJSONFromMarkdown(content)
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return nil, fmt.Errorf("cannot parse script JSON: %w, raw content: %.200s", err, content)
	}
	return &output, nil
}
