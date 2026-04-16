// web/src/api/workspace.ts
import request from './request'
import type { PaginatedResponse } from './types'

export interface Workspace {
  id: number
  name: string
  type: string
  owner_id: number
  description: string
  created_at: string
  updated_at: string
}

export interface WorkspaceMember {
  id: number
  workspace_id: number
  user_id: number
  role: string
  created_at: string
  username?: string
  email?: string
}

export interface CreateWorkspacePayload {
  name: string
  type: string
  description?: string
}

export interface AddMemberPayload {
  user_id: number
  role: string
}

export const workspaceApi = {
  // 获取工作空间列表
  list: () => request.get<PaginatedResponse<Workspace>>('/workspaces'),

  // 获取单个工作空间
  get: (id: number) => request.get<Workspace>(`/workspaces/${id}`),

  // 创建工作空间
  create: (data: CreateWorkspacePayload) => request.post<Workspace>('/workspaces', data),

  // 更新工作空间
  update: (id: number, data: Partial<CreateWorkspacePayload>) =>
    request.put<Workspace>(`/workspaces/${id}`, data),

  // 删除工作空间
  delete: (id: number) => request.delete(`/workspaces/${id}`),

  // 获取成员列表
  getMembers: (id: number) => request.get<WorkspaceMember[]>(`/workspaces/${id}/members`),

  // 添加成员
  addMember: (id: number, data: AddMemberPayload) =>
    request.post<WorkspaceMember>(`/workspaces/${id}/members`, data),

  // 移除成员
  removeMember: (id: number, userId: number) =>
    request.delete(`/workspaces/${id}/members/${userId}`),

  // 更新成员角色
  updateMemberRole: (id: number, userId: number, role: string) =>
    request.put(`/workspaces/${id}/members/${userId}`, { role }),
}
