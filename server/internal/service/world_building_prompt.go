// server/internal/service/world_building_prompt.go
package service

import (
	"context"
	"fmt"
	"strings"

	"ai-curton/server/internal/model"
)

// ========== Prompt 构建（WorldBuildingService 的私有方法） ==========

// buildWriterSystemPrompt 构建 Writer 角色的 system prompt
func (s *WorldBuildingService) buildWriterSystemPrompt(phase string) string {
	base := "你是一位资深小说创作者，擅长构建完整、自洽的小说世界。请根据用户的需求，"
	switch phase {
	case model.ReflectionPhaseWorldview:
		return base + "生成详细的世界观设定，包括时代背景、地理环境、力量体系、社会规则和文化风俗。请以 JSON 数组格式返回，每个元素包含 category、title、content 字段。"
	case model.ReflectionPhaseCharacter:
		return base + "生成丰满的人物设定，包括外貌、性格、背景故事、动机和成长弧线。请以 JSON 数组格式返回，每个元素包含 name（角色名）、content（详细设定描述）、tags（标签，逗号分隔）字段。"
	case model.ReflectionPhaseRelation:
		return base + "生成人物关系网络，明确角色之间的关系类型、互动模式和冲突点。请以 JSON 数组格式返回，每个元素包含 from_name（角色A名称）、to_name（角色B名称）、relation_type（关系类型：ally/enemy/mentor/lover/family/rival/custom）、label（关系描述）字段。"
	case model.ReflectionPhaseForeshadow:
		return base + "设计精妙的伏笔体系，包括埋设时机、揭示节点和前后呼应逻辑。请以 JSON 数组格式返回，每个元素包含 title、description、plant_chapter、reveal_chapter 字段。"
	case model.ReflectionPhasePlot:
		return base + "规划完整的剧情大纲，按幕次组织，包含关键事件和转折点。请以 JSON 数组格式返回，每个元素包含 act、sort_order、title、summary、key_events 字段。"
	default:
		return base + "完成相关创作任务。"
	}
}

// buildWriterUserPrompt 构建 Writer 角色的 user prompt（包含已完成阶段上下文）
func (s *WorldBuildingService) buildWriterUserPrompt(ctx context.Context, phase string, novelID uint, userInput, prevContext string) string {
	var prompt strings.Builder

	// 加载已完成阶段的设定作为上下文
	if contextParts := s.gatherPhaseContext(novelID, phase); contextParts != "" {
		prompt.WriteString("【已有设定参考】\n")
		prompt.WriteString(contextParts)
		prompt.WriteString("\n\n")
	}

	if prevContext != "" {
		prompt.WriteString("【补充上下文】\n")
		prompt.WriteString(prevContext)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("【用户需求】\n")
	if userInput != "" {
		prompt.WriteString(userInput)
	} else {
		prompt.WriteString(s.defaultUserInputForPhase(phase))
	}
	return prompt.String()
}

// buildEditorSystemPrompt 构建 Editor 角色的 system prompt
func (s *WorldBuildingService) buildEditorSystemPrompt() string {
	return `你是一位资深文学编辑，负责审查小说的世界构建质量。请从以下5个维度进行评分（1-10分），并以严格的 JSON 格式返回结果：
1. 逻辑一致性 (logic_consistency) - 设定之间是否自洽
2. 角色弧光 (character_arc) - 人物是否有成长空间
3. 伏笔合理性 (foreshadow_quality) - 伏笔设计是否巧妙
4. 世界观完整性 (worldview_completeness) - 世界观是否全面
5. 情节张力 (plot_tension) - 情节是否有吸引力

返回格式：
{"dimensions":[{"name":"逻辑一致性","score":7.5,"comment":"..."},{"name":"角色弧光","score":7.0,"comment":"..."},{"name":"伏笔合理性","score":6.5,"comment":"..."},{"name":"世界观完整性","score":7.0,"comment":"..."},{"name":"情节张力","score":7.5,"comment":"..."}],"total_score":7.1,"summary":"总体评价...","suggestion":"修改建议..."}`
}

// buildEditorUserPrompt 构建 Editor 角色的 user prompt
func (s *WorldBuildingService) buildEditorUserPrompt(phase, content string) string {
	label := phaseLabel(phase)
	return fmt.Sprintf("请审查以下【%s】内容，按5个维度评分并给出修改建议：\n\n%s", label, content)
}

// buildOptimizeUserPrompt 构建优化 Prompt（上轮内容 + 审查意见）
func (s *WorldBuildingService) buildOptimizeUserPrompt(phase, prevContent string, review *model.ReviewResult) string {
	var prompt strings.Builder
	prompt.WriteString("请根据编辑的审查意见，对以下内容进行针对性优化改进。\n\n")
	prompt.WriteString("【上轮生成内容】\n")
	prompt.WriteString(prevContent)
	prompt.WriteString("\n\n【编辑审查意见】\n")
	fmt.Fprintf(&prompt, "总分：%.1f\n总评：%s\n修改建议：%s\n", review.TotalScore, review.Summary, review.Suggestion)
	for _, d := range review.Dimensions {
		fmt.Fprintf(&prompt, "- %s（%.1f分）：%s\n", d.Name, d.Score, d.Comment)
	}
	prompt.WriteString("\n请输出优化后的完整内容，保持原有格式。")
	return prompt.String()
}

// gatherPhaseContext 收集已完成阶段的设定作为上下文
func (s *WorldBuildingService) gatherPhaseContext(novelID uint, currentPhase string) string {
	// 按阶段顺序收集：worldview → character → relation → foreshadow → plot
	orderedPhases := []string{
		model.ReflectionPhaseWorldview,
		model.ReflectionPhaseCharacter,
		model.ReflectionPhaseRelation,
		model.ReflectionPhaseForeshadow,
		model.ReflectionPhasePlot,
	}

	var parts []string
	for _, p := range orderedPhases {
		if p == currentPhase {
			break // 只收集当前阶段之前的
		}
		best, err := s.dao.GetBestReflectionLog(novelID, p)
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("【%s】\n%s", phaseLabel(p), best.Content))
	}
	return strings.Join(parts, "\n\n")
}

// defaultUserInputForPhase 各阶段的默认用户输入
func (s *WorldBuildingService) defaultUserInputForPhase(phase string) string {
	defaults := map[string]string{
		model.ReflectionPhaseWorldview:  "请为这部小说构建完整的世界观设定。",
		model.ReflectionPhaseCharacter:  "请基于已有世界观，设计主要角色的详细设定。",
		model.ReflectionPhaseRelation:   "请基于已有角色，构建人物关系网络。",
		model.ReflectionPhaseForeshadow: "请基于已有设定，设计伏笔体系。",
		model.ReflectionPhasePlot:       "请基于所有已有设定，规划完整的剧情大纲。",
	}
	if v, ok := defaults[phase]; ok {
		return v
	}
	return "请完成相关创作任务。"
}

// phaseLabel 阶段中文标签（包级私有辅助函数）
func phaseLabel(phase string) string {
	labels := map[string]string{
		model.ReflectionPhaseWorldview:  "世界观设定",
		model.ReflectionPhaseCharacter:  "人物设定",
		model.ReflectionPhaseRelation:   "人物关系",
		model.ReflectionPhaseForeshadow: "伏笔设计",
		model.ReflectionPhasePlot:       "剧情大纲",
	}
	if v, ok := labels[phase]; ok {
		return v
	}
	return phase
}
