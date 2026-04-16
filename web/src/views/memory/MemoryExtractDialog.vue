<!-- web/src/views/memory/MemoryExtractDialog.vue -->
<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    title="提取写作记忆"
    width="600px"
    :close-on-click-modal="false"
  >
    <el-form :model="form" label-position="top" :rules="rules" ref="formRef">
      <el-form-item label="记忆类别" prop="category">
        <div class="category-selector">
          <button
            v-for="opt in memoryCategoryOptions"
            :key="opt.value"
            class="category-chip"
            :class="{ 'category-chip--active': form.category === opt.value }"
            type="button"
            @click="form.category = opt.value"
          >
            <span class="category-chip__icon">{{ categoryIcon(opt.value) }}</span>
            {{ opt.label }}
          </button>
        </div>
      </el-form-item>
      <el-form-item label="记忆标题" prop="title">
        <el-input v-model="form.title" placeholder="给这个记忆起个名字" maxlength="200" show-word-limit />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="2" placeholder="简要描述这个记忆的特点（可选）" />
      </el-form-item>
      <el-form-item label="样本文本" prop="sample_text">
        <el-input
          v-model="form.sample_text"
          type="textarea"
          :rows="10"
          placeholder="粘贴你的小说样本文本（至少 200 字）。&#10;系统将从中提取写作风格、人设、世界观等特征。"
        />
        <div class="sample-meter">
          <div class="sample-meter__bar">
            <div class="sample-meter__fill" :style="{ width: meterWidth }" :class="meterClass" />
          </div>
          <span class="sample-meter__text" :class="meterClass">{{ sampleLen }} 字 · {{ meterLabel }}</span>
        </div>
      </el-form-item>
      <el-form-item label="标签">
        <el-input v-model="form.tags" placeholder="逗号分隔，如：玄幻,热血,升级流" />
      </el-form-item>
      <el-form-item label="赛道分类">
        <el-select v-model="form.genre_ids" multiple placeholder="选择赛道（可多选）" style="width: 100%;">
          <el-option
            v-for="g in flatGenres"
            :key="g.id"
            :label="g.level > 0 ? '　'.repeat(g.level) + g.name : g.name"
            :value="g.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="提取模型">
        <ModelSelector v-model="form.model_name" capability="text_gen" size="small" />
      </el-form-item>
    </el-form>

    <template #footer>
      <button class="neon-cancel-btn" @click="$emit('update:modelValue', false)">取消</button>
      <NeonButton type="primary" :loading="submitting" @click="handleSubmit">
        {{ submitting ? '提取中...' : '开始提取' }}
      </NeonButton>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { useMemoryStore } from '@/store/memory'
import { useGenreStore } from '@/store/genre'
import { memoryCategoryOptions } from '@/api/memory'
import NeonButton from '@/components/common/NeonButton.vue'
import ModelSelector from '@/components/common/ModelSelector.vue'

const props = defineProps<{ modelValue: boolean; initialCategory?: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void; (e: 'created'): void }>()

const memoryStore = useMemoryStore()
const genreStore = useGenreStore()
const formRef = ref<FormInstance>()
const submitting = ref(false)

const defaultCategory = memoryCategoryOptions[0]?.value || 'style'

const form = ref({
  category: props.initialCategory || defaultCategory,
  title: '',
  description: '',
  sample_text: '',
  tags: '',
  model_name: 'qwen',
  genre_ids: [] as number[],
})

// 弹窗打开时，用父级传入的类别初始化
watch(() => props.modelValue, (visible) => {
  if (visible) {
    form.value.category = props.initialCategory || defaultCategory
    genreStore.fetchGenreTree()
  }
})

const flatGenres = computed(() => genreStore.flatGenres())


const sampleLen = computed(() => [...form.value.sample_text].length)
const meterWidth = computed(() => Math.min(100, (sampleLen.value / 2000) * 100) + '%')
const meterClass = computed(() => {
  if (sampleLen.value < 200) return 'meter--warn'
  if (sampleLen.value < 500) return 'meter--ok'
  return 'meter--good'
})
const meterLabel = computed(() => {
  if (sampleLen.value < 200) return '至少需要 200 字'
  if (sampleLen.value < 500) return '长度合格，建议 500+ 字效果更佳'
  return '样本充足'
})

function categoryIcon(val: string) {
  return { style: '✍️', character: '👤', worldview: '🌍', plot_preference: '📖' }[val] || '📋'
}

const rules: FormRules = {
  category: [{ required: true, message: '请选择类别', trigger: 'change' }],
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
  sample_text: [{ required: true, message: '请输入样本文本', trigger: 'blur' }],
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  if (sampleLen.value < 200) {
    ElMessage.warning('样本文本至少需要 200 字')
    return
  }

  submitting.value = true
  try {
    await memoryStore.createMemory({
      category: form.value.category,
      title: form.value.title,
      description: form.value.description,
      sample_text: form.value.sample_text,
      tags: form.value.tags,
      model_name: form.value.model_name,
      genre_ids: form.value.genre_ids.length > 0 ? form.value.genre_ids : undefined,
    })
    ElMessage.success('记忆创建成功，提取工作流已启动')
    emit('update:modelValue', false)
    emit('created')
    form.value = { category: props.initialCategory || defaultCategory, title: '', description: '', sample_text: '', tags: '', model_name: 'qwen' }
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped lang="scss">
.neon-cancel-btn {
  padding: 10px 24px;
  border: 1px solid var(--border-glow, rgba(124, 140, 248, 0.3));
  border-radius: 8px;
  background: transparent;
  color: var(--color-text-secondary, #9CA3C0);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--color-primary, #7C8CF8);
    color: var(--color-text-primary, #E8EAF6);
    box-shadow: 0 0 10px rgba(124, 140, 248, 0.15);
  }
}

.category-selector {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.category-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border-glow);
  border-radius: 8px;
  background: var(--color-bg-card);
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.25s ease;

  &:hover {
    border-color: var(--color-primary);
    color: var(--color-text-primary);
  }

  &--active {
    border-color: var(--color-primary);
    background: rgba(124, 140, 248, 0.15);
    color: var(--color-primary-light);
    box-shadow: 0 0 10px rgba(124, 140, 248, 0.15);
  }

  &__icon {
    font-size: 15px;
  }
}

.sample-meter {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 12px;

  &__bar {
    flex: 1;
    height: 4px;
    background: var(--color-bg-hover);
    border-radius: 2px;
    overflow: hidden;
  }

  &__fill {
    height: 100%;
    border-radius: 2px;
    transition: width 0.3s ease;

    &.meter--warn { background: #EF4444; }
    &.meter--ok { background: var(--color-accent-amber); }
    &.meter--good { background: var(--color-accent-green); }
  }

  &__text {
    font-size: 12px;
    white-space: nowrap;

    &.meter--warn { color: #EF4444; }
    &.meter--ok { color: var(--color-accent-amber); }
    &.meter--good { color: var(--color-accent-green); }
  }
}


</style>
