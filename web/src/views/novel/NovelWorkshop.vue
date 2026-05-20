<!-- web/src/views/novel/NovelWorkshop.vue -->
<template>
  <div class="novel-workshop">
    <!-- 顶部导航 -->
    <div class="workshop-header">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">工作空间</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">作品集</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}/novels` }">小说</el-breadcrumb-item>
        <el-breadcrumb-item>{{ novelStore.currentNovel?.title || '...' }}</el-breadcrumb-item>
      </el-breadcrumb>
      <el-button size="small" type="primary" text @click="goToOverview">总览</el-button>
      <el-button size="small" type="success" text @click="showExportDialog = true">
        <el-icon><Download /></el-icon>
        导出
      </el-button>

      <!-- Token 使用指示器 -->
      <div class="token-indicator" @click="budgetInput = novelStore.tokenBudget; showBudgetDialog = true">
        <template v-if="novelStore.tokenBudget > 0">
          <el-progress
            :percentage="tokenDisplayPercentage"
            :stroke-width="8"
            :status="tokenProgressStatus"
            :show-text="false"
            style="width: 120px"
          />
          <span class="token-indicator__text">
            {{ formatTokenCount(novelStore.tokenUsed) }} / {{ formatTokenCount(novelStore.tokenBudget) }} tokens ({{ tokenDisplayPercentage }}%)
          </span>
        </template>
        <template v-else>
          <span class="token-indicator__text token-indicator__text--link">
            已用 {{ formatTokenCount(novelStore.tokenUsed) }} tokens · 点击设置预算
          </span>
        </template>
      </div>
    </div>

    <!-- Token 预算设置弹窗 -->
    <el-dialog v-model="showBudgetDialog" title="Token 预算设置" width="400px">
      <el-form label-width="80px">
        <el-form-item label="当前已用">
          <span>{{ formatTokenCount(novelStore.tokenUsed) }} tokens</span>
        </el-form-item>
        <el-form-item label="预算上限">
          <el-input-number
            v-model="budgetInput"
            :min="0"
            :step="10000"
            :max="10000000"
            controls-position="right"
            style="width: 100%"
          />
          <div style="color: var(--el-text-color-secondary); font-size: 12px; margin-top: 4px">
            设为 0 表示不限制预算
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showBudgetDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveBudget">保存</el-button>
      </template>
    </el-dialog>

    <div class="workshop-body">
      <!-- 左栏：章节列表 -->
      <aside class="chapter-list">
        <div class="chapter-list__header">
          <h3>章节</h3>
          <el-button size="small" type="primary" text @click="handleAddChapter">+ 新建</el-button>
        </div>
        <div class="chapter-list__items">
          <div
            v-for="ch in novelStore.chapters"
            :key="ch.id"
            class="chapter-item"
            :class="{ 'chapter-item--active': novelStore.currentChapter?.id === ch.id, 'chapter-item--has-content': !!ch.content }"
            @click="handleSelectChapter(ch)"
          >
            <span class="chapter-item__order">{{ ch.sort_order }}.</span>
            <el-tooltip :content="formatChapterTitle(ch)" placement="right" :show-after="300" :disabled="!isTitleOverflow(formatChapterTitle(ch))">
              <span class="chapter-item__title">{{ formatChapterTitle(ch) }}</span>
            </el-tooltip>
            <span class="chapter-item__status">{{ statusIcon(ch.status) }}</span>
            <el-icon v-if="novelStore.isChapterAIPending(ch.id) || batchState.tasks.some(t => t.chapterId === ch.id && t.status === 'running')" class="chapter-item__ai-indicator is-loading"><Loading /></el-icon>
            <span class="chapter-item__expand-btn" title="在此章节后扩写" @click.stop="openExpandAfter(ch.sort_order)">+</span>
          </div>
          <div v-if="novelStore.chapters.length === 0" class="chapter-list__empty">
            暂无章节
          </div>
        </div>
        <div class="chapter-list__footer">
          <el-button size="small" type="success" text @click="openExpandAtEnd">📖 扩写章节</el-button>
        </div>
      </aside>

      <!-- 中栏：编辑区 -->
      <main class="editor-area">
        <template v-if="novelStore.currentChapter">
          <div class="editor-area__header">
            <el-input
              v-model="editTitle"
              class="editor-area__title-input"
              placeholder="章节标题"
              @blur="handleTitleBlur"
            />
            <el-tag :type="chapterStatusType(novelStore.currentChapter.status)" size="small">
              {{ statusLabel(novelStore.currentChapter.status) }}
            </el-tag>
          </div>

          <!-- 概要区 -->
          <div class="editor-section">
            <label class="editor-section__label">概要</label>
            <!-- 概要润色 loading -->
            <div v-if="novelStore.aiPending && pendingAction === 'summary_polish'" class="summary-diff-loading">
              <div class="summary-diff-loading__headers">
                <div class="summary-diff-loading__header-cell">原文</div>
                <div class="summary-diff-loading__header-divider" />
                <div class="summary-diff-loading__header-cell">
                  AI 新内容
                  <el-tag type="warning" size="small" style="margin-left: 8px;">生成中...</el-tag>
                  <el-button type="danger" text size="small" style="margin-left: auto;" @click="handleCancelAIAction">取消</el-button>
                </div>
              </div>
              <div class="summary-diff-loading__panels">
                <div class="summary-diff-loading__panel">
                  <div class="summary-diff-loading__body">{{ editSummary }}</div>
                </div>
                <div class="summary-diff-loading__divider" />
                <div class="summary-diff-loading__panel">
                  <div class="summary-diff-loading__body">
                    <el-skeleton :rows="2" animated />
                  </div>
                </div>
              </div>
            </div>
            <!-- 概要对比视图：AI 返回概要时显示分栏对比 -->
            <ContentDiffView
              v-else-if="showSummaryDiff"
              :original="editSummary"
              :modified="novelStore.aiSummary || ''"
              :font-size="14"
              compact
              @accept="handleSummaryAccept"
              @discard="handleSummaryDiscard"
            />
            <el-input
              v-else
              v-model="editSummary"
              type="textarea"
              :rows="3"
              placeholder="章节概要..."
              resize="vertical"
            />
          </div>

          <!-- 正文区 -->
          <div class="editor-section editor-section--content">
            <div class="editor-section__toolbar">
              <label class="editor-section__label">正文</label>
              <div class="editor-section__controls">
                <el-tooltip content="联想开关 (Ctrl+Space 手动触发)" placement="top">
                  <el-switch
                    v-model="suggestionEnabled"
                    size="small"
                    active-text="联想"
                    style="margin-right: 8px;"
                  />
                </el-tooltip>
                <el-select v-model="editorFontFamily" size="small" class="font-select" placeholder="字体">
                  <el-option label="系统默认" value="default" />
                  <el-option label="思源宋体" value="'Noto Serif SC', serif" />
                  <el-option label="思源黑体" value="'Noto Sans SC', sans-serif" />
                  <el-option label="楷体" value="'KaiTi', 'STKaiti', serif" />
                  <el-option label="仿宋" value="'FangSong', 'STFangsong', serif" />
                  <el-option label="Consolas" value="'Consolas', monospace" />
                </el-select>
                <div class="font-size-ctrl">
                  <button class="font-size-btn" @click="editorFontSize = Math.max(12, editorFontSize - 1)">A-</button>
                  <span class="font-size-val">{{ editorFontSize }}px</span>
                  <button class="font-size-btn" @click="editorFontSize = Math.min(28, editorFontSize + 1)">A+</button>
                </div>
              </div>
            </div>

            <!-- 对比视图：AI 结果返回后替代编辑器 -->
            <ContentDiffView
              v-if="showDiffView"
              :original="diffOriginal"
              :modified="diffModified"
              :font-family="editorFontFamily"
              :font-size="editorFontSize"
              @accept="handleAccept"
              @discard="handleDiscard"
            />
            <!-- AI 生成中：正文区内左右分栏（左原文 + 右骨架屏） -->
            <div v-else-if="contentAIPending" class="content-diff-loading">
              <!-- 独立 header 行，确保左右对齐 -->
              <div class="content-diff-loading__headers">
                <div class="content-diff-loading__header-cell">原文</div>
                <div class="content-diff-loading__header-divider" />
                <div class="content-diff-loading__header-cell">
                  AI 新内容
                  <el-tag type="warning" size="small" style="margin-left: 8px;">{{ pendingActionLabel }}中...</el-tag>
                  <el-button type="danger" text size="small" style="margin-left: auto;" @click="handleCancelAIAction">取消</el-button>
                </div>
              </div>
              <div class="content-diff-loading__panels">
                <div class="content-diff-loading__panel">
                  <div class="content-diff-loading__body" :style="contentLoadingStyle">{{ editContent }}</div>
                </div>
                <div class="content-diff-loading__divider" />
                <div class="content-diff-loading__panel">
                  <div class="content-diff-loading__body">
                    <el-skeleton :rows="12" animated />
                  </div>
                </div>
              </div>
            </div>
            <!-- 正常编辑模式 -->
            <template v-else>
              <InlineSuggestionEditor
                ref="contentEditorRef"
                v-model="editContent"
                placeholder="开始写作..."
                :suggestion-enabled="suggestionEnabled"
                :novel-id="Number(nid)"
                :font-family="editorFontFamily"
                :font-size="editorFontSize"
                @selection-change="onEditorSelectionChange"
                @suggest-accept="onSuggestAccept"
                @suggest-reject="onSuggestReject"
              />
              <div class="editor-section__meta">
                {{ wordCount }} 字 · v{{ novelStore.currentChapter.current_version }}
              </div>
            </template>
          </div>

          <div class="editor-area__actions">
            <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
            <el-button type="danger" text @click="handleDeleteChapter">删除章节</el-button>
          </div>

          <!-- 章节插图画廊 -->
          <ImageGenerator
            :chapter-id="novelStore.currentChapter?.id || 0"
            :chapter-content="editContent"
            :portfolio-id="Number(pid)"
          />

        </template>
        <div v-else class="editor-area__empty">
          从左侧面板选择一个章节开始编辑
        </div>
      </main>

      <!-- 右栏：AI 操作 + 版本历史 -->
      <aside class="side-panel">
        <template v-if="novelStore.currentChapter">
          <!-- AI 操作 -->
          <div class="side-section">
            <el-tooltip content="AI 续写、润色、扩写、缩写等智能写作辅助" placement="left" :show-after="500">
              <div class="side-section__header" @click="aiSectionCollapsed = !aiSectionCollapsed">
                <h4 class="side-section__title">AI 操作</h4>
                <el-icon class="side-section__arrow" :class="{ 'is-collapsed': aiSectionCollapsed }"><ArrowDown /></el-icon>
              </div>
            </el-tooltip>
            <div v-show="!aiSectionCollapsed" class="side-section__body">
            <!-- 进行中工作流提示 -->
            <div v-if="workflowRunningElsewhere" class="workflow-hint" @click="jumpToWorkflowChapter">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span class="workflow-hint__text">
                「{{ workflowChapterTitle }}」正在生成中 · {{ workflowPercentage }}%
              </span>
              <el-button type="primary" link size="small">查看</el-button>
            </div>

            <!-- 生成操作 -->
            <div class="workflow-btns">
              <el-button
                v-if="workflowStore.pending && !batchState.active"
                type="danger"
                text bg
                size="small"
                class="workflow-btn"
                @click="handleCancelWorkflow"
              >
                <el-icon><CircleClose /></el-icon>
                取消生成
              </el-button>
              <el-button
                v-else
                type="success"
                text bg
                size="small"
                class="workflow-btn"
                title="根据大纲和上下文自动生成当前章节内容"
                :disabled="novelStore.aiPending || batchState.active"
                @click="handleFullChapter"
              >
                <el-icon><Promotion /></el-icon>
                生成章节
              </el-button>
              <el-button
                type="warning"
                text bg
                size="small"
                class="workflow-btn"
                title="批量生成多个章节，自动按顺序执行"
                :loading="batchState.active && !isFinished"
                :disabled="workflowStore.pending && !batchState.active"
                @click="handleBatchGenerate"
              >
                <el-icon><Promotion /></el-icon>
                批量生成
              </el-button>
            </div>
            <el-button
              v-if="batchState.active"
              type="primary"
              link
              size="small"
              style="margin-top: 4px;"
              @click="showBatchPanel = true"
            >
              查看批量进度 ({{ completedCount }}/{{ batchState.tasks.length }})
            </el-button>
            <el-progress
              v-if="showWorkflowProgress && !batchState.active"
              :percentage="workflowPercentage"
              :stroke-width="4"
              :status="workflowProgressStatus"
              style="margin-top: 6px;"
            />
            <div v-if="workflowStore.nodes.length && showWorkflowProgress && !batchState.active" class="workflow-nodes">
              <div
                v-for="node in sortedWorkflowNodes"
                :key="node.node_id"
                class="workflow-node"
                :class="`workflow-node--${node.status}`"
              >
                <span class="workflow-node__id">{{ nodeLabel(node.node_id) }}</span>
                <el-tag :type="workflowNodeTagType(node.status)" size="small">{{ nodeStatusLabel(node) }}</el-tag>
              </div>
            </div>

            <!-- 模型 & 设置 -->
            <div class="ai-config">
              <div class="ai-config__row">
                <ModelSelector v-model="selectedModel" size="small" placeholder="模型" />
                <el-button size="small" text class="ai-config__setting" @click="showTemplateDialog = true">
                  <el-icon><Setting /></el-icon>
                </el-button>
              </div>
              <el-select
                v-if="writingStyleStore.presets.length > 0"
                v-model="selectedScenePresetId"
                size="small"
                placeholder="场景预设（默认全局风格）"
                clearable
                style="margin-top: 6px;"
              >
                <el-option
                  v-for="preset in writingStyleStore.presets"
                  :key="preset.id"
                  :label="preset.name"
                  :value="preset.id"
                />
              </el-select>
            </div>

            <div v-if="selectedText" class="ai-selection-hint">
              <el-tag type="info" size="small">已选中: {{ selectedText.length }} 字符</el-tag>
            </div>

            <!-- AI 操作：2×2 网格，润色按钮带下拉菜单 -->
            <div class="ai-grid">
              <template v-for="action in aiActions" :key="action.value">
                <el-dropdown
                  v-if="action.value === 'polish'"
                  split-button
                  size="small"
                  :disabled="novelStore.aiPending"
                  class="ai-grid__dropdown"
                  @click="handleAIAction('polish')"
                  @command="(mode: string) => handleAIAction('polish', mode)"
                >
                  <el-icon class="ai-grid__icon"><component :is="action.icon" /></el-icon>
                  <span class="ai-grid__label">{{ action.label }}</span>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item
                        v-for="mode in polishModes"
                        :key="mode.value"
                        :command="mode.value"
                      >{{ mode.label }}</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
                <button
                  v-else
                  class="ai-grid__item"
                  :disabled="novelStore.aiPending"
                  @click="handleAIAction(action.value)"
                >
                  <el-icon class="ai-grid__icon"><component :is="action.icon" /></el-icon>
                  <span class="ai-grid__label">{{ action.label }}</span>
                </button>
              </template>
            </div>
            </div>
          </div>

          <!-- 知识库面板 -->
          <div class="side-section">
            <el-tooltip content="管理人物档案、世界观、剧情线索等结构化知识，AI 写作时自动引用" placement="left" :show-after="500">
              <div class="side-section__header" @click="knowledgeSectionCollapsed = !knowledgeSectionCollapsed">
                <h4 class="side-section__title">知识库</h4>
                <el-icon class="side-section__arrow" :class="{ 'is-collapsed': knowledgeSectionCollapsed }"><ArrowDown /></el-icon>
              </div>
            </el-tooltip>
            <div v-show="!knowledgeSectionCollapsed" class="side-section__body">
              <KnowledgePanel
                :novel-id="Number(nid)"
                :current-chapter-id="novelStore.currentChapter?.id"
                :model-name="selectedModel"
              />
            </div>
          </div>

          <!-- 写作风格面板（大神写手专属） -->
          <div v-if="userStore.showAdvancedUI" class="side-section">
            <el-tooltip content="设置叙事视角、文风调性、语言风格等，控制 AI 生成内容的风格" placement="left" :show-after="500">
              <div class="side-section__header" @click="styleSectionCollapsed = !styleSectionCollapsed">
                <h4 class="side-section__title">写作风格</h4>
                <el-icon class="side-section__arrow" :class="{ 'is-collapsed': styleSectionCollapsed }"><ArrowDown /></el-icon>
              </div>
            </el-tooltip>
            <div v-show="!styleSectionCollapsed" class="side-section__body">
              <WritingStylePanel :novel-id="Number(nid)" />
            </div>
          </div>

          <!-- 写作记忆（大神写手专属） -->
          <div v-if="userStore.showAdvancedUI" class="side-section">
            <el-tooltip content="绑定写作记忆，AI 生成时自动注入风格、人设等约束" placement="left" :show-after="500">
              <div class="side-section__header" @click="memorySectionCollapsed = !memorySectionCollapsed">
                <h4 class="side-section__title">写作记忆</h4>
                <el-icon class="side-section__arrow" :class="{ 'is-collapsed': memorySectionCollapsed }"><ArrowDown /></el-icon>
              </div>
            </el-tooltip>
            <div v-show="!memorySectionCollapsed" class="side-section__body">
              <MemoryBindingPanel :novel-id="Number(nid)" />
            </div>
          </div>

          <!-- 版本历史 -->
          <div class="side-section">
            <el-tooltip content="查看和回滚章节的历史版本" placement="left" :show-after="500">
              <div class="side-section__header" @click="versionSectionCollapsed = !versionSectionCollapsed">
                <h4 class="side-section__title">版本历史</h4>
                <el-icon class="side-section__arrow" :class="{ 'is-collapsed': versionSectionCollapsed }"><ArrowDown /></el-icon>
              </div>
            </el-tooltip>
            <div v-show="!versionSectionCollapsed" class="side-section__body">
            <div class="version-list">
              <div
                v-for="v in novelStore.versions"
                :key="v.id"
                class="version-item"
                @click="handleRevert(v.id)"
              >
                <div class="version-item__header">
                  <span class="version-item__num">v{{ v.version }}</span>
                  <el-tag size="small" :type="sourceTagType(v.source)">{{ v.source }}</el-tag>
                </div>
                <div class="version-item__meta">
                  {{ v.word_count }} 字 · {{ formatDate(v.created_at) }}
                </div>
              </div>
              <div v-if="novelStore.versions.length === 0" class="version-list__empty">
                暂无版本
              </div>
            </div>
            </div>
          </div>

          <!-- 多媒体面板 -->
          <div class="side-section">
            <el-tooltip content="为章节生成配音和视频，管理多媒体资产" placement="left" :show-after="500">
              <div class="side-section__header" @click="mediaSectionCollapsed = !mediaSectionCollapsed">
                <h4 class="side-section__title">多媒体</h4>
                <el-icon class="side-section__arrow" :class="{ 'is-collapsed': mediaSectionCollapsed }"><ArrowDown /></el-icon>
              </div>
            </el-tooltip>
            <div v-show="!mediaSectionCollapsed" class="side-section__body">
              <ChapterMediaPanel
                :chapter-id="novelStore.currentChapter?.id || 0"
                :chapter-content="editContent"
                :chapter-summary="editSummary"
                :portfolio-id="Number(pid)"
              />
            </div>
          </div>
        </template>
      </aside>
    </div>

    <!-- 导出对话框 -->
    <ExportDialog v-model="showExportDialog" :novel-id="Number(nid)" />

    <!-- Prompt 模板编辑对话框 -->
    <el-dialog v-model="showTemplateDialog" title="Prompt 模板" width="800px" destroy-on-close>
      <div class="template-dialog">
        <div class="template-dialog__vars">
          <span class="template-dialog__vars-label">可用变量：</span>
          <el-tag v-for="v in templateVars" :key="v" size="small" type="info" style="margin: 2px;">{{ v }}</el-tag>
        </div>
        <el-tabs v-model="activeTemplateTab">
          <el-tab-pane v-for="action in aiActions" :key="action.value" :label="action.label" :name="action.value">
            <div class="template-editor">
              <div class="template-editor__section">
                <label>系统提示词补充 <span style="font-weight:normal;color:var(--el-text-color-placeholder);font-size:11px">（追加到默认系统提示词之后）</span></label>
                <el-input
                  v-model="templateEdits[action.value + ':system']"
                  type="textarea"
                  :rows="4"
                  placeholder="输入补充指令，将追加到系统默认提示词之后生效..."
                />
              </div>
              <div class="template-editor__section">
                <label>用户提示词补充 <span style="font-weight:normal;color:var(--el-text-color-placeholder);font-size:11px">（追加到默认用户提示词之后）</span></label>
                <el-input
                  v-model="templateEdits[action.value + ':user']"
                  type="textarea"
                  :rows="8"
                  placeholder="输入补充指令，如写作禁忌、风格要求等，将追加到默认提示词之后生效..."
                />
              </div>
              <div class="template-editor__section" v-if="templatePreviewResult">
                <label>预览</label>
                <div class="template-editor__preview">{{ templatePreviewResult }}</div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
      <template #footer>
        <el-button @click="handlePreviewTemplate">预览</el-button>
        <el-button @click="handleResetTemplate">恢复默认</el-button>
        <el-button type="primary" :loading="templateSaving" @click="handleSaveTemplates">保存</el-button>
      </template>
    </el-dialog>

    <!-- 批量生成任务面板 -->
    <el-drawer
      v-model="showBatchPanel"
      title="批量生成空白章节"
      direction="rtl"
      size="380px"
      :close-on-click-modal="false"
      @close="handleCloseBatchPanel"
    >
      <div class="batch-panel">
        <!-- 整体进度 -->
        <div class="batch-panel__summary">
          <el-progress :percentage="batchProgress" :stroke-width="8" />
          <div class="batch-panel__stats">
            <span>{{ completedCount }} / {{ batchState.tasks.length }} 完成</span>
            <span v-if="failedCount > 0" style="color: var(--el-color-danger);">{{ failedCount }} 失败</span>
          </div>
        </div>

        <!-- 任务列表 -->
        <div class="batch-panel__list">
          <div
            v-for="(task, idx) in batchState.tasks"
            :key="task.chapterId"
            class="batch-task-item"
            :class="`batch-task-item--${task.status}`"
          >
            <span class="batch-task-item__order">{{ task.sortOrder }}.</span>
            <span class="batch-task-item__title">{{ task.title }}</span>
            <div class="batch-task-item__right">
              <el-progress
                v-if="task.status === 'running'"
                type="circle"
                :percentage="task.progress"
                :width="28"
                :stroke-width="3"
              />
              <el-tag v-else :type="batchTaskTagType(task.status)" size="small">
                {{ batchTaskStatusIcon(task.status) }} {{ task.status }}
              </el-tag>
            </div>
            <div v-if="task.error" class="batch-task-item__error">{{ task.error }}</div>
          </div>
        </div>

        <!-- 底部操作 -->
        <div class="batch-panel__footer">
          <el-button
            v-if="!isFinished"
            type="danger"
            @click="cancelBatch"
          >
            取消批量生成
          </el-button>
          <el-button
            v-else
            type="primary"
            @click="handleCloseBatchPanel"
          >
            关闭
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 扩写章节弹窗 -->
    <ExpandChapterDialog
      v-model:visible="showExpandDialog"
      :chapters="novelStore.chapters"
      :novel-id="Number(nid)"
      :default-insert-after="expandInsertAfterDefault"
      @inserted="handleExpandInserted"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { Loading, ArrowDown, MagicStick, EditPen, Reading, Edit as EditIcon, Promotion, Setting, Headset, Download, CircleClose } from '@element-plus/icons-vue'
import { useNovelStore } from '@/store/novel'
import { useWorkflowStore, isCompleted } from '@/store/workflow'
import { useWritingStyleStore } from '@/store/writing_style'
import { useUserStore } from '@/store/user'
import { useBatchGenerate } from '@/composables/useBatchGenerate'
import { connectWebSocket, disconnectWebSocket } from '@/utils/websocket'
import { promptTemplateApi } from '@/api/prompt_template'
import { workflowApi } from '@/api/workflow'
import type { PromptTemplate } from '@/api/prompt_template'
import type { Chapter } from '@/api/novel'
import KnowledgePanel from './KnowledgePanel.vue'
import WritingStylePanel from './WritingStylePanel.vue'
import MemoryBindingPanel from './MemoryBindingPanel.vue'
import ExpandChapterDialog from './ExpandChapterDialog.vue'
import ChapterMediaPanel from './ChapterMediaPanel.vue'
import ImageGenerator from './ImageGenerator.vue'
import ExportDialog from './ExportDialog.vue'
import InlineSuggestionEditor from '@/components/editor/InlineSuggestionEditor.vue'
import ContentDiffView from '@/components/editor/ContentDiffView.vue'
import ModelSelector from '@/components/common/ModelSelector.vue'

const props = defineProps<{ id: string; pid: string; nid: string }>()
const router = useRouter()
const novelStore = useNovelStore()
const workflowStore = useWorkflowStore()
const writingStyleStore = useWritingStyleStore()
const userStore = useUserStore()

function goToOverview() {
  router.push({ name: 'NovelOverview', params: { id: props.id, pid: props.pid, nid: props.nid } })
}

const editTitle = ref('')
const editSummary = ref('')
const editContent = ref('')
const saving = ref(false)
const pendingAction = ref('')  // 当前 AI 操作类型

// 编辑器字体设置（持久化到 localStorage）
const editorFontSize = ref(Number(localStorage.getItem('editor-font-size')) || 15)
const editorFontFamily = ref(localStorage.getItem('editor-font-family') || 'default')

watch(editorFontSize, (v) => localStorage.setItem('editor-font-size', String(v)))
watch(editorFontFamily, (v) => localStorage.setItem('editor-font-family', v))

const contentStyle = computed(() => ({
  '--content-font-size': `${editorFontSize.value}px`,
  '--content-font-family': editorFontFamily.value === 'default' ? 'inherit' : editorFontFamily.value,
}))
// 对比视图：AI 结果返回后显示（包括 AI 操作和工作流生成）
const workflowDiffResult = ref<string | null>(null) // 工作流生成结果（待对比）
const workflowDiffOriginal = ref<string>('') // 工作流对比时的原文快照

const showDiffView = computed(() =>
  ((novelStore.aiResult !== null && novelStore.aiResult !== '' && !novelStore.aiPending) || workflowDiffResult.value !== null)
)

// 正文类 AI 操作 loading 状态（润色/扩写/续写，排除概要润色）
const contentAIPending = computed(() =>
  novelStore.aiPending && pendingAction.value !== '' && pendingAction.value !== 'summary_polish'
)

const pendingActionLabel = computed(() => {
  const map: Record<string, string> = { polish: '润色', expand: '扩写', continue: '续写' }
  return map[pendingAction.value] || '生成'
})

const contentLoadingStyle = computed(() => ({
  fontFamily: editorFontFamily.value === 'default' ? 'inherit' : editorFontFamily.value,
  fontSize: `${editorFontSize.value}px`,
}))

// 概要对比视图：AI 返回了概要且不在 pending 状态
const showSummaryDiff = computed(() =>
  !!novelStore.aiSummary && !novelStore.aiPending
)

const diffModified = computed(() => novelStore.aiResult ?? workflowDiffResult.value ?? '')

// 对比视图的原文：划词模式只展示选中片段，全文模式展示完整正文
const diffOriginal = computed(() => {
  if (workflowDiffResult.value !== null) {
    return workflowDiffOriginal.value
  }
  const range = novelStore.selectionRange
  if (range) {
    return editContent.value.slice(range.start, range.end)
  }
  return editContent.value
})

const workflowChapterId = ref<number | null>(null)
const selectedModel = ref('qwen')
const aiSectionCollapsed = ref(false)
const versionSectionCollapsed = ref(false)
const knowledgeSectionCollapsed = ref(true)
const styleSectionCollapsed = ref(true)
const memorySectionCollapsed = ref(true)
const mediaSectionCollapsed = ref(true)
const showExportDialog = ref(false)
const selectedScenePresetId = ref<number | undefined>(undefined)

// 批量生成
const {
  batchState, batchProgress, completedCount, failedCount, isFinished,
  startBatch, cancelBatch, resetBatch, recoverBatch,
} = useBatchGenerate(
  () => Number(props.pid),
  () => selectedModel.value,
  () => Number(props.nid),
)
const showBatchPanel = ref(false)
const contentInputRef = ref<InstanceType<typeof import('element-plus')['ElInput']> | null>(null)
const contentEditorRef = ref<InstanceType<typeof InlineSuggestionEditor> | null>(null)
const suggestionEnabled = ref(localStorage.getItem('suggestion_enabled') !== 'false')

// 划词选中文本状态
const selectedText = ref('')
const selectionStart = ref(0)
const selectionEnd = ref(0)

// 扩写章节弹窗
const showExpandDialog = ref(false)
const expandInsertAfterDefault = ref(0)

// Token 预算设置
const showBudgetDialog = ref(false)
const budgetInput = ref(0)
const tokenWarningShown = ref(false) // 防止重复弹出 80% 提醒
const tokenDangerShown = ref(false)  // 防止重复弹出 95% 提醒

// Token 进度条状态
const tokenProgressStatus = computed(() => {
  const pct = novelStore.tokenPercentage
  if (pct >= 95) return 'exception'
  if (pct >= 80) return 'warning'
  return ''
})

// 格式化 token 数量
function formatTokenCount(count: number): string {
  if (count >= 1000000) return (count / 1000000).toFixed(1) + 'M'
  if (count >= 1000) return (count / 1000).toFixed(1) + 'K'
  return String(count)
}

// 百分比四舍五入到最多 2 位小数
const tokenDisplayPercentage = computed(() => {
  return Math.round(novelStore.tokenPercentage * 100) / 100
})

// 保存预算
async function handleSaveBudget() {
  try {
    await novelStore.updateTokenBudget(Number(props.nid), budgetInput.value)
    showBudgetDialog.value = false
    tokenWarningShown.value = false
    tokenDangerShown.value = false
    ElMessage.success('预算已更新')
  } catch {
    ElMessage.error('更新预算失败')
  }
}

// 监听 token 百分比变化，触发阈值提醒
watch(() => novelStore.tokenPercentage, (pct) => {
  if (novelStore.tokenBudget <= 0) return
  if (pct >= 100) {
    ElMessageBox.confirm(
      'Token 预算已用完，是否调大预算继续使用？',
      'Token 预算已满',
      { confirmButtonText: '调大预算', cancelButtonText: '暂停使用', type: 'warning' }
    ).then(() => {
      budgetInput.value = novelStore.tokenBudget * 2
      showBudgetDialog.value = true
    }).catch(() => { /* 用户选择暂停 */ })
  } else if (pct >= 95 && !tokenDangerShown.value) {
    tokenDangerShown.value = true
    ElNotification({ title: 'Token 预算警告', message: `已使用 ${pct.toFixed(1)}%，即将达到预算上限`, type: 'error', duration: 5000 })
  } else if (pct >= 80 && !tokenWarningShown.value) {
    tokenWarningShown.value = true
    ElNotification({ title: 'Token 预算提醒', message: `已使用 ${pct.toFixed(1)}%，接近预算上限`, type: 'warning', duration: 5000 })
  }
})

function updateSelection() {
  const textarea = contentInputRef.value?.textarea as HTMLTextAreaElement | undefined
  if (!textarea) {
    selectedText.value = ''
    return
  }
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  if (start !== end) {
    selectedText.value = editContent.value.slice(start, end)
    selectionStart.value = start
    selectionEnd.value = end
  } else {
    selectedText.value = ''
  }
}

// Tiptap 编辑器选区变化回调
function onEditorSelectionChange(text: string, start: number, end: number) {
  selectedText.value = text
  selectionStart.value = start
  selectionEnd.value = end
}

// 联想采纳/拒绝回调
function onSuggestAccept(_suggestion: string) {
  // 行为已在 useSuggestion 中上报
}
function onSuggestReject(_suggestion: string) {
  // 行为已在 useSuggestion 中上报
}

const aiActions = [
  { value: 'summary_polish', label: '概要润色', icon: EditPen },
  { value: 'polish', label: '润色', icon: MagicStick },
  { value: 'expand', label: '扩写', icon: Reading },
  { value: 'continue', label: '续写', icon: EditIcon },
]

// 润色方向预设
const polishModes = [
  { value: '', label: '全面润色' },
  { value: 'dialogue', label: '对话优化' },
  { value: 'pacing', label: '节奏调整' },
  { value: 'sensory', label: '感官强化' },
  { value: 'emotion', label: '情感深化' },
  { value: 'trim', label: '精简去冗' },
]

const wordCount = computed(() => {
  return editContent.value.length
})

// 工作流进度显示控制
const showWorkflowProgress = ref(false)
let hideProgressTimer: ReturnType<typeof setTimeout> | null = null

const workflowPercentage = computed(() => {
  return Math.min(100, Math.round(workflowStore.progress * 100))
})

const workflowProgressStatus = computed(() => {
  const status = workflowStore.currentWorkflow?.status
  if (isCompleted(status || '')) return 'success'
  if (status === 'failed') return 'exception'
  return undefined
})

// 是否有工作流在其他章节执行中
const workflowRunningElsewhere = computed(() => {
  return workflowStore.pending
    && workflowChapterId.value !== null
    && novelStore.currentChapter?.id !== workflowChapterId.value
})

// 进行中工作流的章节标题
const workflowChapterTitle = computed(() => {
  if (!workflowChapterId.value) return ''
  const ch = novelStore.chapters.find(c => c.id === workflowChapterId.value)
  return ch?.title || ''
})

// 跳转到工作流发起章节
function jumpToWorkflowChapter() {
  if (!workflowChapterId.value) return
  const ch = novelStore.chapters.find(c => c.id === workflowChapterId.value)
  if (ch) {
    novelStore.selectChapter(ch)
  }
}

// 监听当前章节变化，同步编辑区
watch(() => novelStore.currentChapter, (ch) => {
  if (ch) {
    editTitle.value = stripChapterPrefix(ch.title)
    editSummary.value = ch.summary || ''
    editContent.value = ch.content || ''
    // 加载版本历史
    novelStore.fetchVersions(ch.id)
    // 自动设置章节绑定的默认场景预设
    selectedScenePresetId.value = ch.scene_preset_id ?? undefined
  }
}, { immediate: true })

onMounted(async () => {
  connectWebSocket()
  await novelStore.fetchNovel(Number(props.nid))
  await novelStore.fetchChapters(Number(props.nid))
  // 加载写作风格和场景预设
  writingStyleStore.fetchPresets(Number(props.nid))
  // 加载 token 使用情况
  novelStore.fetchTokenUsage(Number(props.nid))
  // 初始化预算输入框
  budgetInput.value = novelStore.tokenBudget
  // 默认选中第一章
  if (novelStore.chapters.length > 0) {
    novelStore.selectChapter(novelStore.chapters[0])
  }

  // 恢复进行中的任务
  try {
    const res: any = await workflowApi.listActive(Number(props.nid))
    const activeWorkflows = res?.workflows || []
    if (activeWorkflows.length > 0) {
      // 尝试恢复批量生成
      const recovered = recoverBatch(activeWorkflows)
      if (recovered) {
        showBatchPanel.value = true
      } else {
        // 非批量 → 恢复单章工作流
        const wf = activeWorkflows[0]
        workflowStore.recoverFromWorkflow(wf)
        // 尝试从工作流的 initial_context 中推断章节
        try {
          const ctx = JSON.parse(wf.initial_context || '{}')
          // 优先用 chapter_id 精确匹配，fallback 到 title
          if (ctx.chapter_id) {
            const ch = novelStore.chapters.find(c => c.id === ctx.chapter_id)
            workflowChapterId.value = ch?.id || null
          } else if (ctx.title) {
            const ch = novelStore.chapters.find(c => c.title === ctx.title)
            workflowChapterId.value = ch?.id || null
          }
        } catch { /* 解析失败忽略 */ }
        showWorkflowProgress.value = true
      }
    }
  } catch {
    // 恢复失败不影响正常使用
  }
})

onUnmounted(() => {
  disconnectWebSocket()
  if (hideProgressTimer) {
    clearTimeout(hideProgressTimer)
    hideProgressTimer = null
  }
})

// 监听工作流完成，将生成结果回填到编辑区（批量模式下由 composable 处理）
watch(() => workflowStore.currentWorkflow?.status, async (status) => {
  if (batchState.value.active) return
  if (isCompleted(status || '') && workflowStore.currentWorkflow?.result_json) {
    try {
      const result = JSON.parse(workflowStore.currentWorkflow.result_json)
      // 一键生成完整章节的最终结果在 final_result 字段
      // executor 返回的是 { content: "..." } 结构
      const finalResult = result.final_result
      const content = typeof finalResult === 'string' ? finalResult : finalResult?.content
      if (content && workflowChapterId.value) {
        const targetChapter = novelStore.chapters.find(c => c.id === workflowChapterId.value)
        const existingContent = (targetChapter?.content || '').trim()
        const isCurrentChapter = novelStore.currentChapter?.id === workflowChapterId.value

        if (existingContent && isCurrentChapter) {
          // 章节有内容且正在查看：进入 diff 对比视图，用户确认后再保存
          workflowDiffOriginal.value = editContent.value
          workflowDiffResult.value = content
        } else {
          // 章节为空或不在当前查看：直接保存
          await novelStore.updateChapter(workflowChapterId.value, {
            title: targetChapter?.title || '',
            summary: targetChapter?.summary || '',
            content,
          })
          if (isCurrentChapter) {
            editContent.value = content
            ElMessage.success('章节内容已生成并保存')
          } else {
            ElMessage.success('章节内容已生成并保存（切换回该章节查看）')
          }
        }
        workflowChapterId.value = null
      }
    } catch (e) {
      console.error('Failed to parse workflow result:', e)
    }
    // 完成后延迟隐藏进度条，让用户看到 100%
    // 同时刷新 token 使用情况
    novelStore.fetchTokenUsage(Number(props.nid))
    hideProgressTimer = setTimeout(() => {
      showWorkflowProgress.value = false
      workflowStore.reset()
    }, 3000)
  } else if (status === 'failed') {
    ElMessage.error('工作流失败: ' + (workflowStore.currentWorkflow?.error_msg || '未知错误'))
    hideProgressTimer = setTimeout(() => {
      showWorkflowProgress.value = false
      workflowStore.reset()
    }, 3000)
  }
})

// ========== 章节操作 ==========

async function handleAddChapter() {
  try {
    const { value } = await ElMessageBox.prompt('请输入章节标题', '新建章节', {
      inputPlaceholder: '章节标题',
      inputValidator: (v) => !!v?.trim() || '标题不能为空',
    })
    const ch = await novelStore.createChapter(Number(props.nid), value)
    novelStore.selectChapter(ch)
    ElMessage.success('章节已创建')
  } catch { /* cancelled */ }
}

function handleSelectChapter(ch: Chapter) {
  // 如果有未保存的更改，提示用户
  novelStore.selectChapter(ch)
}

function handleTitleBlur() {
  // 标题变更在保存时一起提交
}

async function handleSave() {
  if (!novelStore.currentChapter) return
  saving.value = true
  try {
    await novelStore.updateChapter(novelStore.currentChapter.id, {
      title: editTitle.value,
      summary: editSummary.value,
      content: editContent.value,
    })
    ElMessage.success('已保存')
  } finally {
    saving.value = false
  }
}

async function handleDeleteChapter() {
  if (!novelStore.currentChapter) return
  try {
    await ElMessageBox.confirm('确定删除该章节？', '警告', { type: 'warning' })
    await novelStore.deleteChapter(novelStore.currentChapter.id)
    // 选中第一章或清空
    if (novelStore.chapters.length > 0) {
      novelStore.selectChapter(novelStore.chapters[0])
    }
    ElMessage.success('章节已删除')
  } catch { /* cancelled */ }
}

// ========== AI 操作 ==========

async function handleAIAction(action: string, polishMode?: string) {
  if (!novelStore.currentChapter) return
  try {
    pendingAction.value = action
    // polish/expand 支持划词模式，其他操作忽略选中文本
    const supportsSelection = action === 'polish' || action === 'expand'
    const selText = supportsSelection && selectedText.value ? selectedText.value : undefined
    const selRange = supportsSelection && selectedText.value
      ? { start: selectionStart.value, end: selectionEnd.value }
      : null
    // 概要润色不需要传正文内容
    const content = action === 'summary_polish' ? undefined : editContent.value
    await novelStore.submitAIAction(
      novelStore.currentChapter.id, action, selectedModel.value,
      editSummary.value, content, selText, selRange,
      selectedScenePresetId.value, polishMode || undefined,
    )
    ElMessage.info('AI 任务已提交，等待结果...')
  } catch {
    ElMessage.error('提交 AI 任务失败')
  }
}

async function handleCancelAIAction() {
  if (!novelStore.pendingTaskId) return
  try {
    await import('@/api/ai').then(m => m.aiApi.cancelTask(novelStore.pendingTaskId!))
    novelStore.discardResult()
    pendingAction.value = ''
    ElMessage.info('已取消')
  } catch {
    ElMessage.error('取消失败')
  }
}

async function handleAccept() {
  if (!novelStore.currentChapter) return
  try {
    // 工作流生成结果的采纳
    if (workflowDiffResult.value !== null) {
      const content = workflowDiffResult.value
      await novelStore.updateChapter(novelStore.currentChapter.id, {
        title: editTitle.value,
        summary: editSummary.value,
        content,
      })
      editContent.value = content
      workflowDiffResult.value = null
      workflowDiffOriginal.value = ''
      ElMessage.success('已采纳生成结果')
      return
    }
    // AI 操作结果的采纳
    const range = novelStore.selectionRange
    if (range && novelStore.aiResult) {
      // 划词模式：用 AI 结果替换选中区间
      editContent.value = editContent.value.slice(0, range.start)
        + novelStore.aiResult
        + editContent.value.slice(range.end)
      // 保存完整内容
      await novelStore.updateChapter(novelStore.currentChapter.id, {
        title: editTitle.value,
        summary: editSummary.value,
        content: editContent.value,
      })
      novelStore.discardResult()
      selectedText.value = ''
    } else {
      // 全文模式：原有逻辑
      await novelStore.acceptResult(novelStore.currentChapter.id)
      editSummary.value = novelStore.currentChapter?.summary || ''
      editContent.value = novelStore.currentChapter?.content || ''
    }
    ElMessage.success('已采纳 AI 结果')
  } catch {
    ElMessage.error('采纳结果失败')
  }
}

function handleDiscard() {
  if (workflowDiffResult.value !== null) {
    workflowDiffResult.value = null
    workflowDiffOriginal.value = ''
    return
  }
  novelStore.discardResult()
}

// 概要对比：采纳
async function handleSummaryAccept() {
  if (!novelStore.currentChapter || !novelStore.aiSummary) return
  try {
    editSummary.value = novelStore.aiSummary
    await novelStore.updateChapter(novelStore.currentChapter.id, {
      title: editTitle.value,
      summary: editSummary.value,
      content: editContent.value,
    })
    // 清掉概要部分，保留正文对比（如果有）
    novelStore.aiSummary = null
    // 清理 chapterAIMap 中的概要状态，防止切换章节后残留
    novelStore.clearChapterAISummary(novelStore.currentChapter.id)
    ElMessage.success('已采纳概要')
  } catch {
    ElMessage.error('采纳概要失败')
  }
}

// 概要对比：丢弃
function handleSummaryDiscard() {
  novelStore.aiSummary = null
  if (novelStore.currentChapter) {
    novelStore.clearChapterAISummary(novelStore.currentChapter.id)
  }
}

// ========== 版本管理 ==========

async function handleRevert(versionId: number) {
  if (!novelStore.currentChapter) return
  try {
    await ElMessageBox.confirm('回退到此版本？当前内容将保存为新版本。', '回退', { type: 'warning' })
    await novelStore.revertToVersion(novelStore.currentChapter.id, versionId)
    editSummary.value = novelStore.currentChapter?.summary || ''
    editContent.value = novelStore.currentChapter?.content || ''
    ElMessage.success('已回退')
  } catch { /* cancelled */ }
}

// ========== 工作流操作 ==========

async function handleFullChapter() {
  if (!novelStore.currentChapter) return
  try {
    // 清理上次的定时器和状态
    if (hideProgressTimer) {
      clearTimeout(hideProgressTimer)
      hideProgressTimer = null
    }
    showWorkflowProgress.value = true
    // 记录发起工作流的章节 ID，防止切换章节后结果写入错误章节
    workflowChapterId.value = novelStore.currentChapter.id

    // 构建前几章上下文，帮助 AI 区分相邻章节
    const curSort = novelStore.currentChapter.sort_order
    const prevChapters = novelStore.chapters
      .filter(c => c.sort_order < curSort)
      .sort((a, b) => a.sort_order - b.sort_order)
      .slice(-5)
    const prevContext = prevChapters
      .map(c => `第${c.sort_order}章「${c.title}」：${c.summary || '（暂无概要）'}`)
      .join('\n')

    await workflowStore.submitWorkflow(
      Number(props.pid),
      'full_chapter',
      selectedModel.value,
      {
        title: novelStore.currentChapter.title,
        background: novelStore.currentChapter.summary || '',
        prev_context: prevContext,
        novel_id: Number(props.nid),
        chapter_id: novelStore.currentChapter.id,
        chapter_sort_order: novelStore.currentChapter.sort_order,
      },
    )
    ElMessage.success('工作流已提交')
  } catch {
    showWorkflowProgress.value = false
    ElMessage.error('提交工作流失败')
  }
}

async function handleCancelWorkflow() {
  const wfId = workflowStore.currentWorkflow?.id
  if (!wfId) return
  try {
    await ElMessageBox.confirm('确定取消当前生成任务？', '取消生成', { type: 'warning' })
    await workflowStore.cancelWorkflow(wfId)
    showWorkflowProgress.value = false
    ElMessage.info('已取消生成')
  } catch { /* 用户取消确认 */ }
}

function workflowNodeTagType(status: string) {
  const map: Record<string, string> = {
    pending: 'info',
    running: 'warning',
    completed: 'success',
    completed_with_warning: 'warning',
    failed: 'danger',
    skipped: 'info',
    loop_round: 'warning',
  }
  return map[status] || 'info'
}

const nodeNameMap: Record<string, string> = {
  outline: '大纲生成',
  draft: '骨架初稿',
  review_loop: '审核修订',
  review: '审核',
  revision: '修订',
}

// 节点固定排序（按 DAG 执行顺序）
const nodeOrderMap: Record<string, number> = {
  outline: 0,
  draft: 1,
  review_loop: 2,
  review: 3,
  revision: 4,
}

const sortedWorkflowNodes = computed(() => {
  return [...workflowStore.nodes].sort((a, b) => {
    const oa = nodeOrderMap[a.node_id] ?? 99
    const ob = nodeOrderMap[b.node_id] ?? 99
    return oa - ob
  })
})

function nodeLabel(nodeId: string) {
  return nodeNameMap[nodeId] || nodeId
}

function nodeStatusLabel(node: { status: string; result_json: string }) {
  if (node.status === 'loop_round') {
    try {
      const data = JSON.parse(node.result_json || '{}')
      if (data.round && data.max_rounds) return `第${data.round}轮`
    } catch { /* ignore */ }
    return '进行中'
  }
  if (node.status === 'completed_with_warning') return '完成(有警告)'
  const map: Record<string, string> = {
    pending: '等待中',
    running: '执行中',
    completed: '已完成',
    failed: '失败',
    skipped: '跳过',
  }
  return map[node.status] || node.status
}

// ========== 批量生成操作 ==========

function handleBatchGenerate() {
  const started = startBatch()
  if (started) {
    showBatchPanel.value = true
  }
}

function handleCloseBatchPanel() {
  if (isFinished.value) {
    resetBatch()
  }
  showBatchPanel.value = false
}

// 批量任务状态图标
function batchTaskStatusIcon(status: string) {
  const map: Record<string, string> = {
    pending: '⏳',
    running: '🔄',
    completed: '✅',
    failed: '❌',
    cancelled: '⛔',
  }
  return map[status] || '⏳'
}

function batchTaskTagType(status: string) {
  const map: Record<string, string> = {
    pending: 'info',
    running: 'warning',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info',
  }
  return map[status] || 'info'
}

// ========== 扩写章节操作 ==========

function openExpandAfter(sortOrder: number) {
  expandInsertAfterDefault.value = sortOrder
  showExpandDialog.value = true
}

function openExpandAtEnd() {
  const chs = novelStore.chapters
  expandInsertAfterDefault.value = chs.length > 0 ? chs[chs.length - 1].sort_order : 0
  showExpandDialog.value = true
}

async function handleExpandInserted() {
  await novelStore.fetchChapters(Number(props.nid))
}

// ========== 工具函数 ==========

function statusIcon(s: string) {
  const map: Record<string, string> = { draft: '○', polished: '●', final: '✓' }
  return map[s] || '○'
}

function isTitleOverflow(title: string) {
  return title.length > 8
}

// 去掉 title 中已有的"第X章"前缀，用 sort_order 动态拼接章节号
function stripChapterPrefix(title: string): string {
  return title.replace(/^第[零一二三四五六七八九十百千万\d]+章半?\s*/, '')
}

function formatChapterTitle(ch: { sort_order: number; title: string }): string {
  const pureTitle = stripChapterPrefix(ch.title)
  return `第${ch.sort_order}章 ${pureTitle}`
}

function chapterStatusType(s: string) {
  const map: Record<string, string> = { draft: 'info', polished: 'warning', final: 'success' }
  return map[s] || 'info'
}

function statusLabel(s: string) {
  const map: Record<string, string> = { draft: '草稿', polished: '已润色', final: '定稿' }
  return map[s] || s
}

function sourceTagType(s: string) {
  if (s.startsWith('ai_')) return 'warning'
  return 'info'
}

function formatDate(d: string) {
  return new Date(d).toLocaleString()
}

// ========== Prompt 模板管理 ==========

const showTemplateDialog = ref(false)
const activeTemplateTab = ref('summary_polish')
const templateEdits = reactive<Record<string, string>>({})
const templateSaving = ref(false)
const templatePreviewResult = ref('')
const loadedTemplates = ref<PromptTemplate[]>([])

const templateVars = [
  '{{.NovelTitle}}', '{{.NovelDescription}}',
  '{{.ChapterTitle}}', '{{.ChapterSummary}}', '{{.ChapterContent}}',
  '{{.PrevSummaries}}', '{{.PrevContent}}',
  '{{.WordCount}}', '{{.TargetWords}}', '{{.SelectedText}}',
]

// 打开对话框时加载模板
watch(showTemplateDialog, async (visible) => {
  if (visible) {
    templatePreviewResult.value = ''
    try {
      const res = await promptTemplateApi.list(Number(props.nid))
      const templates: PromptTemplate[] = (res as any)?.data || res || []
      loadedTemplates.value = templates
      // 填充编辑区：只填充用户自定义的补充内容（novel_id > 0），默认模板不填入编辑框
      for (const tpl of templates) {
        const key = tpl.action + ':' + tpl.prompt_type
        if (tpl.novel_id > 0) {
          templateEdits[key] = tpl.content
        } else {
          // 默认模板：编辑框留空，表示无自定义补充
          if (!templateEdits[key]) {
            templateEdits[key] = ''
          }
        }
      }
    } catch {
      ElMessage.error('加载模板失败')
    }
  }
})

async function handleSaveTemplates() {
  templateSaving.value = true
  try {
    for (const action of aiActions) {
      for (const ptype of ['system', 'user'] as const) {
        const key = action.value + ':' + ptype
        const content = templateEdits[key]?.trim()
        if (!content) continue // 空内容不保存，使用纯默认模板
        // 查找原始模板获取 name
        const orig = loadedTemplates.value.find(t => t.action === action.value && t.prompt_type === ptype)
        const name = orig?.name || `${action.label}-${ptype}`
        await promptTemplateApi.upsert(Number(props.nid), {
          action: action.value,
          prompt_type: ptype,
          name,
          content,
        })
      }
    }
    ElMessage.success('模板已保存')
  } catch {
    ElMessage.error('保存模板失败')
  } finally {
    templateSaving.value = false
  }
}

async function handlePreviewTemplate() {
  const key = activeTemplateTab.value + ':user'
  const content = templateEdits[key]
  if (!content) {
    ElMessage.warning('没有可预览的用户提示词内容')
    return
  }
  try {
    const res = await promptTemplateApi.preview(Number(props.nid), {
      content,
      data: {
        NovelTitle: novelStore.currentNovel?.title || '',
        NovelDescription: novelStore.currentNovel?.description || '',
        ChapterTitle: novelStore.currentChapter?.title || '',
        ChapterSummary: editSummary.value,
        ChapterContent: editContent.value,
        PrevSummaries: '(preview data)',
        PrevContent: '(preview data)',
        WordCount: wordCount.value,
        TargetWords: wordCount.value * 2 || 1000,
      },
    })
    templatePreviewResult.value = res.data?.data?.rendered || ''
  } catch {
    ElMessage.error('预览失败')
  }
}

async function handleResetTemplate() {
  try {
    await ElMessageBox.confirm('将当前标签页的模板恢复为默认？', '恢复默认', { type: 'warning' })
    // 删除当前 action 的自定义模板
    const action = activeTemplateTab.value
    for (const tpl of loadedTemplates.value) {
      if (tpl.action === action && !tpl.is_default && tpl.novel_id > 0) {
        await promptTemplateApi.delete(Number(props.nid), tpl.id)
      }
    }
    // 重新加载
    const res = await promptTemplateApi.list(Number(props.nid))
    const templates: PromptTemplate[] = res.data?.data || []
    loadedTemplates.value = templates
    for (const tpl of templates) {
      templateEdits[tpl.action + ':' + tpl.prompt_type] = tpl.content
    }
    ElMessage.success('已恢复默认')
  } catch { /* cancelled */ }
}
</script>

<style scoped lang="scss">
.novel-workshop {
  height: 100%; display: flex; flex-direction: column; overflow: hidden;
}

.workshop-header {
  padding: 10px 20px;
  display: flex; align-items: center; gap: 12px;
  background: var(--color-bg-card);
  border-radius: 12px;
  border: 1px solid var(--border-glow);
  box-shadow: var(--shadow-sm);
  margin-bottom: 1px;
  transition: box-shadow 0.3s ease;

  &:hover {
    box-shadow: var(--shadow-md);
  }
}

.token-indicator {
  margin-left: auto;
  display: flex; align-items: center; gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background 0.2s;
  &:hover { background: var(--el-fill-color-light); }

  &__text {
    font-size: 12px; color: var(--el-text-color-secondary); white-space: nowrap;
    &--link { color: var(--el-color-primary); }
  }
}

.workshop-body {
  flex: 1; display: flex; overflow: hidden;
}

// 左栏：章节列表
.chapter-list {
  width: 220px; border-right: 1px solid var(--border-glow);
  display: flex; flex-direction: column; flex-shrink: 0;
  background: var(--color-bg-surface);

  &__header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 12px 16px; border-bottom: 1px solid var(--border-glow);
    h3 { font-size: 13px; font-weight: 600; color: var(--color-text-primary); letter-spacing: 0.03em; }
  }

  &__items {
    flex: 1; overflow-y: auto; padding: 8px;
    &::-webkit-scrollbar { width: 4px; }
    &::-webkit-scrollbar-track { background: transparent; }
    &::-webkit-scrollbar-thumb { background: rgba(124, 140, 248, 0.1); border-radius: 2px; }
  }
  &__empty { padding: 20px; text-align: center; font-size: 13px; color: var(--color-text-muted); }
  &__footer { padding: 8px 16px; border-top: 1px solid var(--border-glow); text-align: center; }
}

.chapter-item {
  display: flex; align-items: center; gap: 4px;
  padding: 8px 8px; border-radius: 6px; cursor: pointer;
  font-size: 13px; color: var(--color-text-secondary);
  transition: all 0.15s ease;

  &:hover { background-color: var(--color-bg-hover); }
  &--active {
    background-color: var(--color-bg-hover); color: var(--color-primary);
    font-weight: 500;
  }
  &--has-content {
    background-color: rgba(103, 194, 58, 0.08);
    &:hover { background-color: rgba(103, 194, 58, 0.15); }
    &.chapter-item--active { background-color: rgba(103, 194, 58, 0.15); color: var(--color-primary); }
  }
  &__order { font-size: 12px; color: var(--color-text-muted); min-width: 20px; }
  &__title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &__status { font-size: 12px; }
  &__ai-indicator { font-size: 14px; color: var(--color-primary); margin-left: 2px; }
  &__expand-btn {
    display: none; width: 20px; height: 20px; border-radius: 50%;
    font-size: 14px; font-weight: 700; line-height: 20px; text-align: center;
    color: var(--color-primary); background: rgba(124, 140, 248, 0.1);
    cursor: pointer; margin-left: auto; flex-shrink: 0;
    transition: all 0.15s ease;
    &:hover { color: #fff; background: var(--color-primary); }
  }
  &:hover &__expand-btn { display: block; }
}

// 中栏：编辑区
.editor-area {
  flex: 1; overflow-y: auto; padding: 24px 28px; display: flex; flex-direction: column; gap: 16px;
  background: var(--color-bg-deep);

  // 自定义滚动条
  &::-webkit-scrollbar { width: 6px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb {
    background: rgba(124, 140, 248, 0.1);
    border-radius: 3px;
  }
  &::-webkit-scrollbar-thumb:hover {
    background: rgba(124, 140, 248, 0.2);
  }

  &__header {
    display: flex; align-items: center; gap: 12px;
  }
  &__title-input { flex: 1; :deep(.el-input__inner) { font-size: 20px; font-weight: 600; } }
  &__actions { display: flex; gap: 12px; align-items: center; }
  &__empty {
    flex: 1; display: flex; align-items: center; justify-content: center;
    color: var(--color-text-muted); font-size: 14px;
  }
}

.editor-section {
  &__label {
    display: block; font-size: 11px; font-weight: 600;
    color: var(--color-text-muted); margin-bottom: 6px;
    text-transform: uppercase; letter-spacing: 0.08em;
  }
  &__toolbar {
    display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px;
  }
  &__toolbar &__label { margin-bottom: 0; }
  &__controls {
    display: flex; align-items: center; gap: 10px;
  }
  &--content { flex: 1; display: flex; flex-direction: column;
    :deep(.el-textarea) { flex: 1; display: flex; flex-direction: column; }
    :deep(.el-textarea__inner) {
      flex: 1;
      font-size: var(--content-font-size, 15px);
      font-family: var(--content-font-family, inherit);
      line-height: 2;
      color: var(--color-text-primary);
      background: var(--color-bg-editor);
      border: 1px solid var(--border-glow);
      border-radius: 10px;
      padding: 28px 36px;
      caret-color: var(--color-primary);
    }
  }
  &__meta {
    font-size: 11px; color: var(--color-text-muted); margin-top: 6px; text-align: right;
    letter-spacing: 0.03em;
  }
}

.font-select {
  width: 110px;
  :deep(.el-input__inner) { font-size: 12px; }
  :deep(.el-input__wrapper) {
    background-color: var(--color-bg-card) !important;
    border-radius: 6px;
  }
}
.font-size-ctrl {
  display: flex; align-items: center; gap: 2px;
  background: var(--color-bg-card); border: 1px solid var(--border-glow);
  border-radius: 8px; padding: 2px 6px;
}
.font-size-btn {
  width: 26px; height: 24px; border: none; border-radius: 5px;
  background: transparent; color: var(--color-text-secondary);
  font-size: 12px; font-weight: 600; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all 0.15s;
  &:hover { background: var(--color-bg-hover); color: var(--color-primary-light); }
}
.font-size-val {
  font-size: 11px; color: var(--color-text-muted); min-width: 34px; text-align: center;
  font-variant-numeric: tabular-nums;
}

// 概要润色 loading 分栏
.summary-diff-loading {
  border: 1px solid var(--border-glow, rgba(124, 140, 248, 0.12));
  border-radius: 10px;
  overflow: hidden;

  &__headers {
    display: flex;
    border-bottom: 1px solid var(--border-glow, rgba(124, 140, 248, 0.08));
  }

  &__header-cell {
    flex: 1;
    min-width: 0;
    padding: 4px 12px;
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-muted);
    background: var(--color-bg-surface);
    display: flex;
    align-items: center;
  }

  &__header-divider {
    width: 1px;
    background: var(--border-glow, rgba(124, 140, 248, 0.12));
    flex-shrink: 0;
  }

  &__panels {
    display: flex;
  }

  &__panel {
    flex: 1;
    min-width: 0;
  }

  &__divider {
    width: 1px;
    background: var(--border-glow, rgba(124, 140, 248, 0.12));
    flex-shrink: 0;
  }

  &__body {
    padding: 8px 12px;
    font-size: 14px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    white-space: pre-wrap;
    min-height: 60px;
  }
}

// 正文区 AI loading 分栏（与 ContentDiffView 对齐）
.content-diff-loading {
  border: 1px solid var(--border-glow, rgba(124, 140, 248, 0.12));
  border-radius: 10px;
  overflow: hidden;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;

  &__headers {
    display: flex;
    flex-shrink: 0;
    border-bottom: 1px solid var(--border-glow, rgba(124, 140, 248, 0.08));
  }

  &__header-cell {
    flex: 1;
    min-width: 0;
    padding: 8px 16px;
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-muted, #999);
    background: var(--color-bg-surface, #fafafa);
    display: flex;
    align-items: center;
  }

  &__header-divider {
    width: 1px;
    background: var(--border-glow, rgba(124, 140, 248, 0.12));
    flex-shrink: 0;
  }

  &__panels {
    display: flex;
    flex: 1;
    min-height: 0;
  }

  &__panel {
    flex: 1;
    min-width: 0;
    overflow-y: auto;
  }

  &__divider {
    width: 1px;
    background: var(--border-glow, rgba(124, 140, 248, 0.12));
    flex-shrink: 0;
  }

  &__body {
    padding: 10px 14px;
    font-size: 14px;
    color: var(--color-text-secondary);
    line-height: 1.8;
    white-space: pre-wrap;
  }
}

// 右栏：AI 操作 + 版本历史
.side-panel {
  width: 240px; min-width: 240px; max-width: 240px;
  border-left: 1px solid var(--border-glow);
  overflow-y: auto; overflow-x: hidden; flex-shrink: 0;
  background: var(--color-bg-surface);
  scrollbar-gutter: stable;

  &::-webkit-scrollbar { width: 4px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb { background: rgba(124, 140, 248, 0.1); border-radius: 2px; }
}

.side-section {
  padding: 12px 14px; border-bottom: 1px solid var(--border-glow);

  &__header {
    display: flex; justify-content: space-between; align-items: center;
    cursor: pointer; user-select: none;
    padding: 4px 0;
    border-radius: 4px;
    transition: opacity 0.15s ease;

    &:hover { opacity: 0.8; }
    &:active { opacity: 0.6; }
  }

  &__title {
    font-size: 12px; font-weight: 600; color: var(--color-text-secondary);
    margin: 0; letter-spacing: 0.06em; text-transform: uppercase;
  }

  &__arrow {
    font-size: 14px; color: var(--color-text-muted);
    transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);

    &.is-collapsed { transform: rotate(-90deg); }
  }

  &__body {
    margin-top: 10px;
    overflow: hidden;
  }
}

.ai-selection-hint {
  margin-top: 12px;
  margin-bottom: 2px;
}

// AI 操作 2×2 网格
.ai-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-glow);

  &__item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 10px 4px;
    border-radius: 8px;
    border: 1px solid var(--border-glow);
    background: var(--color-bg-surface);
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: all 0.15s ease;

    &:hover {
      background: var(--color-bg-hover);
      color: var(--color-text-primary);
      border-color: rgba(124, 140, 248, 0.4);
    }

    &:active {
      transform: scale(0.96);
    }

    &:disabled {
      opacity: 0.4;
      cursor: not-allowed;
      &:hover { background: var(--color-bg-surface); color: var(--color-text-secondary); border-color: var(--border-glow); }
      &:active { transform: none; }
    }
  }

  &__icon {
    font-size: 18px;
    color: var(--color-primary-light);
  }

  &__label {
    font-size: 12px;
    line-height: 1;
  }

  // 润色下拉菜单样式适配网格
  &__dropdown {
    display: flex;
    flex-direction: column;
    align-items: center;
    overflow: hidden;
    max-width: 100%;
    height: 100%;

    :deep(.el-button-group) {
      display: flex;
      width: 100%;
      height: 100%;

      .el-button {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 4px;
        padding: 10px 2px;
        border-radius: 8px 0 0 8px;
        border: 1px solid var(--border-glow);
        border-right: none;
        background: var(--color-bg-surface);
        color: var(--color-text-secondary);
        flex: 1;
        min-width: 0;
        height: 100%;
        font-size: 12px;

        &:hover {
          background: var(--color-bg-hover);
          color: var(--color-text-primary);
          border-color: rgba(124, 140, 248, 0.4);
        }

        // 下拉箭头按钮
        &:last-child {
          flex: 0 0 22px;
          border-radius: 0 8px 8px 0;
          border: 1px solid var(--border-glow);
          border-left: 1px solid var(--border-glow);
          padding: 0;
        }
      }
    }
  }
}

.workflow-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 12px;
  border-radius: 8px;
  background: linear-gradient(135deg, rgba(64, 158, 255, 0.08), rgba(103, 194, 58, 0.08));
  border: 1px solid rgba(64, 158, 255, 0.2);
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: rgba(64, 158, 255, 0.4);
    background: linear-gradient(135deg, rgba(64, 158, 255, 0.12), rgba(103, 194, 58, 0.12));
  }

  &__text {
    flex: 1;
    font-size: 13px;
    color: var(--color-text-secondary);
  }
}

.workflow-btns {
  display: flex;
  gap: 6px;
  width: 100%;
}

.workflow-btn {
  flex: 1;
  min-width: 0 !important;
  max-width: 50%;
  font-size: 12px !important;
  padding: 8px 4px !important;
  justify-content: center;
  border-radius: 8px !important;
  box-sizing: border-box;
  overflow: hidden;
  // 固定背景色，防止 text bg 模式在暗色主题下样式丢失
  &.el-button--warning {
    --el-button-text-color: var(--el-color-warning);
    --el-button-bg-color: var(--el-color-warning-light-9);
    --el-button-hover-bg-color: var(--el-color-warning-light-7);
    background-color: var(--el-button-bg-color);
    color: var(--el-button-text-color);
  }
}

.workflow-nodes {
  margin-top: 8px; display: flex; flex-direction: column; gap: 4px;
}

.workflow-node {
  display: flex; align-items: center; justify-content: space-between;
  padding: 4px 8px; border-radius: 4px; font-size: 12px;
  background-color: var(--color-bg-surface);

  &__id { color: var(--color-text-secondary); }
  &--completed { opacity: 0.7; }
  &--failed { background-color: rgba(245, 108, 108, 0.1); }
}

// AI 配置区
.ai-config {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-glow);

  &__row {
    display: flex; gap: 6px; align-items: center;
    .el-select { flex: 1; }
  }

  &__setting {
    flex-shrink: 0;
    padding: 6px !important;
    color: var(--color-text-muted);
    &:hover { color: var(--color-primary-light); }
  }
}

.version-list {
  display: flex; flex-direction: column; gap: 4px;
  &__empty { font-size: 13px; color: var(--color-text-muted); text-align: center; padding: 12px; }
}

.version-item {
  padding: 8px 10px; border-radius: 6px; cursor: pointer;
  transition: background-color 0.15s ease;
  &:hover { background-color: var(--color-bg-hover); }

  &__header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
  &__num { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
  &__meta { font-size: 11px; color: var(--color-text-muted); }
}

// Prompt 模板对话框
.template-dialog {
  &__vars {
    margin-bottom: 12px; display: flex; flex-wrap: wrap; align-items: center; gap: 4px;
    &-label { font-size: 12px; color: var(--color-text-muted); margin-right: 4px; }
  }
}

.template-editor {
  display: flex; flex-direction: column; gap: 16px;

  &__section {
    label {
      display: block; font-size: 12px; font-weight: 500;
      color: var(--color-text-muted); margin-bottom: 6px; text-transform: uppercase;
    }
  }

  &__preview {
    padding: 12px; border-radius: 6px; font-size: 13px;
    background-color: var(--color-bg-surface); border: 1px solid var(--border-glow);
    white-space: pre-wrap; max-height: 200px; overflow-y: auto;
    color: var(--color-text-secondary);
  }
}

// 批量生成面板
.batch-panel {
  display: flex; flex-direction: column; height: 100%; gap: 16px;

  &__summary {
    padding-bottom: 12px; border-bottom: 1px solid var(--border-glow);
  }

  &__stats {
    display: flex; justify-content: space-between; margin-top: 8px;
    font-size: 13px; color: var(--color-text-secondary);
  }

  &__list {
    flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 6px;
  }

  &__footer {
    padding-top: 12px; border-top: 1px solid var(--border-glow);
    display: flex; justify-content: center;
  }
}

.batch-task-item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px; border-radius: 6px;
  background-color: var(--color-bg-surface);
  border: 1px solid var(--border-glow);
  flex-wrap: wrap;

  &__order {
    font-size: 12px; color: var(--color-text-muted); min-width: 24px;
  }

  &__title {
    flex: 1; font-size: 13px; color: var(--color-text-primary);
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }

  &__right {
    flex-shrink: 0;
  }

  &__error {
    width: 100%; font-size: 11px; color: var(--el-color-danger);
    margin-top: 2px;
  }

  &--running {
    border-color: rgba(230, 162, 60, 0.4);
    background: rgba(230, 162, 60, 0.05);
  }

  &--completed {
    opacity: 0.7;
  }

  &--failed {
    border-color: rgba(245, 108, 108, 0.3);
    background: rgba(245, 108, 108, 0.05);
  }

  &--cancelled {
    opacity: 0.5;
  }
}
</style>
