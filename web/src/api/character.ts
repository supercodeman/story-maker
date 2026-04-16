// web/src/api/character.ts
import request from './request'

export interface Character {
  id: number
  portfolio_id: number
  name: string
  description: string
  reference_images: string
  attributes: string
  created_at: string
  updated_at: string
}

export interface CreateCharacterPayload {
  name: string
  description?: string
  attributes?: Record<string, string>
}

export const characterApi = {
  list: (portfolioId: number) =>
    request.get(`/portfolios/${portfolioId}/characters`),
  get: (id: number) => request.get(`/characters/${id}`),
  create: (portfolioId: number, data: CreateCharacterPayload) =>
    request.post(`/portfolios/${portfolioId}/characters`, data),
  update: (id: number, data: Partial<CreateCharacterPayload>) =>
    request.put(`/characters/${id}`, data),
  delete: (id: number) => request.delete(`/characters/${id}`),
  uploadReference: (id: number, file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request.post(`/characters/${id}/reference`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}
