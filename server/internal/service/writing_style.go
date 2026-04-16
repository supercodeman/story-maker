// server/internal/service/writing_style.go
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// WritingStyleService 写作风格业务逻辑层
type WritingStyleService struct {
	dao          *dao.WritingStyleDAO
	memoryDAO    *dao.WritingMemoryDAO
	prefDAO      *dao.UserPreferenceDAO
	userStyleDAO *dao.UserStyleDAO
}

// NewWritingStyleService 创建 WritingStyleService 实例
func NewWritingStyleService() *WritingStyleService {
	return &WritingStyleService{
		dao:          dao.NewWritingStyleDAO(),
		memoryDAO:    dao.NewWritingMemoryDAO(),
		prefDAO:      dao.NewUserPreferenceDAO(),
		userStyleDAO: dao.NewUserStyleDAO(),
	}
}

// ========== WritingStyle CRUD ==========

// GetByNovelID 获取小说的写作风格配置
func (s *WritingStyleService) GetByNovelID(novelID uint) (*model.WritingStyle, error) {
	return s.dao.GetByNovelID(novelID)
}

// Upsert 创建或更新写作风格
func (s *WritingStyleService) Upsert(style *model.WritingStyle) error {
	return s.dao.Upsert(style)
}

// Delete 删除写作风格配置
func (s *WritingStyleService) Delete(novelID uint) error {
	return s.dao.Delete(novelID)
}

// ========== ScenePreset CRUD ==========

// ListPresets 获取小说下的所有场景预设
func (s *WritingStyleService) ListPresets(novelID uint) ([]model.ScenePreset, error) {
	return s.dao.ListPresets(novelID)
}

// GetPreset 获取场景预设
func (s *WritingStyleService) GetPreset(id uint) (*model.ScenePreset, error) {
	return s.dao.GetPreset(id)
}

// CreatePreset 创建场景预设
func (s *WritingStyleService) CreatePreset(preset *model.ScenePreset) error {
	return s.dao.CreatePreset(preset)
}

// UpdatePreset 更新场景预设
func (s *WritingStyleService) UpdatePreset(preset *model.ScenePreset) error {
	return s.dao.UpdatePreset(preset)
}

// DeletePreset 删除场景预设
func (s *WritingStyleService) DeletePreset(id uint) error {
	return s.dao.DeletePreset(id)
}

// ========== Prompt 格式化 ==========

// FormatStyleForPrompt 将写作风格 + 可选场景预设格式化为 prompt 文本
func (s *WritingStyleService) FormatStyleForPrompt(novelID uint, scenePresetID *uint) string {
	style, err := s.dao.GetByNovelID(novelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ""
		}
		log.Printf("[WritingStyle] failed to query style for novel %d: %v", novelID, err)
		return ""
	}

	// 如果绑定了用户风格，优先使用 UserStyle 的配置
	if style.BoundUserStyleID != nil && *style.BoundUserStyleID > 0 {
		userStyle, usErr := s.userStyleDAO.GetByID(*style.BoundUserStyleID)
		if usErr == nil && userStyle != nil {
			// 用 UserStyle 的字段覆盖
			style.NarrativeVoice = userStyle.NarrativeVoice
			style.Tone = userStyle.Tone
			style.LanguageLevel = userStyle.LanguageLevel
			style.ReferenceAuthors = userStyle.ReferenceAuthors
			style.ForbiddenPatterns = userStyle.ForbiddenPatterns
			style.CustomRules = userStyle.CustomRules
		}
	}

	var parts []string

	// 叙事视角
	if label, ok := model.ValidNarrativeVoices[style.NarrativeVoice]; ok {
		parts = append(parts, fmt.Sprintf("- 叙事视角：%s", label))
	}

	// 文风调性
	if label, ok := model.ValidTones[style.Tone]; ok {
		parts = append(parts, fmt.Sprintf("- 文风调性：%s", label))
	}

	// 语言水平
	if label, ok := model.ValidLanguageLevels[style.LanguageLevel]; ok {
		parts = append(parts, fmt.Sprintf("- 语言风格：%s", label))
	}

	// 参考作家
	if style.ReferenceAuthors != "" {
		parts = append(parts, fmt.Sprintf("- 参考作家风格：%s", style.ReferenceAuthors))
	}

	// 禁用句式
	if style.ForbiddenPatterns != "" {
		parts = append(parts, fmt.Sprintf("- 禁止使用的句式/表达：%s", style.ForbiddenPatterns))
	}

	// 自定义规范
	if style.CustomRules != "" {
		parts = append(parts, fmt.Sprintf("- 额外写作规范：%s", style.CustomRules))
	}

	result := strings.Join(parts, "\n")

	// 注入小说绑定的写作记忆
	bindings, err := s.memoryDAO.ListBindingsByNovel(novelID)
	if err == nil && len(bindings) > 0 {
		for _, b := range bindings {
			memory, mErr := s.memoryDAO.Get(b.MemoryID)
			if mErr != nil || memory == nil || memory.PromptTpl == "" {
				continue
			}
			label := memory.Category
			if catLabel, ok := model.ValidMemoryCategories[memory.Category]; ok {
				label = catLabel
			}

			// 尝试解析 style 类记忆的子维度特征
			if memory.Category == model.MemoryCategoryStyle {
				var styleFeatures model.StyleFeatures
				if json.Unmarshal([]byte(memory.Features), &styleFeatures) == nil && styleFeatures.Tone.Description != "" {
					result += fmt.Sprintf("\n\n【%s记忆·%s】", label, memory.Title)
					if styleFeatures.Tone.PromptPart != "" {
						result += fmt.Sprintf("\n[文风] %s", styleFeatures.Tone.PromptPart)
					}
					if styleFeatures.Rhythm.PromptPart != "" {
						result += fmt.Sprintf("\n[句式] %s", styleFeatures.Rhythm.PromptPart)
					}
					if styleFeatures.Vocabulary.PromptPart != "" {
						result += fmt.Sprintf("\n[语感] %s", styleFeatures.Vocabulary.PromptPart)
					}
					if styleFeatures.DialogueStyle.PromptPart != "" {
						result += fmt.Sprintf("\n[对话] %s", styleFeatures.DialogueStyle.PromptPart)
					}
					if len(styleFeatures.ForbiddenPatterns) > 0 {
						result += fmt.Sprintf("\n[禁用] %s", strings.Join(styleFeatures.ForbiddenPatterns, "、"))
					}
					if styleFeatures.ReferenceStyle != "" {
						result += fmt.Sprintf("\n[参考] %s", styleFeatures.ReferenceStyle)
					}
				} else {
					// 旧格式回退：直接注入 PromptTpl
					result += fmt.Sprintf("\n\n【%s记忆·%s】\n%s", label, memory.Title, memory.PromptTpl)
				}
			} else {
				result += fmt.Sprintf("\n\n【%s记忆·%s】\n%s", label, memory.Title, memory.PromptTpl)
			}

			// 注入锚定句作为 few-shot
			if memory.AnchorTexts != "" {
				var anchors []string
				if json.Unmarshal([]byte(memory.AnchorTexts), &anchors) == nil && len(anchors) > 0 {
					result += "\n\n【风格参考句】\n"
					for i, a := range anchors {
						result += fmt.Sprintf("%d. %s\n", i+1, a)
					}
				}
			}
		}
	}

	// 追加场景预设规范（校验 preset 归属当前小说）
	if scenePresetID != nil {
		preset, err := s.dao.GetPreset(*scenePresetID)
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				log.Printf("[WritingStyle] failed to query preset %d: %v", *scenePresetID, err)
			}
		} else if preset.NovelID != novelID {
			log.Printf("[WritingStyle] preset %d belongs to novel %d, not %d — skipped", *scenePresetID, preset.NovelID, novelID)
		} else {
			sceneName := preset.Name
			if label, ok := model.ValidSceneTypes[preset.SceneType]; ok {
				sceneName = label + "·" + preset.Name
			}
			result += fmt.Sprintf("\n\n【当前场景：%s】\n%s", sceneName, preset.Rules)
		}
	}

	return result
}

// FormatStyleWithUserPref 在 FormatStyleForPrompt 基础上追加用户偏好摘要
func (s *WritingStyleService) FormatStyleWithUserPref(novelID uint, scenePresetID *uint, userID uint) string {
	result := s.FormatStyleForPrompt(novelID, scenePresetID)

	// 注入用户偏好摘要
	if s.prefDAO != nil && userID > 0 {
		pref, err := s.prefDAO.GetByUserNovel(userID, novelID)
		if err == nil && pref != nil && pref.PromptSummary != "" {
			result += fmt.Sprintf("\n\n【用户写作偏好】\n%s", pref.PromptSummary)
		}
	}

	return result
}

