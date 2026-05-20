# 分层记忆系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 story-maker 实现分层记忆能力——P1 会话管理、P2 Redis 缓存 + 历史压缩、P3 长期记忆提取。

**Architecture:** 新增 Conversation/Message/MemoryEntry 三个模型，通过 ConversationService 统一管理对话生命周期。MemoryService 负责 Redis 缓存加速、历史压缩摘要、长期记忆提取。发消息时由 MemoryService.BuildContext() 组装完整上下文（长期记忆 + 摘要 + 最近消息 + 当前 Prompt）再交给 Dispatcher 执行。

**Tech Stack:** Go 1.22, Gin, GORM, Redis (go-redis/v9), MySQL, Vue 3 + TypeScript + Pinia

---

## File Structure

### 后端新增文件

| 文件 | 职责 |
|------|------|
| `server/internal/model/conversation.go` | Conversation + Message + MemoryEntry 模型定义 |
| `server/internal/dao/conversation.go` | ConversationDAO — 会话 CRUD |
| `server/internal/dao/message.go` | MessageDAO — 消息 CRUD + 批量操作 |
| `server/internal/dao/memory.go` | MemoryEntryDAO — 长期记忆 CRUD + Upsert |
| `server/internal/service/conversation.go` | ConversationService — 会话管理 + 发消息 |
| `server/internal/service/memory.go` | MemoryService — Redis 缓存 + 压缩 + 提取 |
| `server/internal/handler/conversation.go` | ConversationHandler — HTTP 接口 |
| `server/internal/handler/memory.go` | MemoryHandler — 记忆管理接口 |

### 后端修改文件

| 文件 | 改动 |
|------|------|
| `server/internal/model/base.go:54-63` | autoMigrate 添加新模型 |
| `server/internal/router/router.go` | 注册 conversation + memory 路由，注入依赖 |
| `server/internal/agent/dispatcher.go` | 新增 ExecuteTextDirect 方法供摘要/提取调用 |

### 前端新增文件

| 文件 | 职责 |
|------|------|
| `web/src/api/conversation.ts` | 会话 + 记忆 API 接口 |
| `web/src/store/conversation.ts` | 会话状态管理 Store |

### 前端修改文件

| 文件 | 改动 |
|------|------|
| `web/src/views/studio/AIStudio.vue` | 从本地 messages 改为基于 Conversation API |

---

## P1: 会话管理 — 服务端持久化对话历史

### Task 1: 数据模型 — Conversation + Message

**Files:**
- Create: `server/internal/model/conversation.go`
- Modify: `server/internal/model/base.go:54-63`

- [ ] **Step 1: 创建 Conversation + Message 模型**

```go
// server/internal/model/conversation.go
package model

import "time"

// Conversation 会话表，管理一次连贯的对话生命周期
type Conversation struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	PortfolioID  uint      `gorm:"index" json:"portfolio_id"`
	Title        string    `gorm:"size:200" json:"title"`
	ModelName    string    `gorm:"size:50" json:"model_name"`
	Summary      string    `gorm:"type:text" json:"summary"`
	MessageCount int       `json:"message_count"`
	Status       string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 会话状态常量
const (
	ConversationStatusActive   = "active"
	ConversationStatusArchived = "archived"
)

// Message 消息表，记录每一轮对话
type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"index" json:"conversation_id"`
	Role           string    `gorm:"size:20" json:"role"`
	Content        string    `gorm:"type:text" json:"content"`
	TokenCount     int       `json:"token_count"`
	TaskID         *uint     `json:"task_id"`
	CreatedAt      time.Time `json:"created_at"`
}
```

- [ ] **Step 2: 注册自动迁移**

修改 `server/internal/model/base.go` 的 `autoMigrate` 函数：

```go
func autoMigrate() error {
	return DB.AutoMigrate(
		&User{},
		&Workspace{},
		&WorkspaceMember{},
		&Portfolio{},
		&Character{},
		&Asset{},
		&AITask{},
		&Conversation{},
		&Message{},
	)
}
```

- [ ] **Step 3: 验证编译通过**

Run: `cd /Users/sangchenglong/go/src/story-maker/server && go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 4: Commit**

```bash
git add server/internal/model/conversation.go server/internal/model/base.go
git commit -m "feat(model): 新增 Conversation + Message 模型，支持服务端会话管理"
```

---

### Task 2: DAO 层 — ConversationDAO + MessageDAO

**Files:**
- Create: `server/internal/dao/conversation.go`
- Create: `server/internal/dao/message.go`

- [ ] **Step 1: 创建 ConversationDAO**

```go
// server/internal/dao/conversation.go
package dao

import (
	"context"

	"story-maker/server/internal/model"
	"gorm.io/gorm"
)

// ConversationDAO 会话数据访问层
type ConversationDAO struct {
	db *gorm.DB
}

func NewConversationDAO(db *gorm.DB) *ConversationDAO {
	return &ConversationDAO{db: db}
}

func (d *ConversationDAO) Create(ctx context.Context, conv *model.Conversation) error {
	return d.db.WithContext(ctx).Create(conv).Error
}

func (d *ConversationDAO) GetByID(ctx context.Context, id uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := d.db.WithContext(ctx).First(&conv, id).Error
	return &conv, err
}

func (d *ConversationDAO) Update(ctx context.Context, conv *model.Conversation) error {
	return d.db.WithContext(ctx).Save(conv).Error
}

func (d *ConversationDAO) ListByUser(ctx context.Context, userID uint, portfolioID *uint, limit, offset int) ([]*model.Conversation, int64, error) {
	var convs []*model.Conversation
	var total int64

	query := d.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, model.ConversationStatusActive)
	if portfolioID != nil {
		query = query.Where("portfolio_id = ?", *portfolioID)
	}

	if err := query.Model(&model.Conversation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&convs).Error
	return convs, total, err
}

func (d *ConversationDAO) Archive(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", id).
		Update("status", model.ConversationStatusArchived).Error
}
```

- [ ] **Step 2: 创建 MessageDAO**

```go
// server/internal/dao/message.go
package dao

import (
	"context"

	"story-maker/server/internal/model"
	"gorm.io/gorm"
)

// MessageDAO 消息数据访问层
type MessageDAO struct {
	db *gorm.DB
}

func NewMessageDAO(db *gorm.DB) *MessageDAO {
	return &MessageDAO{db: db}
}

func (d *MessageDAO) Create(ctx context.Context, msg *model.Message) error {
	return d.db.WithContext(ctx).Create(msg).Error
}

func (d *MessageDAO) ListByConversation(ctx context.Context, convID uint, limit, offset int) ([]*model.Message, error) {
	var msgs []*model.Message
	query := d.db.WithContext(ctx).Where("conversation_id = ?", convID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	err := query.Find(&msgs).Error
	return msgs, err
}

// GetRecentMessages 获取最近 N 条消息（按时间倒序取，再正序返回）
func (d *MessageDAO) GetRecentMessages(ctx context.Context, convID uint, limit int) ([]*model.Message, error) {
	var msgs []*model.Message
	err := d.db.WithContext(ctx).Where("conversation_id = ?", convID).
		Order("created_at DESC").Limit(limit).Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	// 反转为正序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (d *MessageDAO) CountByConversation(ctx context.Context, convID uint) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Message{}).Where("conversation_id = ?", convID).Count(&count).Error
	return count, err
}

func (d *MessageDAO) DeleteBatch(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Message{}).Error
}
```

- [ ] **Step 3: 验证编译通过**

Run: `cd /Users/sangchenglong/go/src/story-maker/server && go build ./...`
Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add server/internal/dao/conversation.go server/internal/dao/message.go
git commit -m "feat(dao): 新增 ConversationDAO + MessageDAO 数据访问层"
```

---

### Task 3: Service 层 — ConversationService（核心）

**Files:**
- Create: `server/internal/service/conversation.go`
- Modify: `server/internal/agent/dispatcher.go`

- [ ] **Step 1: 为 Dispatcher 新增同步文本执行方法**

在 `server/internal/agent/dispatcher.go` 末尾添加：

```go
// ExecuteTextDirect 同步执行文本生成（供内部摘要/提取使用，不创建 AITask）
func (d *Dispatcher) ExecuteTextDirect(ctx context.Context, modelName, apiKey, prompt string) (string, error) {
	provider, err := d.GetProvider(modelName)
	if err != nil {
		return "", err
	}

	// 动态设置 API Key
	if _, ok := provider.(*MockProvider); !ok {
		if kp, ok := provider.(*KimiProvider); ok {
			kp.SetAPIKey(apiKey)
		}
		if zp, ok := provider.(*ZhipuProvider); ok {
			zp.SetAPIKey(apiKey)
		}
	}

	req := &TextRequest{
		Prompt:    prompt,
		MaxTokens: 2048,
	}

	resp, err := provider.GenerateText(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
```

- [ ] **Step 2: 创建 ConversationService**

```go
// server/internal/service/conversation.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// ConversationService 会话服务层
type ConversationService struct {
	convDAO    *dao.ConversationDAO
	msgDAO     *dao.MessageDAO
	taskDAO    *dao.AITaskDAO
	dispatcher *agent.Dispatcher
	keyStore   agent.KeyStore
}

func NewConversationService(
	convDAO *dao.ConversationDAO,
	msgDAO *dao.MessageDAO,
	taskDAO *dao.AITaskDAO,
	dispatcher *agent.Dispatcher,
	keyStore agent.KeyStore,
) *ConversationService {
	return &ConversationService{
		convDAO:    convDAO,
		msgDAO:     msgDAO,
		taskDAO:    taskDAO,
		dispatcher: dispatcher,
		keyStore:   keyStore,
	}
}

// CreateConversation 创建新会话
func (s *ConversationService) CreateConversation(ctx context.Context, userID, portfolioID uint, modelName, title string) (*model.Conversation, error) {
	if modelName == "" {
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
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conv, nil
}

// SendMessage 发送消息并触发 AI 回复（核心方法）
func (s *ConversationService) SendMessage(ctx context.Context, userID, convID uint, content string) (uint, error) {
	// 1. 校验会话归属
	conv, err := s.convDAO.GetByID(ctx, convID)
	if err != nil {
		return 0, errors.New("conversation not found")
	}
	if conv.UserID != userID {
		return 0, errors.New("permission denied")
	}

	// 2. 保存用户消息
	userMsg := &model.Message{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
		TokenCount:     estimateTokens(content),
	}
	if err := s.msgDAO.Create(ctx, userMsg); err != nil {
		return 0, fmt.Errorf("failed to save message: %w", err)
	}

	// 3. 构建上下文：加载最近消息作为 history
	recentMsgs, _ := s.msgDAO.GetRecentMessages(ctx, convID, 20)
	var history []agent.ChatMessage
	for _, m := range recentMsgs {
		if m.ID == userMsg.ID {
			continue // 不包含刚存的这条，它会作为 Prompt
		}
		history = append(history, agent.ChatMessage{Role: m.Role, Content: m.Content})
	}

	// 4. 如果有会话摘要，注入为 system 消息
	if conv.Summary != "" {
		summaryMsg := agent.ChatMessage{
			Role:    "system",
			Content: "以下是之前对话的摘要：\n" + conv.Summary,
		}
		history = append([]agent.ChatMessage{summaryMsg}, history...)
	}

	// 5. 序列化 history，创建 AITask 并分发
	historyJSON, _ := json.Marshal(history)
	task := &model.AITask{
		UserID:      userID,
		PortfolioID: conv.PortfolioID,
		TaskType:    model.TaskTypeTextGen,
		ModelName:   conv.ModelName,
		Prompt:      content,
		History:     string(historyJSON),
	}

	if err := s.dispatcher.Dispatch(ctx, task); err != nil {
		return 0, err
	}

	// 6. 更新会话消息计数和标题
	conv.MessageCount++
	if conv.Title == "New Conversation" && len(content) > 0 {
		title := content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conv.Title = title
	}
	_ = s.convDAO.Update(ctx, conv)

	return task.ID, nil
}

// SaveAssistantMessage 保存 AI 回复消息（任务完成后由回调触发）
func (s *ConversationService) SaveAssistantMessage(ctx context.Context, convID uint, taskID uint, content string) error {
	msg := &model.Message{
		ConversationID: convID,
		Role:           "assistant",
		Content:        content,
		TokenCount:     estimateTokens(content),
		TaskID:         &taskID,
	}
	return s.msgDAO.Create(ctx, msg)
}

// GetConversation 获取会话详情
func (s *ConversationService) GetConversation(ctx context.Context, convID, userID uint) (*model.Conversation, error) {
	conv, err := s.convDAO.GetByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, errors.New("permission denied")
	}
	return conv, nil
}

// GetMessages 获取会话消息列表
func (s *ConversationService) GetMessages(ctx context.Context, convID, userID uint, limit, offset int) ([]*model.Message, error) {
	conv, err := s.convDAO.GetByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, errors.New("permission denied")
	}

	if limit <= 0 {
		limit = 50
	}
	return s.msgDAO.ListByConversation(ctx, convID, limit, offset)
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

// estimateTokens 粗略估算 token 数（中文约 1.5 字/token，英文约 4 字符/token）
func estimateTokens(text string) int {
	runeCount := len([]rune(text))
	return (runeCount*3 + 3) / 4 // 取中间值，约 0.75 token/字符
}
```

- [ ] **Step 3: 验证编译通过**

Run: `cd /Users/sangchenglong/go/src/story-maker/server && go build ./...`
Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add server/internal/service/conversation.go server/internal/agent/dispatcher.go
git commit -m "feat(service): 新增 ConversationService，支持会话管理和上下文组装"
```

---

<!-- TASK4_PLACEHOLDER -->
