// web/src/store/knowledge.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { knowledgeApi } from '@/api/knowledge'
import { aiApi } from '@/api/ai'
import type { NovelKnowledge } from '@/api/knowledge'
import { ElMessage } from 'element-plus'

export type { NovelKnowledge } from '@/api/knowledge'

export const useKnowledgeStore = defineStore('knowledge', () => {
  const items = ref<NovelKnowledge[]>([])
  const loading = ref(false)
  const currentNovelId = ref<number | null>(null)
  const activeCategory = ref<string>('')
  const extractPending = ref(false)
  const extractTaskId = ref<number | null>(null)
  const extractChapterId = ref<number | null>(null)
  let extractPollTimer: ReturnType<typeof setInterval> | null = null

  function stopExtractPolling() {
    if (extractPollTimer) {
      clearInterval(extractPollTimer)
      extractPollTimer = null
    }
  }

  // 按类别分组
  const groupedItems = computed(() => {
    const groups: Record<string, NovelKnowledge[]> = {}
    for (const item of items.value) {
      if (!groups[item.category]) {
        groups[item.category] = []
      }
      groups[item.category].push(item)
    }
    return groups
  })

  // 待审核条目数
  const pendingCount = computed(() =>
    items.value.filter(i => i.status === 'pending').length
  )

  // 已确认条目数
  const confirmedCount = computed(() =>
    items.value.filter(i => i.status === 'confirmed').length
  )

  // 获取知识条目列表
  async function fetchItems(novelId: number, category?: string) {
    loading.value = true
    currentNovelId.value = novelId
    try {
      const data: any = await knowledgeApi.list(novelId, { category })
      items.value = Array.isArray(data) ? data : []
    } finally {
      loading.value = false
    }
  }

  // 创建知识条目
  async function createItem(novelId: number, data: {
    category: string
    title: string
    content: string
    tags?: string
    chapter_ref?: string
    priority?: number
  }) {
    const item: any = await knowledgeApi.create(novelId, data)
    items.value.push(item)
    return item
  }

  // 更新知识条目
  async function updateItem(kid: number, data: Partial<NovelKnowledge>) {
    const updated: any = await knowledgeApi.update(kid, data)
    const idx = items.value.findIndex(i => i.id === kid)
    if (idx >= 0) {
      items.value[idx] = updated
    }
    return updated
  }

  // 删除知识条目
  async function deleteItem(kid: number) {
    await knowledgeApi.delete(kid)
    items.value = items.value.filter(i => i.id !== kid)
  }

  // 确认待审核条目
  async function confirmItem(kid: number) {
    await knowledgeApi.confirm(kid)
    const idx = items.value.findIndex(i => i.id === kid)
    if (idx >= 0) {
      items.value[idx].status = 'confirmed'
    }
  }

  // 批量确认
  async function batchConfirm(novelId: number) {
    await knowledgeApi.batchConfirm(novelId)
    items.value.forEach(i => {
      if (i.status === 'pending') {
        i.status = 'confirmed'
      }
    })
  }

  // 搜索
  async function searchItems(novelId: number, keyword: string) {
    loading.value = true
    try {
      const data: any = await knowledgeApi.search(novelId, keyword)
      items.value = Array.isArray(data) ? data : []
    } finally {
      loading.value = false
    }
  }

  // AI 提取知识
  async function extractFromChapter(novelId: number, chapterId: number, modelName?: string) {
    extractPending.value = true
    extractChapterId.value = chapterId
    try {
      const data: any = await knowledgeApi.extract(novelId, {
        chapter_id: chapterId,
        model_name: modelName,
      })
      extractTaskId.value = data.task_id
      // 轮询 fallback
      stopExtractPolling()
      extractPollTimer = setInterval(async () => {
        if (!extractTaskId.value) { stopExtractPolling(); return }
        try {
          const task: any = await aiApi.getTask(extractTaskId.value)
          if (task.status === 'completed' || task.status === 'failed') {
            stopExtractPolling()
            handleExtractTaskUpdate({ task_id: task.id, status: task.status, error: task.error_msg })
          }
        } catch { /* 轮询失败忽略 */ }
      }, 3000)
      return data.task_id
    } catch (e) {
      extractPending.value = false
      extractChapterId.value = null
      throw e
    }
  }

  // 解析 AI 提取结果
  async function parseExtractResult(novelId: number, chapterId: number, taskId: number) {
    try {
      const data: any = await knowledgeApi.parseExtract(novelId, {
        chapter_id: chapterId,
        task_id: taskId,
      })
      // 将新提取的条目追加到列表
      const newItems = Array.isArray(data) ? data : []
      items.value.push(...newItems)
      return newItems
    } finally {
      extractPending.value = false
      extractTaskId.value = null
      extractChapterId.value = null
    }
  }

  // 处理 WebSocket 推送的任务完成通知
  function handleExtractTaskUpdate(data: { task_id: number; status: string; result?: any; error?: string }) {
    if (!extractTaskId.value || data.task_id !== extractTaskId.value) return
    stopExtractPolling()

    if (data.status === 'completed') {
      // 任务完成，自动调用 parseExtractResult 解析并入库
      const novelId = currentNovelId.value
      const chapterId = extractChapterId.value
      const taskId = extractTaskId.value
      if (novelId && chapterId && taskId) {
        parseExtractResult(novelId, chapterId, taskId)
          .then((newItems) => {
            if (newItems && newItems.length > 0) {
              ElMessage.success(`AI 提取了 ${newItems.length} 条知识`)
            } else {
              ElMessage.info('AI 未从该章节提取到新知识')
            }
          })
          .catch(() => {
            ElMessage.error('解析提取结果失败')
            extractPending.value = false
            extractTaskId.value = null
            extractChapterId.value = null
          })
      }
    } else if (data.status === 'failed') {
      extractPending.value = false
      extractTaskId.value = null
      extractChapterId.value = null
      ElMessage.error(data.error || '知识提取失败')
    }
  }

  // 清空状态
  function clear() {
    stopExtractPolling()
    items.value = []
    currentNovelId.value = null
    activeCategory.value = ''
    extractPending.value = false
    extractTaskId.value = null
    extractChapterId.value = null
  }

  return {
    items,
    loading,
    currentNovelId,
    activeCategory,
    extractPending,
    extractTaskId,
    extractChapterId,
    groupedItems,
    pendingCount,
    confirmedCount,
    fetchItems,
    createItem,
    updateItem,
    deleteItem,
    confirmItem,
    batchConfirm,
    searchItems,
    extractFromChapter,
    parseExtractResult,
    handleExtractTaskUpdate,
    clear,
  }
})
