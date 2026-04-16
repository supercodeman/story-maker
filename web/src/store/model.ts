// web/src/store/model.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getAvailableModels } from '@/api/model'
import type { ModelInfo } from '@/api/model'

export type { ModelInfo } from '@/api/model'

export const useModelStore = defineStore('model', () => {
  const models = ref<ModelInfo[]>([])
  const loading = ref(false)
  const lastFetched = ref(0)

  // 获取模型列表（带 5 分钟缓存）
  async function fetchModels(capability?: string, force = false) {
    const now = Date.now()
    if (!force && models.value.length > 0 && now - lastFetched.value < 5 * 60 * 1000) {
      return
    }

    loading.value = true
    try {
      const res = await getAvailableModels(capability)
      models.value = res as unknown as ModelInfo[]
      lastFetched.value = now
    } catch (e) {
      console.error('[ModelStore] fetch models failed:', e)
    } finally {
      loading.value = false
    }
  }

  // 按能力过滤模型
  function getModels(capability?: string): ModelInfo[] {
    return models.value
  }

  // 获取扁平化的模型列表（包含子模型）
  function getFlatModels(): ModelInfo[] {
    const result: ModelInfo[] = []
    for (const m of models.value) {
      result.push(m)
      if (m.sub_models) {
        result.push(...m.sub_models)
      }
    }
    return result
  }

  return { models, loading, fetchModels, getModels, getFlatModels }
})
