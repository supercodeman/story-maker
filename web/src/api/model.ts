// web/src/api/model.ts
import request from './request'

export interface ModelInfo {
  value: string
  label: string
  provider: string
  available: boolean
  key_source: string
  latency_ms: number
  sub_models?: ModelInfo[]
}

// 模型状态详情（调试面板用）
export interface ModelStatusDetail {
  id: number
  provider: string
  model_name: string
  capability: string
  available: boolean
  latency_ms: number
  priority: number
  last_check: string | null
  last_error: string
}

// 获取可用模型列表
export function getAvailableModels(capability?: string) {
  return request.get<ModelInfo[]>('/models/available', { params: { capability } })
}

// 获取所有模型状态详情（管理员）
export function getModelStatus() {
  return request.get<ModelStatusDetail[]>('/models/status')
}

// 手动触发健康检查
export function triggerHealthCheck() {
  return request.post('/models/check')
}

// 新增模型请求
export interface AddModelReq {
  provider: string
  model_name: string
  capability: string
}

// 单模型测试结果
export interface TestModelResult {
  available: boolean
  latency_ms: number
  error: string
}

// 新增模型
export function addModel(data: AddModelReq) {
  return request.post('/models', data)
}

// 删除模型
export function deleteModel(id: number) {
  return request.delete(`/models/${id}`)
}

// 测试单个模型
export function testModel(data: { provider: string; model_name: string; capability: string }) {
  return request.post<TestModelResult>('/models/test', data)
}

// 更新模型优先级
export function updateModelPriority(id: number, priority: number) {
  return request.put(`/models/${id}/priority`, { priority })
}
