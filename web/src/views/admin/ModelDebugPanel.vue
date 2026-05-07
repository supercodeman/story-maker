<!-- web/src/views/admin/ModelDebugPanel.vue -->
<!-- 模型调试面板：展示各 Provider/模型的可用性状态，支持新增模型、手动测试、删除 -->
<template>
  <div class="model-debug">
    <div class="page-header">
      <div>
        <h1 class="page-title">模型调试面板</h1>
        <p class="page-desc">查看各 Key 对应模型的可用性状态，手动触发健康检查</p>
      </div>
      <div class="header-actions">
        <button class="btn-add" @click="showAddDialog = true">+ 新增模型</button>
        <button class="btn-check" :disabled="checking" @click="handleHealthCheck">
          <span v-if="checking" class="spinner" />
          {{ checking ? '检测中...' : '一键健康检查' }}
        </button>
      </div>
    </div>

    <div v-loading="loading" class="provider-grid">
      <FadePanel v-for="group in groupedData" :key="group.provider" class="provider-card">
        <div class="provider-header">
          <span class="provider-name">{{ group.displayName }}</span>
          <span class="provider-tag" :class="group.allAvailable ? 'tag--ok' : 'tag--err'">
            {{ group.allAvailable ? '全部可用' : '部分异常' }}
          </span>
        </div>

        <div class="model-table">
          <div class="model-table__header">
            <span class="col col--model">模型</span>
            <span class="col col--cap">能力</span>
            <span class="col col--status">状态</span>
            <span class="col col--latency">延迟</span>
            <span class="col col--priority">优先级</span>
            <span class="col col--time">最后检查</span>
            <span class="col col--error">错误信息</span>
            <span class="col col--actions">操作</span>
          </div>
          <div v-for="item in group.items" :key="item.id" class="model-table__row">
            <span class="col col--model">{{ item.model_name || '默认' }}</span>
            <span class="col col--cap">
              <span class="cap-badge">{{ capLabel(item.capability) }}</span>
            </span>
            <span class="col col--status">
              <span class="dot" :class="item.available ? 'dot--ok' : 'dot--err'" />
              {{ item.available ? '可用' : '不可用' }}
            </span>
            <span class="col col--latency">
              {{ item.latency_ms > 0 ? item.latency_ms + 'ms' : '-' }}
            </span>
            <span class="col col--priority">
              <el-input-number
                :model-value="item.priority"
                :min="0"
                :max="999"
                :step="1"
                size="small"
                controls-position="right"
                @change="(val: number) => handlePriorityChange(item, val ?? 0)"
              />
            </span>
            <span class="col col--time">{{ item.last_check || '未检测' }}</span>
            <span class="col col--error" :title="item.last_error">
              {{ item.last_error || '-' }}
            </span>
            <span class="col col--actions">
              <button
                class="action-btn action-btn--test"
                :disabled="testingId === item.id"
                @click="handleTestModel(item)"
              >
                {{ testingId === item.id ? '测试中' : '测试' }}
              </button>
              <button class="action-btn action-btn--delete" @click="handleDeleteModel(item)">删除</button>
            </span>
          </div>
        </div>
      </FadePanel>

      <div v-if="!loading && groupedData.length === 0" class="empty-state">
        暂无模型状态数据
      </div>
    </div>

    <!-- 新增模型对话框 -->
    <el-dialog v-model="showAddDialog" title="新增模型" width="480px" :close-on-click-modal="false">
      <el-form :model="addForm" label-width="80px">
        <el-form-item label="Provider">
          <el-select v-model="addForm.provider" placeholder="选择 Provider" style="width: 100%">
            <el-option v-for="p in providerOptions" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型名称">
          <el-input v-model="addForm.model_name" placeholder="留空表示 Provider 默认模型" />
        </el-form-item>
        <el-form-item label="能力">
          <el-select v-model="addForm.capability" placeholder="选择能力" style="width: 100%">
            <el-option v-for="c in capOptions" :key="c.value" :label="c.label" :value="c.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <button class="action-btn action-btn--cancel" @click="showAddDialog = false">取消</button>
        <button class="action-btn action-btn--confirm" :disabled="adding" @click="handleAddModel">
          {{ adding ? '添加中...' : '确认添加' }}
        </button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getModelStatus, triggerHealthCheck, addModel, deleteModel, testModel, updateModelPriority } from '@/api/model'
import type { ModelStatusDetail } from '@/api/model'
import FadePanel from '@/components/common/FadePanel.vue'

const loading = ref(false)
const checking = ref(false)
const adding = ref(false)
const testingId = ref<number | null>(null)
const showAddDialog = ref(false)
const statusList = ref<ModelStatusDetail[]>([])

const addForm = reactive({
  provider: '',
  model_name: '',
  capability: '',
})

// 能力标签映射
const capMap: Record<string, string> = {
  text_gen: '文本生成',
  text_polish: '文本润色',
  embedding: '向量化',
  image_gen: '图片生成',
  image_edit: '图片编辑',
}

function capLabel(cap: string) {
  return capMap[cap] || cap
}

// Provider 显示名映射
const providerMap: Record<string, string> = {
  minimax: 'Minimax',
  qwen: '通义千问',
  zhipu: '智谱 AI',
  deepseek: 'DeepSeek',
  kimi: 'Kimi',
}

const providerOptions = [
  { value: 'minimax', label: 'Minimax' },
  { value: 'qwen', label: '通义千问' },
  { value: 'zhipu', label: '智谱 AI' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'kimi', label: 'Kimi' },
]

const capOptions = [
  { value: 'text_gen', label: '文本生成' },
  { value: 'text_polish', label: '文本润色' },
  { value: 'embedding', label: '向量化' },
  { value: 'image_gen', label: '图片生成' },
  { value: 'image_edit', label: '图片编辑' },
]

// 按 Provider 分组
const groupedData = computed(() => {
  const map = new Map<string, ModelStatusDetail[]>()
  for (const item of statusList.value) {
    const list = map.get(item.provider) || []
    list.push(item)
    map.set(item.provider, list)
  }

  const groups: { provider: string; displayName: string; allAvailable: boolean; items: ModelStatusDetail[] }[] = []
  const order = ['minimax', 'qwen', 'zhipu', 'deepseek', 'kimi']
  const sorted = [...map.keys()].sort((a, b) => {
    const ia = order.indexOf(a)
    const ib = order.indexOf(b)
    return (ia === -1 ? 999 : ia) - (ib === -1 ? 999 : ib)
  })

  for (const provider of sorted) {
    const items = map.get(provider)!
    groups.push({
      provider,
      displayName: providerMap[provider] || provider,
      allAvailable: items.every((i) => i.available),
      items,
    })
  }
  return groups
})

async function loadStatus() {
  loading.value = true
  try {
    const data: any = await getModelStatus()
    statusList.value = Array.isArray(data) ? data : []
  } catch {
    ElMessage.error('加载模型状态失败')
  } finally {
    loading.value = false
  }
}

async function handleHealthCheck() {
  checking.value = true
  try {
    await triggerHealthCheck()
    ElMessage.success('健康检查完成，正在刷新...')
    await loadStatus()
  } catch {
    ElMessage.error('健康检查失败')
  } finally {
    checking.value = false
  }
}

async function handleAddModel() {
  if (!addForm.provider || !addForm.capability) {
    ElMessage.warning('请选择 Provider 和能力')
    return
  }
  adding.value = true
  try {
    await addModel({
      provider: addForm.provider,
      model_name: addForm.model_name,
      capability: addForm.capability,
    })
    ElMessage.success('模型添加成功')
    showAddDialog.value = false
    addForm.provider = ''
    addForm.model_name = ''
    addForm.capability = ''
    await loadStatus()
  } catch {
    ElMessage.error('添加模型失败')
  } finally {
    adding.value = false
  }
}

async function handleDeleteModel(item: ModelStatusDetail) {
  const name = item.model_name || '默认'
  try {
    await ElMessageBox.confirm(
      `确定删除 ${providerMap[item.provider] || item.provider} 的「${name}」(${capLabel(item.capability)}) 吗？`,
      '确认删除',
    )
    await deleteModel(item.id)
    ElMessage.success('删除成功')
    await loadStatus()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

async function handleTestModel(item: ModelStatusDetail) {
  testingId.value = item.id
  try {
    const result: any = await testModel({
      provider: item.provider,
      model_name: item.model_name,
      capability: item.capability,
    })
    if (result?.available) {
      ElMessage.success(`测试通过，延迟 ${result.latency_ms}ms`)
    } else {
      ElMessage.warning(`测试失败: ${result?.error || '未知错误'}`)
    }
    await loadStatus()
  } catch {
    ElMessage.error('测试请求失败')
  } finally {
    testingId.value = null
  }
}

async function handlePriorityChange(item: ModelStatusDetail, newVal: number) {
  try {
    await updateModelPriority(item.id, newVal)
    item.priority = newVal
    ElMessage.success('优先级已更新')
  } catch {
    ElMessage.error('更新优先级失败')
  }
}

onMounted(() => loadStatus())
</script>

<style scoped lang="scss">
.model-debug {
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

.header-actions {
  display: flex;
  gap: 10px;
}

.btn-add {
  padding: 10px 20px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  background: var(--color-bg-surface);
  color: var(--color-text-primary);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }
}

.btn-check {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  border-radius: 8px;
  border: none;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  color: white;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;

  &:hover:not(:disabled) {
    opacity: 0.9;
    transform: translateY(-1px);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.provider-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.provider-card {
  padding: 0;
  overflow: hidden;
}

.provider-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-glow);
}

.provider-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.provider-tag {
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;

  &.tag--ok {
    background: rgba(52, 211, 153, 0.1);
    color: #34d399;
  }

  &.tag--err {
    background: rgba(248, 113, 113, 0.1);
    color: #f87171;
  }
}

.model-table {
  width: 100%;

  &__header {
    display: flex;
    align-items: center;
    padding: 10px 24px;
    background: var(--color-bg-hover);

    .col {
      font-size: 12px;
      font-weight: 600;
      color: var(--color-text-muted);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
  }

  &__row {
    display: flex;
    align-items: center;
    padding: 12px 24px;
    border-bottom: 1px solid var(--border-light);
    transition: background 0.15s;

    &:hover {
      background: rgba(124, 140, 248, 0.03);
    }

    &:last-child {
      border-bottom: none;
    }

    .col {
      font-size: 13px;
      color: var(--color-text-secondary);
    }
  }
}

.col {
  &--model { width: 140px; font-weight: 500; }
  &--cap { width: 90px; }
  &--status { width: 80px; display: flex; align-items: center; gap: 6px; }
  &--latency { width: 70px; }
  &--priority { width: 110px; }
  &--time { width: 150px; }
  &--error { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &--actions { width: 120px; display: flex; gap: 6px; flex-shrink: 0; }
}

// 限制优先级输入框宽度
.col--priority :deep(.el-input-number) {
  width: 90px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;

  &--ok {
    background: #34d399;
    box-shadow: 0 0 6px rgba(52, 211, 153, 0.4);
  }

  &--err {
    background: #f87171;
    box-shadow: 0 0 6px rgba(248, 113, 113, 0.4);
  }
}

.cap-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: var(--color-bg-hover);
  color: var(--color-text-muted);
}

.action-btn {
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  border: 1px solid var(--border-glow);
  background: var(--color-bg-surface);
  color: var(--color-text-secondary);
  transition: all 0.2s;

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &--test {
    &:hover:not(:disabled) {
      border-color: var(--color-primary);
      color: var(--color-primary);
    }
  }

  &--delete {
    &:hover {
      border-color: #f87171;
      color: #f87171;
    }
  }

  &--cancel {
    padding: 8px 16px;
    margin-right: 8px;
  }

  &--confirm {
    padding: 8px 16px;
    background: var(--color-primary);
    color: white;
    border-color: var(--color-primary);

    &:hover:not(:disabled) {
      opacity: 0.9;
    }
  }
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 14px;
}
</style>
