<!-- web/src/views/memory/MyMemories.vue -->
<template>
  <div class="my-memories">
    <div class="page-header">
      <div>
        <h1 class="page-title">我的记忆</h1>
        <p class="page-desc">提取、管理和上架你的写作记忆</p>
      </div>
      <NeonButton type="primary" @click="showExtractDialog = true">+ 提取新记忆</NeonButton>
    </div>

    <!-- 类别筛选 -->
    <FadePanel class="filter-panel">
      <div class="filter-tabs">
        <button
          v-for="tab in categoryTabs"
          :key="tab.value"
          class="filter-tab"
          :class="{ 'filter-tab--active': filterCategory === tab.value }"
          @click="filterCategory = tab.value; loadMemories()"
        >
          <span class="filter-tab__icon">{{ tab.icon }}</span>
          <span class="filter-tab__label">{{ tab.label }}</span>
        </button>
      </div>
    </FadePanel>

    <!-- 记忆列表 -->
    <div v-loading="memoryStore.loading" class="memory-grid">
      <GlowCard
        v-for="memory in memoryStore.memories"
        :key="memory.id"
        hoverable
        class="memory-card"
        @click="viewDetail(memory)"
      >
        <div class="memory-card__header">
          <el-tag size="small" :type="categoryTagType(memory.category)" effect="dark">
            {{ categoryLabel(memory.category) }}
          </el-tag>
          <span class="memory-card__status" :class="`status--${memory.status}`">
            {{ statusLabel(memory.status) }}
          </span>
        </div>
        <h3 class="memory-card__title">{{ memory.title }}</h3>
        <p class="memory-card__desc">{{ memory.description || '暂无描述' }}</p>

        <!-- 提取进度条 -->
        <div v-if="isExtracting(memory)" class="extract-progress">
          <div class="extract-progress__header">
            <span class="extract-progress__label">
              <span class="pulse-dot" />
              正在提取...
            </span>
          </div>
          <div class="extract-progress__steps">
            <div
              v-for="step in extractSteps"
              :key="step.id"
              class="extract-step"
              :class="getStepClass(memory, step.id)"
            >
              <span class="extract-step__icon">{{ step.icon }}</span>
              <span class="extract-step__label">{{ step.label }}</span>
            </div>
          </div>
        </div>

        <!-- 审核进度条 -->
        <div v-else-if="memory.status === 'reviewing'" class="extract-progress">
          <div class="extract-progress__header">
            <span class="extract-progress__label">
              <span class="pulse-dot" />
              审核中...
            </span>
          </div>
          <div class="extract-progress__steps">
            <div
              v-for="step in reviewSteps"
              :key="step.id"
              class="extract-step"
              :class="getReviewStepClass(memory, step.id)"
            >
              <span class="extract-step__icon">{{ step.icon }}</span>
              <span class="extract-step__label">{{ step.label }}</span>
            </div>
          </div>
        </div>

        <!-- 提取失败 -->
        <div v-else-if="memory.extract_status === 'failed'" class="extract-error">
          <span class="extract-error__icon">✕</span>
          <span>提取失败：{{ memory.extract_error || '未知错误' }}</span>
        </div>

        <!-- 审核拒绝 -->
        <div v-else-if="memory.status === 'rejected'" class="extract-error">
          <span class="extract-error__icon">✕</span>
          <span>审核拒绝：{{ memory.review_reason || '未说明原因' }}</span>
        </div>

        <!-- 正常展示 -->
        <template v-else>
          <div class="memory-card__meta">
            <span>📝 {{ memory.sample_len }} 字</span>
            <span v-if="memory.quality > 0">⭐ {{ memory.quality.toFixed(0) }}分</span>
            <span v-if="memory.quality_grade" class="grade-badge" :class="`grade-badge--${memory.quality_grade}`">{{ memory.quality_grade }}</span>
            <span>v{{ memory.version }}</span>
          </div>
          <div v-if="memory.preview_text" class="memory-card__preview">
            {{ memory.preview_text.slice(0, 100) }}...
          </div>
        </template>

        <div class="memory-card__footer">
          <NeonButton v-if="memory.status === 'draft' && memory.extract_status === 'completed'" type="primary" @click.stop="handlePublish(memory)">上架</NeonButton>
          <NeonButton v-if="memory.status === 'published'" type="warning" @click.stop="handleArchive(memory)">下架</NeonButton>
          <button class="ghost-btn ghost-btn--danger" @click.stop="handleDelete(memory)">删除</button>
        </div>
      </GlowCard>

      <div v-if="!memoryStore.loading && memoryStore.memories.length === 0" class="empty-state">
        <div class="empty-state__icon">🧠</div>
        <p>还没有记忆</p>
        <span>点击上方按钮，从你的小说样本中提取写作记忆</span>
      </div>
    </div>

    <!-- 提取对话框 -->
    <MemoryExtractDialog v-model="showExtractDialog" :initial-category="filterCategory" @created="loadMemories" />

    <!-- 详情对话框 -->
    <el-dialog v-model="showDetail" title="记忆详情" width="720px" :close-on-click-modal="false">
      <template v-if="selectedMemory">
        <!-- 提取中状态 -->
        <div v-if="isExtracting(selectedMemory)" class="detail-extracting">
          <div class="detail-extracting__anim">
            <span class="pulse-dot pulse-dot--lg" />
          </div>
          <h3>正在提取记忆特征...</h3>
          <p>系统正在分析你的样本文本，提取写作风格特征。请稍候，完成后会自动更新。</p>
          <div class="extract-progress__steps extract-progress__steps--detail">
            <div
              v-for="step in extractSteps"
              :key="step.id"
              class="extract-step"
              :class="getStepClass(selectedMemory, step.id)"
            >
              <span class="extract-step__icon">{{ step.icon }}</span>
              <span class="extract-step__label">{{ step.label }}</span>
              <span class="extract-step__desc">{{ step.desc }}</span>
            </div>
          </div>
        </div>

        <!-- 审核中状态 -->
        <div v-else-if="selectedMemory.status === 'reviewing'" class="detail-extracting">
          <div class="detail-extracting__anim">
            <span class="pulse-dot pulse-dot--lg" />
          </div>
          <h3>正在审核中...</h3>
          <p>AI 正在对记忆进行质量检查和合规检查，请稍候。</p>
          <div class="extract-progress__steps extract-progress__steps--detail">
            <div
              v-for="step in reviewSteps"
              :key="step.id"
              class="extract-step"
              :class="getReviewStepClass(selectedMemory, step.id)"
            >
              <span class="extract-step__icon">{{ step.icon }}</span>
              <span class="extract-step__label">{{ step.label }}</span>
              <span class="extract-step__desc">{{ step.desc }}</span>
            </div>
          </div>
        </div>

        <!-- 提取失败 -->
        <div v-else-if="selectedMemory.extract_status === 'failed'" class="detail-error">
          <div class="detail-error__icon">✕</div>
          <h3>提取失败</h3>
          <p>{{ selectedMemory.extract_error || '未知错误，请重试' }}</p>
        </div>

        <!-- 审核拒绝 -->
        <div v-else-if="selectedMemory.status === 'rejected'" class="detail-error">
          <div class="detail-error__icon">✕</div>
          <h3>审核未通过</h3>
          <p>{{ selectedMemory.review_reason || '未说明原因' }}</p>
        </div>

        <!-- 正常详情 -->
        <template v-else>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-item__label">标题</span>
              <span class="detail-item__value">{{ selectedMemory.title }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">类别</span>
              <span class="detail-item__value">{{ categoryLabel(selectedMemory.category) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">状态</span>
              <span class="detail-item__value status-text" :class="`status--${selectedMemory.status}`">{{ statusLabel(selectedMemory.status) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">版本</span>
              <span class="detail-item__value">v{{ selectedMemory.version }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">样本长度</span>
              <span class="detail-item__value">{{ selectedMemory.sample_len }} 字</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">质量评分</span>
              <span class="detail-item__value">
                {{ selectedMemory.quality.toFixed(1) }}
                <span v-if="selectedMemory.quality_grade" class="grade-badge" :class="`grade-badge--${selectedMemory.quality_grade}`">{{ selectedMemory.quality_grade }}</span>
              </span>
            </div>
            <div v-if="selectedMemory.price > 0" class="detail-item">
              <span class="detail-item__label">价格</span>
              <span class="detail-item__value detail-item__value--accent">{{ selectedMemory.price }} 积分</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">销量</span>
              <span class="detail-item__value">{{ selectedMemory.sales_count }}</span>
            </div>
          </div>

          <!-- 多维质量评分 -->
          <div v-if="parsedQualityDetail" class="detail-section">
            <h4 class="detail-section__title">质量评分详情</h4>
            <div class="quality-dimensions">
              <div class="quality-dim">
                <span class="quality-dim__label">风格一致性</span>
                <el-progress :percentage="parsedQualityDetail.consistency" :stroke-width="8" />
              </div>
              <div class="quality-dim">
                <span class="quality-dim__label">可复现性</span>
                <el-progress :percentage="parsedQualityDetail.reproducibility" :stroke-width="8" />
              </div>
              <div class="quality-dim">
                <span class="quality-dim__label">独特性</span>
                <el-progress :percentage="parsedQualityDetail.uniqueness" :stroke-width="8" />
              </div>
              <div class="quality-dim">
                <span class="quality-dim__label">实用性</span>
                <el-progress :percentage="parsedQualityDetail.practicality" :stroke-width="8" />
              </div>
              <div v-if="parsedQualityDetail.evaluation" class="quality-dim__eval">
                {{ parsedQualityDetail.evaluation }}
              </div>
            </div>
          </div>

          <!-- 提取特征 -->
          <div v-if="selectedMemory.features" class="detail-section">
            <h4 class="detail-section__title">提取特征</h4>
            <pre class="code-block">{{ formatJSON(selectedMemory.features) }}</pre>
          </div>

          <!-- Prompt 模板 -->
          <div v-if="selectedMemory.prompt_tpl" class="detail-section">
            <h4 class="detail-section__title">Prompt 模板</h4>
            <div class="prompt-block">{{ selectedMemory.prompt_tpl }}</div>
          </div>

          <!-- 锚定句 -->
          <div v-if="selectedMemory.anchor_texts" class="detail-section">
            <h4 class="detail-section__title">锚定句（风格参考）</h4>
            <div class="anchor-list">
              <div v-for="(anchor, i) in parseAnchors(selectedMemory.anchor_texts)" :key="i" class="anchor-item">
                <span class="anchor-item__idx">{{ i + 1 }}</span>
                <span class="anchor-item__text">{{ anchor }}</span>
              </div>
            </div>
          </div>

          <!-- 效果预览 -->
          <div v-if="selectedMemory.preview_text" class="detail-section">
            <h4 class="detail-section__title">效果预览</h4>
            <div class="preview-block">{{ selectedMemory.preview_text }}</div>
          </div>
        </template>
      </template>
    </el-dialog>

    <!-- 上架对话框 -->
    <el-dialog v-model="showPublishDialog" title="申请上架" width="420px" :close-on-click-modal="false">
      <el-form label-position="top">
        <el-form-item label="定价（积分）">
          <el-input-number v-model="publishPrice" :min="1" :max="10000" style="width: 100%;" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPublishDialog = false">取消</el-button>
        <NeonButton type="primary" @click="confirmPublish">确认上架</NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useMemoryStore } from '@/store/memory'
import { useWorkflowStore } from '@/store/workflow'
import type { WritingMemory, QualityDetail } from '@/api/memory'
import { memoryCategoryOptions, memoryStatusOptions } from '@/api/memory'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import FadePanel from '@/components/common/FadePanel.vue'
import MemoryExtractDialog from './MemoryExtractDialog.vue'

const memoryStore = useMemoryStore()
const workflowStore = useWorkflowStore()
const filterCategory = ref('')
const showExtractDialog = ref(false)
const showDetail = ref(false)
const showPublishDialog = ref(false)
const selectedMemory = ref<WritingMemory | null>(null)
const publishPrice = ref(100)
const publishMemoryId = ref(0)

const extractSteps = [
  { id: 'feature_extract', icon: '🔍', label: '特征提取', desc: '分析样本文本，提取结构化写作特征' },
  { id: 'prompt_compile', icon: '📝', label: 'Prompt 编译', desc: '将特征编译为可执行的写作指令' },
  { id: 'quality_eval', icon: '⭐', label: '质量评估', desc: '生成样本并评估风格一致性' },
]

const reviewSteps = [
  { id: 'quality_check', icon: '⭐', label: '质量检查', desc: '评估记忆质量并生成样本打分' },
  { id: 'compliance_check', icon: '🛡️', label: '合规检查', desc: '检查内容是否合规' },
  { id: 'review_decision', icon: '📋', label: '审核决策', desc: '综合质量和合规结果做出决策' },
]

const categoryTabs = [
  { value: '', label: '全部', icon: '📋' },
  ...memoryCategoryOptions.map(o => ({
    value: o.value,
    label: o.label,
    icon: { style: '✍️', character: '👤', worldview: '🌍', plot_preference: '📖' }[o.value] || '📋',
  })),
]

onMounted(() => loadMemories())

function loadMemories() {
  memoryStore.fetchMemories(filterCategory.value || undefined)
}

function isExtracting(memory: WritingMemory) {
  return memory.extract_status === 'pending' || memory.extract_status === 'running'
}

function getStepClass(memory: WritingMemory, stepId: string) {
  // 通过 workflowStore 的节点状态来判断
  if (!memory.extract_workflow_id) return ''
  const wf = workflowStore.currentWorkflow
  if (wf && wf.id === memory.extract_workflow_id) {
    const node = workflowStore.nodes.find(n => n.node_id === stepId)
    if (node) {
      if (node.status === 'completed') return 'extract-step--done'
      if (node.status === 'running') return 'extract-step--active'
      if (node.status === 'failed') return 'extract-step--failed'
    }
  }
  // 如果整体已完成，所有步骤都标记完成
  if (memory.extract_status === 'completed') return 'extract-step--done'
  return ''
}

function getReviewStepClass(memory: WritingMemory, stepId: string) {
  if (!memory.review_workflow_id) return ''
  const wf = workflowStore.currentWorkflow
  if (wf && wf.id === memory.review_workflow_id) {
    const node = workflowStore.nodes.find(n => n.node_id === stepId)
    if (node) {
      if (node.status === 'completed') return 'extract-step--done'
      if (node.status === 'running') return 'extract-step--active'
      if (node.status === 'failed') return 'extract-step--failed'
    }
  }
  return ''
}

function categoryLabel(cat: string) {
  return memoryCategoryOptions.find(o => o.value === cat)?.label || cat
}

function statusLabel(status: string) {
  return memoryStatusOptions.find(o => o.value === status)?.label || status
}

function categoryTagType(cat: string): string {
  const map: Record<string, string> = { style: '', character: 'success', worldview: 'warning', plot_preference: 'danger' }
  return map[cat] || ''
}

function viewDetail(memory: WritingMemory) {
  selectedMemory.value = memory
  showDetail.value = true
  // 如果正在提取，关联 workflowStore 以获取节点级进度
  if (memory.extract_workflow_id && isExtracting(memory)) {
    workflowStore.fetchWorkflow(memory.extract_workflow_id)
  }
  // 如果正在审核，关联 workflowStore 以获取审核节点进度
  if (memory.review_workflow_id && memory.status === 'reviewing') {
    workflowStore.fetchWorkflow(memory.review_workflow_id)
  }
}

function handlePublish(memory: WritingMemory) {
  publishMemoryId.value = memory.id
  publishPrice.value = memory.price || 100
  showPublishDialog.value = true
}

async function confirmPublish() {
  try {
    await memoryStore.publishMemory(publishMemoryId.value, publishPrice.value)
    ElMessage.success('已提交审核')
    showPublishDialog.value = false
    loadMemories()
  } catch (e: any) {
    ElMessage.error(e.message || '上架失败')
  }
}

async function handleArchive(memory: WritingMemory) {
  await ElMessageBox.confirm('确定下架该记忆？', '提示')
  await memoryStore.archiveMemory(memory.id)
  ElMessage.success('已下架')
  loadMemories()
}

async function handleDelete(memory: WritingMemory) {
  await ElMessageBox.confirm('确定删除该记忆？此操作不可恢复', '警告', { type: 'warning' })
  await memoryStore.deleteMemory(memory.id)
  ElMessage.success('已删除')
}

function formatJSON(str: string) {
  try { return JSON.stringify(JSON.parse(str), null, 2) } catch { return str }
}

function parseAnchors(str: string): string[] {
  try {
    const arr = JSON.parse(str)
    return Array.isArray(arr) ? arr : []
  } catch {
    return str ? [str] : []
  }
}

// 解析多维质量评分
const parsedQualityDetail = computed<QualityDetail | null>(() => {
  if (!selectedMemory.value?.quality_detail) return null
  try {
    const d = JSON.parse(selectedMemory.value.quality_detail)
    if (d.consistency > 0) return d as QualityDetail
    return null
  } catch {
    return null
  }
})
</script>

<style scoped lang="scss">
.my-memories {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.page-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.filter-panel {
  margin-bottom: 24px;
  padding: 12px 16px;
}

.filter-tabs {
  display: flex;
  gap: 4px;
}

.filter-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.25s ease;

  &:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
  }

  &--active {
    background: var(--color-primary);
    color: #fff;
    box-shadow: 0 0 12px rgba(124, 140, 248, 0.3);

    &:hover {
      background: var(--color-primary-light);
      color: #fff;
    }
  }

  &__icon { font-size: 14px; }
}

.memory-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.memory-card {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }

  &__status {
    font-size: 12px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 4px;

    &.status--draft { color: var(--color-text-muted); background: rgba(92, 98, 128, 0.2); }
    &.status--reviewing { color: var(--color-accent-amber); background: rgba(252, 211, 77, 0.15); }
    &.status--published { color: var(--color-accent-green); background: rgba(110, 231, 183, 0.15); }
    &.status--rejected { color: #EF4444; background: rgba(239, 68, 68, 0.15); }
    &.status--archived { color: var(--color-text-muted); background: rgba(92, 98, 128, 0.2); }
  }

  &__title {
    font-size: 16px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 6px;
  }

  &__desc {
    font-size: 13px;
    color: var(--color-text-secondary);
    margin-bottom: 10px;
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  &__meta {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: var(--color-text-muted);
    margin-bottom: 10px;
  }

  &__preview {
    font-size: 12px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    padding: 10px;
    background: var(--color-bg-deep);
    border-radius: 6px;
    margin-bottom: 12px;
  }

  &__footer {
    display: flex;
    gap: 8px;
    align-items: center;
    padding-top: 12px;
    border-top: 1px solid var(--border-glow);
  }
}

// 提取进度
.extract-progress {
  margin: 10px 0;
  padding: 12px;
  background: var(--color-bg-deep);
  border-radius: 8px;
  border: 1px solid var(--border-glow);

  &__header {
    margin-bottom: 10px;
  }

  &__label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--color-primary-light);
    font-weight: 500;
  }

  &__steps {
    display: flex;
    gap: 4px;

    &--detail {
      flex-direction: column;
      gap: 8px;
    }
  }
}

.extract-step {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-radius: 6px;
  background: var(--color-bg-card);
  font-size: 11px;
  color: var(--color-text-muted);
  transition: all 0.3s ease;

  &__icon { font-size: 12px; }
  &__label { white-space: nowrap; }
  &__desc { font-size: 11px; color: var(--color-text-muted); margin-left: auto; }

  &--active {
    background: rgba(124, 140, 248, 0.15);
    color: var(--color-primary-light);
    box-shadow: 0 0 8px rgba(124, 140, 248, 0.2);
  }

  &--done {
    background: rgba(110, 231, 183, 0.1);
    color: var(--color-accent-green);
  }

  &--failed {
    background: rgba(239, 68, 68, 0.1);
    color: #EF4444;
  }
}

.pulse-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: pulse-glow 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;

  &--lg {
    width: 16px;
    height: 16px;
  }
}

@keyframes pulse-glow {
  0%, 100% { box-shadow: 0 0 4px rgba(124, 140, 248, 0.3); opacity: 1; }
  50% { box-shadow: 0 0 12px rgba(124, 140, 248, 0.6); opacity: 0.7; }
}

.extract-error {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 10px 0;
  padding: 10px 12px;
  background: rgba(239, 68, 68, 0.08);
  border-radius: 6px;
  font-size: 12px;
  color: #EF4444;

  &__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: rgba(239, 68, 68, 0.2);
    font-size: 10px;
    flex-shrink: 0;
  }
}

.ghost-btn {
  padding: 6px 14px;
  border: 1px solid var(--border-glow);
  border-radius: 6px;
  background: transparent;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  color: var(--color-text-secondary);

  &:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  &--danger:hover {
    border-color: #EF4444;
    color: #EF4444;
  }
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 80px 20px;

  &__icon { font-size: 48px; margin-bottom: 16px; }
  p { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 8px; }
  span { font-size: 13px; color: var(--color-text-muted); }
}

// 详情弹窗 - 提取中
.detail-extracting {
  text-align: center;
  padding: 20px 0;

  &__anim { margin-bottom: 16px; }
  h3 { color: var(--color-text-primary); font-size: 16px; margin-bottom: 8px; }
  p { color: var(--color-text-secondary); font-size: 13px; margin-bottom: 20px; }
}

.detail-error {
  text-align: center;
  padding: 20px 0;

  &__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: rgba(239, 68, 68, 0.15);
    color: #EF4444;
    font-size: 18px;
    margin-bottom: 12px;
  }

  h3 { color: #EF4444; font-size: 16px; margin-bottom: 8px; }
  p { color: var(--color-text-secondary); font-size: 13px; }
}

// 详情弹窗 - 正常
.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 20px;
}

.detail-item {
  &__label {
    display: block;
    font-size: 12px;
    color: var(--color-text-muted);
    margin-bottom: 4px;
  }

  &__value {
    font-size: 14px;
    color: var(--color-text-primary);
    font-weight: 500;

    &--accent { color: var(--color-accent-amber); }
  }
}

.status-text {
  &.status--draft { color: var(--color-text-muted); }
  &.status--reviewing { color: var(--color-accent-amber); }
  &.status--published { color: var(--color-accent-green); }
  &.status--rejected { color: #EF4444; }
  &.status--archived { color: var(--color-text-muted); }
}

.detail-section {
  margin-top: 20px;

  &__title {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 8px;
  }
}

.code-block {
  background: var(--color-bg-deep);
  padding: 14px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-family: var(--font-mono);
  overflow-x: auto;
  max-height: 200px;
}

.prompt-block {
  background: var(--color-bg-deep);
  padding: 16px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.anchor-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.anchor-item {
  display: flex;
  gap: 10px;
  padding: 10px 14px;
  background: var(--color-bg-deep);
  border-radius: 8px;
  border: 1px solid var(--border-glow);

  &__idx {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: rgba(124, 140, 248, 0.15);
    color: var(--color-primary-light);
    font-size: 11px;
    font-weight: 600;
    flex-shrink: 0;
  }

  &__text {
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    font-style: italic;
  }
}

.preview-block {
  background: var(--color-bg-deep);
  padding: 16px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.8;
}

// 评级徽章
.grade-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  margin-left: 4px;

  &--S { background: linear-gradient(135deg, #f59e0b, #ef4444); }
  &--A { background: #8b5cf6; }
  &--B { background: #3b82f6; }
  &--C { background: #6b7280; }
  &--D { background: #374151; }
}

// 多维质量评分
.quality-dimensions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.quality-dim {
  display: flex;
  align-items: center;
  gap: 12px;

  &__label {
    width: 80px;
    font-size: 13px;
    color: var(--color-text-secondary);
    flex-shrink: 0;
  }

  :deep(.el-progress) {
    flex: 1;
  }

  &__eval {
    font-size: 13px;
    color: var(--color-text-muted);
    padding: 8px 12px;
    background: var(--color-bg-deep);
    border-radius: 6px;
    border: 1px solid var(--border-glow);
  }
}
</style>
