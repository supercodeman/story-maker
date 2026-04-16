// server/internal/agent/provider_minimax_tts.go
package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-curton/server/internal/storage"
)

// MiniMaxTTSProvider MiniMax 文本转语音 Provider
type MiniMaxTTSProvider struct {
	apiKey  string
	groupID string
	baseURL string
	storage storage.Storage
	client  *http.Client
}

// NewMiniMaxTTSProvider 创建 MiniMax TTS Provider 实例
func NewMiniMaxTTSProvider(apiKey, groupID, baseURL string, store storage.Storage) *MiniMaxTTSProvider {
	if baseURL == "" {
		baseURL = "https://api.minimax.chat/v1"
	}
	return &MiniMaxTTSProvider{
		apiKey:  apiKey,
		groupID: groupID,
		baseURL: baseURL,
		storage: store,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Name 返回 Provider 名称
func (p *MiniMaxTTSProvider) Name() string { return "minimax_tts" }

// SetAPIKey 动态注入 API Key
func (p *MiniMaxTTSProvider) SetAPIKey(key string) { p.apiKey = key }

// miniMaxTTSResponse MiniMax T2A API 响应体
type miniMaxTTSResponse struct {
	AudioFile string `json:"audio_file"` // base64 编码的音频数据
	ExtraInfo *struct {
		AudioLength float64 `json:"audio_length"` // 音频时长（毫秒）
	} `json:"extra_info,omitempty"`
	BaseResp *struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp,omitempty"`
}

// GenerateSpeech 调用 MiniMax T2A API 生成语音
func (p *MiniMaxTTSProvider) GenerateSpeech(ctx context.Context, req *TTSRequest) (*TTSResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax TTS API key not configured")
	}

	voiceID := req.VoiceID
	if voiceID == "" {
		voiceID = "female-yujie"
	}
	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}

	// 构造请求体
	body := map[string]interface{}{
		"model": "speech-01-turbo",
		"text":  req.Text,
		"voice_setting": map[string]interface{}{
			"voice_id": voiceID,
			"speed":    speed,
		},
		"audio_setting": map[string]interface{}{
			"sample_rate": 32000,
			"format":      "mp3",
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/t2a_v2?GroupId=%s", p.baseURL, p.groupID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("MiniMax TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MiniMax TTS API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var ttsResp miniMaxTTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ttsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if ttsResp.BaseResp != nil && ttsResp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("MiniMax TTS error: %s", ttsResp.BaseResp.StatusMsg)
	}

	if ttsResp.AudioFile == "" {
		return nil, fmt.Errorf("MiniMax TTS returned empty audio")
	}

	// 解码 base64 音频数据
	audioData, err := base64.StdEncoding.DecodeString(ttsResp.AudioFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

	// 保存到 Storage
	storagePath := fmt.Sprintf("audio/%d_%s.mp3", time.Now().UnixMilli(), voiceID)
	fileURL, err := p.storage.Upload(ctx, bytes.NewReader(audioData), storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to save audio: %w", err)
	}

	var duration float64
	if ttsResp.ExtraInfo != nil {
		duration = ttsResp.ExtraInfo.AudioLength / 1000.0
	}

	return &TTSResponse{
		AudioURL: fileURL,
		FilePath: storagePath,
		Duration: duration,
	}, nil
}
