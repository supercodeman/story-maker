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

// --- comicMockTTSProvider 漫剧测试用 TTS Provider mock ---

type comicMockTTSProvider struct {
	resp *TTSResponse
	err  error
}

func (m *comicMockTTSProvider) GenerateSpeech(ctx context.Context, req *TTSRequest) (*TTSResponse, error) {
	return m.resp, m.err
}
func (m *comicMockTTSProvider) Name() string { return "comic_mock_tts" }

// --- comicMockVideoProvider 漫剧测试用 Video Provider mock ---

type comicMockVideoProvider struct {
	resp *VideoGenResponse
	err  error
}

func (m *comicMockVideoProvider) GenerateVideo(ctx context.Context, req *VideoGenRequest) (*VideoGenResponse, error) {
	return m.resp, m.err
}
func (m *comicMockVideoProvider) Name() string { return "comic_mock_video" }

// --- ComicAudioExecutor Tests ---

func TestComicAudioExecutor_Success(t *testing.T) {
	tts := &comicMockTTSProvider{
		resp: &TTSResponse{AudioURL: "https://example.com/audio.mp3", FilePath: "/tmp/audio.mp3", Duration: 2.5},
	}
	input := comicAudioInput{Text: "你好世界", VoiceID: "voice_001", Speed: 1.2, Emotion: "happy", StoryboardID: 10}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicAudioExecutor(tts)
	result, err := executor.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := result.(*comicAudioResult)
	if !ok {
		t.Fatalf("result should be *comicAudioResult, got %T", result)
	}
	if r.AudioURL != "https://example.com/audio.mp3" {
		t.Fatalf("unexpected audio_url: %s", r.AudioURL)
	}
	if r.StoryboardID != 10 {
		t.Fatalf("expected storyboard_id 10, got %d", r.StoryboardID)
	}
}

func TestComicAudioExecutor_MissingText(t *testing.T) {
	tts := &comicMockTTSProvider{}
	input := comicAudioInput{VoiceID: "voice_001"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicAudioExecutor(tts)
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for missing text")
	}
}

func TestComicAudioExecutor_MissingVoiceID(t *testing.T) {
	tts := &comicMockTTSProvider{}
	input := comicAudioInput{Text: "hello"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicAudioExecutor(tts)
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for missing voice_id")
	}
}

func TestComicAudioExecutor_TTSError(t *testing.T) {
	tts := &comicMockTTSProvider{err: fmt.Errorf("tts service unavailable")}
	input := comicAudioInput{Text: "hello", VoiceID: "v1"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicAudioExecutor(tts)
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error when TTS fails")
	}
}

func TestComicAudioExecutor_DefaultSpeed(t *testing.T) {
	tts := &comicMockTTSProvider{
		resp: &TTSResponse{AudioURL: "url", FilePath: "/tmp/a.mp3", Duration: 1.0},
	}
	input := comicAudioInput{Text: "test", VoiceID: "v1", Speed: 0}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicAudioExecutor(tts)
	_, err := executor.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ComicMediaExecutor Tests ---

func TestComicMediaExecutor_VideoSuccess(t *testing.T) {
	video := &comicMockVideoProvider{
		resp: &VideoGenResponse{VideoURL: "https://example.com/video.mp4", FilePath: "/tmp/video.mp4", Duration: 4.0},
	}
	input := comicMediaInput{FrameDesc: "城市天台远景", MediaType: "video", Importance: "high", Duration: 4.0, StoryboardID: 5}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}, ModelVersion: "cogvideox-2"}
	executor := NewComicMediaExecutor(video, nil)
	result, err := executor.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := result.(*comicMediaResult)
	if !ok {
		t.Fatalf("result should be *comicMediaResult, got %T", result)
	}
	if r.MediaURL != "https://example.com/video.mp4" {
		t.Fatalf("unexpected media_url: %s", r.MediaURL)
	}
	if r.StoryboardID != 5 {
		t.Fatalf("expected storyboard_id 5, got %d", r.StoryboardID)
	}
}

func TestComicMediaExecutor_StaticImageFallback(t *testing.T) {
	imgProvider := &comicMockImageGenProvider{
		generateResp: &T2IResponse{Images: []ImageResult{{URL: "https://example.com/img.png", FilePath: "/tmp/img.png"}}},
	}
	input := comicMediaInput{FrameDesc: "背景远景", MediaType: "static_image", Importance: "low", Duration: 2.0, StoryboardID: 3}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicMediaExecutor(nil, imgProvider)
	result, err := executor.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := result.(*comicMediaResult)
	if !ok {
		t.Fatalf("result should be *comicMediaResult, got %T", result)
	}
	if r.MediaType != "static_image" {
		t.Fatalf("expected media_type static_image, got %s", r.MediaType)
	}
}

func TestComicMediaExecutor_MissingFrameDesc(t *testing.T) {
	executor := NewComicMediaExecutor(nil, nil)
	input := comicMediaInput{MediaType: "video"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error for missing frame_desc")
	}
}

func TestComicMediaExecutor_NoVideoProvider(t *testing.T) {
	input := comicMediaInput{FrameDesc: "test", MediaType: "video", Importance: "high"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicMediaExecutor(nil, nil)
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error when VideoProvider is nil")
	}
}

func TestComicMediaExecutor_NoImageProvider(t *testing.T) {
	input := comicMediaInput{FrameDesc: "test", MediaType: "static_image", Importance: "low"}
	inputJSON, _ := json.Marshal(input)

	ec := &ExecContext{Task: &model.AITask{Prompt: string(inputJSON)}}
	executor := NewComicMediaExecutor(nil, nil)
	_, err := executor.Execute(context.Background(), ec)
	if err == nil {
		t.Fatal("expected error when ImageGenProvider is nil")
	}
}
