<!-- web/src/views/novel/HitAnalysisDialog.vue -->
<template>
  <el-dialog
    v-model="visible"
    title="爆款拆解"
    width="700px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <!-- 提交表单 -->
    <div v-if="!currentAnalysis">
      <el-alert
        type="info"
        :closable="false"
        style="margin-bottom: 16px"
      >
        <template #title>
          输入小说标题和简介/梗概（建议不超过10000字），AI 将从结构、节奏、人物三个维度进行拆解分析。
          分析结果仅提取抽象技法，不会引用原文内容。
        </template>
      </el-alert>

      <el-form :model="form" label-position="top">
        <el-form-item label="小说标题" required>
          <el-input v-model="form.title" placeholder="输入被拆解的小说标题" maxlength="200" />
        </el-form-item>
        <el-form-item label="作者">
          <el-input v-model="form.author" placeholder="作者名（可选）" maxlength="100" />
        </el-form-item>
        <el-form-item label="分析素材" required>
          <el-input
            v-model="form.sourceText"
            type="textarea"
            :rows="8"
            placeholder="输入小说简介、梗概或片段（建议使用简介/梗概，不超过10000字）"
            show-word-limit
            maxlength="10000"
          />
        </el-form-item>
        <el-form-item label="AI 模型">
          <ModelSelector v-model="form.modelName" capability="text_gen" size="small" />
        </el-form-item>
      </el-form>

      <!-- 历史记录 -->
      <div v-if="historyList.length > 0" style="margin-top: 16px">
        <h4 style="margin-bottom: 8px; color: var(--el-text-color-secondary)">历史拆解记录</h4>
        <div class="history-list">
          <div
            v-for="item in historyList"
            :key="item.id"
            class="history-item"
            @click="loadAnalysis(item.id)"
          >
            <div class="history-item__title">{{ item.title }}</div>
            <div class="history-item__meta">
              <el-tag :type="statusTagType(item.status)" size="small">{{ statusLabel(item.status) }}</el-tag>
              <span>{{ item.author }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 拆解报告展示 -->
    <div v-else>
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
        <h3 style="margin: 0">{{ currentAnalysis.title }} <span style="color: var(--el-text-color-secondary); font-size: 14px">{{ currentAnalysis.author }}</span></h3>
        <el-button text @click="currentAnalysis = null">返回</el-button>
      </div>

      <div v-if="currentAnalysis.status === 'running'" style="text-align: center; padding: 40px 0">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <p style="margin-top: 12px; color: var(--el-text-color-secondary)">AI 正在分析中，请稍候...</p>
        <el-button text @click="refreshAnalysis">刷新状态</el-button>
      </div>

      <div v-else-if="currentAnalysis.status === 'completed' && parsedReport">
        <div class="report-section" v-if="parsedReport.structure_analysis">
          <h4>剧情结构分析</h4>
          <p>{{ parsedReport.structure_analysis }}</p>
        </div>
        <div class="report-section" v-if="parsedReport.rhythm_analysis">
          <h4>节奏分析</h4>
          <p>{{ parsedReport.rhythm_analysis }}</p>
        </div>
        <div class="report-section" v-if="parsedReport.character_arcs">
          <h4>人物弧线分析</h4>
          <p>{{ parsedReport.character_arcs }}</p>
        </div>
        <div class="report-section" v-if="parsedReport.hook_points?.length">
          <h4>钩子/爆点分布</h4>
          <div v-for="(hook, idx) in parsedReport.hook_points" :key="idx" class="hook-item">
            <el-tag size="small" type="warning">{{ hook.position }}</el-tag>
            <span class="hook-type">{{ hook.type }}</span>
            <span class="hook-technique">{{ hook.technique }}</span>
          </div>
        </div>
        <div class="report-section" v-if="parsedReport.style_features">
          <h4>文风特征</h4>
          <p>{{ parsedReport.style_features }}</p>
        </div>
        <div class="report-section" v-if="parsedReport.summary">
          <h4>综合技法总结</h4>
          <p>{{ parsedReport.summary }}</p>
        </div>
      </div>

      <div v-else-if="currentAnalysis.status === 'failed'" style="text-align: center; padding: 40px 0">
        <el-result icon="error" title="分析失败" sub-title="请重试或更换模型" />
      </div>
    </div>

    <template #footer>
      <div v-if="!currentAnalysis">
        <el-button @click="visible = false">Cancel</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          :disabled="!form.title.trim() || !form.sourceText.trim()"
          @click="handleSubmit"
        >
          开始拆解
        </el-button>
      </div>
      <div v-else-if="currentAnalysis.status === 'completed'">
        <el-button @click="visible = false">Close</el-button>
        <el-button type="primary" @click="handleImportToOutline">
          导入到大纲生成
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { hitAnalysisApi, type HitAnalysis, type HitAnalysisReport } from '@/api/hit_analysis'
import ModelSelector from '@/components/common/ModelSelector.vue'

const props = defineProps<{
  portfolioId: number
}>()

const emit = defineEmits<{
  (e: 'import', analysisId: number): void
}>()

const visible = ref(false)
const submitting = ref(false)
const historyList = ref<HitAnalysis[]>([])
const currentAnalysis = ref<HitAnalysis | null>(null)

const form = ref({
  title: '',
  author: '',
  sourceText: '',
  modelName: 'zhipu',
})

const parsedReport = computed<HitAnalysisReport | null>(() => {
  if (!currentAnalysis.value?.report) return null
  try {
    return JSON.parse(currentAnalysis.value.report)
  } catch {
    return null
  }
})

const statusTagType = (status: string) => {
  const map: Record<string, string> = { pending: 'info', running: 'warning', completed: 'success', failed: 'danger' }
  return map[status] || 'info'
}

const statusLabel = (status: string) => {
  const map: Record<string, string> = { pending: '等待中', running: '分析中', completed: '已完成', failed: '失败' }
  return map[status] || status
}

const open = async () => {
  visible.value = true
  currentAnalysis.value = null
  form.value = { title: '', author: '', sourceText: '', modelName: 'zhipu' }
  await loadHistory()
}

const handleClose = () => {
  currentAnalysis.value = null
}

const loadHistory = async () => {
  try {
    const res = await hitAnalysisApi.list()
    historyList.value = res.data?.data || []
  } catch {
    historyList.value = []
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    const res = await hitAnalysisApi.submit({
      portfolio_id: props.portfolioId,
      title: form.value.title,
      author: form.value.author,
      source_text: form.value.sourceText,
      model_name: form.value.modelName,
    })
    const ha = res.data?.data
    if (ha) {
      currentAnalysis.value = ha
      ElMessage.success('拆解任务已提交')
      await loadHistory()
    }
  } catch (err: any) {
    ElMessage.error(err?.response?.data?.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

const loadAnalysis = async (id: number) => {
  try {
    const res = await hitAnalysisApi.get(id)
    currentAnalysis.value = res.data?.data || null
  } catch {
    ElMessage.error('加载失败')
  }
}

const refreshAnalysis = async () => {
  if (!currentAnalysis.value) return
  await loadAnalysis(currentAnalysis.value.id)
}

const handleImportToOutline = () => {
  if (currentAnalysis.value) {
    emit('import', currentAnalysis.value.id)
    visible.value = false
  }
}

defineExpose({ open })
</script>

<style scoped>
.history-list { display: flex; flex-direction: column; gap: 8px; max-height: 200px; overflow-y: auto; }
.history-item {
  padding: 8px 12px; border: 1px solid var(--el-border-color-lighter); border-radius: 6px;
  cursor: pointer; transition: border-color 0.2s;
}
.history-item:hover { border-color: var(--el-color-primary); }
.history-item__title { font-weight: 500; margin-bottom: 4px; }
.history-item__meta { display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--el-text-color-secondary); }

.report-section { margin-bottom: 16px; padding: 12px; background: var(--el-fill-color-lighter); border-radius: 6px; }
.report-section h4 { margin: 0 0 8px; color: var(--el-color-primary); font-size: 14px; }
.report-section p { margin: 0; line-height: 1.6; font-size: 13px; white-space: pre-wrap; }

.hook-item { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; font-size: 13px; }
.hook-type { font-weight: 500; }
.hook-technique { color: var(--el-text-color-secondary); }
</style>
