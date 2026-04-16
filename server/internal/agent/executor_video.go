// server/internal/agent/executor_video.go
package agent

import (
	"context"
	"fmt"
)

// VideoTaskExecutor 视频生成任务执行器（video_gen）
type VideoTaskExecutor struct {
	videoProvider VideoProvider
}

// NewVideoTaskExecutor 创建视频任务执行器
func NewVideoTaskExecutor(vp VideoProvider) *VideoTaskExecutor {
	return &VideoTaskExecutor{videoProvider: vp}
}

// Execute 执行视频生成任务
func (e *VideoTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	if e.videoProvider == nil {
		return nil, fmt.Errorf("video provider not configured")
	}

	resp, err := e.videoProvider.GenerateVideo(ctx, &VideoGenRequest{
		Prompt: ec.Task.Prompt,
		Model:  ec.ModelVersion,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"video_url": resp.VideoURL,
		"file_path": resp.FilePath,
		"duration":  resp.Duration,
	}, nil
}
