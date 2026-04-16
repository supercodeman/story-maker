<!-- web/src/views/portfolio/PortfolioDetail.vue -->
<template>
  <div class="portfolio-detail">
    <div class="page-header">
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">工作区</el-breadcrumb-item>
          <el-breadcrumb-item>{{ portfolio?.name || '...' }}</el-breadcrumb-item>
        </el-breadcrumb>
        <div class="page-title-row">
          <template v-if="editingName">
            <el-input
              ref="nameInputRef"
              v-model="editNameValue"
              size="large"
              maxlength="100"
              class="name-input"
              @keyup.enter="saveName"
              @keyup.escape="cancelEditName"
              @blur="saveName"
            />
          </template>
          <template v-else>
            <h1 class="page-title" @click="startEditName">{{ portfolio?.name || '加载中...' }}</h1>
            <span class="edit-icon" @click="startEditName">✎</span>
          </template>
        </div>
        <p class="page-desc">{{ portfolio?.description || '' }}</p>
      </div>
    </div>

    <div class="action-cards">
      <GlowCard hoverable class="action-card" @click="goToNovels">
        <div class="action-card__icon">📚</div>
        <h3>小说工坊</h3>
        <p>借助 AI 辅助创作小说</p>
      </GlowCard>
      <GlowCard hoverable class="action-card" @click="goToStudio">
        <div class="action-card__icon">🤖</div>
        <h3>AI 工作室</h3>
        <p>使用 AI 生成文本和图片</p>
      </GlowCard>
      <GlowCard hoverable class="action-card" @click="goToCharacters">
        <div class="action-card__icon">🎭</div>
        <h3>角色管理</h3>
        <p>管理该作品集的角色</p>
      </GlowCard>
      
    </div>

    <div class="section">
      <h2 class="section-title">最近的 AI 任务</h2>
      <div v-loading="tasksLoading" class="task-list">
        <div v-if="tasks.length === 0" class="empty-state">暂无任务，前往 AI 工作室创建一个吧。</div>
        <GlowCard v-for="task in tasks" :key="task.id" class="task-card">
          <div class="task-card__header">
            <el-tag :type="statusTagType(task.status)" size="small">{{ task.status }}</el-tag>
            <span class="task-card__type">{{ task.task_type }}</span>
          </div>
          <p class="task-card__prompt">{{ task.prompt }}</p>
          <div v-if="task.status === 'completed' && task.result" class="task-card__result">
            <TaskResult :task="task" />
          </div>
          <div class="task-card__footer">{{ formatDate(task.created_at) }}</div>
        </GlowCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { portfolioApi } from '@/api/portfolio'
import { aiApi } from '@/api/ai'
import type { Portfolio } from '@/api/portfolio'
import type { AITask } from '@/api/ai'
import GlowCard from '@/components/common/GlowCard.vue'
import TaskResult from '@/components/ai/TaskResult.vue'

const props = defineProps<{ id: string; pid: string }>()
const router = useRouter()

const portfolio = ref<Portfolio | null>(null)
const tasks = ref<AITask[]>([])
const tasksLoading = ref(false)

const editingName = ref(false)
const editNameValue = ref('')
const nameInputRef = ref<InstanceType<typeof import('element-plus')['ElInput']> | null>(null)

function startEditName() {
  if (!portfolio.value) return
  editNameValue.value = portfolio.value.name
  editingName.value = true
  nextTick(() => nameInputRef.value?.focus())
}

function cancelEditName() {
  editingName.value = false
}

async function saveName() {
  if (!editingName.value) return
  const trimmed = editNameValue.value.trim()
  if (!trimmed || !portfolio.value || trimmed === portfolio.value.name) {
    editingName.value = false
    return
  }
  try {
    await portfolioApi.update(portfolio.value.id, { name: trimmed })
    portfolio.value.name = trimmed
    ElMessage.success('名称已更新')
  } catch {
    ElMessage.error('更新失败')
  }
  editingName.value = false
}

onMounted(async () => {
  try {
    portfolio.value = await portfolioApi.get(Number(props.pid)) as any
  } catch { ElMessage.error('加载作品集失败') }

  tasksLoading.value = true
  try {
    const data: any = await aiApi.listTasks({ portfolio_id: Number(props.pid) })
    tasks.value = data.tasks || []
  } finally { tasksLoading.value = false }
})

function goToCharacters() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/characters`)
}
function goToStudio() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/studio`)
}
function goToNovels() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/novels`)
}
function statusTagType(s: string) {
  const map: Record<string, string> = { completed: 'success', running: 'warning', failed: 'danger', pending: 'info' }
  return map[s] || 'info'
}
function formatDate(d: string) { return new Date(d).toLocaleString() }
</script>

<style scoped lang="scss">
.portfolio-detail { width: 100%; max-width: 1200px; margin: 0 auto; }
.page-header {
  margin-bottom: 24px; padding: 14px 20px; background: var(--color-bg-card); border-radius: 12px;
  border: 1px solid var(--border-glow); box-shadow: var(--shadow-sm);
  transition: box-shadow 0.3s ease;
  &:hover { box-shadow: var(--shadow-md); }
}
.page-title { font-size: 22px; font-weight: 700; color: var(--color-text-primary); margin-top: 8px; cursor: pointer;
  &:hover { color: #6378ff; }
}
.page-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}
.page-title-row .page-title { margin-top: 0; }
.edit-icon {
  font-size: 16px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: color 0.2s;
  &:hover { color: var(--color-primary-light); }
}
.name-input {
  max-width: 400px;
  :deep(.el-input__wrapper) {
    font-size: 24px;
    font-weight: 700;
  }
}
.page-desc { font-size: 14px; color: var(--color-text-secondary); margin-top: 4px; }
.action-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 20px; margin-bottom: 40px; }
.action-card { cursor: pointer; text-align: center; padding: 24px;
  &__icon { font-size: 36px; margin-bottom: 12px; }
  h3 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
  p { font-size: 13px; color: var(--color-text-secondary); }
}
.section { margin-bottom: 32px; }
.section-title { font-size: 20px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 16px; }
.task-list { display: flex; flex-direction: column; gap: 12px; }
.empty-state { text-align: center; padding: 40px; color: var(--color-text-muted); font-size: 14px; }
.task-card {
  &__header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
  &__type { font-size: 12px; color: var(--color-text-muted); }
  &__prompt { font-size: 14px; color: var(--color-text-secondary); margin-bottom: 8px; white-space: pre-wrap; max-height: 80px; overflow: hidden; }
  &__result { margin-bottom: 8px; }
  &__footer { font-size: 12px; color: var(--color-text-muted); }
}
</style>
