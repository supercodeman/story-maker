// web/src/store/memory.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { memoryApi } from '@/api/memory'
import type { WritingMemory, NovelMemoryBinding } from '@/api/memory'

export type { WritingMemory, NovelMemoryBinding } from '@/api/memory'

// 记忆提取/审核的 WebSocket 更新数据
export interface MemoryUpdateData {
  id: number
  status: string
  extract_status: string
  extract_workflow_id: number
  extract_error: string
  features: string
  prompt_tpl: string
  anchor_texts: string
  preview_text: string
  quality: number
  review_workflow_id: number
  review_reason: string
  is_public: boolean
}

export const useMemoryStore = defineStore('memory', () => {
  const memories = ref<WritingMemory[]>([])
  const currentMemory = ref<WritingMemory | null>(null)
  const bindings = ref<NovelMemoryBinding[]>([])
  const bindingMemories = ref<WritingMemory[]>([])
  const accessibleMemories = ref<WritingMemory[]>([])
  const loading = ref(false)

  // 处理 WebSocket 推送的记忆更新
  function handleMemoryUpdate(data: MemoryUpdateData) {
    const idx = memories.value.findIndex(m => m.id === data.id)
    if (idx !== -1) {
      const m = memories.value[idx]
      if (data.status) m.status = data.status
      m.extract_status = data.extract_status
      m.extract_workflow_id = data.extract_workflow_id
      m.extract_error = data.extract_error
      if (data.features) m.features = data.features
      if (data.prompt_tpl) m.prompt_tpl = data.prompt_tpl
      if (data.anchor_texts) m.anchor_texts = data.anchor_texts
      if (data.preview_text) m.preview_text = data.preview_text
      if (data.quality) m.quality = data.quality
      if (data.review_workflow_id) m.review_workflow_id = data.review_workflow_id
      if (data.review_reason) m.review_reason = data.review_reason
      if (data.is_public !== undefined) m.is_public = data.is_public
    }
    // 同步更新 currentMemory
    if (currentMemory.value?.id === data.id) {
      if (data.status) currentMemory.value.status = data.status
      currentMemory.value.extract_status = data.extract_status
      currentMemory.value.extract_workflow_id = data.extract_workflow_id
      currentMemory.value.extract_error = data.extract_error
      if (data.features) currentMemory.value.features = data.features
      if (data.prompt_tpl) currentMemory.value.prompt_tpl = data.prompt_tpl
      if (data.anchor_texts) currentMemory.value.anchor_texts = data.anchor_texts
      if (data.preview_text) currentMemory.value.preview_text = data.preview_text
      if (data.quality) currentMemory.value.quality = data.quality
      if (data.review_workflow_id) currentMemory.value.review_workflow_id = data.review_workflow_id
      if (data.review_reason) currentMemory.value.review_reason = data.review_reason
      if (data.is_public !== undefined) currentMemory.value.is_public = data.is_public
    }
  }

  async function fetchMemories(category?: string) {
    loading.value = true
    try {
      const data: any = await memoryApi.list({ category })
      memories.value = Array.isArray(data) ? data : []
    } finally {
      loading.value = false
    }
  }

  async function fetchMemory(mid: number) {
    const data: any = await memoryApi.get(mid)
    currentMemory.value = data || null
    return data
  }

  async function createMemory(data: { category: string; title: string; description?: string; sample_text: string; tags?: string; model_name?: string }) {
    const result: any = await memoryApi.create(data)
    if (result) memories.value.unshift(result)
    return result
  }

  async function updateMemory(mid: number, data: Partial<{ title: string; description: string; tags: string }>) {
    const result: any = await memoryApi.update(mid, data)
    const idx = memories.value.findIndex(m => m.id === mid)
    if (idx !== -1 && result) memories.value[idx] = result
    return result
  }

  async function deleteMemory(mid: number) {
    await memoryApi.delete(mid)
    memories.value = memories.value.filter(m => m.id !== mid)
  }

  async function refineMemory(mid: number, additionalText: string) {
    return await memoryApi.refine(mid, { additional_text: additionalText })
  }

  async function publishMemory(mid: number, price: number) {
    await memoryApi.publish(mid, { price })
    const idx = memories.value.findIndex(m => m.id === mid)
    if (idx !== -1) memories.value[idx].status = 'reviewing'
  }

  async function archiveMemory(mid: number) {
    await memoryApi.archive(mid)
    const idx = memories.value.findIndex(m => m.id === mid)
    if (idx !== -1) memories.value[idx].status = 'archived'
  }

  async function generatePreview(mid: number, modelName?: string) {
    const data: any = await memoryApi.preview(mid, { model_name: modelName })
    return data?.preview_text || ''
  }

  async function fetchBindings(novelId: number) {
    const data: any = await memoryApi.listBindings(novelId)
    bindings.value = Array.isArray(data?.bindings) ? data.bindings : []
    bindingMemories.value = Array.isArray(data?.memories) ? data.memories : []
  }

  async function setBindings(novelId: number, bindingList: Array<{ category: string; memory_id: number }>) {
    await memoryApi.setBindings(novelId, { bindings: bindingList })
    await fetchBindings(novelId)
  }

  async function fetchAccessible(category?: string) {
    const data: any = await memoryApi.listAccessible({ category })
    accessibleMemories.value = Array.isArray(data) ? data : []
  }

  function reset() {
    memories.value = []
    currentMemory.value = null
    bindings.value = []
    bindingMemories.value = []
    accessibleMemories.value = []
  }

  return {
    memories,
    currentMemory,
    bindings,
    bindingMemories,
    accessibleMemories,
    loading,
    handleMemoryUpdate,
    fetchMemories,
    fetchMemory,
    createMemory,
    updateMemory,
    deleteMemory,
    refineMemory,
    publishMemory,
    archiveMemory,
    generatePreview,
    fetchBindings,
    setBindings,
    fetchAccessible,
    reset,
  }
})
