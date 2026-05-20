// server/internal/agent/executor_audio.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"story-maker/server/internal/model"
)

// AssetWriter 资产写入接口，由 dao.AssetDAO 实现
// 以接口解耦：agent 不反向依赖 dao 包，由 router 注入时绑定实现
type AssetWriter interface {
	Create(a *model.Asset) error
}

// AudioTaskExecutor 音频生成任务执行器（audio_gen）
type AudioTaskExecutor struct {
	ttsProvider TTSProvider
	assetWriter AssetWriter
}

// NewAudioTaskExecutor 创建音频任务执行器
// assetWriter 可为 nil（仅在单元测试或未接入存储时），生产必须传入
func NewAudioTaskExecutor(tts TTSProvider, assetWriter AssetWriter) *AudioTaskExecutor {
	return &AudioTaskExecutor{ttsProvider: tts, assetWriter: assetWriter}
}

// audioTaskParams 音频生成任务参数（从 AITask.Prompt 中解析）
type audioTaskParams struct {
	Text    string  `json:"text"`
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
	Emotion string  `json:"emotion"`
}

// Execute 执行音频生成任务：调 TTS → 落盘（Provider 内部完成）→ 写 Asset 记录
func (e *AudioTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	if e.ttsProvider == nil {
		return nil, fmt.Errorf("TTS provider not configured")
	}

	// 从 Task.Prompt 解析参数（JSON 格式），兼容纯文本
	var params audioTaskParams
	if err := json.Unmarshal([]byte(ec.Task.Prompt), &params); err != nil {
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

	// 写入 Asset 表，前端才能在章节详情页查询到
	if e.assetWriter != nil {
		metadata, _ := json.Marshal(map[string]interface{}{
			"provider": ec.Task.ModelName,
			"voice_id": params.VoiceID,
			"speed":    params.Speed,
			"emotion":  params.Emotion,
		})
		asset := &model.Asset{
			PortfolioID: ec.Task.PortfolioID,
			Type:        model.AssetTypeAudio,
			FilePath:    resp.FilePath,
			Duration:    resp.Duration,
			ChapterID:   ec.Task.ChapterID,
			CreatedBy:   ec.Task.UserID,
			Metadata:    string(metadata),
		}
		if err := e.assetWriter.Create(asset); err != nil {
			// Asset 写入失败不应让任务失败（音频已落盘），但要记录以便排查
			return nil, fmt.Errorf("audio generated but failed to create asset record: %w", err)
		}
	}

	return map[string]interface{}{
		"audio_url": resp.AudioURL,
		"file_path": resp.FilePath,
		"duration":  resp.Duration,
	}, nil
}
