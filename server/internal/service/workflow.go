// server/internal/service/workflow.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"ai-curton/server/internal/agent"
	"ai-curton/server/internal/agent/orchestrator"
	"ai-curton/server/internal/agent/tools"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// WorkflowService 工作流服务层
type WorkflowService struct {
	workflowDAO        *dao.AIWorkflowDAO
	chapterReviewDAO   *dao.ChapterReviewDAO
	dispatcher         *agent.Dispatcher
	notifier           agent.Notifier
	writingStyleSvc    *WritingStyleService
	knowledgeSvc       *KnowledgeService
	chapterSummarySvc  *ChapterSummaryService
	modelRegistry      *ModelRegistryService
}

// NewWorkflowService 创建 WorkflowService 实例
func NewWorkflowService(workflowDAO *dao.AIWorkflowDAO, dispatcher *agent.Dispatcher, notifier agent.Notifier, writingStyleSvc *WritingStyleService, knowledgeSvc *KnowledgeService) *WorkflowService {
	return &WorkflowService{
		workflowDAO:      workflowDAO,
		chapterReviewDAO: dao.NewChapterReviewDAO(),
		dispatcher:       dispatcher,
		notifier:         notifier,
		writingStyleSvc:  writingStyleSvc,
		knowledgeSvc:     knowledgeSvc,
	}
}

// SetChapterSummarySvc 注入递归摘要树服务（避免循环依赖）
func (s *WorkflowService) SetChapterSummarySvc(svc *ChapterSummaryService) {
	s.chapterSummarySvc = svc
}

// SetModelRegistry 注入模型注册表服务（延迟注入，避免循环依赖）
func (s *WorkflowService) SetModelRegistry(svc *ModelRegistryService) {
	s.modelRegistry = svc
}

// SubmitWorkflowRequest 提交工作流请求
type SubmitWorkflowRequest struct {
	PortfolioID  uint                   `json:"portfolio_id"`
	WorkflowType string                 `json:"workflow_type"`
	ModelName    string                 `json:"model_name"`
	Params       map[string]interface{} `json:"params"`
}

// SubmitWorkflow 提交并异步执行工作流
func (s *WorkflowService) SubmitWorkflow(ctx context.Context, userID uint, req *SubmitWorkflowRequest) (uint, error) {
	log.Printf("[workflow] SubmitWorkflow: type=%s, model=%s, userID=%d", req.WorkflowType, req.ModelName, userID)

	// 校验工作流类型
	if !isValidWorkflowType(req.WorkflowType) {
		return 0, errors.New("invalid workflow type")
	}
	if s.modelRegistry != nil {
		req.ModelName = s.modelRegistry.ResolveModel(req.ModelName, model.CapTextGen)
	} else if req.ModelName == "" {
		req.ModelName = "zhipu"
	}

	// 构建 DAG
	graph, err := s.buildGraph(req)
	if err != nil {
		return 0, fmt.Errorf("build graph failed: %w", err)
	}

	// 校验 DAG
	if err := graph.Validate(); err != nil {
		return 0, fmt.Errorf("invalid graph: %w", err)
	}

	// 序列化初始参数（包含 model_name 以便恢复时使用）
	if req.Params == nil {
		req.Params = make(map[string]interface{})
	}
	req.Params["model_name"] = req.ModelName
	paramsJSON, _ := json.Marshal(req.Params)

	// 从 Params 中提取 novel_id
	var novelID uint
	if novelIDRaw, ok := req.Params["novel_id"]; ok {
		switch v := novelIDRaw.(type) {
		case float64:
			novelID = uint(v)
		case string:
			if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
				novelID = uint(parsed)
			}
		}
	}

	// 创建工作流记录
	workflow := &model.AIWorkflow{
		UserID:         userID,
		PortfolioID:    req.PortfolioID,
		NovelID:        novelID,
		WorkflowType:   req.WorkflowType,
		Status:         model.WorkflowStatusPending,
		InitialContext: string(paramsJSON),
		TotalNodes:     graph.NodeCount(),
		CompletedNodes: 0,
	}

	if err := s.workflowDAO.Create(ctx, workflow); err != nil {
		return 0, fmt.Errorf("create workflow failed: %w", err)
	}

	log.Printf("[workflow] workflow %d created: type=%s, nodes=%d", workflow.ID, req.WorkflowType, graph.NodeCount())

	// 为每个节点创建记录（包括 LoopNode）
	for _, node := range graph.Nodes {
		wfNode := &model.AIWorkflowNode{
			WorkflowID: workflow.ID,
			NodeID:     node.ID,
			Status:     model.WorkflowStatusPending,
		}
		if err := s.workflowDAO.CreateNode(ctx, wfNode); err != nil {
			return 0, fmt.Errorf("create workflow node failed: %w", err)
		}
	}
	for _, loop := range graph.LoopNodes {
		wfNode := &model.AIWorkflowNode{
			WorkflowID: workflow.ID,
			NodeID:     loop.ID,
			Status:     model.WorkflowStatusPending,
		}
		if err := s.workflowDAO.CreateNode(ctx, wfNode); err != nil {
			return 0, fmt.Errorf("create workflow loop node failed: %w", err)
		}
	}

	// 异步执行工作流
	go s.runWorkflow(workflow, graph, req.Params)

	return workflow.ID, nil
}

// runWorkflow 异步执行工作流
func (s *WorkflowService) runWorkflow(workflow *model.AIWorkflow, graph *orchestrator.Graph, params map[string]interface{}) {
	log.Printf("[workflow] runWorkflow started: id=%d, type=%s", workflow.ID, workflow.WorkflowType)
	ctx := context.Background()

	// 更新状态为 running
	workflow.Status = model.WorkflowStatusRunning
	_ = s.workflowDAO.Update(ctx, workflow)
	s.notifyWorkflowUpdate(workflow)

	// 初始化共享状态，注入初始参数
	state := orchestrator.NewSharedState()
	for k, v := range params {
		state.Set(k, v)
	}

	// 创建引擎
	engine := orchestrator.NewEngine(
		s.makeNodeExecutor(workflow),
		s.makeProgressCallback(workflow, graph),
	)

	// 执行 DAG
	if err := engine.Run(ctx, workflow.ID, graph, state); err != nil {
		log.Printf("[workflow] workflow %d failed: %v", workflow.ID, err)
		workflow.Status = model.WorkflowStatusFailed
		workflow.ErrorMsg = err.Error()
		_ = s.workflowDAO.Update(ctx, workflow)
		s.notifyWorkflowUpdate(workflow)
		return
	}

	// 聚合最终结果
	resultJSON, _ := json.Marshal(state.Snapshot())
	workflow.Status = model.WorkflowStatusCompleted
	workflow.ResultJSON = string(resultJSON)
	workflow.CompletedNodes = workflow.TotalNodes
	_ = s.workflowDAO.Update(ctx, workflow)
	s.notifyWorkflowUpdate(workflow)

	log.Printf("[workflow] workflow %d completed", workflow.ID)
}

// makeNodeExecutor 创建节点执行器，桥接 Dispatcher.ExecuteSingle
// 支持模型降级：主模型限流/失败时自动切换 FallbackModels 中的备选模型
// 支持节点级 tool calling：EnableTools=true 时注入知识库工具
func (s *WorkflowService) makeNodeExecutor(workflow *model.AIWorkflow) orchestrator.NodeExecutor {
	// 预创建知识库 ToolRegistry（如果有知识库服务且 novelID 有效）
	var novelToolRegistry *agent.ToolRegistry
	if s.knowledgeSvc != nil && workflow.NovelID > 0 {
		novelToolRegistry = tools.NewNovelKnowledgeTools(workflow.NovelID, dao.NewKnowledgeDAO())
	}

	return func(ctx context.Context, node *orchestrator.Node, state *orchestrator.SharedState) (interface{}, error) {
		// 构建尝试列表：主模型 + 降级模型
		models := append([]string{node.ModelName}, node.FallbackModels...)

		// 节点级工具注入
		var nodeTools *agent.ToolRegistry
		if node.EnableTools && novelToolRegistry != nil {
			nodeTools = novelToolRegistry
		}

		var lastErr error
		for i, modelName := range models {
			if i > 0 {
				log.Printf("[workflow] node %s: model %s failed, falling back to %s", node.ID, models[i-1], modelName)
			}

			task := &model.AITask{
				UserID:      workflow.UserID,
				PortfolioID: workflow.PortfolioID,
				NovelID:     workflow.NovelID,
				TaskType:    node.TaskType,
				ModelName:   modelName,
				Prompt:      node.Prompt,
			}

			// 将节点级 MaxTokens 和 SystemPrompt 通过 History JSON 传递给 executor
			if node.MaxTokens > 0 || node.SystemPrompt != "" {
				historyData := map[string]interface{}{}
				if node.MaxTokens > 0 {
					historyData["max_tokens"] = node.MaxTokens
				}
				if node.SystemPrompt != "" {
					historyData["system_prompt"] = node.SystemPrompt
				}
				historyJSON, _ := json.Marshal(historyData)
				task.History = string(historyJSON)
			}

			result, err := s.dispatcher.ExecuteSingleWithTools(ctx, task, nodeTools)
			if err == nil {
				// 更新节点关联的 TaskID
				wfNode, _ := s.workflowDAO.GetNode(ctx, workflow.ID, node.ID)
				if wfNode != nil {
					wfNode.TaskID = task.ID
					_ = s.workflowDAO.UpdateNode(ctx, wfNode)
				}
				return result, nil
			}

			lastErr = err

			// 非可重试错误直接返回，不再降级
			if !agent.IsRetryableError(err) {
				if node.SkipOnAllFail {
					log.Printf("[workflow] node %s: non-retryable error and SkipOnAllFail=true, skipping", node.ID)
					return nil, nil
				}
				return nil, err
			}
		}

		// 所有模型耗尽：SkipOnAllFail 时跳过而非报错
		if node.SkipOnAllFail {
			log.Printf("[workflow] node %s: all models exhausted, SkipOnAllFail=true, skipping", node.ID)
			return nil, nil
		}
		return nil, fmt.Errorf("all models exhausted for node %s: %w", node.ID, lastErr)
	}
}

// makeProgressCallback 创建进度回调
func (s *WorkflowService) makeProgressCallback(workflow *model.AIWorkflow, graph *orchestrator.Graph) orchestrator.ProgressCallback {
	// 构建顶层节点 ID 集合，用于准确判断是否递增 completed_nodes
	topLevelNodes := make(map[string]bool, len(graph.Nodes)+len(graph.LoopNodes))
	for id := range graph.Nodes {
		topLevelNodes[id] = true
	}
	for id := range graph.LoopNodes {
		topLevelNodes[id] = true
	}

	// 已递增过的顶层节点集合，防止重复递增 completed_nodes
	completedSet := make(map[string]bool)
	var mu sync.Mutex

	return func(workflowID uint, nodeID string, status string, result interface{}) {
		ctx := context.Background()

		// 更新节点状态（找不到则动态创建，用于 loop 子节点）
		wfNode, _ := s.workflowDAO.GetNode(ctx, workflowID, nodeID)
		if wfNode == nil {
			wfNode = &model.AIWorkflowNode{
				WorkflowID: workflowID,
				NodeID:     nodeID,
				Status:     status,
			}
			if result != nil {
				resultJSON, _ := json.Marshal(result)
				wfNode.ResultJSON = string(resultJSON)
			}
			_ = s.workflowDAO.CreateNode(ctx, wfNode)
		} else {
			wfNode.Status = status
			if result != nil {
				resultJSON, _ := json.Marshal(result)
				wfNode.ResultJSON = string(resultJSON)
			}
			_ = s.workflowDAO.UpdateNode(ctx, wfNode)
		}

		// 只有顶层节点完成才递增 completed_nodes（loop 子节点不参与整体进度）
		// 使用 completedSet 防止同一节点重复递增
		if topLevelNodes[nodeID] && (status == "completed" || status == "completed_with_warning") {
			mu.Lock()
			alreadyCounted := completedSet[nodeID]
			if !alreadyCounted {
				completedSet[nodeID] = true
			}
			mu.Unlock()

			if !alreadyCounted {
				_ = s.workflowDAO.IncrCompletedNodes(ctx, workflowID)
			}
			// 重新读取最新状态后通知前端
			wf, _ := s.workflowDAO.Get(ctx, workflowID)
			if wf != nil {
				s.notifyWorkflowUpdate(wf)
			}
		}

		// 审核评分持久化：review 子节点完成时也需要保存评分
		if status == "completed" {
			s.persistReviewScore(ctx, workflow, nodeID, result)
		}

		// WebSocket 通知节点更新
		s.notifyNodeUpdate(workflow.UserID, workflowID, nodeID, status, result)
	}
}

// GetWorkflow 获取工作流详情
func (s *WorkflowService) GetWorkflow(ctx context.Context, workflowID, userID uint) (*model.AIWorkflow, []model.AIWorkflowNode, error) {
	wf, err := s.workflowDAO.Get(ctx, workflowID)
	if err != nil {
		return nil, nil, err
	}
	if wf.UserID != userID {
		return nil, nil, errors.New("permission denied")
	}

	nodes, err := s.workflowDAO.GetNodesByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, nil, err
	}

	return wf, nodes, nil
}

// CancelWorkflow 取消工作流
func (s *WorkflowService) CancelWorkflow(ctx context.Context, workflowID, userID uint) error {
	wf, err := s.workflowDAO.Get(ctx, workflowID)
	if err != nil {
		return err
	}
	if wf.UserID != userID {
		return errors.New("permission denied")
	}
	if wf.Status == model.WorkflowStatusCompleted || wf.Status == model.WorkflowStatusFailed {
		return errors.New("cannot cancel completed or failed workflow")
	}

	wf.Status = model.WorkflowStatusCancelled
	return s.workflowDAO.Update(ctx, wf)
}

// notifyWorkflowUpdate 通知工作流状态更新
func (s *WorkflowService) notifyWorkflowUpdate(wf *model.AIWorkflow) {
	if s.notifier == nil {
		return
	}
	data := map[string]interface{}{
		"id":              wf.ID,
		"status":          wf.Status,
		"completed_nodes": wf.CompletedNodes,
		"total_nodes":     wf.TotalNodes,
		"error_msg":       wf.ErrorMsg,
		"result_json":     wf.ResultJSON,
	}
	_ = s.notifier.NotifyUserWithType(wf.UserID, "workflow_update", data)
}

// notifyNodeUpdate 通知节点状态更新
func (s *WorkflowService) notifyNodeUpdate(userID, workflowID uint, nodeID, status string, result interface{}) {
	if s.notifier == nil {
		return
	}
	data := map[string]interface{}{
		"workflow_id": workflowID,
		"node_id":     nodeID,
		"status":      status,
		"result_json": result,
	}
	_ = s.notifier.NotifyUserWithType(userID, "workflow_node_update", data)
}

// buildGraph 根据工作流类型构建 DAG
func (s *WorkflowService) buildGraph(req *SubmitWorkflowRequest) (*orchestrator.Graph, error) {
	// 从 Params 中提取 novel_id
	var novelID uint
	if novelIDRaw, ok := req.Params["novel_id"]; ok {
		switch v := novelIDRaw.(type) {
		case float64:
			novelID = uint(v)
		case string:
			if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
				novelID = uint(parsed)
			}
		}
	}

	// 查询写作风格
	var writingStyle string
	if s.writingStyleSvc != nil && novelID > 0 {
		// 提取可选的 scene_preset_id
		var scenePresetID *uint
		if spRaw, ok := req.Params["scene_preset_id"]; ok {
			var spID uint
			switch v := spRaw.(type) {
			case float64:
				spID = uint(v)
			case string:
				if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
					spID = uint(parsed)
				}
			}
			if spID > 0 {
				scenePresetID = &spID
			}
		}
		writingStyle = s.writingStyleSvc.FormatStyleForPrompt(novelID, scenePresetID)
	}

	switch req.WorkflowType {
	case model.WorkflowTypeFullChapter:
		// 使用 Token Budget Manager 替代硬编码 maxChars
		budget := agent.NewTokenBudget(req.ModelName)
		var knowledgeContext string
		if s.knowledgeSvc != nil && novelID > 0 {
			knowledgeContext, _, _ = s.knowledgeSvc.BuildKnowledgeContext(novelID, budget.CharsForKnowledge())
			if knowledgeContext != "" {
				req.Params["knowledge_context"] = knowledgeContext
			}
		}
		// 注入递归摘要树上下文替代前端传入的 prev_context
		if s.chapterSummarySvc != nil && novelID > 0 {
			if chapterOrderRaw, ok := req.Params["chapter_sort_order"]; ok {
				var chapterOrder int
				switch v := chapterOrderRaw.(type) {
				case float64:
					chapterOrder = int(v)
				case int:
					chapterOrder = v
				}
				if chapterOrder > 0 {
					summaryContext := s.chapterSummarySvc.GetRelevantContext(novelID, chapterOrder, budget.CharsForPrevContext())
					if summaryContext != "" {
						req.Params["prev_context"] = summaryContext
					}
				}
			}
		}
		return orchestrator.BuildFullChapterGraph(req.ModelName, writingStyle, knowledgeContext, func(primary string) []string {
			if s.modelRegistry != nil {
				return s.modelRegistry.GetFallbackChain(0, primary, model.CapTextGen)
			}
			return orchestrator.FallbackModels(primary)
		}), nil
	case model.WorkflowTypeBatchExpand:
		chapters, err := parseBatchExpandParams(req.Params)
		if err != nil {
			return nil, err
		}
		return orchestrator.BuildBatchExpandGraph(req.ModelName, chapters, writingStyle, func(primary string) []string {
			if s.modelRegistry != nil {
				return s.modelRegistry.GetFallbackChain(0, primary, model.CapTextGen)
			}
			return orchestrator.FallbackModels(primary)
		}), nil
	case model.WorkflowTypeNovelRevision:
		diffText, _ := req.Params["diff_text"].(string)
		chapterIndex, _ := req.Params["chapter_index"].(string)
		if diffText == "" || chapterIndex == "" {
			return nil, fmt.Errorf("missing diff_text or chapter_index")
		}
		return orchestrator.BuildNovelRevisionGraph(req.ModelName, diffText, chapterIndex), nil
	case model.WorkflowTypeNovelRevisionExec:
		revisionPlan, _ := req.Params["revision_plan"].(string)
		chapterIndex, _ := req.Params["chapter_index"].(string)
		chapterCountRaw, _ := req.Params["chapter_count"].(float64)
		chapterCount := int(chapterCountRaw)
		if revisionPlan == "" || chapterCount == 0 {
			return nil, fmt.Errorf("missing revision_plan or chapter_count")
		}
		return orchestrator.BuildNovelRevisionExecuteGraph(req.ModelName, revisionPlan, chapterIndex, chapterCount), nil
	case model.WorkflowTypeMemoryExtract:
		category, _ := req.Params["category"].(string)
		if category == "" {
			return nil, fmt.Errorf("missing category parameter")
		}
		return orchestrator.BuildMemoryExtractGraph(req.ModelName, category), nil
	case model.WorkflowTypeMemoryReview:
		return orchestrator.BuildMemoryReviewGraph(req.ModelName), nil
	case model.WorkflowTypeHitAnalysis:
		sourceText, _ := req.Params["source_text"].(string)
		if sourceText == "" {
			return nil, fmt.Errorf("missing source_text parameter")
		}
		return orchestrator.BuildHitAnalysisGraph(req.ModelName, sourceText), nil
	default:
		return nil, fmt.Errorf("unsupported workflow type: %s", req.WorkflowType)
	}
}

// parseBatchExpandParams 解析批量扩写参数
func parseBatchExpandParams(params map[string]interface{}) ([]orchestrator.ChapterInput, error) {
	chaptersRaw, ok := params["chapters"]
	if !ok {
		return nil, errors.New("missing chapters parameter")
	}

	chaptersJSON, err := json.Marshal(chaptersRaw)
	if err != nil {
		return nil, err
	}

	var chapters []orchestrator.ChapterInput
	if err := json.Unmarshal(chaptersJSON, &chapters); err != nil {
		return nil, fmt.Errorf("invalid chapters format: %w", err)
	}

	if len(chapters) == 0 {
		return nil, errors.New("chapters cannot be empty")
	}

	return chapters, nil
}

// RecoverStaleWorkflows 恢复卡住的工作流（服务启动时调用）
// 查找状态为 pending/running 但超过 2 分钟未更新的工作流，重新提交执行
func (s *WorkflowService) RecoverStaleWorkflows() {
	ctx := context.Background()
	staleBefore := time.Now().Add(-2 * time.Minute)

	staleWorkflows, err := s.workflowDAO.ListStale(ctx, staleBefore)
	if err != nil {
		log.Printf("[workflow-recovery] failed to query stale workflows: %v", err)
		return
	}

	if len(staleWorkflows) == 0 {
		log.Printf("[workflow-recovery] no stale workflows found")
		return
	}

	log.Printf("[workflow-recovery] found %d stale workflows, recovering...", len(staleWorkflows))

	for i := range staleWorkflows {
		wf := &staleWorkflows[i]
		log.Printf("[workflow-recovery] recovering workflow %d (type=%s, status=%s, updated=%s)",
			wf.ID, wf.WorkflowType, wf.Status, wf.UpdatedAt.Format(time.RFC3339))

		// 从 InitialContext 恢复原始参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(wf.InitialContext), &params); err != nil {
			log.Printf("[workflow-recovery] workflow %d: failed to parse InitialContext: %v", wf.ID, err)
			s.markWorkflowFailed(ctx, wf, "恢复失败: 无法解析初始参数")
			continue
		}

		// 推断模型名（从已有节点或默认 qwen）
		modelName := s.inferModelName(ctx, wf)

		// 重建 DAG
		req := &SubmitWorkflowRequest{
			PortfolioID:  wf.PortfolioID,
			WorkflowType: wf.WorkflowType,
			ModelName:    modelName,
			Params:       params,
		}
		graph, err := s.buildGraph(req)
		if err != nil {
			log.Printf("[workflow-recovery] workflow %d: failed to rebuild graph: %v", wf.ID, err)
			s.markWorkflowFailed(ctx, wf, "恢复失败: 无法重建 DAG")
			continue
		}

		// 重置工作流状态，删除所有旧节点（避免残留状态混乱）
		wf.Status = model.WorkflowStatusPending
		wf.CompletedNodes = 0
		wf.ErrorMsg = ""
		_ = s.workflowDAO.Update(ctx, wf)
		_ = s.workflowDAO.DeleteNodes(ctx, wf.ID)

		// 异步重新执行
		go s.runWorkflow(wf, graph, params)
		log.Printf("[workflow-recovery] workflow %d re-submitted", wf.ID)
	}
}

// inferModelName 从工作流的已有节点推断模型名，找不到则返回默认值
func (s *WorkflowService) inferModelName(ctx context.Context, wf *model.AIWorkflow) string {
	// 尝试从 InitialContext 中的 model_name 字段获取
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(wf.InitialContext), &params); err == nil {
		if mn, ok := params["model_name"].(string); ok && mn != "" {
			return mn
		}
	}
	if s.modelRegistry != nil {
		return s.modelRegistry.GetDefaultModel(model.CapTextGen)
	}
	return "qwen"
}

// markWorkflowFailed 将工作流标记为失败
func (s *WorkflowService) markWorkflowFailed(ctx context.Context, wf *model.AIWorkflow, errMsg string) {
	wf.Status = model.WorkflowStatusFailed
	wf.ErrorMsg = errMsg
	_ = s.workflowDAO.Update(ctx, wf)
	s.notifyWorkflowUpdate(wf)
}

// ListActiveByNovel 查询指定小说下当前用户的活跃工作流
func (s *WorkflowService) ListActiveByNovel(ctx context.Context, novelID uint, userID uint) ([]model.AIWorkflow, error) {
	workflows, err := s.workflowDAO.ListActiveByNovel(ctx, novelID)
	if err != nil {
		return nil, fmt.Errorf("list active workflows failed: %w", err)
	}
	// 过滤只返回当前用户的工作流
	result := make([]model.AIWorkflow, 0, len(workflows))
	for _, wf := range workflows {
		if wf.UserID == userID {
			result = append(result, wf)
		}
	}
	return result, nil
}

// isValidWorkflowType 校验工作流类型白名单
func isValidWorkflowType(wfType string) bool {
	switch wfType {
	case model.WorkflowTypeFullChapter, model.WorkflowTypeBatchExpand,
		model.WorkflowTypeNovelRevision, model.WorkflowTypeNovelRevisionExec,
		model.WorkflowTypeMemoryExtract, model.WorkflowTypeMemoryReview,
		model.WorkflowTypeHitAnalysis:
		return true
	}
	return false
}

// reviewScoreResult 审核评分 JSON 解析结构
// issues 使用 json.RawMessage 兼容旧版 []string 和新版 []{quote,problem,fix} 三元组
type reviewScoreResult struct {
	Passed       bool `json:"passed"`
	OverallScore int  `json:"overall_score"`
	Dimensions   struct {
		CharacterConsistency struct {
			Score  int                `json:"score"`
			Issues []json.RawMessage  `json:"issues"`
		} `json:"character_consistency"`
		PlotCoherence struct {
			Score  int                `json:"score"`
			Issues []json.RawMessage  `json:"issues"`
		} `json:"plot_coherence"`
		WorldviewCompliance struct {
			Score  int                `json:"score"`
			Issues []json.RawMessage  `json:"issues"`
		} `json:"worldview_compliance"`
		NarrativeQuality struct {
			Score  int                `json:"score"`
			Issues []json.RawMessage  `json:"issues"`
		} `json:"narrative_quality"`
		Continuity struct {
			Score  int                `json:"score"`
			Issues []json.RawMessage  `json:"issues"`
		} `json:"continuity"`
		Formatting struct {
			Score  int                `json:"score"`
			Issues []json.RawMessage  `json:"issues"`
		} `json:"formatting"`
		AIArtifacts struct {
			Score  int                `json:"score"`
			Issues []json.RawMessage  `json:"issues"`
		} `json:"ai_artifacts"`
	} `json:"dimensions"`
	RevisionInstructions string `json:"revision_instructions"`
}

// persistReviewScore 持久化审核评分
// 当审核节点完成时，解析 JSON 结果并创建 ChapterReview 记录
// 兼容旧版静态节点（supervisor_review_1/2）和新版 LoopNode 子节点（review）
func (s *WorkflowService) persistReviewScore(ctx context.Context, workflow *model.AIWorkflow, nodeID string, result interface{}) {
	// 匹配审核节点：旧版 supervisor_review_N 或新版 review（LoopNode 子节点）
	var round int
	switch nodeID {
	case "supervisor_review_1":
		round = 1
	case "supervisor_review_2":
		round = 2
	case "review":
		// LoopNode 子节点，从 result 中尝试提取轮次，默认为 1
		round = 1
		if m, ok := result.(map[string]interface{}); ok {
			if r, ok := m["loop_round"].(int); ok {
				round = r
			}
		}
	default:
		return
	}

	// 获取结果文本
	resultStr, ok := result.(string)
	if !ok {
		return
	}

	// 解析 JSON
	var score reviewScoreResult
	if err := json.Unmarshal([]byte(resultStr), &score); err != nil {
		log.Printf("[workflow] 解析审核评分 JSON 失败 (node=%s): %v", nodeID, err)
		return
	}

	// 从 workflow 的 InitialContext 中提取 novel_id 和 chapter_id
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(workflow.InitialContext), &params); err != nil {
		return
	}
	var novelID, chapterID uint
	if v, ok := params["novel_id"].(float64); ok {
		novelID = uint(v)
	}
	if v, ok := params["chapter_id"].(float64); ok {
		chapterID = uint(v)
	}

	// 序列化 issues 为 JSON 字符串（兼容新旧格式：[]json.RawMessage 直接序列化）
	marshalIssues := func(issues []json.RawMessage) string {
		if len(issues) == 0 {
			return "[]"
		}
		b, _ := json.Marshal(issues)
		return string(b)
	}

	review := &model.ChapterReview{
		WorkflowID:           workflow.ID,
		NovelID:              novelID,
		ChapterID:            chapterID,
		Round:                round,
		OverallScore:         score.OverallScore,
		Passed:               score.Passed,
		CharacterScore:       score.Dimensions.CharacterConsistency.Score,
		CharacterIssues:      marshalIssues(score.Dimensions.CharacterConsistency.Issues),
		PlotScore:            score.Dimensions.PlotCoherence.Score,
		PlotIssues:           marshalIssues(score.Dimensions.PlotCoherence.Issues),
		WorldviewScore:       score.Dimensions.WorldviewCompliance.Score,
		WorldviewIssues:      marshalIssues(score.Dimensions.WorldviewCompliance.Issues),
		NarrativeScore:       score.Dimensions.NarrativeQuality.Score,
		NarrativeIssues:      marshalIssues(score.Dimensions.NarrativeQuality.Issues),
		ContinuityScore:      score.Dimensions.Continuity.Score,
		ContinuityIssues:     marshalIssues(score.Dimensions.Continuity.Issues),
		FormattingScore:      score.Dimensions.Formatting.Score,
		FormattingIssues:     marshalIssues(score.Dimensions.Formatting.Issues),
		AIArtifactsScore:     score.Dimensions.AIArtifacts.Score,
		AIArtifactsIssues:    marshalIssues(score.Dimensions.AIArtifacts.Issues),
		RevisionInstructions: score.RevisionInstructions,
	}

	if err := s.chapterReviewDAO.Create(review); err != nil {
		log.Printf("[workflow] 保存审核评分失败 (workflow=%d, node=%s): %v", workflow.ID, nodeID, err)
	} else {
		log.Printf("[workflow] 审核评分已保存: workflow=%d, round=%d, score=%d, passed=%v",
			workflow.ID, round, score.OverallScore, score.Passed)
	}
}
