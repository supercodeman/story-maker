// web/src/api/prompt_template.ts
import request from './request'

export interface PromptTemplate {
  id: number
  novel_id: number
  action: string
  prompt_type: string
  name: string
  content: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface PromptTemplateData {
  NovelTitle?: string
  NovelDescription?: string
  ChapterTitle?: string
  ChapterSummary?: string
  ChapterContent?: string
  PrevSummaries?: string
  PrevContent?: string
  WordCount?: number
  TargetWords?: number
  SelectedText?: string
}

export const promptTemplateApi = {
  // 列出小说的模板（合并默认+自定义）
  list: (novelId: number) =>
    request.get(`/novels/${novelId}/prompt-templates`),

  // 创建/更新自定义模板
  upsert: (novelId: number, data: { action: string; prompt_type: string; name: string; content: string }) =>
    request.put(`/novels/${novelId}/prompt-templates`, data),

  // 删除自定义模板（恢复默认）
  delete: (novelId: number, templateId: number) =>
    request.delete(`/novels/${novelId}/prompt-templates/${templateId}`),

  // 预览渲染结果
  preview: (novelId: number, data: { content: string; data: PromptTemplateData }) =>
    request.post(`/novels/${novelId}/prompt-templates/preview`, data),
}
