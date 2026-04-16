<!-- web/src/views/novel/NovelOverview.vue -->
<template>
  <div class="novel-overview">
    <!-- 顶部导航 -->
    <div class="overview-header">
      <div class="overview-header__left">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">工作空间</el-breadcrumb-item>
          <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">作品集</el-breadcrumb-item>
          <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}/novel/${nid}` }">小说工坊</el-breadcrumb-item>
          <el-breadcrumb-item>总览</el-breadcrumb-item>
        </el-breadcrumb>
        <h2 class="overview-header__title">{{ novelStore.currentNovel?.title || '' }}</h2>
      </div>

      <div class="overview-header__actions">
        <el-badge :value="overviewStore.pendingChanges.length" :hidden="!overviewStore.hasChanges" :max="99">
          <el-button
            type="warning"
            size="small"
            :disabled="!overviewStore.hasChanges"
            @click="showRevisionDialog = true"
          >
            提交变更
          </el-button>
        </el-badge>
        <el-button
          type="primary"
          size="small"
          :loading="overviewStore.extractPending"
          @click="handleExtract"
        >
          AI 提取总览
        </el-button>
        <el-select v-model="selectedModel" size="small" style="width: 100px;">
          <el-option label="通义" value="qwen" />
          <el-option label="智谱" value="zhipu" />
          <el-option label="Deepseek" value="deepseek" />
        </el-select>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="overviewStore.loading" class="overview-loading">
      <el-skeleton :rows="8" animated />
    </div>

    <!-- 主体内容 -->
    <div v-else class="overview-body">
      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="情节线" name="plotline">
          <PlotTimeline :novel-id="Number(nid)" />
        </el-tab-pane>
        <el-tab-pane label="人物关系" name="character">
          <CharacterGraph :novel-id="Number(nid)" />
        </el-tab-pane>
        <el-tab-pane label="伏笔追踪" name="foreshadow">
          <ForeshadowPanel :novel-id="Number(nid)" />
        </el-tab-pane>
      </el-tabs>

      <!-- 章节索引侧栏 -->
      <aside class="overview-chapters">
        <h4>章节索引</h4>
        <div class="overview-chapters__list">
          <div
            v-for="ch in overviewStore.chapters"
            :key="ch.id"
            class="overview-chapter-item"
          >
            <span class="overview-chapter-item__order">{{ ch.sort_order }}.</span>
            <span class="overview-chapter-item__title">{{ ch.title }}</span>
          </div>
          <div v-if="overviewStore.chapters.length === 0" class="overview-chapters__empty">
            暂无章节
          </div>
        </div>
      </aside>
    </div>

    <!-- 变更确认对话框 -->
    <RevisionDialog
      v-model="showRevisionDialog"
      :novel-id="Number(nid)"
      :portfolio-id="Number(pid)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useOverviewStore } from '@/store/overview'
import { useNovelStore } from '@/store/novel'
import { connectWebSocket, disconnectWebSocket } from '@/utils/websocket'
import PlotTimeline from './PlotTimeline.vue'
import CharacterGraph from './CharacterGraph.vue'
import ForeshadowPanel from './ForeshadowPanel.vue'
import RevisionDialog from './RevisionDialog.vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{ id: string; pid: string; nid: string }>()
const overviewStore = useOverviewStore()
const novelStore = useNovelStore()

const activeTab = ref('plotline')
const selectedModel = ref('qwen')
const showRevisionDialog = ref(false)

onMounted(async () => {
  connectWebSocket()
  try {
    await Promise.all([
      overviewStore.fetchOverview(Number(props.nid)),
      novelStore.currentNovel?.id === Number(props.nid)
        ? Promise.resolve()
        : novelStore.fetchNovel(Number(props.nid)),
    ])
  } catch {
    ElMessage.error('加载总览数据失败')
  }
})

onUnmounted(() => {
  disconnectWebSocket()
  overviewStore.reset()
})

async function handleExtract() {
  try {
    await overviewStore.extractOverview(Number(props.nid), selectedModel.value)
    ElMessage.info('AI 提取任务已提交，完成后将自动更新')
  } catch {
    ElMessage.error('提交提取任务失败')
  }
}
</script>

<style scoped lang="scss">
.novel-overview {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 16px 24px;
}

.overview-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding: 14px 20px;
  background: var(--color-bg-card);
  border-radius: 12px;
  border: 1px solid var(--border-glow);
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.3s ease;

  &:hover {
    box-shadow: var(--shadow-md);
  }

  &__left {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  &__title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
    line-height: 1.3;
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
    padding-top: 2px;
  }
}

.overview-loading {
  padding: 40px;
}

.overview-body {
  flex: 1;
  display: flex;
  gap: 16px;
  min-height: 0;

  :deep(.el-tabs) {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;

    .el-tabs__header {
      margin-bottom: 0;
    }

    .el-tabs__content {
      flex: 1;
      overflow-y: auto;
      min-height: 0;
      padding: 8px 12px;
    }

    .el-tab-pane {
      padding: 0;
    }
  }
}

.overview-chapters {
  width: 220px;
  flex-shrink: 0;
  background: var(--el-bg-color-page);
  border-radius: 8px;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  overflow-y: auto;

  h4 {
    margin: 0 0 12px;
    font-size: 14px;
    color: var(--el-text-color-primary);
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__empty {
    text-align: center;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    padding: 20px 0;
  }
}

.overview-chapter-item {
  display: flex;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  cursor: default;

  &:hover {
    background: var(--el-fill-color-light);
  }

  &__order {
    color: var(--el-text-color-secondary);
    min-width: 24px;
  }

  &__title {
    color: var(--el-text-color-regular);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
