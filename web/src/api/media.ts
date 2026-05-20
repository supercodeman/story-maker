// web/src/api/media.ts
import request from './request'

// 音频生成参数
export interface AudioGenParams {
  portfolio_id: number
  chapter_id: number
  text: string
  voice_id?: string
  speed?: number
  emotion?: string
}

// 视频生成参数
export interface VideoGenParams {
  portfolio_id: number
  chapter_id: number
  model_name?: string
  prompt: string
}

// 图片生成参数
export interface ImageGenParams {
  portfolio_id: number
  chapter_id: number
  prompt?: string
  aspect_ratio?: string
  n?: number
}

// 音频导出参数
export interface AudioExportParams {
  voice_id?: string
  speed?: number
}

// 资产类型
export interface MediaAsset {
  id: number
  portfolio_id: number
  type: 'audio' | 'video' | 'image'
  file_path: string
  metadata: string
  duration: number
  chapter_id?: number
  created_by: number
  created_at: string
  role?: string
}

// 任务响应
export interface TaskResponse {
  task_id: number
  status: string
}

export const mediaApi = {
  // 生成音频
  generateAudio: (data: AudioGenParams) =>
    request.post<any, TaskResponse>('/ai/audio/generate', data),

  // 生成视频
  generateVideo: (data: VideoGenParams) =>
    request.post<any, TaskResponse>('/ai/video/generate', data),

  // 生成图片
  generateImage: (data: ImageGenParams) =>
    request.post<any, TaskResponse>('/ai/image/generate', data),

  // 获取章节资产列表
  getChapterAssets: (chapterId: number, type?: string) =>
    request.get<any, MediaAsset[]>(`/chapters/${chapterId}/assets`, { params: { type } }),

  // 删除资产
  deleteAsset: (assetId: number) =>
    request.delete(`/assets/${assetId}`),

  // 获取任务状态
  getTask: (taskId: number) =>
    request.get<any, { task_id: number; status: string; result?: any; error_msg?: string }>(`/ai/tasks/${taskId}`),

  // 设为角色参考图
  setCharacterRef: (assetId: number) =>
    request.put(`/assets/${assetId}/set-character-ref`),

  // 取消角色参考图
  unsetCharacterRef: (assetId: number) =>
    request.put(`/assets/${assetId}/unset-character-ref`),

  // 导出 Word
  exportWord: (novelId: number) =>
    request.post(`/novels/${novelId}/export/word`),

  // 导出音频
  exportAudio: (novelId: number, params?: AudioExportParams) =>
    request.post(`/novels/${novelId}/export/audio`, params),
}
