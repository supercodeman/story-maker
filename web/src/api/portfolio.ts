// web/src/api/portfolio.ts
import request from './request'

export interface Portfolio {
  id: number
  workspace_id: number
  name: string
  description: string
  cover_image: string
  status: string
  created_at: string
  updated_at: string
}

export interface CreatePortfolioPayload {
  workspace_id: number
  name: string
  description?: string
}

export interface UpdatePortfolioPayload {
  name?: string
  description?: string
  cover_image?: string
  status?: string
}

export const portfolioApi = {
  list: (workspaceId: number) =>
    request.get('/portfolios', { params: { workspace_id: workspaceId } }),
  get: (id: number) => request.get(`/portfolios/${id}`),
  create: (data: CreatePortfolioPayload) => request.post('/portfolios', data),
  update: (id: number, data: UpdatePortfolioPayload) =>
    request.put(`/portfolios/${id}`, data),
  delete: (id: number) => request.delete(`/portfolios/${id}`),
  uploadCover: (portfolioId: number, file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('portfolio_id', String(portfolioId))
    formData.append('type', 'image')
    return request.post('/assets/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}
