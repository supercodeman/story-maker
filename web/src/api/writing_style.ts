// web/src/api/writing_style.ts
import request from './request'

// ========== 类型定义 ==========

export interface WritingStyle {
  id: number
  novel_id: number
  narrative_voice: string
  tone: string
  language_level: string
  reference_authors: string
  forbidden_patterns: string
  custom_rules: string
  created_at: string
  updated_at: string
}

export interface ScenePreset {
  id: number
  novel_id: number
  scene_type: string
  name: string
  rules: string
  created_at: string
  updated_at: string
}

// 枚举选项
export const narrativeVoiceOptions = [
  { value: 'first', label: '第一人称' },
  { value: 'third_limited', label: '第三人称有限' },
  { value: 'third_omniscient', label: '第三人称全知' },
  { value: 'multi_pov', label: '多视角' },
]

export const toneOptions = [
  { value: 'serious', label: '严肃克制' },
  { value: 'humorous', label: '幽默诙谐' },
  { value: 'lyrical', label: '抒情优美' },
  { value: 'sharp', label: '犀利冷峻' },
  { value: 'warm', label: '温暖治愈' },
  { value: 'neutral', label: '中性' },
]

export const languageLevelOptions = [
  { value: 'literary', label: '文学性' },
  { value: 'standard', label: '标准' },
  { value: 'colloquial', label: '口语化' },
  { value: 'web_novel', label: '网文风' },
]

export const sceneTypeOptions = [
  { value: 'battle', label: '战斗' },
  { value: 'dialogue', label: '对话' },
  { value: 'psychology', label: '心理描写' },
  { value: 'environment', label: '环境描写' },
  { value: 'flashback', label: '回忆' },
  { value: 'daily', label: '日常' },
]

// 预设风格模板
export interface StyleTemplate {
  name: string
  narrative_voice: string
  tone: string
  language_level: string
  reference_authors: string
  forbidden_patterns: string
  custom_rules: string
}

export const styleTemplates: StyleTemplate[] = [
  {
    name: '严肃文学',
    narrative_voice: 'third_limited',
    tone: 'serious',
    language_level: 'literary',
    reference_authors: '余华、莫言、陈忠实',
    forbidden_patterns: '不要使用网络流行语；避免"突然"、"忽然"等廉价转折词；禁止使用"不禁"、"竟然"等AI常见表达',
    custom_rules: '注重细节描写和心理刻画；对话要贴合人物身份和时代背景；叙事节奏沉稳，不追求爽感',
  },
  {
    name: '网文爽文',
    narrative_voice: 'third_omniscient',
    tone: 'sharp',
    language_level: 'web_novel',
    reference_authors: '辰东、天蚕土豆、我吃西红柿',
    forbidden_patterns: '避免过长的心理描写；不要大段环境描写拖慢节奏',
    custom_rules: '节奏明快，每章必须有爽点或钩子；打斗场面要有画面感；适当使用短句增强节奏感；章末留悬念',
  },
  {
    name: '轻小说',
    narrative_voice: 'first',
    tone: 'humorous',
    language_level: 'colloquial',
    reference_authors: '谷川流、�的良米兰、渡航',
    forbidden_patterns: '避免过于沉重的叙事；不要使用文言文或过于书面的表达',
    custom_rules: '对话占比高，推动情节；善用吐槽和内心独白；场景描写简洁明快；人物性格鲜明，有标志性口癖或行为',
  },
  {
    name: '悬疑推理',
    narrative_voice: 'third_limited',
    tone: 'sharp',
    language_level: 'standard',
    reference_authors: '东野圭吾、阿加莎·克里斯蒂、紫金陈',
    forbidden_patterns: '不要过早暴露关键线索；避免无关的感情戏冲淡悬疑氛围',
    custom_rules: '严格遵循公平推理原则，线索必须在揭晓前出现过；控制信息释放节奏，每章埋设至少一个伏笔；环境描写服务于氛围营造',
  },
  {
    name: '武侠仙侠',
    narrative_voice: 'third_omniscient',
    tone: 'lyrical',
    language_level: 'literary',
    reference_authors: '金庸、古龙、萧鼎',
    forbidden_patterns: '避免现代网络用语；不要使用过于直白的打斗描写（如"一拳打过去"）',
    custom_rules: '武功招式要有意境和画面感；善用古诗词和典故增添韵味；江湖恩怨要有因果逻辑；人物对话要有古风韵味但不晦涩',
  },
  {
    name: '科幻',
    narrative_voice: 'third_limited',
    tone: 'serious',
    language_level: 'standard',
    reference_authors: '刘慈欣、阿西莫夫、特德·姜',
    forbidden_patterns: '避免违反已设定的科学体系；不要用魔法思维解释科技问题',
    custom_rules: '科技设定要自洽，遵循内部逻辑；宏大叙事与个体命运交织；技术描写要有质感但不堆砌术语；探讨科技对人性和社会的影响',
  },
]

// ========== API ==========

export const writingStyleApi = {
  // WritingStyle CRUD
  getStyle: (novelId: number) =>
    request.get(`/novels/${novelId}/writing-style`),
  upsertStyle: (novelId: number, data: Partial<WritingStyle>) =>
    request.put(`/novels/${novelId}/writing-style`, data),
  deleteStyle: (novelId: number) =>
    request.delete(`/novels/${novelId}/writing-style`),

  // ScenePreset CRUD
  listPresets: (novelId: number) =>
    request.get(`/novels/${novelId}/scene-presets`),
  createPreset: (novelId: number, data: { scene_type: string; name: string; rules: string }) =>
    request.post(`/novels/${novelId}/scene-presets`, data),
  updatePreset: (novelId: number, presetId: number, data: Partial<ScenePreset>) =>
    request.put(`/novels/${novelId}/scene-presets/${presetId}`, data),
  deletePreset: (novelId: number, presetId: number) =>
    request.delete(`/novels/${novelId}/scene-presets/${presetId}`),
}
