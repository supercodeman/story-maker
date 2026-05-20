// server/internal/service/ai.go
package service

import (
	"context"
	"encoding/json"
	"errors"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// ChatMessage 对话历史消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIService AI 服务层
type AIService struct {
	taskDAO    *dao.AITaskDAO
	dispatcher *agent.Dispatcher
}

// NewAIService 创建 AIService 实例
func NewAIService(taskDAO *dao.AITaskDAO, dispatcher *agent.Dispatcher) *AIService {
	return &AIService{
		taskDAO:    taskDAO,
		dispatcher: dispatcher,
	}
}

// SubmitTextTask 提交文本生成任务
func (s *AIService) SubmitTextTask(ctx context.Context, userID, portfolioID uint, modelName, prompt string, history []ChatMessage) (uint, error) {
	// 校验输入
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}
	if modelName == "" {
		modelName = model.ProviderZhipu
	}

	// 将 history 序列化存入 prompt 的扩展字段（用 JSON 编码追加到任务中）
	var historyJSON string
	if len(history) > 0 {
		hBytes, _ := json.Marshal(history)
		historyJSON = string(hBytes)
	}

	// 创建任务
	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeTextGen,
		ModelName:   modelName,
		Prompt:      prompt,
		History:     historyJSON,
	}

	// 分发任务
	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// SubmitImageTask 提交图像生成任务（旧接口，保留兼容）
func (s *AIService) SubmitImageTask(ctx context.Context, userID, portfolioID uint, modelName, prompt string) (uint, error) {
	return s.SubmitImageGenTask(ctx, userID, portfolioID, 0, prompt, "", 1)
}

// SubmitImageGenTask 提交文生图任务（新接口，支持章节绑定和参数）
func (s *AIService) SubmitImageGenTask(ctx context.Context, userID, portfolioID, chapterID uint, prompt, aspectRatio string, n int) (uint, error) {
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}
	if n <= 0 {
		n = 1
	}

	// 将参数序列化为 JSON 作为 prompt 传入
	promptJSON, _ := json.Marshal(map[string]interface{}{
		"prompt":       prompt,
		"aspect_ratio": aspectRatio,
		"n":            n,
	})

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeImageGen,
		ModelName:   "qwen_image",
		Prompt:      string(promptJSON),
	}
	if chapterID > 0 {
		task.ChapterID = &chapterID
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// SubmitCharacterAdjustTask 提交角色调整任务
func (s *AIService) SubmitCharacterAdjustTask(ctx context.Context, userID, portfolioID uint, modelName, prompt string) (uint, error) {
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}
	if modelName == "" {
		modelName = model.ProviderZhipu
	}

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeCharacterAdjust,
		ModelName:   modelName,
		Prompt:      prompt,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// GetTask 获取任务详情
func (s *AIService) GetTask(ctx context.Context, taskID, userID uint) (*model.AITask, error) {
	task, err := s.taskDAO.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 权限校验：只能查看自己的任务
	if task.UserID != userID {
		return nil, errors.New("permission denied")
	}

	return task, nil
}

// ListTasks 获取任务列表
func (s *AIService) ListTasks(ctx context.Context, userID uint, portfolioID *uint, page, pageSize int, taskTypes []string, butlerSessionID string, novelID uint) ([]*model.AITask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	if portfolioID != nil {
		return s.taskDAO.ListTasksByPortfolio(ctx, *portfolioID, pageSize, offset, taskTypes, butlerSessionID, novelID)
	}

	return s.taskDAO.ListTasksByUser(ctx, userID, pageSize, offset)
}

// CancelTask 取消任务
func (s *AIService) CancelTask(ctx context.Context, taskID, userID uint) error {
	task, err := s.taskDAO.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// 权限校验
	if task.UserID != userID {
		return errors.New("permission denied")
	}

	return s.dispatcher.CancelTask(ctx, taskID)
}

// SubmitAudioTask 提交音频生成任务
// chapterID 用于完成后把生成的音频绑定到 assets 表对应章节
func (s *AIService) SubmitAudioTask(ctx context.Context, userID, portfolioID, chapterID uint, prompt string) (uint, error) {
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeAudioGen,
		ModelName:   "minimax_tts",
		Prompt:      prompt,
	}
	if chapterID > 0 {
		task.ChapterID = &chapterID
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}

// SubmitVideoTask 提交视频生成任务
func (s *AIService) SubmitVideoTask(ctx context.Context, userID, portfolioID uint, modelName, prompt string) (uint, error) {
	if prompt == "" {
		return 0, errors.New("prompt cannot be empty")
	}
	if modelName == "" {
		modelName = "cogvideo"
	}

	task := &model.AITask{
		UserID:      userID,
		PortfolioID: portfolioID,
		TaskType:    model.TaskTypeVideoGen,
		ModelName:   modelName,
		Prompt:      prompt,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	return task.ID, nil
}
