// web/src/api/suggestion.ts
import request from './request'

export const suggestionApi = {
  // 获取联想文本
  fetchSuggestion: (novelId: number, precedingText: string, signal?: AbortSignal) =>
    request.post(`/novels/${novelId}/suggest`, { novel_id: novelId, preceding_text: precedingText }, { signal }),

  // 上报行为事件
  recordBehavior: (data: { novel_id: number; chapter_id?: number; event_type: string; payload?: any }) =>
    request.post('/behavior/events', data),
}
