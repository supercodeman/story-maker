// web/src/api/novel.ts
import request from './request'

// ========== 类型定义 ==========

export interface Novel {
  id: number
  portfolio_id: number
  title: string
  description: string
  status: string
  source: string
  chapter_count: number
  word_count: number
  token_budget: number
  token_used: number
  butler_topic: string
  butler_storyline: string
  butler_characters: string
  created_at: string
  updated_at: string
}

export interface Chapter {
  id: number
  novel_id: number
  title: string
  sort_order: number
  summary: string
  content: string
  word_count: number
  status: string
  current_version: number
  scene_preset_id: number | null
  created_at: string
  updated_at: string
}

export interface ChapterVersion {
  id: number
  chapter_id: number
  version: number
  content: string
  summary: string
  source: string
  task_id: number | null
  word_count: number
  created_at: string
}

// 热门小说搜索结果
export interface NovelSearchResult {
  title: string
  author: string
  category: string
  cover: string
  intro: string
  setting: string
  characters: string
  plot: string
  source_url: string
}

// ========== API ==========

export const novelApi = {
  // Novel CRUD
  create: (data: { portfolio_id: number; title: string; description?: string }) =>
    request.post('/novels', data),
  list: (portfolioId: number, source?: string) =>
    request.get('/novels', { params: { portfolio_id: portfolioId, source } }),
  get: (id: number) => request.get(`/novels/${id}`),
  update: (id: number, data: Partial<{ title: string; description: string; status: string }>) =>
    request.put(`/novels/${id}`, data),
  delete: (id: number) => request.delete(`/novels/${id}`),

  // Chapter CRUD
  createChapter: (novelId: number, data: { title: string; summary?: string }) =>
    request.post(`/novels/${novelId}/chapters`, data),
  listChapters: (novelId: number) =>
    request.get(`/novels/${novelId}/chapters`),
  updateChapter: (id: number, data: { title?: string; summary?: string; content?: string }) =>
    request.put(`/chapters/${id}`, data),
  deleteChapter: (id: number) => request.delete(`/chapters/${id}`),
  reorderChapters: (novelId: number, chapterIds: number[]) =>
    request.put(`/novels/${novelId}/chapters/reorder`, { chapter_ids: chapterIds }),

  // 扩写章节目录
  expandChapters: (novelId: number, data: { mode: string; insert_after?: number; chapter_num: number; model_name?: string; user_prompt?: string }) =>
    request.post(`/novels/${novelId}/expand-chapters`, data),

  // AI 操作
  chapterAIAction: (chapterId: number, data: { action: string; model_name?: string; summary?: string; content?: string; selected_text?: string; scene_preset_id?: number; polish_mode?: string }) =>
    request.post(`/chapters/${chapterId}/ai`, data),
  acceptAIResult: (chapterId: number, taskId: number) =>
    request.post(`/chapters/${chapterId}/accept`, { task_id: taskId }),
  rejectAIResult: (chapterId: number, taskId: number) =>
    request.post(`/chapters/${chapterId}/reject`, { task_id: taskId }),

  // 大纲生成
  generateOutline: (data: {
    portfolio_id: number
    setting: string
    characters?: string
    background?: string
    plot: string
    chapter_num?: number
    model_name?: string
    user_prompt?: string
    structure_template_id?: number
    hit_analysis_id?: number
    iteration_task_id?: number
    feedback?: string
    butler_session_id?: string
  }) => request.post('/outline/generate', data),
  adoptOutline: (data: {
    portfolio_id: number; task_id: number; title: string; description?: string;
    source?: string; butler_topic?: string; butler_storyline?: string; butler_characters?: string;
    butler_session_id?: string;
    chapters: { title: string; summary: string }[]
  }) =>
    request.post('/outline/adopt', data),

  // 大纲页面章节级 AI 操作
  outlineChapterAI: (data: {
    portfolio_id: number
    action: string
    title: string
    summary: string
    context?: { setting?: string; prev_chapters?: { title: string; summary: string }[]; next_chapters?: { title: string; summary: string }[] }
    model_name?: string
    user_prompt?: string
    butler_session_id?: string
  }) => request.post('/outline/chapter-ai', data),

  // 管家多轮迭代
  startButlerIteration: (data: {
    portfolio_id: number
    action: 'storyline' | 'characters'
    setting: string
    prev_step_result: string
    user_prompt?: string
    model_name: string
    butler_session_id: string
  }) => request.post('/outline/butler-iterate', data),
  getButlerIterationStatus: (iterationId: string) =>
    request.get(`/outline/butler-iterate/${iterationId}`),

  // 热门小说搜索
  searchNovels: (keyword: string) =>
    request.get('/outline/search-novels', { params: { keyword }, timeout: 30000 }),

  // 版本管理
  listVersions: (chapterId: number) =>
    request.get(`/chapters/${chapterId}/versions`),
  revertVersion: (chapterId: number, versionId: number) =>
    request.post(`/chapters/${chapterId}/revert`, { version_id: versionId }),

  // Token 使用情况
  getTokenUsage: (novelId: number) =>
    request.get(`/novels/${novelId}/token-usage`),
  updateTokenBudget: (novelId: number, budget: number) =>
    request.put(`/novels/${novelId}/token-budget`, { token_budget: budget }),

  // 修复历史管家任务关联
  repairButlerLinks: (portfolioId: number) =>
    request.post('/novels/repair-butler', { portfolio_id: portfolioId }),
}
