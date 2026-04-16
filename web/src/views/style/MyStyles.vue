<template>
  <div class="my-styles">
    <div class="page-header">
      <div>
        <h1 class="page-title">风格库</h1>
        <p class="page-desc">管理你的写作风格模板，可绑定到小说使用</p>
      </div>
      <div class="page-header__actions">
        <NeonButton @click="showAIDialog = true">AI 生成</NeonButton>
        <NeonButton type="primary" @click="openCreate">+ 新建风格</NeonButton>
      </div>
    </div>

    <!-- 风格卡片列表 -->
    <div v-loading="styleStore.loading" class="style-grid">
      <GlowCard
        v-for="style in styleStore.styles"
        :key="style.id"
        hoverable
        class="style-card"
        @click="openEdit(style)"
      >
        <div class="style-card__header">
          <h3 class="style-card__title">{{ style.name }}</h3>
          <el-tag v-if="style.is_ai_generated" size="small" type="warning" effect="dark">AI</el-tag>
        </div>
        <p class="style-card__desc">{{ style.description || '暂无描述' }}</p>
        <div class="style-card__tags">
          <span class="style-tag">{{ voiceLabel(style.narrative_voice) }}</span>
          <span class="style-tag">{{ toneLabel(style.tone) }}</span>
          <span class="style-tag">{{ levelLabel(style.language_level) }}</span>
        </div>
        <div v-if="style.reference_authors" class="style-card__authors">
          参考：{{ style.reference_authors }}
        </div>
        <div class="style-card__footer">
          <span class="style-card__time">{{ formatDate(style.updated_at) }}</span>
          <button class="ghost-btn ghost-btn--danger" @click.stop="handleDelete(style)">删除</button>
        </div>
      </GlowCard>

      <div v-if="!styleStore.loading && styleStore.styles.length === 0" class="empty-state">
        <div class="empty-state__icon">🎨</div>
        <p>还没有风格模板</p>
        <span>点击上方按钮创建，或让 AI 帮你生成</span>
      </div>
    </div>

    <!-- 创建/编辑弹窗 -->
    <el-dialog v-model="showFormDialog" :title="editingStyle ? '编辑风格' : '新建风格'" width="640px" :close-on-click-modal="false">
      <el-form :model="form" label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="如：温暖治愈风" maxlength="100" show-word-limit />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="简要描述这个风格的特点" maxlength="500" show-word-limit />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="叙事视角">
              <el-select v-model="form.narrative_voice" placeholder="选择视角">
                <el-option v-for="o in narrativeVoiceOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="文风调性">
              <el-select v-model="form.tone" placeholder="选择调性">
                <el-option v-for="o in toneOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="语言风格">
              <el-select v-model="form.language_level" placeholder="选择风格">
                <el-option v-for="o in languageLevelOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="参考作家">
          <el-input v-model="form.reference_authors" placeholder="如：余华、东野圭吾" maxlength="500" />
        </el-form-item>
        <el-form-item label="禁用句式">
          <el-input v-model="form.forbidden_patterns" type="textarea" :rows="2" placeholder="不希望出现的表达方式" />
        </el-form-item>
        <el-form-item label="自定义规范">
          <el-input v-model="form.custom_rules" type="textarea" :rows="2" placeholder="额外的写作规范要求" />
        </el-form-item>
        <el-form-item label="自定义 Prompt">
          <el-input v-model="form.custom_prompt" type="textarea" :rows="4" placeholder="完整的风格指令，供 AI 写作时直接使用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showFormDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- AI 生成弹窗 -->
    <el-dialog v-model="showAIDialog" title="AI 生成风格" width="500px" :close-on-click-modal="false">
      <p class="ai-dialog__hint">描述你想要的写作风格，AI 会自动生成完整配置</p>
      <el-input
        v-model="aiDescription"
        type="textarea"
        :rows="4"
        placeholder="如：类似余华《活着》的风格，朴实无华但直击人心，多用短句，避免华丽辞藻"
        maxlength="500"
        show-word-limit
      />
      <template #footer>
        <el-button @click="showAIDialog = false">取消</el-button>
        <el-button type="primary" :loading="styleStore.aiGenerating" @click="handleAIGenerate">生成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStyleStore } from '@/store/user-style'
import type { UserStyle } from '@/api/user-style'
import { narrativeVoiceOptions, toneOptions, languageLevelOptions } from '@/api/user-style'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const styleStore = useUserStyleStore()
const showFormDialog = ref(false)
const showAIDialog = ref(false)
const editingStyle = ref<UserStyle | null>(null)
const saving = ref(false)
const aiDescription = ref('')

const defaultForm = () => ({
  name: '',
  description: '',
  narrative_voice: 'third_limited',
  tone: 'neutral',
  language_level: 'standard',
  reference_authors: '',
  forbidden_patterns: '',
  custom_rules: '',
  custom_prompt: '',
})

const form = ref(defaultForm())

onMounted(() => styleStore.fetchStyles())

function voiceLabel(v: string) {
  return narrativeVoiceOptions.find(o => o.value === v)?.label || v
}
function toneLabel(v: string) {
  return toneOptions.find(o => o.value === v)?.label || v
}
function levelLabel(v: string) {
  return languageLevelOptions.find(o => o.value === v)?.label || v
}

function formatDate(d: string) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

function openCreate() {
  editingStyle.value = null
  form.value = defaultForm()
  showFormDialog.value = true
}

function openEdit(style: UserStyle) {
  editingStyle.value = style
  form.value = {
    name: style.name,
    description: style.description,
    narrative_voice: style.narrative_voice,
    tone: style.tone,
    language_level: style.language_level,
    reference_authors: style.reference_authors,
    forbidden_patterns: style.forbidden_patterns,
    custom_rules: style.custom_rules,
    custom_prompt: style.custom_prompt,
  }
  showFormDialog.value = true
}

async function handleSave() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入风格名称')
    return
  }
  saving.value = true
  try {
    if (editingStyle.value) {
      await styleStore.updateStyle(editingStyle.value.id, form.value)
      ElMessage.success('已更新')
    } else {
      await styleStore.createStyle(form.value)
      ElMessage.success('已创建')
    }
    showFormDialog.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(style: UserStyle) {
  await ElMessageBox.confirm(`确定删除风格「${style.name}」？`, '警告', { type: 'warning' })
  try {
    await styleStore.deleteStyle(style.id)
    ElMessage.success('已删除')
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

async function handleAIGenerate() {
  if (!aiDescription.value.trim()) {
    ElMessage.warning('请输入风格描述')
    return
  }
  try {
    const result = await styleStore.aiGenerate(aiDescription.value)
    if (result) {
      // 将 AI 生成结果填入表单
      form.value = {
        name: '',
        description: aiDescription.value,
        narrative_voice: result.narrative_voice || 'third_limited',
        tone: result.tone || 'neutral',
        language_level: result.language_level || 'standard',
        reference_authors: result.reference_authors || '',
        forbidden_patterns: result.forbidden_patterns || '',
        custom_rules: result.custom_rules || '',
        custom_prompt: result.custom_prompt || '',
      }
      showAIDialog.value = false
      showFormDialog.value = true
      ElMessage.success('AI 已生成风格配置，请确认并保存')
    }
  } catch (e: any) {
    ElMessage.error(e.message || 'AI 生成失败')
  }
}
</script>

<style scoped lang="scss">
.my-styles {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;

  &__actions {
    display: flex;
    gap: 8px;
  }
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.page-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.style-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.style-card {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }

  &__title {
    font-size: 16px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__desc {
    font-size: 13px;
    color: var(--color-text-secondary);
    margin-bottom: 10px;
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  &__tags {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
    margin-bottom: 8px;
  }

  &__authors {
    font-size: 12px;
    color: var(--color-text-muted);
    margin-bottom: 10px;
  }

  &__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 12px;
    border-top: 1px solid var(--border-glow);
  }

  &__time {
    font-size: 12px;
    color: var(--color-text-muted);
  }
}

.style-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  background: rgba(124, 140, 248, 0.12);
  color: var(--color-primary-light);
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 60px 20px;
  color: var(--color-text-muted);

  &__icon {
    font-size: 48px;
    margin-bottom: 12px;
  }

  p {
    font-size: 16px;
    font-weight: 500;
    margin-bottom: 4px;
  }

  span {
    font-size: 13px;
  }
}

.ai-dialog__hint {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}

.ghost-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 13px;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;

  &--danger {
    color: var(--color-text-muted);
    &:hover {
      color: #EF4444;
      background: rgba(239, 68, 68, 0.1);
    }
  }
}
</style>
