// web/src/api/workflow.ts
import request from './request'

export interface Workflow {
  id: number
  user_id: number
  portfolio_id: number
  workflow_type: string
  status: string
  total_nodes: number
  completed_nodes: number
  result_json: string
  error_msg: string
  initial_context: string
  created_at: string
  updated_at: string
}

export interface WorkflowNode {
  id: number
  workflow_id: number
  node_id: string
  task_id: number
  status: string
  result_json: string
  error_msg: string
  created_at: string
  updated_at: string
}

export const workflowApi = {
  submit: (data: { portfolio_id: number; workflow_type: string; model_name: string; params: Record<string, any> }) =>
    request.post('/ai/workflows/submit', data),
  get: (id: number) => request.get(`/ai/workflows/${id}`),
  cancel: (id: number) => request.delete(`/ai/workflows/${id}`),
  listActive: (novelId: number) =>
    request.get('/ai/workflows/active', { params: { novel_id: novelId } }),
}
