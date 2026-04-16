// web/src/api/user-style.ts
import request from './request'

// ========== 类型定义 ==========

export interface UserStyle {
  id: number
  user_id: number
  name: string
  description: string
  narrative_voice: string
  tone: string
  language_level: string
  reference_authors: string
  forbidden_patterns: string
  custom_rules: string
  custom_prompt: string
  is_ai_generated: boolean
  created_at: string
  updated_at: string
}

// 叙事视角选项
export const narrativeVoiceOptions = [
  { value: 'first', label: '第一人称' },
  { value: 'third_limited', label: '第三人称有限' },
  { value: 'third_omniscient', label: '第三人称全知' },
  { value: 'multi_pov', label: '多视角' },
]

// 文风调性选项
export const toneOptions = [
  { value: 'serious', label: '严肃' },
  { value: 'humorous', label: '幽默' },
  { value: 'lyrical', label: '抒情' },
  { value: 'sharp', label: '犀利' },
  { value: 'warm', label: '温暖' },
  { value: 'neutral', label: '中性' },
]

// 语言风格选项
export const languageLevelOptions = [
  { value: 'literary', label: '文学' },
  { value: 'standard', label: '标准' },
  { value: 'colloquial', label: '口语化' },
  { value: 'web_novel', label: '网文' },
]

// ========== API ==========

export const userStyleApi = {
  list: () =>
    request.get('/user-styles'),
  create: (data: Partial<UserStyle>) =>
    request.post('/user-styles', data),
  update: (id: number, data: Partial<UserStyle>) =>
    request.put(`/user-styles/${id}`, data),
  delete: (id: number) =>
    request.delete(`/user-styles/${id}`),
  aiGenerate: (description: string) =>
    request.post('/user-styles/ai-generate', { description }),
  bindToNovel: (novelId: number, userStyleId: number) =>
    request.put(`/novels/${novelId}/bind-style`, { user_style_id: userStyleId }),
  unbindFromNovel: (novelId: number) =>
    request.delete(`/novels/${novelId}/bind-style`),
}
