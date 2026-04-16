<!-- web/src/views/novel/RevisionDialog.vue -->
<template>
  <el-dialog
    v-model="visible"
    title="变更确认与执行"
    width="700px"
    :close-on-click-modal="false"
    destroy-on-close
    @closed="handleClosed"
  >
    <!-- 阶段 1: 展示变更列表 -->
    <template v-if="stage === 'confirm'">
      <p style="margin-bottom: 12px; color: var(--el-text-color-secondary);">
        以下变更将提交给 AI 分析，确定受影响的章节并生成修改计划。
      </p>
      <div class="revision-changes">
        <div v-for="(change, i) in overviewStore.pendingChanges" :key="i" class="revision-change-item">
          <el-tag :type="actionTagType(change.action)" size="small">{{ actionLabel(change.action) }}</el-tag>
          <el-tag type="info" size="small">{{ change.type }}</el-tag>
          <span class="revision-change-item__desc">
            {{ changeDescription(change) }}
          </span>
        </div>
      </div>
      <div class="revision-model-select">
        <label>AI 模型:</label>
        <ModelSelector v-model="modelName" capability="text_gen" size="small" />
      </div>
    </template>

    <!-- 阶段 2: 分析工作流进度 -->
    <template v-if="stage === 'analyzing'">
      <div class="revision-progress">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
        <p>AI 正在分析变更影响...</p>
        <el-progress :percentage="analysisProgress" :stroke-width="6" style="margin-top: 16px;" />
      </div>
    </template>

    <!-- 阶段 3: 展示修改计划 -->
    <template v-if="stage === 'plan'">
      <p style="margin-bottom: 12px; color: var(--el-text-color-secondary);">
        AI 已完成分析，以下是修改计划。确认后将开始执行章节修改。
      </p>
      <div class="revision-plan">
        <pre>{{ overviewStore.revisionPlan }}</pre>
      </div>
    </template>

    <!-- 阶段 4: 执行工作流进度 -->
    <template v-if="stage === 'executing'">
      <div class="revision-progress">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
        <p>AI 正在执行章节修改...</p>
        <el-progress :percentage="executeProgress" :stroke-width="6" style="margin-top: 16px;" />
      </div>
    </template>

    <!-- 阶段 5: 完成 -->
    <template v-if="stage === 'done'">
      <div class="revision-done">
        <el-result icon="success" title="变更执行完成" sub-title="受影响的章节已更新，请刷新页面查看" />
      </div>
    </template>

    <template #footer>
      <template v-if="stage === 'confirm'">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" :loading="overviewStore.revisionPending" @click="handleSubmit">
          提交分析
        </el-button>
      </template>
      <template v-if="stage === 'plan'">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" :loading="overviewStore.executePending" @click="handleExecute">
          确认执行
        </el-button>
      </template>
      <template v-if="stage === 'done'">
        <el-button type="primary" @click="handleDone">完成</el-button>
      </template>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import { useOverviewStore } from '@/store/overview'
import type { OverviewChange } from '@/api/overview'
import { ElMessage } from 'element-plus'
import ModelSelector from '@/components/common/ModelSelector.vue'

const props = defineProps<{
  novelId: number
  portfolioId: number
}>()

const visible = defineModel<boolean>('modelValue', { default: false })
const overviewStore = useOverviewStore()

const stage = ref<'confirm' | 'analyzing' | 'plan' | 'executing' | 'done'>('confirm')
const modelName = ref('qwen')
const analysisProgress = ref(0)
const executeProgress = ref(0)

// 监听分析工作流完成
watch(() => overviewStore.revisionPlan, (plan) => {
  if (plan && stage.value === 'analyzing') {
    stage.value = 'plan'
  }
})

// 监听执行工作流完成
watch(() => overviewStore.executePending, (pending) => {
  if (!pending && stage.value === 'executing') {
    stage.value = 'done'
  }
})

function actionTagType(action: string) {
  switch (action) {
    case 'create': return 'success'
    case 'update': return 'warning'
    case 'delete': return 'danger'
    default: return 'info'
  }
}

function actionLabel(action: string) {
  switch (action) {
    case 'create': return '新增'
    case 'update': return '修改'
    case 'delete': return '删除'
    default: return action
  }
}

function changeDescription(change: OverviewChange): string {
  if (change.data && typeof change.data === 'object' && 'title' in change.data) {
    return (change.data as any).title
  }
  if (change.old_data && typeof change.old_data === 'object' && 'title' in change.old_data) {
    return (change.old_data as any).title
  }
  return `ID: ${change.id || '新建'}`
}

async function handleSubmit() {
  try {
    stage.value = 'analyzing'
    analysisProgress.value = 30
    await overviewStore.submitRevision(props.novelId, props.portfolioId, modelName.value)
    // 模拟进度（实际由 WebSocket 推送）
    analysisProgress.value = 60
  } catch {
    stage.value = 'confirm'
    ElMessage.error('提交分析失败')
  }
}

async function handleExecute() {
  try {
    stage.value = 'executing'
    executeProgress.value = 30
    await overviewStore.executeRevision(props.novelId, props.portfolioId, modelName.value)
    executeProgress.value = 60
  } catch {
    stage.value = 'plan'
    ElMessage.error('执行失败')
  }
}

function handleClose() {
  visible.value = false
}

function handleClosed() {
  stage.value = 'confirm'
  analysisProgress.value = 0
  executeProgress.value = 0
}

function handleDone() {
  overviewStore.clearChanges()
  visible.value = false
  stage.value = 'confirm'
}
</script>

<style scoped lang="scss">
.revision-changes {
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 8px;
}

.revision-change-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid var(--el-border-color-extra-light);

  &:last-child {
    border-bottom: none;
  }

  &__desc {
    font-size: 13px;
    color: var(--el-text-color-regular);
  }
}

.revision-model-select {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;

  label {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }
}

.revision-progress {
  text-align: center;
  padding: 40px 20px;

  p {
    margin-top: 12px;
    color: var(--el-text-color-secondary);
  }
}

.revision-plan {
  max-height: 400px;
  overflow-y: auto;
  background: var(--el-bg-color-page);
  border-radius: 6px;
  padding: 12px;

  pre {
    white-space: pre-wrap;
    word-break: break-word;
    font-size: 13px;
    line-height: 1.6;
    margin: 0;
  }
}

.revision-done {
  padding: 20px 0;
}
</style>
