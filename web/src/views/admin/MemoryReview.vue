<!-- web/src/views/admin/MemoryReview.vue -->
<template>
  <div class="memory-review">
    <div class="page-header">
      <div>
        <h1 class="page-title">记忆审核</h1>
        <p class="page-desc">管理所有待审核的写作记忆</p>
      </div>
    </div>

    <div v-loading="loading" class="memory-grid">
      <GlowCard
        v-for="memory in memories"
        :key="memory.id"
        hoverable
        class="memory-card"
      >
        <div class="memory-card__header">
          <el-tag size="small" :type="categoryTagType(memory.category)" effect="dark">
            {{ categoryLabel(memory.category) }}
          </el-tag>
          <span class="memory-card__status status--reviewing">审核中</span>
        </div>
        <h3 class="memory-card__title">{{ memory.title }}</h3>
        <p class="memory-card__desc">{{ memory.description || '暂无描述' }}</p>

        <div class="memory-card__meta">
          <span>📝 {{ memory.sample_len }} 字</span>
          <span v-if="memory.quality > 0">⭐ {{ memory.quality.toFixed(0) }}分</span>
          <span>v{{ memory.version }}</span>
          <span>💰 {{ memory.price }} 积分</span>
        </div>

        <!-- 特征摘要 -->
        <div v-if="memory.features" class="memory-card__preview">
          {{ memory.features.slice(0, 120) }}...
        </div>

        <div class="memory-card__footer">
          <NeonButton type="primary" @click="handleApprove(memory)">通过</NeonButton>
          <NeonButton type="danger" @click="openReject(memory)">拒绝</NeonButton>
        </div>
      </GlowCard>

      <div v-if="!loading && memories.length === 0" class="empty-state">
        <div class="empty-state__icon">🛡️</div>
        <p>暂无待审核记忆</p>
      </div>
    </div>

    <!-- 拒绝原因弹窗 -->
    <el-dialog v-model="showRejectDialog" title="拒绝原因" width="420px" :close-on-click-modal="false">
      <el-form label-position="top">
        <el-form-item label="请填写拒绝原因">
          <el-input v-model="rejectReason" type="textarea" :rows="3" placeholder="请说明拒绝原因..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRejectDialog = false">取消</el-button>
        <NeonButton type="danger" @click="confirmReject">确认拒绝</NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { memoryApi } from '@/api/memory'
import type { WritingMemory } from '@/api/memory'
import { memoryCategoryOptions } from '@/api/memory'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const memories = ref<WritingMemory[]>([])
const loading = ref(false)
const showRejectDialog = ref(false)
const rejectReason = ref('')
const rejectMemoryId = ref(0)

onMounted(() => loadReviewing())

async function loadReviewing() {
  loading.value = true
  try {
    const data: any = await memoryApi.adminListReviewing()
    memories.value = Array.isArray(data) ? data : []
  } finally {
    loading.value = false
  }
}

function categoryLabel(cat: string) {
  return memoryCategoryOptions.find(o => o.value === cat)?.label || cat
}

function categoryTagType(cat: string): string {
  const map: Record<string, string> = { style: '', character: 'success', worldview: 'warning', plot_preference: 'danger' }
  return map[cat] || ''
}

async function handleApprove(memory: WritingMemory) {
  await ElMessageBox.confirm(`确定通过「${memory.title}」的审核？`, '确认通过')
  try {
    await memoryApi.adminApprove(memory.id)
    ElMessage.success('已通过')
    loadReviewing()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

function openReject(memory: WritingMemory) {
  rejectMemoryId.value = memory.id
  rejectReason.value = ''
  showRejectDialog.value = true
}

async function confirmReject() {
  if (!rejectReason.value.trim()) {
    ElMessage.warning('请填写拒绝原因')
    return
  }
  try {
    await memoryApi.adminReject(rejectMemoryId.value, { reason: rejectReason.value })
    ElMessage.success('已拒绝')
    showRejectDialog.value = false
    loadReviewing()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}
</script>

<style scoped lang="scss">
.memory-review {
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
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 4px;
}

.page-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.memory-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
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
    padding: 2px 8px;
    border-radius: 4px;

    &.status--reviewing { color: var(--color-primary-light); background: rgba(124, 140, 248, 0.15); }
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
    justify-content: flex-end;
    padding-top: 10px;
    border-top: 1px solid var(--border-glow);
  }
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 60px 20px;
  color: var(--color-text-muted);

  &__icon {
    font-size: 48px;
    margin-bottom: 12px;
  }

  p {
    font-size: 16px;
    margin-bottom: 6px;
  }
}
</style>
