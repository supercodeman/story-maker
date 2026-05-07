// server/internal/service/user_behavior.go
package service

import (
	"encoding/json"
	"log"
	"time"

	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// UserBehaviorService 用户行为采集服务
type UserBehaviorService struct {
	behaviorDAO *dao.UserBehaviorDAO
	prefSvc     *UserPreferenceService
}

// NewUserBehaviorService 创建 UserBehaviorService 实例
func NewUserBehaviorService(prefSvc *UserPreferenceService) *UserBehaviorService {
	return &UserBehaviorService{
		behaviorDAO: dao.NewUserBehaviorDAO(),
		prefSvc:     prefSvc,
	}
}

// RecordEvent 记录用户行为事件
// payload 会被序列化为 JSON 存储
func (s *UserBehaviorService) RecordEvent(userID, novelID, chapterID uint, eventType string, payload interface{}) error {
	if !model.ValidBehaviorTypes[eventType] {
		return nil // 忽略未知事件类型
	}

	var payloadStr string
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[UserBehavior] failed to marshal payload: %v", err)
			payloadStr = "{}"
		} else {
			payloadStr = string(data)
		}
	}

	event := &model.UserBehaviorEvent{
		UserID:    userID,
		NovelID:   novelID,
		ChapterID: chapterID,
		EventType: eventType,
		Payload:   payloadStr,
		CreatedAt: time.Now(),
	}

	if err := s.behaviorDAO.CreateEvent(event); err != nil {
		log.Printf("[UserBehavior] failed to record event: %v", err)
		return err
	}

	// 异步检查是否需要触发偏好提取（每累积 10 个新事件触发一次）
	go s.maybeExtractPreference(userID, novelID)

	return nil
}

// maybeExtractPreference 检查是否需要触发偏好提取
func (s *UserBehaviorService) maybeExtractPreference(userID, novelID uint) {
	if s.prefSvc == nil {
		return
	}

	// 统计最近事件数
	count, err := s.behaviorDAO.CountEventsByUserNovel(userID, novelID, time.Time{})
	if err != nil {
		return
	}

	// 获取当前偏好的已处理事件数
	pref, _ := s.prefSvc.prefDAO.GetByUserNovel(userID, novelID)
	processedCount := 0
	if pref != nil {
		processedCount = pref.EventCount
	}

	// 每累积 10 个新事件触发一次提取
	if int(count)-processedCount >= 10 {
		if err := s.prefSvc.ExtractPreference(userID, novelID); err != nil {
			log.Printf("[UserBehavior] failed to extract preference: %v", err)
		}
	}
}

// RecordBehaviorRequest 前端上报行为请求
type RecordBehaviorRequest struct {
	NovelID   uint        `json:"novel_id" binding:"required"`
	ChapterID uint        `json:"chapter_id"`
	EventType string      `json:"event_type" binding:"required"`
	Payload   interface{} `json:"payload"`
}

// PurgeOldEvents 清理 90 天前的事件
func (s *UserBehaviorService) PurgeOldEvents() error {
	before := time.Now().AddDate(0, 0, -90)
	return s.behaviorDAO.PurgeOldEvents(before)
}
