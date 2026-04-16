// web/src/store/overview.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { overviewApi } from '@/api/overview'
import { aiApi } from '@/api/ai'
import { workflowApi } from '@/api/workflow'
import type { OverviewData, OverviewChange, CharacterRelation, ChapterBrief } from '@/api/overview'
import type { NovelKnowledge } from '@/api/knowledge'
import { ElMessage } from 'element-plus'

const POLL_INTERVAL = 3000    // 轮询间隔 3s
const POLL_TIMEOUT = 360000   // 超时 3min
const EXTRACT_EXPIRE_MS = 24 * 60 * 60 * 1000 // localStorage 过期 24h

export const useOverviewStore = defineStore('overview', () => {
  // 元数据
  const plotlines = ref<NovelKnowledge[]>([])
  const characters = ref<NovelKnowledge[]>([])
  const foreshadows = ref<NovelKnowledge[]>([])
  const relations = ref<CharacterRelation[]>([])
  const chapters = ref<ChapterBrief[]>([])

  // 变更追踪
  const pendingChanges = ref<OverviewChange[]>([])

  // 工作流状态
  const revisionPlan = ref<string>('')
  const revisionWorkflowId = ref<number | null>(null)
  const revisionPending = ref(false)
  const executePending = ref(false)
  const extractPending = ref(false)
  const extractTaskId = ref<number | null>(null)
  const extractNovelId = ref<number | null>(null)
  let pollTimer: ReturnType<typeof setTimeout> | null = null
  let pollStartTime = 0
  let extractProcessing = false  // 防止 onExtractCompleted 重入
  let revisionPollTimer: ReturnType<typeof setInterval> | null = null

  const loading = ref(false)

  // 是否有未提交的变更
  const hasChanges = computed(() => pendingChanges.value.length > 0)

  // 获取总览数据
  async function fetchOverview(novelId: number, silent = false) {
    if (!silent) loading.value = true
    try {
      const data = (await overviewApi.get(novelId)) as unknown as OverviewData
      plotlines.value = data.plotlines || []
      characters.value = data.characters || []
      foreshadows.value = data.foreshadows || []
      relations.value = data.relations || []
      chapters.value = data.chapters || []
    } finally {
      loading.value = false
    }
    // 恢复提取中的任务
    recoverExtract(novelId)
  }

  // 从 localStorage 恢复提取任务状态
  function recoverExtract(novelId: number) {
    if (extractPending.value) return // 已有提取任务在进行
    try {
      const raw = localStorage.getItem(`extract_${novelId}`)
      if (!raw) return
      const data = JSON.parse(raw)
      // 过期检查
      if (Date.now() - data.startedAt > EXTRACT_EXPIRE_MS) {
        localStorage.removeItem(`extract_${novelId}`)
        return
      }
      // 恢复状态并启动轮询
      extractTaskId.value = data.taskId
      extractNovelId.value = novelId
      extractPending.value = true
      startPolling()
    } catch {
      localStorage.removeItem(`extract_${novelId}`)
    }
  }

  // 记录变更
  function addChange(change: OverviewChange) {
    pendingChanges.value.push(change)
  }

  // 清空变更
  function clearChanges() {
    pendingChanges.value = []
  }

  // 创建人物关系
  async function createRelation(novelId: number, data: {
    from_knowledge_id: number
    to_knowledge_id: number
    relation_type: string
    label?: string
    chapter_ref?: string
  }) {
    const r = (await overviewApi.createRelation(novelId, data)) as unknown as CharacterRelation
    relations.value.push(r)
    return r
  }

  // 更新人物关系
  async function updateRelation(novelId: number, rid: number, data: {
    relation_type?: string
    label?: string
    chapter_ref?: string
  }) {
    const r = (await overviewApi.updateRelation(novelId, rid, data)) as unknown as CharacterRelation
    const idx = relations.value.findIndex(i => i.id === rid)
    if (idx >= 0) relations.value[idx] = r
    return r
  }

  // 删除人物关系
  async function deleteRelation(novelId: number, rid: number) {
    await overviewApi.deleteRelation(novelId, rid)
    relations.value = relations.value.filter(i => i.id !== rid)
  }

  // AI 提取总览
  async function extractOverview(novelId: number, modelName?: string) {
    extractPending.value = true
    extractNovelId.value = novelId
    try {
      const data: any = await overviewApi.extract(novelId, { model_name: modelName })
      extractTaskId.value = data.task_id
      // 持久化到 localStorage，用于页面刷新后恢复
      localStorage.setItem(`extract_${novelId}`, JSON.stringify({
        taskId: data.task_id,
        startedAt: Date.now(),
      }))
      // 启动轮询兜底（WebSocket 可能丢消息）
      startPolling()
      return data.task_id
    } catch (e) {
      extractPending.value = false
      extractNovelId.value = null
      throw e
    }
  }

  // 轮询检查任务状态
  function startPolling() {
    stopPolling()
    pollStartTime = Date.now()
    schedulePoll()
  }

  function stopPolling() {
    if (pollTimer) {
      clearTimeout(pollTimer)
      pollTimer = null
    }
  }

  function schedulePoll() {
    pollTimer = setTimeout(async () => {
      if (!extractTaskId.value || !extractNovelId.value) {
        stopPolling()
        return
      }
      // 超时保护
      if (Date.now() - pollStartTime > POLL_TIMEOUT) {
        extractPending.value = false
        extractTaskId.value = null
        extractNovelId.value = null
        ElMessage.error('提取任务超时，请重试')
        return
      }
      try {
        const task: any = await aiApi.getTask(extractTaskId.value)
        if (task.status === 'completed') {
          await onExtractCompleted()
        } else if (task.status === 'failed') {
          onExtractFailed()
        } else {
          schedulePoll() // 继续轮询
        }
      } catch {
        schedulePoll() // 查询失败也继续轮询
      }
    }, POLL_INTERVAL)
  }

  // 提取完成：解析结果并刷新数据
  async function onExtractCompleted() {
    if (extractProcessing) return  // 防重入
    extractProcessing = true
    stopPolling()
    const novelId = extractNovelId.value
    const taskId = extractTaskId.value
    if (!novelId || !taskId) {
      extractProcessing = false
      return
    }

    try {
      await overviewApi.parseExtract(novelId, taskId)
      await fetchOverview(novelId, true)  // silent 模式，不触发 loading 闪烁
      ElMessage.success('总览提取完成')
    } catch {
      ElMessage.error('提取结果解析失败')
    } finally {
      extractPending.value = false
      extractTaskId.value = null
      if (extractNovelId.value) {
        localStorage.removeItem(`extract_${extractNovelId.value}`)
      }
      extractNovelId.value = null
      extractProcessing = false
    }
  }

  // 提取失败
  function onExtractFailed() {
    stopPolling()
    extractPending.value = false
    extractTaskId.value = null
    if (extractNovelId.value) {
      localStorage.removeItem(`extract_${extractNovelId.value}`)
    }
    extractNovelId.value = null
    ElMessage.error('总览提取失败')
  }

  // WebSocket 回调（作为加速器，收到后立即处理，跳过轮询等待）
  async function handleExtractTaskUpdate(data: { task_id: number; status: string }) {
    if (!extractTaskId.value || data.task_id !== extractTaskId.value) return

    if (data.status === 'completed') {
      await onExtractCompleted()
    } else if (data.status === 'failed') {
      onExtractFailed()
    }
  }

  function stopRevisionPolling() {
    if (revisionPollTimer) {
      clearInterval(revisionPollTimer)
      revisionPollTimer = null
    }
  }

  // 提交变更（触发分析工作流）
  async function submitRevision(novelId: number, portfolioId: number, modelName?: string) {
    if (pendingChanges.value.length === 0) {
      ElMessage.warning('没有待提交的变更')
      return
    }
    revisionPending.value = true
    try {
      const data: any = await overviewApi.submitRevision(novelId, portfolioId, {
        model_name: modelName,
        changes: pendingChanges.value,
      })
      revisionWorkflowId.value = data.workflow_id
      // 轮询 fallback
      stopRevisionPolling()
      revisionPollTimer = setInterval(async () => {
        if (!revisionWorkflowId.value) { stopRevisionPolling(); return }
        try {
          const detail: any = await workflowApi.get(revisionWorkflowId.value)
          const wf = detail.workflow || detail
          if (wf.status === 'completed' || wf.status === 'failed') {
            stopRevisionPolling()
            handleRevisionWorkflowUpdate({
              id: wf.id,
              status: wf.status,
              result_json: wf.result_json,
            })
          }
        } catch { /* 轮询失败忽略 */ }
      }, 3000)
      return data.workflow_id
    } catch (e) {
      revisionPending.value = false
      throw e
    }
  }

  // 确认执行变更
  async function executeRevision(novelId: number, portfolioId: number, modelName?: string) {
    if (!revisionWorkflowId.value || !revisionPlan.value) {
      ElMessage.warning('缺少修改计划')
      return
    }
    executePending.value = true
    try {
      const data: any = await overviewApi.executeRevision(novelId, portfolioId, {
        model_name: modelName,
        workflow_id: revisionWorkflowId.value,
        revision_plan: revisionPlan.value,
      })
      return data.workflow_id
    } catch (e) {
      executePending.value = false
      throw e
    }
  }

  // 处理分析工作流完成（WebSocket 回调）
  function handleRevisionWorkflowUpdate(data: {
    id: number
    status: string
    result_json?: string
  }) {
    if (!revisionWorkflowId.value || data.id !== revisionWorkflowId.value) return
    stopRevisionPolling()

    if (data.status === 'completed' && data.result_json) {
      revisionPlan.value = data.result_json
      revisionPending.value = false
    } else if (data.status === 'failed') {
      revisionPending.value = false
      ElMessage.error('变更分析失败')
    }
  }

  function reset() {
    stopPolling()
    stopRevisionPolling()
    plotlines.value = []
    characters.value = []
    foreshadows.value = []
    relations.value = []
    chapters.value = []
    pendingChanges.value = []
    revisionPlan.value = ''
    revisionWorkflowId.value = null
    revisionPending.value = false
    executePending.value = false
    extractPending.value = false
    extractTaskId.value = null
    extractNovelId.value = null
    loading.value = false
  }

  return {
    plotlines,
    characters,
    foreshadows,
    relations,
    chapters,
    pendingChanges,
    revisionPlan,
    revisionWorkflowId,
    revisionPending,
    executePending,
    extractPending,
    extractTaskId,
    loading,
    hasChanges,
    fetchOverview,
    addChange,
    clearChanges,
    createRelation,
    updateRelation,
    deleteRelation,
    extractOverview,
    submitRevision,
    executeRevision,
    handleRevisionWorkflowUpdate,
    handleExtractTaskUpdate,
    reset,
  }
})
