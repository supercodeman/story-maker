// web/src/composables/useNovelButler.ts
// 小说管家状态机 — 5 步 AI 创作向导
import { computed, reactive, watch } from 'vue'
import { novelApi } from '@/api/novel'
import { aiApi } from '@/api/ai'
import { overviewApi } from '@/api/overview'
import { useNovelStore } from '@/store/novel'
import { useWorkflowStore } from '@/store/workflow'

// 步骤定义
export type ButlerStep = 'topic' | 'storyline' | 'characters' | 'chapters' | 'knowledge' | 'content'

const STEPS: ButlerStep[] = ['topic', 'storyline', 'characters', 'chapters', 'knowledge', 'content']

const STEP_LABELS: Record<ButlerStep, string> = {
  topic: '选题',
  storyline: '故事线',
  characters: '人物设计',
  chapters: '章节生成',
  knowledge: '知识图谱',
  content: '内容填充',
}

// 每步状态
export interface StepState {
  status: 'idle' | 'generating' | 'choosing' | 'confirmed'
  options: string[]       // 步骤1：4个方案文本
  selectedIndex: number   // 用户选择的方案索引
  result: string          // 确认后的结果文本
  userHint: string        // 用户输入的想法/引导
  error: string | null
  // 多轮迭代相关（步骤2/3）
  iterationId?: string         // 迭代 ID
  iterationRound?: number      // 当前轮次
  iterationMaxRounds?: number  // 最大轮次
  iterationPhase?: string      // generating/reviewing/completed/failed
}

// 管家整体状态
export interface ButlerState {
  active: boolean
  currentStep: ButlerStep
  autoMode: boolean
  steps: Record<ButlerStep, StepState>
  // 步骤4 特有：大纲章节列表
  chapterNum: number
  outlineChapters: { title: string; summary: string }[]
  outlineTaskId: number | null
  // 步骤5 特有：知识图谱提取进度
  knowledgeProgress: { phase: 'idle' | 'extracting' | 'parsing' | 'done'; taskId: number | null }
  // 步骤6 特有：内容生成进度
  contentProgress: { total: number; completed: number; current: string }
  // 用户最初输入的创作方向（与选题结果区分）
  originalSetting: string
  // 创建的小说信息
  createdNovelId: number | null
  // 管家会话 ID，串联同一次管家创作的所有任务
  butlerSessionId: string
}

function createStepState(): StepState {
  return { status: 'idle', options: [], selectedIndex: -1, result: '', userHint: '', error: null }
}

function createButlerState(): ButlerState {
  return {
    active: false,
    currentStep: 'topic',
    autoMode: false,
    steps: {
      topic: createStepState(),
      storyline: createStepState(),
      characters: createStepState(),
      chapters: createStepState(),
      knowledge: createStepState(),
      content: createStepState(),
    },
    chapterNum: 30,
    outlineChapters: [],
    outlineTaskId: null,
    knowledgeProgress: { phase: 'idle', taskId: null },
    contentProgress: { total: 0, completed: 0, current: '' },
    originalSetting: '',
    createdNovelId: null,
    butlerSessionId: '',
  }
}

/**
 * 小说管家 composable
 * @param portfolioId - 作品集 ID（响应式 getter）
 * @param selectedModel - 当前选中的模型名（响应式 getter）
 */
export function useNovelButler(
  portfolioId: () => number,
  selectedModel: () => string,
) {
  const novelStore = useNovelStore()
  const workflowStore = useWorkflowStore()

  const state = reactive<ButlerState>(createButlerState())

  // ========== 状态持久化 ==========

  const STORAGE_KEY = `butler-state-${portfolioId()}`

  let saveTimer: ReturnType<typeof setTimeout> | null = null

  /** 序列化 state 到 localStorage（debounce 500ms） */
  function saveState() {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => {
      try {
        const snapshot = {
          state: JSON.parse(JSON.stringify(state)),
          selectedModel: selectedModel(),
          started: true,
        }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot))
      } catch { /* 写入失败忽略 */ }
    }, 500)
  }

  /** 从 localStorage 恢复 state，返回是否恢复成功 */
  function restoreState(): boolean {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (!raw) return false
      const snapshot = JSON.parse(raw)
      if (!snapshot?.state?.active) return false

      const saved: ButlerState = snapshot.state

      // 恢复前处理：generating 状态重置为 idle（轮询已丢失）
      for (const step of STEPS) {
        if (saved.steps[step].status === 'generating') {
          saved.steps[step].status = 'idle'
          saved.steps[step].error = null
        }
      }
      // 知识图谱 extracting/parsing 也重置
      if (saved.knowledgeProgress.phase !== 'done' && saved.knowledgeProgress.phase !== 'idle') {
        saved.knowledgeProgress.phase = 'idle'
        saved.knowledgeProgress.taskId = null
        saved.steps.knowledge.status = 'idle'
      }

      Object.assign(state, saved)
      return true
    } catch {
      return false
    }
  }

  /** 清除 localStorage 中保存的状态 */
  function clearSavedState() {
    localStorage.removeItem(STORAGE_KEY)
    if (saveTimer) {
      clearTimeout(saveTimer)
      saveTimer = null
    }
  }

  // 轮询定时器映射
  const pollTimers = new Map<number, ReturnType<typeof setInterval>>()

  // 取消标志
  let cancelled = false

  // ========== 计算属性 ==========

  const currentStepIndex = computed(() => STEPS.indexOf(state.currentStep))
  const currentStepLabel = computed(() => STEP_LABELS[state.currentStep])
  const currentStepState = computed(() => state.steps[state.currentStep])
  const isAllDone = computed(() => state.steps.content.status === 'confirmed')

  // ========== 轮询工具 ==========

  function startTaskPoll(taskId: number, onResult: (task: any) => void) {
    stopTaskPoll(taskId)
    pollTimers.set(taskId, setInterval(async () => {
      try {
        const task: any = await aiApi.getTask(taskId)
        if (task.status === 'completed' || task.status === 'failed') {
          stopTaskPoll(taskId)
          onResult(task)
        }
      } catch { /* 轮询失败忽略 */ }
    }, 3000))
  }

  function stopTaskPoll(taskId: number) {
    const timer = pollTimers.get(taskId)
    if (timer) {
      clearInterval(timer)
      pollTimers.delete(taskId)
    }
  }

  function stopAllPolls() {
    pollTimers.forEach((timer) => clearInterval(timer))
    pollTimers.clear()
  }

  // ========== 核心方法 ==========

  /** 启动管家 */
  function startButler(setting: string) {
    Object.assign(state, createButlerState())
    state.active = true
    state.butlerSessionId = crypto.randomUUID()
    cancelled = false
    // 保存用户最初输入的创作方向（与后续选题结果区分）
    state.originalSetting = setting
    // 将用户输入的创作方向存入 topic 步骤的 result 作为种子
    state.steps.topic.result = setting
    // 自动模式直接开始生成，否则进入 idle 等用户输入想法
    if (state.autoMode) {
      generateOptions('topic', setting)
    } else {
      state.currentStep = 'topic'
      state.steps.topic.status = 'idle'
    }
  }

  /** 关闭管家 */
  function closeButler() {
    cancelled = true
    stopAllPolls()
    state.active = false
    clearSavedState()
  }

  /** 执行指定步骤 */
  async function runStep(step: ButlerStep, inputHint?: string) {
    if (cancelled) return
    state.currentStep = step
    const stepState = state.steps[step]

    if (step === 'chapters') {
      // 非自动模式：先等用户输入想法
      if (!state.autoMode && !inputHint) {
        stepState.status = 'idle'
        stepState.error = null
        return
      }
      await runChapterStep(inputHint)
      return
    }
    if (step === 'knowledge') {
      await runKnowledgeStep()
      return
    }
    if (step === 'content') {
      // 默认不自动执行内容填充，等用户手动触发或跳过
      if (!state.autoMode) {
        stepState.status = 'idle'
        stepState.error = null
        return
      }
      await runContentStep()
      return
    }

    // 步骤 1-3：非自动模式先等用户输入想法
    if (!state.autoMode && !inputHint) {
      stepState.status = 'idle'
      stepState.error = null
      return
    }

    // 开始生成
    await generateOptions(step, inputHint)
  }

  /** 用户输入想法后触发生成（由页面调用） */
  function submitStepHint(step: ButlerStep) {
    const stepState = state.steps[step]
    const hint = stepState.userHint.trim()
    if (step === 'chapters') {
      runChapterStep(hint || undefined)
      return
    }
    if (step === 'knowledge') {
      runKnowledgeStep()
      return
    }
    if (step === 'content') {
      runContentStep()
      return
    }
    // 步骤2/3 走多轮迭代，步骤1 走原有4选1
    if (step === 'storyline' || step === 'characters') {
      runIterativeStep(step, hint || undefined)
    } else {
      generateOptions(step, hint || undefined)
    }
  }

  /** 步骤2/3：多轮迭代生成 */
  async function runIterativeStep(step: 'storyline' | 'characters', userHint?: string) {
    const stepState = state.steps[step]
    stepState.status = 'generating'
    stepState.options = []
    stepState.error = null
    stepState.iterationRound = 0
    stepState.iterationPhase = 'generating'

    const prevResult = step === 'storyline'
      ? state.steps.topic.result
      : state.steps.storyline.result

    try {
      const data: any = await novelApi.startButlerIteration({
        portfolio_id: portfolioId(),
        action: step,
        setting: state.originalSetting,
        prev_step_result: prevResult,
        user_prompt: userHint,
        model_name: selectedModel(),
        butler_session_id: state.butlerSessionId || '',
      })

      const iterationId = data.iteration_id as string
      stepState.iterationId = iterationId

      // 轮询迭代状态
      const poll = setInterval(async () => {
        if (cancelled) {
          clearInterval(poll)
          return
        }
        try {
          const status: any = await novelApi.getButlerIterationStatus(iterationId)
          stepState.iterationRound = status.round
          stepState.iterationMaxRounds = status.max_rounds
          stepState.iterationPhase = status.phase

          if (status.phase === 'completed') {
            clearInterval(poll)
            stepState.result = status.final_result
            stepState.status = 'confirmed'
            // 自动推进到下一步
            const idx = STEPS.indexOf(step)
            if (idx < STEPS.length - 1) {
              runStep(STEPS[idx + 1])
            }
          } else if (status.phase === 'failed') {
            clearInterval(poll)
            stepState.status = 'idle'
            stepState.error = status.error || '生成失败，请重试'
          }
        } catch {
          // 轮询失败不中断，继续等待
        }
      }, 3000)
    } catch (e: any) {
      stepState.status = 'idle'
      stepState.error = e?.message || '启动迭代失败'
    }
  }

  /** 步骤1 的实际生成逻辑（保留原有4选1模式） */
  async function generateOptions(step: ButlerStep, userHint?: string) {
    const stepState = state.steps[step]
    stepState.status = 'generating'
    stepState.options = []
    stepState.error = null

    const actionMap: Record<string, string> = {
      topic: 'generate_topic',
      storyline: 'generate_storyline',
      characters: 'generate_characters_ensemble',
    }
    const action = actionMap[step]

    // 构建上下文：setting 始终传原始创作方向，title 传上一步的结果
    const setting = state.originalSetting
    let title = ''
    if (step === 'storyline') {
      title = state.steps.topic.result
    } else if (step === 'characters') {
      title = state.steps.storyline.result
    }

    const results: (string | null)[] = [null, null, null, null]
    let completedCount = 0

    for (let i = 0; i < 4; i++) {
      try {
        const data: any = await novelApi.outlineChapterAI({
          portfolio_id: portfolioId(),
          action,
          title,
          summary: '',
          context: { setting },
          model_name: selectedModel(),
          user_prompt: userHint,
          butler_session_id: state.butlerSessionId || undefined,
        })
        const taskId = data.task_id as number
        startTaskPoll(taskId, (task) => {
          if (cancelled) return
          completedCount++
          if (task.status === 'completed') {
            const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
            results[i] = result.content || result.characters || ''
          }
          // 所有 4 个都完成
          if (completedCount >= 4) {
            stepState.options = results.map(r => r || '（生成失败）')
            if (stepState.options.every(o => o === '（生成失败）')) {
              stepState.status = 'idle'
              stepState.error = '所有方案生成失败，请重试'
              return
            }
            stepState.status = 'choosing'
            // 自动模式：选第一个有效方案
            if (state.autoMode) {
              const validIdx = results.findIndex(r => r !== null)
              confirmStep(step, validIdx >= 0 ? validIdx : 0)
            }
          }
        })
      } catch {
        completedCount++
        if (completedCount >= 4) {
          stepState.options = results.map(r => r || '（生成失败）')
          stepState.status = 'choosing'
        }
      }
    }
  }

  /** 确认选择，推进到下一步 */
  function confirmStep(step: ButlerStep, selectedIndex: number) {
    const stepState = state.steps[step]
    stepState.selectedIndex = selectedIndex
    stepState.result = stepState.options[selectedIndex] || ''
    stepState.status = 'confirmed'

    // 推进到下一步
    const idx = STEPS.indexOf(step)
    if (idx < STEPS.length - 1) {
      const nextStep = STEPS[idx + 1]
      runStep(nextStep)
    }
  }

  /** 步骤4：章节生成 */
  async function runChapterStep(userHint?: string) {
    const stepState = state.steps.chapters
    stepState.status = 'generating'
    stepState.error = null
    state.outlineChapters = []

    try {
      const data: any = await novelApi.generateOutline({
        portfolio_id: portfolioId(),
        setting: state.steps.topic.result,
        characters: state.steps.characters.result,
        background: '',
        plot: state.steps.storyline.result,
        chapter_num: state.chapterNum || 30,
        model_name: selectedModel(),
        user_prompt: userHint,
        butler_session_id: state.butlerSessionId || undefined,
      })
      state.outlineTaskId = data.task_id
      startTaskPoll(data.task_id, (task) => {
        if (cancelled) return
        if (task.status === 'completed') {
          const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
          if (result.chapters) {
            state.outlineChapters = result.chapters
            stepState.status = 'choosing'
            if (state.autoMode) {
              confirmChapters()
            }
          } else {
            stepState.status = 'idle'
            stepState.error = '大纲生成结果异常'
          }
        } else {
          stepState.status = 'idle'
          stepState.error = task.error_msg || '大纲生成失败'
        }
      })
    } catch {
      stepState.status = 'idle'
      stepState.error = '大纲生成请求失败'
    }
  }

  /** 确认大纲，创建小说和章节 */
  async function confirmChapters() {
    const stepState = state.steps.chapters
    if (!state.outlineTaskId || state.outlineChapters.length === 0) return

    try {
      // 从选题结果中提取标题（取第一行或前20字）
      const topicResult = state.steps.topic.result
      const titleMatch = topicResult.match(/(?:标题|书名)[：:]\s*[《「]?(.+?)[》」]?(?:\n|$)/i)
      const title = titleMatch ? titleMatch[1].trim() : topicResult.slice(0, 20)

      const data: any = await novelApi.adoptOutline({
        portfolio_id: portfolioId(),
        task_id: state.outlineTaskId,
        title,
        description: state.steps.storyline.result.slice(0, 200),
        source: 'butler',
        butler_topic: state.steps.topic.result,
        butler_storyline: state.steps.storyline.result,
        butler_characters: state.steps.characters.result,
        butler_session_id: state.butlerSessionId || undefined,
        chapters: state.outlineChapters,
      })
      state.createdNovelId = data.novel?.id || data.id || null
      stepState.status = 'confirmed'
      stepState.result = `已创建小说「${title}」，共 ${state.outlineChapters.length} 章`

      // 推进到知识图谱提取
      runStep('knowledge')
    } catch (e: any) {
      stepState.error = e.message || '创建小说失败'
    }
  }

  /** 步骤5：知识图谱提取 */
  async function runKnowledgeStep() {
    const stepState = state.steps.knowledge
    stepState.status = 'generating'
    stepState.error = null
    state.knowledgeProgress = { phase: 'extracting', taskId: null }

    if (!state.createdNovelId) {
      stepState.status = 'confirmed'
      stepState.result = '跳过知识提取（无小说 ID）'
      runStep('content')
      return
    }

    try {
      // 触发 AI 提取知识图谱
      const data: any = await overviewApi.extract(state.createdNovelId, {
        model_name: selectedModel(),
      })
      const taskId = data.task_id
      state.knowledgeProgress.taskId = taskId

      // 轮询等待提取完成
      startTaskPoll(taskId, async (task) => {
        if (cancelled) return
        if (task.status === 'completed') {
          try {
            state.knowledgeProgress.phase = 'parsing'
            await overviewApi.parseExtract(state.createdNovelId!, taskId)
            state.knowledgeProgress.phase = 'done'
            stepState.status = 'confirmed'
            stepState.result = '知识图谱已生成（人物、情节线、伏笔、关系）'
          } catch {
            stepState.status = 'confirmed'
            stepState.result = '知识解析失败，继续生成内容'
          }
          runStep('content')
        } else {
          stepState.status = 'confirmed'
          stepState.result = '知识提取失败，继续生成内容'
          runStep('content')
        }
      })
    } catch {
      stepState.status = 'confirmed'
      stepState.result = '知识提取请求失败，继续生成内容'
      runStep('content')
    }
  }

  /** 步骤6：内容填充 */
  async function runContentStep() {
    const stepState = state.steps.content
    stepState.status = 'generating'
    stepState.error = null

    if (!state.createdNovelId) {
      stepState.status = 'confirmed'
      stepState.result = '小说已创建，请在主界面使用批量生成功能填充内容'
      return
    }

    // 加载新创建的小说章节
    try {
      await novelStore.fetchChapters(state.createdNovelId)
      const chapters = novelStore.chapters
      if (chapters.length === 0) {
        stepState.status = 'confirmed'
        stepState.result = '小说已创建但无章节数据'
        return
      }

      state.contentProgress = { total: chapters.length, completed: 0, current: '' }

      // 逐章生成内容
      for (const chapter of chapters) {
        if (cancelled) break
        state.contentProgress.current = chapter.title

        try {
          // 使用工作流生成章节内容
          await workflowStore.submitWorkflow(
            portfolioId(),
            'full_chapter',
            selectedModel(),
            {
              chapter_id: chapter.id,
              novel_id: state.createdNovelId,
              chapter_sort_order: chapter.sort_order,
            },
          )

          // 等待工作流完成
          await waitForWorkflow()
          state.contentProgress.completed++
        } catch {
          state.contentProgress.completed++
          // 单章失败不中断整体流程
        }
      }

      stepState.status = 'confirmed'
      stepState.result = `内容生成完成：${state.contentProgress.completed}/${state.contentProgress.total} 章`
    } catch {
      stepState.status = 'confirmed'
      stepState.result = '小说已创建，请在主界面使用批量生成功能填充内容'
    }
  }

  /** 等待当前工作流完成（带超时保护） */
  function waitForWorkflow(): Promise<void> {
    return new Promise((resolve) => {
      const timeout = 180_000 // 单章最长等 3 分钟
      const start = Date.now()
      // 先等 pending 变为 true（确保工作流已开始），再等它变为 false（完成）
      let seenPending = workflowStore.pending
      const check = setInterval(() => {
        if (cancelled || Date.now() - start > timeout) {
          clearInterval(check)
          resolve()
          return
        }
        if (workflowStore.pending) {
          seenPending = true
        }
        if (seenPending && !workflowStore.pending) {
          clearInterval(check)
          resolve()
        }
      }, 2000)
    })
  }

  /** 跳过内容填充 */
  function skipContent() {
    const stepState = state.steps.content
    stepState.status = 'confirmed'
    stepState.result = '已跳过内容填充，可在小说工坊中手动生成'
  }

  /** 重试当前步骤 */
  function retryCurrentStep() {
    const step = state.currentStep
    const stepState = state.steps[step]
    stepState.status = 'idle'
    stepState.error = null
    stepState.options = []
  }

  /**
   * 从小说记录恢复管家状态
   * 直接读取 Novel 上的 butler_topic/storyline/characters 字段
   */
  async function restoreFromTasks(novelId: number) {
    try {
      // 获取小说详情
      const novel: any = await novelApi.get(novelId)
      if (!novel) return false

      // 重置状态
      Object.assign(state, createButlerState())
      state.active = true
      state.createdNovelId = novelId
      cancelled = false

      // 恢复步骤 1-3：从 Novel 字段读取
      const stepFieldMap: { step: ButlerStep; field: string }[] = [
        { step: 'topic', field: 'butler_topic' },
        { step: 'storyline', field: 'butler_storyline' },
        { step: 'characters', field: 'butler_characters' },
      ]

      let firstIncomplete: ButlerStep | null = null

      for (const { step, field } of stepFieldMap) {
        const value = novel[field]
        if (value) {
          state.steps[step].status = 'confirmed'
          state.steps[step].result = value
        } else if (!firstIncomplete) {
          firstIncomplete = step
        }
      }

      // 步骤 4（章节）：检查小说是否有章节
      if (!firstIncomplete) {
        if (novel.chapter_count && novel.chapter_count > 0) {
          state.steps.chapters.status = 'confirmed'
          state.steps.chapters.result = `小说已创建，共 ${novel.chapter_count} 章`
        } else {
          // 有选题/故事线/人物但没有章节，说明大纲还没生成
          firstIncomplete = 'chapters'
        }
      }

      // 步骤 5（知识图谱）：查 AI 任务确认是否已提取
      if (!firstIncomplete) {
        try {
          const data: any = await aiApi.listTasks({
            portfolio_id: portfolioId(),
            task_types: 'knowledge_extract',
            page: 1,
            page_size: 10,
          })
          const tasks: any[] = data.tasks || []
          const done = tasks.some((t: any) => t.status === 'completed')
          if (done) {
            state.steps.knowledge.status = 'confirmed'
            state.steps.knowledge.result = '知识图谱已生成'
          } else {
            firstIncomplete = 'knowledge'
          }
        } catch {
          // 查询失败，跳过知识步骤
          state.steps.knowledge.status = 'confirmed'
          state.steps.knowledge.result = '知识图谱状态未知'
        }
      }

      // 步骤 6（内容填充）：检查章节是否有内容
      if (!firstIncomplete) {
        try {
          await novelStore.fetchChapters(novelId)
          const chapters = novelStore.chapters
          const withContent = chapters.filter((ch: any) => ch.content && ch.content.trim())
          if (withContent.length > 0) {
            state.steps.content.status = 'confirmed'
            state.steps.content.result = `已有 ${withContent.length}/${chapters.length} 章内容`
          } else {
            firstIncomplete = 'content'
          }
        } catch {
          firstIncomplete = 'content'
        }
      }

      state.currentStep = firstIncomplete || 'content'
      return true
    } catch {
      return false
    }
  }

  return {
    state,
    STEPS,
    STEP_LABELS,
    currentStepIndex,
    currentStepLabel,
    currentStepState,
    isAllDone,
    startButler,
    closeButler,
    runStep,
    submitStepHint,
    confirmStep,
    runIterativeStep,
    confirmChapters,
    skipContent,
    retryCurrentStep,
    saveState,
    restoreState,
    clearSavedState,
    restoreFromTasks,
  }
}
