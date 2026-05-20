<!-- web/src/views/novel/AudioGenerator.vue -->
<template>
  <div class="audio-generator">
    <div class="audio-generator__config">
      <el-select v-model="voiceId" size="small" placeholder="音色" style="flex: 1">
        <el-option label="Cherry（中文女声·通用）" value="Cherry" />
        <el-option label="Ethan（中文男声·通用）" value="Ethan" />
        <el-option label="Chelsie（中文女声）" value="Chelsie" />
        <el-option label="Serena（中文女声）" value="Serena" />
        <el-option label="Dylan（北京话·男）" value="Dylan" />
        <el-option label="Jada（上海话·女）" value="Jada" />
        <el-option label="Sunny（四川话·女）" value="Sunny" />
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
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { mediaApi } from '@/api/media'
import type { MediaAsset } from '@/api/media'
import { aiApi } from '@/api/ai'
import MediaAssetList from './MediaAssetList.vue'

const props = defineProps<{
  chapterId: number
  chapterContent: string
  portfolioId: number
}>()

const voiceId = ref('Cherry')
const speed = ref(1.0)
const loading = ref(false)
const audioAssets = ref<MediaAsset[]>([])

// 轮询控制
const POLL_INTERVAL_MS = 2000
// 长章节会按段落切分串行合成，每段 5-15s，给 5 分钟兜底
const POLL_TIMEOUT_MS = 300_000
let pollTimer: ReturnType<typeof setInterval> | null = null

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function loadAssets() {
  if (!props.chapterId) return
  try {
    audioAssets.value = await mediaApi.getChapterAssets(props.chapterId, 'audio')
  } catch {
    audioAssets.value = []
  }
}

// 轮询任务状态直到完成/失败/超时
function pollTask(taskId: number) {
  const startAt = Date.now()
  stopPolling()
  pollTimer = setInterval(async () => {
    if (Date.now() - startAt > POLL_TIMEOUT_MS) {
      stopPolling()
      loading.value = false
      ElMessage.warning('音频生成超时，请稍后在列表刷新查看')
      return
    }
    try {
      const task: any = await aiApi.getTask(taskId)
      if (task?.status === 'completed') {
        stopPolling()
        loading.value = false
        ElMessage.success('音频生成完成')
        await loadAssets()
      } else if (task?.status === 'failed' || task?.status === 'cancelled') {
        stopPolling()
        loading.value = false
        ElMessage.error(task?.error_msg || '音频生成失败')
      }
    } catch (e) {
      // 单次轮询失败不终止，等下一次
    }
  }, POLL_INTERVAL_MS)
}

async function handleGenerate() {
  if (!props.chapterContent) {
    ElMessage.warning('章节内容为空')
    return
  }
  loading.value = true
  try {
    const resp = await mediaApi.generateAudio({
      portfolio_id: props.portfolioId,
      chapter_id: props.chapterId,
      text: props.chapterContent,
      voice_id: voiceId.value,
      speed: speed.value,
    })
    ElMessage.info('已提交，正在生成中...')
    pollTask(resp.task_id)
  } catch (e: any) {
    loading.value = false
    ElMessage.error(e.message || '提交失败')
  }
}

watch(() => props.chapterId, () => {
  stopPolling()
  loading.value = false
  loadAssets()
})
onMounted(loadAssets)
onBeforeUnmount(stopPolling)
</script>

<style scoped>
.audio-generator__config {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
</style>
