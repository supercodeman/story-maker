// web/src/store/wallet.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { walletApi } from '@/api/wallet'
import type { UserWallet, WalletTransaction } from '@/api/wallet'

export type { UserWallet, WalletTransaction } from '@/api/wallet'

export const useWalletStore = defineStore('wallet', () => {
  const wallet = ref<UserWallet | null>(null)
  const transactions = ref<WalletTransaction[]>([])
  const txTotal = ref(0)
  const loading = ref(false)

  async function fetchWallet() {
    loading.value = true
    try {
      const data: any = await walletApi.getWallet()
      wallet.value = data || null
    } finally {
      loading.value = false
    }
  }

  async function fetchTransactions(page = 1, pageSize = 20) {
    const data: any = await walletApi.listTransactions({ page, page_size: pageSize })
    transactions.value = Array.isArray(data?.list) ? data.list : []
    txTotal.value = data?.total || 0
  }

  async function recharge(userId: number, amount: number) {
    await walletApi.recharge({ user_id: userId, amount })
    await fetchWallet()
  }

  function reset() {
    wallet.value = null
    transactions.value = []
    txTotal.value = 0
  }

  return {
    wallet,
    transactions,
    txTotal,
    loading,
    fetchWallet,
    fetchTransactions,
    recharge,
    reset,
  }
})
