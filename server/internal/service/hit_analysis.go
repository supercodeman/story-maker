// server/internal/service/hit_analysis.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"unicode/utf8"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"
)

// HitAnalysisService 爆款拆解服务层
type HitAnalysisService struct {
	dao           *dao.HitAnalysisDAO
	workflowSvc   *WorkflowService
	modelRegistry *ModelRegistryService
}

// SetModelRegistry 延迟注入模型注册中心
func (s *HitAnalysisService) SetModelRegistry(mr *ModelRegistryService) {
	s.modelRegistry = mr
}

// NewHitAnalysisService 创建 HitAnalysisService 实例
func NewHitAnalysisService(haDAO *dao.HitAnalysisDAO, workflowSvc *WorkflowService) *HitAnalysisService {
	return &HitAnalysisService{dao: haDAO, workflowSvc: workflowSvc}
}

// ========== 请求参数定义 ==========

// SubmitHitAnalysisRequest 提交爆款拆解请求
type SubmitHitAnalysisRequest struct {
	PortfolioID uint   `json:"portfolio_id" binding:"required"`
	Title       string `json:"title" binding:"required,max=200"`
	Author      string `json:"author"`
	SourceText  string `json:"source_text" binding:"required"` // 分析素材
	ModelName   string `json:"model_name"`
}

// ========== 业务方法 ==========

// Submit 提交爆款拆解：创建记录 + 提交工作流
func (s *HitAnalysisService) Submit(ctx context.Context, userID uint, req *SubmitHitAnalysisRequest) (*model.HitAnalysis, error) {
	// 版权合规：限制输入长度
	if utf8.RuneCountInString(req.SourceText) > 10000 {
		return nil, errors.New("source text exceeds 10000 characters limit, please use synopsis instead of full text")
	}

	modelName := req.ModelName
	if modelName == "" {
		if s.modelRegistry != nil {
			modelName = s.modelRegistry.GetDefaultModel(model.CapTextGen)
		} else {
			modelName = "zhipu"
		}
	}

	// 创建拆解记录
	ha := &model.HitAnalysis{
		UserID:      userID,
		PortfolioID: req.PortfolioID,
		Title:       req.Title,
		Author:      req.Author,
		SourceText:  req.SourceText,
		Status:      "pending",
		ModelName:   modelName,
	}
	if err := s.dao.Create(ctx, ha); err != nil {
		return nil, fmt.Errorf("create hit analysis failed: %w", err)
	}

	// 提交工作流
	wfReq := &SubmitWorkflowRequest{
		PortfolioID:  req.PortfolioID,
		WorkflowType: model.WorkflowTypeHitAnalysis,
		ModelName:    modelName,
		Params: map[string]interface{}{
			"source_text":     req.SourceText,
			"title":           req.Title,
			"author":          req.Author,
			"hit_analysis_id": ha.ID,
		},
	}

	workflowID, err := s.workflowSvc.SubmitWorkflow(ctx, userID, wfReq)
	if err != nil {
		// 工作流提交失败，更新状态
		ha.Status = "failed"
		_ = s.dao.Update(ctx, ha)
		return nil, fmt.Errorf("submit workflow failed: %w", err)
	}

	// 回写 workflowID
	ha.WorkflowID = workflowID
	ha.Status = "running"
	_ = s.dao.Update(ctx, ha)

	return ha, nil
}

// Get 获取拆解记录（含工作流完成后的回写逻辑）
func (s *HitAnalysisService) Get(ctx context.Context, id, userID uint) (*model.HitAnalysis, error) {
	ha, err := s.dao.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if ha.UserID != userID {
		return nil, errors.New("permission denied")
	}

	// 如果状态为 running 且有 workflowID，检查工作流是否已完成
	if ha.Status == "running" && ha.WorkflowID > 0 {
		wf, _, wfErr := s.workflowSvc.GetWorkflow(ctx, ha.WorkflowID, userID)
		if wfErr == nil && wf.Status == model.WorkflowStatusCompleted {
			// 从工作流结果中提取 synthesis_result
			var resultMap map[string]interface{}
			if err := json.Unmarshal([]byte(wf.ResultJSON), &resultMap); err == nil {
				if synthesis, ok := resultMap["synthesis_result"]; ok {
					reportJSON, _ := json.Marshal(synthesis)
					ha.Report = string(reportJSON)
				}
			}
			ha.Status = "completed"
			// 版权合规：清除原文
			ha.SourceText = ""
			_ = s.dao.Update(ctx, ha)
			_ = s.dao.ClearSourceText(ctx, ha.ID)
		} else if wfErr == nil && wf.Status == model.WorkflowStatusFailed {
			ha.Status = "failed"
			_ = s.dao.Update(ctx, ha)
		}
	}

	return ha, nil
}

// List 获取用户的拆解记录列表
func (s *HitAnalysisService) List(ctx context.Context, userID uint) ([]model.HitAnalysis, error) {
	list, err := s.dao.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	// 列表中不返回 source_text 和完整 report，减少传输量
	for i := range list {
		list[i].SourceText = ""
		if len(list[i].Report) > 200 {
			list[i].Report = list[i].Report[:200] + "..."
		}
	}
	return list, nil
}

// Delete 删除拆解记录
func (s *HitAnalysisService) Delete(ctx context.Context, id, userID uint) error {
	ha, err := s.dao.Get(ctx, id)
	if err != nil {
		return err
	}
	if ha.UserID != userID {
		return errors.New("permission denied")
	}
	return s.dao.Delete(ctx, id)
}

// FormatReportForPrompt 将拆解报告格式化为 prompt 参考文本
func (s *HitAnalysisService) FormatReportForPrompt(ctx context.Context, id, userID uint) (string, error) {
	ha, err := s.Get(ctx, id, userID)
	if err != nil {
		return "", err
	}
	if ha.Status != "completed" || ha.Report == "" {
		return "", errors.New("hit analysis report not ready")
	}

	return fmt.Sprintf("【爆款拆解参考：《%s》（%s）】\n%s\n\n请参考以上拆解报告中的结构模式、节奏规律和叙事技法，但不要复制其具体内容。", ha.Title, ha.Author, ha.Report), nil
}

func init() {
	log.Println("[init] hit_analysis service loaded")
}
