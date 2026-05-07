// server/internal/service/memory.go
package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"story-maker/server/internal/agent"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// timeAfter 返回一个 channel，在 seconds 秒后发送
func timeAfter(seconds int) <-chan time.Time {
	return time.After(time.Duration(seconds) * time.Second)
}

// MemoryService 记忆业务逻辑层
type MemoryService struct {
	memoryDAO       *dao.WritingMemoryDAO
	genreDAO        *dao.GenreDAO
	dispatcher      *agent.Dispatcher
	workflowService *WorkflowService
	notifier        agent.Notifier
}

// NewMemoryService 创建 MemoryService 实例
func NewMemoryService(dispatcher *agent.Dispatcher, workflowService *WorkflowService, notifier agent.Notifier) *MemoryService {
	return &MemoryService{
		memoryDAO:       dao.NewWritingMemoryDAO(),
		genreDAO:        dao.NewGenreDAO(),
		dispatcher:      dispatcher,
		workflowService: workflowService,
		notifier:        notifier,
	}
}

// CreateMemoryRequest 创建记忆请求
type CreateMemoryRequest struct {
	Category    string `json:"category" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	SampleText  string `json:"sample_text" binding:"required"`
	Tags        string `json:"tags"`
	ModelName   string `json:"model_name"`
	GenreIDs    []uint `json:"genre_ids"`
}

// CreateMemory 创建记忆并触发提取工作流
func (s *MemoryService) CreateMemory(ctx context.Context, userID uint, req *CreateMemoryRequest) (*model.WritingMemory, error) {
	// 校验类别
	if _, ok := model.ValidMemoryCategories[req.Category]; !ok {
		return nil, errors.New("invalid memory category")
	}

	// 样本长度校验
	sampleLen := utf8.RuneCountInString(req.SampleText)
	if sampleLen < 200 {
		return nil, errors.New("sample text too short, minimum 200 characters")
	}

	// 样本去重
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.SampleText)))
	if existing, err := s.memoryDAO.GetBySampleHash(hash); err == nil && existing != nil {
		return nil, fmt.Errorf("duplicate sample text, existing memory id: %d", existing.ID)
	}

	memory := &model.WritingMemory{
		UserID:      userID,
		Category:    req.Category,
		Title:       req.Title,
		Description: req.Description,
		SourceText:  req.SampleText,
		SampleHash:  hash,
		SampleLen:   sampleLen,
		Tags:        req.Tags,
		Status:      model.MemoryStatusDraft,
		Version:     1,
	}

	if err := s.memoryDAO.Create(memory); err != nil {
		return nil, fmt.Errorf("create memory failed: %w", err)
	}

	// 写入赛道关联
	if len(req.GenreIDs) > 0 {
		if err := s.genreDAO.SetMemoryGenres(memory.ID, req.GenreIDs); err != nil {
			log.Printf("[memory] set genre associations failed for memory %d: %v", memory.ID, err)
		}
	}

	// 异步触发提取工作流
	go s.triggerExtractWorkflow(memory, req.ModelName)

	return memory, nil
}

// triggerExtractWorkflow 触发记忆提取工作流
func (s *MemoryService) triggerExtractWorkflow(memory *model.WritingMemory, modelName string) {
	if s.workflowService == nil {
		log.Printf("[memory] workflowService is nil, skip extract for memory %d", memory.ID)
		return
	}
	if modelName == "" {
		modelName = "qwen"
	}

	log.Printf("[memory] triggering extract workflow for memory %d, model=%s, category=%s", memory.ID, modelName, memory.Category)

	ctx := context.Background()
	wfID, err := s.workflowService.SubmitWorkflow(ctx, memory.UserID, &SubmitWorkflowRequest{
		WorkflowType: model.WorkflowTypeMemoryExtract,
		ModelName:    modelName,
		Params: map[string]interface{}{
			"sample_text": memory.SourceText,
			"category":    memory.Category,
			"memory_id":   memory.ID,
		},
	})
	if err != nil {
		log.Printf("[memory] trigger extract workflow failed for memory %d: %v", memory.ID, err)
		memory.ExtractStatus = "failed"
		memory.ExtractError = err.Error()
		s.memoryDAO.UpdateExtractStatus(memory.ID, "failed", err.Error(), 0)
		s.notifyMemoryUpdate(memory)
		return
	}

	log.Printf("[memory] extract workflow submitted: memory=%d, workflow=%d", memory.ID, wfID)

	memory.ExtractWorkflowID = wfID
	memory.ExtractStatus = "running"
	s.memoryDAO.UpdateExtractStatus(memory.ID, "running", "", wfID)
	s.notifyMemoryUpdate(memory)

	// 监听工作流完成
	go s.watchExtractWorkflow(memory, wfID)
}

// watchExtractWorkflow 监听提取工作流完成并更新记忆
func (s *MemoryService) watchExtractWorkflow(memory *model.WritingMemory, workflowID uint) {
	ctx := context.Background()
	log.Printf("[memory] watching extract workflow %d for memory %d", workflowID, memory.ID)

	// 轮询工作流状态（简单实现，后续可改为回调）
	for i := 0; i < 300; i++ { // 最多等 5 分钟
		select {
		case <-ctx.Done():
			return
		default:
		}

		wf, nodes, err := s.workflowService.GetWorkflow(ctx, workflowID, memory.UserID)
		if err != nil {
			log.Printf("[memory] watch workflow %d error: %v", workflowID, err)
			return
		}

		if wf.Status == "completed" {
			log.Printf("[memory] extract workflow %d completed, processing results", workflowID)
			s.handleExtractComplete(memory, wf, nodes)
			return
		}
		if wf.Status == "failed" || wf.Status == "cancelled" {
			log.Printf("[memory] extract workflow %d %s: %s", workflowID, wf.Status, wf.ErrorMsg)
			memory.ExtractStatus = "failed"
			memory.ExtractError = wf.ErrorMsg
			s.memoryDAO.UpdateExtractStatus(memory.ID, "failed", wf.ErrorMsg, workflowID)
			s.notifyMemoryUpdate(memory)
			return
		}

		// 等 1 秒再查
		<-timeAfter(1)
	}
	log.Printf("[memory] watch extract workflow %d timed out for memory %d", workflowID, memory.ID)
}

// handleExtractComplete 处理提取完成
func (s *MemoryService) handleExtractComplete(memory *model.WritingMemory, wf *model.AIWorkflow, nodes []model.AIWorkflowNode) {
	// 从工作流结果中提取各节点输出
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(wf.ResultJSON), &resultMap); err != nil {
		log.Printf("[memory] parse workflow result failed: %v", err)
		return
	}

	// 提取 features
	if features, ok := resultMap["features"]; ok {
		if featStr, ok := features.(string); ok {
			memory.Features = featStr
		} else {
			featJSON, _ := json.Marshal(features)
			memory.Features = string(featJSON)
		}
	}

	// 提取 compiled（包含 prompt_template、anchor_texts 和可选的 dimension_prompts）
	if compiled, ok := resultMap["compiled"]; ok {
		compiledStr := ""
		switch v := compiled.(type) {
		case string:
			compiledStr = v
		default:
			b, _ := json.Marshal(v)
			compiledStr = string(b)
		}

		var compiledData struct {
			PromptTemplate   string            `json:"prompt_template"`
			AnchorTexts      []string          `json:"anchor_texts"`
			DimensionPrompts map[string]string `json:"dimension_prompts"`
		}
		if json.Unmarshal([]byte(compiledStr), &compiledData) == nil {
			memory.PromptTpl = compiledData.PromptTemplate
			anchorsJSON, _ := json.Marshal(compiledData.AnchorTexts)
			memory.AnchorTexts = string(anchorsJSON)

			// 如果有 dimension_prompts，回写到 features 的 prompt_part 字段
			if len(compiledData.DimensionPrompts) > 0 && memory.Features != "" {
				var styleFeatures model.StyleFeatures
				if json.Unmarshal([]byte(memory.Features), &styleFeatures) == nil && styleFeatures.Tone.Description != "" {
					if v, ok := compiledData.DimensionPrompts["tone"]; ok {
						styleFeatures.Tone.PromptPart = v
					}
					if v, ok := compiledData.DimensionPrompts["rhythm"]; ok {
						styleFeatures.Rhythm.PromptPart = v
					}
					if v, ok := compiledData.DimensionPrompts["vocabulary"]; ok {
						styleFeatures.Vocabulary.PromptPart = v
					}
					if v, ok := compiledData.DimensionPrompts["dialogue_style"]; ok {
						styleFeatures.DialogueStyle.PromptPart = v
					}
					updatedFeatures, _ := json.Marshal(styleFeatures)
					memory.Features = string(updatedFeatures)
				}
			}
		} else {
			memory.PromptTpl = compiledStr
		}
	}

	// 提取 quality（包含多维评分或旧格式单一评分）
	if quality, ok := resultMap["quality"]; ok {
		qualityStr := ""
		switch v := quality.(type) {
		case string:
			qualityStr = v
		default:
			b, _ := json.Marshal(v)
			qualityStr = string(b)
		}

		// 尝试解析为多维评分格式
		var qualityDetail model.QualityDetail
		if json.Unmarshal([]byte(qualityStr), &qualityDetail) == nil && qualityDetail.Consistency > 0 {
			// 新格式：多维评分
			memory.PreviewText = qualityDetail.PreviewText
			avg, grade := model.CalcQualityGrade(&qualityDetail)
			memory.Quality = avg
			memory.QualityGrade = grade
			detailJSON, _ := json.Marshal(qualityDetail)
			memory.QualityDetail = string(detailJSON)
		} else {
			// 旧格式兼容：单一 quality_score
			var legacyData struct {
				PreviewText  string  `json:"preview_text"`
				QualityScore float64 `json:"quality_score"`
			}
			if json.Unmarshal([]byte(qualityStr), &legacyData) == nil {
				memory.PreviewText = legacyData.PreviewText
				memory.Quality = legacyData.QualityScore
			}
		}
	}

	memory.ExtractStatus = "completed"
	memory.ExtractError = ""

	// 保存到数据库
	if err := s.memoryDAO.UpdateAfterExtract(memory); err != nil {
		log.Printf("[memory] update after extract failed: %v", err)
		return
	}

	// 创建版本快照
	s.memoryDAO.CreateVersion(&model.WritingMemoryVersion{
		MemoryID:  memory.ID,
		Version:   memory.Version,
		Features:  memory.Features,
		PromptTpl: memory.PromptTpl,
		ChangeLog: "初始提取",
	})

	s.notifyMemoryUpdate(memory)
	log.Printf("[memory] extract completed for memory %d, quality=%.1f", memory.ID, memory.Quality)

	// 异步生成 Embedding（不阻塞主流程）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := s.GenerateEmbeddings(ctx, memory, "qwen"); err != nil {
			log.Printf("[memory] generate embeddings failed for memory %d: %v", memory.ID, err)
		} else {
			log.Printf("[memory] embeddings generated for memory %d", memory.ID)
		}
	}()
}

// notifyMemoryUpdate 通过 WebSocket 推送记忆状态更新
func (s *MemoryService) notifyMemoryUpdate(memory *model.WritingMemory) {
	if s.notifier == nil {
		return
	}
	data := map[string]interface{}{
		"id":                  memory.ID,
		"status":              memory.Status,
		"extract_status":      memory.ExtractStatus,
		"extract_workflow_id": memory.ExtractWorkflowID,
		"extract_error":       memory.ExtractError,
		"features":            memory.Features,
		"prompt_tpl":          memory.PromptTpl,
		"anchor_texts":        memory.AnchorTexts,
		"preview_text":        memory.PreviewText,
		"quality":             memory.Quality,
		"quality_detail":      memory.QualityDetail,
		"quality_grade":       memory.QualityGrade,
		"review_workflow_id":  memory.ReviewWorkflowID,
		"review_reason":       memory.ReviewReason,
		"is_public":           memory.IsPublic,
	}
	_ = s.notifier.NotifyUserWithType(memory.UserID, "memory_update", data)
}

// GetMemory 获取记忆详情
func (s *MemoryService) GetMemory(id uint) (*model.WritingMemory, error) {
	return s.memoryDAO.Get(id)
}

// ListMyMemories 获取用户的记忆列表
func (s *MemoryService) ListMyMemories(userID uint, category string) ([]model.WritingMemory, error) {
	return s.memoryDAO.ListByUser(userID, category)
}

// UpdateMemory 更新记忆信息
func (s *MemoryService) UpdateMemory(memory *model.WritingMemory, title, description, tags string) error {
	if title != "" {
		memory.Title = title
	}
	if description != "" {
		memory.Description = description
	}
	if tags != "" {
		memory.Tags = tags
	}
	return s.memoryDAO.Update(memory)
}

// DeleteMemory 删除记忆
func (s *MemoryService) DeleteMemory(id uint) error {
	return s.memoryDAO.Delete(id)
}

// UpdateExtractResult 更新提取结果（由工作流回调调用）
func (s *MemoryService) UpdateExtractResult(memoryID uint, features, promptTpl, anchorTexts, previewText string, quality float64) error {
	memory, err := s.memoryDAO.Get(memoryID)
	if err != nil {
		return err
	}

	memory.Features = features
	memory.PromptTpl = promptTpl
	memory.AnchorTexts = anchorTexts
	memory.PreviewText = previewText
	memory.Quality = quality

	// 保存版本
	version := &model.WritingMemoryVersion{
		MemoryID:  memoryID,
		Version:   memory.Version,
		Features:  features,
		PromptTpl: promptTpl,
		ChangeLog: "自动提取",
	}
	if err := s.memoryDAO.CreateVersion(version); err != nil {
		log.Printf("[Memory] create version failed: %v", err)
	}

	return s.memoryDAO.Update(memory)
}

// RefineMemory 追加样本重新提取
func (s *MemoryService) RefineMemory(memory *model.WritingMemory, additionalText string) error {
	memory.SourceText += "\n\n---\n\n" + additionalText
	memory.SampleLen = utf8.RuneCountInString(memory.SourceText)
	memory.SampleHash = fmt.Sprintf("%x", sha256.Sum256([]byte(memory.SourceText)))
	memory.Version++
	return s.memoryDAO.Update(memory)
}

// PublishMemory 申请上架，触发 AI 审核工作流
func (s *MemoryService) PublishMemory(memory *model.WritingMemory, price int) error {
	if memory.Features == "" || memory.PromptTpl == "" {
		return errors.New("memory not extracted yet")
	}
	// 评级门槛：quality_grade 非空时，必须 B 级及以上才能上架
	if memory.QualityGrade != "" && memory.QualityGrade > "B" {
		return fmt.Errorf("quality grade %s is below B, cannot publish", memory.QualityGrade)
	}
	memory.Price = price
	memory.Status = model.MemoryStatusReviewing
	if err := s.memoryDAO.Update(memory); err != nil {
		return err
	}

	// 异步触发审核工作流
	go s.triggerReviewWorkflow(memory)

	return nil
}

// triggerReviewWorkflow 触发记忆审核工作流
func (s *MemoryService) triggerReviewWorkflow(memory *model.WritingMemory) {
	if s.workflowService == nil {
		log.Printf("[memory] workflowService is nil, skip review for memory %d", memory.ID)
		return
	}

	log.Printf("[memory] triggering review workflow for memory %d", memory.ID)

	ctx := context.Background()
	wfID, err := s.workflowService.SubmitWorkflow(ctx, memory.UserID, &SubmitWorkflowRequest{
		WorkflowType: model.WorkflowTypeMemoryReview,
		ModelName:    "qwen",
		Params: map[string]interface{}{
			"memory_id":    memory.ID,
			"features":     memory.Features,
			"prompt_tpl":   memory.PromptTpl,
			"anchor_texts": memory.AnchorTexts,
			"preview_text": memory.PreviewText,
		},
	})
	if err != nil {
		log.Printf("[memory] trigger review workflow failed for memory %d: %v", memory.ID, err)
		s.memoryDAO.UpdateReviewResult(memory.ID, model.MemoryStatusRejected, "审核工作流启动失败: "+err.Error(), 0)
		memory.Status = model.MemoryStatusRejected
		memory.ReviewReason = "审核工作流启动失败: " + err.Error()
		s.notifyMemoryUpdate(memory)
		return
	}

	log.Printf("[memory] review workflow submitted: memory=%d, workflow=%d", memory.ID, wfID)

	memory.ReviewWorkflowID = wfID
	s.memoryDAO.UpdateReviewResult(memory.ID, model.MemoryStatusReviewing, "", wfID)
	s.notifyMemoryUpdate(memory)

	// 监听审核工作流完成
	go s.watchReviewWorkflow(memory, wfID)
}

// watchReviewWorkflow 监听审核工作流完成并更新记忆状态
func (s *MemoryService) watchReviewWorkflow(memory *model.WritingMemory, workflowID uint) {
	ctx := context.Background()
	log.Printf("[memory] watching review workflow %d for memory %d", workflowID, memory.ID)

	for i := 0; i < 300; i++ { // 最多等 5 分钟
		select {
		case <-ctx.Done():
			return
		default:
		}

		wf, _, err := s.workflowService.GetWorkflow(ctx, workflowID, memory.UserID)
		if err != nil {
			log.Printf("[memory] watch review workflow %d error: %v", workflowID, err)
			return
		}

		if wf.Status == "completed" {
			log.Printf("[memory] review workflow %d completed, processing decision", workflowID)
			s.handleReviewComplete(memory, wf)
			return
		}
		if wf.Status == "failed" || wf.Status == "cancelled" {
			log.Printf("[memory] review workflow %d %s: %s", workflowID, wf.Status, wf.ErrorMsg)
			reason := "审核工作流失败: " + wf.ErrorMsg
			s.memoryDAO.UpdateReviewResult(memory.ID, model.MemoryStatusRejected, reason, workflowID)
			memory.Status = model.MemoryStatusRejected
			memory.ReviewReason = reason
			s.notifyMemoryUpdate(memory)
			return
		}

		<-timeAfter(1)
	}
	log.Printf("[memory] watch review workflow %d timed out for memory %d", workflowID, memory.ID)
}

// handleReviewComplete 处理审核工作流完成
func (s *MemoryService) handleReviewComplete(memory *model.WritingMemory, wf *model.AIWorkflow) {
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(wf.ResultJSON), &resultMap); err != nil {
		log.Printf("[memory] parse review workflow result failed: %v", err)
		return
	}

	// 解析 review_decision 节点输出
	decision := "rejected"
	reason := "无法解析审核结果"

	if decisionRaw, ok := resultMap["decision"]; ok {
		decisionStr := ""
		switch v := decisionRaw.(type) {
		case string:
			decisionStr = v
		default:
			b, _ := json.Marshal(v)
			decisionStr = string(b)
		}

		var decisionData struct {
			Decision    string   `json:"decision"`
			Reason      string   `json:"reason"`
			Suggestions []string `json:"suggestions"`
		}
		if json.Unmarshal([]byte(decisionStr), &decisionData) == nil {
			decision = decisionData.Decision
			reason = decisionData.Reason
		}
	}

	if decision == "approved" {
		memory.Status = model.MemoryStatusPublished
		memory.IsPublic = true
		memory.ReviewReason = reason
		s.memoryDAO.UpdateReviewResult(memory.ID, model.MemoryStatusPublished, reason, memory.ReviewWorkflowID)
	} else {
		memory.Status = model.MemoryStatusRejected
		memory.ReviewReason = reason
		s.memoryDAO.UpdateReviewResult(memory.ID, model.MemoryStatusRejected, reason, memory.ReviewWorkflowID)
	}

	s.notifyMemoryUpdate(memory)
	log.Printf("[memory] review completed for memory %d, decision=%s", memory.ID, decision)
}

// ListReviewingMemories 获取所有审核中的记忆（管理员用）
func (s *MemoryService) ListReviewingMemories() ([]model.WritingMemory, error) {
	return s.memoryDAO.ListByStatus(model.MemoryStatusReviewing)
}

// AdminApproveMemory 管理员强制通过审核
func (s *MemoryService) AdminApproveMemory(memoryID uint) error {
	memory, err := s.memoryDAO.Get(memoryID)
	if err != nil {
		return err
	}
	memory.Status = model.MemoryStatusPublished
	memory.IsPublic = true
	memory.ReviewReason = "管理员手动通过"
	s.memoryDAO.UpdateReviewResult(memoryID, model.MemoryStatusPublished, "管理员手动通过", 0)
	s.notifyMemoryUpdate(memory)
	return nil
}

// AdminRejectMemory 管理员强制拒绝审核
func (s *MemoryService) AdminRejectMemory(memoryID uint, reason string) error {
	memory, err := s.memoryDAO.Get(memoryID)
	if err != nil {
		return err
	}
	memory.Status = model.MemoryStatusRejected
	memory.ReviewReason = reason
	s.memoryDAO.UpdateReviewResult(memoryID, model.MemoryStatusRejected, reason, 0)
	s.notifyMemoryUpdate(memory)
	return nil
}

// ArchiveMemory 下架
func (s *MemoryService) ArchiveMemory(id uint) error {
	return s.memoryDAO.UpdateStatus(id, model.MemoryStatusArchived)
}

// GeneratePreview 生成预览文本
func (s *MemoryService) GeneratePreview(ctx context.Context, memory *model.WritingMemory, modelName string) (string, error) {
	if memory.PromptTpl == "" {
		return "", errors.New("memory not extracted yet")
	}

	if modelName == "" {
		modelName = "zhipu"
	}

	prompt := fmt.Sprintf("使用以下写作指令生成 100 字小说片段：\n\n%s\n\n请直接输出小说片段，不要添加任何说明。", memory.PromptTpl)

	provider, err := s.dispatcher.GetProvider(modelName)
	if err != nil {
		return "", fmt.Errorf("provider %s not found: %w", modelName, err)
	}

	resp, err := provider.GenerateText(ctx, &agent.TextRequest{
		Prompt:    prompt,
		MaxTokens: 512,
	})
	if err != nil {
		return "", err
	}

	// 截取前 500 字符
	preview := resp.Content
	if utf8.RuneCountInString(preview) > 500 {
		runes := []rune(preview)
		preview = string(runes[:500])
	}

	memory.PreviewText = preview
	s.memoryDAO.Update(memory)

	return preview, nil
}

// ========== 小说-记忆绑定 ==========

// SetBinding 设置小说-记忆绑定
func (s *MemoryService) SetBinding(novelID uint, category string, memoryID uint) error {
	if _, ok := model.ValidMemoryCategories[category]; !ok {
		return errors.New("invalid memory category")
	}

	binding := &model.NovelMemoryBinding{
		NovelID:  novelID,
		Category: category,
		MemoryID: memoryID,
	}
	return s.memoryDAO.UpsertBinding(binding)
}

// RemoveBinding 移除绑定
func (s *MemoryService) RemoveBinding(novelID uint, category string) error {
	return s.memoryDAO.DeleteBinding(novelID, category)
}

// ListBindings 获取小说的记忆绑定
func (s *MemoryService) ListBindings(novelID uint) ([]model.NovelMemoryBinding, error) {
	return s.memoryDAO.ListBindingsByNovel(novelID)
}

// GetBindingMemories 获取小说绑定的记忆详情
func (s *MemoryService) GetBindingMemories(novelID uint) ([]model.WritingMemory, error) {
	bindings, err := s.memoryDAO.ListBindingsByNovel(novelID)
	if err != nil {
		return nil, err
	}

	var memories []model.WritingMemory
	for _, b := range bindings {
		memory, err := s.memoryDAO.Get(b.MemoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		memories = append(memories, *memory)
	}
	return memories, nil
}

// ========== Embedding 相关 ==========

// GenerateEmbeddings 为记忆生成 Embedding
func (s *MemoryService) GenerateEmbeddings(ctx context.Context, memory *model.WritingMemory, modelName string) error {
	if modelName == "" {
		modelName = "zhipu"
	}

	provider, err := s.dispatcher.GetProvider(modelName)
	if err != nil {
		return fmt.Errorf("provider %s not found: %w", modelName, err)
	}

	// 按 500 字分块
	chunks := splitText(memory.SourceText, 500)
	if len(chunks) == 0 {
		return errors.New("no text to embed")
	}

	resp, err := provider.Embedding(ctx, &agent.EmbeddingRequest{
		Texts: chunks,
	})
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	// 删除旧 Embedding
	s.memoryDAO.DeleteEmbeddings(memory.ID)

	// 批量创建新 Embedding
	var embeddings []model.MemoryEmbedding
	for i, vec := range resp.Vectors {
		vecJSON, _ := json.Marshal(vec)
		embeddings = append(embeddings, model.MemoryEmbedding{
			MemoryID:  memory.ID,
			ChunkIdx:  i,
			ChunkText: chunks[i],
			Vector:    string(vecJSON),
			Dimension: resp.Dimension,
		})
	}

	return s.memoryDAO.BatchCreateEmbeddings(embeddings)
}

// GetRelevantChunks 根据查询文本检索最相关的样本片段
func (s *MemoryService) GetRelevantChunks(ctx context.Context, memoryID uint, queryText string, modelName string, topK int) ([]string, error) {
	if modelName == "" {
		modelName = "zhipu"
	}

	provider, err := s.dispatcher.GetProvider(modelName)
	if err != nil {
		return nil, fmt.Errorf("provider %s not found: %w", modelName, err)
	}

	// 获取查询文本的 Embedding
	queryResp, err := provider.Embedding(ctx, &agent.EmbeddingRequest{
		Texts: []string{queryText},
	})
	if err != nil {
		return nil, err
	}
	if len(queryResp.Vectors) == 0 {
		return nil, errors.New("empty query embedding")
	}
	queryVec := queryResp.Vectors[0]

	// 获取记忆的所有 Embedding
	embeddings, err := s.memoryDAO.ListEmbeddings(memoryID)
	if err != nil {
		return nil, err
	}

	// 计算余弦相似度并排序
	type scored struct {
		text  string
		score float64
	}
	var results []scored
	for _, emb := range embeddings {
		var vec []float64
		if err := json.Unmarshal([]byte(emb.Vector), &vec); err != nil {
			continue
		}
		sim := cosineSimilarity(queryVec, vec)
		results = append(results, scored{text: emb.ChunkText, score: sim})
	}

	// 按相似度降序排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// 取 topK
	if topK > len(results) {
		topK = len(results)
	}
	var chunks []string
	for i := 0; i < topK; i++ {
		chunks = append(chunks, results[i].text)
	}
	return chunks, nil
}

// ListAccessibleMemories 获取用户可用的记忆（自己的 + 已购买的）
func (s *MemoryService) ListAccessibleMemories(userID uint, category string) ([]model.WritingMemory, error) {
	return s.memoryDAO.ListAccessible(userID, category)
}

// ========== 工具函数 ==========

// splitText 按语义边界分块，优先按段落 > 句子 > 硬切
// chunkSize 为目标块大小（字符数），overlap 为块间重叠字符数
func splitText(text string, chunkSize int) []string {
	return splitTextSemantic(text, chunkSize, chunkSize/5)
}

func splitTextSemantic(text string, chunkSize, overlap int) []string {
	// 1. 按段落拆分（双换行 或 单换行）
	paragraphs := splitByParagraphs(text)
	if len(paragraphs) == 0 {
		return nil
	}

	var chunks []string
	var buf []rune

	for _, para := range paragraphs {
		paraRunes := []rune(para)

		// 如果单段落就超过 chunkSize，按句子进一步拆
		if len(paraRunes) > chunkSize {
			// 先 flush 当前 buf
			if len(buf) > 0 {
				chunks = append(chunks, strings.TrimSpace(string(buf)))
				buf = keepOverlap(buf, overlap)
			}
			sentenceChunks := splitLongParagraph(para, chunkSize, overlap)
			chunks = append(chunks, sentenceChunks...)
			continue
		}

		// 加入当前段落后是否超限
		if len(buf)+len(paraRunes)+1 > chunkSize && len(buf) > 0 {
			chunks = append(chunks, strings.TrimSpace(string(buf)))
			buf = keepOverlap(buf, overlap)
		}

		if len(buf) > 0 {
			buf = append(buf, '\n')
		}
		buf = append(buf, paraRunes...)
	}

	if len(buf) > 0 {
		s := strings.TrimSpace(string(buf))
		if s != "" {
			chunks = append(chunks, s)
		}
	}

	return chunks
}

// splitByParagraphs 按段落边界拆分，保留非空段落
func splitByParagraphs(text string) []string {
	// 先按双换行拆
	raw := strings.Split(text, "\n\n")
	var result []string
	for _, block := range raw {
		// 双换行内的单换行也作为段落边界
		lines := strings.Split(block, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, line)
			}
		}
	}
	return result
}

// splitLongParagraph 对超长段落按句子边界拆分
func splitLongParagraph(para string, chunkSize, overlap int) []string {
	sentences := splitBySentences(para)
	var chunks []string
	var buf []rune

	for _, sent := range sentences {
		sentRunes := []rune(sent)

		// 单句就超长，硬切
		if len(sentRunes) > chunkSize {
			if len(buf) > 0 {
				chunks = append(chunks, strings.TrimSpace(string(buf)))
				buf = keepOverlap(buf, overlap)
			}
			for i := 0; i < len(sentRunes); i += chunkSize - overlap {
				end := i + chunkSize
				if end > len(sentRunes) {
					end = len(sentRunes)
				}
				s := strings.TrimSpace(string(sentRunes[i:end]))
				if s != "" {
					chunks = append(chunks, s)
				}
			}
			continue
		}

		if len(buf)+len(sentRunes) > chunkSize && len(buf) > 0 {
			chunks = append(chunks, strings.TrimSpace(string(buf)))
			buf = keepOverlap(buf, overlap)
		}
		buf = append(buf, sentRunes...)
	}

	if len(buf) > 0 {
		s := strings.TrimSpace(string(buf))
		if s != "" {
			chunks = append(chunks, s)
		}
	}
	return chunks
}

// splitBySentences 按中英文句子终止符拆分
func splitBySentences(text string) []string {
	var sentences []string
	var buf []rune
	runes := []rune(text)

	for i, r := range runes {
		buf = append(buf, r)
		// 中文句末：。！？；  英文句末：. ! ? 后跟空格或结尾
		if r == '。' || r == '！' || r == '？' || r == '；' {
			sentences = append(sentences, string(buf))
			buf = nil
		} else if (r == '.' || r == '!' || r == '?') && (i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n') {
			sentences = append(sentences, string(buf))
			buf = nil
		}
	}
	if len(buf) > 0 {
		sentences = append(sentences, string(buf))
	}
	return sentences
}

// keepOverlap 保留 buf 末尾 overlap 个字符作为下一块的开头，保持上下文连贯
func keepOverlap(buf []rune, overlap int) []rune {
	if overlap <= 0 || len(buf) <= overlap {
		return nil
	}
	cp := make([]rune, overlap)
	copy(cp, buf[len(buf)-overlap:])
	return cp
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// RecoverStaleMemories 恢复卡住的记忆（服务启动时调用）
// 处理两种场景：
// 1. extract_status = "running" 但工作流已完成/失败/丢失 → 重新触发提取
// 2. status = "reviewing" 但审核工作流已完成/失败/丢失 → 重新触发审核
func (s *MemoryService) RecoverStaleMemories() {
	s.recoverStaleExtracts()
	s.recoverStaleReviews()
}

// recoverStaleExtracts 恢复卡住的提取任务
func (s *MemoryService) recoverStaleExtracts() {
	memories, err := s.memoryDAO.ListByExtractStatus("running")
	if err != nil {
		log.Printf("[memory-recovery] failed to query running extracts: %v", err)
		return
	}

	if len(memories) == 0 {
		return
	}

	log.Printf("[memory-recovery] found %d stale extracting memories", len(memories))

	ctx := context.Background()
	for i := range memories {
		mem := &memories[i]

		// 检查关联的工作流状态
		if mem.ExtractWorkflowID > 0 {
			wf, _, err := s.workflowService.GetWorkflow(ctx, mem.ExtractWorkflowID, mem.UserID)
			if err == nil && (wf.Status == model.WorkflowStatusPending || wf.Status == model.WorkflowStatusRunning) {
				// 工作流还在跑（由 WorkflowService.RecoverStaleWorkflows 处理），只需重新挂 watcher
				log.Printf("[memory-recovery] memory %d: extract workflow %d still %s, re-attaching watcher", mem.ID, wf.ID, wf.Status)
				go s.watchExtractWorkflow(mem, mem.ExtractWorkflowID)
				continue
			}
			if err == nil && wf.Status == model.WorkflowStatusCompleted {
				// 工作流已完成但记忆状态没更新，直接处理结果
				log.Printf("[memory-recovery] memory %d: extract workflow %d completed, processing results", mem.ID, wf.ID)
				nodes, _ := s.workflowService.workflowDAO.GetNodesByWorkflow(ctx, wf.ID)
				s.handleExtractComplete(mem, wf, nodes)
				continue
			}
		}

		// 工作流丢失或失败，重新触发提取
		log.Printf("[memory-recovery] memory %d: re-triggering extract workflow", mem.ID)
		go s.triggerExtractWorkflow(mem, "qwen")
	}
}

// recoverStaleReviews 恢复卡住的审核任务
func (s *MemoryService) recoverStaleReviews() {
	memories, err := s.memoryDAO.ListByStatus(model.MemoryStatusReviewing)
	if err != nil {
		log.Printf("[memory-recovery] failed to query reviewing memories: %v", err)
		return
	}

	if len(memories) == 0 {
		return
	}

	log.Printf("[memory-recovery] found %d stale reviewing memories", len(memories))

	ctx := context.Background()
	for i := range memories {
		mem := &memories[i]

		// 检查关联的审核工作流状态
		if mem.ReviewWorkflowID > 0 {
			wf, _, err := s.workflowService.GetWorkflow(ctx, mem.ReviewWorkflowID, mem.UserID)
			if err == nil && (wf.Status == model.WorkflowStatusPending || wf.Status == model.WorkflowStatusRunning) {
				// 工作流还在跑，重新挂 watcher
				log.Printf("[memory-recovery] memory %d: review workflow %d still %s, re-attaching watcher", mem.ID, wf.ID, wf.Status)
				go s.watchReviewWorkflow(mem, mem.ReviewWorkflowID)
				continue
			}
			if err == nil && wf.Status == model.WorkflowStatusCompleted {
				// 工作流已完成但记忆状态没更新
				log.Printf("[memory-recovery] memory %d: review workflow %d completed, processing decision", mem.ID, wf.ID)
				s.handleReviewComplete(mem, wf)
				continue
			}
		}

		// 工作流丢失或失败，重新触发审核
		log.Printf("[memory-recovery] memory %d: re-triggering review workflow", mem.ID)
		go s.triggerReviewWorkflow(mem)
	}
}
