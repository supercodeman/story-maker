<!-- web/src/views/novel/WorldBuilderPage.vue -->
<!-- 世界构建：5 阶段反思循环向导页面 -->
<template>
  <div class="wb-page">
    <!-- 顶部导航栏 -->
    <div class="wb-topbar">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">工作空间</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">作品集</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}/novels` }">小说</el-breadcrumb-item>
        <el-breadcrumb-item>世界构建</el-breadcrumb-item>
      </el-breadcrumb>
      <div class="wb-topbar__right">
        <label class="cfg-label">最大轮次</label>
        <el-input-number v-model="state.config.max_rounds" :min="1" :max="10" size="small" style="width:100px" />
        <label class="cfg-label">评分阈值</label>
        <el-input-number v-model="state.config.threshold" :min="1" :max="10" :step="0.5" :precision="1" size="small" style="width:100px" />
        <span class="cfg-label">{{ state.config.auto_mode ? '自动' : '半自动' }}</span>
        <el-switch v-model="state.config.auto_mode" size="small" />
        <ModelSelector v-model="state.config.model_name" capability="text_gen" size="small" />
      </div>
    </div>

    <!-- 阶段步骤条 -->
    <div class="wb-steps">
      <div class="steps-track">
        <div v-for="(phase, idx) in PHASES" :key="phase" class="step-node"
          :class="{ 'step-node--done': state.phases[phase].status === 'done',
                    'step-node--active': phase === state.currentPhase && state.active,
                    'step-node--pending': phase !== state.currentPhase || !state.active }">
          <div class="step-node__dot">
            <svg v-if="state.phases[phase].status === 'done'" viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
              <path d="M13.78 4.22a.75.75 0 010 1.06l-7.25 7.25a.75.75 0 01-1.06 0L2.22 9.28a.75.75 0 011.06-1.06L6 10.94l6.72-6.72a.75.75 0 011.06 0z"/>
            </svg>
            <span v-else>{{ idx + 1 }}</span>
          </div>
          <span class="step-node__label">{{ PHASE_LABELS[phase] }}</span>
          <div v-if="idx < PHASES.length - 1" class="step-node__line"
            :class="{ 'step-node__line--done': state.phases[phase].status === 'done' }" />
        </div>
      </div>
    </div>

    <!-- 主体区域 -->
    <div class="wb-body">
      <!-- 未启动 -->
      <div v-if="!state.active" class="wb-start">
        <div class="start-glow" />
        <div class="start-icon">&#9670;</div>
        <div class="start-title">开始世界构建</div>
        <p class="start-desc">系统将依次完成 世界观设定 → 人物设定 → 关系设定 → 伏笔设定 → 剧情大纲 的完整构建流程</p>
        <button class="action-btn action-btn--primary" @click="handleStart">开始世界构建</button>
      </div>

      <!-- 已启动 -->
      <template v-else>
        <div class="phase-main">
          <!-- generating -->
          <div v-if="phaseStatus === 'generating'" class="loading-panel">
            <div class="spinner" />
            <p class="loading-text">正在生成 {{ currentPhaseLabel }}...</p>
          </div>

          <!-- optimizing -->
          <div v-else-if="phaseStatus === 'optimizing'" class="loading-panel">
            <div class="spinner" />
            <p class="loading-text">正在优化 {{ currentPhaseLabel }}（第 {{ currentPhaseState.round }} 轮）...</p>
          </div>

          <!-- reviewing（半自动模式） -->
          <div v-else-if="phaseStatus === 'reviewing'" class="review-layout">
            <div class="review-content">
              <h3 class="section-title">生成内容预览</h3>
              <el-collapse>
                <el-collapse-item title="查看内容">
                  <pre class="content-pre">{{ formatContent(currentPhaseState.content) }}</pre>
                </el-collapse-item>
              </el-collapse>
            </div>
            <div class="review-score" v-if="currentPhaseState.reviewResult">
              <h3 class="section-title">审查评分</h3>
              <div v-for="dim in currentPhaseState.reviewResult.dimensions" :key="dim.name" class="dim-row">
                <div class="dim-header">
                  <span class="dim-name">{{ dim.name }}</span>
                  <span class="dim-score">{{ dim.score.toFixed(1) }}</span>
                </div>
                <div class="dim-bar"><div class="dim-bar__fill" :style="barStyle(dim.score)" /></div>
                <p class="dim-comment">{{ dim.comment }}</p>
              </div>
              <div class="total-score">
                总分：<span class="total-score__val">{{ currentPhaseState.reviewResult.total_score.toFixed(1) }}</span>
              </div>
              <p class="review-summary">{{ currentPhaseState.reviewResult.summary }}</p>
              <p class="review-suggestion" v-if="currentPhaseState.reviewResult.suggestion">
                修改建议：{{ currentPhaseState.reviewResult.suggestion }}
              </p>
            </div>
            <div class="review-actions">
              <button class="action-btn action-btn--ghost" @click="continueOptimize">继续优化</button>
              <button class="action-btn action-btn--accent" @click="confirmAccept">接受当前结果</button>
            </div>
          </div>

          <!-- done -->
          <div v-else-if="phaseStatus === 'done'" class="done-panel">
            <h3 class="section-title">{{ currentPhaseLabel }} — 已完成</h3>
            <pre class="content-pre">{{ formatContent(currentPhaseState.content) }}</pre>
            <div v-if="currentPhaseState.reviewResult" class="done-score">
              最终评分：<span class="total-score__val">{{ currentPhaseState.score.toFixed(1) }}</span>
            </div>
            <button v-if="!isAllDone && currentPhaseIndex < PHASES.length - 1"
              class="action-btn action-btn--primary" @click="goNextPhase">进入下一阶段</button>
            <div v-if="isAllDone" class="all-done-msg">全部阶段已完成</div>
          </div>

          <!-- 错误 -->
          <div v-else-if="currentPhaseState.error" class="error-panel">
            <p class="error-text">{{ currentPhaseState.error }}</p>
            <button class="action-btn action-btn--ghost" @click="retryPhase">重试</button>
          </div>

          <!-- idle（等待启动阶段） -->
          <div v-else class="idle-panel">
            <p>当前阶段：{{ currentPhaseLabel }}</p>
            <button class="action-btn action-btn--primary" @click="runPhase()">开始生成</button>
          </div>
        </div>
      </template>

      <!-- 底部概览 -->
      <div v-if="completedPhases.length > 0" class="wb-overview">
        <el-collapse>
          <el-collapse-item title="已完成阶段概览">
            <div v-for="p in completedPhases" :key="p" class="overview-item">
              <span class="overview-label">{{ PHASE_LABELS[p] }}</span>
              <span class="overview-score">{{ state.phases[p].score.toFixed(1) }} 分</span>
              <pre class="overview-pre">{{ truncate(state.phases[p].content, 200) }}</pre>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useWorldBuilder, PHASES, PHASE_LABELS } from '@/composables/useWorldBuilder'
import type { ReflectionPhase } from '@/api/worldBuilding'
import ModelSelector from '@/components/common/ModelSelector.vue'

const route = useRoute()
const id = computed(() => route.params.id as string)
const pid = computed(() => route.params.pid as string)
const nid = computed(() => route.params.nid as string)

const {
  state, currentPhaseIndex, currentPhaseLabel, currentPhaseState, isAllDone,
  start, close, runPhase, continueOptimize, confirmAccept, retryPhase,
} = useWorldBuilder(() => Number(pid.value), () => Number(nid.value))

// 当前阶段状态快捷访问
const phaseStatus = computed(() => currentPhaseState.value.status)

// 已完成的阶段列表
const completedPhases = computed(() =>
  PHASES.filter(p => state.phases[p].status === 'done' && p !== state.currentPhase)
)

// 格式化内容：尝试 JSON 美化，失败则原文
function formatContent(raw: string): string {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

// 截断文本
function truncate(text: string, len: number): string {
  return text.length > len ? text.slice(0, len) + '...' : text
}

// 评分条样式：低分红 → 中分黄 → 高分绿
function barStyle(score: number) {
  const pct = Math.min(score / 10 * 100, 100)
  let color: string
  if (score < 4) color = '#ef4444'
  else if (score < 7) color = '#f59e0b'
  else color = '#10b981'
  return { width: `${pct}%`, background: color }
}

// 进入下一阶段
function goNextPhase() {
  const idx = currentPhaseIndex.value
  if (idx < PHASES.length - 1) {
    state.currentPhase = PHASES[idx + 1]
    runPhase()
  }
}

function handleStart() {
  start(Number(nid.value))
  runPhase()
}

onMounted(() => {
  start(Number(nid.value))
})

onUnmounted(() => {
  close()
})
</script>

<style scoped>
/* ===== 世界构建页面 — 暗色主题 ===== */
.wb-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--color-bg-deep);
  color: var(--color-text-primary);
}

/* --- 顶部导航 --- */
.wb-topbar {
  padding: 12px 24px;
  background: var(--color-bg-card);
  border-radius: 12px;
  border: 1px solid var(--border-glow);
  box-shadow: var(--shadow-sm);
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 12px 16px 0;
}
.wb-topbar__right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.cfg-label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

/* --- 步骤条 --- */
.wb-steps {
  padding: 20px 28px;
  background: rgba(26, 29, 46, 0.5);
  border-bottom: 1px solid var(--border-glow);
}
.steps-track {
  display: flex;
  align-items: flex-start;
  max-width: 800px;
  margin: 0 auto;
}
.step-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex: 1;
}
.step-node__dot {
  width: 36px; height: 36px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700;
  transition: all 0.4s ease;
  z-index: 1;
}
.step-node--done .step-node__dot {
  background: linear-gradient(135deg, var(--color-accent-green), var(--color-accent-cyan));
  color: var(--color-text-primary);
  box-shadow: 0 0 16px rgba(110, 231, 183, 0.35);
}
.step-node--active .step-node__dot {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  color: white;
  box-shadow: 0 0 20px rgba(124, 140, 248, 0.5);
  animation: pulse-dot 2s ease-in-out infinite;
}
.step-node--pending .step-node__dot {
  background: var(--color-bg-card);
  color: var(--color-text-muted);
  border: 2px solid rgba(92, 98, 128, 0.3);
}
.step-node__label {
  margin-top: 8px; font-size: 12px; font-weight: 500; white-space: nowrap;
}
.step-node--done .step-node__label { color: var(--color-accent-green); }
.step-node--active .step-node__label { color: var(--color-primary-light); }
.step-node--pending .step-node__label { color: var(--color-text-muted); }
.step-node__line {
  position: absolute; top: 18px;
  left: calc(50% + 22px); right: calc(-50% + 22px);
  height: 2px; background: rgba(92, 98, 128, 0.25); border-radius: 1px;
}
.step-node__line--done {
  background: linear-gradient(90deg, var(--color-accent-green), var(--color-accent-cyan));
  box-shadow: 0 0 8px rgba(110, 231, 183, 0.2);
}
@keyframes pulse-dot {
  0%, 100% { box-shadow: 0 0 20px rgba(124, 140, 248, 0.5); }
  50% { box-shadow: 0 0 30px rgba(124, 140, 248, 0.7); }
}

/* --- 主体 --- */
.wb-body {
  flex: 1; overflow-y: auto; padding: 24px;
}

/* --- 启动面板 --- */
.wb-start {
  display: flex; flex-direction: column; align-items: center;
  justify-content: center; padding: 60px 24px; position: relative;
}
.start-glow {
  position: absolute; width: 400px; height: 400px; border-radius: 50%;
  background: radial-gradient(circle, rgba(124, 140, 248, 0.12) 0%, transparent 70%);
  top: 50%; left: 50%; transform: translate(-50%, -50%); pointer-events: none;
}
.start-icon {
  font-size: 40px; margin-bottom: 16px;
  background: linear-gradient(135deg, var(--color-primary-light), var(--color-accent-cyan));
  -webkit-background-clip: text; -webkit-text-fill-color: transparent;
  animation: float-icon 3s ease-in-out infinite;
}
@keyframes float-icon {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}
.start-title {
  font-size: 22px; font-weight: 700; margin-bottom: 10px;
  background: linear-gradient(135deg, var(--color-text-primary), var(--color-primary-light));
  -webkit-background-clip: text; -webkit-text-fill-color: transparent;
}
.start-desc {
  font-size: 14px; color: var(--color-text-secondary);
  margin-bottom: 28px; text-align: center; line-height: 1.6;
}

/* --- 加载面板 --- */
.loading-panel {
  display: flex; flex-direction: column; align-items: center;
  justify-content: center; padding: 80px 0;
}
.spinner {
  width: 40px; height: 40px; border-radius: 50%;
  border: 3px solid var(--border-glow);
  border-top-color: var(--color-primary);
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.loading-text {
  margin-top: 16px; font-size: 15px; color: var(--color-text-secondary);
}

/* --- 审查布局 --- */
.review-layout {
  max-width: 1100px; margin: 0 auto;
  display: grid; grid-template-columns: 1fr 1fr; gap: 20px;
}
.review-actions {
  grid-column: 1 / -1;
  display: flex; gap: 12px; justify-content: center; padding-top: 8px;
}
.section-title {
  font-size: 15px; font-weight: 600; margin-bottom: 12px;
  color: var(--color-primary-light);
}
.content-pre {
  background: var(--color-bg-surface); border-radius: 8px;
  padding: 14px; font-size: 13px; line-height: 1.7;
  white-space: pre-wrap; word-break: break-all;
  max-height: 400px; overflow-y: auto;
  color: var(--color-text-primary);
}

/* --- 评分维度 --- */
.dim-row { margin-bottom: 14px; }
.dim-header { display: flex; justify-content: space-between; margin-bottom: 4px; }
.dim-name { font-size: 13px; color: var(--color-text-secondary); }
.dim-score { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.dim-bar {
  height: 6px; border-radius: 3px; background: rgba(92, 98, 128, 0.2); overflow: hidden;
}
.dim-bar__fill {
  height: 100%; border-radius: 3px; transition: width 0.5s ease;
}
.dim-comment { font-size: 12px; color: var(--color-text-muted); margin-top: 4px; }
.total-score {
  font-size: 14px; color: var(--color-text-secondary); margin: 12px 0 8px;
}
.total-score__val {
  font-size: 20px; font-weight: 700;
  background: linear-gradient(135deg, var(--color-accent-cyan), var(--color-accent-green));
  -webkit-background-clip: text; -webkit-text-fill-color: transparent;
}
.review-summary { font-size: 13px; color: var(--color-text-secondary); line-height: 1.6; }
.review-suggestion {
  font-size: 13px; color: var(--color-accent-amber); margin-top: 8px; line-height: 1.6;
}

/* --- done / error / idle --- */
.done-panel, .error-panel, .idle-panel {
  max-width: 800px; margin: 0 auto; text-align: center; padding: 32px 0;
}
.done-score { margin: 16px 0; }
.all-done-msg {
  margin-top: 20px; font-size: 16px; font-weight: 600;
  color: var(--color-accent-green);
}
.error-text {
  color: #ef4444; margin-bottom: 16px; font-size: 14px;
}

/* --- 底部概览 --- */
.wb-overview {
  margin-top: 24px; max-width: 1100px; margin-left: auto; margin-right: auto;
}
.overview-item {
  padding: 10px 0; border-bottom: 1px solid var(--border-glow);
}
.overview-label {
  font-size: 13px; font-weight: 600; color: var(--color-primary-light); margin-right: 12px;
}
.overview-score { font-size: 12px; color: var(--color-accent-green); }
.overview-pre {
  font-size: 12px; color: var(--color-text-muted); margin-top: 6px;
  white-space: pre-wrap; line-height: 1.5;
}

/* --- 通用按钮 --- */
.action-btn {
  padding: 10px 28px; border: none; border-radius: 10px;
  font-size: 14px; font-weight: 600; cursor: pointer;
  transition: all 0.3s ease;
}
.action-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.action-btn--primary {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  color: white; box-shadow: 0 4px 20px rgba(124, 140, 248, 0.35);
  position: relative; z-index: 1;
}
.action-btn--primary:hover:not(:disabled) {
  transform: translateY(-2px); box-shadow: 0 6px 28px rgba(124, 140, 248, 0.5);
}
.action-btn--accent {
  background: linear-gradient(135deg, var(--color-accent-cyan), var(--color-accent-green));
  color: var(--color-text-primary); font-weight: 600;
  box-shadow: 0 2px 12px rgba(103, 232, 249, 0.25);
}
.action-btn--accent:hover:not(:disabled) {
  box-shadow: 0 4px 20px rgba(103, 232, 249, 0.4); transform: translateY(-1px);
}
.action-btn--ghost {
  background: var(--color-bg-card); color: var(--color-text-secondary);
  border: 1px solid var(--border-glow);
}
.action-btn--ghost:hover:not(:disabled) {
  border-color: var(--color-primary); color: var(--color-primary-light);
  background: var(--color-bg-hover);
}
</style>
