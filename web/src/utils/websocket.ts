// web/src/utils/websocket.ts
import { useAIStore } from '@/store/ai'
import { useNovelStore } from '@/store/novel'
import { useWorkflowStore } from '@/store/workflow'
import { useKnowledgeStore } from '@/store/knowledge'
import { useOverviewStore } from '@/store/overview'
import { useMemoryStore } from '@/store/memory'
import { useComicDramaStore } from '@/store/comicDrama'

let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 10
const RECONNECT_INTERVAL = 3000

function buildWsUrl(token: string): string {
  // 优先使用环境变量
  const envUrl = import.meta.env.VITE_WS_URL
  if (envUrl) {
    return `${envUrl}?token=${token}`
  }
  // 根据当前页面协议和 host 动态构建
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws?token=${token}`
}

export function connectWebSocket() {
  const token = localStorage.getItem('access_token')
  if (!token) {
    console.warn('No access token, skipping WebSocket')
    return
  }

  // 清理旧连接
  if (ws) {
    ws.close()
    ws = null
  }

  const wsUrl = buildWsUrl(token)
  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('WebSocket connected')
    reconnectAttempts = 0
    const aiStore = useAIStore()
    aiStore.setWsConnected(true)
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'task_update' && msg.data) {
        const aiStore = useAIStore()
        aiStore.handleTaskUpdate(msg.data)

        const novelStore = useNovelStore()
        novelStore.handleTaskUpdate(msg.data)
        novelStore.handleOutlineTaskUpdate(msg.data)
        novelStore.handleOutlineChapterAIUpdate(msg.data)
        novelStore.handleCharacterGenUpdate(msg.data)
        novelStore.handleExpandTaskUpdate(msg.data)

        const knowledgeStore = useKnowledgeStore()
        knowledgeStore.handleExtractTaskUpdate(msg.data)

        const overviewStore = useOverviewStore()
        overviewStore.handleExtractTaskUpdate(msg.data)
      }
      if (msg.type === 'workflow_update' && msg.data) {
        const workflowStore = useWorkflowStore()
        workflowStore.handleWorkflowUpdate(msg.data)
      }
      if (msg.type === 'workflow_node_update' && msg.data) {
        const workflowStore = useWorkflowStore()
        workflowStore.handleNodeUpdate(msg.data)
      }
      if (msg.type === 'memory_update' && msg.data) {
        const memoryStore = useMemoryStore()
        memoryStore.handleMemoryUpdate(msg.data)
      }
      if (msg.type === 'token_update' && msg.data) {
        const novelStore = useNovelStore()
        novelStore.handleTokenUpdate(msg.data)
      }
      if (msg.type === 'comic_drama_stage_done' && msg.data) {
        const comicDramaStore = useComicDramaStore()
        comicDramaStore.handleStageDone(msg.data)
      }
    } catch (e) {
      console.error('Failed to parse WS message:', e)
    }
  }

  ws.onerror = () => {
    const aiStore = useAIStore()
    aiStore.setWsConnected(false)
  }

  ws.onclose = () => {
    console.log('WebSocket disconnected')
    const aiStore = useAIStore()
    aiStore.setWsConnected(false)
    ws = null

    // 自动重连
    if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
      reconnectAttempts++
      reconnectTimer = setTimeout(connectWebSocket, RECONNECT_INTERVAL)
    }
  }
}

export function disconnectWebSocket() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  reconnectAttempts = MAX_RECONNECT_ATTEMPTS // 阻止重连
  if (ws) {
    ws.close()
    ws = null
  }
}
