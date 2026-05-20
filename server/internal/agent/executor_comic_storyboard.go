package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// storyboardInput 分镜拆分输入
type storyboardInput struct {
	Scenes []ComicScriptScene `json:"scenes"`
	Style  string             `json:"style"`
}

// StoryboardOutput 分镜拆分输出
type StoryboardOutput struct {
	Frames []StoryboardFrame `json:"frames"`
}

// StoryboardFrame 单个分镜帧
type StoryboardFrame struct {
	SceneSeqNo  int      `json:"scene_seq_no"`
	FrameSeqNo  int      `json:"frame_seq_no"`
	FrameDesc   string   `json:"frame_desc"`
	CameraAngle string   `json:"camera_angle"`
	Characters  []string `json:"characters"`
	Importance  string   `json:"importance"`
	MediaType   string   `json:"media_type"`
	Duration    float64  `json:"duration"`
}

// ComicStoryboardExecutor 漫剧分镜拆分执行器
type ComicStoryboardExecutor struct{}

func (e *ComicStoryboardExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	var input storyboardInput
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &input); err != nil {
		return nil, fmt.Errorf("failed to parse storyboard input: %w", err)
	}
	if len(input.Scenes) == 0 {
		return nil, fmt.Errorf("scenes list is empty, nothing to split")
	}

	systemPrompt := e.buildSystemPrompt(input)
	scenesJSON, _ := json.MarshalIndent(input.Scenes, "", "  ")
	userPrompt := fmt.Sprintf("请将以下剧本场景拆分为具体分镜：\n\n%s", string(scenesJSON))

	req := &TextRequest{
		Model:        ec.ModelVersion,
		Prompt:       userPrompt,
		CharacterCtx: systemPrompt,
		MaxTokens:    8192,
		Temperature:  0.5,
	}

	resp, err := ec.Provider.GenerateText(ctx, req)
	if err != nil {
		return nil, err
	}

	output, err := e.parseOutput(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse storyboard output: %w", err)
	}
	if len(output.Frames) == 0 {
		return nil, fmt.Errorf("AI returned empty frames list")
	}

	return map[string]interface{}{
		"frames": output.Frames,
		"usage":  resp.Usage,
	}, nil
}

// buildSystemPrompt 构建分镜师系统提示词
func (e *ComicStoryboardExecutor) buildSystemPrompt(input storyboardInput) string {
	var sb strings.Builder
	sb.WriteString(`你是一个专业的漫剧分镜师。你的任务是将剧本场景拆分为具体的镜头画面。

## 输出格式要求
严格按照以下 JSON schema 输出，不要输出任何其他内容：

{
  "frames": [
    {
      "scene_seq_no": 1,
      "frame_seq_no": 1,
      "frame_desc": "具体的画面描述，包含构图、光线、人物动作等细节",
      "camera_angle": "wide_shot",
      "characters": ["角色名"],
      "importance": "high",
      "media_type": "video",
      "duration": 3.0
    }
  ]
}

## 镜头角度选项
- wide_shot: 远景/全景
- medium_shot: 中景
- close_up: 近景/特写
- extreme_close_up: 极端特写

## importance 标注规则
- high: 情节转折点、高潮、关键动作
- medium: 重要但非关键的过渡镜头
- low: 环境铺垫、静态展示

## media_type 决策规则
- video: importance=high 且有明显动作
- dynamic_image: importance=medium 或有轻微动态
- static_image: importance=low 的纯静态展示

## 拆分规则
1. 每个场景拆分为 2-4 个镜头
2. 镜头之间要有节奏感
3. frame_desc 要足够具体，能直接作为图像生成 prompt
4. duration 总和应接近原场景的 duration`)

	if input.Style != "" {
		sb.WriteString(fmt.Sprintf("\n\n## 视觉风格\n%s", input.Style))
	}
	return sb.String()
}

// parseOutput 解析 AI 返回的分镜 JSON
func (e *ComicStoryboardExecutor) parseOutput(content string) (*StoryboardOutput, error) {
	var output StoryboardOutput
	if err := json.Unmarshal([]byte(content), &output); err == nil {
		return &output, nil
	}
	cleaned := extractJSONFromMarkdown(content)
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return nil, fmt.Errorf("cannot parse storyboard JSON: %w, raw: %.200s", err, content)
	}
	return &output, nil
}
