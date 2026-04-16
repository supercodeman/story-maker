// web/src/store/ai.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { aiApi } from '@/api/ai'
import type { AITask } from '@/api/ai'

export type { AITask } from '@/api/ai'

export const useAIStore = defineStore('ai', () => {
  const tasks = ref<AITask[]>([])
  const wsConnected = ref(false)
  const loading = ref(false)

  const pendingTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'pending' || t.status === 'running')
  )

  const completedTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'completed')
  )

  // 加载任务列表
  async function fetchTasks(portfolioId?: number) {
    loading.value = true
    try {
      const data: any = await aiApi.listTasks({
        portfolio_id: portfolioId,
        page: 1,
        page_size: 50,
      })
      tasks.value = data.tasks || []
    } finally {
      loading.value = false
    }
  }

  // 提交文本生成任务
  async function submitTextTask(portfolioId: number, modelName: string, prompt: string, history?: { role: string; content: string }[]) {
    const data: any = await aiApi.generateText({
      portfolio_id: portfolioId,
      model_name: modelName,
      prompt,
      history: history as any,
    })
    tasks.value.unshift({
      id: data.task_id,
      user_id: 0,
      portfolio_id: portfolioId,
      task_type: 'text_gen',
      model_name: modelName,
      prompt,
      status: 'pending',
      result: '',
      error_msg: '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    })
    return data.task_id
  }

  // 提交图像生成任务
  async function submitImageTask(portfolioId: number, modelName: string, prompt: string) {
    const data: any = await aiApi.generateImage({
      portfolio_id: portfolioId,
      model_name: modelName,
      prompt,
    })
    tasks.value.unshift({
      id: data.task_id,
      user_id: 0,
      portfolio_id: portfolioId,
      task_type: 'image_gen',
      model_name: modelName,
      prompt,
      status: 'pending',
      result: '',
      error_msg: '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    })
    return data.task_id
  }

  // 提交角色调整任务
  async function submitCharacterAdjustTask(
    portfolioId: number,
    modelName: string,
    prompt: string,
  ) {
    const data: any = await aiApi.adjustCharacter({
      portfolio_id: portfolioId,
      model_name: modelName,
      prompt,
    })
    tasks.value.unshift({
      id: data.task_id,
      user_id: 0,
      portfolio_id: portfolioId,
      task_type: 'character_adjust',
      model_name: modelName,
      prompt,
      status: 'pending',
      result: '',
      error_msg: '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    })
    return data.task_id
  }

  // 处理 WebSocket 任务更新
  function handleTaskUpdate(data: { task_id: number; status: string; result?: any; error?: string }) {
    const idx = tasks.value.findIndex((t) => t.id === data.task_id)
    if (idx !== -1) {
      tasks.value[idx].status = data.status
      if (data.result) {
        tasks.value[idx].result = JSON.stringify(data.result)
      }
      if (data.error) {
        tasks.value[idx].error_msg = data.error
      }
      tasks.value[idx].updated_at = new Date().toISOString()
    }
  }

  function setWsConnected(connected: boolean) {
    wsConnected.value = connected
  }

  function clearCompleted() {
    tasks.value = tasks.value.filter(
      (t) => t.status === 'pending' || t.status === 'running'
    )
  }

  return {
    tasks,
    wsConnected,
    loading,
    pendingTasks,
    completedTasks,
    fetchTasks,
    submitTextTask,
    submitImageTask,
    submitCharacterAdjustTask,
    handleTaskUpdate,
    setWsConnected,
    clearCompleted,
  }
})