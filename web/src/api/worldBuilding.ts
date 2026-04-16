// web/src/api/worldBuilding.ts
import request from './request'

// ========== 类型定义 ==========

export interface ReviewDimension {
  name: string
  score: number
  comment: string
}

export interface ReviewResult {
  dimensions: ReviewDimension[]
  total_score: number
  summary: string
  suggestion: string
}

export interface ReflectionConfig {
  max_rounds: number
  threshold: number
  auto_mode: boolean
  model_name?: string
}

export interface ReflectionLog {
  id: number
  novel_id: number
  phase: string
  round: number
  content: string
  review_json: string
  total_score: number
  task_id: number
  review_task_id: number
  created_at: string
}

export interface PhaseResultResponse {
  done: boolean
  round: number
  content?: string
  score?: number
  review_result?: ReviewResult
}

export interface WorldBuildingSummary {
  world_settings: any[]
  foreshadows: any[]
  plot_outlines: any[]
}

export interface PhaseStatusResponse {
  phase: string
  round: number
  status: string
  content: string
  review_result?: ReviewResult
  config: ReflectionConfig
}

// 反思阶段类型
export type ReflectionPhase = 'worldview' | 'character' | 'relation' | 'foreshadow' | 'plot'

// ========== API ==========

export const worldBuildingApi = {
  // 启动阶段生成
  startPhase: (data: {
    novel_id: number
    portfolio_id: number
    phase: ReflectionPhase
    user_input?: string
    config?: Partial<ReflectionConfig>
  }) => request.post('/world-building/start', data),

  // 触发审查
  reviewResult: (data: {
    novel_id: number
    portfolio_id: number
    phase: ReflectionPhase
    task_id: number
  }) => request.post('/world-building/review', data),

  // 处理审查结果
  processReview: (data: {
    novel_id: number
    phase: ReflectionPhase
    generate_task_id: number
    review_task_id: number
    config?: Partial<ReflectionConfig>
  }) => request.post('/world-building/process-review', data),

  // 基于审查意见优化
  optimize: (data: {
    novel_id: number
    portfolio_id: number
    phase: ReflectionPhase
    prev_content: string
    review_json: string
    max_rounds?: number
    threshold?: number
  }) => request.post('/world-building/optimize', data),

  // 接受阶段结果
  acceptResult: (data: {
    novel_id: number
    phase: ReflectionPhase
    content: string
    score?: number
  }) => request.post('/world-building/accept', data),

  // 获取阶段状态
  getStatus: (novelId: number, phase: ReflectionPhase) =>
    request.get('/world-building/status', { params: { novel_id: novelId, phase } }),

  // 获取世界构建概览
  getSummary: (novelId: number) =>
    request.get('/world-building/summary', { params: { novel_id: novelId } }),
}
