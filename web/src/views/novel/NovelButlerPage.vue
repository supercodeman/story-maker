<!-- web/src/views/novel/NovelButlerPage.vue -->
<!-- 小说管家：列表页 + 7 步 AI 创作向导 -->
<template>
  <div class="novel-butler-page">
    <!-- 顶部导航 -->
    <div class="butler-topbar">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">工作空间</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">作品集</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}/novels` }">小说</el-breadcrumb-item>
        <el-breadcrumb-item>小说管家</el-breadcrumb-item>
      </el-breadcrumb>
      <div class="butler-topbar__right">
        <template v-if="mode === 'create'">
          <span class="auto-label">跳过确认</span>
          <el-switch v-model="state.autoMode" size="small" />
        </template>
        <ModelSelector v-model="selectedModel" capability="text_gen" size="small" style="min-width: 120px;" />
      </div>
    </div>

    <!-- ========== 列表模式 ========== -->
    <template v-if="mode === 'list'">
      <div class="butler-list">
        <div class="butler-list__header">
          <div class="butler-list__title">管家创作的小说</div>
          <div style="display: flex; gap: 8px;">
            <button class="action-btn action-btn--ghost" @click="repairButlerData" :disabled="repairing">
              <span>{{ repairing ? '修复中...' : '修复历史数据' }}</span>
            </button>
            <button class="action-btn action-btn--accent" @click="enterCreateMode">
              <span>+ 新建创作</span>
            </button>
          </div>
        </div>

        <div v-if="listLoading" class="butler-list__loading">
          <el-skeleton :rows="3" animated />
        </div>

        <div v-else-if="butlerNovels.length === 0" class="butler-list__empty">
          <div class="empty-icon">✦</div>
          <div class="empty-text">还没有管家创作的小说</div>
          <div class="empty-sub">点击"新建创作"开始你的第一次 AI 辅助创作</div>
          <button class="action-btn action-btn--primary" style="margin-top: 20px;" @click="enterCreateMode">
            开始创作
          </button>
        </div>

        <div v-else class="butler-novel-grid">
          <div
            v-for="novel in butlerNovels"
            :key="novel.id"
            class="butler-novel-card"
            @click="handleResumeNovel(novel)"
          >
            <div class="butler-novel-card__title">{{ novel.title }}</div>
            <div class="butler-novel-card__desc">{{ novel.description || '暂无描述' }}</div>
            <div class="butler-novel-card__meta">
              <span>{{ novel.chapter_count }} 章</span>
              <span>{{ novel.word_count }} 字</span>
              <span>{{ formatDate(novel.created_at) }}</span>
            </div>
            <div class="butler-novel-card__status">
              <span class="status-dot" :class="`status-dot--${novel.status}`"></span>
              {{ novelStatusText(novel.status) }}
            </div>
            <div v-if="restoringNovelId === novel.id" class="butler-novel-card__restoring">
              恢复中...
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- ========== 创作模式（原有流程） ========== -->
    <template v-else>

    <!-- 自定义步骤条 -->
    <div class="butler-steps">
      <div class="steps-track">
        <div
          v-for="(step, idx) in STEPS"
          :key="step"
          class="step-node"
          :class="{
            'step-node--done': idx < currentStepIndex,
            'step-node--active': idx === currentStepIndex && started,
            'step-node--pending': idx > currentStepIndex || !started,
            'step-node--clickable': idx < currentStepIndex,
          }"
          @click="idx < currentStepIndex && handleJumpToStep(step)"
        >
          <div class="step-node__dot">
            <svg v-if="idx < currentStepIndex" viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
              <path d="M13.78 4.22a.75.75 0 010 1.06l-7.25 7.25a.75.75 0 01-1.06 0L2.22 9.28a.75.75 0 011.06-1.06L6 10.94l6.72-6.72a.75.75 0 011.06 0z"/>
            </svg>
            <span v-else>{{ idx + 1 }}</span>
          </div>
          <span class="step-node__label">{{ STEP_LABELS[step] }}</span>
          <div v-if="idx < STEPS.length - 1" class="step-node__line" :class="{ 'step-node__line--done': idx < currentStepIndex }" />
        </div>
      </div>
    </div>

    <!-- 启动面板 -->
    <div v-if="!started" class="butler-start">
      <div class="start-glow"></div>
      <div class="start-icon">✦</div>
      <div class="start-title">开启你的创作之旅</div>
      <p class="start-desc">小说管家将自动编排 选题 → 故事线 → 人物设计 → 章节生成 → 知识图谱 → 内容填充 的完整创作流程</p>
      <div class="start-input-wrap">
        <el-input
          v-model="settingInput"
          type="textarea"
          :rows="4"
          placeholder="描述你想创作的故事方向，例如：都市修仙、甜宠校园、末日废土..."
          maxlength="2000"
          show-word-limit
        />
      </div>
      <button class="start-btn" :disabled="!settingInput.trim()" @click="handleStart">
        <span class="start-btn__text">开始创作</span>
        <span class="start-btn__arrow">→</span>
      </button>
    </div>

    <!-- 主体区域 -->
    <div v-else class="butler-body">
      <!-- 步骤 1：选题（保留4选1模式） -->
      <template v-if="state.currentStep === 'topic'">
        <div class="step-section">
          <div class="step-header">
            <div class="step-header__badge">{{ currentStepIndex + 1 }}</div>
            <div class="step-header__title">{{ currentStepLabel }}</div>
            <div class="step-header__status" :class="`step-header__status--${currentStepState.status}`">{{ stepStatusText }}</div>
          </div>

          <!-- idle：等待用户输入想法 -->
          <div v-if="currentStepState.status === 'idle' && !currentStepState.error" class="step-hint-box">
            <p class="step-hint-box__desc">{{ stepHintPlaceholder }}</p>
            <el-input
              v-model="currentStepState.userHint"
              type="textarea"
              :rows="3"
              :placeholder="stepHintPlaceholder"
              maxlength="1000"
              show-word-limit
            />
            <button class="action-btn action-btn--primary" style="margin-top: 14px;" @click="submitStepHint(state.currentStep)">
              开始生成
            </button>
          </div>

          <!-- generating -->
          <div v-if="currentStepState.status === 'generating'" class="options-grid">
            <div v-for="i in 4" :key="i" class="option-card option-card--skeleton">
              <el-skeleton :rows="5" animated />
            </div>
          </div>

          <!-- choosing -->
          <div v-else-if="currentStepState.status === 'choosing'" class="options-grid">
            <div
              v-for="(opt, idx) in currentStepState.options"
              :key="idx"
              class="option-card"
              :class="{ 'option-card--disabled': opt === '（生成失败）' }"
            >
              <div class="option-card__index">方案 {{ idx + 1 }}</div>
              <div class="option-content">{{ opt }}</div>
              <div class="option-footer">
                <button
                  class="action-btn action-btn--accent"
                  :disabled="opt === '（生成失败）'"
                  @click="confirmStep(state.currentStep, idx)"
                >
                  选择此方案
                </button>
              </div>
            </div>
          </div>

          <!-- confirmed -->
          <div v-else-if="currentStepState.status === 'confirmed'" class="confirmed-summary">
            <div class="confirmed-badge">✓ 已确认</div>
            <div class="confirmed-text">{{ currentStepState.result.slice(0, 200) }}{{ currentStepState.result.length > 200 ? '...' : '' }}</div>
            <div v-if="currentStepState.tokenUsage" class="token-usage">
              输入 {{ currentStepState.tokenUsage.prompt_tokens.toLocaleString() }} · 输出 {{ currentStepState.tokenUsage.completion_tokens.toLocaleString() }} · 合计 {{ currentStepState.tokenUsage.total_tokens.toLocaleString() }} tokens
            </div>
            <!-- 对话区域（非 autoMode 时显示） -->
            <div v-if="!state.autoMode && currentStepState.messages.length > 0" class="step-chat">
              <div class="step-chat__history">
                <div v-for="(msg, i) in currentStepState.messages" :key="i"
                     class="step-chat__bubble" :class="`step-chat__bubble--${msg.role}`">
                  {{ msg.content }}
                </div>
              </div>
              <div class="step-chat__input">
                <el-input v-model="currentStepState.refineInput" type="textarea" :rows="2"
                          placeholder="输入调整要求，如：换个更悬疑的方向..." />
                <button class="action-btn action-btn--primary" @click="refineStep('topic')"
                        :disabled="!currentStepState.refineInput?.trim() || currentStepState.status === 'generating'">
                  调整
                </button>
              </div>
            </div>
            <div v-if="!state.autoMode" class="step-chat__confirm">
              <button class="action-btn action-btn--primary" @click="confirmStepAndNext('topic')">
                确认，继续下一步
              </button>
            </div>
          </div>

          <!-- error -->
          <div v-if="currentStepState.error" class="step-error">
            <el-alert :title="currentStepState.error" type="error" show-icon :closable="false" />
            <button class="action-btn action-btn--primary" style="margin-top: 10px;" @click="retryCurrentStep">
              重试
            </button>
          </div>
        </div>
      </template>

      <!-- 步骤 2/3：故事线/人物设计（多轮迭代模式） -->
      <template v-else-if="state.currentStep === 'storyline' || state.currentStep === 'characters'">
        <div class="step-section">
          <div class="step-header">
            <div class="step-header__badge">{{ currentStepIndex + 1 }}</div>
            <div class="step-header__title">{{ currentStepLabel }}</div>
            <div class="step-header__status" :class="`step-header__status--${currentStepState.status}`">{{ stepStatusText }}</div>
          </div>

          <!-- idle：等待用户输入想法 -->
          <div v-if="currentStepState.status === 'idle' && !currentStepState.error" class="step-hint-box">
            <p class="step-hint-box__desc">{{ stepHintPlaceholder }}</p>
            <el-input
              v-model="currentStepState.userHint"
              type="textarea"
              :rows="3"
              :placeholder="stepHintPlaceholder"
              maxlength="1000"
              show-word-limit
            />
            <!-- 故事线步骤：可选开关 -->
            <div v-if="state.currentStep === 'storyline'" class="step-hint-box__options">
              <el-checkbox v-model="state.enableBeats">段落细化（中长篇推荐，>30章时生效）</el-checkbox>
              <el-checkbox v-model="state.enableSubplots">支线交织（增加1-2条支线）</el-checkbox>
            </div>
            <button class="action-btn action-btn--primary" style="margin-top: 14px;" @click="submitStepHint(state.currentStep)">
              开始生成
            </button>
          </div>

          <!-- generating/reviewing：迭代进度展示 -->
          <div v-if="currentStepState.status === 'generating'" class="iteration-progress">
            <div class="iteration-progress__icon">
              <el-icon class="is-loading" :size="32"><Loading /></el-icon>
            </div>
            <p class="iteration-progress__title">
              AI 正在{{ state.currentStep === 'storyline' ? '创作故事线' : '设计人物' }}...
            </p>
            <p class="iteration-progress__detail">
              {{ iterationPhaseText }}
              <template v-if="currentStepState.iterationRound && currentStepState.iterationRound > 0">
                · 第 {{ currentStepState.iterationRound }} / {{ currentStepState.iterationMaxRounds || 5 }} 轮
              </template>
            </p>
            <el-progress
              :percentage="iterationProgress"
              :stroke-width="6"
              :show-text="false"
              style="margin-top: 16px; max-width: 400px;"
            />
          </div>

          <!-- confirmed：最终结果展示 + 编辑 -->
          <div v-else-if="currentStepState.status === 'confirmed'" class="iteration-result">
            <!-- 故事线：四幕卡片展示 -->
            <div v-if="state.currentStep === 'storyline' && storyStructure" class="story-acts">
              <div class="story-acts__synopsis">{{ storyStructure.synopsis }}</div>
              <div class="story-acts__grid">
                <div v-for="(act, idx) in storyStructure.acts" :key="idx" class="act-card">
                  <div class="act-card__header">
                    <span class="act-card__name">第{{ ['一', '二', '三', '四'][idx] }}幕·{{ act.name }}</span>
                    <span class="act-card__ratio">{{ act.ratio }}%</span>
                  </div>
                  <div class="act-card__mission">{{ act.core_mission }}</div>
                  <div class="act-card__events">
                    <div v-for="(evt, ei) in act.key_events" :key="ei" class="act-card__event">{{ ei + 1 }}. {{ evt }}</div>
                  </div>
                  <div class="act-card__footer">
                    <div class="act-card__turning"><span class="act-card__label">转折点</span>{{ act.turning_point }}</div>
                    <div class="act-card__hook"><span class="act-card__label">钩子</span>{{ act.hook }}</div>
                  </div>
                  <!-- 段落细化 -->
                  <div v-if="act.beats && act.beats.length > 0" class="act-card__beats">
                    <div class="act-card__beats-title">段落细化</div>
                    <div v-for="(beat, bi) in act.beats" :key="bi" class="beat-item">
                      <span class="beat-item__name">{{ beat.name }}</span>
                      <span class="beat-item__info">{{ beat.goal }} · 约{{ beat.chapter_count }}章</span>
                    </div>
                  </div>
                </div>
              </div>
              <!-- 支线展示 -->
              <div v-if="storyStructure.subplots && storyStructure.subplots.length > 0" class="story-subplots">
                <div class="story-subplots__title">支线交织</div>
                <div v-for="(sp, si) in storyStructure.subplots" :key="si" class="subplot-item">
                  <span class="subplot-item__name">{{ sp.name }}</span>
                  <span class="subplot-item__desc">{{ sp.description }}</span>
                  <span class="subplot-item__acts">交汇幕：{{ sp.intersect_acts?.join(', ') }}</span>
                </div>
              </div>
            </div>

            <!-- 人物设计：人物卡片网格 -->
            <div v-if="state.currentStep === 'characters' && characterCards.length > 0" class="character-cards">
              <div class="character-cards__title">人物群像（{{ characterCards.length }} 人）</div>
              <div class="character-cards__grid">
                <div v-for="(ch, ci) in characterCards" :key="ci" class="char-card" :class="`char-card--${ch.role_type === '主角' ? 'lead' : ch.role_type === '核心配角' ? 'core' : 'support'}`">
                  <div class="char-card__header">
                    <span class="char-card__name">{{ ch.name }}</span>
                    <el-tag :type="ch.role_type === '主角' ? 'danger' : ch.role_type === '核心配角' ? 'warning' : 'info'" size="small">{{ ch.role_type }}</el-tag>
                  </div>
                  <div class="char-card__identity">{{ ch.identity }}</div>
                  <div class="char-card__row"><span class="char-card__label">外在</span>{{ ch.appearance }}</div>
                  <div class="char-card__row"><span class="char-card__label">表层</span>{{ ch.personality_surface }}</div>
                  <div class="char-card__row"><span class="char-card__label">内核</span>{{ ch.personality_deep }}</div>
                  <div class="char-card__row"><span class="char-card__label">语言</span>{{ ch.speech_style }}</div>
                  <div class="char-card__row"><span class="char-card__label">动机</span>{{ ch.motivation }}</div>
                  <div class="char-card__row"><span class="char-card__label">弧光</span>{{ ch.arc }}</div>
                </div>
              </div>
            </div>

            <!-- 原始文本：折叠展示 -->
            <details v-if="state.currentStep === 'characters'" class="raw-text-details">
              <summary>查看/编辑原始文本</summary>
              <el-input
                v-model="currentStepState.result"
                type="textarea"
                :autosize="{ minRows: 10, maxRows: 30 }"
                placeholder="AI 生成的内容，你可以直接编辑修改"
              />
            </details>
            <el-input
              v-if="state.currentStep !== 'characters'"
              v-model="currentStepState.result"
              type="textarea"
              :autosize="{ minRows: 10, maxRows: 30 }"
              placeholder="AI 生成的内容，你可以直接编辑修改"
            />

            <!-- 人物设计：出场规划表格 -->
            <div v-if="state.currentStep === 'characters' && appearancePlan.length > 0" class="appearance-plan">
              <div class="appearance-plan__title">出场规划</div>
              <el-table :data="appearancePlan" border size="small" style="margin-top: 8px;">
                <el-table-column prop="character" label="人物" width="100" />
                <el-table-column label="首次出场" width="160">
                  <template #default="{ row }">
                    第{{ row.first_appear?.act }}幕 · {{ row.first_appear?.method }}
                  </template>
                </el-table-column>
                <el-table-column label="各幕角色" min-width="200">
                  <template #default="{ row }">
                    <span v-for="(role, key) in row.act_roles" :key="key" class="act-role-tag">
                      {{ key }}: {{ role }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column label="关键场景" min-width="200">
                  <template #default="{ row }">
                    <div v-for="(sc, si) in row.key_scenes" :key="si" class="key-scene-item">
                      第{{ sc.act }}幕: {{ sc.scene }}
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <!-- 人物设计：关系矩阵表格 -->
            <div v-if="state.currentStep === 'characters' && relationMatrix.length > 0" class="relation-matrix">
              <div class="relation-matrix__title">人物关系矩阵</div>
              <el-table :data="relationMatrix" border size="small" style="margin-top: 8px;">
                <el-table-column prop="from" label="人物A" width="100" />
                <el-table-column prop="to" label="人物B" width="100" />
                <el-table-column prop="relation" label="关系" width="120" />
                <el-table-column prop="detail" label="具体描述" />
                <el-table-column prop="tension" label="张力" width="80">
                  <template #default="{ row }">
                    <el-tag :type="row.tension === 'high' ? 'danger' : row.tension === 'medium' ? 'warning' : 'info'" size="small">
                      {{ row.tension }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <div v-if="currentStepState.tokenUsage" class="token-usage">
              输入 {{ currentStepState.tokenUsage.prompt_tokens.toLocaleString() }} · 输出 {{ currentStepState.tokenUsage.completion_tokens.toLocaleString() }} · 合计 {{ currentStepState.tokenUsage.total_tokens.toLocaleString() }} tokens
            </div>
            <!-- 对话区域（非 autoMode 时显示） -->
            <div v-if="!state.autoMode && currentStepState.messages.length > 0" class="step-chat">
              <div class="step-chat__history">
                <div v-for="(msg, i) in currentStepState.messages" :key="i"
                     class="step-chat__bubble" :class="`step-chat__bubble--${msg.role}`">
                  {{ msg.content }}
                </div>
              </div>
              <div class="step-chat__input">
                <el-input v-model="currentStepState.refineInput" type="textarea" :rows="2"
                          placeholder="输入调整要求，如：第二幕节奏加快..." />
                <button class="action-btn action-btn--primary" @click="refineStep(state.currentStep)"
                        :disabled="!currentStepState.refineInput?.trim() || currentStepState.status === 'generating'">
                  调整
                </button>
              </div>
            </div>
            <div class="iteration-result__actions">
              <button class="action-btn action-btn--ghost" @click="rerunIterativeStep">
                不满意，重新生成
              </button>
              <button class="action-btn action-btn--primary" @click="confirmIterativeAndNext">
                确认，继续下一步
              </button>
            </div>
          </div>

          <!-- error -->
          <div v-if="currentStepState.error" class="step-error">
            <el-alert :title="currentStepState.error" type="error" show-icon :closable="false" />
            <button class="action-btn action-btn--primary" style="margin-top: 10px;" @click="retryCurrentStep">
              重试
            </button>
          </div>
        </div>
      </template>

      <!-- 步骤 4：章节生成 -->
      <template v-else-if="state.currentStep === 'chapters'">
        <div class="step-section">
          <div class="step-header">
            <div class="step-header__badge">4</div>
            <div class="step-header__title">章节生成</div>
            <div class="step-header__status" :class="`step-header__status--${state.steps.chapters.status}`">{{ stepStatusText }}</div>
          </div>

          <!-- idle -->
          <div v-if="state.steps.chapters.status === 'idle' && !state.steps.chapters.error" class="step-hint-box">
            <p class="step-hint-box__desc">输入你对章节结构的想法，如节奏安排、高潮分布等（可留空直接生成）</p>
            <el-input
              v-model="state.steps.chapters.userHint"
              type="textarea"
              :rows="3"
              placeholder="例如：希望前5章节奏紧凑，中间有反转..."
              maxlength="1000"
              show-word-limit
            />
            <div class="step-hint-box__row">
              <label class="step-hint-box__label">章节数量</label>
              <el-input-number
                v-model="state.chapterNum"
                :min="20"
                :max="100"
                :step="5"
                size="default"
              />
              <span class="step-hint-box__tip">默认 56 章（推荐 50-62 章）</span>
            </div>
            <button class="action-btn action-btn--primary" style="margin-top: 16px;" @click="submitStepHint('chapters')">
              开始生成
            </button>
          </div>

          <div v-else-if="state.steps.chapters.status === 'generating'" class="chapter-skeleton">
            <el-skeleton :rows="8" animated />
          </div>

          <div v-else-if="state.steps.chapters.status === 'choosing'" class="chapter-preview">
            <div class="chapter-preview__header">
              <span class="chapter-preview__count">共 {{ state.outlineChapters.length }} 章</span>
              <span v-if="state.steps.chapters.tokenUsage" class="token-usage token-usage--inline">
                {{ state.steps.chapters.tokenUsage.total_tokens.toLocaleString() }} tokens
              </span>
              <button class="action-btn action-btn--primary" @click="confirmChapters">
                确认大纲，创建小说
              </button>
            </div>
            <div class="chapter-list">
              <div v-for="(ch, idx) in state.outlineChapters" :key="idx" class="chapter-card">
                <div class="chapter-card__num">{{ idx + 1 }}</div>
                <div class="chapter-card__body">
                  <div class="chapter-card__title">{{ ch.title }}</div>
                  <div class="chapter-card__summary">{{ ch.summary }}</div>
                </div>
              </div>
            </div>
            <div class="chapter-preview__footer">
              <button class="action-btn action-btn--ghost" @click="retryCurrentStep">重新生成</button>
              <button class="action-btn action-btn--primary" @click="confirmChapters">
                确认大纲，创建小说
              </button>
            </div>
          </div>

          <div v-else-if="state.steps.chapters.status === 'confirmed'" class="confirmed-summary">
            <div class="confirmed-badge">✓ 已确认</div>
            <div class="confirmed-text">{{ state.steps.chapters.result }}</div>
            <div v-if="state.steps.chapters.tokenUsage" class="token-usage">
              输入 {{ state.steps.chapters.tokenUsage.prompt_tokens.toLocaleString() }} · 输出 {{ state.steps.chapters.tokenUsage.completion_tokens.toLocaleString() }} · 合计 {{ state.steps.chapters.tokenUsage.total_tokens.toLocaleString() }} tokens
            </div>
          </div>

          <div v-if="state.steps.chapters.error" class="step-error">
            <el-alert :title="state.steps.chapters.error" type="error" show-icon :closable="false" />
            <button class="action-btn action-btn--primary" style="margin-top: 10px;" @click="retryCurrentStep">
              重试
            </button>
          </div>
        </div>
      </template>

      <!-- 步骤 5：开篇打磨 -->
      <template v-else-if="state.currentStep === 'opening_polish'">
        <div class="step-section">
          <div class="step-header">
            <div class="step-header__badge">5</div>
            <div class="step-header__title">开篇打磨</div>
            <div class="step-header__status" :class="`step-header__status--${state.steps.opening_polish.status}`">{{ stepStatusText }}</div>
          </div>

          <!-- generating：两阶段进度 -->
          <div v-if="state.steps.opening_polish.status === 'generating'" class="iteration-progress">
            <div class="iteration-progress__icon">
              <el-icon class="is-loading" :size="32"><Loading /></el-icon>
            </div>
            <p class="iteration-progress__title">
              {{ state.openingProgress.phase === 'polishing' ? '阶段1：前5章概要精细化' : '阶段2：前5章内容生成' }}
            </p>
            <p class="iteration-progress__detail">
              <template v-if="state.openingProgress.phase === 'polishing'">
                AI 正在精细化前5章概要...
                <template v-if="state.steps.opening_polish.iterationRound && state.steps.opening_polish.iterationRound > 0">
                  · 第 {{ state.steps.opening_polish.iterationRound }} / {{ state.steps.opening_polish.iterationMaxRounds || 5 }} 轮
                </template>
              </template>
              <template v-else>
                正在生成第 {{ state.openingProgress.current }} / {{ state.openingProgress.total }} 章
              </template>
            </p>
            <el-progress
              :percentage="state.openingProgress.phase === 'polishing'
                ? (state.steps.opening_polish.iterationRound || 0) * 10
                : Math.round(state.openingProgress.current / state.openingProgress.total * 100)"
              :stroke-width="6"
              :show-text="false"
              style="margin-top: 16px; max-width: 400px;"
            />
          </div>

          <!-- confirmed -->
          <div v-else-if="state.steps.opening_polish.status === 'confirmed'" class="confirmed-summary">
            <div class="confirmed-badge">✓ 已完成</div>
            <div class="confirmed-text">{{ state.steps.opening_polish.result }}</div>
            <div v-if="state.steps.opening_polish.tokenUsage" class="token-usage">
              输入 {{ state.steps.opening_polish.tokenUsage.prompt_tokens.toLocaleString() }} · 输出 {{ state.steps.opening_polish.tokenUsage.completion_tokens.toLocaleString() }} · 合计 {{ state.steps.opening_polish.tokenUsage.total_tokens.toLocaleString() }} tokens
            </div>
            <!-- 展示精细化概要卡片 -->
            <div v-if="state.openingChapters && state.openingChapters.length > 0" class="opening-chapters-preview">
              <div v-for="(ch, idx) in state.openingChapters" :key="idx" class="opening-chapter-card">
                <div class="opening-chapter-card__num">{{ ch.chapter_index || idx + 1 }}</div>
                <div class="opening-chapter-card__body">
                  <div class="opening-chapter-card__title">{{ ch.title }}</div>
                  <div v-if="ch.emotion_curve" class="opening-chapter-card__curve">节奏：{{ ch.emotion_curve }}</div>
                  <div v-if="ch.chapter_hook" class="opening-chapter-card__hook">钩子：{{ ch.chapter_hook }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- error -->
          <div v-if="state.steps.opening_polish.error" class="step-error">
            <el-alert :title="state.steps.opening_polish.error" type="error" show-icon :closable="false" />
            <button class="action-btn action-btn--primary" style="margin-top: 10px;" @click="retryCurrentStep">
              重试
            </button>
          </div>
        </div>
      </template>

      <!-- 步骤 6：知识图谱 -->
      <template v-else-if="state.currentStep === 'knowledge'">
        <div class="step-section">
          <div class="step-header">
            <div class="step-header__badge">6</div>
            <div class="step-header__title">知识图谱</div>
          </div>

          <div v-if="state.steps.knowledge.status === 'generating'" class="content-progress">
            <div class="progress-info">
              {{ state.knowledgeProgress.phase === 'extracting' ? '正在从章节概要中提取知识图谱...' : '正在解析并保存知识数据...' }}
            </div>
            <div class="progress-sub">自动提取人物档案、情节线、伏笔追踪、人物关系</div>
            <el-progress :percentage="state.knowledgeProgress.phase === 'parsing' ? 80 : 40" :stroke-width="8" :show-text="false" />
          </div>

          <div v-else-if="state.steps.knowledge.status === 'confirmed'" class="confirmed-summary">
            <div class="confirmed-badge">✓ 已完成</div>
            <div class="confirmed-text">{{ state.steps.knowledge.result }}</div>
          </div>
        </div>
      </template>

      <!-- 步骤 7：内容填充 -->
      <template v-else-if="state.currentStep === 'content'">
        <div class="step-section">
          <div class="step-header">
            <div class="step-header__badge">7</div>
            <div class="step-header__title">内容填充</div>
          </div>

          <!-- idle：等待用户选择 -->
          <div v-if="state.steps.content.status === 'idle'" class="content-progress">
            <div class="progress-info">小说和章节已创建完成，是否立即生成全部章节内容？</div>
            <div class="progress-sub">共 {{ state.outlineChapters.length }} 章，每章需要 1-2 分钟，可在小说工坊中单独生成</div>
            <div style="display: flex; gap: 12px; margin-top: 16px;">
              <button class="action-btn action-btn--accent" @click="submitStepHint('content')">开始生成</button>
              <button class="action-btn action-btn--ghost" @click="skipContent">跳过，稍后生成</button>
            </div>
          </div>

          <div v-else-if="state.steps.content.status === 'generating'" class="content-progress">
            <div class="progress-info">
              正在生成第 {{ state.contentProgress.completed + 1 }} 章（共 {{ state.contentProgress.total }} 章）：{{ state.contentProgress.current }}
            </div>
            <div class="progress-sub">已完成 {{ state.contentProgress.completed }} 章，每章需要 1-2 分钟</div>
            <el-progress
              :percentage="state.contentProgress.total > 0 ? Math.round(state.contentProgress.completed / state.contentProgress.total * 100) : 0"
              :stroke-width="8"
              :show-text="false"
            />
          </div>

          <div v-else-if="state.steps.content.status === 'confirmed'" class="confirmed-summary">
            <div class="confirmed-badge">✓ 创作完成</div>
            <div class="confirmed-text">{{ state.steps.content.result }}</div>
            <button
              v-if="state.createdNovelId"
              class="action-btn action-btn--primary"
              style="margin-top: 14px;"
              @click="goToNovel"
            >
              进入小说工坊 →
            </button>
          </div>
        </div>
      </template>

      <!-- 已完成步骤回顾 -->
      <div v-if="currentStepIndex > 0" class="completed-steps">
        <div class="completed-steps__title">已完成步骤</div>
        <div v-for="(step, idx) in STEPS.slice(0, currentStepIndex)" :key="step" class="completed-step-item">
          <span class="completed-step-num">{{ idx + 1 }}</span>
          <span class="completed-step-label">{{ STEP_LABELS[step] }}：</span>
          <span class="completed-step-text">{{ state.steps[step].result.slice(0, 80) }}{{ state.steps[step].result.length > 80 ? '...' : '' }}</span>
        </div>
      </div>
    </div>
    </template>

    <!-- 取消按钮（右下角悬浮） -->
    <button v-if="mode === 'create'" class="cancel-float-btn" @click="handleBack">
      {{ isAllDone ? '返回列表' : '取消创建' }}
    </button>
    <button v-else class="cancel-float-btn" @click="handleBackToNovels">返回小说列表</button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { useNovelButler } from '@/composables/useNovelButler'
import { novelApi, type Novel } from '@/api/novel'
import { connectWebSocket, disconnectWebSocket } from '@/utils/websocket'
import ModelSelector from '@/components/common/ModelSelector.vue'

const props = defineProps<{ id: string; pid: string }>()
const router = useRouter()

const selectedModel = ref('qwen')
const settingInput = ref('')
const started = ref(false)

// 列表/创作模式切换
const mode = ref<'list' | 'create'>('list')
const butlerNovels = ref<Novel[]>([])
const listLoading = ref(false)
const restoringNovelId = ref<number | null>(null)
const repairing = ref(false)

const {
  state,
  STEPS,
  STEP_LABELS,
  currentStepIndex,
  currentStepLabel,
  currentStepState,
  isAllDone,
  startButler,
  closeButler,
  submitStepHint,
  confirmStep,
  confirmStepAndNext,
  refineStep,
  runIterativeStep,
  confirmChapters,
  skipContent,
  retryCurrentStep,
  saveState,
  restoreState,
  clearSavedState,
  restoreFromTasks,
} = useNovelButler(
  () => Number(props.pid),
  () => selectedModel.value,
)

connectWebSocket()

// 加载管家小说列表
async function loadButlerNovels() {
  listLoading.value = true
  try {
    const data: any = await novelApi.list(Number(props.pid), 'butler')
    butlerNovels.value = data || []
  } catch {
    butlerNovels.value = []
  } finally {
    listLoading.value = false
  }
}

// 进入创作模式（新建）
function enterCreateMode() {
  mode.value = 'create'
  started.value = false
}

// 修复历史管家数据
async function repairButlerData() {
  repairing.value = true
  try {
    const data: any = await novelApi.repairButlerLinks(Number(props.pid))
    ElMessage.success(`修复完成，共修复 ${data.repaired || 0} 条任务关联`)
    await loadButlerNovels()
  } catch (e: any) {
    ElMessage.error(e.message || '修复失败')
  } finally {
    repairing.value = false
  }
}

// 恢复已有小说的管家状态
async function handleResumeNovel(novel: Novel) {
  restoringNovelId.value = novel.id
  try {
    const ok = await restoreFromTasks(novel.id)
    if (ok) {
      mode.value = 'create'
      started.value = true
    }
  } finally {
    restoringNovelId.value = null
  }
}

// 日期格式化
function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

// 小说状态文本
function novelStatusText(status: string) {
  const map: Record<string, string> = { draft: '草稿', writing: '写作中', completed: '已完成' }
  return map[status] || status
}

// 页面加载
onMounted(async () => {
  // 先尝试从 localStorage 恢复
  const restored = restoreState()
  if (restored) {
    mode.value = 'create'
    started.value = true
  } else {
    // 默认展示列表
    await loadButlerNovels()
  }
})

onUnmounted(() => {
  closeButler()
  disconnectWebSocket()
})

// state 变化时自动保存（仅在管家激活时）
watch(
  () => state,
  () => {
    if (state.active) saveState()
  },
  { deep: true },
)

const stepStatusText = computed(() => {
  const s = currentStepState.value
  if (s.status === 'generating') return '生成中...'
  if (s.status === 'choosing') return '请选择方案'
  if (s.status === 'confirmed') return '已确认'
  if (s.status === 'idle') return '等待输入'
  return ''
})

const stepHintPlaceholder = computed(() => {
  const step = state.currentStep
  const map: Record<string, string> = {
    topic: '输入你对选题的想法，如题材偏好、目标读者等（可留空直接生成）',
    storyline: '输入你对故事线的想法，如主线冲突、情感走向等（可留空直接生成）',
    characters: '输入你对人物设计的想法，如主角性格、人物数量等（可留空直接生成）',
  }
  return map[step] || ''
})

function handleStart() {
  started.value = true
  startButler(settingInput.value.trim())
}

function handleBack() {
  closeButler()
  mode.value = 'list'
  started.value = false
  loadButlerNovels()
}

function handleBackToNovels() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/novels`)
}

// 跳回已完成步骤
function handleJumpToStep(step: string) {
  ElMessageBox.confirm(
    '重新生成此步骤将清除后续所有步骤的结果，确认？',
    '跳回重新生成',
    { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning' }
  ).then(() => {
    jumpToStep(step)
  }).catch(() => {})
}

// 迭代进度文本
const iterationPhaseText = computed(() => {
  const phase = currentStepState.value.iterationPhase
  if (phase === 'generating') return '草稿生成中'
  if (phase === 'reviewing') return '审查优化中'
  return '处理中'
})

// 迭代进度百分比
const iterationProgress = computed(() => {
  const s = currentStepState.value
  if (!s.iterationMaxRounds) return 10
  if (s.iterationPhase === 'generating') return 10
  const round = s.iterationRound || 0
  const max = s.iterationMaxRounds || 5
  // 草稿占 20%，每轮 Review 占剩余 80% 的等分
  return Math.min(95, 20 + (round / max) * 75)
})

// 解析人物关系矩阵
const relationMatrix = computed(() => {
  if (state.currentStep !== 'characters') return []
  const text = currentStepState.value.result || ''
  const startTag = '---RELATION_MATRIX---'
  const endTag = '---END_MATRIX---'
  const startIdx = text.indexOf(startTag)
  const endIdx = text.indexOf(endTag)
  if (startIdx < 0 || endIdx < 0) return []
  const jsonStr = text.slice(startIdx + startTag.length, endIdx).trim()
  try {
    return JSON.parse(jsonStr) as { from: string; to: string; relation: string; detail: string; tension: string }[]
  } catch {
    return []
  }
})

// 解析故事线四幕结构
const storyStructure = computed(() => {
  if (state.currentStep !== 'storyline') return null
  const text = currentStepState.value.result || ''
  const startTag = '---STORY_STRUCTURE---'
  const endTag = '---END_STRUCTURE---'
  const startIdx = text.indexOf(startTag)
  const endIdx = text.indexOf(endTag)
  if (startIdx < 0 || endIdx < 0) return null
  const jsonStr = text.slice(startIdx + startTag.length, endIdx).trim()
  try {
    return JSON.parse(jsonStr) as {
      synopsis: string
      acts: { name: string; ratio: number; core_mission: string; key_events: string[]; turning_point: string; hook: string; beats: { name: string; goal: string; climax: string; hook: string; chapter_count: number }[] }[]
      subplots?: { name: string; description: string; intersect_acts: number[] }[]
    }
  } catch {
    return null
  }
})

// 解析人物出场规划
const appearancePlan = computed(() => {
  if (state.currentStep !== 'characters') return []
  const text = currentStepState.value.result || ''
  const startTag = '---APPEARANCE_PLAN---'
  const endTag = '---END_APPEARANCE---'
  const startIdx = text.indexOf(startTag)
  const endIdx = text.indexOf(endTag)
  if (startIdx < 0 || endIdx < 0) return []
  const jsonStr = text.slice(startIdx + startTag.length, endIdx).trim()
  try {
    return JSON.parse(jsonStr) as {
      character: string
      first_appear: { act: number; beat: string; method: string }
      act_roles: Record<string, string>
      key_scenes: { act: number; scene: string }[]
    }[]
  } catch {
    return []
  }
})

// 解析人物卡片
const characterCards = computed(() => {
  if (state.currentStep !== 'characters') return []
  const text = currentStepState.value.result || ''
  const startTag = '---CHARACTER_CARDS---'
  const endTag = '---END_CARDS---'
  const startIdx = text.indexOf(startTag)
  const endIdx = text.indexOf(endTag)
  if (startIdx < 0 || endIdx < 0) return []
  const jsonStr = text.slice(startIdx + startTag.length, endIdx).trim()
  try {
    return JSON.parse(jsonStr) as {
      name: string
      identity: string
      role_type: string
      appearance: string
      personality_surface: string
      personality_deep: string
      speech_style: string
      motivation: string
      arc: string
    }[]
  } catch {
    return []
  }
})

// 跳回已完成步骤重新生成
function jumpToStep(targetStep: string) {
  const targetIdx = STEPS.indexOf(targetStep as any)
  if (targetIdx < 0) return
  // 重置目标步骤及其后续所有步骤
  for (let i = targetIdx; i < STEPS.length; i++) {
    const s = state.steps[STEPS[i]]
    s.status = 'idle'
    s.options = []
    s.selectedIndex = -1
    s.result = ''
    s.error = null
    s.messages = []
    s.refineInput = ''
  }
  state.currentStep = targetStep as any
  // 清除相关中间数据
  if (targetIdx <= STEPS.indexOf('chapters' as any)) {
    state.outlineChapters = []
  }
}

// 重新生成（步骤2/3）— 从头重新生成，清空对话历史
function rerunIterativeStep() {
  const step = state.currentStep as 'storyline' | 'characters'
  const stepState = state.steps[step]
  stepState.status = 'idle'
  stepState.result = ''
  stepState.error = null
  stepState.messages = []
  stepState.refineInput = ''
}

// 确认迭代结果并推进下一步
function confirmIterativeAndNext() {
  const step = state.currentStep
  const idx = STEPS.indexOf(step)
  if (idx < STEPS.length - 1) {
    state.currentStep = STEPS[idx + 1]
    // 自动模式下自动开始下一步
    if (state.autoMode) {
      submitStepHint(state.currentStep)
    }
  }
}

function goToNovel() {
  if (state.createdNovelId) {
    clearSavedState()
    router.push(`/workspace/${props.id}/portfolio/${props.pid}/novel/${state.createdNovelId}`)
  }
}
</script>

<style scoped>
/* ===== 深色灵动风格 — 小说管家 ===== */
.novel-butler-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--color-bg-deep);
  color: var(--color-text-primary);
}

/* --- 顶部导航 --- */
.butler-topbar {
  padding: 12px 24px;
  background: var(--color-bg-card);
  border-radius: 12px;
  border: 1px solid var(--border-glow);
  box-shadow: var(--shadow-sm);
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  transition: box-shadow 0.3s ease;
  &:hover { box-shadow: 0 4px 12px rgba(99, 120, 255, 0.08); }
}
.butler-topbar__right {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
}
.auto-label {
  font-size: 13px;
  color: var(--color-text-secondary);
  white-space: nowrap;
}

/* --- 自定义步骤条 --- */
.butler-steps {
  padding: 24px 28px;
  background: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  border-radius: 12px;
  margin: 16px 20px 0;
}
.steps-track {
  display: flex;
  align-items: flex-start;
  max-width: 900px;
  margin: 0 auto;
}
.step-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex: 1;
  min-width: 0;
}
.step-node__dot {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  transition: all 0.4s ease;
  position: relative;
  z-index: 1;
}
.step-node--done .step-node__dot {
  background: linear-gradient(135deg, var(--color-accent-green), var(--color-accent-cyan));
  color: var(--color-text-primary);
  box-shadow: 0 0 16px rgba(110, 231, 183, 0.35);
}
.step-node--active .step-node__dot {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  color: white;
  box-shadow: 0 0 20px rgba(124, 140, 248, 0.5);
  animation: pulse-dot 2s ease-in-out infinite;
}
.step-node--pending .step-node__dot {
  background: var(--color-bg-card);
  color: var(--color-text-muted);
  border: 2px solid var(--border-glow);
}
.step-node__label {
  margin-top: 8px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}
.step-node--done .step-node__label { color: var(--color-accent-green); }
.step-node--clickable { cursor: pointer; }
.step-node--clickable:hover .step-node__dot {
  transform: scale(1.15);
  box-shadow: 0 0 20px rgba(110, 231, 183, 0.55);
}
.step-node--clickable:hover .step-node__label {
  text-decoration: underline;
}
.step-node--active .step-node__label { color: var(--color-primary-light); }
.step-node--pending .step-node__label { color: var(--color-text-muted); }
.step-node__line {
  position: absolute;
  top: 18px;
  left: calc(50% + 22px);
  right: calc(-50% + 22px);
  height: 2px;
  background: var(--border-glow);
  border-radius: 1px;
}
.step-node__line--done {
  background: linear-gradient(90deg, var(--color-accent-green), var(--color-accent-cyan));
  box-shadow: 0 0 8px rgba(110, 231, 183, 0.2);
}
@keyframes pulse-dot {
  0%, 100% { box-shadow: 0 0 20px rgba(124, 140, 248, 0.5); }
  50% { box-shadow: 0 0 30px rgba(124, 140, 248, 0.7); }
}

/* --- 启动面板 --- */
.butler-start {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  position: relative;
  overflow: hidden;
}
.start-glow {
  position: absolute;
  width: 500px;
  height: 500px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(124, 140, 248, 0.12) 0%, transparent 70%);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none;
}
.start-icon {
  font-size: 40px;
  margin-bottom: 16px;
  background: linear-gradient(135deg, var(--color-primary-light), var(--color-accent-cyan));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  animation: float-icon 3s ease-in-out infinite;
}
@keyframes float-icon {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}
.start-title {
  font-size: 24px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--color-text-primary), var(--color-primary-light));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 10px;
}
.start-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 28px;
  text-align: center;
  line-height: 1.6;
}
.start-input-wrap {
  width: 100%;
  max-width: 580px;
  position: relative;
  z-index: 1;
}
.start-btn {
  margin-top: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 32px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  color: white;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 20px rgba(124, 140, 248, 0.35);
  position: relative;
  z-index: 1;
}
.start-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 28px rgba(124, 140, 248, 0.5);
}
.start-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.start-btn__arrow {
  transition: transform 0.3s ease;
}
.start-btn:hover:not(:disabled) .start-btn__arrow {
  transform: translateX(4px);
}

/* --- 主体区域 --- */
.butler-body {
  flex: 1;
  overflow-y: auto;
  padding: 28px;
}

/* --- 步骤区块 --- */
.step-section {
  max-width: 1200px;
  margin: 0 auto;
}
.step-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-glow);
}
.step-header__badge {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  color: white;
  font-size: 14px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 10px rgba(124, 140, 248, 0.3);
}
.step-header__title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.step-header__status {
  font-size: 13px;
  padding: 3px 12px;
  border-radius: 20px;
  font-weight: 500;
}
.step-header__status--idle {
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
}
.step-header__status--generating {
  background: rgba(124, 140, 248, 0.15);
  color: var(--color-primary-light);
  animation: pulse-status 1.5s ease-in-out infinite;
}
.step-header__status--choosing {
  background: rgba(252, 211, 77, 0.15);
  color: var(--color-accent-amber);
}
.step-header__status--confirmed {
  background: rgba(110, 231, 183, 0.15);
  color: var(--color-accent-green);
}
@keyframes pulse-status {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

/* --- 用户想法输入框 --- */
.step-hint-box {
  max-width: 620px;
  padding: 24px;
  background: var(--color-bg-surface);
  border: 1px solid var(--border-glow);
  border-radius: 14px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}
.step-hint-box__desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 14px;
  line-height: 1.6;
}
.step-hint-box__row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
}
.step-hint-box__label {
  font-size: 14px;
  color: var(--color-text-primary);
  white-space: nowrap;
  font-weight: 500;
}
.step-hint-box__tip {
  font-size: 12px;
  color: var(--color-text-muted);
}

/* --- 方案卡片网格 --- */
.options-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}
.option-card {
  cursor: pointer;
  border-radius: 14px;
  border: 1px solid var(--border-glow);
  background: var(--color-bg-surface);
  padding: 20px;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}
.option-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--color-primary), var(--color-accent-cyan));
  opacity: 0;
  transition: opacity 0.3s ease;
}
.option-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 8px 30px rgba(124, 140, 248, 0.15);
  transform: translateY(-2px);
}
.option-card:hover::before {
  opacity: 1;
}
.option-card__index {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--color-primary-light);
  margin-bottom: 10px;
}
.option-card--skeleton {
  cursor: default;
}
.option-card--skeleton:hover {
  transform: none;
  box-shadow: none;
  border-color: var(--border-glow);
}
.option-card--disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.option-content {
  white-space: pre-wrap;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-secondary);
  max-height: 300px;
  overflow-y: auto;
}
.option-footer {
  margin-top: 16px;
  text-align: right;
}

/* --- 已确认 --- */
.confirmed-summary {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 22px;
  background: var(--color-bg-surface);
  border: 1px solid rgba(110, 231, 183, 0.2);
  border-radius: 14px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}
.confirmed-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-accent-green);
  padding: 4px 14px;
  background: rgba(110, 231, 183, 0.1);
  border-radius: 20px;
  width: fit-content;
}
.confirmed-text {
  white-space: pre-wrap;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-secondary);
}

/* Token 消耗统计 */
.token-usage {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-muted);
  padding: 4px 0;
}
.token-usage--inline {
  margin: 0 12px 0 0;
  font-size: 12px;
  color: var(--color-text-muted);
}

/* --- 错误提示 --- */
.step-error {
  margin-top: 16px;
}

/* --- 章节预览 --- */
.chapter-preview {
  max-height: calc(100vh - 300px);
  display: flex;
  flex-direction: column;
}
.chapter-preview__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  position: sticky;
  top: 0;
  z-index: 1;
}
.chapter-preview__count {
  font-size: 14px;
  color: var(--color-text-secondary);
  font-weight: 500;
}
.chapter-list {
  flex: 1;
  overflow-y: auto;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
  padding-bottom: 16px;
}
.chapter-card {
  display: flex;
  gap: 14px;
  padding: 18px;
  background: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  border-radius: 12px;
  transition: all 0.3s ease;
}
.chapter-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 4px 16px rgba(124, 140, 248, 0.12);
  transform: translateY(-1px);
}
.chapter-card__num {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  color: white;
  font-size: 13px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(124, 140, 248, 0.25);
}
.chapter-card__body {
  flex: 1;
  min-width: 0;
}
.chapter-card__title {
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text-primary);
  margin-bottom: 6px;
  line-height: 1.4;
}
.chapter-card__summary {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.7;
}
.chapter-preview__footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--border-glow);
}

/* --- 章节骨架屏 --- */
.chapter-skeleton {
  max-width: 600px;
  padding: 24px;
  background: var(--color-bg-surface);
  border: 1px solid var(--border-glow);
  border-radius: 14px;
}

/* --- 开篇打磨章节卡片 --- */
.opening-chapters-preview {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
}
.opening-chapter-card {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  background: var(--color-bg-surface);
  border: 1px solid var(--border-glow);
  border-radius: 10px;
}
.opening-chapter-card__num {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-primary-light), var(--color-accent-cyan));
  color: #fff;
  font-size: 13px;
  font-weight: 600;
}
.opening-chapter-card__body {
  flex: 1;
  min-width: 0;
}
.opening-chapter-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 4px;
}
.opening-chapter-card__curve,
.opening-chapter-card__hook {
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

/* --- 内容填充进度 --- */
.content-progress {
  max-width: 600px;
  margin: 48px auto;
  padding: 28px;
  background: var(--color-bg-surface);
  border-radius: 14px;
  border: 1px solid var(--border-glow);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}
.progress-info {
  font-size: 14px;
  margin-bottom: 8px;
  color: var(--color-text-primary);
}
.progress-sub {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 14px;
}

/* --- 已完成步骤回顾 --- */
.completed-steps {
  max-width: 1200px;
  margin: 32px auto 0;
  padding-top: 20px;
  border-top: 1px dashed var(--border-glow);
}
.completed-steps__title {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-bottom: 10px;
  font-weight: 500;
}
.completed-step-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 0;
  font-size: 13px;
}
.completed-step-num {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  background: var(--color-bg-card);
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}
.completed-step-label {
  font-weight: 600;
  color: var(--color-text-primary);
}
.completed-step-text {
  color: var(--color-text-secondary);
}

/* --- 列表模式 --- */
.butler-list {
  flex: 1;
  overflow-y: auto;
  padding: 20px 28px;
}
.butler-list__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}
.butler-list__title {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
}
.butler-list__loading {
  padding: 40px 0;
}
.butler-list__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px 0;
}
.empty-icon {
  font-size: 48px;
  color: var(--color-primary);
  margin-bottom: 16px;
}
.empty-text {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}
.empty-sub {
  font-size: 14px;
  color: var(--color-text-secondary);
}
.butler-novel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}
.butler-novel-card {
  background: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  border-radius: 12px;
  padding: 20px;
  cursor: pointer;
  transition: all 0.25s ease;
  position: relative;
}
.butler-novel-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 4px 20px rgba(124, 140, 248, 0.15);
  transform: translateY(-2px);
}
.butler-novel-card__title {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.butler-novel-card__desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.butler-novel-card__meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 8px;
}
.butler-novel-card__status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-text-muted);
}
.status-dot--draft { background: #f59e0b; }
.status-dot--writing { background: #3b82f6; }
.status-dot--completed { background: #10b981; }
.butler-novel-card__restoring {
  position: absolute;
  inset: 0;
  background: var(--color-bg-overlay);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: var(--color-primary-light);
}

/* --- 右下角取消按钮 --- */
.cancel-float-btn {
  position: fixed;
  right: 32px;
  bottom: 32px;
  padding: 8px 18px;
  border: 1px solid var(--border-glow);
  border-radius: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
  background: var(--color-bg-card);
  cursor: pointer;
  transition: all 0.2s;
  z-index: 10;
}
.cancel-float-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary-light);
  background: var(--color-bg-hover);
}

/* --- 通用按钮 --- */
.action-btn {
  padding: 9px 22px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.25s ease;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.action-btn--primary {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  color: white;
  box-shadow: 0 2px 12px rgba(124, 140, 248, 0.3);
}
.action-btn--primary:hover:not(:disabled) {
  box-shadow: 0 4px 20px rgba(124, 140, 248, 0.45);
  transform: translateY(-1px);
}
.action-btn--accent {
  background: linear-gradient(135deg, var(--color-accent-cyan), var(--color-accent-green));
  color: var(--color-text-primary);
  font-weight: 600;
  box-shadow: 0 2px 12px rgba(103, 232, 249, 0.25);
}
.action-btn--accent:hover:not(:disabled) {
  box-shadow: 0 4px 20px rgba(103, 232, 249, 0.4);
  transform: translateY(-1px);
}
.action-btn--ghost {
  background: var(--color-bg-card);
  color: var(--color-text-secondary);
  border: 1px solid var(--border-glow);
}
.action-btn--ghost:hover:not(:disabled) {
  border-color: var(--color-primary);
  color: var(--color-primary-light);
  background: var(--color-bg-hover);
}

/* --- 迭代进度 --- */
.iteration-progress {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48px 24px;
  text-align: center;
}
.iteration-progress__icon {
  color: var(--color-primary);
  margin-bottom: 16px;
}
.iteration-progress__title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}
.iteration-progress__detail {
  font-size: 14px;
  color: var(--color-text-secondary);
}

/* --- 迭代结果 --- */
.iteration-result {
  padding: 16px 0;
}
.iteration-result__actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}

/* --- 关系矩阵 --- */
.relation-matrix {
  margin-top: 20px;
  padding: 16px;
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-glow);
}
.relation-matrix__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}

/* ===== 故事线可选开关 ===== */
.step-hint-box__options {
  display: flex;
  gap: 20px;
  margin-top: 12px;
  padding: 10px 14px;
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-glow);
}

/* ===== 四幕卡片 ===== */
.story-acts {
  margin-bottom: 16px;
}
.story-acts__synopsis {
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-secondary);
  padding: 12px 16px;
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-glow);
  margin-bottom: 12px;
}
.story-acts__grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}
.act-card {
  padding: 14px;
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-glow);
}
.act-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.act-card__name {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.act-card__ratio {
  font-size: 12px;
  color: var(--color-accent);
  background: rgba(var(--color-accent-rgb, 99, 102, 241), 0.1);
  padding: 2px 8px;
  border-radius: 10px;
}
.act-card__mission {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}
.act-card__events {
  margin-bottom: 8px;
}
.act-card__event {
  font-size: 12px;
  color: var(--color-text-tertiary);
  line-height: 1.6;
}
.act-card__footer {
  border-top: 1px solid var(--border-glow);
  padding-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.act-card__turning,
.act-card__hook {
  font-size: 12px;
  color: var(--color-text-secondary);
}
.act-card__label {
  display: inline-block;
  font-weight: 600;
  color: var(--color-accent);
  margin-right: 6px;
  font-size: 11px;
}
.act-card__beats {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed var(--border-glow);
}
.act-card__beats-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 4px;
}
.beat-item {
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-tertiary);
  line-height: 1.6;
}
.beat-item__name {
  font-weight: 500;
  color: var(--color-text-secondary);
  white-space: nowrap;
}

/* ===== 支线展示 ===== */
.story-subplots {
  margin-top: 12px;
  padding: 12px 16px;
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-glow);
}
.story-subplots__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}
.subplot-item {
  display: flex;
  gap: 12px;
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.6;
}
.subplot-item__name {
  font-weight: 500;
  color: var(--color-text-primary);
  white-space: nowrap;
}
.subplot-item__acts {
  color: var(--color-accent);
  font-size: 12px;
  white-space: nowrap;
}

/* ===== 出场规划 ===== */
.appearance-plan {
  margin-top: 16px;
  padding: 16px;
  background: var(--color-bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-glow);
}
.appearance-plan__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.act-role-tag {
  display: inline-block;
  font-size: 12px;
  margin-right: 8px;
  color: var(--color-text-tertiary);
}
.key-scene-item {
  font-size: 12px;
  color: var(--color-text-tertiary);
  line-height: 1.6;
}

/* 人物卡片网格 */
.character-cards { margin-top: 20px; }
.character-cards__title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 12px;
}
.character-cards__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
}
.char-card {
  background: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  border-radius: 10px;
  padding: 14px 16px;
  transition: box-shadow 0.2s;
}
.char-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}
.char-card--lead { border-left: 3px solid #f56c6c; }
.char-card--core { border-left: 3px solid #e6a23c; }
.char-card--support { border-left: 3px solid #909399; }
.char-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.char-card__name {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.char-card__identity {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}
.char-card__row {
  font-size: 12px;
  color: var(--color-text-tertiary);
  line-height: 1.7;
}
.char-card__label {
  display: inline-block;
  width: 36px;
  font-weight: 600;
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

/* 原始文本折叠 */
.raw-text-details {
  margin-top: 16px;
}
.raw-text-details summary {
  cursor: pointer;
  font-size: 13px;
  color: var(--color-text-muted);
  margin-bottom: 8px;
}

/* --- 对话调整区域 --- */
.step-chat {
  margin-top: 16px;
  border-top: 1px solid var(--color-border, rgba(255,255,255,0.08));
  padding-top: 12px;
}
.step-chat__history {
  max-height: 300px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.step-chat__bubble {
  padding: 8px 12px;
  border-radius: 12px;
  max-width: 80%;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
.step-chat__bubble--user {
  align-self: flex-end;
  background: rgba(59, 130, 246, 0.1);
  color: var(--color-text-primary);
}
.step-chat__bubble--assistant {
  align-self: flex-start;
  background: var(--color-bg-secondary, rgba(255,255,255,0.04));
  color: var(--color-text-secondary);
}
.step-chat__input {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  align-items: flex-end;
}
.step-chat__input .el-textarea {
  flex: 1;
}
.step-chat__confirm {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
</style>
