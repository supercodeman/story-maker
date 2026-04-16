// server/internal/service/world_building_accept.go
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"ai-curton/server/internal/model"
)

// ========== JSON 提取工具 ==========

// extractJSONArray 从 LLM 返回文本中提取 JSON 数组（处理 markdown code block 包裹）
func extractJSONArray(raw string) string {
	raw = strings.TrimSpace(raw)
	// 移除 markdown code block
	if idx := strings.Index(raw, "```json"); idx != -1 {
		raw = raw[idx+7:]
		if end := strings.Index(raw, "```"); end != -1 {
			raw = raw[:end]
		}
	} else if idx := strings.Index(raw, "```"); idx != -1 {
		raw = raw[idx+3:]
		if end := strings.Index(raw, "```"); end != -1 {
			raw = raw[:end]
		}
	}
	raw = strings.TrimSpace(raw)
	// 兜底：找第一个 [ 和最后一个 ]
	if start := strings.Index(raw, "["); start >= 0 {
		if end := strings.LastIndex(raw, "]"); end > start {
			return raw[start : end+1]
		}
	}
	return raw
}

// extractJSONObject 从 LLM 返回文本中提取 JSON 对象（处理 markdown code block 包裹）
func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	// 移除 markdown code block
	if idx := strings.Index(raw, "```json"); idx != -1 {
		raw = raw[idx+7:]
		if end := strings.Index(raw, "```"); end != -1 {
			raw = raw[:end]
		}
	} else if idx := strings.Index(raw, "```"); idx != -1 {
		raw = raw[idx+3:]
		if end := strings.Index(raw, "```"); end != -1 {
			raw = raw[:end]
		}
	}
	raw = strings.TrimSpace(raw)
	// 兜底：找第一个 { 和最后一个 }
	if start := strings.Index(raw, "{"); start >= 0 {
		if end := strings.LastIndex(raw, "}"); end > start {
			return raw[start : end+1]
		}
	}
	return raw
}

// ========== 数据写入（WorldBuildingService 的私有方法） ==========

// acceptWorldview 解析世界观 JSON 并批量写入
func (s *WorldBuildingService) acceptWorldview(userID, novelID uint, content string, score float64) error {
	var items []struct {
		Category string `json:"category"`
		Title    string `json:"title"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(extractJSONArray(content)), &items); err != nil {
		return fmt.Errorf("parse worldview content failed: %w", err)
	}

	// 先清除旧数据再写入
	if err := s.dao.DeleteWorldSettingsByNovel(novelID); err != nil {
		return fmt.Errorf("delete old world settings failed: %w", err)
	}

	settings := make([]model.NovelWorldSetting, 0, len(items))
	facts := make([]*model.NovelMemoryFact, 0, len(items))
	for _, item := range items {
		settings = append(settings, model.NovelWorldSetting{
			NovelID:  novelID,
			UserID:   userID,
			Category: item.Category,
			Title:    item.Title,
			Content:  item.Content,
			Score:    score,
		})
		// 同步写入 NovelMemoryFact 供章节生成使用
		facts = append(facts, &model.NovelMemoryFact{
			NovelID:  novelID,
			UserID:   userID,
			FactType: model.FactTypeWorldviewRule,
			Title:    item.Title,
			Content:  item.Content,
		})
	}

	if err := s.dao.BatchCreateWorldSettings(settings); err != nil {
		return fmt.Errorf("batch create world settings failed: %w", err)
	}
	if err := s.factDAO.BatchCreate(facts); err != nil {
		log.Printf("[world-building] batch create worldview facts failed: %v", err)
	}
	return nil
}

// acceptForeshadow 解析伏笔 JSON 并批量写入
func (s *WorldBuildingService) acceptForeshadow(userID, novelID uint, content string, score float64) error {
	var items []struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		PlantChapter  string `json:"plant_chapter"`
		RevealChapter string `json:"reveal_chapter"`
	}
	if err := json.Unmarshal([]byte(extractJSONArray(content)), &items); err != nil {
		return fmt.Errorf("parse foreshadow content failed: %w", err)
	}

	if err := s.dao.DeleteForeshadowsByNovel(novelID); err != nil {
		return fmt.Errorf("delete old foreshadows failed: %w", err)
	}

	list := make([]model.NovelForeshadow, 0, len(items))
	facts := make([]*model.NovelMemoryFact, 0, len(items))
	for _, item := range items {
		list = append(list, model.NovelForeshadow{
			NovelID:       novelID,
			UserID:        userID,
			Title:         item.Title,
			Description:   item.Description,
			PlantChapter:  item.PlantChapter,
			RevealChapter: item.RevealChapter,
			Status:        model.ForeshadowStatusPlanned,
			Score:         score,
		})
		facts = append(facts, &model.NovelMemoryFact{
			NovelID:  novelID,
			UserID:   userID,
			FactType: model.FactTypeForeshadow,
			Title:    item.Title,
			Content:  item.Description,
		})
	}

	if err := s.dao.BatchCreateForeshadows(list); err != nil {
		return fmt.Errorf("batch create foreshadows failed: %w", err)
	}
	if err := s.factDAO.BatchCreate(facts); err != nil {
		log.Printf("[world-building] batch create foreshadow facts failed: %v", err)
	}
	return nil
}

// acceptPlotOutline 解析剧情大纲 JSON 并批量写入
func (s *WorldBuildingService) acceptPlotOutline(userID, novelID uint, content string, score float64) error {
	var items []struct {
		Act       int    `json:"act"`
		SortOrder int    `json:"sort_order"`
		Title     string `json:"title"`
		Summary   string `json:"summary"`
		KeyEvents string `json:"key_events"`
	}
	if err := json.Unmarshal([]byte(extractJSONArray(content)), &items); err != nil {
		return fmt.Errorf("parse plot outline content failed: %w", err)
	}

	if err := s.dao.DeletePlotOutlinesByNovel(novelID); err != nil {
		return fmt.Errorf("delete old plot outlines failed: %w", err)
	}

	list := make([]model.NovelPlotOutline, 0, len(items))
	facts := make([]*model.NovelMemoryFact, 0, len(items))
	for _, item := range items {
		list = append(list, model.NovelPlotOutline{
			NovelID:   novelID,
			UserID:    userID,
			Act:       item.Act,
			SortOrder: item.SortOrder,
			Title:     item.Title,
			Summary:   item.Summary,
			KeyEvents: item.KeyEvents,
			Score:     score,
		})
		facts = append(facts, &model.NovelMemoryFact{
			NovelID:  novelID,
			UserID:   userID,
			FactType: model.FactTypePlotEvent,
			Title:    item.Title,
			Content:  item.Summary,
		})
	}

	if err := s.dao.BatchCreatePlotOutlines(list); err != nil {
		return fmt.Errorf("batch create plot outlines failed: %w", err)
	}
	if err := s.factDAO.BatchCreate(facts); err != nil {
		log.Printf("[world-building] batch create plot facts failed: %v", err)
	}
	return nil
}

// acceptCharacter 解析人物设定 JSON 并写入 NovelKnowledge + NovelMemoryFact
func (s *WorldBuildingService) acceptCharacter(userID, novelID uint, content string, score float64) error {
	var items []struct {
		Name       string `json:"name"`
		Content    string `json:"content"`
		Tags       string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(extractJSONArray(content)), &items); err != nil {
		return fmt.Errorf("parse character content failed: %w", err)
	}

	// 先清除旧的 character 类型知识条目
	if err := s.knowledgeDAO.DeleteByNovelAndCategory(novelID, model.KnowledgeCategoryCharacter); err != nil {
		return fmt.Errorf("delete old character knowledge failed: %w", err)
	}

	knowledgeItems := make([]model.NovelKnowledge, 0, len(items))
	facts := make([]*model.NovelMemoryFact, 0, len(items))
	for _, item := range items {
		knowledgeItems = append(knowledgeItems, model.NovelKnowledge{
			NovelID:  novelID,
			Category: model.KnowledgeCategoryCharacter,
			Title:    item.Name,
			Content:  item.Content,
			Tags:     item.Tags,
			Status:   model.KnowledgeStatusConfirmed,
		})
		facts = append(facts, &model.NovelMemoryFact{
			NovelID:  novelID,
			UserID:   userID,
			FactType: model.FactTypeCharacterTrait,
			Title:    item.Name,
			Content:  item.Content,
		})
	}

	if err := s.knowledgeDAO.BatchCreate(knowledgeItems); err != nil {
		return fmt.Errorf("batch create character knowledge failed: %w", err)
	}
	if err := s.factDAO.BatchCreate(facts); err != nil {
		log.Printf("[world-building] batch create character facts failed: %v", err)
	}
	return nil
}

// acceptRelation 解析人物关系 JSON 并写入 NovelCharacterRelation + NovelMemoryFact
func (s *WorldBuildingService) acceptRelation(userID, novelID uint, content string, score float64) error {
	var items []struct {
		FromName     string `json:"from_name"`
		ToName       string `json:"to_name"`
		RelationType string `json:"relation_type"`
		Label        string `json:"label"`
	}
	if err := json.Unmarshal([]byte(extractJSONArray(content)), &items); err != nil {
		return fmt.Errorf("parse relation content failed: %w", err)
	}

	// 先清除旧的关系数据
	if err := s.overviewDAO.DeleteRelationsByNovel(novelID); err != nil {
		return fmt.Errorf("delete old relations failed: %w", err)
	}

	// 构建角色名 → KnowledgeID 的映射
	characters, err := s.knowledgeDAO.ListByNovelAndCategory(novelID, model.KnowledgeCategoryCharacter)
	if err != nil {
		return fmt.Errorf("list character knowledge failed: %w", err)
	}
	nameToID := make(map[string]uint, len(characters))
	for _, c := range characters {
		nameToID[c.Title] = c.ID
	}

	facts := make([]*model.NovelMemoryFact, 0, len(items))
	for _, item := range items {
		fromID, fromOK := nameToID[item.FromName]
		toID, toOK := nameToID[item.ToName]
		if !fromOK || !toOK {
			log.Printf("[world-building] skip relation: from=%s(found=%v) to=%s(found=%v)", item.FromName, fromOK, item.ToName, toOK)
			continue
		}

		relType := item.RelationType
		if !model.ValidRelationTypes[relType] {
			relType = model.RelationTypeCustom
		}

		rel := &model.NovelCharacterRelation{
			NovelID:         novelID,
			FromKnowledgeID: fromID,
			ToKnowledgeID:   toID,
			RelationType:    relType,
			Label:           item.Label,
		}
		if err := s.overviewDAO.CreateRelation(rel); err != nil {
			log.Printf("[world-building] create relation failed: from=%s to=%s err=%v", item.FromName, item.ToName, err)
			continue
		}

		facts = append(facts, &model.NovelMemoryFact{
			NovelID:  novelID,
			UserID:   userID,
			FactType: model.FactTypeRelationshipChange,
			Title:    fmt.Sprintf("%s → %s", item.FromName, item.ToName),
			Content:  fmt.Sprintf("关系类型：%s，描述：%s", relType, item.Label),
		})
	}

	if err := s.factDAO.BatchCreate(facts); err != nil {
		log.Printf("[world-building] batch create relation facts failed: %v", err)
	}
	return nil
}
