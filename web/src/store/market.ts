// web/src/store/market.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { marketApi } from '@/api/market'
import type { WritingMemory } from '@/api/memory'
import type { MemoryReview } from '@/api/market'

export const useMarketStore = defineStore('market', () => {
  const memories = ref<WritingMemory[]>([])
  const total = ref(0)
  const currentMemory = ref<Record<string, any> | null>(null)
  const reviews = ref<MemoryReview[]>([])
  const reviewTotal = ref(0)
  const loading = ref(false)

  async function fetchMemories(params?: { category?: string; keyword?: string; order_by?: string; page?: number; page_size?: number }) {
    loading.value = true
    try {
      const data: any = await marketApi.listMemories(params)
      memories.value = Array.isArray(data?.list) ? data.list : []
      total.value = data?.total || 0
    } finally {
      loading.value = false
    }
  }

  async function fetchMemory(mid: number) {
    const data: any = await marketApi.getMemory(mid)
    currentMemory.value = data || null
    return data
  }

  async function buyMemory(mid: number) {
    return await marketApi.buy(mid)
  }

  async function fetchReviews(mid: number, page = 1, pageSize = 20) {
    const data: any = await marketApi.listReviews(mid, { page, page_size: pageSize })
    reviews.value = Array.isArray(data?.list) ? data.list : []
    reviewTotal.value = data?.total || 0
  }

  async function submitReview(mid: number, rating: number, comment?: string) {
    await marketApi.submitReview(mid, { rating, comment })
    await fetchReviews(mid)
  }

  function reset() {
    memories.value = []
    total.value = 0
    currentMemory.value = null
    reviews.value = []
    reviewTotal.value = 0
  }

  return {
    memories,
    total,
    currentMemory,
    reviews,
    reviewTotal,
    loading,
    fetchMemories,
    fetchMemory,
    buyMemory,
    fetchReviews,
    submitReview,
    reset,
  }
})
