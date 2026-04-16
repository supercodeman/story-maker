// server/internal/agent/executor_audio.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// AudioTaskExecutor 音频生成任务执行器（audio_gen）
type AudioTaskExecutor struct {
	ttsProvider TTSProvider
}

// NewAudioTaskExecutor 创建音频任务执行器
func NewAudioTaskExecutor(tts TTSProvider) *AudioTaskExecutor {
	return &AudioTaskExecutor{ttsProvider: tts}
}

// audioTaskParams 音频生成任务参数（从 AITask.Prompt 中解析）
type audioTaskParams struct {
	Text    string  `json:"text"`
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
	Emotion string  `json:"emotion"`
}

// Execute 执行音频生成任务
func (e *AudioTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	if e.ttsProvider == nil {
		return nil, fmt.Errorf("TTS provider not configured")
	}

	// 从 Task.Prompt 解析参数（JSON 格式）
	var params audioTaskParams
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &params); err != nil {
		// 兼容纯文本 prompt
		params = audioTaskParams{Text: ec.Task.Prompt}
	}

	if params.Text == "" {
		return nil, fmt.Errorf("text content is empty")
	}

	resp, err := e.ttsProvider.GenerateSpeech(ctx, &TTSRequest{
		Text:    params.Text,
		VoiceID: params.VoiceID,
		Speed:   params.Speed,
		Emotion: params.Emotion,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"audio_url": resp.AudioURL,
		"file_path": resp.FilePath,
		"duration":  resp.Duration,
	}, nil
}
