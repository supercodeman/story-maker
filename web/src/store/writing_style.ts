// web/src/store/writing_style.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { writingStyleApi } from '@/api/writing_style'
import type { WritingStyle, ScenePreset } from '@/api/writing_style'

export type { WritingStyle, ScenePreset } from '@/api/writing_style'

export const useWritingStyleStore = defineStore('writingStyle', () => {
  const style = ref<WritingStyle | null>(null)
  const presets = ref<ScenePreset[]>([])
  const loading = ref(false)

  async function fetchStyle(novelId: number) {
    loading.value = true
    try {
      const data: any = await writingStyleApi.getStyle(novelId)
      style.value = data || null
    } finally {
      loading.value = false
    }
  }

  async function saveStyle(novelId: number, data: Partial<WritingStyle>) {
    const result: any = await writingStyleApi.upsertStyle(novelId, data)
    style.value = result
    return result
  }

  async function deleteStyle(novelId: number) {
    await writingStyleApi.deleteStyle(novelId)
    style.value = null
  }

  async function fetchPresets(novelId: number) {
    try {
      const data: any = await writingStyleApi.listPresets(novelId)
      presets.value = Array.isArray(data) ? data : []
    } catch {
      presets.value = []
    }
  }

  async function createPreset(novelId: number, data: { scene_type: string; name: string; rules: string }) {
    const result: any = await writingStyleApi.createPreset(novelId, data)
    if (result) presets.value.push(result)
    return result
  }

  async function updatePreset(novelId: number, presetId: number, data: Partial<ScenePreset>) {
    const result: any = await writingStyleApi.updatePreset(novelId, presetId, data)
    const idx = presets.value.findIndex(p => p.id === presetId)
    if (idx !== -1) presets.value[idx] = result
    return result
  }

  async function deletePreset(novelId: number, presetId: number) {
    await writingStyleApi.deletePreset(novelId, presetId)
    presets.value = presets.value.filter(p => p.id !== presetId)
  }

  function reset() {
    style.value = null
    presets.value = []
  }

  return {
    style,
    presets,
    loading,
    fetchStyle,
    saveStyle,
    deleteStyle,
    fetchPresets,
    createPreset,
    updatePreset,
    deletePreset,
    reset,
  }
})
