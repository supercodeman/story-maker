// server/internal/handler/knowledge.go
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识库请求处理层
type KnowledgeHandler struct {
	svc *service.KnowledgeService
}

// NewKnowledgeHandler 创建 KnowledgeHandler 实例
func NewKnowledgeHandler(svc *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc}
}

// Create 创建知识条目
// POST /api/v1/novels/:id/knowledge
func (h *KnowledgeHandler) Create(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.CreateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	k, err := h.svc.Create(uint(novelID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, k)
}

// List 获取知识条目列表
// GET /api/v1/novels/:id/knowledge?category=xxx&status=xxx
func (h *KnowledgeHandler) List(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	category := c.Query("category")
	status := c.Query("status")

	items, err := h.svc.List(uint(novelID), category, status)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}

// Get 获取知识条目详情
// GET /api/v1/knowledge/:kid
func (h *KnowledgeHandler) Get(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	k, err := h.svc.Get(uint(kid))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, k)
}

// Update 更新知识条目
// PUT /api/v1/knowledge/:kid
func (h *KnowledgeHandler) Update(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	var req service.UpdateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	k, err := h.svc.Update(uint(kid), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, k)
}

// Delete 删除知识条目
// DELETE /api/v1/knowledge/:kid
func (h *KnowledgeHandler) Delete(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	if err := h.svc.Delete(uint(kid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Confirm 确认待审核条目
// POST /api/v1/knowledge/:kid/confirm
func (h *KnowledgeHandler) Confirm(c *gin.Context) {
	kid, err := strconv.ParseUint(c.Param("kid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge id")
		return
	}

	if err := h.svc.Confirm(uint(kid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// BatchConfirm 批量确认小说下所有待审核条目
// POST /api/v1/novels/:id/knowledge/batch-confirm
func (h *KnowledgeHandler) BatchConfirm(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	if err := h.svc.BatchConfirm(uint(novelID)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Search 搜索知识条目
// GET /api/v1/novels/:id/knowledge/search?keyword=xxx
func (h *KnowledgeHandler) Search(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		BadRequest(c, "keyword is required")
		return
	}

	items, err := h.svc.Search(uint(novelID), keyword)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}

// Extract AI 提取知识条目
// POST /api/v1/novels/:id/knowledge/extract
func (h *KnowledgeHandler) Extract(c *gin.Context) {
	userID := c.GetUint("user_id")
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req service.ExtractKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	taskID, err := h.svc.ExtractFromChapter(c.Request.Context(), userID, uint(novelID), &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"task_id": taskID})
}

// ParseExtract 解析 AI 提取结果并写入知识库
// POST /api/v1/novels/:id/knowledge/parse-extract
func (h *KnowledgeHandler) ParseExtract(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req struct {
		ChapterID uint `json:"chapter_id" binding:"required"`
		TaskID    uint `json:"task_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	items, err := h.svc.ParseExtractResult(uint(novelID), req.ChapterID, req.TaskID)
	if err != nil {
		if err.Error() == "task is not completed" {
			Error(c, http.StatusConflict, err.Error())
			return
		}
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}

// GenerateQuestions 基于知识点生成面试题
// POST /api/v1/knowledge/points/:id/generate-questions
func (h *KnowledgeHandler) GenerateQuestions(c *gin.Context) {
	pointID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid knowledge point id")
		return
	}

	var req struct {
		Count      int    `json:"count"`
		Difficulty string `json:"difficulty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 默认值
	if req.Count <= 0 {
		req.Count = 3
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	userID := c.GetUint("user_id")
	result, err := h.svc.GenerateQuestions(c.Request.Context(), userID, uint(pointID), req.Count, req.Difficulty)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	log.Printf("[knowledge] GenerateQuestions 原始返回: %s", result)

	// result可能是完整的TextResponse JSON: {"content": "...", "usage": {...}}
	// 需要先解析提取content字段
	var resultObj map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultObj); err == nil {
		log.Printf("[knowledge] 成功解析为对象，字段: %v", resultObj)
		// 如果能解析为对象，尝试提取content字段
		if content, ok := resultObj["content"].(string); ok {
			log.Printf("[knowledge] 提取到content字段，长度: %d", len(content))
			result = content
		} else {
			log.Printf("[knowledge] content字段不存在或不是字符串")
		}
	} else {
		log.Printf("[knowledge] 无法解析为对象: %v", err)
	}

	// 尝试解析AI返回的JSON字符串
	var questions []map[string]interface{}

	// 清理可能的markdown代码块标记
	cleanResult := result
	if strings.HasPrefix(result, "```json") {
		cleanResult = strings.TrimPrefix(result, "```json")
		cleanResult = strings.TrimSuffix(cleanResult, "```")
		cleanResult = strings.TrimSpace(cleanResult)
		log.Printf("[knowledge] 清理了markdown json标记")
	} else if strings.HasPrefix(result, "```") {
		cleanResult = strings.TrimPrefix(result, "```")
		cleanResult = strings.TrimSuffix(cleanResult, "```")
		cleanResult = strings.TrimSpace(cleanResult)
		log.Printf("[knowledge] 清理了markdown标记")
	}

	log.Printf("[knowledge] 准备解析的内容(前100字符): %s", cleanResult[:min(100, len(cleanResult))])

	if err := json.Unmarshal([]byte(cleanResult), &questions); err != nil {
		// 解析失败，记录日志并返回错误
		log.Printf("[knowledge] 解析AI返回的JSON失败: %v", err)
		log.Printf("[knowledge] 清理后内容(前200字符): %s", cleanResult[:min(200, len(cleanResult))])
		InternalError(c, "AI返回的数据格式错误，请检查AI服务配置")
		return
	}

	log.Printf("[knowledge] 成功解析，题目数量: %d", len(questions))

	// 返回解析后的数组
	Success(c, questions)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
