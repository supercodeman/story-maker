// server/internal/service/comic_drama.go
package service

import (
	"context"
	"fmt"

	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// ComicDramaService 漫剧业务服务
type ComicDramaService struct {
	comicDramaDAO *dao.ComicDramaDAO
	orchestrator  *PipelineOrchestrator
}

func NewComicDramaService(cdDAO *dao.ComicDramaDAO) *ComicDramaService {
	return &ComicDramaService{comicDramaDAO: cdDAO}
}

// SetOrchestrator 延迟注入编排器（避免循环依赖）
func (s *ComicDramaService) SetOrchestrator(o *PipelineOrchestrator) {
	s.orchestrator = o
}

// CreateComicDramaReq 创建漫剧请求
type CreateComicDramaReq struct {
	NovelID   uint   `json:"novel_id"`
	ChapterID uint   `json:"chapter_id"`
	Title     string `json:"title"`
	Config    string `json:"config"`
}

func (s *ComicDramaService) CreateComicDrama(ctx context.Context, userID uint, req *CreateComicDramaReq) (*model.ComicDrama, error) {
	drama := &model.ComicDrama{
		UserID:    userID,
		NovelID:   req.NovelID,
		ChapterID: req.ChapterID,
		Title:     req.Title,
		Stage:     model.ComicStageDraft,
		Status:    model.ComicStatusPending,
		Config:    req.Config,
	}
	if err := s.comicDramaDAO.CreateComicDrama(ctx, drama); err != nil {
		return nil, fmt.Errorf("failed to create comic drama: %w", err)
	}
	return drama, nil
}

func (s *ComicDramaService) GetComicDrama(ctx context.Context, dramaID, userID uint) (*model.ComicDrama, error) {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return nil, fmt.Errorf("comic drama not found")
	}
	return drama, nil
}

func (s *ComicDramaService) ListComicDramas(ctx context.Context, userID uint, page, pageSize int) ([]*model.ComicDrama, int64, error) {
	offset := (page - 1) * pageSize
	return s.comicDramaDAO.ListComicDramasByUser(ctx, userID, pageSize, offset)
}

func (s *ComicDramaService) DeleteComicDrama(ctx context.Context, dramaID, userID uint) error {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	if drama.Status == model.ComicStatusRunning {
		return fmt.Errorf("cannot delete comic drama while running")
	}
	return s.comicDramaDAO.DeleteComicDrama(ctx, dramaID)
}

func (s *ComicDramaService) UpdateConfig(ctx context.Context, dramaID, userID uint, config string) error {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	drama.Config = config
	return s.comicDramaDAO.UpdateComicDrama(ctx, drama)
}

func (s *ComicDramaService) StartPipeline(ctx context.Context, dramaID, userID uint) error {
	if s.orchestrator == nil {
		return fmt.Errorf("orchestrator not initialized")
	}
	return s.orchestrator.StartPipeline(ctx, userID, dramaID)
}

func (s *ComicDramaService) AdvanceStage(ctx context.Context, dramaID, userID uint) error {
	if s.orchestrator == nil {
		return fmt.Errorf("orchestrator not initialized")
	}
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	return s.orchestrator.AdvanceStage(ctx, dramaID, drama.Stage)
}

func (s *ComicDramaService) RetryFailed(ctx context.Context, dramaID, userID uint) error {
	if s.orchestrator == nil {
		return fmt.Errorf("orchestrator not initialized")
	}
	return s.orchestrator.RetryFailed(ctx, userID, dramaID)
}

func (s *ComicDramaService) GetScript(ctx context.Context, dramaID, userID uint) ([]*model.ComicScript, error) {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return nil, fmt.Errorf("comic drama not found")
	}
	return s.comicDramaDAO.ListScriptsByDrama(ctx, dramaID)
}

func (s *ComicDramaService) UpdateScript(ctx context.Context, dramaID, userID uint, scripts []*model.ComicScript) error {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	// 删除旧剧本，批量写入新剧本
	if err := s.comicDramaDAO.DeleteScriptsByDrama(ctx, dramaID); err != nil {
		return fmt.Errorf("failed to delete old scripts: %w", err)
	}
	for i := range scripts {
		scripts[i].ComicDramaID = dramaID
		scripts[i].SeqNo = i
	}
	return s.comicDramaDAO.BatchCreateScripts(ctx, scripts)
}

func (s *ComicDramaService) ApproveScript(ctx context.Context, dramaID, userID uint) error {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	if drama.Stage != model.ComicStageScript {
		return fmt.Errorf("script stage not ready for approval")
	}
	return s.orchestrator.AdvanceStage(ctx, dramaID, model.ComicStageScript)
}

func (s *ComicDramaService) GetStoryboard(ctx context.Context, dramaID, userID uint) ([]*model.Storyboard, error) {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return nil, fmt.Errorf("comic drama not found")
	}
	return s.comicDramaDAO.ListStoryboardsByDrama(ctx, dramaID)
}

func (s *ComicDramaService) UpdateStoryboardShot(ctx context.Context, shotID, userID uint, updates map[string]interface{}) error {
	board, err := s.comicDramaDAO.GetStoryboard(ctx, shotID)
	if err != nil {
		return fmt.Errorf("failed to get storyboard: %w", err)
	}
	// 验证所属漫剧的 user_id
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, board.ComicDramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("storyboard not found")
	}
	// 白名单更新字段
	allowedFields := map[string]bool{
		"frame_desc": true, "camera_angle": true, "characters": true,
	}
	for k := range updates {
		if !allowedFields[k] {
			delete(updates, k)
		}
	}
	if v, ok := updates["frame_desc"]; ok {
		board.FrameDesc = fmt.Sprintf("%v", v)
	}
	if v, ok := updates["camera_angle"]; ok {
		board.CameraAngle = fmt.Sprintf("%v", v)
	}
	if v, ok := updates["characters"]; ok {
		board.Characters = fmt.Sprintf("%v", v)
	}
	return s.comicDramaDAO.UpdateStoryboard(ctx, board)
}

func (s *ComicDramaService) ApproveStoryboard(ctx context.Context, dramaID, userID uint) error {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	if drama.Stage != model.ComicStageStoryboard {
		return fmt.Errorf("storyboard stage not ready for approval")
	}
	return s.orchestrator.AdvanceStage(ctx, dramaID, model.ComicStageStoryboard)
}

func (s *ComicDramaService) GetCharacters(ctx context.Context, dramaID, userID uint) ([]*model.CharacterRef, error) {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return nil, fmt.Errorf("comic drama not found")
	}
	return s.comicDramaDAO.ListCharacterRefsByDrama(ctx, dramaID)
}

func (s *ComicDramaService) RegenerateCharacter(ctx context.Context, charID, userID uint) error {
	ref, err := s.comicDramaDAO.GetCharacterRef(ctx, charID)
	if err != nil {
		return fmt.Errorf("failed to get character ref: %w", err)
	}
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, ref.ComicDramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("character not found")
	}
	ref.Status = model.ComicStatusPending
	ref.RefImageURL = ""
	if err := s.comicDramaDAO.UpdateCharacterRef(ctx, ref); err != nil {
		return err
	}
	return s.orchestrator.SubmitCharRefTask(ctx, drama, ref)
}

func (s *ComicDramaService) ApproveCharacters(ctx context.Context, dramaID, userID uint) error {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	if drama.Stage != model.ComicStageCharRef {
		return fmt.Errorf("char_ref stage not ready for approval")
	}
	return s.orchestrator.AdvanceStage(ctx, dramaID, model.ComicStageCharRef)
}

func (s *ComicDramaService) GetSegments(ctx context.Context, dramaID, userID uint) ([]*model.Storyboard, error) {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return nil, fmt.Errorf("comic drama not found")
	}
	return s.comicDramaDAO.ListStoryboardsByDrama(ctx, dramaID)
}

func (s *ComicDramaService) TriggerCompose(ctx context.Context, dramaID, userID uint) error {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	return s.orchestrator.SubmitComposeTask(ctx, drama)
}

func (s *ComicDramaService) GetDownloadURL(ctx context.Context, dramaID, userID uint) (string, error) {
	drama, err := s.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return "", fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return "", fmt.Errorf("comic drama not found")
	}
	if drama.Stage != model.ComicStageDone {
		return "", fmt.Errorf("comic drama not yet completed")
	}
	// 返回合成视频的存储路径
	return fmt.Sprintf("/uploads/comic-drama/%d/output/final.mp4", dramaID), nil
}

// --- PipelineOrchestrator ---

// PipelineOrchestrator 漫剧流水线编排器
type PipelineOrchestrator struct {
	comicDramaDAO *dao.ComicDramaDAO
	aiTaskDAO     *dao.AITaskDAO
	dispatcher    PipelineDispatcher
	notifier      PipelineNotifier
}

// PipelineDispatcher 任务分发接口（解耦 agent.Dispatcher）
type PipelineDispatcher interface {
	Dispatch(ctx context.Context, task *model.AITask) error
}

// PipelineNotifier 通知接口
type PipelineNotifier interface {
	NotifyUserWithType(userID uint, msgType string, message interface{}) error
}

func NewPipelineOrchestrator(cdDAO *dao.ComicDramaDAO, aiTaskDAO *dao.AITaskDAO, dispatcher PipelineDispatcher, notifier PipelineNotifier) *PipelineOrchestrator {
	return &PipelineOrchestrator{
		comicDramaDAO: cdDAO,
		aiTaskDAO:     aiTaskDAO,
		dispatcher:    dispatcher,
		notifier:      notifier,
	}
}

func (o *PipelineOrchestrator) StartPipeline(ctx context.Context, userID, dramaID uint) error {
	drama, err := o.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	if drama.Status != model.ComicStatusPending && drama.Status != model.ComicStatusFailed {
		return fmt.Errorf("comic drama is already in progress or completed")
	}

	if err := o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, model.ComicStageScript, 0, model.ComicStatusRunning); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	task := &model.AITask{
		UserID:     userID,
		NovelID:    drama.NovelID,
		TaskType:   model.TaskTypeComicScript,
		Status:     model.TaskStatusPending,
		Prompt:     o.buildScriptPrompt(drama),
		PipelineID: dramaID,
		Stage:      model.ComicStageScript,
		StageIndex: 0,
	}
	if err := o.dispatcher.Dispatch(ctx, task); err != nil {
		_ = o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, model.ComicStageScript, 0, model.ComicStatusFailed)
		return fmt.Errorf("failed to dispatch script task: %w", err)
	}
	return nil
}

func (o *PipelineOrchestrator) AdvanceStage(ctx context.Context, dramaID uint, currentStage string) error {
	nextStage := o.getNextStage(currentStage)
	if nextStage == "" {
		return o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, model.ComicStageDone, 0, model.ComicStatusCompleted)
	}
	if err := o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, nextStage, 0, model.ComicStatusRunning); err != nil {
		return fmt.Errorf("failed to advance stage: %w", err)
	}
	return o.createStageTasks(ctx, dramaID, nextStage)
}

func (o *PipelineOrchestrator) RetryFailed(ctx context.Context, userID, dramaID uint) error {
	drama, err := o.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}
	if drama.UserID != userID {
		return fmt.Errorf("comic drama not found")
	}
	if drama.Status != model.ComicStatusFailed {
		return fmt.Errorf("comic drama is not in failed state")
	}
	if err := o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, drama.Stage, 0, model.ComicStatusRunning); err != nil {
		return err
	}
	return o.createStageTasks(ctx, dramaID, drama.Stage)
}

// OnTaskDone 任务完成回调：检查同 pipeline 同 stage 的所有任务是否完成
func (o *PipelineOrchestrator) OnTaskDone(ctx context.Context, task *model.AITask) error {
	if task.PipelineID == 0 {
		return nil
	}
	dramaID := task.PipelineID
	stage := task.Stage

	// 检查同 pipeline 同 stage 是否还有未完成的任务
	pendingCount, err := o.aiTaskDAO.CountPipelineTasksByStatus(ctx, dramaID, stage, []string{model.TaskStatusPending, model.TaskStatusRunning})
	if err != nil {
		return fmt.Errorf("failed to check stage tasks: %w", err)
	}
	if pendingCount > 0 {
		return nil
	}

	// 通知前端进度更新
	drama, _ := o.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if drama != nil && o.notifier != nil {
		_ = o.notifier.NotifyUserWithType(drama.UserID, "comic_drama_stage_done", map[string]interface{}{
			"drama_id": dramaID,
			"stage":    stage,
		})
	}

	// 审核阶段等待人工确认
	if o.isReviewStage(stage) {
		return o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, stage, 0, model.ComicStatusPaused)
	}

	// 非审核阶段自动推进
	return o.AdvanceStage(ctx, dramaID, stage)
}

// SubmitCharRefTask 提交单个角色定妆照任务
func (o *PipelineOrchestrator) SubmitCharRefTask(ctx context.Context, drama *model.ComicDrama, ref *model.CharacterRef) error {
	task := &model.AITask{
		UserID:     drama.UserID,
		NovelID:    drama.NovelID,
		TaskType:   model.TaskTypeComicCharRef,
		Status:     model.TaskStatusPending,
		Prompt:     fmt.Sprintf(`{"character_id":%d,"name":"%s","style_prompt":"%s"}`, ref.ID, ref.Name, ref.StylePrompt),
		PipelineID: drama.ID,
		Stage:      model.ComicStageCharRef,
		StageIndex: int(ref.ID),
	}
	return o.dispatcher.Dispatch(ctx, task)
}

// SubmitComposeTask 提交合成任务
func (o *PipelineOrchestrator) SubmitComposeTask(ctx context.Context, drama *model.ComicDrama) error {
	task := &model.AITask{
		UserID:     drama.UserID,
		NovelID:    drama.NovelID,
		TaskType:   model.TaskTypeComicCompose,
		Status:     model.TaskStatusPending,
		Prompt:     fmt.Sprintf(`{"comic_drama_id":%d}`, drama.ID),
		PipelineID: drama.ID,
		Stage:      model.ComicStageCompose,
		StageIndex: 0,
	}
	return o.dispatcher.Dispatch(ctx, task)
}

func (o *PipelineOrchestrator) getNextStage(current string) string {
	stageOrder := []string{
		model.ComicStageScript,
		model.ComicStageStoryboard,
		model.ComicStageCharRef,
		model.ComicStageAudio,
		model.ComicStageMedia,
		model.ComicStageCompose,
	}
	for i, s := range stageOrder {
		if s == current && i+1 < len(stageOrder) {
			return stageOrder[i+1]
		}
	}
	return ""
}

func (o *PipelineOrchestrator) isReviewStage(stage string) bool {
	return stage == model.ComicStageScript || stage == model.ComicStageStoryboard || stage == model.ComicStageCharRef
}

func (o *PipelineOrchestrator) createStageTasks(ctx context.Context, dramaID uint, stage string) error {
	drama, err := o.comicDramaDAO.GetComicDrama(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("failed to get comic drama: %w", err)
	}

	switch stage {
	case model.ComicStageScript:
		return o.dispatchSingle(ctx, drama, model.TaskTypeComicScript, o.buildScriptPrompt(drama), stage, 0)
	case model.ComicStageStoryboard:
		return o.dispatchSingle(ctx, drama, model.TaskTypeComicStoryboard, o.buildStoryboardPrompt(drama), stage, 0)
	case model.ComicStageCharRef:
		return o.dispatchCharRefTasks(ctx, drama)
	case model.ComicStageAudio:
		return o.dispatchAudioTasks(ctx, drama)
	case model.ComicStageMedia:
		return o.dispatchMediaTasks(ctx, drama)
	case model.ComicStageCompose:
		return o.dispatchSingle(ctx, drama, model.TaskTypeComicCompose, o.buildComposePrompt(drama), stage, 0)
	default:
		return fmt.Errorf("unknown stage: %s", stage)
	}
}

func (o *PipelineOrchestrator) dispatchSingle(ctx context.Context, drama *model.ComicDrama, taskType, prompt, stage string, index int) error {
	task := &model.AITask{
		UserID:     drama.UserID,
		NovelID:    drama.NovelID,
		TaskType:   taskType,
		Status:     model.TaskStatusPending,
		Prompt:     prompt,
		PipelineID: drama.ID,
		Stage:      stage,
		StageIndex: index,
	}
	return o.dispatcher.Dispatch(ctx, task)
}

func (o *PipelineOrchestrator) dispatchCharRefTasks(ctx context.Context, drama *model.ComicDrama) error {
	refs, err := o.comicDramaDAO.ListCharacterRefsByDrama(ctx, drama.ID)
	if err != nil {
		return fmt.Errorf("failed to list character refs: %w", err)
	}
	if len(refs) == 0 {
		// 没有角色需要生成定妆照，直接推进
		return o.AdvanceStage(ctx, drama.ID, model.ComicStageCharRef)
	}
	for i, ref := range refs {
		prompt := fmt.Sprintf(`{"character_id":%d,"name":"%s","style_prompt":"%s"}`, ref.ID, ref.Name, ref.StylePrompt)
		if err := o.dispatchSingle(ctx, drama, model.TaskTypeComicCharRef, prompt, model.ComicStageCharRef, i); err != nil {
			return err
		}
	}
	return nil
}

func (o *PipelineOrchestrator) dispatchAudioTasks(ctx context.Context, drama *model.ComicDrama) error {
	boards, err := o.comicDramaDAO.ListStoryboardsByDrama(ctx, drama.ID)
	if err != nil {
		return fmt.Errorf("failed to list storyboards: %w", err)
	}
	dispatched := 0
	for i, sb := range boards {
		if sb.AudioURL != "" {
			continue
		}
		prompt := fmt.Sprintf(`{"storyboard_id":%d,"text":"%s","voice_id":"default","speed":1.0}`, sb.ID, sb.FrameDesc)
		if err := o.dispatchSingle(ctx, drama, model.TaskTypeComicAudio, prompt, model.ComicStageAudio, i); err != nil {
			return err
		}
		dispatched++
	}
	if dispatched == 0 {
		return o.AdvanceStage(ctx, drama.ID, model.ComicStageAudio)
	}
	return nil
}

func (o *PipelineOrchestrator) dispatchMediaTasks(ctx context.Context, drama *model.ComicDrama) error {
	boards, err := o.comicDramaDAO.ListStoryboardsByDrama(ctx, drama.ID)
	if err != nil {
		return fmt.Errorf("failed to list storyboards: %w", err)
	}
	dispatched := 0
	for i, sb := range boards {
		if sb.MediaURL != "" {
			continue
		}
		prompt := fmt.Sprintf(`{"storyboard_id":%d,"frame_desc":"%s","media_type":"video","importance":"medium","duration":4.0}`, sb.ID, sb.FrameDesc)
		if err := o.dispatchSingle(ctx, drama, model.TaskTypeComicMedia, prompt, model.ComicStageMedia, i); err != nil {
			return err
		}
		dispatched++
	}
	if dispatched == 0 {
		return o.AdvanceStage(ctx, drama.ID, model.ComicStageMedia)
	}
	return nil
}

func (o *PipelineOrchestrator) buildScriptPrompt(drama *model.ComicDrama) string {
	return fmt.Sprintf(`{"novel_id":%d,"chapter_id":%d,"config":%s}`, drama.NovelID, drama.ChapterID, drama.Config)
}

func (o *PipelineOrchestrator) buildStoryboardPrompt(drama *model.ComicDrama) string {
	return fmt.Sprintf(`{"comic_drama_id":%d}`, drama.ID)
}

func (o *PipelineOrchestrator) buildComposePrompt(drama *model.ComicDrama) string {
	return fmt.Sprintf(`{"comic_drama_id":%d}`, drama.ID)
}
