<!-- web/src/views/settings/WriterLevel.vue -->
<template>
  <div class="writer-level-page">
    <div class="page-header">
      <h1 class="page-title">写手等级</h1>
      <p class="page-desc">查看你的创作进度，解锁大神写手高级功能</p>
    </div>

    <!-- 等级状态卡片 -->
    <FadePanel blur class="level-hero">
      <div class="level-hero__badge" :class="{ 'level-hero__badge--advanced': isAdvanced }">
        <span class="level-hero__icon">{{ isAdvanced ? '🏆' : '📝' }}</span>
        <div class="level-hero__info">
          <span class="level-hero__title">{{ isAdvanced ? '大神写手' : '小白写手' }}</span>
          <span v-if="levelInfo?.level_source" class="level-hero__source">{{ sourceLabel }}</span>
        </div>
      </div>
      <div v-if="isAdvanced" class="level-hero__actions">
        <div class="view-mode-switch">
          <span class="view-mode-switch__label">视图模式</span>
          <div class="view-mode-switch__btns">
            <button
              class="mode-btn"
              :class="{ 'mode-btn--active': userStore.profile?.view_mode === 'simple' }"
              @click="handleViewModeChange('simple')"
            >简洁</button>
            <button
              class="mode-btn"
              :class="{ 'mode-btn--active': userStore.profile?.view_mode === 'advanced' }"
              @click="handleViewModeChange('advanced')"
            >高级</button>
          </div>
        </div>
      </div>
    </FadePanel>

    <!-- 成长进度（仅小白写手） -->
    <GlowCard v-if="!isAdvanced && levelInfo" class="progress-card">
      <h3 class="section-title">
        <span class="section-title__icon">📊</span>
        成长进度
      </h3>
      <p class="section-hint">满足以下任一条件即可自动解锁大神写手</p>

      <div class="progress-list">
        <div class="progress-row">
          <div class="progress-row__header">
            <span class="progress-row__label">累计字数</span>
            <span class="progress-row__value">{{ formatNumber(levelInfo.progress.total_word_count) }} / {{ formatNumber(levelInfo.progress.word_target) }}</span>
          </div>
          <div class="progress-bar">
            <div
              class="progress-bar__fill progress-bar__fill--cyan"
              :style="{ width: wordPercent + '%' }"
            ></div>
          </div>
        </div>

        <div class="progress-row">
          <div class="progress-row__header">
            <span class="progress-row__label">累计章节</span>
            <span class="progress-row__value">{{ levelInfo.progress.total_chapters }} / {{ levelInfo.progress.chapter_target }}</span>
          </div>
          <div class="progress-bar">
            <div
              class="progress-bar__fill progress-bar__fill--green"
              :style="{ width: chapterPercent + '%' }"
            ></div>
          </div>
        </div>

        <div class="progress-row">
          <div class="progress-row__header">
            <span class="progress-row__label">完本数</span>
            <span class="progress-row__value">{{ levelInfo.progress.completed_novels }} / {{ levelInfo.progress.novel_target }}</span>
          </div>
          <div class="progress-bar">
            <div
              class="progress-bar__fill progress-bar__fill--amber"
              :style="{ width: novelPercent + '%' }"
            ></div>
          </div>
        </div>
      </div>
    </GlowCard>

    <!-- 付费解锁入口（仅小白写手） -->
    <FadePanel v-if="!isAdvanced" class="purchase-card">
      <div class="purchase-card__content">
        <div class="purchase-card__left">
          <h3 class="section-title">
            <span class="section-title__icon">⚡</span>
            立即解锁
          </h3>
          <p class="purchase-card__desc">不想等？花费 9900 积分立即解锁大神写手全部高级功能。</p>
        </div>
        <NeonButton type="primary" :loading="purchasing" @click="handlePurchase">
          付费解锁（9900 积分）
        </NeonButton>
      </div>
    </FadePanel>

    <!-- 功能对比 -->
    <GlowCard class="compare-card">
      <h3 class="section-title">
        <span class="section-title__icon">🔍</span>
        功能对比
      </h3>
      <div class="compare-table">
        <div class="compare-table__header">
          <span class="compare-table__col compare-table__col--feature">功能</span>
          <span class="compare-table__col compare-table__col--level">小白写手</span>
          <span class="compare-table__col compare-table__col--level">大神写手</span>
        </div>
        <div v-for="item in featureList" :key="item.feature" class="compare-table__row">
          <span class="compare-table__col compare-table__col--feature">{{ item.feature }}</span>
          <span class="compare-table__col compare-table__col--level">
            <span :class="item.beginner ? 'status-dot status-dot--on' : 'status-dot status-dot--off'"></span>
          </span>
          <span class="compare-table__col compare-table__col--level">
            <span :class="item.advanced ? 'status-dot status-dot--on' : 'status-dot status-dot--off'"></span>
          </span>
        </div>
      </div>
    </GlowCard>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/store/user'
import GlowCard from '@/components/common/GlowCard.vue'
import FadePanel from '@/components/common/FadePanel.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const userStore = useUserStore()
const purchasing = ref(false)

const levelInfo = computed(() => userStore.levelInfo)
const isAdvanced = computed(() => userStore.isAdvancedUnlocked)

const sourceLabel = computed(() => {
  switch (levelInfo.value?.level_source) {
    case 'growth': return '成长解锁'
    case 'purchase': return '付费解锁'
    case 'admin': return '管理员开通'
    default: return ''
  }
})

const wordPercent = computed(() =>
  Math.min(100, Math.round(((levelInfo.value?.progress.total_word_count || 0) / (levelInfo.value?.progress.word_target || 1)) * 100))
)
const chapterPercent = computed(() =>
  Math.min(100, Math.round(((levelInfo.value?.progress.total_chapters || 0) / (levelInfo.value?.progress.chapter_target || 1)) * 100))
)
const novelPercent = computed(() =>
  Math.min(100, Math.round(((levelInfo.value?.progress.completed_novels || 0) / (levelInfo.value?.progress.novel_target || 1)) * 100))
)

const featureList = [
  { feature: 'AI 续写 / 扩写 / 润色', beginner: true, advanced: true },
  { feature: '知识库管理', beginner: true, advanced: true },
  { feature: '大纲生成', beginner: true, advanced: true },
  { feature: '版本历史', beginner: true, advanced: true },
  { feature: '写作风格面板', beginner: false, advanced: true },
  { feature: '写作记忆绑定', beginner: false, advanced: true },
  { feature: '剧情结构模板', beginner: false, advanced: true },
  { feature: '爆款拆解', beginner: false, advanced: true },
]

onMounted(() => {
  userStore.fetchLevelInfo()
})

async function handlePurchase() {
  try {
    await ElMessageBox.confirm(
      '确定花费 9900 积分解锁大神写手？解锁后可使用写作风格、写作记忆、剧情结构模板、爆款拆解等高级功能。',
      '确认付费解锁',
      { confirmButtonText: '确认解锁', cancelButtonText: '取消' }
    )
    purchasing.value = true
    await userStore.purchaseUpgrade()
    ElMessage.success('恭喜！已成功解锁大神写手')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '解锁失败，请检查积分余额')
    }
  } finally {
    purchasing.value = false
  }
}

async function handleViewModeChange(mode: string) {
  if (userStore.profile?.view_mode === mode) return
  try {
    await userStore.updateViewMode(mode as 'simple' | 'advanced')
    ElMessage.success(mode === 'advanced' ? '已切换到高级模式' : '已切换到简洁模式')
  } catch (e: any) {
    ElMessage.error(e.message || '切换失败')
  }
}

function formatNumber(n: number) {
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  return n.toString()
}
</script>

<style scoped lang="scss">
.writer-level-page {
  width: 100%;
  max-width: 860px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header {
  margin-bottom: 4px;
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

// 等级英雄卡片
.level-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28px 32px;

  &__badge {
    display: flex;
    align-items: center;
    gap: 16px;

    &--advanced .level-hero__icon {
      background: linear-gradient(135deg, rgba(252, 211, 77, 0.15), rgba(251, 191, 36, 0.08));
      border-color: rgba(252, 211, 77, 0.3);
      box-shadow: 0 0 20px rgba(252, 211, 77, 0.15);
    }
  }

  &__icon {
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 28px;
    border-radius: 14px;
    background: linear-gradient(135deg, rgba(124, 140, 248, 0.12), rgba(124, 140, 248, 0.05));
    border: 1px solid rgba(124, 140, 248, 0.2);
  }

  &__info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__title {
    font-size: 20px;
    font-weight: 700;
    color: var(--color-text-primary);
  }

  &__source {
    font-size: 12px;
    color: var(--color-text-muted);
    padding: 2px 8px;
    background: rgba(124, 140, 248, 0.08);
    border-radius: 4px;
    width: fit-content;
  }
}

// 视图模式切换
.view-mode-switch {
  display: flex;
  align-items: center;
  gap: 12px;

  &__label {
    font-size: 13px;
    color: var(--color-text-secondary);
  }

  &__btns {
    display: flex;
    background: var(--color-bg-hover);
    border-radius: 8px;
    padding: 3px;
    border: 1px solid var(--border-glow);
  }
}

.mode-btn {
  padding: 6px 16px;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;

  &--active {
    background: var(--color-primary);
    color: white;
    box-shadow: 0 0 12px rgba(124, 140, 248, 0.3);
  }

  &:hover:not(&--active) {
    color: var(--color-text-primary);
    background: rgba(124, 140, 248, 0.08);
  }
}

// 进度卡片
.progress-card {
  padding: 24px 28px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;

  &__icon {
    font-size: 16px;
  }
}

.section-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 20px;
}

.progress-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.progress-row {
  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  &__label {
    font-size: 13px;
    color: var(--color-text-secondary);
  }

  &__value {
    font-size: 12px;
    color: var(--color-text-muted);
    font-family: var(--font-mono);
  }
}

.progress-bar {
  height: 6px;
  background: var(--border-glow);
  border-radius: 3px;
  overflow: hidden;

  &__fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.6s ease;

    &--cyan {
      background: linear-gradient(90deg, var(--color-accent-cyan), rgba(103, 232, 249, 0.6));
      box-shadow: 0 0 8px rgba(103, 232, 249, 0.3);
    }

    &--green {
      background: linear-gradient(90deg, var(--color-accent-green), rgba(110, 231, 183, 0.6));
      box-shadow: 0 0 8px rgba(110, 231, 183, 0.3);
    }

    &--amber {
      background: linear-gradient(90deg, var(--color-accent-amber), rgba(252, 211, 77, 0.6));
      box-shadow: 0 0 8px rgba(252, 211, 77, 0.3);
    }
  }
}

// 付费解锁卡片
.purchase-card {
  padding: 24px 28px;
  border-color: rgba(124, 140, 248, 0.2);

  &__content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
  }

  &__left {
    flex: 1;
  }

  &__desc {
    font-size: 13px;
    color: var(--color-text-secondary);
    margin-top: 8px;
  }
}

// 功能对比
.compare-card {
  padding: 24px 28px;
}

.compare-table {
  margin-top: 16px;

  &__header {
    display: flex;
    padding: 10px 0;
    border-bottom: 1px solid var(--border-glow);
    margin-bottom: 4px;
  }

  &__row {
    display: flex;
    padding: 10px 0;
    border-bottom: 1px solid var(--border-light);
    transition: background 0.15s;

    &:hover {
      background: rgba(124, 140, 248, 0.03);
    }

    &:last-child {
      border-bottom: none;
    }
  }

  &__col {
    font-size: 13px;

    &--feature {
      flex: 1;
      color: var(--color-text-secondary);
    }

    &--level {
      width: 100px;
      text-align: center;
      color: var(--color-text-muted);
    }
  }

  &__header &__col {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;

  &--on {
    background: var(--color-accent-green);
    box-shadow: 0 0 6px rgba(110, 231, 183, 0.4);
  }

  &--off {
    background: var(--border-glow);
    border: 1px solid var(--border-light);
  }
}
</style>
