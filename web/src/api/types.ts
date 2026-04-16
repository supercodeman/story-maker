// web/src/api/types.ts

// 通用 API 响应结构
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// 分页响应
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

// 错误响应
export interface ApiError {
  code: number
  message: string
  details?: any
}
