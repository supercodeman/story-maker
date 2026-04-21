// server/internal/agent/content_cleaner.go
package agent

import (
	"regexp"
	"strings"
)

// CleanNovelContent 清洗 AI 生成的小说正文内容，移除非正文信息
// 按优先级依次处理：分隔线截断 → 尾部标签块移除 → 尾部注释行移除 → 首部元信息移除
func CleanNovelContent(content string) string {
	if content == "" {
		return content
	}

	content = removeAIPreamble(content)
	content = truncateAtSeparator(content)
	content = removeTailBracketBlocks(content)
	content = removeTailNoteLines(content)
	content = removeHeadMetaLines(content)

	return strings.TrimSpace(content)
}

// removeAIPreamble 移除 AI 输出的前导说明和尾部说明
// 如 "以下是润色后的正文：" "润色说明：" "修改要点：" "扩写说明：" 等
func removeAIPreamble(content string) string {
	// 移除开头的引导语（"以下是..." "好的，..." 等）
	headRe := regexp.MustCompile(`^(?:\s*(?:以下是|好的[，,]|下面是|这是)[^\n]*(?:：|:)\s*\n)+`)
	content = headRe.ReplaceAllString(content, "")
	// 移除尾部的说明段落
	tailRe := regexp.MustCompile(`(?s)\n\s*(?:润色说明|修改要点|扩写说明|改动说明|修改说明|主要改动|主要修改)[：:][^\n]*(\n[\s\S]*)?$`)
	content = tailRe.ReplaceAllString(content, "")
	return content
}

// truncateAtSeparator 遇到独占一行的分隔线（---、===、***），截取之前的内容
func truncateAtSeparator(content string) string {
	separatorRe := regexp.MustCompile(`(?m)^\s*([-]{3,}|[=]{3,}|[*]{3,})\s*$`)
	loc := separatorRe.FindStringIndex(content)
	if loc == nil {
		return content
	}
	// 只截断位于后半部分的分隔线，避免误伤正文中间的场景分隔
	if loc[0] > len(content)/2 {
		return strings.TrimSpace(content[:loc[0]])
	}
	return content
}

// removeTailBracketBlocks 移除末尾以中文方括号标签开头的整段
// 如【附注】、【前情提要】、【说明】、【备注】、【注意】等
func removeTailBracketBlocks(content string) string {
	tailBracketRe := regexp.MustCompile(`(?s)\n\s*【(附注|前情提要|说明|备注|注意|提示|温馨提示|作者的话|写作说明)】[^\n]*(\n|$)[\s\S]*$`)
	return tailBracketRe.ReplaceAllString(content, "")
}

// removeTailNoteLines 移除末尾以注释标记开头的行及其后续内容
// 如 "注：" "注意：" "附注：" "P.S." "（注：" "备注：" "说明：" 等
func removeTailNoteLines(content string) string {
	tailNoteRe := regexp.MustCompile(`(?s)\n\s*(注：|注意：|附注：|P\.S\.|P\.S：|（注：|备注：|说明：|——注：|——附注)[^\n]*(\n|$)[\s\S]*$`)
	return tailNoteRe.ReplaceAllString(content, "")
}

// removeHeadMetaLines 移除开头的元信息行（AI 有时会重复输出标题/章节号）
// 如 "标题：xxx" "章节：xxx" "第X章 xxx" 后紧跟空行的情况
func removeHeadMetaLines(content string) string {
	headMetaRe := regexp.MustCompile(`^(\s*(标题[：:].+|章节[：:].+|第[一二三四五六七八九十百千\d]+章\s*.+)\s*\n)+\s*\n`)
	return headMetaRe.ReplaceAllString(content, "")
}
