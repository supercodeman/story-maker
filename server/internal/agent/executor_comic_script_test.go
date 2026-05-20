package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"story-maker/server/internal/model"
)

// comicMockAIProvider 漫剧测试用 AI Provider mock
type comicMockAIProvider struct {
	generateTextResp *TextResponse
	generateTextErr  error
	lastTextReq      *TextRequest
}

func (m *comicMockAIProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	m.lastTextReq = req
	return m.generateTextResp, m.generateTextErr
}
func (m *comicMockAIProvider) GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	return nil, nil
}
func (m *comicMockAIProvider) AdjustCharacter(ctx context.Context, req *CharacterAdjustRequest) (*ImageResponse, error) {
	return nil, nil
}
func (m *comicMockAIProvider) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, nil
}
func (m *comicMockAIProvider) Name() string           { return "comic_mock" }
func (m *comicMockAIProvider) Capabilities() []string { return []string{"text"} }
func (m *comicMockAIProvider) FallbackModels() []string { return nil }

// comicMockImageGenProvider 漫剧测试用图片生成 Provider mock
type comicMockImageGenProvider struct {
	generateResp *T2IResponse
	generateErr  error
	lastReq      *T2IRequest
	callCount    int
}

func (m *comicMockImageGenProvider) GenerateImages(ctx context.Context, req *T2IRequest) (*T2IResponse, error) {
	m.lastReq = req
	m.callCount++
	return m.generateResp, m.generateErr
}
func (m *comicMockImageGenProvider) Name() string { return "comic_mock_image" }

// --- ComicScriptExecutor Tests ---

func TestComicScriptExecutor_Success(t *testing.T) {
	scriptOutput := ComicScriptOutput{
		Scenes: []ComicScriptScene{
			{
				SeqNo:     1,
				SceneDesc: "夕阳下的城市天台，少女站在栏杆旁",
				Dialogue: []SceneDialogue{
					{Character: "小雨", Line: "这里的风景真美", Emotion: "happy"},
				},
				Emotion:   "warm",
				MediaType: "dynamic_image",
				Duration:  4.0,
			},
			{
				SeqNo:     2,
				SceneDesc: "突然一阵强风吹来，少女的帽子被吹走",
				Dialogue:  []SceneDialogue{},
				Emotion:   "exciting",
				MediaType: "video",
				Duration:  3.0,
			},
		},
	}
	respJSON, _ := json.Marshal(scriptOutput)

	provider := &comicMockAIProvider{
		generateTextResp: &TextResponse{
			Content: string(respJSON),
			Usage:   &TokenUsage{PromptTokens: 100, CompletionTokens: 200, TotalTokens: 300},
		},
	}

	input := comicScriptInput{
		ChapterContent: "小雨来到城市天台，看着夕阳下的风景...",
		Characters:     []comicCharacterDef{{Name: "小雨", Desc: "16岁少女"}},
		MaxScenes:      5,
	}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider:     provider,
		Task:         &model.AITask{Prompt: string(inputJSON)},
		ModelVersion: "test-model",
	}

	executor := &ComicScriptExecutor{}
	result, err := executor.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result should be map, got %T", result)
	}
	scenes, ok := resultMap["scenes"].([]ComicScriptScene)
	if !ok {
		t.Fatalf("scenes should be []ComicScriptScene, got %T", resultMap["scenes"])
	}
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}
	if scenes[0].SceneDesc == "" {
		t.Fatal("scene_desc should not be empty")
	}
}

func TestComicScriptExecutor_InvalidJSON(t *testing.T) {
	provider := &comicMockAIProvider{
		generateTextResp: &TextResponse{
			Content: "this is not valid json at all",
		},
	}

	input := comicScriptInput{ChapterContent: "some content"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider: provider,
		Task:     &model.AITask{Prompt: string(inputJSON)},
	}

	executor := &ComicScriptExecutor{}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestComicScriptExecutor_ProviderError(t *testing.T) {
	provider := &comicMockAIProvider{
		generateTextErr: fmt.Errorf("provider unavailable"),
	}

	input := comicScriptInput{ChapterContent: "some content"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider: provider,
		Task:     &model.AITask{Prompt: string(inputJSON)},
	}

	executor := &ComicScriptExecutor{}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
	if err.Error() != "provider unavailable" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestComicScriptExecutor_EmptyScenes(t *testing.T) {
	emptyOutput := ComicScriptOutput{Scenes: []ComicScriptScene{}}
	respJSON, _ := json.Marshal(emptyOutput)

	provider := &comicMockAIProvider{
		generateTextResp: &TextResponse{Content: string(respJSON)},
	}

	input := comicScriptInput{ChapterContent: "some content"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider: provider,
		Task:     &model.AITask{Prompt: string(inputJSON)},
	}

	executor := &ComicScriptExecutor{}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for empty scenes")
	}
}

// --- ComicStoryboardExecutor Tests ---

func TestComicStoryboardExecutor_Success(t *testing.T) {
	storyboardOutput := StoryboardOutput{
		Frames: []StoryboardFrame{
			{
				SceneSeqNo:  1,
				FrameSeqNo:  1,
				FrameDesc:   "远景：夕阳下的城市天台全貌",
				CameraAngle: "wide_shot",
				Characters:  []string{"小雨"},
				Importance:  "medium",
				MediaType:   "dynamic_image",
				Duration:    2.0,
			},
			{
				SceneSeqNo:  1,
				FrameSeqNo:  2,
				FrameDesc:   "中景：少女倚靠栏杆，微风吹动发丝",
				CameraAngle: "medium_shot",
				Characters:  []string{"小雨"},
				Importance:  "high",
				MediaType:   "video",
				Duration:    2.0,
			},
		},
	}
	respJSON, _ := json.Marshal(storyboardOutput)

	provider := &comicMockAIProvider{
		generateTextResp: &TextResponse{
			Content: string(respJSON),
			Usage:   &TokenUsage{PromptTokens: 150, CompletionTokens: 250, TotalTokens: 400},
		},
	}

	input := storyboardInput{
		Scenes: []ComicScriptScene{
			{SeqNo: 1, SceneDesc: "天台场景", Duration: 4.0},
		},
		Style: "日系动漫风格",
	}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider:     provider,
		Task:         &model.AITask{Prompt: string(inputJSON)},
		ModelVersion: "test-model",
	}

	executor := &ComicStoryboardExecutor{}
	result, err := executor.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result should be map, got %T", result)
	}
	frames, ok := resultMap["frames"].([]StoryboardFrame)
	if !ok {
		t.Fatalf("frames should be []StoryboardFrame, got %T", resultMap["frames"])
	}
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
}

func TestComicStoryboardExecutor_EmptyInput(t *testing.T) {
	input := storyboardInput{Scenes: []ComicScriptScene{}}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider: &comicMockAIProvider{},
		Task:     &model.AITask{Prompt: string(inputJSON)},
	}

	executor := &ComicStoryboardExecutor{}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for empty scenes input")
	}
}

// --- ComicCharRefExecutor Tests ---

func TestComicCharRefExecutor_Success(t *testing.T) {
	imgProvider := &comicMockImageGenProvider{
		generateResp: &T2IResponse{
			Images: []ImageResult{
				{URL: "https://example.com/char1.png", FilePath: "/tmp/char1.png"},
			},
		},
	}

	input := charRefInput{
		Characters: []charRefCharacter{
			{Name: "小雨", Appearance: "黑色长发，大眼睛", StylePrompt: "anime style"},
			{Name: "阿明", Appearance: "短发，戴眼镜", StylePrompt: "anime style"},
		},
		AspectRatio: "2:3",
	}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider: &comicMockAIProvider{},
		Task:     &model.AITask{Prompt: string(inputJSON)},
	}

	executor := NewComicCharRefExecutor(imgProvider)
	result, err := executor.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result should be map, got %T", result)
	}
	refs, ok := resultMap["character_refs"].([]charRefResult)
	if !ok {
		t.Fatalf("character_refs should be []charRefResult, got %T", resultMap["character_refs"])
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if imgProvider.callCount != 2 {
		t.Fatalf("expected 2 image gen calls, got %d", imgProvider.callCount)
	}
}

func TestComicCharRefExecutor_NoProvider(t *testing.T) {
	input := charRefInput{
		Characters: []charRefCharacter{{Name: "test", Appearance: "test"}},
	}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{
		Provider: &comicMockAIProvider{},
		Task:     &model.AITask{Prompt: string(inputJSON)},
	}

	executor := NewComicCharRefExecutor(nil)
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error when imageProvider is nil")
	}
}
