// web/src/store/novel.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { novelApi } from '@/api/novel'
import { aiApi } from '@/api/ai'
import type { Novel, Chapter, ChapterVersion, NovelSearchResult } from '@/api/novel'

export type { Novel, Chapter, ChapterVersion, NovelSearchResult } from '@/api/novel'

// 每个章节的 AI 任务状态
interface ChapterAIState {
  taskId: number
  pending: boolean
  result: string | null
  summary: string | null
  selRange: { start: number; end: number } | null // 划词区间，用于 Accept 时局部替换
}

export const useNovelStore = defineStore('novel', () => {
  // ========== 通用轮询工具 ==========
  const pollTimers = new Map<string, ReturnType<typeof setInterval>>()

  function startPoll(key: string, taskId: number, onResult: (task: any) => void) {
    stopPoll(key)
    pollTimers.set(key, setInterval(async () => {
      try {
        const task: any = await aiApi.getTask(taskId)
        if (task.status === 'completed' || task.status === 'failed') {
          stopPoll(key)
          onResult(task)
        }
      } catch { /* 轮询失败忽略，下次重试 */ }
    }, 3000))
  }

  function stopPoll(key: string) {
    const timer = pollTimers.get(key)
    if (timer) {
      clearInterval(timer)
      pollTimers.delete(key)
    }
  }
  const novels = ref<Novel[]>([])
  const currentNovel = ref<Novel | null>(null)
  const chapters = ref<Chapter[]>([])
  const currentChapter = ref<Chapter | null>(null)
  const versions = ref<ChapterVersion[]>([])
  const loading = ref(false)
  const aiPending = ref(false)
  const aiResult = ref<string | null>(null)
  const aiSummary = ref<string | null>(null)
  const pendingTaskId = ref<number | null>(null)

  // 划词操作时记录选中区间，用于 Accept 时精确替换
  const selectionRange = ref<{ start: number; end: number } | null>(null)

  // 按章节 ID 存储 AI 状态，支持多章节并发任务
  const chapterAIMap = ref<Map<number, ChapterAIState>>(new Map())
  // taskId → chapterId 反向映射，用于 WebSocket 推送时快速定位
  const taskChapterMap = ref<Map<number, number>>(new Map())

  // ========== 热门小说搜索状态 ==========
  const searchResults = ref<NovelSearchResult[]>([])
  const searchLoading = ref(false)
  const searchWarning = ref('')
  const searchSource = ref('')  // 数据来源标识（bing/baidu/fanqie/ai）

  // ========== 大纲生成状态 ==========
  const outlineChapters = ref<{ title: string; summary: string }[]>([])
  const outlinePending = ref(false)
  const outlineTaskId = ref<number | null>(null)

  // ========== 大纲章节级 AI 操作状态 ==========
  const outlineAIPending = ref<number | null>(null) // 正在执行的章节索引，null 表示无
  const outlineAIResult = ref<{ title?: string; summary?: string; characters?: string } | null>(null)
  const outlineAITaskId = ref<number | null>(null)

  // ========== 人物生成状态 ==========
  const characterGenPending = ref(false)
  const characterGenResult = ref<string | null>(null)
  const characterGenTaskId = ref<number | null>(null)

  // ========== 扩写章节目录状态 ==========
  const expandPending = ref(false)
  const expandTaskId = ref<number | null>(null)
  const expandResult = ref<{ title: string; summary: string }[]>([])
  const expandMode = ref<'append' | 'insert'>('append')
  const expandInsertAfter = ref(0)

  // ========== Token 使用状态 ==========
  const tokenBudget = ref(0)
  const tokenUsed = ref(0)
  const tokenPercentage = ref(0)

  // ========== Novel 操作 ==========

  async function fetchNovels(portfolioId: number) {
    loading.value = true
    try {
      const data: any = await novelApi.list(portfolioId)
      novels.value = Array.isArray(data) ? data : []
    } finally {
      loading.value = false
    }
  }

  async function fetchNovel(id: number) {
    const data: any = await novelApi.get(id)
    currentNovel.value = data
  }

  async function createNovel(portfolioId: number, title: string, description?: string) {
    const data: any = await novelApi.create({ portfolio_id: portfolioId, title, description })
    novels.value.unshift(data)
    return data
  }

  async function updateNovel(id: number, payload: Partial<{ title: string; description: string; status: string }>) {
    const data: any = await novelApi.update(id, payload)
    currentNovel.value = data
    const idx = novels.value.findIndex((n) => n.id === id)
    if (idx !== -1) novels.value[idx] = data
  }

  async function deleteNovel(id: number) {
    await novelApi.delete(id)
    novels.value = novels.value.filter((n) => n.id !== id)
    if (currentNovel.value?.id === id) currentNovel.value = null
  }

  // ========== Chapter 操作 ==========

  async function fetchChapters(novelId: number) {
    loading.value = true
    try {
      const data: any = await novelApi.listChapters(novelId)
      chapters.value = Array.isArray(data) ? data : []
    } finally {
      loading.value = false
    }
  }

  function selectChapter(chapter: Chapter | null) {
    // 切换前：将当前章节的 selectionRange 保存到 chapterAIMap
    if (currentChapter.value) {
      const prevState = chapterAIMap.value.get(currentChapter.value.id)
      if (prevState) {
        prevState.selRange = selectionRange.value
      }
    }

    currentChapter.value = chapter
    // 从 Map 中恢复当前章节的 AI 状态
    if (chapter) {
      const state = chapterAIMap.value.get(chapter.id)
      if (state) {
        aiPending.value = state.pending
        aiResult.value = state.result
        aiSummary.value = state.summary
        pendingTaskId.value = state.taskId
        selectionRange.value = state.selRange
        return
      }
    }
    // 无状态时清空
    aiPending.value = false
    aiResult.value = null
    aiSummary.value = null
    pendingTaskId.value = null
    selectionRange.value = null
  }

  async function createChapter(novelId: number, title: string, summary?: string) {
    const data: any = await novelApi.createChapter(novelId, { title, summary })
    chapters.value.push(data)
    return data
  }

  async function updateChapter(id: number, payload: { title?: string; summary?: string; content?: string }) {
    const data: any = await novelApi.updateChapter(id, payload)
    const idx = chapters.value.findIndex((c) => c.id === id)
    if (idx !== -1) chapters.value[idx] = data
    if (currentChapter.value?.id === id) currentChapter.value = data
    return data
  }

  async function deleteChapter(id: number) {
    await novelApi.deleteChapter(id)
    chapters.value = chapters.value.filter((c) => c.id !== id)
    if (currentChapter.value?.id === id) currentChapter.value = null
  }

  async function reorderChapters(novelId: number, chapterIds: number[]) {
    await novelApi.reorderChapters(novelId, chapterIds)
    await fetchChapters(novelId)
  }

  // ========== AI 操作 ==========

  async function submitAIAction(chapterId: number, action: string, modelName?: string, summary?: string, content?: string, selectedText?: string, selRange?: { start: number; end: number } | null, scenePresetId?: number, polishMode?: string) {
    aiPending.value = true
    aiResult.value = null
    aiSummary.value = null
    selectionRange.value = selRange || null
    try {
      const data: any = await novelApi.chapterAIAction(chapterId, { action, model_name: modelName, summary, content, selected_text: selectedText, scene_preset_id: scenePresetId, polish_mode: polishMode })
      pendingTaskId.value = data.task_id
      // 写入 Map 和反向映射
      chapterAIMap.value.set(chapterId, { taskId: data.task_id, pending: true, result: null, summary: null, selRange: selRange || null })
      taskChapterMap.value.set(data.task_id, chapterId)
      // 轮询 fallback
      startPoll(`chapter_ai_${data.task_id}`, data.task_id, (task) => {
        const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
        handleTaskUpdate({ task_id: task.id, status: task.status, result, error: task.error_msg })
      })
      return data.task_id
    } catch {
      aiPending.value = false
      throw new Error('AI 操作失败')
    }
  }

  // 处理 WebSocket 推送的任务更新
  function handleTaskUpdate(data: { task_id: number; status: string; result?: any; error?: string }) {
    // 通过反向映射找到章节 ID
    const chapterId = taskChapterMap.value.get(data.task_id)
    if (!chapterId) return

    const state = chapterAIMap.value.get(chapterId)
    if (!state || state.taskId !== data.task_id) return

    stopPoll(`chapter_ai_${data.task_id}`)

    if (data.status === 'completed' && data.result) {
      state.pending = false
      state.result = data.result.content || ''
      state.summary = data.result.summary || ''
    } else if (data.status === 'failed') {
      state.pending = false
      state.result = null
      state.summary = null
      // 清理映射
      taskChapterMap.value.delete(data.task_id)
      chapterAIMap.value.delete(chapterId)
    }

    // 如果是当前选中的章节，同步视图状态
    if (currentChapter.value?.id === chapterId) {
      aiPending.value = state.pending
      aiResult.value = state.result
      aiSummary.value = state.summary
    }
  }

  async function acceptResult(chapterId: number) {
    if (!pendingTaskId.value) return
    await novelApi.acceptAIResult(chapterId, pendingTaskId.value)
    // 清理 Map
    taskChapterMap.value.delete(pendingTaskId.value)
    chapterAIMap.value.delete(chapterId)
    // 刷新章节数据
    const data: any = await novelApi.listChapters(currentNovel.value!.id)
    chapters.value = Array.isArray(data) ? data : []
    const updated = chapters.value.find((c) => c.id === chapterId)
    if (updated) currentChapter.value = updated
    // 清空 AI 预览
    discardResult()
  }

  function discardResult() {
    if (pendingTaskId.value) {
      const chapterId = taskChapterMap.value.get(pendingTaskId.value)
      if (chapterId) {
        // 上报 ai_reject 行为事件
        novelApi.rejectAIResult(chapterId, pendingTaskId.value).catch(() => {})
        chapterAIMap.value.delete(chapterId)
        taskChapterMap.value.delete(pendingTaskId.value)
      }
    }
    aiResult.value = null
    aiSummary.value = null
    pendingTaskId.value = null
    aiPending.value = false
    selectionRange.value = null
  }

  // 清理指定章节的概要 AI 状态（采纳/丢弃概要后调用，防止切换章节时残留）
  function clearChapterAISummary(chapterId: number) {
    const state = chapterAIMap.value.get(chapterId)
    if (state) {
      state.summary = null
      // 如果正文结果也没了，整条记录可以删掉
      if (!state.result && !state.pending) {
        chapterAIMap.value.delete(chapterId)
        if (state.taskId) taskChapterMap.value.delete(state.taskId)
      }
    }
  }

  // ========== 版本管理 ==========

  async function fetchVersions(chapterId: number) {
    const data: any = await novelApi.listVersions(chapterId)
    versions.value = Array.isArray(data) ? data : []
  }

  async function revertToVersion(chapterId: number, versionId: number) {
    await novelApi.revertVersion(chapterId, versionId)
    // 刷新章节和版本
    const chData: any = await novelApi.listChapters(currentNovel.value!.id)
    chapters.value = Array.isArray(chData) ? chData : []
    const updated = chapters.value.find((c) => c.id === chapterId)
    if (updated) currentChapter.value = updated
    await fetchVersions(chapterId)
  }

  // ========== 大纲生成操作 ==========

  async function submitOutlineGenerate(
    portfolioId: number, setting: string, characters: string,
    background: string, plot: string, chapterNum: number, modelName?: string, userPrompt?: string,
    structureTemplateId?: number, hitAnalysisId?: number, iterationTaskId?: number, feedback?: string,
  ) {
    outlinePending.value = true
    outlineChapters.value = []
    try {
      const data: any = await novelApi.generateOutline({
        portfolio_id: portfolioId, setting, characters, background, plot,
        chapter_num: chapterNum, model_name: modelName, user_prompt: userPrompt,
        structure_template_id: structureTemplateId, hit_analysis_id: hitAnalysisId,
        iteration_task_id: iterationTaskId, feedback,
      })
      outlineTaskId.value = data.task_id
      startPoll('outline', data.task_id, (task) => {
        const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
        handleOutlineTaskUpdate({ task_id: task.id, status: task.status, result, error: task.error_msg })
      })
      return data.task_id
    } catch {
      outlinePending.value = false
      throw new Error('大纲生成失败')
    }
  }

  function handleOutlineTaskUpdate(data: { task_id: number; status: string; result?: any; error?: string }) {
    if (data.task_id !== outlineTaskId.value) return
    stopPoll('outline')
    if (data.status === 'completed' && data.result?.chapters) {
      outlineChapters.value = data.result.chapters
      outlinePending.value = false
    } else if (data.status === 'failed') {
      outlinePending.value = false
      outlineChapters.value = []
      outlineTaskId.value = null
    }
  }

  async function adoptOutline(
    portfolioId: number, title: string, description: string,
    chapters: { title: string; summary: string }[],
  ) {
    if (!outlineTaskId.value) throw new Error('没有大纲任务')
    const data: any = await novelApi.adoptOutline({
      portfolio_id: portfolioId, task_id: outlineTaskId.value,
      title, description, chapters,
    })
    clearOutline()
    return data
  }

  function clearOutline() {
    stopPoll('outline')
    stopPoll('outline_chapter_ai')
    outlineChapters.value = []
    outlinePending.value = false
    outlineTaskId.value = null
    outlineAIPending.value = null
    outlineAIResult.value = null
    outlineAITaskId.value = null
  }

  // ========== 热门小说搜索 ==========

  async function searchNovels(keyword: string) {
    searchLoading.value = true
    searchWarning.value = ''
    searchSource.value = ''
    try {
      const data: any = await novelApi.searchNovels(keyword)
      searchResults.value = data.results || []
      if (data.source) {
        searchSource.value = data.source
      }
      if (data.warning) {
        searchWarning.value = data.warning
      }
    } catch {
      searchResults.value = []
      searchWarning.value = '搜索失败，请稍后重试'
    } finally {
      searchLoading.value = false
    }
  }

  function clearSearchResults() {
    searchResults.value = []
    searchWarning.value = ''
    searchSource.value = ''
  }

  // ========== 扩写章节目录 ==========

  async function submitExpandChapters(
    novelId: number,
    mode: 'append' | 'insert',
    insertAfter: number,
    chapterNum: number,
    modelName?: string,
    userPrompt?: string,
  ) {
    expandPending.value = true
    expandResult.value = []
    expandMode.value = mode
    expandInsertAfter.value = insertAfter
    try {
      const data: any = await novelApi.expandChapters(novelId, {
        mode,
        insert_after: mode === 'insert' ? insertAfter : undefined,
        chapter_num: chapterNum,
        model_name: modelName,
        user_prompt: userPrompt || undefined,
      })
      expandTaskId.value = data.task_id
      startPoll('expand', data.task_id, (task) => {
        const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
        handleExpandTaskUpdate({ task_id: task.id, status: task.status, result })
      })
      return data.task_id
    } catch {
      expandPending.value = false
      throw new Error('扩写章节目录请求失败')
    }
  }

  function handleExpandTaskUpdate(data: { task_id: number; status: string; result?: any }) {
    if (data.task_id !== expandTaskId.value) return
    stopPoll('expand')
    if (data.status === 'completed' && data.result?.chapters) {
      expandResult.value = data.result.chapters
      expandPending.value = false
    } else if (data.status === 'failed') {
      expandPending.value = false
      expandResult.value = []
      expandTaskId.value = null
    }
  }

  function clearExpandResult() {
    stopPoll('expand')
    expandPending.value = false
    expandTaskId.value = null
    expandResult.value = []
    expandMode.value = 'append'
    expandInsertAfter.value = 0
  }

  // ========== 大纲章节级 AI 操作 ==========

  async function submitOutlineChapterAI(
    portfolioId: number,
    index: number,
    action: string,
    chapter: { title: string; summary: string },
    context?: { setting?: string; prev_chapters?: { title: string; summary: string }[]; next_chapters?: { title: string; summary: string }[] },
    modelName?: string,
    userPrompt?: string,
  ) {
    outlineAIPending.value = index
    outlineAIResult.value = null
    try {
      const data: any = await novelApi.outlineChapterAI({
        portfolio_id: portfolioId,
        action,
        title: chapter.title,
        summary: chapter.summary,
        context,
        model_name: modelName,
        user_prompt: userPrompt,
      })
      outlineAITaskId.value = data.task_id
      startPoll('outline_chapter_ai', data.task_id, (task) => {
        const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
        handleOutlineChapterAIUpdate({ task_id: task.id, status: task.status, result, error: task.error_msg })
      })
      return data.task_id
    } catch {
      outlineAIPending.value = null
      throw new Error('大纲章节 AI 操作失败')
    }
  }

  function handleOutlineChapterAIUpdate(data: { task_id: number; status: string; result?: any; error?: string }) {
    if (data.task_id !== outlineAITaskId.value) return
    stopPoll('outline_chapter_ai')
    if (data.status === 'completed' && data.result) {
      outlineAIResult.value = data.result
      // 保持 outlineAIPending 不清空，用于标识哪个章节正在预览
    } else if (data.status === 'failed') {
      outlineAIPending.value = null
      outlineAIResult.value = null
      outlineAITaskId.value = null
    }
  }

  function acceptOutlineAIResult(_index: number) {
    if (outlineAIResult.value == null) return
    // 注意：这里不直接修改 editableChapters，由 OutlineWorkshop.vue 调用时处理
    const result = { ...outlineAIResult.value }
    discardOutlineAIResult()
    return result
  }

  function discardOutlineAIResult() {
    stopPoll('outline_chapter_ai')
    outlineAIPending.value = null
    outlineAIResult.value = null
    outlineAITaskId.value = null
  }

  // ========== 人物生成操作 ==========

  async function submitGenerateCharacters(
    portfolioId: number,
    setting: string,
    background: string,
    plot: string,
    modelName?: string,
    userPrompt?: string,
  ) {
    characterGenPending.value = true
    characterGenResult.value = null
    try {
      const data: any = await novelApi.outlineChapterAI({
        portfolio_id: portfolioId,
        action: 'generate_characters',
        title: background,   // 复用 title 字段传递背景信息
        summary: plot,        // 复用 summary 字段传递剧情思路
        context: { setting },
        model_name: modelName,
        user_prompt: userPrompt,
      })
      characterGenTaskId.value = data.task_id
      startPoll('character_gen', data.task_id, (task) => {
        const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
        handleCharacterGenUpdate({ task_id: task.id, status: task.status, result, error: task.error_msg })
      })
      return data.task_id
    } catch {
      characterGenPending.value = false
      throw new Error('人物生成请求失败')
    }
  }

  function handleCharacterGenUpdate(data: { task_id: number; status: string; result?: any; error?: string }) {
    if (data.task_id !== characterGenTaskId.value) return
    stopPoll('character_gen')
    if (data.status === 'completed' && data.result?.characters) {
      characterGenResult.value = data.result.characters
      characterGenPending.value = false
    } else if (data.status === 'failed') {
      characterGenPending.value = false
      characterGenResult.value = null
      characterGenTaskId.value = null
    }
  }

  function acceptCharacterGenResult() {
    const result = characterGenResult.value
    discardCharacterGenResult()
    return result
  }

  function discardCharacterGenResult() {
    stopPoll('character_gen')
    characterGenPending.value = false
    characterGenResult.value = null
    characterGenTaskId.value = null
  }

  // 判断指定章节是否有进行中的 AI 任务
  function isChapterAIPending(chapterId: number): boolean {
    const state = chapterAIMap.value.get(chapterId)
    return !!state?.pending
  }

  // ========== Token 使用操作 ==========

  async function fetchTokenUsage(novelId: number) {
    try {
      const data: any = await novelApi.getTokenUsage(novelId)
      tokenBudget.value = data.budget ?? 0
      tokenUsed.value = data.used ?? 0
      tokenPercentage.value = data.percentage ?? 0
    } catch { /* 忽略 */ }
  }

  async function updateTokenBudget(novelId: number, budget: number) {
    await novelApi.updateTokenBudget(novelId, budget)
    tokenBudget.value = budget
    // 重新计算百分比
    if (budget > 0) {
      tokenPercentage.value = Math.min((tokenUsed.value / budget) * 100, 100)
    } else {
      tokenPercentage.value = 0
    }
  }

  // WebSocket 推送处理
  function handleTokenUpdate(data: { novel_id: number; budget: number; used: number; percentage: number }) {
    if (currentNovel.value && data.novel_id === currentNovel.value.id) {
      tokenBudget.value = data.budget ?? 0
      tokenUsed.value = data.used ?? 0
      tokenPercentage.value = data.percentage ?? 0
    }
  }

  return {
    novels,
    currentNovel,
    chapters,
    currentChapter,
    versions,
    loading,
    aiPending,
    aiResult,
    aiSummary,
    pendingTaskId,
    selectionRange,
    searchResults,
    searchLoading,
    searchWarning,
    searchSource,
    outlineChapters,
    outlinePending,
    outlineTaskId,
    outlineAIPending,
    outlineAIResult,
    outlineAITaskId,
    expandPending,
    expandTaskId,
    expandResult,
    expandMode,
    expandInsertAfter,
    fetchNovels,
    fetchNovel,
    createNovel,
    updateNovel,
    deleteNovel,
    fetchChapters,
    selectChapter,
    createChapter,
    updateChapter,
    deleteChapter,
    reorderChapters,
    submitAIAction,
    handleTaskUpdate,
    acceptResult,
    discardResult,
    clearChapterAISummary,
    isChapterAIPending,
    fetchVersions,
    revertToVersion,
    submitOutlineGenerate,
    handleOutlineTaskUpdate,
    handleOutlineChapterAIUpdate,
    adoptOutline,
    clearOutline,
    searchNovels,
    clearSearchResults,
    submitOutlineChapterAI,
    acceptOutlineAIResult,
    discardOutlineAIResult,
    characterGenPending,
    characterGenResult,
    characterGenTaskId,
    submitGenerateCharacters,
    handleCharacterGenUpdate,
    acceptCharacterGenResult,
    discardCharacterGenResult,
    submitExpandChapters,
    handleExpandTaskUpdate,
    clearExpandResult,
    tokenBudget,
    tokenUsed,
    tokenPercentage,
    fetchTokenUsage,
    updateTokenBudget,
    handleTokenUpdate,
  }
})
