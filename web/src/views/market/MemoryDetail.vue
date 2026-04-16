<!-- web/src/views/market/MemoryDetail.vue -->
<template>
  <div class="memory-detail" v-loading="loading">
    <template v-if="memory">
      <div class="detail-nav">
        <button class="back-btn" @click="$router.back()">
          <span>←</span> 返回市场
        </button>
      </div>

      <div class="detail-layout">
        <!-- 左侧：信息区 -->
        <div class="detail-content">
          <FadePanel class="info-panel">
            <div class="info-panel__tags">
              <el-tag :type="categoryTagType(memory.category)" effect="dark">{{ categoryLabel(memory.category) }}</el-tag>
              <span v-for="tag in parseTags(memory.tags)" :key="tag" class="mini-tag">{{ tag }}</span>
            </div>
            <h1 class="info-panel__title">{{ memory.title }}</h1>
            <p class="info-panel__desc">{{ memory.description }}</p>

            <div class="stat-grid">
              <div class="stat-card">
                <span class="stat-card__icon">📝</span>
                <span class="stat-card__value">{{ memory.sample_len }}</span>
                <span class="stat-card__label">样本字数</span>
              </div>
              <div class="stat-card">
                <span class="stat-card__icon">⭐</span>
                <span class="stat-card__value">{{ Number(memory.quality).toFixed(1) }}</span>
                <span class="stat-card__label">质量评分</span>
              </div>
              <div class="stat-card">
                <span class="stat-card__icon">📦</span>
                <span class="stat-card__value">{{ memory.sales_count }}</span>
                <span class="stat-card__label">购买次数</span>
              </div>
              <div class="stat-card">
                <span class="stat-card__icon">💬</span>
                <span class="stat-card__value">{{ Number(memory.avg_rating).toFixed(1) }}</span>
                <span class="stat-card__label">用户评分 ({{ memory.rating_count }})</span>
              </div>
            </div>
          </FadePanel>

          <!-- 预览文本 -->
          <GlowCard v-if="memory.preview_text" class="preview-card">
            <h3 class="section-title">效果预览</h3>
            <div class="preview-text">{{ memory.preview_text }}</div>
          </GlowCard>

          <!-- 已购买：显示完整信息 -->
          <template v-if="memory.licensed || memory.user_id === currentUserId">
            <GlowCard v-if="memory.features" class="feature-card">
              <h3 class="section-title">提取特征</h3>
              <pre class="code-block">{{ formatJSON(memory.features) }}</pre>
            </GlowCard>
            <GlowCard v-if="memory.prompt_tpl" class="prompt-card">
              <h3 class="section-title">Prompt 模板</h3>
              <div class="prompt-text">{{ memory.prompt_tpl }}</div>
            </GlowCard>
            <GlowCard v-if="memory.anchor_texts" class="anchor-card">
              <h3 class="section-title">锚定句（风格参考）</h3>
              <div class="anchor-list">
                <div v-for="(anchor, i) in parseAnchors(memory.anchor_texts)" :key="i" class="anchor-item">
                  <span class="anchor-item__idx">{{ i + 1 }}</span>
                  <span class="anchor-item__text">{{ anchor }}</span>
                </div>
              </div>
            </GlowCard>
          </template>
        </div>

        <!-- 右侧：购买面板 -->
        <div class="detail-sidebar">
          <FadePanel blur class="purchase-panel">
            <div class="purchase-panel__price">
              {{ memory.price }}
              <span class="purchase-panel__unit">积分</span>
            </div>
            <div class="purchase-panel__balance" v-if="walletStore.wallet">
              余额：{{ walletStore.wallet.balance }} 积分
            </div>
            <template v-if="memory.user_id === currentUserId">
              <div class="purchase-panel__owned">这是你的记忆</div>
            </template>
            <template v-else-if="memory.licensed">
              <div class="purchase-panel__licensed">
                <span class="licensed-icon">✓</span> 已购买
              </div>
            </template>
            <template v-else>
              <NeonButton type="primary" class="purchase-btn" @click="handleBuy">立即购买</NeonButton>
            </template>
          </FadePanel>
        </div>
      </div>

      <!-- 评价区 -->
      <div class="reviews-section">
        <h2 class="section-title section-title--lg">用户评价 ({{ reviewTotal }})</h2>

        <!-- 提交评价 -->
        <FadePanel v-if="memory.licensed && !hasReviewed" class="review-form">
          <div class="review-form__rating">
            <span class="review-form__label">你的评分</span>
            <div class="star-rating">
              <button
                v-for="i in 5"
                :key="i"
                class="star-btn"
                :class="{ 'star-btn--active': i <= reviewRating }"
                @click="reviewRating = i"
              >
                ★
              </button>
            </div>
          </div>
          <el-input v-model="reviewComment" type="textarea" :rows="2" placeholder="写下你的评价..." />
          <NeonButton type="primary" @click="handleSubmitReview">提交评价</NeonButton>
        </FadePanel>

        <div class="review-list">
          <div v-for="review in reviews" :key="review.id" class="review-item">
            <div class="review-item__header">
              <div class="review-item__stars">
                <span v-for="i in 5" :key="i" class="star" :class="{ 'star--filled': i <= review.rating }">★</span>
              </div>
              <span class="review-item__time">{{ review.created_at?.slice(0, 10) }}</span>
            </div>
            <p v-if="review.comment" class="review-item__comment">{{ review.comment }}</p>
          </div>
        </div>

        <div v-if="reviews.length === 0" class="empty-reviews">
          <span>暂无评价</span>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useMarketStore } from '@/store/market'
import { useWalletStore } from '@/store/wallet'
import { memoryCategoryOptions } from '@/api/memory'
import GlowCard from '@/components/common/GlowCard.vue'
import FadePanel from '@/components/common/FadePanel.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const route = useRoute()
const marketStore = useMarketStore()
const walletStore = useWalletStore()
const loading = ref(true)
const reviewRating = ref(5)
const reviewComment = ref('')
const hasReviewed = ref(false)

const mid = computed(() => Number(route.params.mid))
const memory = computed(() => marketStore.currentMemory)
const reviews = computed(() => marketStore.reviews)
const reviewTotal = computed(() => marketStore.reviewTotal)
const currentUserId = computed(() => {
  const token = localStorage.getItem('access_token')
  if (!token) return 0
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.user_id || 0
  } catch { return 0 }
})

onMounted(async () => {
  try {
    await Promise.all([
      marketStore.fetchMemory(mid.value),
      marketStore.fetchReviews(mid.value),
      walletStore.fetchWallet(),
    ])
  } finally {
    loading.value = false
  }
})

function categoryLabel(cat: string) {
  return memoryCategoryOptions.find(o => o.value === cat)?.label || cat
}

function categoryTagType(cat: string): string {
  const map: Record<string, string> = { style: '', character: 'success', worldview: 'warning', plot_preference: 'danger' }
  return map[cat] || ''
}

function parseTags(tags: string): string[] {
  if (!tags) return []
  return tags.split(',').map(t => t.trim()).filter(Boolean)
}

function formatJSON(str: string) {
  try { return JSON.stringify(JSON.parse(str), null, 2) } catch { return str }
}

function parseAnchors(str: string): string[] {
  try {
    const arr = JSON.parse(str)
    return Array.isArray(arr) ? arr : []
  } catch {
    return str ? [str] : []
  }
}

async function handleBuy() {
  const wallet = walletStore.wallet
  if (!wallet || wallet.balance < (memory.value?.price || 0)) {
    ElMessage.warning('积分不足，请先充值')
    return
  }
  await ElMessageBox.confirm(`确认花费 ${memory.value?.price} 积分购买该记忆？`, '确认购买')
  try {
    await marketStore.buyMemory(mid.value)
    ElMessage.success('购买成功')
    await marketStore.fetchMemory(mid.value)
    await walletStore.fetchWallet()
  } catch (e: any) {
    ElMessage.error(e.message || '购买失败')
  }
}

async function handleSubmitReview() {
  if (reviewRating.value < 1) {
    ElMessage.warning('请选择评分')
    return
  }
  try {
    await marketStore.submitReview(mid.value, reviewRating.value, reviewComment.value)
    ElMessage.success('评价已提交')
    hasReviewed.value = true
    reviewComment.value = ''
  } catch (e: any) {
    ElMessage.error(e.message || '评价失败')
  }
}
</script>

<style scoped lang="scss">
.memory-detail {
  width: 100%;
  max-width: 1100px;
  margin: 0 auto;
}

.detail-nav {
  margin-bottom: 20px;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--border-glow);
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }
}

.detail-layout {
  display: grid;
  grid-template-columns: 1fr 300px;
  gap: 24px;
  margin-bottom: 32px;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.info-panel {
  &__tags {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
    margin-bottom: 12px;
  }

  &__title {
    font-size: 24px;
    font-weight: 700;
    color: var(--color-text-primary);
    margin-bottom: 8px;
  }

  &__desc {
    font-size: 14px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    margin-bottom: 20px;
  }
}

.mini-tag {
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--color-bg-hover);
  color: var(--color-text-muted);
  font-size: 11px;
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 14px 8px;
  background: var(--color-bg-deep);
  border-radius: 8px;
  border: 1px solid var(--border-glow);

  &__icon { font-size: 18px; }
  &__value { font-size: 20px; font-weight: 700; color: var(--color-text-primary); }
  &__label { font-size: 11px; color: var(--color-text-muted); text-align: center; }
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 12px;

  &--lg {
    font-size: 18px;
    margin-bottom: 16px;
  }
}

.preview-text {
  background: var(--color-bg-deep);
  padding: 16px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.8;
}

.code-block {
  background: var(--color-bg-deep);
  padding: 14px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-family: var(--font-mono);
  overflow-x: auto;
  max-height: 200px;
}

.prompt-text {
  background: var(--color-bg-deep);
  padding: 16px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.anchor-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.anchor-item {
  display: flex;
  gap: 10px;
  padding: 10px 14px;
  background: var(--color-bg-deep);
  border-radius: 8px;
  border: 1px solid var(--border-glow);

  &__idx {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: rgba(124, 140, 248, 0.15);
    color: var(--color-primary-light);
    font-size: 11px;
    font-weight: 600;
    flex-shrink: 0;
  }

  &__text {
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    font-style: italic;
  }
}

// 购买面板
.detail-sidebar {
  position: sticky;
  top: 24px;
  align-self: start;
}

.purchase-panel {
  text-align: center;

  &__price {
    font-size: 36px;
    font-weight: 700;
    color: var(--color-accent-amber);
    margin-bottom: 4px;
  }

  &__unit {
    font-size: 14px;
    font-weight: 400;
    opacity: 0.7;
  }

  &__balance {
    font-size: 12px;
    color: var(--color-text-muted);
    margin-bottom: 20px;
  }

  &__owned {
    padding: 10px;
    border-radius: 8px;
    background: var(--color-bg-hover);
    color: var(--color-text-muted);
    font-size: 13px;
  }

  &__licensed {
    padding: 10px;
    border-radius: 8px;
    background: rgba(110, 231, 183, 0.1);
    color: var(--color-accent-green);
    font-size: 14px;
    font-weight: 500;
  }
}

.licensed-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--color-accent-green);
  color: #0F1117;
  font-size: 12px;
  margin-right: 4px;
}

.purchase-btn {
  width: 100%;
  padding: 12px 24px;
  font-size: 15px;
}

// 评价区
.reviews-section {
  margin-top: 8px;
}

.review-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;

  &__rating {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  &__label {
    font-size: 13px;
    color: var(--color-text-secondary);
  }
}

.star-rating {
  display: flex;
  gap: 4px;
}

.star-btn {
  border: none;
  background: transparent;
  font-size: 22px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: color 0.15s;
  padding: 0;

  &--active {
    color: var(--color-accent-amber);
  }

  &:hover {
    color: var(--color-accent-amber);
  }
}

.review-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.review-item {
  padding: 14px 0;
  border-bottom: 1px solid var(--border-glow);

  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }

  &__stars {
    display: flex;
    gap: 2px;
  }

  &__time {
    font-size: 12px;
    color: var(--color-text-muted);
  }

  &__comment {
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.5;
  }
}

.star {
  font-size: 14px;
  color: var(--color-text-muted);

  &--filled {
    color: var(--color-accent-amber);
  }
}

.empty-reviews {
  text-align: center;
  padding: 40px;
  color: var(--color-text-muted);
  font-size: 13px;
}
</style>
