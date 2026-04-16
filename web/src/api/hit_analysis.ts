// web/src/api/hit_analysis.ts
import request from './request'

// ========== 类型定义 ==========

export interface HitAnalysis {
  id: number
  user_id: number
  portfolio_id: number
  title: string
  author: string
  source_text: string
  report: string // JSON string
  workflow_id: number
  status: string // pending/running/completed/failed
  model_name: string
  created_at: string
  updated_at: string
}

export interface HitAnalysisReport {
  structure_analysis: string
  rhythm_analysis: string
  character_arcs: string
  hook_points: { position: string; type: string; technique: string }[]
  style_features: string
  summary: string
}

// ========== API ==========

export const hitAnalysisApi = {
  submit: (data: { portfolio_id: number; title: string; author?: string; source_text: string; model_name?: string }) =>
    request.post('/hit-analysis', data),
  list: () => request.get('/hit-analysis'),
  get: (id: number) => request.get(`/hit-analysis/${id}`),
  delete: (id: number) => request.delete(`/hit-analysis/${id}`),
}
