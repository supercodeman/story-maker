<!-- web/src/views/novel/MediaAssetList.vue -->
<template>
  <div class="media-asset-list">
    <div v-if="assets.length === 0" class="media-asset-list__empty">
      暂无{{ typeLabel }}
    </div>
    <div v-for="asset in assets" :key="asset.id" class="media-asset-item">
      <div class="media-asset-item__info">
        <el-icon v-if="asset.type === 'audio'" class="media-asset-item__icon"><Headset /></el-icon>
        <el-icon v-else class="media-asset-item__icon"><VideoCamera /></el-icon>
        <span class="media-asset-item__name">{{ formatName(asset) }}</span>
        <span class="media-asset-item__duration">{{ formatDuration(asset.duration) }}</span>
      </div>
      <div class="media-asset-item__actions">
        <!-- 音频播放器 -->
        <audio
          v-if="asset.type === 'audio'"
          :src="`/uploads/${asset.file_path}`"
          controls
          preload="none"
          class="media-asset-item__player"
        />
        <!-- 视频播放器 -->
        <video
          v-if="asset.type === 'video'"
          :src="`/uploads/${asset.file_path}`"
          controls
          preload="none"
          class="media-asset-item__video"
        />
        <el-button
          type="danger"
          text
          size="small"
          @click="handleDelete(asset.id)"
        >
          删除
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Headset, VideoCamera } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import type { MediaAsset } from '@/api/media'
import { mediaApi } from '@/api/media'

const props = defineProps<{
  assets: MediaAsset[]
  typeLabel: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
}>()

function formatName(asset: MediaAsset) {
  const date = new Date(asset.created_at)
  return `${date.toLocaleDateString()} ${date.toLocaleTimeString()}`
}

function formatDuration(seconds: number) {
  if (!seconds) return ''
  const min = Math.floor(seconds / 60)
  const sec = Math.floor(seconds % 60)
  return `${min}:${sec.toString().padStart(2, '0')}`
}

async function handleDelete(assetId: number) {
  try {
    await ElMessageBox.confirm('确定删除该资源？', '确认', { type: 'warning' })
    await mediaApi.deleteAsset(assetId)
    emit('refresh')
  } catch {
    // 用户取消
  }
}
</script>

<style scoped>
.media-asset-list__empty {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding: 8px 0;
}
.media-asset-item {
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.media-asset-item:last-child {
  border-bottom: none;
}
.media-asset-item__info {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  margin-bottom: 6px;
}
.media-asset-item__icon {
  color: var(--el-color-primary);
}
.media-asset-item__duration {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.media-asset-item__player {
  width: 100%;
  height: 32px;
}
.media-asset-item__video {
  width: 100%;
  max-height: 180px;
  border-radius: 4px;
}
.media-asset-item__actions {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
}
</style>
