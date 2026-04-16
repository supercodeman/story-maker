// web/src/api/market.ts
import request from './request'

// ========== 类型定义 ==========

export interface MemoryOrder {
  id: number
  order_no: string
  buyer_id: number
  seller_id: number
  memory_id: number
  amount: number
  platform_fee: number
  seller_income: number
  status: string
  created_at: string
}

export interface MemoryReview {
  id: number
  user_id: number
  memory_id: number
  rating: number
  comment: string
  created_at: string
}

// ========== API ==========

export const marketApi = {
  // 市场浏览
  listMemories: (params?: { category?: string; keyword?: string; order_by?: string; page?: number; page_size?: number; genre_id?: number }) =>
    request.get('/market/memories', { params }),

  // 记忆详情
  getMemory: (mid: number) =>
    request.get(`/market/memories/${mid}`),

  // 购买
  buy: (mid: number) =>
    request.post(`/market/memories/${mid}/buy`),

  // 评价
  listReviews: (mid: number, params?: { page?: number; page_size?: number }) =>
    request.get(`/market/memories/${mid}/reviews`, { params }),
  submitReview: (mid: number, data: { rating: number; comment?: string }) =>
    request.post(`/market/memories/${mid}/reviews`, data),
}
