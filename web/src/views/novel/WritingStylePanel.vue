<!-- web/src/views/novel/WritingStylePanel.vue -->
<template>
  <div class="writing-style-panel">
    <!-- 全局写作风格 -->
    <div class="style-section">
      <div class="style-section__header">
        <span class="style-section__label">全局风格</span>
        <el-button v-if="hasStyle" type="danger" link size="small" @click="handleDeleteStyle">重置</el-button>
      </div>

      <!-- 预设风格快速选择 -->
      <div class="style-chips">
        <button
          v-for="tpl in styleTemplates"
          :key="tpl.name"
          class="style-chip"
          @click="applyTemplate(tpl)"
        >
          {{ tpl.name }}
        </button>
      </div>

      <div class="style-form">
        <div class="style-form__row">
          <label>叙事视角</label>
          <el-select v-model="form.narrative_voice" size="small" placeholder="选择视角">
            <el-option
              v-for="opt in narrativeVoiceOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </div>
        <div class="style-form__row">
          <label>文风调性</label>
          <el-select v-model="form.tone" size="small" placeholder="选择调性">
            <el-option
              v-for="opt in toneOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </div>
        <div class="style-form__row">
          <label>语言风格</label>
          <el-select v-model="form.language_level" size="small" placeholder="选择风格">
            <el-option
              v-for="opt in languageLevelOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </div>
        <div class="style-form__row">
          <label>参考作家</label>
          <el-input v-model="form.reference_authors" size="small" placeholder="如：余华、莫言" />
        </div>
        <div class="style-form__row">
          <label>禁用句式</label>
          <el-input
            v-model="form.forbidden_patterns"
            type="textarea"
            :rows="2"
            size="small"
            placeholder="不希望 AI 使用的表达方式..."
            resize="vertical"
          />
        </div>
        <div class="style-form__row">
          <label>自定义规范</label>
          <el-input
            v-model="form.custom_rules"
            type="textarea"
            :rows="2"
            size="small"
            placeholder="其他写作要求..."
            resize="vertical"
          />
        </div>
        <el-button
          type="primary"
          size="small"
          :loading="saving"
          class="style-form__save"
          @click="handleSaveStyle"
        >
          保存风格
        </el-button>
      </div>
    </div>

    <!-- 场景预设 -->
    <div class="style-section style-section--preset">
      <div class="style-section__header">
        <span class="style-section__label">场景预设</span>
        <el-button type="primary" link size="small" @click="showPresetForm = true">+ 添加</el-button>
      </div>

      <div v-if="writingStyleStore.presets.length === 0" class="style-empty">
        暂无场景预设
      </div>
      <div v-else class="preset-list">
        <div
          v-for="preset in writingStyleStore.presets"
          :key="preset.id"
          class="preset-card"
        >
          <div class="preset-card__top">
            <el-tag size="small" type="info" effect="dark" round>{{ sceneTypeLabel(preset.scene_type) }}</el-tag>
            <span class="preset-card__name">{{ preset.name }}</span>
            <div class="preset-card__actions">
              <el-button type="primary" link size="small" @click="editPreset(preset)">编辑</el-button>
              <el-button type="danger" link size="small" @click="handleDeletePreset(preset.id)">删除</el-button>
            </div>
          </div>
          <div v-if="preset.rules" class="preset-card__rules">{{ preset.rules }}</div>
        </div>
      </div>
    </div>

    <!-- 场景预设编辑弹窗 -->
    <el-dialog
      v-model="showPresetForm"
      :title="editingPresetId ? '编辑场景预设' : '添加场景预设'"
      width="420px"
      :close-on-click-modal="false"
    >
      <el-form label-width="80px" size="small">
        <el-form-item label="场景类型">
          <el-select v-model="presetForm.scene_type" placeholder="选择类型">
            <el-option
              v-for="opt in sceneTypeOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="预设名称">
          <el-input v-model="presetForm.name" placeholder="如：激烈打斗" />
        </el-form-item>
        <el-form-item label="写作规范">
          <el-input
            v-model="presetForm.rules"
            type="textarea"
            :rows="5"
            placeholder="该场景下的具体写作要求..."
            resize="vertical"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPresetForm = false">取消</el-button>
        <el-button type="primary" :loading="presetSaving" @click="handleSavePreset">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useWritingStyleStore } from '@/store/writing_style'
import {
  narrativeVoiceOptions,
  toneOptions,
  languageLevelOptions,
  sceneTypeOptions,
  styleTemplates,
} from '@/api/writing_style'
import type { ScenePreset, StyleTemplate } from '@/api/writing_style'

const props = defineProps<{ novelId: number }>()
const writingStyleStore = useWritingStyleStore()

const saving = ref(false)
const presetSaving = ref(false)
const showPresetForm = ref(false)
const editingPresetId = ref<number | null>(null)

const form = reactive({
  narrative_voice: 'third_limited',
  tone: 'neutral',
  language_level: 'standard',
  reference_authors: '',
  forbidden_patterns: '',
  custom_rules: '',
})

const presetForm = reactive({
  scene_type: 'battle',
  name: '',
  rules: '',
})

const hasStyle = ref(false)

// 重置表单到默认值
function resetForm() {
  form.narrative_voice = 'third_limited'
  form.tone = 'neutral'
  form.language_level = 'standard'
  form.reference_authors = ''
  form.forbidden_patterns = ''
  form.custom_rules = ''
}

// 监听 novelId 变化，加载数据
watch(() => props.novelId, async (id) => {
  if (!id) return
  // 先重置，防止切换小说时残留旧数据
  writingStyleStore.reset()
  hasStyle.value = false
  resetForm()

  try {
    await Promise.all([
      writingStyleStore.fetchStyle(id),
      writingStyleStore.fetchPresets(id),
    ])
  } catch {
    // fetchStyle 和 fetchPresets 内部已有 error handling
  }
  // 回填表单
  if (writingStyleStore.style) {
    hasStyle.value = true
    form.narrative_voice = writingStyleStore.style.narrative_voice || 'third_limited'
    form.tone = writingStyleStore.style.tone || 'neutral'
    form.language_level = writingStyleStore.style.language_level || 'standard'
    form.reference_authors = writingStyleStore.style.reference_authors || ''
    form.forbidden_patterns = writingStyleStore.style.forbidden_patterns || ''
    form.custom_rules = writingStyleStore.style.custom_rules || ''
  }
}, { immediate: true })

function applyTemplate(tpl: StyleTemplate) {
  form.narrative_voice = tpl.narrative_voice
  form.tone = tpl.tone
  form.language_level = tpl.language_level
  form.reference_authors = tpl.reference_authors
  form.forbidden_patterns = tpl.forbidden_patterns
  form.custom_rules = tpl.custom_rules
  ElMessage.info(`已应用「${tpl.name}」风格模板，点击保存生效`)
}

async function handleSaveStyle() {
  saving.value = true
  try {
    await writingStyleStore.saveStyle(props.novelId, { ...form })
    hasStyle.value = true
    ElMessage.success('写作风格已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDeleteStyle() {
  try {
    await ElMessageBox.confirm('确定重置写作风格配置？', '提示', { type: 'warning' })
    await writingStyleStore.deleteStyle(props.novelId)
    hasStyle.value = false
    resetForm()
    ElMessage.success('已重置')
  } catch { /* cancelled */ }
}

function sceneTypeLabel(type: string) {
  return sceneTypeOptions.find(o => o.value === type)?.label || type
}

function editPreset(preset: ScenePreset) {
  editingPresetId.value = preset.id
  presetForm.scene_type = preset.scene_type
  presetForm.name = preset.name
  presetForm.rules = preset.rules
  showPresetForm.value = true
}

async function handleSavePreset() {
  if (!presetForm.name || !presetForm.rules) {
    ElMessage.warning('请填写名称和规范')
    return
  }
  presetSaving.value = true
  try {
    if (editingPresetId.value) {
      await writingStyleStore.updatePreset(props.novelId, editingPresetId.value, { ...presetForm })
      ElMessage.success('场景预设已更新')
    } else {
      await writingStyleStore.createPreset(props.novelId, { ...presetForm })
      ElMessage.success('场景预设已创建')
    }
    showPresetForm.value = false
    editingPresetId.value = null
    presetForm.scene_type = 'battle'
    presetForm.name = ''
    presetForm.rules = ''
  } catch {
    ElMessage.error('保存失败')
  } finally {
    presetSaving.value = false
  }
}

async function handleDeletePreset(presetId: number) {
  try {
    await ElMessageBox.confirm('确定删除该场景预设？', '提示', { type: 'warning' })
    await writingStyleStore.deletePreset(props.novelId, presetId)
    ElMessage.success('已删除')
  } catch { /* cancelled */ }
}

// 关闭弹窗时重置编辑状态
watch(showPresetForm, (val) => {
  if (!val) {
    editingPresetId.value = null
    presetForm.scene_type = 'battle'
    presetForm.name = ''
    presetForm.rules = ''
  }
})
</script>

<style lang="scss" scoped>
.writing-style-panel {
  font-size: 13px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.style-section {
  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
  }

  &__label {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-secondary);
    letter-spacing: 0.3px;
  }

  &--preset {
    padding-top: 14px;
    border-top: 1px solid var(--border-glow, rgba(255,255,255,0.06));
  }
}

// 风格快选 chips
.style-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.style-chip {
  padding: 4px 10px;
  border-radius: 20px;
  border: 1px solid var(--border-glow, rgba(255,255,255,0.08));
  background: var(--color-bg-surface);
  color: var(--color-text-muted);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;

  &:hover {
    color: var(--color-primary-light);
    border-color: rgba(124, 140, 248, 0.4);
    background: rgba(124, 140, 248, 0.06);
  }
}

// 表单
.style-form {
  display: flex;
  flex-direction: column;
  gap: 10px;

  &__row {
    label {
      display: block;
      font-size: 11px;
      color: var(--color-text-muted, #999);
      margin-bottom: 4px;
      letter-spacing: 0.2px;
    }

    .el-select, .el-input {
      width: 100%;
    }
  }

  &__save {
    width: 100%;
    margin-top: 4px;
    border-radius: 8px;
  }
}

.style-empty {
  font-size: 12px;
  color: var(--color-text-muted, #999);
  text-align: center;
  padding: 16px 0;
}

// 场景预设卡片
.preset-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preset-card {
  border: 1px solid var(--border-glow, rgba(255,255,255,0.08));
  border-radius: 8px;
  padding: 10px;
  background: var(--color-bg-surface);
  transition: border-color 0.15s ease;

  &:hover {
    border-color: rgba(124, 140, 248, 0.25);
  }

  &__top {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  &__name {
    flex: 1;
    font-size: 12px;
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__actions {
    flex-shrink: 0;
    display: flex;
    gap: 2px;
    opacity: 0;
    transition: opacity 0.15s ease;
  }

  &:hover &__actions {
    opacity: 1;
  }

  &__rules {
    font-size: 11px;
    color: var(--color-text-muted, #666);
    margin-top: 6px;
    line-height: 1.5;
    white-space: pre-wrap;
    max-height: 48px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
</style>
