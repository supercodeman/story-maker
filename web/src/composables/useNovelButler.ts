// web/src/composables/useNovelButler.ts
// 小说管家状态机 — 7 步 AI 创作向导
import { computed, reactive, watch } from 'vue'
import { novelApi } from '@/api/novel'
import { aiApi } from '@/api/ai'
import { overviewApi } from '@/api/overview'
import { useNovelStore } from '@/store/novel'
import { useWorkflowStore } from '@/store/workflow'

// 步骤定义
export type ButlerStep = 'topic' | 'storyline' | 'characters' | 'chapters' | 'opening_polish' | 'knowledge' | 'content'

const STEPS: ButlerStep[] = ['topic', 'storyline', 'characters', 'chapters', 'opening_polish', 'knowledge', 'content']

const STEP_LABELS: Record<ButlerStep, string> = {
  topic: '选题',
  storyline: '故事线',
  characters: '人物设计',
  chapters: '章节生成',
  opening_polish: '开篇打磨',
  knowledge: '知识图谱',
  content: '内容填充',
}

// 对话历史消息（对话模式调整用）
export interface StepMessage {
  role: 'assistant' | 'user'
  content: string
  timestamp: number
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
  // token 消耗统计
  tokenUsage?: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
  // 对话模式相关（步骤1/2/3）
  messages: StepMessage[]   // 对话历史
  refineInput: string       // 调整输入框内容
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
  // 步骤5 特有：开篇打磨（前5章精细化概要）
  openingChapters: any[]
  openingProgress: { phase: 'idle' | 'polishing' | 'generating' | 'done'; current: number; total: number }
  // 步骤6 特有：知识图谱提取进度
  knowledgeProgress: { phase: 'idle' | 'extracting' | 'parsing' | 'done'; taskId: number | null }
  // 步骤6 特有：内容生成进度
  contentProgress: { total: number; completed: number; current: string }
  // 用户最初输入的创作方向（与选题结果区分）
  originalSetting: string
  // 创建的小说信息
  createdNovelId: number | null
  // 管家会话 ID，串联同一次管家创作的所有任务
  butlerSessionId: string
  // 故事线结构化数据（从 ---STORY_STRUCTURE--- 提取的 JSON）
  storylineStructure: string
  // 故事线可选开关
  enableBeats: boolean
  enableSubplots: boolean
}

function createStepState(): StepState {
  return { status: 'idle', options: [], selectedIndex: -1, result: '', userHint: '', error: null, messages: [], refineInput: '' }
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
      opening_polish: createStepState(),
      knowledge: createStepState(),
      content: createStepState(),
    },
    chapterNum: 56,
    outlineChapters: [],
    outlineTaskId: null,
    openingChapters: [],
    openingProgress: { phase: 'idle', current: 0, total: 5 },
    knowledgeProgress: { phase: 'idle', taskId: null },
    contentProgress: { total: 0, completed: 0, current: '' },
    originalSetting: '',
    createdNovelId: null,
    butlerSessionId: '',
    storylineStructure: '',
    enableBeats: false,
    enableSubplots: false,
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
  const SESSION_KEY = `novel_butler_session_id`

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
        // 兼容旧数据：补充 messages/refineInput 默认值
        if (!saved.steps[step].messages) saved.steps[step].messages = []
        if (!saved.steps[step].refineInput) saved.steps[step].refineInput = ''
      }
      // 知识图谱 extracting/parsing 也重置
      if (saved.knowledgeProgress.phase !== 'done' && saved.knowledgeProgress.phase !== 'idle') {
        saved.knowledgeProgress.phase = 'idle'
        saved.knowledgeProgress.taskId = null
        saved.steps.knowledge.status = 'idle'
      }
      // 兼容：result 可能存了 JSON 格式，提取纯文本
      for (const step of ['topic', 'storyline', 'characters'] as const) {
        if (saved.steps[step].result && saved.steps[step].result.startsWith('{')) {
          saved.steps[step].result = extractTaskResultText(saved.steps[step].result)
        }
      }
      // 开篇打磨进行中也重置
      if (saved.openingProgress && saved.openingProgress.phase !== 'done' && saved.openingProgress.phase !== 'idle') {
        saved.openingProgress.phase = 'idle'
        saved.steps.opening_polish && (saved.steps.opening_polish.status = 'idle')
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
    // 将 session_id 独立存储，clearSavedState 不会清除它
    localStorage.setItem(SESSION_KEY, state.butlerSessionId)
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
    localStorage.removeItem(SESSION_KEY)
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
    if (step === 'opening_polish') {
      await runOpeningPolishStep()
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
  async function runIterativeStep(step: 'storyline' | 'characters', userHint?: string, conversationHistory?: { role: string; content: string }[]) {
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
        conversation_history: conversationHistory,
        // 故事线步骤传递可选开关
        ...(step === 'storyline' ? {
          enable_beats: state.enableBeats,
          enable_subplots: state.enableSubplots,
          chapter_num: state.chapterNum || 56,
        } : {}),
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
            // 记录 token 消耗
            if (status.total_tokens > 0) {
              stepState.tokenUsage = {
                prompt_tokens: status.prompt_tokens || 0,
                completion_tokens: status.completion_tokens || 0,
                total_tokens: status.total_tokens || 0,
              }
            }
            // 故事线完成时提取结构化数据
            if (step === 'storyline' && status.structured_data) {
              state.storylineStructure = status.structured_data
            }
            // 追加 assistant message（对话模式用）
            if (!state.autoMode) {
              const summary = status.final_result.slice(0, 200) + (status.final_result.length > 200 ? '...' : '')
              stepState.messages.push({ role: 'assistant', content: summary, timestamp: Date.now() })
            }
            // 自动推进到下一步
            if (state.autoMode) {
              const idx = STEPS.indexOf(step)
              if (idx < STEPS.length - 1) {
                runStep(STEPS[idx + 1])
              }
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
  async function generateOptions(step: ButlerStep, userHint?: string, conversationHistory?: { role: string; content: string }[]) {
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
    let totalPromptTokens = 0
    let totalCompletionTokens = 0
    let totalTotalTokens = 0

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
          conversation_history: conversationHistory,
        })
        const taskId = data.task_id as number
        startTaskPoll(taskId, (task) => {
          if (cancelled) return
          completedCount++
          // 累加 token
          totalPromptTokens += task.prompt_tokens || 0
          totalCompletionTokens += task.completion_tokens || 0
          totalTotalTokens += task.total_tokens || 0
          if (task.status === 'completed') {
            const result = typeof task.result === 'string' ? JSON.parse(task.result) : task.result
            results[i] = result.content || result.characters || ''
          }
          // 所有 4 个都完成
          if (completedCount >= 4) {
            stepState.options = results.map(r => r || '（生成失败）')
            // 记录 token 消耗
            if (totalTotalTokens > 0) {
              stepState.tokenUsage = {
                prompt_tokens: totalPromptTokens,
                completion_tokens: totalCompletionTokens,
                total_tokens: totalTotalTokens,
              }
            }
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

    // 追加 assistant message（对话模式用，步骤1选题）
    if (!state.autoMode && step === 'topic') {
      const summary = stepState.result.slice(0, 200) + (stepState.result.length > 200 ? '...' : '')
      stepState.messages.push({ role: 'assistant', content: summary, timestamp: Date.now() })
    }

    // 非自动模式下步骤1不自动推进（用户可能想对话调整）
    if (!state.autoMode && step === 'topic') {
      return
    }

    // 推进到下一步
    const idx = STEPS.indexOf(step)
    if (idx < STEPS.length - 1) {
      const nextStep = STEPS[idx + 1]
      runStep(nextStep)
    }
  }

  /** 对话模式：确认当前步骤并推进到下一步 */
  function confirmStepAndNext(step: ButlerStep) {
    const idx = STEPS.indexOf(step)
    if (idx < STEPS.length - 1) {
      const nextStep = STEPS[idx + 1]
      runStep(nextStep)
    }
  }

  /** 对话模式：基于用户反馈调整当前步骤 */
  async function refineStep(step: ButlerStep) {
    const stepState = state.steps[step]
    const feedback = stepState.refineInput?.trim()
    if (!feedback) return

    // 追加 user message
    stepState.messages.push({ role: 'user', content: feedback, timestamp: Date.now() })

    // 构建 conversation_history
    const conversationHistory = stepState.messages.map(m => ({ role: m.role, content: m.content }))

    // 清空输入
    stepState.refineInput = ''

    // 根据步骤类型调用不同的生成方法
    if (step === 'topic') {
      // 步骤1：重新调用 generateOptions，传 conversation_history
      await generateOptions(step, feedback, conversationHistory)
    } else if (step === 'storyline' || step === 'characters') {
      // 步骤2/3：重新调用 runIterativeStep，传 conversation_history
      await runIterativeStep(step, feedback, conversationHistory)
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
        chapter_num: state.chapterNum || 56,
        model_name: selectedModel(),
        user_prompt: userHint,
        butler_session_id: state.butlerSessionId || undefined,
      })
      state.outlineTaskId = data.task_id
      startTaskPoll(data.task_id, (task) => {
        if (cancelled) return
        // 记录 token 消耗
        if (task.total_tokens > 0) {
          stepState.tokenUsage = {
            prompt_tokens: task.prompt_tokens || 0,
            completion_tokens: task.completion_tokens || 0,
            total_tokens: task.total_tokens || 0,
          }
        }
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

      // 推进到开篇打磨步骤
      runStep('opening_polish')
    } catch (e: any) {
      stepState.error = e.message || '创建小说失败'
    }
  }

  /** 步骤5：开篇打磨（前5章概要精细化 + 内容生成） */
  async function runOpeningPolishStep() {
    const stepState = state.steps.opening_polish
    stepState.status = 'generating'
    stepState.error = null
    state.openingProgress = { phase: 'polishing', current: 0, total: 5 }

    // 取前5章概要
    const first5 = state.outlineChapters.slice(0, 5)
    if (first5.length === 0) {
      stepState.status = 'confirmed'
      stepState.result = '无章节数据，跳过开篇打磨'
      runStep('knowledge')
      return
    }

    const chaptersText = first5.map((ch, i) => `第${i + 1}章「${ch.title}」：${ch.summary}`).join('\n\n')

    try {
      // 阶段1：概要精细化（多轮迭代）
      const data: any = await novelApi.startButlerIteration({
        portfolio_id: portfolioId(),
        action: 'opening_polish',
        setting: state.steps.storyline.result,
        prev_step_result: chaptersText,
        user_prompt: state.steps.characters.result,
        model_name: selectedModel(),
        butler_session_id: state.butlerSessionId || '',
      })

      const iterationId = data.iteration_id as string
      stepState.iterationId = iterationId
      stepState.iterationPhase = 'generating'

      // 轮询迭代状态
      await new Promise<void>((resolve, reject) => {
        const poll = setInterval(async () => {
          if (cancelled) {
            clearInterval(poll)
            reject(new Error('cancelled'))
            return
          }
          try {
            const status: any = await novelApi.getButlerIterationStatus(iterationId)
            stepState.iterationRound = status.round
            stepState.iterationMaxRounds = status.max_rounds
            stepState.iterationPhase = status.phase

            if (status.phase === 'completed') {
              clearInterval(poll)
              // 记录 token 消耗
              if (status.total_tokens > 0) {
                stepState.tokenUsage = {
                  prompt_tokens: status.prompt_tokens || 0,
                  completion_tokens: status.completion_tokens || 0,
                  total_tokens: status.total_tokens || 0,
                }
              }
              // 提取精细化概要
              if (status.structured_data) {
                try {
                  state.openingChapters = JSON.parse(status.structured_data)
                  // 用精细化概要替换 outlineChapters 前5章的 summary
                  for (const enhanced of state.openingChapters) {
                    const idx = (enhanced.chapter_index || 1) - 1
                    if (idx >= 0 && idx < state.outlineChapters.length && enhanced.enhanced_summary) {
                      state.outlineChapters[idx].summary = enhanced.enhanced_summary
                    }
                  }
                } catch { /* JSON 解析失败忽略 */ }
              }
              state.openingProgress.phase = 'generating'
              resolve()
            } else if (status.phase === 'failed') {
              clearInterval(poll)
              reject(new Error(status.error || '概要精细化失败'))
            }
          } catch { /* 轮询失败不中断 */ }
        }, 3000)
      })

      // 阶段2：逐章生成前5章正文（使用 opening_chapter 工作流）
      if (state.createdNovelId) {
        await novelStore.fetchChapters(state.createdNovelId)
        const chapters = novelStore.chapters
        const openingCount = Math.min(5, chapters.length)
        state.openingProgress.total = openingCount

        for (let i = 0; i < openingCount; i++) {
          if (cancelled) break
          state.openingProgress.current = i + 1
          const chapter = chapters[i]

          try {
            await workflowStore.submitWorkflow(
              portfolioId(),
              'opening_chapter',
              selectedModel(),
              {
                chapter_id: chapter.id,
                novel_id: state.createdNovelId,
                chapter_sort_order: chapter.sort_order,
              },
            )
            await waitForWorkflow()
          } catch {
            // 单章失败不中断
          }
        }
      }

      state.openingProgress.phase = 'done'
      stepState.status = 'confirmed'
      stepState.result = `前${Math.min(5, first5.length)}章开篇打磨完成`

      // 推进到知识图谱
      runStep('knowledge')
    } catch (e: any) {
      if (e?.message === 'cancelled') return
      stepState.status = 'idle'
      stepState.error = e?.message || '开篇打磨失败'
    }
  }

  /** 步骤6：知识图谱提取 */
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

      // 逐章生成内容（跳过开篇打磨已生成的前5章）
      const openingDone = state.openingProgress?.phase === 'done'
      const skipCount = openingDone ? Math.min(5, chapters.length) : 0
      const chaptersToGenerate = chapters.slice(skipCount)
      state.contentProgress.total = chaptersToGenerate.length
      state.contentProgress.completed = 0

      for (const chapter of chaptersToGenerate) {
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

  /** 跳过开篇打磨 */
  function skipOpeningPolish() {
    const stepState = state.steps.opening_polish
    stepState.status = 'confirmed'
    stepState.result = '已跳过开篇打磨'
    state.openingProgress.phase = 'done'
    runStep('knowledge')
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
   * 从 ai_task 表恢复管家状态（localStorage 丢失时的兜底）
   * 支持两种查询方式：
   *   1. 传入 novelId → 通过 novel_id 查 ai_task（从列表页恢复已有小说）
   *   2. 不传 novelId → 通过 localStorage 中的 butler_session_id 查（页面刷新恢复）
   *
   * 恢复策略：
   * - 从后往前推断：如果步骤2有数据，说明步骤1一定已确认
   * - currentStep 定位到最远已完成步骤的下一步
   * - 步骤1：若后续步骤已有数据则直接 confirmed（用户已选过），否则 choosing
   */
  async function restoreFromAITasks(novelId?: number): Promise<boolean> {
    try {
      const sessionId = localStorage.getItem(SESSION_KEY)
      // 两种查询条件都没有，无法恢复
      if (!sessionId && !novelId) return false

      // 查询管家相关任务（兼容新旧两种 task_type）
      const taskTypes = [
        'butler_generate_topic',
        'butler_storyline_draft', 'butler_storyline_review',
        'butler_characters_draft', 'butler_characters_review',
        'butler_generate_storyline', 'butler_generate_characters',
      ].join(',')

      const queryParams: any = {
        portfolio_id: portfolioId(),
        task_types: taskTypes,
        page: 1,
        page_size: 100,
      }
      if (novelId) {
        queryParams.novel_id = novelId
      } else {
        queryParams.butler_session_id = sessionId
      }

      const data: any = await aiApi.listTasks(queryParams)

      const tasks: any[] = data.tasks || []
      if (tasks.length === 0) return false

      // 从任务中提取 butler_session_id（用于后续步骤继续创作）
      const recoveredSessionId = sessionId || tasks.find((t: any) => t.butler_session_id)?.butler_session_id || ''

      // 按 task_type 分组，取每种类型中最新的 completed 任务
      const completedByType = new Map<string, any>()
      for (const t of tasks) {
        if (t.status !== 'completed') continue
        const existing = completedByType.get(t.task_type)
        if (!existing || new Date(t.created_at) > new Date(existing.created_at)) {
          completedByType.set(t.task_type, t)
        }
      }

      // 提取各步骤的恢复数据（兼容新旧 task_type）
      const topicTasks = tasks.filter(
        (t: any) => t.task_type === 'butler_generate_topic' && t.status === 'completed'
      )
      const storylineResult = extractTaskResultText(
        completedByType.get('butler_storyline_draft')?.result
        || completedByType.get('butler_generate_storyline')?.result
        || completedByType.get('butler_storyline_review')?.result
        || ''
      )
      const charsResult = extractTaskResultText(
        completedByType.get('butler_characters_draft')?.result
        || completedByType.get('butler_generate_characters')?.result
        || completedByType.get('butler_characters_review')?.result
        || ''
      )

      // 重置状态
      Object.assign(state, createButlerState())
      state.active = true
      state.butlerSessionId = recoveredSessionId
      state.createdNovelId = novelId || null
      cancelled = false

      // 将恢复的 session_id 存入 localStorage，后续步骤可继续使用
      if (recoveredSessionId) {
        localStorage.setItem(SESSION_KEY, recoveredSessionId)
      }

      // 从后往前确定最远已完成步骤，同时恢复数据
      // 步骤3（人物设计）
      if (charsResult) {
        state.steps.characters.status = 'confirmed'
        state.steps.characters.result = charsResult
      }
      // 步骤2（故事线）
      if (storylineResult) {
        state.steps.storyline.status = 'confirmed'
        state.steps.storyline.result = storylineResult
      }
      // 步骤1（选题）
      if (topicTasks.length > 0) {
        topicTasks.sort((a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
        const options: string[] = []
        for (const t of topicTasks) {
          if (t.result) options.push(extractTaskResultText(t.result))
        }
        state.steps.topic.options = options

        if (storylineResult) {
          // 步骤2已有数据，说明步骤1一定已确认过，直接标记 confirmed
          state.steps.topic.status = 'confirmed'
          state.steps.topic.result = options[0] || ''
        } else if (options.length > 0) {
          // 步骤2还没数据，步骤1回到 choosing 让用户重新选
          state.steps.topic.status = 'choosing'
        }
      }

      // 定位 currentStep：找到第一个未 confirmed 的步骤
      const restorableSteps: ButlerStep[] = ['topic', 'storyline', 'characters', 'chapters']
      let nextStep: ButlerStep = 'topic'
      for (const step of restorableSteps) {
        if (state.steps[step].status !== 'confirmed') {
          nextStep = step
          break
        }
        if (step === 'characters') {
          nextStep = 'chapters'
        }
      }
      state.currentStep = nextStep

      return true
    } catch {
      return false
    }
  }

  /**
   * 内部方法：从 ai_task 表补充恢复步骤1-3缺失的数据（不重置 state）
   */
  async function _fillFromAITasks(novelId: number) {
    try {
      const taskTypes = [
        'butler_generate_topic',
        'butler_storyline_draft', 'butler_storyline_review',
        'butler_characters_draft', 'butler_characters_review',
        'butler_generate_storyline', 'butler_generate_characters',
      ].join(',')

      // 第一次：通过 novel_id 查
      const data: any = await aiApi.listTasks({
        portfolio_id: portfolioId(),
        novel_id: novelId,
        task_types: taskTypes,
        page: 1,
        page_size: 100,
      })

      let tasks: any[] = data.tasks || []

      _applyTasksToState(tasks)

      // 如果 storyline 或 characters 仍然缺失，扩大范围查同 portfolio 下所有管家任务
      if (state.steps.storyline.status !== 'confirmed' || state.steps.characters.status !== 'confirmed') {
        const data2: any = await aiApi.listTasks({
          portfolio_id: portfolioId(),
          task_types: taskTypes,
          page: 1,
          page_size: 200,
        })
        const allTasks: any[] = data2.tasks || []
        _applyTasksToState(allTasks)
      }
    } catch {
      // 补充失败不影响主流程
    }
  }

  /** 从 AI 任务的 result 字段中提取纯文本（处理嵌套 JSON） */
  function extractTaskResultText(raw: string | undefined): string {
    if (!raw) return ''
    // 如果不是 JSON，直接返回
    if (!raw.startsWith('{')) return raw
    try {
      const obj = JSON.parse(raw)
      // 直接是 review 结果：{"score":..., "revised_content":"..."}
      if (obj.revised_content && typeof obj.revised_content === 'string') {
        return obj.revised_content
      }
      // executor 返回格式：{"content": "..."} 或 {"summary": "..."}
      const text = obj.content || obj.summary || obj.text || ''
      if (typeof text !== 'string') return raw
      // content 本身可能还是 JSON（review 的原始输出），尝试提取 revised_content
      if (text.startsWith('{')) {
        try {
          const inner = JSON.parse(text)
          return inner.revised_content || inner.content || text
        } catch { return text }
      }
      return text
    } catch { return raw }
  }

  /** 从任务列表中提取数据填充 state（只补充缺失的步骤） */
  function _applyTasksToState(tasks: any[]) {
    if (tasks.length === 0) return

    // 恢复 butler_session_id
    if (!state.butlerSessionId) {
      const sessionId = tasks.find((t: any) => t.butler_session_id)?.butler_session_id
      if (sessionId) {
        state.butlerSessionId = sessionId
        localStorage.setItem(SESSION_KEY, sessionId)
      }
    }

    // 按 task_type 分组，取每种类型中最新的 completed 任务
    const completedByType = new Map<string, any>()
    for (const t of tasks) {
      if (t.status !== 'completed') continue
      const existing = completedByType.get(t.task_type)
      if (!existing || new Date(t.created_at) > new Date(existing.created_at)) {
        completedByType.set(t.task_type, t)
      }
    }

    // 步骤1（选题）：只在当前未 confirmed 时补充
    if (state.steps.topic.status !== 'confirmed') {
      const topicTasks = tasks.filter(
        (t: any) => t.task_type === 'butler_generate_topic' && t.status === 'completed'
      )
      if (topicTasks.length > 0) {
        topicTasks.sort((a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
        const options: string[] = []
        for (const t of topicTasks) {
          if (t.result) options.push(extractTaskResultText(t.result))
        }
        state.steps.topic.options = options
        // 如果步骤2有数据，说明步骤1已确认过
        const hasStoryline = completedByType.has('butler_storyline_draft') || completedByType.has('butler_storyline_review')
          || completedByType.has('butler_generate_storyline')
        if (hasStoryline) {
          state.steps.topic.status = 'confirmed'
          state.steps.topic.result = options[0] || ''
        } else if (options.length > 0) {
          state.steps.topic.status = 'choosing'
        }
      }
    }

    // 步骤2（故事线）：只在当前未 confirmed 时补充
    if (state.steps.storyline.status !== 'confirmed') {
      // 优先 draft（纯文本），review 的 result 是嵌套 JSON 不适合直接展示
      const rawResult = completedByType.get('butler_storyline_draft')?.result
        || completedByType.get('butler_generate_storyline')?.result
        || completedByType.get('butler_storyline_review')?.result
      const result = extractTaskResultText(rawResult)
      if (result) {
        state.steps.storyline.status = 'confirmed'
        state.steps.storyline.result = result
      }
    }

    // 步骤3（人物设计）：只在当前未 confirmed 时补充
    if (state.steps.characters.status !== 'confirmed') {
      // 优先 draft（纯文本），review 的 result 是嵌套 JSON 不适合直接展示
      const rawResult = completedByType.get('butler_characters_draft')?.result
        || completedByType.get('butler_generate_characters')?.result
        || completedByType.get('butler_characters_review')?.result
      const result = extractTaskResultText(rawResult)
      if (result) {
        state.steps.characters.status = 'confirmed'
        state.steps.characters.result = result
      }
    }
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

      for (const { step, field } of stepFieldMap) {
        const value = novel[field]
        if (value) {
          state.steps[step].status = 'confirmed'
          // 兼容：字段可能存了 JSON 格式，提取纯文本
          state.steps[step].result = extractTaskResultText(value)
        }
      }

      // 如果步骤1-3有缺失，从 ai_task 表补充恢复
      const hasIncompleteEarlySteps = stepFieldMap.some(
        ({ step }) => state.steps[step].status !== 'confirmed'
      )
      if (hasIncompleteEarlySteps) {
        await _fillFromAITasks(novelId)
      }

      // 重新计算 firstIncomplete
      let firstIncomplete: ButlerStep | null = null
      for (const { step } of stepFieldMap) {
        if (state.steps[step].status !== 'confirmed' && !firstIncomplete) {
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

      // 步骤 5（开篇打磨）：检查前5章是否已有内容（有内容说明开篇打磨已完成）
      if (!firstIncomplete) {
        try {
          await novelStore.fetchChapters(novelId)
          const chapters = novelStore.chapters
          const first5WithContent = chapters.slice(0, 5).filter((ch: any) => ch.content && ch.content.trim())
          if (first5WithContent.length >= Math.min(5, chapters.length)) {
            state.steps.opening_polish.status = 'confirmed'
            state.steps.opening_polish.result = '前5章开篇打磨已完成'
            state.openingProgress.phase = 'done'
          } else {
            firstIncomplete = 'opening_polish'
          }
        } catch {
          // 查询失败，标记为已完成跳过
          state.steps.opening_polish.status = 'confirmed'
          state.steps.opening_polish.result = '开篇打磨状态未知'
        }
      }

      // 步骤 6（知识图谱）：查 AI 任务确认是否已提取
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
    stopAllPolls,
    runStep,
    submitStepHint,
    confirmStep,
    confirmStepAndNext,
    refineStep,
    runIterativeStep,
    confirmChapters,
    skipContent,
    skipOpeningPolish,
    retryCurrentStep,
    saveState,
    restoreState,
    clearSavedState,
    restoreFromTasks,
    restoreFromAITasks,
  }
}
