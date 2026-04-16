// web/src/store/character.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { characterApi } from '@/api/character'
import type { Character } from '@/api/character'

export const useCharacterStore = defineStore('character', () => {
  const characters = ref<Character[]>([])
  const loading = ref(false)

  async function fetchCharacters(portfolioId: number) {
    loading.value = true
    try {
      const data: any = await characterApi.list(portfolioId)
      characters.value = Array.isArray(data) ? data : data.items || []
    } finally {
      loading.value = false
    }
  }

  return {
    characters,
    loading,
    fetchCharacters,
  }
})
