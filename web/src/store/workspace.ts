// web/src/store/workspace.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import request from '@/api/request'

export interface Workspace {
  id: number
  name: string
  type: string
  owner_id: number
  description: string
  created_at: string
  updated_at: string
}

export const useWorkspaceStore = defineStore('workspace', () => {
  const workspaces = ref<Workspace[]>([])
  const currentWorkspace = ref<Workspace | null>(null)
  const loading = ref(false)

  // 获取工作空间列表
  async function fetchWorkspaces() {
    loading.value = true
    try {
      const data: any = await request.get('/workspaces')
      workspaces.value = data.items || data || []
    } finally {
      loading.value = false
    }
  }

  // 获取单个工作空间
  async function fetchWorkspace(id: number) {
    loading.value = true
    try {
      const data: any = await request.get(`/workspaces/${id}`)
      currentWorkspace.value = data
    } finally {
      loading.value = false
    }
  }

  // 设置当前工作空间（同时持久化到 localStorage）
  function setCurrentWorkspace(ws: Workspace) {
    currentWorkspace.value = ws
    localStorage.setItem('currentWorkspaceId', String(ws.id))
  }

  // 更新工作空间信息，同步 currentWorkspace 和 workspaces 列表
  async function updateWorkspace(id: number, data: Partial<Pick<Workspace, 'name' | 'description'>>) {
    const updated: any = await request.put(`/workspaces/${id}`, data)
    if (currentWorkspace.value?.id === id) {
      Object.assign(currentWorkspace.value, data)
    }
    const idx = workspaces.value.findIndex(w => w.id === id)
    if (idx !== -1) {
      Object.assign(workspaces.value[idx], data)
    }
    return updated
  }

  return {
    workspaces,
    currentWorkspace,
    loading,
    fetchWorkspaces,
    fetchWorkspace,
    setCurrentWorkspace,
    updateWorkspace,
  }
})
