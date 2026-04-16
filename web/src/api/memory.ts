// web/src/api/memory.ts
import request from './request'

// ========== 类型定义 ==========

export interface WritingMemory {
  id: number
  user_id: number
  category: string
  title: string
  description: string
  features: string
  prompt_tpl: string
  anchor_texts: string
  preview_text: string
  version: number
  quality: number
  quality_detail: string
  quality_grade: string
  status: string
  is_public: boolean
  price: number
  sales_count: number
  avg_rating: number
  rating_count: number
  tags: string
  sample_len: number
  extract_workflow_id: number
  extract_status: string
  extract_error: string
  review_workflow_id: number
  review_reason: string
  created_at: string
  updated_at: string
}

// 质量评分详情
export interface QualityDetail {
  consistency: number
  reproducibility: number
  uniqueness: number
  practicality: number
  preview_text: string
  evaluation: string
}

// 风格子维度
export interface StyleDimension {
  description: string
  score: number
  examples: string[]
  prompt_part: string
}

// 风格特征（新格式）
export interface StyleFeatures {
  tone: StyleDimension
  rhythm: StyleDimension
  vocabulary: StyleDimension
  dialogue_style: StyleDimension
  forbidden_patterns: string[]
  reference_style: string
}

export interface NovelMemoryBinding {
  id: number
  novel_id: number
  category: string
  memory_id: number
  created_at: string
}

// 记忆类别选项
export const memoryCategoryOptions = [
  { value: 'style', label: '写作风格' },
  { value: 'character', label: '人设模板' },
  { value: 'worldview', label: '世界观框架' },
  { value: 'plot_preference', label: '剧情偏好' },
]

// 记忆状态选项
export const memoryStatusOptions = [
  { value: 'draft', label: '草稿' },
  { value: 'reviewing', label: '审核中' },
  { value: 'published', label: '已上架' },
  { value: 'rejected', label: '已拒绝' },
  { value: 'archived', label: '已下架' },
]

// ========== API ==========

export const memoryApi = {
  // 记忆 CRUD
  create: (data: { category: string; title: string; description?: string; sample_text: string; tags?: string; model_name?: string; genre_ids?: number[] }) =>
    request.post('/memories', data),
  list: (params?: { category?: string }) =>
    request.get('/memories', { params }),
  get: (mid: number) =>
    request.get(`/memories/${mid}`),
  update: (mid: number, data: Partial<{ title: string; description: string; tags: string; genre_ids: number[] }>) =>
    request.put(`/memories/${mid}`, data),
  delete: (mid: number) =>
    request.delete(`/memories/${mid}`),

  // 追加样本
  refine: (mid: number, data: { additional_text: string }) =>
    request.post(`/memories/${mid}/refine`, data),

  // 上架/下架
  publish: (mid: number, data: { price: number }) =>
    request.post(`/memories/${mid}/publish`, data),
  archive: (mid: number) =>
    request.post(`/memories/${mid}/archive`),

  // 生成预览
  preview: (mid: number, data?: { model_name?: string }) =>
    request.post(`/memories/${mid}/preview`, data),

  // 可用记忆列表（自己的 + 已购买的）
  listAccessible: (params?: { category?: string }) =>
    request.get('/memories/accessible', { params }),

  // 小说-记忆绑定
  listBindings: (novelId: number) =>
    request.get(`/novels/${novelId}/memory-bindings`),
  setBindings: (novelId: number, data: { bindings: Array<{ category: string; memory_id: number }> }) =>
    request.put(`/novels/${novelId}/memory-bindings`, data),

  // 管理员审核接口
  adminListReviewing: () =>
    request.get('/admin/memories/reviewing'),
  adminApprove: (mid: number) =>
    request.post(`/admin/memories/${mid}/approve`),
  adminReject: (mid: number, data: { reason: string }) =>
    request.post(`/admin/memories/${mid}/reject`, data),
}
