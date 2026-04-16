// server/internal/agent/provider_cogvideo.go
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-curton/server/internal/storage"
)

// CogVideoProvider 智谱 CogVideoX 视频生成 Provider
type CogVideoProvider struct {
	apiKey  string
	baseURL string
	storage storage.Storage
	client  *http.Client
}

// NewCogVideoProvider 创建智谱视频生成 Provider 实例
func NewCogVideoProvider(apiKey, baseURL string, store storage.Storage) *CogVideoProvider {
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	return &CogVideoProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		storage: store,
		client:  &http.Client{Timeout: 600 * time.Second}, // 视频生成耗时较长
	}
}

// Name 返回 Provider 名称
func (p *CogVideoProvider) Name() string { return "cogvideo" }

// SetAPIKey 动态注入 API Key
func (p *CogVideoProvider) SetAPIKey(key string) { p.apiKey = key }

// cogVideoSubmitResponse 提交任务响应
type cogVideoSubmitResponse struct {
	ID         string `json:"id"`
	TaskStatus string `json:"task_status"` // PROCESSING, SUCCESS, FAIL
}

// cogVideoQueryResponse 查询任务响应
type cogVideoQueryResponse struct {
	TaskStatus string `json:"task_status"`
	VideoResult []struct {
		URL          string  `json:"url"`
		CoverImageURL string `json:"cover_image_url"`
	} `json:"video_result,omitempty"`
}

// GenerateVideo 调用智谱 CogVideoX API 生成视频（异步轮询）
func (p *CogVideoProvider) GenerateVideo(ctx context.Context, req *VideoGenRequest) (*VideoGenResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("CogVideo API key not configured")
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "cogvideox"
	}

	// 1. 提交视频生成任务
	taskID, err := p.submitTask(ctx, modelName, req.Prompt)
	if err != nil {
		return nil, err
	}

	// 2. 轮询任务状态（最多等待 10 分钟）
	videoURL, err := p.pollResult(ctx, taskID, 10*time.Minute)
	if err != nil {
		return nil, err
	}

	// 3. 下载视频并保存到 Storage
	storagePath := fmt.Sprintf("video/%d_%s.mp4", time.Now().UnixMilli(), taskID)
	fileURL, err := p.downloadAndSave(ctx, videoURL, storagePath)
	if err != nil {
		return nil, err
	}

	return &VideoGenResponse{
		VideoURL: fileURL,
		FilePath: storagePath,
		Duration: 6.0, // CogVideoX 默认生成 6 秒视频
	}, nil
}

// submitTask 提交视频生成任务
func (p *CogVideoProvider) submitTask(ctx context.Context, model, prompt string) (string, error) {
	body := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/videos/generations", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("CogVideo submit request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("CogVideo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var submitResp cogVideoSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return "", fmt.Errorf("failed to decode submit response: %w", err)
	}

	return submitResp.ID, nil
}

// pollResult 轮询视频生成结果
func (p *CogVideoProvider) pollResult(ctx context.Context, taskID string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
		}

		url := fmt.Sprintf("%s/async-result/%s", p.baseURL, taskID)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create poll request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			continue // 网络错误时继续轮询
		}

		var queryResp cogVideoQueryResponse
		if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		switch queryResp.TaskStatus {
		case "SUCCESS":
			if len(queryResp.VideoResult) > 0 {
				return queryResp.VideoResult[0].URL, nil
			}
			return "", fmt.Errorf("video generation succeeded but no result URL")
		case "FAIL":
			return "", fmt.Errorf("video generation failed")
		}
		// PROCESSING 状态继续轮询
	}

	return "", fmt.Errorf("video generation timed out after %v", timeout)
}

// downloadAndSave 下载远程视频并保存到 Storage
func (p *CogVideoProvider) downloadAndSave(ctx context.Context, videoURL, storagePath string) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	fileURL, err := p.storage.Upload(ctx, resp.Body, storagePath)
	if err != nil {
		return "", fmt.Errorf("failed to save video: %w", err)
	}

	return fileURL, nil
}
