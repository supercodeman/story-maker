// server/internal/agent/provider_minimax_tts.go
package agent

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"story-maker/server/internal/storage"
)

// MiniMaxTTSProvider MiniMax 文本转语音 Provider
// 官方文档：POST https://api.minimaxi.com/v1/t2a_v2
// 鉴权：Bearer Token
type MiniMaxTTSProvider struct {
	apiKey  string
	groupID string
	baseURL string
	model   string // 合成模型，默认 speech-01-turbo
	storage storage.Storage
	client  *http.Client
}

// NewMiniMaxTTSProvider 创建 MiniMax TTS Provider 实例
// baseURL 为空时使用官方默认；传入已兼容 "/v1" 结尾或完整 endpoint（如 ".../v1/text/chatcompletion_v2"）两种写法
func NewMiniMaxTTSProvider(apiKey, groupID, baseURL string, store storage.Storage) *MiniMaxTTSProvider {
	return &MiniMaxTTSProvider{
		apiKey:  apiKey,
		groupID: groupID,
		baseURL: normalizeMiniMaxBaseURL(baseURL),
		model:   "speech-01-turbo",
		storage: store,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// normalizeMiniMaxBaseURL 规整 base_url：无论传入的是根域名、/v1、还是具体 endpoint，统一归一到 ".../v1"
func normalizeMiniMaxBaseURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "https://api.minimaxi.com/v1"
	}
	s = strings.TrimRight(s, "/")
	// 如果传入的是具体接口路径（如 .../v1/text/chatcompletion_v2），截到 /v1
	if idx := strings.Index(s, "/v1"); idx != -1 {
		return s[:idx+3]
	}
	return s + "/v1"
}

// Name 返回 Provider 名称
func (p *MiniMaxTTSProvider) Name() string { return "minimax_tts" }

// SetAPIKey 动态注入 API Key
func (p *MiniMaxTTSProvider) SetAPIKey(key string) { p.apiKey = key }

// miniMaxTTSResponse 对齐官方 OpenAPI：{ data, extra_info, trace_id, base_resp }
type miniMaxTTSResponse struct {
	Data *struct {
		Audio  string `json:"audio"`  // 当 output_format=hex 时为 hex 编码；url 模式下该字段为下载链接
		Status int    `json:"status"` // 1 合成中，2 合成结束
	} `json:"data,omitempty"`
	ExtraInfo *struct {
		AudioLength     int64  `json:"audio_length"` // 毫秒
		AudioSampleRate int64  `json:"audio_sample_rate"`
		AudioSize       int64  `json:"audio_size"`
		Bitrate         int64  `json:"bitrate"`
		AudioFormat     string `json:"audio_format"`
		AudioChannel    int    `json:"audio_channel"`
		UsageCharacters int64  `json:"usage_characters"`
		WordCount       int64  `json:"word_count"`
	} `json:"extra_info,omitempty"`
	TraceID  string `json:"trace_id,omitempty"`
	BaseResp *struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp,omitempty"`
}

// GenerateSpeech 调用 MiniMax T2A V2 API 生成语音
// - 文本 ≤ maxChunkChars：单次同步调用
// - 文本 > maxChunkChars：按段落/句子切分 → 串行调 API → MP3 字节拼接成单文件
// 采用 output_format=url 模式，API 直接返回音频 URL（24h 有效），由本方法下载落盘
func (p *MiniMaxTTSProvider) GenerateSpeech(ctx context.Context, req *TTSRequest) (*TTSResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax TTS API key not configured")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("text content is empty")
	}

	voiceID := req.VoiceID
	if voiceID == "" {
		voiceID = "female-yujie"
	}
	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}

	// 切分文本：每段 ≤ maxChunkChars（3000，对齐官方推荐流式阈值）
	chunks := splitTTSText(req.Text, maxChunkChars)

	var (
		combinedAudio bytes.Buffer
		totalDuration float64
	)
	for i, chunk := range chunks {
		audioBytes, chunkDurationMs, err := p.synthesizeChunk(ctx, chunk, voiceID, speed, req.Emotion)
		if err != nil {
			return nil, fmt.Errorf("tts chunk %d/%d failed: %w", i+1, len(chunks), err)
		}
		combinedAudio.Write(audioBytes)
		totalDuration += float64(chunkDurationMs) / 1000.0
	}

	// 保存到 Storage
	storagePath := fmt.Sprintf("audio/%d_%s.mp3", time.Now().UnixMilli(), voiceID)
	fileURL, err := p.storage.Upload(ctx, bytes.NewReader(combinedAudio.Bytes()), storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to save audio: %w", err)
	}

	return &TTSResponse{
		AudioURL: fileURL,
		FilePath: storagePath,
		Duration: totalDuration,
	}, nil
}

// maxChunkChars 单次 TTS 请求最大字符数
// 官方限制：单次 text ≤ 10000；超过 3000 推荐流式
// 我们用 3000 同步分片，稳定性优先
const maxChunkChars = 3000

// synthesizeChunk 单次调用 MiniMax T2A V2，返回音频字节和毫秒时长
func (p *MiniMaxTTSProvider) synthesizeChunk(ctx context.Context, text, voiceID string, speed float64, emotion string) ([]byte, int64, error) {
	voiceSetting := map[string]interface{}{
		"voice_id": voiceID,
		"speed":    speed,
	}
	if emotion != "" && isValidMiniMaxEmotion(emotion) {
		voiceSetting["emotion"] = emotion
	}

	body := map[string]interface{}{
		"model":         p.model,
		"text":          text,
		"stream":        false,
		"output_format": "url",
		"voice_setting": voiceSetting,
		"audio_setting": map[string]interface{}{
			"sample_rate": 32000,
			"bitrate":     128000,
			"format":      "mp3",
			"channel":     1,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := p.baseURL + "/t2a_v2"
	if p.groupID != "" {
		endpoint += "?GroupId=" + p.groupID
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("minimax TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("minimax TTS API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var ttsResp miniMaxTTSResponse
	if err := json.Unmarshal(respBytes, &ttsResp); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}
	if ttsResp.BaseResp != nil && ttsResp.BaseResp.StatusCode != 0 {
		return nil, 0, fmt.Errorf("minimax TTS error [code=%d]: %s", ttsResp.BaseResp.StatusCode, describeMiniMaxError(ttsResp.BaseResp.StatusCode, ttsResp.BaseResp.StatusMsg))
	}
	if ttsResp.Data == nil || ttsResp.Data.Audio == "" {
		return nil, 0, fmt.Errorf("minimax TTS returned empty audio (trace_id=%s)", ttsResp.TraceID)
	}

	audioData, err := fetchMiniMaxAudio(ctx, p.client, ttsResp.Data.Audio)
	if err != nil {
		return nil, 0, err
	}

	var durationMs int64
	if ttsResp.ExtraInfo != nil {
		durationMs = ttsResp.ExtraInfo.AudioLength
	}
	return audioData, durationMs, nil
}

// splitTTSText 按字符数上限切分文本，尽量在段落/句子边界断开，保证合成自然
// 规则：
//  1. 先按双换行切段落；段落若仍过长，按 。！？!?.\n 切句；
//  2. 单个"句子"仍过长（极端情况），按 maxLen 硬切；
//  3. 不产生空串。
func splitTTSText(text string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = maxChunkChars
	}
	if runesLen(text) <= maxLen {
		return []string{text}
	}

	var result []string
	var buf strings.Builder

	flushBuf := func() {
		if buf.Len() > 0 {
			result = append(result, strings.TrimSpace(buf.String()))
			buf.Reset()
		}
	}

	// 段落级切分
	paragraphs := strings.Split(text, "\n\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		// 句子级切分
		for _, sentence := range splitIntoSentences(para) {
			if sentence == "" {
				continue
			}
			// 若当前缓冲 + 分隔符 + 新句 > maxLen，先 flush
			sep := 0
			if buf.Len() > 0 {
				sep = 1 // "\n"
			}
			if runesLen(buf.String())+sep+runesLen(sentence) > maxLen && buf.Len() > 0 {
				flushBuf()
			}
			// 单句本身超 maxLen，硬切
			if runesLen(sentence) > maxLen {
				flushBuf()
				for _, piece := range hardSplit(sentence, maxLen) {
					result = append(result, piece)
				}
				continue
			}
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(sentence)
		}
		flushBuf()
	}
	flushBuf()

	if len(result) == 0 {
		return []string{text}
	}
	return result
}

// splitIntoSentences 按中英文常见句末标点切句，保留标点在句尾
func splitIntoSentences(s string) []string {
	var sentences []string
	var buf strings.Builder
	for _, r := range s {
		buf.WriteRune(r)
		switch r {
		case '。', '！', '？', '!', '?', '.', '\n':
			if buf.Len() > 0 {
				sentences = append(sentences, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
		}
	}
	if buf.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(buf.String()))
	}
	return sentences
}

// hardSplit 按 rune 数硬切，用于处理极长无标点文本
func hardSplit(s string, maxLen int) []string {
	runes := []rune(s)
	var result []string
	for i := 0; i < len(runes); i += maxLen {
		end := i + maxLen
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[i:end]))
	}
	return result
}

// runesLen 返回字符串的 rune 数（中文按字计，不按字节）
func runesLen(s string) int {
	return len([]rune(s))
}

// fetchMiniMaxAudio 从 MiniMax 返回的 audio 字段拿到音频字节
// 兼容两种情况：url 模式返回 http(s) 链接；hex 模式返回 hex 编码字符串
func fetchMiniMaxAudio(ctx context.Context, client *http.Client, audio string) ([]byte, error) {
	if strings.HasPrefix(audio, "http://") || strings.HasPrefix(audio, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, audio, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create audio download request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to download audio: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download audio failed: status %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
	// hex 兜底（官方流式或 output_format=hex 时使用）
	data, err := hex.DecodeString(audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex audio: %w", err)
	}
	return data, nil
}

// isValidMiniMaxEmotion 校验 emotion 参数合法性（白名单）
// 官方文档：happy / sad / angry / fearful / disgusted / surprised / calm / fluent / whisper
func isValidMiniMaxEmotion(e string) bool {
	switch e {
	case "happy", "sad", "angry", "fearful", "disgusted", "surprised", "calm", "fluent", "whisper":
		return true
	}
	return false
}

// describeMiniMaxError 对已知的 MiniMax 业务错误码附加中文说明，便于排查
// 未知错误码原样返回 status_msg
func describeMiniMaxError(code int, msg string) string {
	var hint string
	switch code {
	case 1002:
		hint = "触发限流，请稍后重试"
	case 1004:
		hint = "鉴权失败，请检查 API Key 是否正确"
	case 1008:
		hint = "账号余额不足，请充值"
	case 1026:
		hint = "文本涉及敏感内容，被安全策略拦截"
	case 1039:
		hint = "触发 TPM 限流，请稍后重试"
	case 1042:
		hint = "非法字符占比超过 10%"
	case 2013:
		hint = "请求参数异常，请检查入参"
	case 2049:
		hint = "无效的 API Key"
	case 2056:
		hint = "账号 Token 用量达到计划上限（如 5 小时窗口），请等窗口重置或升级计划"
	}
	if hint == "" {
		return msg
	}
	return fmt.Sprintf("%s（%s）", msg, hint)
}
