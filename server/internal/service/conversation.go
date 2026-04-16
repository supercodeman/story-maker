// server/internal/service/conversation.go
package service

import (
	"context"
	"encoding/json"
	"errors"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// ConversationService 会话管理服务
type ConversationService struct {
	convDAO       *dao.ConversationDAO
	msgDAO        *dao.MessageDAO
	dispatcher    *agent.Dispatcher
	modelRegistry *ModelRegistryService
}

// NewConversationService 创建 ConversationService 实例
func NewConversationService(
	convDAO *dao.ConversationDAO,
	msgDAO *dao.MessageDAO,
	dispatcher *agent.Dispatcher,
) *ConversationService {
	return &ConversationService{
		convDAO:    convDAO,
		msgDAO:     msgDAO,
		dispatcher: dispatcher,
	}
}

// SetModelRegistry 注入模型注册服务（延迟注入，避免循环依赖）
func (s *ConversationService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// CreateConversation 创建新会话
func (s *ConversationService) CreateConversation(ctx context.Context, userID, portfolioID uint, modelName, title string) (*model.Conversation, error) {
	if s.modelRegistry != nil {
		modelName = s.modelRegistry.ResolveModel(modelName, model.CapTextGen)
	} else if modelName == "" {
		modelName = model.ProviderZhipu
	}
	if title == "" {
		title = "New Conversation"
	}

	conv := &model.Conversation{
		UserID:      userID,
		PortfolioID: portfolioID,
		ModelName:   modelName,
		Title:       title,
		Status:      model.ConversationStatusActive,
	}

	if err := s.convDAO.Create(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// SendMessage 在会话中发送消息并触发 AI 回复
// 返回创建的 AITask ID，前端通过 WebSocket 监听任务完成
func (s *ConversationService) SendMessage(ctx context.Context, convID, userID uint, content string) (uint, error) {
	// 1. 校验会话归属
	conv, err := s.convDAO.GetByID(ctx, convID)
	if err != nil {
		return 0, err
	}
	if conv.UserID != userID {
		return 0, errors.New("permission denied")
	}
	if conv.Status != model.ConversationStatusActive {
		return 0, errors.New("conversation is archived")
	}

	// 2. 保存用户消息
	userMsg := &model.Message{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
		TokenCount:     estimateTokens(content),
	}
	if err := s.msgDAO.Create(ctx, userMsg); err != nil {
		return 0, err
	}

	// 3. 构建对话历史（取最近 20 轮消息）
	recentMsgs, err := s.msgDAO.GetRecentMessages(ctx, convID, 40)
	if err != nil {
		return 0, err
	}

	history := make([]agent.ChatMessage, 0, len(recentMsgs))
	for _, msg := range recentMsgs {
		// 排除刚插入的用户消息（会作为 Prompt 传入）
		if msg.ID == userMsg.ID {
			continue
		}
		history = append(history, agent.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 4. 序列化 history
	var historyJSON string
	if len(history) > 0 {
		hBytes, _ := json.Marshal(history)
		historyJSON = string(hBytes)
	}

	// 5. 创建 AITask 并分发
	task := &model.AITask{
		UserID:      userID,
		PortfolioID: conv.PortfolioID,
		TaskType:    model.TaskTypeTextGen,
		ModelName:   conv.ModelName,
		Prompt:      content,
		History:     historyJSON,
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	// 6. 更新会话元信息
	conv.MessageCount += 1
	if conv.Title == "New Conversation" && len(content) > 0 {
		// 用第一条消息的前 50 字符作为标题
		title := []rune(content)
		if len(title) > 50 {
			title = title[:50]
		}
		conv.Title = string(title)
	}
	_ = s.convDAO.Update(ctx, conv)

	return task.ID, nil
}

// SaveAssistantMessage 保存 AI 回复消息（任务完成后由回调触发）
func (s *ConversationService) SaveAssistantMessage(ctx context.Context, convID uint, content string, taskID uint) error {
	msg := &model.Message{
		ConversationID: convID,
		Role:           "assistant",
		Content:        content,
		TokenCount:     estimateTokens(content),
		TaskID:         &taskID,
	}
	if err := s.msgDAO.Create(ctx, msg); err != nil {
		return err
	}

	// 更新会话消息计数
	conv, err := s.convDAO.GetByID(ctx, convID)
	if err != nil {
		return err
	}
	conv.MessageCount += 1
	return s.convDAO.Update(ctx, conv)
}

// GetConversation 获取会话详情（含最近消息）
func (s *ConversationService) GetConversation(ctx context.Context, convID, userID uint) (*model.Conversation, []*model.Message, error) {
	conv, err := s.convDAO.GetByID(ctx, convID)
	if err != nil {
		return nil, nil, err
	}
	if conv.UserID != userID {
		return nil, nil, errors.New("permission denied")
	}

	msgs, err := s.msgDAO.ListByConversation(ctx, convID, 100, 0)
	if err != nil {
		return nil, nil, err
	}

	return conv, msgs, nil
}

// ListConversations 获取会话列表
func (s *ConversationService) ListConversations(ctx context.Context, userID uint, portfolioID *uint, page, pageSize int) ([]*model.Conversation, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.convDAO.ListByUser(ctx, userID, portfolioID, pageSize, offset)
}

// ArchiveConversation 归档会话
func (s *ConversationService) ArchiveConversation(ctx context.Context, convID, userID uint) error {
	conv, err := s.convDAO.GetByID(ctx, convID)
	if err != nil {
		return err
	}
	if conv.UserID != userID {
		return errors.New("permission denied")
	}
	return s.convDAO.Archive(ctx, convID)
}

// estimateTokens 粗略估算 token 数（委托给 agent 包的精确实现）
func estimateTokens(text string) int {
	return agent.EstimateTokens(text)
}
