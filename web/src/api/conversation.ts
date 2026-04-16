// web/src/api/conversation.ts
import request from './request'

export interface Conversation {
  id: number
  user_id: number
  portfolio_id: number
  title: string
  model_name: string
  summary: string
  message_count: number
  status: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: number
  conversation_id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  token_count: number
  task_id: number | null
  created_at: string
}

export interface CreateConversationRequest {
  portfolio_id: number
  model_name?: string
  title?: string
}

export interface SendMessageRequest {
  content: string
}

export const conversationApi = {
  create: (data: CreateConversationRequest) =>
    request.post('/conversations', data),

  list: (params?: { portfolio_id?: number; page?: number; page_size?: number }) =>
    request.get('/conversations', { params }),

  get: (id: number) =>
    request.get(`/conversations/${id}`),

  sendMessage: (id: number, data: SendMessageRequest) =>
    request.post(`/conversations/${id}/messages`, data),

  archive: (id: number) =>
    request.delete(`/conversations/${id}`),
}
