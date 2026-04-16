// web/src/store/portfolio.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { portfolioApi } from '@/api/portfolio'
import type { Portfolio } from '@/api/portfolio'

export const usePortfolioStore = defineStore('portfolio', () => {
  const portfolios = ref<Portfolio[]>([])
  const currentPortfolio = ref<Portfolio | null>(null)
  const loading = ref(false)

  async function fetchPortfolios(workspaceId: number) {
    loading.value = true
    try {
      const data: any = await portfolioApi.list(workspaceId)
      portfolios.value = Array.isArray(data) ? data : data.items || []
    } finally {
      loading.value = false
    }
  }

  async function fetchPortfolio(id: number) {
    loading.value = true
    try {
      const data: any = await portfolioApi.get(id)
      currentPortfolio.value = data
    } finally {
      loading.value = false
    }
  }

  return {
    portfolios,
    currentPortfolio,
    loading,
    fetchPortfolios,
    fetchPortfolio,
  }
})
