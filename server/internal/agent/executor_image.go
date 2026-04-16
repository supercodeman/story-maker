// server/internal/agent/executor_image.go
package agent

import "context"

// ImageTaskExecutor 图像任务执行器（image_gen / image_edit）
type ImageTaskExecutor struct{}

func (e *ImageTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	req := &ImageRequest{
		Model:  ec.ModelVersion,
		Prompt: ec.Task.Prompt,
		Width:  1024,
		Height: 1024,
	}

	resp, err := ec.Provider.GenerateImage(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image_url": resp.ImageURL,
		"file_path": resp.FilePath,
	}, nil
}

// CharacterTaskExecutor 角色调整任务执行器（character_adjust）
type CharacterTaskExecutor struct{}

func (e *CharacterTaskExecutor) Execute(ctx context.Context, ec *ExecContext) (interface{}, error) {
	req := &CharacterAdjustRequest{
		Prompt: ec.Task.Prompt,
	}

	resp, err := ec.Provider.AdjustCharacter(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"image_url": resp.ImageURL,
		"file_path": resp.FilePath,
	}, nil
}
