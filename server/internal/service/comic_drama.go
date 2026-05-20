// server/internal/service/comic_drama.go
package service

import (
	"context"
	"encoding/json"
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
	novelDAO      *dao.NovelDAO
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

func NewPipelineOrchestrator(cdDAO *dao.ComicDramaDAO, aiTaskDAO *dao.AITaskDAO, novelDAO *dao.NovelDAO, dispatcher PipelineDispatcher, notifier PipelineNotifier) *PipelineOrchestrator {
	return &PipelineOrchestrator{
		comicDramaDAO: cdDAO,
		aiTaskDAO:     aiTaskDAO,
		novelDAO:      novelDAO,
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

	// 按章节分发剧本生成任务
	chapters, err := o.novelDAO.ListChaptersByNovel(drama.NovelID)
	if err != nil || len(chapters) == 0 {
		return fmt.Errorf("failed to load novel chapters or novel is empty")
	}

	if err := o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, model.ComicStageScript, 0, model.ComicStatusRunning); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	for i, ch := range chapters {
		if ch.Content == "" {
			continue
		}
		prompt := o.buildChapterScriptPrompt(ch.Content, i, len(chapters))
		task := &model.AITask{
			UserID:     userID,
			NovelID:    drama.NovelID,
			TaskType:   model.TaskTypeComicScript,
			Status:     model.TaskStatusPending,
			Prompt:     prompt,
			ModelName:  "qwen",
			PipelineID: dramaID,
			Stage:      model.ComicStageScript,
			StageIndex: i,
		}
		if err := o.dispatcher.Dispatch(ctx, task); err != nil {
			_ = o.comicDramaDAO.UpdateComicDramaStage(ctx, dramaID, model.ComicStageScript, 0, model.ComicStatusFailed)
			return fmt.Errorf("failed to dispatch script task for chapter %d: %w", i, err)
		}
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

// OnTaskDone 任务完成回调：持久化结果 → 检查阶段完成 → 通知/推进
func (o *PipelineOrchestrator) OnTaskDone(ctx context.Context, task *model.AITask) error {
	if task.PipelineID == 0 {
		return nil
	}
	dramaID := task.PipelineID
	stage := task.Stage

	// 将 executor 结果持久化到业务表
	if err := o.persistTaskResult(ctx, task); err != nil {
		return fmt.Errorf("failed to persist task result: %w", err)
	}

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

// persistTaskResult 将 executor 结果解析并写入业务表
func (o *PipelineOrchestrator) persistTaskResult(ctx context.Context, task *model.AITask) error {
	if task.Result == "" {
		return nil
	}
	switch task.TaskType {
	case model.TaskTypeComicScript:
		return o.persistScriptResult(ctx, task)
	case model.TaskTypeComicStoryboard:
		return o.persistStoryboardResult(ctx, task)
	case model.TaskTypeComicCharRef:
		return o.persistCharRefResult(ctx, task)
	case model.TaskTypeComicAudio:
		return o.persistAudioResult(ctx, task)
	case model.TaskTypeComicMedia:
		return o.persistMediaResult(ctx, task)
	}
	return nil
}

func (o *PipelineOrchestrator) persistScriptResult(ctx context.Context, task *model.AITask) error {
	var result struct {
		Scenes []struct {
			SeqNo     int         `json:"seq_no"`
			SceneDesc string      `json:"scene_desc"`
			Dialogue  interface{} `json:"dialogue"`
			Emotion   string      `json:"emotion"`
			MediaType string      `json:"media_type"`
			Duration  float64     `json:"duration"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(task.Result), &result); err != nil {
		return fmt.Errorf("parse script result: %w", err)
	}
	var scripts []*model.ComicScript
	seqBase := task.StageIndex * 100
	for i, s := range result.Scenes {
		dialogueJSON, _ := json.Marshal(s.Dialogue)
		scripts = append(scripts, &model.ComicScript{
			ComicDramaID: task.PipelineID,
			SeqNo:        seqBase + i,
			SceneDesc:    s.SceneDesc,
			Dialogue:     string(dialogueJSON),
			Emotion:      s.Emotion,
			MediaType:    s.MediaType,
			Duration:     s.Duration,
		})
	}
	if len(scripts) == 0 {
		return nil
	}
	return o.comicDramaDAO.BatchCreateScripts(ctx, scripts)
}

func (o *PipelineOrchestrator) persistStoryboardResult(ctx context.Context, task *model.AITask) error {
	var result struct {
		Frames []struct {
			SceneSeqNo  int      `json:"scene_seq_no"`
			FrameSeqNo  int      `json:"frame_seq_no"`
			FrameDesc   string   `json:"frame_desc"`
			CameraAngle string   `json:"camera_angle"`
			Characters  []string `json:"characters"`
			MediaType   string   `json:"media_type"`
			Duration    float64  `json:"duration"`
		} `json:"frames"`
	}
	if err := json.Unmarshal([]byte(task.Result), &result); err != nil {
		return fmt.Errorf("parse storyboard result: %w", err)
	}
	var boards []*model.Storyboard
	charSet := make(map[string]bool)
	for i, f := range result.Frames {
		charsJSON, _ := json.Marshal(f.Characters)
		boards = append(boards, &model.Storyboard{
			ComicDramaID: task.PipelineID,
			SeqNo:        i,
			FrameDesc:    f.FrameDesc,
			CameraAngle:  f.CameraAngle,
			Characters:   string(charsJSON),
			Status:       "pending",
		})
		for _, ch := range f.Characters {
			charSet[ch] = true
		}
	}
	if len(boards) == 0 {
		return nil
	}
	if err := o.comicDramaDAO.BatchCreateStoryboards(ctx, boards); err != nil {
		return err
	}
	// 从分镜中提取角色，创建 CharacterRef 记录（供 char_ref 阶段使用）
	if len(charSet) > 0 {
		var refs []*model.CharacterRef
		for name := range charSet {
			refs = append(refs, &model.CharacterRef{
				ComicDramaID: task.PipelineID,
				Name:         name,
				Status:       "pending",
			})
		}
		_ = o.comicDramaDAO.BatchCreateCharacterRefs(ctx, refs)
	}
	return nil
}

func (o *PipelineOrchestrator) persistCharRefResult(ctx context.Context, task *model.AITask) error {
	// executor 返回 {"character_refs": [{"name":"...", "image_url":"...", "file_path":"..."}]}
	var result struct {
		CharacterRefs []struct {
			Name     string `json:"name"`
			ImageURL string `json:"image_url"`
		} `json:"character_refs"`
	}
	if err := json.Unmarshal([]byte(task.Result), &result); err != nil {
		return nil
	}
	// 从 prompt 中提取 character_ref_id
	var promptData struct {
		CharacterRefID uint `json:"character_ref_id"`
	}
	_ = json.Unmarshal([]byte(task.Prompt), &promptData)
	if promptData.CharacterRefID == 0 || len(result.CharacterRefs) == 0 {
		return nil
	}
	ref, err := o.comicDramaDAO.GetCharacterRef(ctx, promptData.CharacterRefID)
	if err != nil {
		return nil
	}
	ref.RefImageURL = result.CharacterRefs[0].ImageURL
	ref.Status = "completed"
	return o.comicDramaDAO.UpdateCharacterRef(ctx, ref)
}

func (o *PipelineOrchestrator) persistAudioResult(ctx context.Context, task *model.AITask) error {
	var result struct {
		StoryboardID uint   `json:"storyboard_id"`
		AudioURL     string `json:"audio_url"`
	}
	if err := json.Unmarshal([]byte(task.Result), &result); err != nil {
		return nil
	}
	if result.StoryboardID == 0 || result.AudioURL == "" {
		return nil
	}
	board, err := o.comicDramaDAO.GetStoryboard(ctx, result.StoryboardID)
	if err != nil {
		return nil
	}
	board.AudioURL = result.AudioURL
	return o.comicDramaDAO.UpdateStoryboard(ctx, board)
}

func (o *PipelineOrchestrator) persistMediaResult(ctx context.Context, task *model.AITask) error {
	var result struct {
		StoryboardID uint   `json:"storyboard_id"`
		MediaURL     string `json:"media_url"`
	}
	if err := json.Unmarshal([]byte(task.Result), &result); err != nil {
		return nil
	}
	if result.StoryboardID == 0 || result.MediaURL == "" {
		return nil
	}
	board, err := o.comicDramaDAO.GetStoryboard(ctx, result.StoryboardID)
	if err != nil {
		return nil
	}
	board.MediaURL = result.MediaURL
	board.Status = "completed"
	return o.comicDramaDAO.UpdateStoryboard(ctx, board)
}
func (o *PipelineOrchestrator) SubmitCharRefTask(ctx context.Context, drama *model.ComicDrama, ref *model.CharacterRef) error {
	input := map[string]interface{}{
		"characters": []map[string]string{
			{"name": ref.Name, "appearance": ref.StylePrompt, "style_prompt": ref.StylePrompt},
		},
		"character_ref_id": ref.ID,
	}
	promptJSON, _ := json.Marshal(input)
	task := &model.AITask{
		UserID:     drama.UserID,
		NovelID:    drama.NovelID,
		TaskType:   model.TaskTypeComicCharRef,
		Status:     model.TaskStatusPending,
		Prompt:     string(promptJSON),
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
		return o.dispatchScriptTasks(ctx, drama)
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

func (o *PipelineOrchestrator) dispatchScriptTasks(ctx context.Context, drama *model.ComicDrama) error {
	chapters, err := o.novelDAO.ListChaptersByNovel(drama.NovelID)
	if err != nil || len(chapters) == 0 {
		return fmt.Errorf("no chapters found for novel %d", drama.NovelID)
	}
	dispatched := 0
	for i, ch := range chapters {
		if ch.Content == "" {
			continue
		}
		prompt := o.buildChapterScriptPrompt(ch.Content, i, len(chapters))
		if err := o.dispatchSingle(ctx, drama, model.TaskTypeComicScript, prompt, model.ComicStageScript, i); err != nil {
			return err
		}
		dispatched++
	}
	if dispatched == 0 {
		return fmt.Errorf("all chapters are empty, nothing to generate")
	}
	return nil
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
	// 文本生成类任务需要指定模型，走 AIProvider 路由
	if taskType == model.TaskTypeComicScript || taskType == model.TaskTypeComicStoryboard {
		task.ModelName = "qwen"
	}
	return o.dispatcher.Dispatch(ctx, task)
}

func (o *PipelineOrchestrator) dispatchCharRefTasks(ctx context.Context, drama *model.ComicDrama) error {
	refs, err := o.comicDramaDAO.ListCharacterRefsByDrama(ctx, drama.ID)
	if err != nil {
		return fmt.Errorf("failed to list character refs: %w", err)
	}
	if len(refs) == 0 {
		return o.AdvanceStage(ctx, drama.ID, model.ComicStageCharRef)
	}
	for i, ref := range refs {
		input := map[string]interface{}{
			"characters": []map[string]string{
				{"name": ref.Name, "appearance": ref.StylePrompt, "style_prompt": ref.StylePrompt},
			},
			"character_ref_id": ref.ID,
		}
		promptJSON, _ := json.Marshal(input)
		if err := o.dispatchSingle(ctx, drama, model.TaskTypeComicCharRef, string(promptJSON), model.ComicStageCharRef, i); err != nil {
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

func (o *PipelineOrchestrator) buildChapterScriptPrompt(chapterContent string, chapterIndex, totalChapters int) string {
	input := map[string]interface{}{
		"chapter_content": chapterContent,
		"chapter_index":   chapterIndex,
		"total_chapters":  totalChapters,
		"max_scenes":      8,
	}
	data, _ := json.Marshal(input)
	return string(data)
}

func (o *PipelineOrchestrator) buildStoryboardPrompt(drama *model.ComicDrama) string {
	// 从数据库读取已保存的剧本 scenes，传给分镜 executor
	scripts, err := o.comicDramaDAO.ListScriptsByDrama(context.Background(), drama.ID)
	if err != nil || len(scripts) == 0 {
		return `{"scenes":[]}`
	}
	type scene struct {
		SeqNo     int         `json:"seq_no"`
		SceneDesc string      `json:"scene_desc"`
		Dialogue  interface{} `json:"dialogue"`
		Emotion   string      `json:"emotion"`
		MediaType string      `json:"media_type"`
		Duration  float64     `json:"duration"`
	}
	var scenes []scene
	for _, s := range scripts {
		var dialogue interface{}
		_ = json.Unmarshal([]byte(s.Dialogue), &dialogue)
		scenes = append(scenes, scene{
			SeqNo:     s.SeqNo,
			SceneDesc: s.SceneDesc,
			Dialogue:  dialogue,
			Emotion:   s.Emotion,
			MediaType: s.MediaType,
			Duration:  s.Duration,
		})
	}
	data, _ := json.Marshal(map[string]interface{}{"scenes": scenes})
	return string(data)
}

func (o *PipelineOrchestrator) buildComposePrompt(drama *model.ComicDrama) string {
	return fmt.Sprintf(`{"comic_drama_id":%d}`, drama.ID)
}
