// web/src/store/genre.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { genreApi, type GenreTree } from '@/api/genre'

export const useGenreStore = defineStore('genre', () => {
  const genres = ref<GenreTree[]>([])
  const loading = ref(false)

  // 获取赛道树
  async function fetchGenreTree() {
    loading.value = true
    try {
      const res = await genreApi.listTree()
      genres.value = res.data?.data || []
    } catch (e) {
      console.error('fetch genre tree failed:', e)
    } finally {
      loading.value = false
    }
  }

  // 扁平化赛道列表（用于选择器）
  function flatGenres(): { id: number; name: string; level: number }[] {
    const result: { id: number; name: string; level: number }[] = []
    function walk(nodes: GenreTree[], level: number) {
      for (const node of nodes) {
        result.push({ id: node.id, name: node.name, level })
        if (node.children?.length) {
          walk(node.children, level + 1)
        }
      }
    }
    walk(genres.value, 0)
    return result
  }

  return { genres, loading, fetchGenreTree, flatGenres }
})
