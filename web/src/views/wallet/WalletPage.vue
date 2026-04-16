<!-- web/src/views/wallet/WalletPage.vue -->
<template>
  <div class="wallet-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">我的钱包</h1>
        <p class="page-desc">管理你的积分余额和交易记录</p>
      </div>
    </div>

    <!-- 余额卡片 -->
    <div v-loading="walletStore.loading" class="balance-grid">
      <FadePanel blur class="balance-card balance-card--primary">
        <div class="balance-card__icon">💎</div>
        <div class="balance-card__label">可用积分</div>
        <div class="balance-card__value">{{ wallet?.balance || 0 }}</div>
      </FadePanel>
      <GlowCard class="balance-card">
        <div class="balance-card__icon">🔒</div>
        <div class="balance-card__label">冻结积分</div>
        <div class="balance-card__value">{{ wallet?.frozen_balance || 0 }}</div>
      </GlowCard>
      <GlowCard class="balance-card">
        <div class="balance-card__icon">📈</div>
        <div class="balance-card__label">累计收入</div>
        <div class="balance-card__value balance-card__value--green">{{ wallet?.total_income || 0 }}</div>
      </GlowCard>
      <GlowCard class="balance-card">
        <div class="balance-card__icon">📉</div>
        <div class="balance-card__label">累计支出</div>
        <div class="balance-card__value balance-card__value--amber">{{ wallet?.total_spent || 0 }}</div>
      </GlowCard>
    </div>

    <!-- 流水列表 -->
    <FadePanel class="transactions-panel">
      <h3 class="section-title">交易流水</h3>

      <div v-if="walletStore.transactions.length === 0" class="empty-transactions">
        暂无交易记录
      </div>

      <div v-else class="tx-list">
        <div v-for="tx in walletStore.transactions" :key="tx.id" class="tx-item">
          <div class="tx-item__left">
            <span class="tx-item__icon" :class="`tx-icon--${tx.type}`">
              {{ txIcon(tx.type) }}
            </span>
            <div class="tx-item__info">
              <span class="tx-item__desc">{{ tx.description }}</span>
              <span class="tx-item__time">{{ formatTime(tx.created_at) }}</span>
            </div>
          </div>
          <div class="tx-item__right">
            <span class="tx-item__amount" :class="tx.amount > 0 ? 'amount--in' : 'amount--out'">
              {{ tx.amount > 0 ? '+' : '' }}{{ tx.amount }}
            </span>
            <span class="tx-item__balance">余额 {{ tx.balance }}</span>
          </div>
        </div>
      </div>

      <div v-if="walletStore.txTotal > 20" class="pagination">
        <el-pagination
          v-model:current-page="page"
          :page-size="20"
          :total="walletStore.txTotal"
          layout="prev, pager, next"
          background
          @current-change="loadTransactions"
        />
      </div>
    </FadePanel>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useWalletStore } from '@/store/wallet'
import { txTypeLabels } from '@/api/wallet'
import GlowCard from '@/components/common/GlowCard.vue'
import FadePanel from '@/components/common/FadePanel.vue'

const walletStore = useWalletStore()
const page = ref(1)
const wallet = computed(() => walletStore.wallet)

onMounted(async () => {
  await walletStore.fetchWallet()
  await loadTransactions()
})

function loadTransactions() {
  walletStore.fetchTransactions(page.value)
}

function txIcon(type: string): string {
  const map: Record<string, string> = { recharge: '💰', purchase: '🛒', income: '💵', withdraw: '🏦', refund: '↩️' }
  return map[type] || '📋'
}

function formatTime(d: string) {
  if (!d) return ''
  return d.slice(0, 19).replace('T', ' ')
}
</script>

<style scoped lang="scss">
.wallet-page {
  width: 100%;
  max-width: 1000px;
  margin: 0 auto;
}

.page-header {
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

.balance-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.balance-card {
  text-align: center;
  padding: 24px 16px;

  &--primary {
    border-color: var(--color-primary);
    box-shadow: 0 0 20px rgba(124, 140, 248, 0.15);
  }

  &__icon {
    font-size: 24px;
    margin-bottom: 8px;
  }

  &__label {
    font-size: 12px;
    color: var(--color-text-muted);
    margin-bottom: 8px;
  }

  &__value {
    font-size: 28px;
    font-weight: 700;
    color: var(--color-text-primary);

    &--green { color: var(--color-accent-green); }
    &--amber { color: var(--color-accent-amber); }
  }
}

.transactions-panel {
  padding: 24px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 16px;
}

.empty-transactions {
  text-align: center;
  padding: 40px;
  color: var(--color-text-muted);
  font-size: 13px;
}

.tx-list {
  display: flex;
  flex-direction: column;
}

.tx-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--border-glow);

  &:last-child {
    border-bottom: none;
  }

  &__left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  &__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: 8px;
    font-size: 16px;
    background: var(--color-bg-deep);

    &.tx-icon--recharge { background: rgba(110, 231, 183, 0.1); }
    &.tx-icon--purchase { background: rgba(252, 211, 77, 0.1); }
    &.tx-icon--income { background: rgba(110, 231, 183, 0.1); }
    &.tx-icon--refund { background: rgba(239, 68, 68, 0.1); }
  }

  &__info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  &__desc {
    font-size: 13px;
    color: var(--color-text-primary);
  }

  &__time {
    font-size: 11px;
    color: var(--color-text-muted);
  }

  &__right {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 2px;
  }

  &__amount {
    font-size: 15px;
    font-weight: 600;

    &.amount--in { color: var(--color-accent-green); }
    &.amount--out { color: #EF4444; }
  }

  &__balance {
    font-size: 11px;
    color: var(--color-text-muted);
  }
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>
