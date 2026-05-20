package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// comicAudioInput 漫剧音频生成输入
type comicAudioInput struct {
	Text         string  `json:"text"`
	VoiceID      string  `json:"voice_id"`
	Speed        float64 `json:"speed"`
	Emotion      string  `json:"emotion"`
	StoryboardID uint    `json:"storyboard_id"`
}

// comicAudioResult 漫剧音频生成结果
type comicAudioResult struct {
	AudioURL     string  `json:"audio_url"`
	FilePath     string  `json:"file_path"`
	Duration     float64 `json:"duration"`
	StoryboardID uint    `json:"storyboard_id"`
}

// ComicAudioExecutor 漫剧音频生成执行器，基于 TTS Provider 将台词转为语音
type ComicAudioExecutor struct {
	ttsProvider TTSProvider
}

// NewComicAudioExecutor 创建漫剧音频执行器
func NewComicAudioExecutor(tts TTSProvider) *ComicAudioExecutor {
	return &ComicAudioExecutor{ttsProvider: tts}
}

// Execute 执行音频生成任务
func (e *ComicAudioExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	var input comicAudioInput
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &input); err != nil {
		return nil, fmt.Errorf("failed to parse audio input: %w", err)
	}
	if input.Text == "" {
		return nil, fmt.Errorf("text is required for audio generation")
	}
	if input.VoiceID == "" {
		return nil, fmt.Errorf("voice_id is required for audio generation")
	}
	speed := input.Speed
	if speed <= 0 {
		speed = 1.0
	}

	resp, err := e.ttsProvider.GenerateSpeech(ctx, &TTSRequest{
		Text:    input.Text,
		VoiceID: input.VoiceID,
		Speed:   speed,
		Emotion: input.Emotion,
	})
	if err != nil {
		return nil, fmt.Errorf("TTS generation failed: %w", err)
	}

	return &comicAudioResult{
		AudioURL:     resp.AudioURL,
		FilePath:     resp.FilePath,
		Duration:     resp.Duration,
		StoryboardID: input.StoryboardID,
	}, nil
}
