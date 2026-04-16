<!-- web/src/views/novel/VideoGenerator.vue -->
<template>
  <div class="video-generator">
    <el-input
      v-model="prompt"
      type="textarea"
      :rows="2"
      size="small"
      placeholder="场景描述（留空则使用章节摘要）"
      style="margin-bottom: 8px"
    />
    <el-button
      type="primary"
      size="small"
      :loading="loading"
      @click="handleGenerate"
    >
      生成本章视频
    </el-button>
    <MediaAssetList
      :assets="videoAssets"
      type-label="视频"
      @refresh="loadAssets"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { mediaApi } from '@/api/media'
import type { MediaAsset } from '@/api/media'
import MediaAssetList from './MediaAssetList.vue'

const props = defineProps<{
  chapterId: number
  chapterSummary: string
  portfolioId: number
}>()

const prompt = ref('')
const loading = ref(false)
const videoAssets = ref<MediaAsset[]>([])

async function loadAssets() {
  if (!props.chapterId) return
  try {
    videoAssets.value = await mediaApi.getChapterAssets(props.chapterId, 'video')
  } catch {
    videoAssets.value = []
  }
}

async function handleGenerate() {
  const text = prompt.value || props.chapterSummary
  if (!text) {
    ElMessage.warning('请输入场景描述或确保章节有摘要')
    return
  }
  loading.value = true
  try {
    await mediaApi.generateVideo({
      portfolio_id: props.portfolioId,
      chapter_id: props.chapterId,
      prompt: text,
    })
    ElMessage.success('视频生成任务已提交，预计需要 2-5 分钟')
  } catch (e: any) {
    ElMessage.error(e.message || '提交失败')
  } finally {
    loading.value = false
  }
}

watch(() => props.chapterId, () => {
  prompt.value = ''
  loadAssets()
})
onMounted(loadAssets)
</script>
