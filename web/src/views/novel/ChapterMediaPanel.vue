<!-- web/src/views/novel/ChapterMediaPanel.vue -->
<template>
  <div class="chapter-media-panel">
    <!-- 音频区 -->
    <div class="media-section">
      <div class="media-section__header">
        <el-icon><Headset /></el-icon>
        <span>音频</span>
      </div>
      <AudioGenerator
        :chapter-id="chapterId"
        :chapter-content="chapterContent"
        :portfolio-id="portfolioId"
      />
    </div>

    <!-- 视频区 -->
    <div class="media-section">
      <div class="media-section__header">
        <el-icon><VideoCamera /></el-icon>
        <span>视频</span>
      </div>
      <VideoGenerator
        :chapter-id="chapterId"
        :chapter-summary="chapterSummary"
        :portfolio-id="portfolioId"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { Headset, VideoCamera } from '@element-plus/icons-vue'
import AudioGenerator from './AudioGenerator.vue'
import VideoGenerator from './VideoGenerator.vue'

defineProps<{
  chapterId: number
  chapterContent: string
  chapterSummary: string
  portfolioId: number
}>()
</script>

<style scoped>
.chapter-media-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.media-section__header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}
</style>
