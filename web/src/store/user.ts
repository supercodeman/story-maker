// web/src/store/user.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import request from '@/api/request'

export interface UserProfile {
  id: number
  username: string
  email: string
  role: string
  writer_level: string
  view_mode: string
  total_word_count: number
  total_chapters: number
  completed_novels: number
  created_at: string
}

export interface LevelInfo {
  writer_level: string
  view_mode: string
  level_source: string
  progress: {
    total_word_count: number
    total_chapters: number
    completed_novels: number
    word_target: number
    chapter_target: number
    novel_target: number
  }
}

export interface LoginPayload {
  email: string
  password: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
}

export const useUserStore = defineStore('user', () => {
  const profile = ref<UserProfile | null>(null)
  const isLoggedIn = ref(!!localStorage.getItem('access_token'))
  const levelInfo = ref<LevelInfo | null>(null)

  // 计算属性：是否已解锁大神写手
  const isAdvancedUnlocked = computed(() => profile.value?.writer_level === 'advanced')

  // 计算属性：是否显示高级 UI（大神 + 高级视图模式）
  const showAdvancedUI = computed(() => isAdvancedUnlocked.value && profile.value?.view_mode === 'advanced')

  // 登录
  async function login(payload: LoginPayload) {
    const data: any = await request.post('/auth/login', payload)
    localStorage.setItem('access_token', data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
    isLoggedIn.value = true
    await fetchProfile()
  }

  // 注册
  async function register(payload: RegisterPayload) {
    await request.post('/auth/register', payload)
  }

  // 获取用户信息
  async function fetchProfile() {
    const data: any = await request.get('/user/profile')
    profile.value = data
  }

  // 获取等级信息
  async function fetchLevelInfo() {
    const data: any = await request.get('/user/level')
    levelInfo.value = data
  }

  // 付费解锁大神写手
  async function purchaseUpgrade() {
    await request.post('/user/level/purchase')
    await fetchProfile()
    await fetchLevelInfo()
  }

  // 切换视图模式
  async function updateViewMode(mode: 'simple' | 'advanced') {
    await request.put('/user/view-mode', { view_mode: mode })
    if (profile.value) {
      profile.value.view_mode = mode
    }
  }

  // 登出
  function logout() {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    profile.value = null
    levelInfo.value = null
    isLoggedIn.value = false
    window.location.href = '/login'
  }

  return {
    profile,
    isLoggedIn,
    levelInfo,
    isAdvancedUnlocked,
    showAdvancedUI,
    login,
    register,
    fetchProfile,
    fetchLevelInfo,
    purchaseUpgrade,
    updateViewMode,
    logout,
  }
})
