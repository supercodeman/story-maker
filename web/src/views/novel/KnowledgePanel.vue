<!-- web/src/views/novel/KnowledgePanel.vue -->
<template>
  <div class="knowledge-panel">
    <div class="knowledge-panel__header">
      <el-button size="small" text @click="showAddDialog = true">
        <el-icon><Plus /></el-icon> 添加
      </el-button>
    </div>

    <!-- 类别网格 -->
    <div class="kp-category-grid">
      <button
        v-for="cat in categories"
        :key="cat.value"
        class="kp-category-grid__item"
        :class="{ 'is-active': activeTab === cat.value }"
        @click="switchTab(cat.value)"
      >
        <el-icon class="kp-category-grid__icon"><component :is="categoryIconMap[cat.value]" /></el-icon>
        <span class="kp-category-grid__label">{{ cat.label }}</span>
      </button>
      <button
        class="kp-category-grid__item"
        :class="{ 'is-active': activeTab === '' }"
        @click="switchTab('')"
      >
        <el-icon class="kp-category-grid__icon"><Grid /></el-icon>
        <span class="kp-category-grid__label">全部</span>
      </button>
    </div>

    <!-- 详情区域：选中类别后展示 -->
    <template v-if="activeTab !== null">
      <!-- 搜索栏 -->
      <div class="knowledge-panel__search">
        <el-input
          v-model="searchKeyword"
          size="small"
          placeholder="搜索知识..."
          clearable
          @keyup.enter="handleSearch"
          @clear="handleClearSearch"
        />
      </div>

      <!-- 待审核提示 -->
      <div v-if="knowledgeStore.pendingCount > 0" class="knowledge-panel__pending-bar">
        <span>{{ knowledgeStore.pendingCount }} 条待审核</span>
        <el-button size="small" type="warning" text @click="handleBatchConfirm">批量确认</el-button>
      </div>

      <!-- AI 提取按钮 -->
      <div v-if="currentChapterId" class="knowledge-panel__extract">
        <el-button
          size="small"
          type="success"
          text bg
          :loading="knowledgeStore.extractPending"
          @click="handleExtract"
        >
          AI 提取知识
        </el-button>
      </div>

      <!-- 知识条目列表 -->
      <div class="knowledge-panel__list">
        <div v-if="knowledgeStore.loading" class="knowledge-panel__loading">
          <el-skeleton :rows="3" animated />
        </div>
        <template v-else>
          <div
            v-for="item in displayItems"
            :key="item.id"
            class="knowledge-item"
            :class="{ 'knowledge-item--pending': item.status === 'pending' }"
          >
            <div class="knowledge-item__header">
              <el-icon class="knowledge-item__category"><component :is="categoryIconMap[item.category] || Notebook" /></el-icon>
              <span class="knowledge-item__title">{{ item.title }}</span>
              <el-tag v-if="item.status === 'pending'" type="warning" size="small">待审核</el-tag>
            </div>
            <div class="knowledge-item__content">{{ item.content }}</div>
            <div v-if="item.tags" class="knowledge-item__tags">
              <el-tag v-for="tag in item.tags.split(',')" :key="tag" size="small" type="info">{{ tag.trim() }}</el-tag>
            </div>
            <div class="knowledge-item__actions">
              <el-button v-if="item.status === 'pending'" size="small" type="warning" text @click="handleConfirm(item.id)">确认</el-button>
              <el-button size="small" text @click="handleEdit(item)">编辑</el-button>
              <el-button size="small" type="danger" text @click="handleDelete(item.id)">删除</el-button>
            </div>
          </div>
          <div v-if="displayItems.length === 0" class="knowledge-panel__empty">
            暂无知识条目
          </div>
        </template>
      </div>
    </template>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="showAddDialog" :title="editingItem ? '编辑知识' : '添加知识'" width="500px">
      <el-form :model="formData" label-position="top">
        <el-form-item label="分类">
          <el-select v-model="formData.category" placeholder="选择分类">
            <el-option
              v-for="cat in categories"
              :key="cat.value"
              :label="`${cat.icon} ${cat.label}`"
              :value="cat.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="formData.title" placeholder="知识标题" maxlength="200" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="formData.content" type="textarea" :rows="4" placeholder="知识内容..." />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="formData.tags" placeholder="逗号分隔的标签" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="formData.priority" :min="0" :max="100" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { User, Compass, Reading, MagicStick, EditPen, Notebook, Grid, Plus } from '@element-plus/icons-vue'
import { useKnowledgeStore } from '@/store/knowledge'
import { knowledgeCategories } from '@/api/knowledge'
import type { NovelKnowledge } from '@/api/knowledge'

const categoryIconMap: Record<string, any> = {
  character: User,
  worldview: Compass,
  plotline: Reading,
  foreshadow: MagicStick,
  style: EditPen,
  custom: Notebook,
}

const props = defineProps<{
  novelId: number
  currentChapterId?: number
  modelName?: string
}>()

const knowledgeStore = useKnowledgeStore()
const categories = knowledgeCategories

const activeTab = ref<string | null>(null)
const searchKeyword = ref('')
const showAddDialog = ref(false)
const editingItem = ref<NovelKnowledge | null>(null)
const saving = ref(false)

const formData = ref({
  category: 'character',
  title: '',
  content: '',
  tags: '',
  priority: 0,
})

// 按当前 Tab 过滤显示（未选中类别时不展示）
const displayItems = computed(() => {
  if (activeTab.value === null) return []
  if (activeTab.value === '') return knowledgeStore.items
  return knowledgeStore.items.filter(i => i.category === activeTab.value)
})

function getCategoryIcon(category: string) {
  return categories.find(c => c.value === category)?.icon || '📌'
}

function switchTab(tab: string) {
  activeTab.value = activeTab.value === tab ? null : tab
}

// 加载知识条目
watch(() => props.novelId, (nid) => {
  if (nid) {
    knowledgeStore.fetchItems(nid)
  }
}, { immediate: true })

async function handleSearch() {
  if (!searchKeyword.value.trim()) return
  await knowledgeStore.searchItems(props.novelId, searchKeyword.value.trim())
}

async function handleClearSearch() {
  searchKeyword.value = ''
  await knowledgeStore.fetchItems(props.novelId)
}

function handleEdit(item: NovelKnowledge) {
  editingItem.value = item
  formData.value = {
    category: item.category,
    title: item.title,
    content: item.content,
    tags: item.tags || '',
    priority: item.priority,
  }
  showAddDialog.value = true
}

async function handleSave() {
  if (!formData.value.title || !formData.value.content) {
    ElMessage.warning('标题和内容不能为空')
    return
  }
  saving.value = true
  try {
    if (editingItem.value) {
      await knowledgeStore.updateItem(editingItem.value.id, formData.value)
      ElMessage.success('已更新')
    } else {
      await knowledgeStore.createItem(props.novelId, formData.value)
      ElMessage.success('已创建')
    }
    showAddDialog.value = false
    editingItem.value = null
    formData.value = { category: 'character', title: '', content: '', tags: '', priority: 0 }
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(kid: number) {
  await ElMessageBox.confirm('确定删除该知识条目？', '确认')
  await knowledgeStore.deleteItem(kid)
  ElMessage.success('已删除')
}

async function handleConfirm(kid: number) {
  await knowledgeStore.confirmItem(kid)
  ElMessage.success('已确认')
}

async function handleBatchConfirm() {
  await ElMessageBox.confirm('确认所有待审核条目？', '批量确认')
  await knowledgeStore.batchConfirm(props.novelId)
  ElMessage.success('全部已确认')
}

async function handleExtract() {
  if (!props.currentChapterId) return
  try {
    await knowledgeStore.extractFromChapter(props.novelId, props.currentChapterId, props.modelName)
    ElMessage.info('正在从章节中提取知识...')
  } catch (e: any) {
    ElMessage.error(e.message || '提取失败')
  }
}
</script>

<style lang="scss" scoped>
.knowledge-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  overflow: hidden;
  max-width: 100%;

  &__header {
    display: flex;
    justify-content: flex-end;
    align-items: center;
  }

  &__search {
    .el-input { font-size: 12px; }
  }

  &__pending-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 10px;
    border-radius: 8px;
    background: var(--el-color-warning-light-9);
    font-size: 12px;
    color: var(--el-color-warning);
  }

  &__extract {
    .el-button { width: 100%; }
  }

  &__list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  &__empty {
    text-align: center;
    color: var(--color-text-muted, #999);
    font-size: 13px;
    padding: 20px 0;
  }

  &__loading {
    padding: 12px;
  }
}

.knowledge-item {
  padding: 10px;
  border-radius: 8px;
  border: 1px solid var(--border-glow, #e4e7ed);
  background: var(--color-bg-surface, #fff);
  font-size: 13px;

  &--pending {
    border-color: var(--el-color-warning-light-5);
    background: var(--el-color-warning-light-9);
  }

  &__header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  &__category { font-size: 14px; color: var(--color-primary-light); }

  &__title {
    font-weight: 600;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--el-text-color-primary);
  }

  &__content {
    color: var(--color-text-secondary, #666);
    font-size: 12px;
    line-height: 1.6;
    max-height: 60px;
    overflow: hidden;
    margin-bottom: 6px;
  }

  &__tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-bottom: 6px;
    .el-tag { font-size: 11px; }
  }

  &__actions {
    display: flex;
    gap: 4px;
    .el-button { padding: 2px 6px; font-size: 11px; }
  }
}

// 类别网格
.kp-category-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;

  &__item {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    padding: 8px 4px;
    border-radius: 8px;
    border: 1px solid var(--border-glow);
    background: var(--color-bg-surface);
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.15s ease;
    font-size: 11px;
    line-height: 1;
    white-space: nowrap;
    min-width: 0;
    overflow: hidden;

    &:hover {
      color: var(--color-text-secondary);
      background: var(--color-bg-hover);
      border-color: rgba(124, 140, 248, 0.3);
    }

    &:active {
      transform: scale(0.96);
    }

    &.is-active {
      color: var(--color-primary-light);
      border-color: rgba(124, 140, 248, 0.35);
      background: rgba(124, 140, 248, 0.08);
    }
  }

  &__icon {
    font-size: 14px;
    flex-shrink: 0;
  }

  &__label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
