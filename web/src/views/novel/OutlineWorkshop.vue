<!-- web/src/views/novel/OutlineWorkshop.vue -->
<template>
  <div class="outline-workshop">
    <div class="workshop-header">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">Workspace</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">Portfolio</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}/novels` }">Novels</el-breadcrumb-item>
        <el-breadcrumb-item>Outline Workshop</el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <div class="workshop-body">
      <!-- 左栏：输入区 -->
      <aside class="input-panel">
        <!-- 热门小说搜索区 -->
        <div class="search-section">
          <h3 class="panel-title">🔍 Search Hot Novels</h3>
          <div class="search-bar">
            <el-input
              v-model="searchKeyword"
              placeholder="输入关键词搜索热门小说..."
              clearable
              @keyup.enter="handleSearch"
            />
            <el-button
              type="primary"
              :loading="novelStore.searchLoading"
              :disabled="!searchKeyword.trim()"
              @click="handleSearch"
            >
              Search
            </el-button>
          </div>

          <el-alert
            v-if="novelStore.searchWarning"
            :title="novelStore.searchWarning"
            type="warning"
            show-icon
            closable
            style="margin-top: 8px"
          />

          <div v-if="novelStore.searchLoading" class="search-loading">
            <el-skeleton :rows="3" animated />
          </div>

          <div v-else-if="novelStore.searchResults.length > 0" class="search-results">
            <div v-if="novelStore.searchSource" style="margin-bottom: 8px">
              <el-tag size="small" :type="sourceTagType">{{ sourceLabel }}</el-tag>
            </div>
            <div
              v-for="(novel, idx) in novelStore.searchResults"
              :key="idx"
              class="search-result-card"
            >
              <div class="search-result-card__header">
                <span class="search-result-card__title">📖 {{ novel.title }}</span>
                <span class="search-result-card__author">- {{ novel.author }}</span>
              </div>
              <el-tag v-if="novel.category" size="small" type="info" style="margin-bottom: 4px">{{ novel.category }}</el-tag>
              <p class="search-result-card__intro">{{ novel.intro }}</p>
              <div class="search-result-card__footer">
                <el-button type="primary" size="small" @click="handleImport(novel)">
                  Import ▶
                </el-button>
              </div>
            </div>
            <el-button text size="small" @click="novelStore.clearSearchResults()" style="margin-top: 4px">
              Clear Results
            </el-button>
          </div>
        </div>

        <el-divider />

        <!-- 剧情结构模板选择（大神写手专属） -->
        <div v-if="userStore.showAdvancedUI" class="structure-section">
          <h3 class="panel-title">📐 剧情结构模板</h3>
          <el-select
            v-model="selectedStructureTemplateId"
            placeholder="选择剧情结构模板（可选）"
            clearable
            style="width: 100%; margin-bottom: 8px"
            @change="handleStructureTemplateChange"
          >
            <el-option
              v-for="tpl in structureTemplates"
              :key="tpl.id"
              :label="`${tpl.name} (${tpl.category})`"
              :value="tpl.id"
            />
          </el-select>
          <!-- 结构预览 -->
          <div v-if="structurePreview.length > 0" class="structure-preview">
            <div v-for="phase in structurePreview" :key="phase.phase" class="structure-phase">
              <div class="structure-phase__header">
                <span class="structure-phase__name">{{ phase.name }}</span>
                <el-tag size="small" type="info">{{ Math.round(phase.ratio * 100) }}%</el-tag>
              </div>
              <div class="structure-phase__beats">
                <el-tag v-for="beat in phase.beats" :key="beat" size="small" effect="plain" style="margin: 2px">{{ beat }}</el-tag>
              </div>
            </div>
          </div>
          <el-button text size="small" @click="showAIGenerateTemplate = true" style="margin-top: 4px">
            AI 生成模板
          </el-button>
        </div>

        <!-- 爆款拆解入口（大神写手专属） -->
        <div v-if="userStore.showAdvancedUI" class="hit-analysis-section" style="margin-top: 12px">
          <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px">
            <h3 class="panel-title" style="margin-bottom: 0">🔥 爆款拆解</h3>
            <el-button type="primary" size="small" @click="hitAnalysisRef?.open()">开始拆解</el-button>
          </div>
          <div v-if="selectedHitAnalysisId" class="hit-analysis-badge">
            <el-tag type="success" closable @close="selectedHitAnalysisId = 0">
              已导入拆解报告 #{{ selectedHitAnalysisId }}
            </el-tag>
          </div>
        </div>

        <el-divider />

        <h3 class="panel-title">Creative Input</h3>
        <el-form label-position="top" :model="form">
          <el-form-item label="世界观/设定" required>
            <el-input v-model="form.setting" type="textarea" :rows="4" placeholder="描述故事的世界观、时代背景、核心设定..." />
          </el-form-item>
          <el-form-item label="背景信息">
            <el-input v-model="form.background" type="textarea" :rows="3" placeholder="故事发生前的背景、前因..." />
          </el-form-item>
          <el-form-item label="剧情思路" required>
            <el-input v-model="form.plot" type="textarea" :rows="5" placeholder="整体剧情走向、核心冲突、高潮和结局..." />
          </el-form-item>

          <!-- 主要人物（增强：支持手动输入 + AI 生成） -->
          <el-form-item label="主要人物">
            <el-input v-model="form.characters" type="textarea" :rows="3" placeholder="主要角色的姓名、性格、关系..." />
          </el-form-item>
          <div class="character-gen-section">
            <div class="character-gen-bar">
              <ModelSelector v-model="characterModelName" size="small" style="width: 120px" />
              <el-button
                type="primary"
                size="small"
                :loading="novelStore.characterGenPending"
                :disabled="!form.setting"
                @click="handleGenerateCharacters"
              >
                AI 生成人物
              </el-button>
            </div>
            <!-- AI 人物生成结果预览 -->
            <div v-if="novelStore.characterGenPending && !novelStore.characterGenResult" class="character-gen-loading">
              <el-skeleton :rows="3" animated />
              <p class="loading-hint">AI 正在生成人物设定...</p>
            </div>
            <div v-if="novelStore.characterGenResult" class="character-gen-preview">
              <div class="character-gen-preview__label">AI 生成结果预览</div>
              <pre class="character-gen-preview__text">{{ novelStore.characterGenResult }}</pre>
              <div class="character-gen-preview__actions">
                <el-button type="primary" size="small" @click="handleAcceptCharacters">保留</el-button>
                <el-button size="small" @click="novelStore.discardCharacterGenResult()">丢弃</el-button>
              </div>
            </div>
          </div>
          <el-form-item label="期望章节数">
            <el-input-number v-model="form.chapterNum" :min="3" :max="50" :step="1" />
          </el-form-item>
          <el-form-item label="AI Model">
            <ModelSelector v-model="form.modelName" placeholder="Select model" />
          </el-form-item>
          <!-- Prompt Settings 可折叠区域 -->
          <el-collapse v-model="promptCollapse" style="margin-bottom: 12px">
            <el-collapse-item title="🎯 Prompt Settings" name="prompt">
              <div class="prompt-settings">
                <el-form-item label="模板选择">
                  <el-select v-model="selectedTemplateId" placeholder="选择模板" clearable style="width: 100%" @change="handleTemplateSelect">
                    <el-option
                      v-for="tpl in outlineTemplates"
                      :key="tpl.id"
                      :label="tpl.name"
                      :value="tpl.id"
                    />
                  </el-select>
                </el-form-item>
                <el-form-item label="System Prompt 预览">
                  <el-input
                    :model-value="currentSystemPromptPreview"
                    type="textarea"
                    :rows="4"
                    readonly
                    placeholder="选择模板后显示 system prompt 预览"
                  />
                </el-form-item>
                <el-form-item label="自定义指令">
                  <el-input
                    v-model="userPrompt"
                    type="textarea"
                    :rows="3"
                    placeholder="输入自定义指令，将追加到 system prompt 末尾。例：风格偏向悬疑推理，每章结尾留悬念"
                  />
                </el-form-item>
                <div class="prompt-actions">
                  <el-button size="small" @click="handleSaveTemplate" :disabled="!userPrompt.trim()">Save as Template</el-button>
                  <el-button size="small" @click="handleLoadTemplates">Refresh Templates</el-button>
                </div>
              </div>
            </el-collapse-item>
          </el-collapse>

          <el-button
            type="primary"
            text bg
            :loading="novelStore.outlinePending"
            :disabled="!form.setting || !form.plot"
            style="width: 100%"
            @click="handleGenerate"
          >
            Generate Outline
          </el-button>
        </el-form>
      </aside>

      <!-- 右栏：预览区 -->
      <main class="preview-panel">
        <h3 class="panel-title">Outline Preview</h3>

        <!-- 生成中 -->
        <div v-if="novelStore.outlinePending" class="loading-state">
          <el-skeleton :rows="8" animated />
          <p class="loading-hint">AI is generating your outline...</p>
        </div>

        <!-- 空状态 -->
        <div v-else-if="editableChapters.length === 0" class="empty-state">
          Fill in the creative input on the left and click "Generate Outline" to get started.
        </div>

        <!-- 大纲预览编辑 -->
        <div v-else class="outline-editor">
          <div class="novel-meta">
            <el-form label-position="top">
              <el-form-item label="Novel Title" required>
                <el-input v-model="novelTitle" placeholder="Enter novel title" maxlength="200" />
              </el-form-item>
              <el-form-item label="Description">
                <el-input v-model="novelDescription" type="textarea" :rows="2" placeholder="Brief description" />
              </el-form-item>
            </el-form>
          </div>

          <div class="chapter-list">
            <div v-for="(ch, index) in editableChapters" :key="index" class="chapter-card" :class="{ 'chapter-card--ai-active': novelStore.outlineAIPending === index }">
              <div class="chapter-card__header">
                <span class="chapter-card__index">{{ index + 1 }}</span>
                <el-input v-model="ch.title" placeholder="Chapter title" class="chapter-card__title-input" />
                <el-dropdown trigger="click" @command="(cmd: string) => handleChapterAI(index, cmd)" :disabled="novelStore.outlineAIPending !== null">
                  <el-button type="primary" text bg size="small" :loading="novelStore.outlineAIPending === index && !novelStore.outlineAIResult">
                    AI
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="title_polish">润色标题</el-dropdown-item>
                      <el-dropdown-item command="summary_polish">润色概要</el-dropdown-item>
                      <el-dropdown-item command="summary_expand">扩写概要</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
                <el-button type="danger" text size="small" @click="removeChapter(index)">Delete</el-button>
              </div>
              <el-input
                v-model="ch.summary"
                type="textarea"
                :rows="4"
                placeholder="Chapter summary"
                class="chapter-card__summary"
              />
              <!-- AI 结果预览区 -->
              <div v-if="novelStore.outlineAIPending === index && novelStore.outlineAIResult" class="ai-preview">
                <div class="ai-preview__label">AI Result Preview</div>
                <div v-if="novelStore.outlineAIResult.title" class="ai-preview__field">
                  <span class="ai-preview__tag">Title</span>
                  <span class="ai-preview__text">{{ novelStore.outlineAIResult.title }}</span>
                </div>
                <div v-if="novelStore.outlineAIResult.summary" class="ai-preview__field">
                  <span class="ai-preview__tag">Summary</span>
                  <p class="ai-preview__text">{{ novelStore.outlineAIResult.summary }}</p>
                </div>
                <div class="ai-preview__actions">
                  <el-button type="primary" size="small" @click="handleAcceptAI(index)">Accept</el-button>
                  <el-button size="small" @click="novelStore.discardOutlineAIResult()">Discard</el-button>
                </div>
              </div>
            </div>
          </div>

          <div class="outline-actions">
            <el-button @click="addChapter">+ Add Chapter</el-button>
            <el-button
              type="primary"
              :loading="adopting"
              :disabled="!novelTitle || editableChapters.length === 0"
              @click="handleAdopt"
            >
              Adopt &amp; Create Novel
            </el-button>
          </div>

          <!-- 多轮迭代：优化反馈 -->
          <div v-if="editableChapters.length > 0 && novelStore.outlineTaskId" class="iteration-section">
            <h4 style="margin: 0 0 8px; color: var(--el-text-color-secondary)">优化反馈</h4>
            <el-input
              v-model="iterationFeedback"
              type="textarea"
              :rows="3"
              placeholder="对当前大纲的修改意见，例如：第三章节奏太快，需要增加过渡..."
            />
            <el-button
              type="warning"
              size="small"
              style="margin-top: 8px"
              :loading="novelStore.outlinePending"
              :disabled="!iterationFeedback.trim()"
              @click="handleIterate"
            >
              根据反馈重新生成
            </el-button>
          </div>
        </div>
      </main>
    </div>

    <!-- AI 生成模板弹窗 -->
    <el-dialog v-model="showAIGenerateTemplate" title="AI 生成剧情结构模板" width="500px">
      <el-input
        v-model="aiTemplateDescription"
        type="textarea"
        :rows="4"
        placeholder="描述你想要的结构风格，例如：类似盗墓笔记的探险解谜结构"
      />
      <template #footer>
        <el-button @click="showAIGenerateTemplate = false">Cancel</el-button>
        <el-button type="primary" :loading="aiTemplateGenerating" :disabled="!aiTemplateDescription.trim()" @click="handleAIGenerateTemplate">
          生成
        </el-button>
      </template>
    </el-dialog>

    <!-- 爆款拆解对话框 -->
    <HitAnalysisDialog
      ref="hitAnalysisRef"
      :portfolio-id="Number(pid)"
      @import="handleImportHitAnalysis"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useNovelStore } from '@/store/novel'
import { useUserStore } from '@/store/user'
import type { NovelSearchResult } from '@/store/novel'
import { promptTemplateApi } from '@/api/prompt_template'
import type { PromptTemplate } from '@/api/prompt_template'
import { plotStructureApi, type PlotStructureTemplate, type PlotPhase } from '@/api/plot_structure'
import HitAnalysisDialog from './HitAnalysisDialog.vue'
import { connectWebSocket, disconnectWebSocket } from '@/utils/websocket'
import ModelSelector from '@/components/common/ModelSelector.vue'

const props = defineProps<{ id: string; pid: string }>()
const router = useRouter()
const novelStore = useNovelStore()
const userStore = useUserStore()

const form = ref({
  setting: '',
  characters: '',
  background: '',
  plot: '',
  chapterNum: 10,
  modelName: 'qwen',
})

// 热门小说搜索
const searchKeyword = ref('')

// 人物 AI 生成模型选择
const characterModelName = ref('qwen')

// ========== 剧情结构模板 ==========
const structureTemplates = ref<PlotStructureTemplate[]>([])
const selectedStructureTemplateId = ref<number | undefined>(undefined)
const structurePreview = ref<PlotPhase[]>([])
const showAIGenerateTemplate = ref(false)
const aiTemplateDescription = ref('')
const aiTemplateGenerating = ref(false)

async function loadStructureTemplates() {
  try {
    const res = await plotStructureApi.list()
    structureTemplates.value = res.data?.data || []
  } catch {
    structureTemplates.value = []
  }
}

function handleStructureTemplateChange(id: number | undefined) {
  if (!id) {
    structurePreview.value = []
    return
  }
  const tpl = structureTemplates.value.find(t => t.id === id)
  if (tpl) {
    try {
      structurePreview.value = JSON.parse(tpl.structure)
    } catch {
      structurePreview.value = []
    }
  }
}

async function handleAIGenerateTemplate() {
  aiTemplateGenerating.value = true
  try {
    const res = await plotStructureApi.aiGenerate({ description: aiTemplateDescription.value })
    const taskId = res.data?.data?.task_id
    if (taskId) {
      ElMessage.success('模板生成任务已提交，请稍后在模板列表中查看')
      showAIGenerateTemplate.value = false
      aiTemplateDescription.value = ''
    }
  } catch {
    ElMessage.error('模板生成失败')
  } finally {
    aiTemplateGenerating.value = false
  }
}

// ========== 爆款拆解 ==========
const hitAnalysisRef = ref<InstanceType<typeof HitAnalysisDialog> | null>(null)
const selectedHitAnalysisId = ref<number>(0)

function handleImportHitAnalysis(analysisId: number) {
  selectedHitAnalysisId.value = analysisId
  ElMessage.success('拆解报告已导入，将在生成大纲时作为参考')
}

// ========== 多轮迭代 ==========
const iterationFeedback = ref('')

// ========== Prompt 模板相关 ==========
const promptCollapse = ref<string[]>([])
const userPrompt = ref('')
const outlineTemplates = ref<PromptTemplate[]>([])
const selectedTemplateId = ref<number | null>(null)
const currentSystemPromptPreview = ref('')

// 加载大纲相关模板（novelId=0 表示系统默认）
async function handleLoadTemplates() {
  try {
    const data: any = await promptTemplateApi.list(0)
    const all: PromptTemplate[] = Array.isArray(data) ? data : []
    // 只保留 outline_ 开头的模板
    outlineTemplates.value = all.filter(t => t.action.startsWith('outline_'))
  } catch {
    outlineTemplates.value = []
  }
}

// 选择模板时预览 system prompt
function handleTemplateSelect(id: number | null) {
  if (!id) {
    currentSystemPromptPreview.value = ''
    return
  }
  const tpl = outlineTemplates.value.find(t => t.id === id)
  if (tpl) {
    currentSystemPromptPreview.value = tpl.content
    // 如果是 user 类型模板，加载到自定义指令
    if (tpl.prompt_type === 'user' && tpl.content) {
      userPrompt.value = tpl.content
    }
  }
}

// 保存当前自定义指令为模板
async function handleSaveTemplate() {
  const instruction = userPrompt.value.trim()
  if (!instruction) return
  try {
    const name = await ElMessageBox.prompt('请输入模板名称', 'Save Template', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputPlaceholder: '例：悬疑风格指令',
    })
    if (name.value) {
      await promptTemplateApi.upsert(0, {
        action: 'outline_generate',
        prompt_type: 'user',
        name: name.value,
        content: instruction,
      })
      ElMessage.success('模板已保存')
      await handleLoadTemplates()
    }
  } catch {
    // 用户取消
  }
}

// 数据来源显示
const sourceMap: Record<string, { label: string; type: string }> = {
  ai_web: { label: 'AI 联网搜索', type: '' },
  fanqie: { label: '番茄小说', type: 'success' },
  ai: { label: 'AI 推荐', type: 'warning' },
}
const sourceLabel = computed(() => sourceMap[novelStore.searchSource]?.label || novelStore.searchSource)
const sourceTagType = computed(() => (sourceMap[novelStore.searchSource]?.type || 'info') as any)

async function handleSearch() {
  const kw = searchKeyword.value.trim()
  if (!kw) return
  await novelStore.searchNovels(kw)
}

async function handleImport(novel: NovelSearchResult) {
  // 构建导入字段映射：优先用结构化字段，为空时层层兜底
  const introOrTitle = novel.intro || novel.title || ''
  const setting = novel.setting || introOrTitle
  const characters = novel.characters || ''
  const plot = novel.plot || ''
  const background = introOrTitle

  const fields = [
    { key: 'setting', label: '世界观/设定', value: setting },
    { key: 'characters', label: '主要人物', value: characters },
    { key: 'plot', label: '剧情思路', value: plot },
    { key: 'background', label: '背景信息', value: background },
  ] as const

  let imported = false
  for (const field of fields) {
    if (!field.value) continue
    imported = true
    const formKey = field.key as keyof typeof form.value
    if (form.value[formKey] && typeof form.value[formKey] === 'string' && (form.value[formKey] as string).trim()) {
      try {
        const action = await ElMessageBox.confirm(
          `"${field.label}" 已有内容，要覆盖还是追加？`,
          'Import',
          { confirmButtonText: '覆盖', cancelButtonText: '追加', distinguishCancelAndClose: true },
        )
        // 点击确认 = 覆盖
        ;(form.value[formKey] as string) = field.value
      } catch (action) {
        if (action === 'cancel') {
          // 追加
          ;(form.value[formKey] as string) += '\n' + field.value
        }
        // close = 跳过
      }
    } else {
      ;(form.value[formKey] as string) = field.value
    }
  }

  if (imported) {
    ElMessage.success('已导入小说设定')
  } else {
    ElMessage.warning('该搜索结果无可导入的内容')
  }
}

// ========== 人物 AI 生成 ==========

async function handleGenerateCharacters() {
  try {
    await novelStore.submitGenerateCharacters(
      Number(props.pid),
      form.value.setting,
      form.value.background,
      form.value.plot,
      characterModelName.value,
      userPrompt.value || undefined,
    )
    ElMessage.info('人物生成已启动')
  } catch {
    ElMessage.error('人物生成请求失败')
  }
}

function handleAcceptCharacters() {
  const result = novelStore.acceptCharacterGenResult()
  if (!result) return
  if (form.value.characters.trim()) {
    form.value.characters += '\n' + result
  } else {
    form.value.characters = result
  }
  ElMessage.success('人物设定已填入')
}

const novelTitle = ref('')
const novelDescription = ref('')
const editableChapters = ref<{ title: string; summary: string }[]>([])
const adopting = ref(false)

// 去掉 title 中已有的"第X章"前缀
function stripChapterPrefix(title: string): string {
  return title.replace(/^第[零一二三四五六七八九十百千万\d]+章半?\s*/, '')
}

// 监听 store 中大纲生成结果
watch(() => novelStore.outlineChapters, (chapters) => {
  if (chapters.length > 0) {
    editableChapters.value = chapters.map((ch) => ({
      ...ch,
      title: stripChapterPrefix(ch.title),
    }))
  }
}, { deep: true })

async function handleGenerate() {
  try {
    await novelStore.submitOutlineGenerate(
      Number(props.pid),
      form.value.setting,
      form.value.characters,
      form.value.background,
      form.value.plot,
      form.value.chapterNum,
      form.value.modelName,
      userPrompt.value || undefined,
      selectedStructureTemplateId.value,
      selectedHitAnalysisId.value || undefined,
    )
    ElMessage.info('Outline generation started')
  } catch {
    ElMessage.error('Failed to start outline generation')
  }
}

function addChapter() {
  editableChapters.value.push({ title: '', summary: '' })
}

function removeChapter(index: number) {
  editableChapters.value.splice(index, 1)
}

// 章节级 AI 操作
async function handleChapterAI(index: number, action: string) {
  const ch = editableChapters.value[index]
  if (!ch) return

  // 构建前后章节上下文
  const prevChapters = editableChapters.value.slice(Math.max(0, index - 3), index).map(c => ({ title: c.title, summary: c.summary }))
  const nextChapters = editableChapters.value.slice(index + 1, index + 3).map(c => ({ title: c.title, summary: c.summary }))

  try {
    await novelStore.submitOutlineChapterAI(
      Number(props.pid),
      index,
      action,
      ch,
      { setting: form.value.setting, prev_chapters: prevChapters, next_chapters: nextChapters },
      form.value.modelName,
      userPrompt.value || undefined,
    )
  } catch {
    ElMessage.error('AI action failed')
  }
}

function handleAcceptAI(index: number) {
  const result = novelStore.acceptOutlineAIResult(index)
  if (!result) return
  const ch = editableChapters.value[index]
  if (!ch) return
  if (result.title) ch.title = stripChapterPrefix(result.title)
  if (result.summary) ch.summary = result.summary
  ElMessage.success('AI result applied')
}

async function handleAdopt() {
  if (!novelTitle.value.trim()) {
    ElMessage.warning('Please enter a novel title')
    return
  }
  adopting.value = true
  try {
    const novel = await novelStore.adoptOutline(
      Number(props.pid),
      novelTitle.value,
      novelDescription.value,
      editableChapters.value,
    )
    ElMessage.success('Novel created from outline')
    router.push(`/workspace/${props.id}/portfolio/${props.pid}/novel/${novel.id}`)
  } catch {
    ElMessage.error('Failed to create novel')
  } finally {
    adopting.value = false
  }
}

// 多轮迭代：根据反馈重新生成
async function handleIterate() {
  if (!iterationFeedback.value.trim() || !novelStore.outlineTaskId) return
  try {
    await novelStore.submitOutlineGenerate(
      Number(props.pid),
      form.value.setting,
      form.value.characters,
      form.value.background,
      form.value.plot,
      form.value.chapterNum,
      form.value.modelName,
      userPrompt.value || undefined,
      selectedStructureTemplateId.value,
      selectedHitAnalysisId.value || undefined,
      novelStore.outlineTaskId,
      iterationFeedback.value,
    )
    iterationFeedback.value = ''
    ElMessage.info('正在根据反馈重新生成大纲...')
  } catch {
    ElMessage.error('迭代生成失败')
  }
}

onMounted(() => {
  connectWebSocket()
  handleLoadTemplates()
  loadStructureTemplates()
})

onUnmounted(() => {
  novelStore.clearOutline()
  novelStore.clearSearchResults()
  disconnectWebSocket()
})
</script>

<style scoped lang="scss">
.outline-workshop { width: 100%; max-width: 1400px; margin: 0 auto; }
.workshop-header {
  margin-bottom: 20px; padding: 14px 20px; background: var(--color-bg-card); border-radius: 12px;
  border: 1px solid var(--border-glow); box-shadow: var(--shadow-sm);
  transition: box-shadow 0.3s ease;
  &:hover { box-shadow: var(--shadow-md); }
}
.workshop-body { display: flex; gap: 24px; min-height: calc(100vh - 160px); }
.panel-title { font-size: 18px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 16px; }

.input-panel {
  width: 380px; flex-shrink: 0; padding: 20px;
  border: 1px solid var(--border-glow, #e4e7ed); border-radius: 8px;
  background: var(--bg-card, #fff); overflow-y: auto; max-height: calc(100vh - 160px);
}

.search-section { margin-bottom: 0; }
.search-bar { display: flex; gap: 8px; }

.character-gen-section { margin-bottom: 18px; }
.character-gen-bar { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }
.character-gen-loading { text-align: center; padding: 12px; }
.character-gen-preview {
  padding: 12px; border-radius: 6px;
  background: var(--el-color-primary-light-9, #ecf5ff); border: 1px solid var(--el-color-primary-light-7, #c6e2ff);
  &__label { font-size: 12px; font-weight: 600; color: var(--el-color-primary); margin-bottom: 8px; }
  &__text {
    font-size: 13px; color: var(--color-text-primary); line-height: 1.6;
    white-space: pre-wrap; word-break: break-word; margin: 0 0 8px;
    max-height: 200px; overflow-y: auto;
  }
  &__actions { display: flex; gap: 8px; justify-content: flex-end; }
}
.search-loading { margin-top: 12px; }
.search-results { margin-top: 12px; display: flex; flex-direction: column; gap: 10px; }

.search-result-card {
  padding: 10px 12px; border: 1px solid var(--border-glow, #e4e7ed); border-radius: 6px;
  background: var(--el-color-primary-light-9, #ecf5ff); transition: border-color 0.2s;
  &:hover { border-color: var(--el-color-primary); }
  &__header { display: flex; align-items: baseline; gap: 6px; margin-bottom: 4px; }
  &__title { font-size: 14px; font-weight: 600; color: #1d2129; }
  &__author { font-size: 12px; color: #4e5969; }
  &__intro {
    font-size: 12px; color: #4e5969; line-height: 1.5;
    margin: 4px 0 8px; display: -webkit-box; -webkit-line-clamp: 3;
    -webkit-box-orient: vertical; overflow: hidden;
  }
  &__footer { display: flex; justify-content: flex-end; }
}

.preview-panel {
  flex: 1; padding: 20px;
  border: 1px solid var(--border-glow, #e4e7ed); border-radius: 8px;
  background: var(--bg-card, #fff); overflow-y: auto; max-height: calc(100vh - 160px);
}

.loading-state { text-align: center; padding: 40px 20px; }
.loading-hint { margin-top: 16px; color: var(--color-text-secondary); font-size: 14px; }
.empty-state { text-align: center; padding: 60px 20px; color: var(--color-text-muted); font-size: 14px; }

.novel-meta { margin-bottom: 20px; padding-bottom: 16px; border-bottom: 1px solid var(--border-glow, #e4e7ed); }

.chapter-list { display: flex; flex-direction: column; gap: 16px; }
.chapter-card {
  padding: 16px; border: 1px solid var(--border-glow, #e4e7ed); border-radius: 8px;
  transition: border-color 0.3s;
  &--ai-active { border-color: var(--el-color-primary); }
  &__header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
  &__index {
    width: 28px; height: 28px; border-radius: 50%; background: var(--el-color-primary);
    color: #fff; display: flex; align-items: center; justify-content: center;
    font-size: 13px; font-weight: 600; flex-shrink: 0;
  }
  &__title-input { flex: 1; }
}

.ai-preview {
  margin-top: 12px; padding: 12px; border-radius: 6px;
  background: var(--el-color-primary-light-9, #ecf5ff); border: 1px solid var(--el-color-primary-light-7, #c6e2ff);
  &__label { font-size: 12px; font-weight: 600; color: var(--el-color-primary); margin-bottom: 8px; }
  &__field { margin-bottom: 8px; }
  &__tag {
    display: inline-block; font-size: 11px; font-weight: 600; color: var(--el-color-primary);
    background: var(--el-color-primary-light-8, #d9ecff); padding: 1px 6px; border-radius: 3px; margin-bottom: 4px;
  }
  &__text { font-size: 13px; color: var(--color-text-primary); line-height: 1.6; margin: 4px 0 0; }
  &__actions { display: flex; gap: 8px; margin-top: 8px; }
}

.outline-actions {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--border-glow, #e4e7ed);
}

.prompt-settings {
  .prompt-actions {
    display: flex; gap: 8px; justify-content: flex-end;
  }
}

.structure-preview {
  display: flex; flex-direction: column; gap: 8px;
  padding: 8px; background: var(--el-fill-color-lighter); border-radius: 6px;
}
.structure-phase {
  &__header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
  &__name { font-weight: 600; font-size: 13px; }
  &__beats { display: flex; flex-wrap: wrap; gap: 2px; }
}

.hit-analysis-badge { margin-top: 4px; }

.iteration-section {
  margin-top: 16px; padding: 12px; border-radius: 6px;
  background: var(--el-color-warning-light-9, #fdf6ec); border: 1px solid var(--el-color-warning-light-5, #f3d19e);
}
</style>
