<!-- web/src/components/ai/TaskResult.vue -->
<template>
  <div class="task-result">
    <template v-if="parsedResult">
      <div v-if="parsedResult.content" class="result-text">
        <pre>{{ parsedResult.content }}</pre>
      </div>
      <div v-if="parsedResult.image_url" class="result-image">
        <img :src="parsedResult.image_url" alt="AI Generated" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AITask } from '@/api/ai'

const props = defineProps<{ task: AITask }>()

const parsedResult = computed(() => {
  if (!props.task.result) return null
  try { return JSON.parse(props.task.result) } catch { return null }
})
</script>

<style scoped lang="scss">
.result-text pre {
  background: var(--color-bg-deep);
  padding: 12px;
  border-radius: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
  max-height: 200px;
  overflow-y: auto;
}
.result-image img {
  max-width: 300px;
  border-radius: 8px;
  border: 1px solid var(--border-glow);
}
</style>
