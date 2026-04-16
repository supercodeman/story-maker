<!-- web/src/views/novel/ForeshadowPanel.vue -->
<template>
  <div class="foreshadow-panel">
    <div class="foreshadow-panel__banner">
      <span class="foreshadow-panel__banner-icon">🔮</span>
      <div class="foreshadow-panel__banner-text">
        <div class="foreshadow-panel__banner-title">伏笔追踪</div>
        <div class="foreshadow-panel__banner-desc">追踪小说中埋设的伏笔及其回收状态。「未揭示」表示伏笔已埋设但尚未在后续章节中揭开；「已回收」表示伏笔已在后续章节中揭开或呼应。帮助你避免遗漏伏笔，保持故事的逻辑自洽。</div>
      </div>
      <div class="foreshadow-panel__banner-actions">
        <el-tooltip content="已埋设但尚未在后续章节中揭开谜底的伏笔" placement="top">
          <el-tag size="small" type="warning">未揭示: {{ unresolvedCount }}</el-tag>
        </el-tooltip>
        <el-tooltip content="已在后续章节中揭开谜底或呼应的伏笔" placement="top">
          <el-tag size="small" type="success">已回收: {{ resolvedCount }}</el-tag>
        </el-tooltip>
        <el-button size="small" type="primary" text @click="handleAdd">+ 新增伏笔</el-button>
      </div>
    </div>

    <div v-if="overviewStore.foreshadows.length === 0" class="foreshadow-panel__empty">
      暂无伏笔数据，可通过 AI 提取或手动添加
    </div>

    <div class="foreshadow-panel__list">
      <div
        v-for="item in overviewStore.foreshadows"
        :key="item.id"
        class="foreshadow-card"
        :class="{ 'foreshadow-card--resolved': item.resolved }"
      >
        <div class="foreshadow-card__header">
          <span class="foreshadow-card__title">{{ item.title }}</span>
          <el-tooltip :content="item.resolved ? '该伏笔已在后续章节中揭开或呼应' : '该伏笔尚未揭开，等待后续章节回收'" placement="top">
            <el-tag :type="item.resolved ? 'success' : 'warning'" size="small">
              {{ item.resolved ? '已回收' : '未揭示' }}
            </el-tag>
          </el-tooltip>
        </div>
        <p class="foreshadow-card__content">{{ item.content }}</p>
        <div v-if="item.chapter_ref" class="foreshadow-card__refs">
          <span class="foreshadow-card__ref-label">埋设章节:</span>
          <el-tag
            v-for="ch in parseChapterRefs(item.chapter_ref)"
            :key="ch"
            size="small"
            type="info"
          >
            Ch.{{ ch }}
          </el-tag>
        </div>
        <div class="foreshadow-card__actions">
          <el-button size="small" text @click="handleEdit(item)">编辑</el-button>
          <el-tooltip v-if="!item.resolved" content="将此伏笔标记为已在后续章节中揭开或呼应" placement="top">
            <el-button
              size="small"
              text
              type="success"
              @click="handleResolve(item)"
            >
              标记回收
            </el-button>
          </el-tooltip>
          <el-tooltip v-else content="撤销回收状态，重新标记为未揭示" placement="top">
            <el-button
              size="small"
              text
              type="warning"
              @click="handleUnresolve(item)"
            >
              取消回收
            </el-button>
          </el-tooltip>
          <el-button size="small" text type="danger" @click="handleDelete(item)">删除</el-button>
        </div>
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="showDialog" :title="isEditing ? '编辑伏笔' : '新增伏笔'" width="500px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="标题">
          <el-input v-model="form.title" placeholder="伏笔标题" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.content" type="textarea" :rows="4" placeholder="伏笔描述" />
        </el-form-item>
        <el-form-item label="埋设章节">
          <el-input v-model="form.chapter_ref" placeholder="逗号分隔，如 2,5" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSave">{{ isEditing ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useOverviewStore } from '@/store/overview'
import { knowledgeApi } from '@/api/knowledge'
import type { NovelKnowledge } from '@/api/knowledge'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps<{ novelId: number }>()
const overviewStore = useOverviewStore()

const showDialog = ref(false)
const isEditing = ref(false)
const editingId = ref<number | null>(null)
const form = ref({ title: '', content: '', chapter_ref: '' })

const resolvedCount = computed(() => overviewStore.foreshadows.filter(f => f.resolved).length)
const unresolvedCount = computed(() => overviewStore.foreshadows.filter(f => !f.resolved).length)

function parseChapterRefs(refs: string): string[] {
  return refs.split(',').map(s => s.trim()).filter(Boolean)
}

function handleAdd() {
  isEditing.value = false
  editingId.value = null
  form.value = { title: '', content: '', chapter_ref: '' }
  showDialog.value = true
}

function handleEdit(item: NovelKnowledge) {
  isEditing.value = true
  editingId.value = item.id
  form.value = {
    title: item.title,
    content: item.content,
    chapter_ref: item.chapter_ref,
  }
  showDialog.value = true
}

async function handleSave() {
  if (!form.value.title) {
    ElMessage.warning('请输入标题')
    return
  }

  try {
    if (isEditing.value && editingId.value) {
      const oldItem = overviewStore.foreshadows.find(f => f.id === editingId.value)
      const updated: any = await knowledgeApi.update(editingId.value, {
        title: form.value.title,
        content: form.value.content,
        chapter_ref: form.value.chapter_ref,
      })
      const idx = overviewStore.foreshadows.findIndex(f => f.id === editingId.value)
      if (idx >= 0) overviewStore.foreshadows[idx] = updated
      overviewStore.addChange({
        type: 'foreshadow',
        action: 'update',
        id: editingId.value,
        data: updated,
        old_data: oldItem,
      })
      ElMessage.success('保存成功')
    } else {
      const item: any = await knowledgeApi.create(props.novelId, {
        category: 'foreshadow',
        title: form.value.title,
        content: form.value.content,
        chapter_ref: form.value.chapter_ref,
      })
      overviewStore.foreshadows.push(item)
      overviewStore.addChange({
        type: 'foreshadow',
        action: 'create',
        data: item,
      })
      ElMessage.success('创建成功')
    }
    showDialog.value = false
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleResolve(item: NovelKnowledge) {
  try {
    const updated: any = await knowledgeApi.update(item.id, { resolved: true } as any)
    const idx = overviewStore.foreshadows.findIndex(f => f.id === item.id)
    if (idx >= 0) {
      overviewStore.foreshadows[idx] = { ...overviewStore.foreshadows[idx], ...updated, resolved: true }
    }
    overviewStore.addChange({
      type: 'foreshadow',
      action: 'update',
      id: item.id,
      data: { resolved: true },
      old_data: { resolved: false },
    })
    ElMessage.success('已标记回收')
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleUnresolve(item: NovelKnowledge) {
  try {
    const updated: any = await knowledgeApi.update(item.id, { resolved: false } as any)
    const idx = overviewStore.foreshadows.findIndex(f => f.id === item.id)
    if (idx >= 0) {
      overviewStore.foreshadows[idx] = { ...overviewStore.foreshadows[idx], ...updated, resolved: false }
    }
    overviewStore.addChange({
      type: 'foreshadow',
      action: 'update',
      id: item.id,
      data: { resolved: false },
      old_data: { resolved: true },
    })
    ElMessage.success('已取消回收')
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleDelete(item: NovelKnowledge) {
  try {
    await ElMessageBox.confirm('确定删除该伏笔？', '确认')
    await knowledgeApi.delete(item.id)
    overviewStore.foreshadows = overviewStore.foreshadows.filter(f => f.id !== item.id)
    overviewStore.addChange({
      type: 'foreshadow',
      action: 'delete',
      id: item.id,
      old_data: item,
    })
    ElMessage.success('删除成功')
  } catch {
    // 用户取消
  }
}
</script>

<style scoped lang="scss">
.foreshadow-panel {
  padding: 0;

  &__banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: var(--el-color-primary-light-9);
    border-radius: 6px;
    margin-bottom: 8px;
  }

  &__banner-text { flex: 1; min-width: 0; }

  &__banner-icon {
    font-size: 18px;
    line-height: 1;
    flex-shrink: 0;
  }

  &__banner-title {
    font-weight: 600;
    font-size: 13px;
    color: var(--el-text-color-primary);
    margin-bottom: 2px;
  }

  &__banner-desc {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
  }

  &__banner-actions {
    margin-left: auto;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__toolbar {
    display: none;
  }

  &__stats {
    display: flex;
    gap: 8px;
  }

  &__empty {
    text-align: center;
    color: var(--el-text-color-secondary);
    padding: 40px 0;
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
}

.foreshadow-card {
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px 16px;
  border-left: 3px solid var(--el-color-warning);
  transition: opacity 0.2s;

  &--resolved {
    border-left-color: var(--el-color-success);
    opacity: 0.75;
  }

  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  &__title {
    font-weight: 600;
    font-size: 14px;
    color: var(--el-text-color-primary);
  }

  &__content {
    margin: 8px 0;
    font-size: 13px;
    color: var(--el-text-color-regular);
    line-height: 1.5;
  }

  &__refs {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-wrap: wrap;
    margin-bottom: 8px;
  }

  &__ref-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  &__actions {
    display: flex;
    gap: 4px;
    border-top: 1px solid var(--el-border-color-lighter);
    padding-top: 8px;
  }
}
</style>
