<!-- web/src/views/novel/NovelList.vue -->
<template>
  <div class="novel-list">
    <div class="page-header">
      <div class="page-header__left">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">Workspace</el-breadcrumb-item>
          <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">Portfolio</el-breadcrumb-item>
          <el-breadcrumb-item>Novels</el-breadcrumb-item>
        </el-breadcrumb>
        <h1 class="page-title">Novel Workshop</h1>
        <p class="page-desc">创建和管理你的小说作品</p>
      </div>
      <div class="page-header__actions">
        <div class="view-toggle">
          <button class="view-toggle__btn" :class="{ active: viewMode === 'grid' }" @click="viewMode = 'grid'" title="卡片视图">▦</button>
          <button class="view-toggle__btn" :class="{ active: viewMode === 'list' }" @click="viewMode = 'list'" title="列表视图">☰</button>
        </div>
        <el-button type="primary" @click="showCreateDialog = true">+ New Novel</el-button>
        <el-button type="success" text bg @click="goToOutline">AI Generate Outline</el-button>
        <el-button type="warning" text bg @click="goToButler">小说管家</el-button>
      </div>
    </div>

    <!-- 卡片视图 -->
    <div v-if="viewMode === 'grid'" v-loading="loading" class="novel-grid">
      <div v-if="novels.length === 0 && !loading" class="empty-state">
        No novels yet. Create your first novel to get started.
      </div>
      <GlowCard
        v-for="novel in novels"
        :key="novel.id"
        hoverable
        class="novel-card"
        @click="goToWorkshop(novel.id)"
      >
        <div class="novel-card__status">
          <el-tag :type="statusType(novel.status)" size="small">{{ novel.status }}</el-tag>
        </div>
        <h3 class="novel-card__title">{{ novel.title }}</h3>
        <p class="novel-card__desc">{{ novel.description || 'No description' }}</p>
        <div class="novel-card__meta">
          <span>{{ novel.chapter_count }} chapters</span>
          <span>{{ novel.word_count }} words</span>
        </div>
        <div class="novel-card__footer">
          <span>{{ formatDate(novel.updated_at) }}</span>
          <el-button type="danger" size="small" text @click.stop="handleDelete(novel.id)">Delete</el-button>
        </div>
      </GlowCard>
    </div>

    <!-- 列表视图 -->
    <div v-else v-loading="loading" class="novel-table">
      <div v-if="novels.length === 0 && !loading" class="empty-state">
        No novels yet. Create your first novel to get started.
      </div>
      <div v-for="novel in novels" :key="novel.id" class="novel-row" @click="goToWorkshop(novel.id)">
        <div class="novel-row__main">
          <el-tag :type="statusType(novel.status)" size="small">{{ novel.status }}</el-tag>
          <span class="novel-row__title">{{ novel.title }}</span>
          <span class="novel-row__desc">{{ novel.description || 'No description' }}</span>
        </div>
        <div class="novel-row__stats">
          <span>{{ novel.chapter_count }} 章</span>
          <span>{{ novel.word_count }} 字</span>
          <span>{{ formatDate(novel.updated_at) }}</span>
          <el-button type="danger" size="small" text @click.stop="handleDelete(novel.id)">Delete</el-button>
        </div>
      </div>
    </div>

    <!-- 创建小说对话框 -->
    <el-dialog v-model="showCreateDialog" title="Create Novel" width="480px">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="Title" required>
          <el-input v-model="createForm.title" placeholder="Enter novel title" maxlength="200" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input
            v-model="createForm.description"
            type="textarea"
            :rows="3"
            placeholder="Brief description of your novel"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">Cancel</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">Create</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useNovelStore } from '@/store/novel'
import GlowCard from '@/components/common/GlowCard.vue'

const props = defineProps<{ id: string; pid: string }>()
const router = useRouter()
const novelStore = useNovelStore()

const novels = ref(novelStore.novels)
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)
const createForm = ref({ title: '', description: '' })
const viewMode = ref(localStorage.getItem('novel-list-view') || 'grid')

import { watch } from 'vue'
watch(viewMode, (v) => localStorage.setItem('novel-list-view', v))

onMounted(async () => {
  loading.value = true
  try {
    await novelStore.fetchNovels(Number(props.pid))
    novels.value = novelStore.novels
  } finally {
    loading.value = false
  }
})

async function handleCreate() {
  if (!createForm.value.title.trim()) {
    ElMessage.warning('Please enter a title')
    return
  }
  creating.value = true
  try {
    const novel = await novelStore.createNovel(
      Number(props.pid),
      createForm.value.title,
      createForm.value.description,
    )
    novels.value = novelStore.novels
    showCreateDialog.value = false
    createForm.value = { title: '', description: '' }
    ElMessage.success('Novel created')
    goToWorkshop(novel.id)
  } finally {
    creating.value = false
  }
}

async function handleDelete(novelId: number) {
  try {
    await ElMessageBox.confirm('Are you sure to delete this novel?', 'Warning', { type: 'warning' })
    await novelStore.deleteNovel(novelId)
    novels.value = novelStore.novels
    ElMessage.success('Novel deleted')
  } catch { /* cancelled */ }
}

function goToWorkshop(novelId: number) {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/novel/${novelId}`)
}

function goToOutline() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/outline`)
}

function goToButler() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/butler`)
}

function statusType(s: string) {
  const map: Record<string, string> = { draft: 'info', writing: 'warning', completed: 'success' }
  return map[s] || 'info'
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString()
}
</script>

<style scoped lang="scss">
.novel-list { width: 100%; max-width: 1200px; margin: 0 auto; }
.page-header {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px;
  padding: 14px 20px; background: var(--color-bg-card); border-radius: 12px;
  border: 1px solid var(--border-glow); box-shadow: var(--shadow-sm);
  transition: box-shadow 0.3s ease;
  &:hover { box-shadow: var(--shadow-md); }

  &__left { display: flex; flex-direction: column; gap: 2px; }

  &__actions {
    display: flex; align-items: center; gap: 10px; flex-shrink: 0;
  }
}
.page-title { font-size: 22px; font-weight: 700; color: var(--color-text-primary); margin-top: 8px; }
.page-desc { font-size: 13px; color: #6b7280; margin-top: 4px; }

.view-toggle {
  display: flex; background: var(--el-fill-color-light, #f3f4f6); border-radius: 6px; padding: 2px; margin-right: 6px;

  &__btn {
    border: none; background: transparent; padding: 5px 10px; border-radius: 4px;
    cursor: pointer; font-size: 16px; color: #9ca3af; transition: all 0.2s;
    &.active { background: #ffffff; color: var(--el-color-primary); box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
    &:hover:not(.active) { color: #6b7280; }
  }
}

.novel-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 20px;
}
.empty-state { grid-column: 1 / -1; text-align: center; padding: 60px; color: var(--color-text-muted); font-size: 14px; }
.novel-card {
  cursor: pointer; padding: 20px;
  &__status { margin-bottom: 8px; }
  &__title { font-size: 18px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 8px; }
  &__desc {
    font-size: 13px; color: var(--color-text-secondary); margin-bottom: 12px;
    display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
  }
  &__meta {
    display: flex; gap: 16px; font-size: 12px; color: var(--color-text-muted); margin-bottom: 12px;
  }
  &__footer {
    display: flex; justify-content: space-between; align-items: center;
    font-size: 12px; color: var(--color-text-muted);
    border-top: 1px solid var(--border-glow); padding-top: 12px;
  }
}

.novel-table {
  display: flex; flex-direction: column; gap: 8px;
}
.novel-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 20px; background: var(--color-bg-card); border-radius: 10px;
  border: 1px solid var(--border-glow); cursor: pointer;
  transition: all 0.2s ease;
  &:hover { box-shadow: 0 2px 8px rgba(99, 120, 255, 0.1); border-color: #c7d0ff; }

  &__main {
    display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0;
  }
  &__title {
    font-size: 15px; font-weight: 600; color: var(--color-text-primary); white-space: nowrap;
  }
  &__desc {
    font-size: 13px; color: #9ca3af; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1;
  }
  &__stats {
    display: flex; align-items: center; gap: 16px; font-size: 12px; color: #9ca3af; flex-shrink: 0;
  }
}
</style>
