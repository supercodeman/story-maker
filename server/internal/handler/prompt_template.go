// server/internal/handler/prompt_template.go
package handler

import (
	"net/http"
	"strconv"

	"ai-curton/server/internal/model"
	"ai-curton/server/internal/service"

	"github.com/gin-gonic/gin"
)

// PromptTemplateHandler Prompt 模板请求处理层
type PromptTemplateHandler struct {
	svc *service.PromptTemplateService
}

// NewPromptTemplateHandler 创建 PromptTemplateHandler 实例
func NewPromptTemplateHandler(svc *service.PromptTemplateService) *PromptTemplateHandler {
	return &PromptTemplateHandler{svc: svc}
}

// ListTemplates 列出小说的模板（合并默认+自定义）
// GET /api/v1/novels/:id/prompt-templates
func (h *PromptTemplateHandler) ListTemplates(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	templates, err := h.svc.ListMerged(uint(novelID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, templates)
}

// UpsertTemplate 创建/更新自定义模板
// PUT /api/v1/novels/:id/prompt-templates
func (h *PromptTemplateHandler) UpsertTemplate(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid novel id")
		return
	}

	var req struct {
		Action     string `json:"action" binding:"required,oneof=summary_polish polish expand continue outline_generate outline_title_polish outline_summary_polish outline_summary_expand"`
		PromptType string `json:"prompt_type" binding:"required,oneof=system user"`
		Name       string `json:"name" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	tpl := &model.PromptTemplate{
		NovelID:    uint(novelID),
		Action:     req.Action,
		PromptType: req.PromptType,
		Name:       req.Name,
		Content:    req.Content,
		IsDefault:  false,
	}

	if err := h.svc.Upsert(tpl); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, tpl)
}

// DeleteTemplate 删除自定义模板（恢复默认）
// DELETE /api/v1/novels/:id/prompt-templates/:tid
func (h *PromptTemplateHandler) DeleteTemplate(c *gin.Context) {
	tid, err := strconv.ParseUint(c.Param("tid"), 10, 64)
	if err != nil {
		BadRequest(c, "invalid template id")
		return
	}

	if err := h.svc.Delete(uint(tid)); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// PreviewTemplate 预览渲染结果
// POST /api/v1/novels/:id/prompt-templates/preview
func (h *PromptTemplateHandler) PreviewTemplate(c *gin.Context) {
	var req struct {
		Content string                  `json:"content" binding:"required"`
		Data    model.PromptTemplateData `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.PreviewTemplate(req.Content, &req.Data)
	if err != nil {
		Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	Success(c, gin.H{"rendered": result})
}
