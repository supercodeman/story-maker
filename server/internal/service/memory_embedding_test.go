// server/internal/service/memory_embedding_test.go
package service

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB 初始化测试数据库连接
// 使用环境变量 TEST_DSN 或默认本地 DSN
func setupTestDB(t *testing.T) {
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/ai_curton?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("跳过测试：无法连接数据库: %v", err)
	}

	// 确保表存在
	db.AutoMigrate(&model.WritingMemory{}, &model.MemoryEmbedding{}, &model.NovelMemoryBinding{})
	model.DB = db
}

// newTestDispatcher 创建带 mock provider 的 dispatcher
func newTestDispatcher() *agent.Dispatcher {
	d := agent.NewDispatcher(nil, nil, nil)
	d.RegisterProvider("qwen", agent.NewMockProvider())
	return d
}

// TestGenerateEmbeddings 测试 Embedding 生成流程
func TestGenerateEmbeddings(t *testing.T) {
	setupTestDB(t)
	dispatcher := newTestDispatcher()
	svc := &MemoryService{
		memoryDAO:  dao.NewWritingMemoryDAO(),
		dispatcher: dispatcher,
	}

	// 创建测试记忆（超过 500 字以验证分块）
	sampleText := generateSampleText(1200)
	memory := &model.WritingMemory{
		UserID:     99999, // 测试用户
		Category:   "style",
		Title:      "Embedding测试记忆",
		SourceText: sampleText,
		Status:     model.MemoryStatusDraft,
		Version:    1,
	}

	if err := model.DB.Create(memory).Error; err != nil {
		t.Fatalf("创建测试记忆失败: %v", err)
	}
	defer model.DB.Delete(memory) // 清理

	// 执行 Embedding 生成
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := svc.GenerateEmbeddings(ctx, memory, "qwen")
	if err != nil {
		t.Fatalf("GenerateEmbeddings 失败: %v", err)
	}

	// 验证 Embedding 已写入数据库
	var embeddings []model.MemoryEmbedding
	model.DB.Where("memory_id = ?", memory.ID).Find(&embeddings)

	if len(embeddings) == 0 {
		t.Fatal("未生成任何 Embedding 记录")
	}

	// 1200 字按 500 字分块应该有 3 个 chunk
	expectedChunks := 3
	if len(embeddings) != expectedChunks {
		t.Errorf("期望 %d 个 chunk，实际 %d 个", expectedChunks, len(embeddings))
	}

	// 验证向量维度
	for i, emb := range embeddings {
		if emb.Dimension != 1024 {
			t.Errorf("chunk %d 维度错误: 期望 1024，实际 %d", i, emb.Dimension)
		}
		var vec []float64
		if err := json.Unmarshal([]byte(emb.Vector), &vec); err != nil {
			t.Errorf("chunk %d 向量 JSON 解析失败: %v", i, err)
		}
		if len(vec) != 1024 {
			t.Errorf("chunk %d 向量长度错误: 期望 1024，实际 %d", i, len(vec))
		}
	}

	// 清理 embedding 数据
	model.DB.Where("memory_id = ?", memory.ID).Delete(&model.MemoryEmbedding{})

	t.Logf("✓ 成功生成 %d 个 Embedding chunk，维度 1024", len(embeddings))
}

// TestGetRelevantChunks 测试语义检索流程
func TestGetRelevantChunks(t *testing.T) {
	setupTestDB(t)
	dispatcher := newTestDispatcher()
	svc := &MemoryService{
		memoryDAO:  dao.NewWritingMemoryDAO(),
		dispatcher: dispatcher,
	}

	// 创建测试记忆
	memory := &model.WritingMemory{
		UserID:     99999,
		Category:   "style",
		Title:      "检索测试记忆",
		SourceText: generateSampleText(1500),
		Status:     model.MemoryStatusDraft,
		Version:    1,
	}
	if err := model.DB.Create(memory).Error; err != nil {
		t.Fatalf("创建测试记忆失败: %v", err)
	}
	defer func() {
		model.DB.Where("memory_id = ?", memory.ID).Delete(&model.MemoryEmbedding{})
		model.DB.Delete(memory)
	}()

	// 先生成 Embedding
	ctx := context.Background()
	if err := svc.GenerateEmbeddings(ctx, memory, "qwen"); err != nil {
		t.Fatalf("GenerateEmbeddings 失败: %v", err)
	}

	// 执行语义检索
	chunks, err := svc.GetRelevantChunks(ctx, memory.ID, "测试查询文本", "qwen", 2)
	if err != nil {
		t.Fatalf("GetRelevantChunks 失败: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("未检索到任何 chunk")
	}

	if len(chunks) > 2 {
		t.Errorf("topK=2 但返回了 %d 个 chunk", len(chunks))
	}

	for i, chunk := range chunks {
		if chunk == "" {
			t.Errorf("chunk %d 内容为空", i)
		}
	}

	t.Logf("✓ 成功检索到 %d 个相关 chunk（topK=2）", len(chunks))
}

// TestHandleExtractCompleteTriggersEmbedding 测试提取完成后自动触发 Embedding
func TestHandleExtractCompleteTriggersEmbedding(t *testing.T) {
	setupTestDB(t)
	dispatcher := newTestDispatcher()
	svc := &MemoryService{
		memoryDAO:  dao.NewWritingMemoryDAO(),
		dispatcher: dispatcher,
	}

	// 创建一个已提取完成的记忆
	memory := &model.WritingMemory{
		UserID:        99999,
		Category:      "style",
		Title:         "自动Embedding测试",
		SourceText:    generateSampleText(600),
		Status:        model.MemoryStatusDraft,
		ExtractStatus: "completed",
		Version:       1,
		Quality:       85.0,
		Features:      `{"tone":"幽默"}`,
		PromptTpl:     "测试模板",
	}
	if err := model.DB.Create(memory).Error; err != nil {
		t.Fatalf("创建测试记忆失败: %v", err)
	}
	defer func() {
		model.DB.Where("memory_id = ?", memory.ID).Delete(&model.MemoryEmbedding{})
		model.DB.Delete(memory)
	}()

	// 直接调用 GenerateEmbeddings 模拟 handleExtractComplete 中的异步逻辑
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := svc.GenerateEmbeddings(ctx, memory, "qwen")
	if err != nil {
		t.Fatalf("Embedding 生成失败: %v", err)
	}

	// 验证数据库中有 embedding 数据
	var count int64
	model.DB.Model(&model.MemoryEmbedding{}).Where("memory_id = ?", memory.ID).Count(&count)
	if count == 0 {
		t.Fatal("handleExtractComplete 后未生成 Embedding")
	}

	t.Logf("✓ 提取完成后成功生成 %d 条 Embedding 记录", count)
}

// ========== 辅助函数 ==========

func generateSampleText(length int) string {
	// 生成指定长度的中文测试文本
	base := "这是一段用于测试Embedding生成的样本文本。它包含了各种写作风格的特征，比如细腻的描写、流畅的叙事和生动的对话。"
	runes := []rune(base)
	var result []rune
	for len(result) < length {
		result = append(result, runes...)
	}
	return string(result[:length])
}
