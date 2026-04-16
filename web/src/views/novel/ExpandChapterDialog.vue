<!-- 扩写章节目录弹窗 -->
<template>
  <el-dialog
    :model-value="visible"
    title="扩写章节目录"
    width="640px"
    destroy-on-close
    @update:model-value="$emit('update:visible', $event)"
    @close="handleClose"
  >
    <!-- 阶段一：参数输入 -->
    <template v-if="phase === 'input'">
      <el-form label-width="100px" class="expand-form">
        <el-form-item label="插入位置">
          <el-select v-model="insertAfter" style="width: 100%;">
            <el-option
              v-for="ch in chapters"
              :key="ch.sort_order"
              :label="`第${ch.sort_order}章「${ch.title}」之后`"
              :value="ch.sort_order"
            />
            <el-option v-if="chapters.length === 0" label="（暂无章节，将直接追加）" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item label="扩写数量">
          <el-input-number v-model="chapterNum" :min="1" :max="20" :step="1" />
        </el-form-item>
        <el-form-item label="模型">
          <ModelSelector v-model="modelName" style="width: 100%" />
        </el-form-item>
        <el-form-item label="补充指令">
          <el-input
            v-model="userPrompt"
            type="textarea"
            :rows="3"
            placeholder="可选：补充扩写指令，如风格要求、情节方向等..."
          />
        </el-form-item>
      </el-form>
    </template>

    <!-- 阶段二：结果预览 -->
    <template v-else-if="phase === 'loading'">
      <div class="expand-loading">
        <p>AI 正在生成章节目录，请稍候...</p>
        <el-skeleton :rows="6" animated />
      </div>
    </template>

    <template v-else-if="phase === 'preview'">
      <div class="expand-preview">
        <p class="expand-preview__hint">生成了 {{ previewChapters.length }} 个章节，可编辑标题和概要后确认插入：</p>
        <div
          v-for="(ch, idx) in previewChapters"
          :key="idx"
          class="expand-preview__item"
        >
          <el-input v-model="ch.title" size="small" placeholder="章节标题" style="margin-bottom: 6px;" />
          <el-input v-model="ch.summary" type="textarea" :rows="2" size="small" placeholder="章节概要" />
        </div>
      </div>
    </template>

    <template #footer>
      <template v-if="phase === 'input'">
        <el-button @click="$emit('update:visible', false)">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">提交扩写</el-button>
      </template>
      <template v-else-if="phase === 'loading'">
        <el-button @click="handleCancel">取消</el-button>
      </template>
      <template v-else-if="phase === 'preview'">
        <el-button @click="handleDiscard">丢弃</el-button>
        <el-button type="primary" :loading="inserting" @click="handleConfirmInsert">确认插入</el-button>
      </template>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { useNovelStore } from '@/store/novel'
import { novelApi } from '@/api/novel'
import type { Chapter } from '@/api/novel'
import ModelSelector from '@/components/common/ModelSelector.vue'

const props = defineProps<{
  visible: boolean
  chapters: Chapter[]
  novelId: number
  defaultInsertAfter: number
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'inserted'): void
}>()

const novelStore = useNovelStore()

// 表单状态
const insertAfter = ref(0)
const chapterNum = ref(3)
const modelName = ref('qwen')
const userPrompt = ref('')
const submitting = ref(false)
const inserting = ref(false)

type Phase = 'input' | 'loading' | 'preview'
const phase = ref<Phase>('input')

// 预览章节列表（可编辑）
const previewChapters = reactive<{ title: string; summary: string }[]>([])

// 弹窗打开时初始化
watch(() => props.visible, (val) => {
  if (val) {
    phase.value = 'input'
    insertAfter.value = props.defaultInsertAfter
    chapterNum.value = 3
    userPrompt.value = ''
    submitting.value = false
    inserting.value = false
    previewChapters.splice(0)
  }
})

// 去掉 title 中已有的"第X章"前缀
function stripChapterPrefix(title: string): string {
  return title.replace(/^第[零一二三四五六七八九十百千万\d]+章半?\s*/, '')
}

// 监听 store 中的扩写结果
watch(() => novelStore.expandResult, (result) => {
  if (result && result.length > 0 && phase.value === 'loading') {
    previewChapters.splice(0, previewChapters.length, ...result.map((ch: any) => ({
      title: stripChapterPrefix(ch.title || ''),
      summary: ch.summary || '',
    })))
    phase.value = 'preview'
  }
}, { deep: true })

// 监听 expandPending 变为 false 且无结果（失败情况）
watch(() => novelStore.expandPending, (pending) => {
  if (!pending && phase.value === 'loading' && previewChapters.length === 0) {
    ElMessage.error('扩写生成失败，请重试')
    phase.value = 'input'
    submitting.value = false
  }
})

// 计算插入模式
function computeMode(): 'append' | 'insert' {
  if (props.chapters.length === 0) return 'append'
  const maxSort = props.chapters[props.chapters.length - 1].sort_order
  return insertAfter.value >= maxSort ? 'append' : 'insert'
}

async function handleSubmit() {
  submitting.value = true
  phase.value = 'loading'
  try {
    const mode = computeMode()
    await novelStore.submitExpandChapters(
      props.novelId, mode, insertAfter.value,
      chapterNum.value, modelName.value, userPrompt.value || undefined,
    )
  } catch {
    ElMessage.error('提交扩写请求失败')
    phase.value = 'input'
    submitting.value = false
  }
}

function handleCancel() {
  novelStore.clearExpandResult()
  phase.value = 'input'
  submitting.value = false
}

function handleDiscard() {
  novelStore.clearExpandResult()
  previewChapters.splice(0)
  phase.value = 'input'
  submitting.value = false
}

async function handleConfirmInsert() {
  if (previewChapters.length === 0) return
  inserting.value = true
  try {
    const mode = computeMode()
    const createdIds: number[] = []

    // 逐个创建章节（后端 CreateChapter 支持 summary 字段）
    for (const ch of previewChapters) {
      const newChapter: any = await novelApi.createChapter(props.novelId, { title: ch.title })
      createdIds.push(newChapter.id)
      // 概要通过 updateChapter 写入（createChapter API 类型未暴露 summary）
      if (ch.summary) {
        await novelApi.updateChapter(newChapter.id, { summary: ch.summary })
      }
    }

    // insert 模式需要重排章节顺序
    if (mode === 'insert') {
      // 刷新章节列表获取最新数据
      await novelStore.fetchChapters(props.novelId)
      const allChapters = [...novelStore.chapters]
      // 将新创建的章节从列表中取出
      const existingChapters = allChapters.filter(c => !createdIds.includes(c.id))
      const newChapters = allChapters.filter(c => createdIds.includes(c.id))

      // 在 insertAfter 位置后插入新章节
      const insertIdx = existingChapters.findIndex(c => c.sort_order > insertAfter.value)
      const reorderedIds: number[] = []
      if (insertIdx === -1) {
        // 插入到末尾
        reorderedIds.push(...existingChapters.map(c => c.id), ...newChapters.map(c => c.id))
      } else {
        reorderedIds.push(
          ...existingChapters.slice(0, insertIdx).map(c => c.id),
          ...newChapters.map(c => c.id),
          ...existingChapters.slice(insertIdx).map(c => c.id),
        )
      }
      await novelApi.reorderChapters(props.novelId, reorderedIds)
    }

    // 清理状态，通知父组件刷新
    novelStore.clearExpandResult()
    previewChapters.splice(0)
    emit('inserted')
    emit('update:visible', false)
    ElMessage.success(`成功插入 ${createdIds.length} 个章节`)
  } catch (e) {
    console.error('插入章节失败:', e)
    ElMessage.error('插入章节失败，请重试')
  } finally {
    inserting.value = false
  }
}

function handleClose() {
  if (phase.value === 'loading') {
    novelStore.clearExpandResult()
  }
  previewChapters.splice(0)
  phase.value = 'input'
  submitting.value = false
}
</script>

<style scoped lang="scss">
.expand-form {
  padding: 8px 0;
}

.expand-loading {
  text-align: center;
  padding: 20px 0;
  p { margin-bottom: 16px; color: var(--color-text-secondary); }
}

.expand-preview {
  max-height: 400px;
  overflow-y: auto;

  &__hint {
    margin-bottom: 12px;
    font-size: 13px;
    color: var(--color-text-secondary);
  }

  &__item {
    padding: 10px;
    margin-bottom: 8px;
    border-radius: 6px;
    background-color: var(--color-bg-surface);
    border: 1px solid var(--border-glow);
  }
}
</style>
