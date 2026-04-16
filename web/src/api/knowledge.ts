// web/src/api/knowledge.ts
import request from './request'

// ========== 类型定义 ==========

export interface NovelKnowledge {
  id: number
  novel_id: number
  category: string
  title: string
  content: string
  tags: string
  chapter_ref: string
  priority: number
  status: string
  sort_order: number
  resolved: boolean
  created_at: string
  updated_at: string
}

// 知识类别选项
export const knowledgeCategories = [
  { value: 'character', label: '人物档案', icon: '👤' },
  { value: 'worldview', label: '世界观设定', icon: '🌍' },
  { value: 'plotline', label: '剧情线索', icon: '📖' },
  { value: 'foreshadow', label: '伏笔追踪', icon: '🔮' },
  { value: 'style', label: '文风规范', icon: '✒️' },
  { value: 'custom', label: '自定义', icon: '📌' },
] as const

// ========== API ==========

export const knowledgeApi = {
  // 获取小说的知识条目列表
  list: (novelId: number, params?: { category?: string; status?: string }) =>
    request.get(`/novels/${novelId}/knowledge`, { params }),

  // 创建知识条目
  create: (novelId: number, data: {
    category: string
    title: string
    content: string
    tags?: string
    chapter_ref?: string
    priority?: number
  }) => request.post(`/novels/${novelId}/knowledge`, data),

  // 获取知识条目详情
  get: (kid: number) => request.get(`/knowledge/${kid}`),

  // 更新知识条目
  update: (kid: number, data: Partial<{
    category: string
    title: string
    content: string
    tags: string
    chapter_ref: string
    priority: number
    status: string
  }>) => request.put(`/knowledge/${kid}`, data),

  // 删除知识条目
  delete: (kid: number) => request.delete(`/knowledge/${kid}`),

  // 确认待审核条目
  confirm: (kid: number) => request.post(`/knowledge/${kid}/confirm`),

  // 批量确认
  batchConfirm: (novelId: number) =>
    request.post(`/novels/${novelId}/knowledge/batch-confirm`),

  // 搜索知识条目
  search: (novelId: number, keyword: string) =>
    request.get(`/novels/${novelId}/knowledge/search`, { params: { keyword } }),

  // AI 提取知识
  extract: (novelId: number, data: { chapter_id: number; model_name?: string }) =>
    request.post(`/novels/${novelId}/knowledge/extract`, data),

  // 解析 AI 提取结果
  parseExtract: (novelId: number, data: { chapter_id: number; task_id: number }) =>
    request.post(`/novels/${novelId}/knowledge/parse-extract`, data),
}
