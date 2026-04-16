// web/src/api/overview.ts
import request from './request'

// ========== 类型定义 ==========

export interface CharacterRelation {
  id: number
  novel_id: number
  from_knowledge_id: number
  to_knowledge_id: number
  relation_type: string
  label: string
  chapter_ref: string
  created_at: string
  updated_at: string
}

export interface ChapterBrief {
  id: number
  title: string
  sort_order: number
  summary: string
  status: string
}

export interface OverviewData {
  plotlines: import('@/api/knowledge').NovelKnowledge[]
  characters: import('@/api/knowledge').NovelKnowledge[]
  foreshadows: import('@/api/knowledge').NovelKnowledge[]
  relations: CharacterRelation[]
  chapters: ChapterBrief[]
}

export interface OverviewChange {
  type: 'plotline' | 'character' | 'foreshadow' | 'relation'
  action: 'create' | 'update' | 'delete'
  id?: number
  data?: Record<string, any>
  old_data?: Record<string, any>
}

// 关系类型选项
export const relationTypes = [
  { value: 'ally', label: '盟友' },
  { value: 'enemy', label: '敌人' },
  { value: 'mentor', label: '师徒' },
  { value: 'lover', label: '恋人' },
  { value: 'family', label: '亲属' },
  { value: 'rival', label: '对手' },
  { value: 'custom', label: '自定义' },
] as const

// ========== API ==========

export const overviewApi = {
  // 获取总览数据
  get: (novelId: number) =>
    request.get(`/novels/${novelId}/overview`),

  // 创建人物关系
  createRelation: (novelId: number, data: {
    from_knowledge_id: number
    to_knowledge_id: number
    relation_type: string
    label?: string
    chapter_ref?: string
  }) => request.post(`/novels/${novelId}/overview/relations`, data),

  // 更新人物关系
  updateRelation: (novelId: number, rid: number, data: {
    relation_type?: string
    label?: string
    chapter_ref?: string
  }) => request.put(`/novels/${novelId}/overview/relations/${rid}`, data),

  // 删除人物关系
  deleteRelation: (novelId: number, rid: number) =>
    request.delete(`/novels/${novelId}/overview/relations/${rid}`),

  // AI 提取总览元数据
  extract: (novelId: number, data: { model_name?: string }) =>
    request.post(`/novels/${novelId}/overview/extract`, data),

  // 解析 AI 提取结果并入库
  parseExtract: (novelId: number, taskId: number) =>
    request.post(`/novels/${novelId}/overview/extract/parse`, { task_id: taskId }),

  // 提交变更（触发分析工作流）
  submitRevision: (novelId: number, portfolioId: number, data: {
    model_name?: string
    changes: OverviewChange[]
  }) => request.post(`/novels/${novelId}/overview/revision?portfolio_id=${portfolioId}`, data),

  // 确认执行变更（触发执行工作流）
  executeRevision: (novelId: number, portfolioId: number, data: {
    model_name?: string
    workflow_id: number
    revision_plan: string
  }) => request.post(`/novels/${novelId}/overview/revision/execute?portfolio_id=${portfolioId}`, data),
}
