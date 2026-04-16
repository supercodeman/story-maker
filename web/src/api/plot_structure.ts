// web/src/api/plot_structure.ts
import request from './request'

// ========== 类型定义 ==========

export interface PlotPhase {
  phase: number
  name: string
  description: string
  ratio: number
  beats: string[]
}

export interface PlotStructureTemplate {
  id: number
  name: string
  category: string
  description: string
  structure: string // JSON string of PlotPhase[]
  is_system: boolean
  user_id: number
  created_at: string
  updated_at: string
}

// ========== API ==========

export const plotStructureApi = {
  list: () => request.get('/plot-templates'),
  get: (id: number) => request.get(`/plot-templates/${id}`),
  create: (data: { name: string; description?: string; structure: string }) =>
    request.post('/plot-templates', data),
  aiGenerate: (data: { description: string; model_name?: string }) =>
    request.post('/plot-templates/ai-generate', data),
  update: (id: number, data: { name?: string; description?: string; structure?: string }) =>
    request.put(`/plot-templates/${id}`, data),
  delete: (id: number) => request.delete(`/plot-templates/${id}`),
}
