// web/src/composables/useBatchGenerate.ts
// 批量生成空白章节 composable — 串行调度 + 上下文构建
import { ref, computed, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import type { Chapter } from '@/api/novel'
import { useNovelStore } from '@/store/novel'
import { useWorkflowStore, isCompleted } from '@/store/workflow'

// 单个批量任务状态
export interface BatchTask {
  chapterId: number
  title: string
  sortOrder: number
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress: number // 当前章节工作流进度 0-100
  error: string | null
}

// 批量生成整体状态
export interface BatchState {
  active: boolean
  tasks: BatchTask[]
  currentIndex: number // 当前执行到的任务索引，-1 表示未开始
}

// 队列上限
const MAX_BATCH_SIZE = 20
// localStorage 过期时间 24 小时
const BATCH_EXPIRE_MS = 24 * 60 * 60 * 1000

/**
 * 批量生成空白章节 composable
 * @param portfolioId - 作品集 ID（响应式 getter）
 * @param selectedModel - 当前选中的模型名（响应式 getter）
 * @param novelId - 小说 ID（响应式 getter，用于传递写作风格上下文）
 */
export function useBatchGenerate(
  portfolioId: () => number,
  selectedModel: () => string,
  novelId?: () => number,
) {
  const novelStore = useNovelStore()
  const workflowStore = useWorkflowStore()

  const batchState = ref<BatchState>({
    active: false,
    tasks: [],
    currentIndex: -1,
  })

  // 内部取消标志
  let cancelled = false

  // ========== 计算属性 ==========

  // 整体进度：已完成（含失败）/ 总数
  const batchProgress = computed(() => {
    const tasks = batchState.value.tasks
    if (tasks.length === 0) return 0
    const done = tasks.filter(t =>
      t.status === 'completed' || t.status === 'failed' || t.status === 'cancelled',
    ).length
    return Math.round((done / tasks.length) * 100)
  })

  // 已完成数量
  const completedCount = computed(() =>
    batchState.value.tasks.filter(t => t.status === 'completed').length,
  )

  // 失败数量
  const failedCount = computed(() =>
    batchState.value.tasks.filter(t => t.status === 'failed').length,
  )

  // 是否全部结束
  const isFinished = computed(() =>
    batchState.value.active
    && batchState.value.tasks.length > 0
    && batchState.value.tasks.every(t =>
      t.status === 'completed' || t.status === 'failed' || t.status === 'cancelled',
    ),
  )

  // ========== 上下文构建 ==========

  /**
   * 构建前情提要上下文
   * 取当前章节之前最近 3 章的 title + summary，拼接为结构化文本
   */
  function buildContext(chapter: Chapter): string {
    const allChapters = [...novelStore.chapters].sort((a, b) => a.sort_order - b.sort_order)
    const idx = allChapters.findIndex(c => c.id === chapter.id)

    // 取前面最近 3 章（优先用已生成的新内容）
    const prevChapters = allChapters.slice(Math.max(0, idx - 3), idx)
    const parts: string[] = []

    if (prevChapters.length > 0) {
      parts.push('【前情提要】')
      for (const prev of prevChapters) {
        const summary = (prev.summary || '').slice(0, 500)
        parts.push(`第${prev.sort_order}章「${prev.title}」：${summary}`)
      }
      parts.push('')
    }

    // 本章概要
    const chapterSummary = chapter.summary || ''
    if (chapterSummary) {
      parts.push('【本章概要】')
      parts.push(chapterSummary)
    }

    return parts.join('\n')
  }

  // ========== 串行调度 ==========

  /**
   * 启动批量生成
   * 筛选空白章节，按 sort_order 排序，最多取 MAX_BATCH_SIZE 章
   */
  function startBatch(): boolean {
    if (batchState.value.active) {
      ElMessage.warning('批量生成正在进行中')
      return false
    }
    if (workflowStore.pending) {
      ElMessage.warning('当前有工作流正在执行，请等待完成后再试')
      return false
    }

    // 筛选空白章节：content 为空或纯空白
    const emptyChapters = [...novelStore.chapters]
      .filter(ch => !ch.content || ch.content.trim() === '')
      .sort((a, b) => a.sort_order - b.sort_order)
      .slice(0, MAX_BATCH_SIZE)

    if (emptyChapters.length === 0) {
      ElMessage.info('没有空白章节需要生成')
      return false
    }

    // 初始化任务列表
    cancelled = false
    batchState.value = {
      active: true,
      currentIndex: -1,
      tasks: emptyChapters.map(ch => ({
        chapterId: ch.id,
        title: ch.title,
        sortOrder: ch.sort_order,
        status: 'pending',
        progress: 0,
        error: null,
      })),
    }

    ElMessage.success(`已加入 ${emptyChapters.length} 个空白章节到生成队列`)
    // 持久化到 localStorage，用于页面刷新后恢复
    saveBatchToStorage()
    executeNext()
    return true
  }

  /**
   * 执行下一个待处理任务
   */
  async function executeNext() {
    if (cancelled) {
      // 将剩余 pending 任务标记为 cancelled
      for (const task of batchState.value.tasks) {
        if (task.status === 'pending') {
          task.status = 'cancelled'
        }
      }
      clearBatchStorage()
      const cancelledCount = batchState.value.tasks.filter(t => t.status === 'cancelled').length
      if (cancelledCount > 0) {
        ElMessage.success(
          `批量生成已停止：${completedCount.value} 成功，${cancelledCount} 已取消`,
        )
      }
      return
    }

    // 找到下一个 pending 任务
    const nextIdx = batchState.value.tasks.findIndex(t => t.status === 'pending')
    if (nextIdx === -1) {
      // 全部完成
      clearBatchStorage()
      ElMessage.success(
        `批量生成完成：${completedCount.value} 成功，${failedCount.value} 失败`,
      )
      return
    }

    batchState.value.currentIndex = nextIdx
    const task = batchState.value.tasks[nextIdx]
    task.status = 'running'
    task.progress = 0

    // 从 chapters 列表中获取最新的章节数据（可能前面的章节已更新了 summary）
    const chapter = novelStore.chapters.find(c => c.id === task.chapterId)
    if (!chapter) {
      task.status = 'failed'
      task.error = '章节不存在'
      executeNext()
      return
    }

    try {
      const context = buildContext(chapter)
      const params: Record<string, any> = {
        title: chapter.title,
        background: context,
      }
      if (novelId) {
        params.novel_id = novelId()
      }
      await workflowStore.submitWorkflow(
        portfolioId(),
        'full_chapter',
        selectedModel(),
        params,
      )
    } catch (e: any) {
      task.status = 'failed'
      task.error = e?.message || '提交工作流失败'
      // 继续下一个
      executeNext()
    }
  }

  /**
   * 取消批量生成
   * 当前正在生成的章节会继续完成，后续 pending 章节停止
   */
  async function cancelBatch() {
    if (!batchState.value.active) return

    cancelled = true
    clearBatchStorage()

    // 只标记 pending 任务为 cancelled，当前 running 任务继续完成
    for (const task of batchState.value.tasks) {
      if (task.status === 'pending') {
        task.status = 'cancelled'
      }
    }

    // 如果当前没有 running 任务（极端情况），直接提示完成
    const hasRunning = batchState.value.tasks.some(t => t.status === 'running')
    if (hasRunning) {
      ElMessage.info('已取消后续章节，当前章节将继续完成')
    } else {
      ElMessage.info('批量生成已取消')
    }
  }

  /**
   * 关闭/重置批量面板（仅在全部结束后可调用）
   */
  function resetBatch() {
    batchState.value = {
      active: false,
      tasks: [],
      currentIndex: -1,
    }
    cancelled = false
    clearBatchStorage()
  }

  // ========== localStorage 持久化 ==========

  function getStorageKey() {
    return novelId ? `batch_${novelId()}` : ''
  }

  function saveBatchToStorage() {
    const key = getStorageKey()
    if (!key) return
    const chapterIds = batchState.value.tasks.map(t => t.chapterId)
    localStorage.setItem(key, JSON.stringify({ chapterIds, startedAt: Date.now() }))
  }

  function clearBatchStorage() {
    const key = getStorageKey()
    if (key) localStorage.removeItem(key)
  }

  function loadBatchFromStorage(): { chapterIds: number[]; startedAt: number } | null {
    const key = getStorageKey()
    if (!key) return null
    try {
      const raw = localStorage.getItem(key)
      if (!raw) return null
      const data = JSON.parse(raw)
      // 过期检查
      if (Date.now() - data.startedAt > BATCH_EXPIRE_MS) {
        localStorage.removeItem(key)
        return null
      }
      return data
    } catch {
      return null
    }
  }

  // ========== 恢复批量生成 ==========

  /**
   * 从活跃工作流恢复批量生成状态
   * @param activeWorkflows - listActive API 返回的活跃工作流列表
   * @returns 是否成功恢复
   */
  function recoverBatch(activeWorkflows: any[]): boolean {
    const stored = loadBatchFromStorage()
    if (!stored || stored.chapterIds.length === 0) return false

    // 用保存的 chapterIds 重建任务列表
    const tasks: BatchTask[] = stored.chapterIds.map(cid => {
      const chapter = novelStore.chapters.find(c => c.id === cid)
      return {
        chapterId: cid,
        title: chapter?.title || `章节 ${cid}`,
        sortOrder: chapter?.sort_order || 0,
        status: 'pending' as const,
        progress: 0,
        error: null,
      }
    })

    // 根据章节内容判断已完成的任务
    for (const task of tasks) {
      const chapter = novelStore.chapters.find(c => c.id === task.chapterId)
      if (chapter?.content && chapter.content.trim() !== '') {
        task.status = 'completed'
        task.progress = 100
      }
    }

    // 检查是否有 full_chapter 类型的活跃工作流
    const fullChapterWf = activeWorkflows.find(wf => wf.workflow_type === 'full_chapter')
    if (fullChapterWf) {
      // 找到第一个还是 pending 的任务，标记为 running
      const runningTask = tasks.find(t => t.status === 'pending')
      if (runningTask) {
        runningTask.status = 'running'
        // 恢复工作流到 workflowStore，启动轮询
        workflowStore.recoverFromWorkflow(fullChapterWf)
      }
    }

    // 如果所有任务都已完成，清除存储并不恢复
    const hasPendingOrRunning = tasks.some(t => t.status === 'pending' || t.status === 'running')
    if (!hasPendingOrRunning) {
      clearBatchStorage()
      return false
    }

    // 恢复批量状态
    cancelled = false
    const currentIdx = tasks.findIndex(t => t.status === 'running')
    batchState.value = {
      active: true,
      tasks,
      currentIndex: currentIdx >= 0 ? currentIdx : -1,
    }

    // 如果没有 running 任务但有 pending 任务，说明上一个已完成但还没推进下一个
    if (currentIdx === -1 && !fullChapterWf) {
      executeNext()
    }

    return true
  }

  // ========== 监听工作流状态变化 ==========

  watch(() => workflowStore.currentWorkflow?.status, async (status) => {
    if (!batchState.value.active) return

    const currentTask = batchState.value.tasks[batchState.value.currentIndex]
    if (!currentTask || currentTask.status !== 'running') return

    if (isCompleted(status || '') && workflowStore.currentWorkflow?.result_json) {
      try {
        const result = JSON.parse(workflowStore.currentWorkflow.result_json)
        const finalResult = result.final_result
        const content = typeof finalResult === 'string' ? finalResult : finalResult?.content

        if (content) {
          // 保存生成的内容到章节
          const chapter = novelStore.chapters.find(c => c.id === currentTask.chapterId)
          await novelStore.updateChapter(currentTask.chapterId, {
            title: chapter?.title || currentTask.title,
            summary: chapter?.summary || '',
            content,
          })
        }

        currentTask.status = 'completed'
        currentTask.progress = 100
      } catch (e) {
        console.error('批量生成：解析工作流结果失败', e)
        currentTask.status = 'failed'
        currentTask.error = '解析结果失败'
      }

      // 重置工作流状态，推进下一个
      await nextTick()
      workflowStore.reset()
      executeNext()
    } else if (status === 'failed') {
      currentTask.status = 'failed'
      currentTask.error = workflowStore.currentWorkflow?.error_msg || '工作流执行失败'
      currentTask.progress = 0

      // 重置工作流状态，继续下一个
      await nextTick()
      workflowStore.reset()
      executeNext()
    } else if (status === 'cancelled') {
      // 被外部取消（cancelBatch 已处理状态）
      await nextTick()
      workflowStore.reset()
    }
  })

  // 监听工作流进度，更新当前任务的 progress
  watch(() => workflowStore.progress, (p) => {
    if (!batchState.value.active) return
    const currentTask = batchState.value.tasks[batchState.value.currentIndex]
    if (currentTask && currentTask.status === 'running') {
      currentTask.progress = Math.round(p * 100)
    }
  })

  return {
    batchState,
    batchProgress,
    completedCount,
    failedCount,
    isFinished,
    startBatch,
    cancelBatch,
    resetBatch,
    recoverBatch,
  }
}
