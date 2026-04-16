// web/src/store/user-style.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { userStyleApi } from '@/api/user-style'
import type { UserStyle } from '@/api/user-style'

export type { UserStyle } from '@/api/user-style'

export const useUserStyleStore = defineStore('userStyle', () => {
  const styles = ref<UserStyle[]>([])
  const loading = ref(false)
  const aiGenerating = ref(false)

  async function fetchStyles() {
    loading.value = true
    try {
      const data: any = await userStyleApi.list()
      styles.value = Array.isArray(data) ? data : []
    } finally {
      loading.value = false
    }
  }

  async function createStyle(data: Partial<UserStyle>) {
    const result: any = await userStyleApi.create(data)
    styles.value.unshift(result)
    return result
  }

  async function updateStyle(id: number, data: Partial<UserStyle>) {
    const result: any = await userStyleApi.update(id, data)
    const idx = styles.value.findIndex(s => s.id === id)
    if (idx !== -1) styles.value[idx] = result
    return result
  }

  async function deleteStyle(id: number) {
    await userStyleApi.delete(id)
    styles.value = styles.value.filter(s => s.id !== id)
  }

  async function aiGenerate(description: string) {
    aiGenerating.value = true
    try {
      const result: any = await userStyleApi.aiGenerate(description)
      return result as UserStyle
    } finally {
      aiGenerating.value = false
    }
  }

  async function bindToNovel(novelId: number, userStyleId: number) {
    await userStyleApi.bindToNovel(novelId, userStyleId)
  }

  async function unbindFromNovel(novelId: number) {
    await userStyleApi.unbindFromNovel(novelId)
  }

  return {
    styles,
    loading,
    aiGenerating,
    fetchStyles,
    createStyle,
    updateStyle,
    deleteStyle,
    aiGenerate,
    bindToNovel,
    unbindFromNovel,
  }
})
