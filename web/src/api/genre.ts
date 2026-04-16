// web/src/api/genre.ts
import request from './request'

// ========== 类型定义 ==========

export interface Genre {
  id: number
  parent_id: number
  name: string
  slug: string
  icon: string
  sort_order: number
  created_at: string
}

export interface GenreTree extends Genre {
  children?: GenreTree[]
}

// ========== API ==========

export const genreApi = {
  // 获取赛道树
  listTree: () =>
    request.get('/genres'),

  // 获取赛道详情
  get: (id: number) =>
    request.get(`/genres/${id}`),

  // 管理员接口
  adminCreate: (data: { parent_id?: number; name: string; slug: string; icon?: string; sort_order?: number }) =>
    request.post('/admin/genres', data),
  adminUpdate: (id: number, data: Partial<{ parent_id: number; name: string; slug: string; icon: string; sort_order: number }>) =>
    request.put(`/admin/genres/${id}`, data),
  adminDelete: (id: number) =>
    request.delete(`/admin/genres/${id}`),
}
