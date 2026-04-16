<!-- web/src/views/novel/AudioGenerator.vue -->
<template>
  <div class="audio-generator">
    <div class="audio-generator__config">
      <el-select v-model="voiceId" size="small" placeholder="音色" style="flex: 1">
        <el-option label="御姐" value="female-yujie" />
        <el-option label="少女" value="female-shaonv" />
        <el-option label="磁性男声" value="male-qn-qingse" />
        <el-option label="沉稳男声" value="male-qn-jingying" />
        <el-option label="童声" value="female-tianmei" />
      </el-select>
      <el-input-number
        v-model="speed"
        size="small"
        :min="0.5"
        :max="2.0"
        :step="0.1"
        :precision="1"
        controls-position="right"
        style="width: 100px"
      />
    </div>
    <el-button
      type="primary"
      size="small"
      :loading="loading"
      :disabled="!chapterContent"
      @click="handleGenerate"
    >
      生成本章音频
    </el-button>
    <MediaAssetList
      :assets="audioAssets"
      type-label="音频"
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
  chapterContent: string
  portfolioId: number
}>()

const voiceId = ref('female-yujie')
const speed = ref(1.0)
const loading = ref(false)
const audioAssets = ref<MediaAsset[]>([])

async function loadAssets() {
  if (!props.chapterId) return
  try {
    audioAssets.value = await mediaApi.getChapterAssets(props.chapterId, 'audio')
  } catch {
    audioAssets.value = []
  }
}

async function handleGenerate() {
  if (!props.chapterContent) {
    ElMessage.warning('章节内容为空')
    return
  }
  loading.value = true
  try {
    await mediaApi.generateAudio({
      portfolio_id: props.portfolioId,
      chapter_id: props.chapterId,
      text: props.chapterContent,
      voice_id: voiceId.value,
      speed: speed.value,
    })
    ElMessage.success('音频生成任务已提交，请等待完成通知')
  } catch (e: any) {
    ElMessage.error(e.message || '提交失败')
  } finally {
    loading.value = false
  }
}

watch(() => props.chapterId, loadAssets)
onMounted(loadAssets)
</script>

<style scoped>
.audio-generator__config {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
</style>
