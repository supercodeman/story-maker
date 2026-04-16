// web/src/composables/useWorldBuilder.ts
// 世界构建状态机 — 5 阶段反思循环向导
import { computed, reactive } from 'vue'
import { worldBuildingApi } from '@/api/worldBuilding'
import { aiApi } from '@/api/ai'
import type { ReflectionPhase, ReviewResult, ReflectionConfig, PhaseResultResponse } from '@/api/worldBuilding'

// 阶段定义
export const PHASES: ReflectionPhase[] = ['worldview', 'character', 'relation', 'foreshadow', 'plot']

export const PHASE_LABELS: Record<ReflectionPhase, string> = {
  worldview: '世界观设定',
  character: '人物设定',
  relation: '关系设定',
  foreshadow: '伏笔设定',
  plot: '剧情大纲',
}

// 每个阶段的状态
export interface PhaseState {
  status: 'idle' | 'generating' | 'reviewing' | 'optimizing' | 'done'
  round: number
  content: string           // 当前生成内容
  reviewResult: ReviewResult | null
  score: number
  taskId: number | null     // 当前生成任务 ID
  reviewTaskId: number | null // 当前审查任务 ID
  userInput: string         // 用户输入的引导
  error: string | null
}

// 整体状态
export interface WorldBuilderState {
  active: boolean
  currentPhase: ReflectionPhase
  phases: Record<ReflectionPhase, PhaseState>
  config: ReflectionConfig
  novelId: number | null
  portfolioId: number | null
}

function createPhaseState(): PhaseState {
  return {
    status: 'idle', round: 0, content: '', reviewResult: null, score: 0,
    taskId: null, reviewTaskId: null, userInput: '', error: null,
  }
}

function createState(): WorldBuilderState {
  return {
    active: false,
    currentPhase: 'worldview',
    phases: {
      worldview: createPhaseState(),
      character: createPhaseState(),
      relation: createPhaseState(),
      foreshadow: createPhaseState(),
      plot: createPhaseState(),
    },
    config: { max_rounds: 3, threshold: 6.0, auto_mode: true },
    novelId: null,
    portfolioId: null,
  }
}

/**
 * 世界构建 composable
 * @param portfolioId - 作品集 ID
 * @param novelId - 小说 ID（可选，已有小说时传入）
 */
export function useWorldBuilder(portfolioId: () => number, novelId?: () => number | null) {
  const state = reactive<WorldBuilderState>(createState())

  // 轮询定时器
  const pollTimers = new Map<string, ReturnType<typeof setInterval>>()

  const currentPhaseIndex = computed(() => PHASES.indexOf(state.currentPhase))
  const currentPhaseLabel = computed(() => PHASE_LABELS[state.currentPhase])
  const currentPhaseState = computed(() => state.phases[state.currentPhase])
  const isAllDone = computed(() => PHASES.every(p => state.phases[p].status === 'done'))

  // ========== 轮询工具 ==========
  const POLL_TIMEOUT = 5 * 60 * 1000 // 5 分钟超时
  const pollStartTimes = new Map<string, number>()

  function startPoll(key: string, taskId: number, onResult: (task: any) => void) {
    stopPoll(key)
    pollStartTimes.set(key, Date.now())
    pollTimers.set(key, setInterval(async () => {
      // 超时保护
      const startTime = pollStartTimes.get(key) || 0
      if (Date.now() - startTime > POLL_TIMEOUT) {
        stopPoll(key)
        onResult({ status: 'failed', error_msg: '任务超时，请重试' })
        return
      }
      try {
        const task: any = await aiApi.getTask(taskId)
        if (task.status === 'completed' || task.status === 'failed') {
          stopPoll(key)
          onResult(task)
        }
      } catch { /* 轮询失败忽略 */ }
    }, 3000))
  }

  function stopPoll(key: string) {
    const timer = pollTimers.get(key)
    if (timer) { clearInterval(timer); pollTimers.delete(key) }
    pollStartTimes.delete(key)
  }

  function stopAllPolls() {
    pollTimers.forEach((_, key) => stopPoll(key))
  }

  // ========== 核心方法 ==========

  /** 启动世界构建 */
  function start(nid: number) {
    state.active = true
    state.novelId = nid
    state.portfolioId = portfolioId()
    state.currentPhase = 'worldview'
  }

  /** 关闭世界构建 */
  function close() {
    stopAllPolls()
    Object.assign(state, createState())
  }

  /** 启动当前阶段的生成 */
  async function runPhase(phase?: ReflectionPhase) {
    const p = phase || state.currentPhase
    const ps = state.phases[p]
    if (!state.novelId || !state.portfolioId) return

    ps.status = 'generating'
    ps.error = null

    try {
      const res: any = await worldBuildingApi.startPhase({
        novel_id: state.novelId,
        portfolio_id: state.portfolioId,
        phase: p,
        user_input: ps.userInput,
        config: state.config,
      })
      ps.taskId = res.task_id

      // 轮询等待生成完成
      startPoll(`gen-${p}`, res.task_id, (task) => {
        if (task.status === 'completed') {
          ps.content = task.result
          // 自动触发审查
          triggerReview(p)
        } else {
          ps.status = 'idle'
          ps.error = task.error_msg || 'generation failed'
        }
      })
    } catch (e: any) {
      ps.status = 'idle'
      ps.error = e.message || 'start phase failed'
    }
  }

  /** 触发审查 */
  async function triggerReview(phase: ReflectionPhase) {
    const ps = state.phases[phase]
    if (!state.novelId || !state.portfolioId || !ps.taskId) return

    ps.status = 'reviewing'

    try {
      const res: any = await worldBuildingApi.reviewResult({
        novel_id: state.novelId,
        portfolio_id: state.portfolioId,
        phase,
        task_id: ps.taskId,
      })
      ps.reviewTaskId = res.review_task_id

      // 轮询等待审查完成
      startPoll(`review-${phase}`, res.review_task_id, (task) => {
        if (task.status === 'completed') {
          processReview(phase)
        } else {
          ps.status = 'generating' // 回退到可重试状态
          ps.error = task.error_msg || 'review failed'
        }
      })
    } catch (e: any) {
      ps.status = 'generating'
      ps.error = e.message || 'trigger review failed'
    }
  }

  /** 处理审查结果 */
  async function processReview(phase: ReflectionPhase) {
    const ps = state.phases[phase]
    if (!state.novelId || !ps.taskId || !ps.reviewTaskId) return

    try {
      const res: PhaseResultResponse = await worldBuildingApi.processReview({
        novel_id: state.novelId,
        phase,
        generate_task_id: ps.taskId,
        review_task_id: ps.reviewTaskId,
        config: state.config,
      }) as any

      ps.round = res.round

      if (res.done) {
        ps.content = res.content || ps.content
        ps.score = res.score || 0
        ps.reviewResult = res.review_result || null

        if (state.config.auto_mode) {
          // 全自动模式：直接接受并进入下一阶段
          await acceptPhase(phase)
        } else {
          // 半自动模式：等待用户确认
          ps.status = 'done'
        }
      } else {
        ps.reviewResult = res.review_result || null

        if (state.config.auto_mode) {
          // 全自动模式：自动优化
          await optimize(phase)
        } else {
          // 半自动模式：展示审查结果，等待用户决定
          ps.status = 'reviewing'
        }
      }
    } catch (e: any) {
      ps.error = e.message || 'process review failed'
    }
  }

  /** 基于审查意见优化 */
  async function optimize(phase: ReflectionPhase) {
    const ps = state.phases[phase]
    if (!state.novelId || !state.portfolioId || !ps.reviewResult) return

    ps.status = 'optimizing'

    try {
      const res: any = await worldBuildingApi.optimize({
        novel_id: state.novelId,
        portfolio_id: state.portfolioId,
        phase,
        prev_content: ps.content,
        review_json: JSON.stringify(ps.reviewResult),
        max_rounds: state.config.max_rounds,
        threshold: state.config.threshold,
      })
      ps.taskId = res.task_id

      // 轮询等待优化完成
      startPoll(`opt-${phase}`, res.task_id, (task) => {
        if (task.status === 'completed') {
          ps.content = task.result
          triggerReview(phase)
        } else {
          ps.status = 'reviewing' // 回退
          ps.error = task.error_msg || 'optimize failed'
        }
      })
    } catch (e: any) {
      ps.status = 'reviewing'
      ps.error = e.message || 'optimize failed'
    }
  }

  /** 接受当前阶段结果，写入数据库 */
  async function acceptPhase(phase?: ReflectionPhase) {
    const p = phase || state.currentPhase
    const ps = state.phases[p]
    if (!state.novelId) return

    try {
      await worldBuildingApi.acceptResult({
        novel_id: state.novelId,
        phase: p,
        content: ps.content,
        score: ps.score,
      })
      ps.status = 'done'
      // 自动进入下一阶段
      advanceToNextPhase()
    } catch (e: any) {
      ps.error = e.message || 'accept failed'
    }
  }

  /** 进入下一阶段 */
  function advanceToNextPhase() {
    const idx = PHASES.indexOf(state.currentPhase)
    if (idx < PHASES.length - 1) {
      state.currentPhase = PHASES[idx + 1]
      if (state.config.auto_mode) {
        runPhase()
      }
    }
  }

  /** 用户手动确认继续优化 */
  function continueOptimize() {
    optimize(state.currentPhase)
  }

  /** 用户手动确认接受当前结果 */
  function confirmAccept() {
    acceptPhase(state.currentPhase)
  }

  /** 重试当前阶段 */
  function retryPhase() {
    const ps = state.phases[state.currentPhase]
    ps.status = 'idle'
    ps.error = null
    ps.round = 0
    ps.content = ''
    ps.reviewResult = null
    ps.taskId = null
    ps.reviewTaskId = null
  }

  /** 更新配置 */
  function updateConfig(cfg: Partial<ReflectionConfig>) {
    Object.assign(state.config, cfg)
  }

  return {
    state,
    PHASES,
    PHASE_LABELS,
    currentPhaseIndex,
    currentPhaseLabel,
    currentPhaseState,
    isAllDone,
    start,
    close,
    runPhase,
    optimize,
    acceptPhase,
    continueOptimize,
    confirmAccept,
    retryPhase,
    updateConfig,
    stopAllPolls,
  }
}
