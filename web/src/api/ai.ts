// web/src/api/ai.ts
import request from './request'

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface TextGenRequest {
  portfolio_id: number
  model_name: string
  prompt: string
  history?: ChatMessage[]
}

export interface ImageGenRequest {
  portfolio_id: number
  model_name: string
  prompt: string
}

export interface CharacterAdjustRequest {
  portfolio_id: number
  model_name: string
  prompt: string
}

export interface TaskResponse {
  task_id: number
  status: string
}

export interface AITask {
  id: number
  user_id: number
  portfolio_id: number
  task_type: string
  model_name: string
  prompt: string
  status: string
  result: string
  error_msg: string
  novel_id: number
  created_at: string
  updated_at: string
}

export const aiApi = {
  generateText: (data: TextGenRequest) =>
    request.post('/ai/text/generate', data),
  generateImage: (data: ImageGenRequest) =>
    request.post('/ai/image/generate', data),
  adjustCharacter: (data: CharacterAdjustRequest) =>
    request.post('/ai/character/adjust', data),
  getTask: (taskId: number) => request.get(`/ai/tasks/${taskId}`),
  listTasks: (params?: { portfolio_id?: number; page?: number; page_size?: number; task_types?: string; butler_session_id?: string; novel_id?: number }) =>
    request.get('/ai/tasks', { params }),
  cancelTask: (taskId: number) => request.delete(`/ai/tasks/${taskId}`),
}
