<!-- web/src/views/studio/AIStudio.vue -->
<template>
  <div class="ai-studio">
    <div class="page-header">
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">工作空间</el-breadcrumb-item>
          <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">作品集</el-breadcrumb-item>
          <el-breadcrumb-item>AI 工作室</el-breadcrumb-item>
        </el-breadcrumb>
        <h1 class="page-title">AI 工作室</h1>
      </div>
      <div class="ws-status">
        <span :class="['ws-dot', { connected: aiStore.wsConnected }]"></span>
        {{ aiStore.wsConnected ? '已连接' : '未连接' }}
      </div>
    </div>

    <div class="studio-layout">
      <!-- 左侧：设置 + 历史面板 -->
      <div class="left-panel">
        <GlowCard class="settings-panel">
          <h3 class="panel-title">设置</h3>
          <el-form label-position="top" size="small">
            <el-form-item label="任务类型">
              <el-radio-group v-model="taskType">
                <el-radio-button value="text_gen">文本</el-radio-button>
                <el-radio-button value="image_gen">图片</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="模型">
              <ModelSelector v-model="modelName" :show-sub-models="true" style="width: 100%" />
            </el-form-item>
          </el-form>
          <el-button size="small" @click="clearChat" :disabled="displayMessages.length === 0">清空对话</el-button>
        </GlowCard>

        <!-- 历史对话列表 -->
        <GlowCard class="history-panel">
          <div class="history-header">
            <h3 class="panel-title">历史记录</h3>
            <el-button size="small" type="primary" @click="newChat">+ 新建</el-button>
          </div>
          <div class="history-list">
            <div v-if="convStore.conversations.length === 0" class="history-empty">
              暂无历史记录
            </div>
            <div
              v-for="conv in convStore.conversations"
              :key="conv.id"
              :class="['history-item', { 'history-item--active': convStore.activeConvId === conv.id }]"
              @click="loadHistory(conv)"
            >
              <div class="history-item__summary">{{ truncatePrompt(conv.title) }}</div>
              <div class="history-item__meta">
                <span class="history-item__status status--completed">{{ conv.message_count }} 条消息</span>
                <span class="history-item__time">{{ formatTime(conv.updated_at) }}</span>
              </div>
            </div>
          </div>
        </GlowCard>
      </div>

      <!-- 右侧：对话区 -->
      <div class="chat-panel">
        <!-- 对话消息流 -->
        <div ref="chatContainer" class="chat-messages">
          <div v-if="displayMessages.length === 0" class="chat-empty">
            与 AI 开始对话吧，在下方输入你的提示词。
          </div>
          <div
            v-for="(msg, idx) in displayMessages"
            :key="idx"
            :class="['chat-msg', `chat-msg--${msg.role}`]"
          >
            <div class="chat-msg__avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
            <div class="chat-msg__body">
              <div class="chat-msg__role">{{ msg.role === 'user' ? '你' : modelName }}</div>
              <!-- 消息内容 -->
              <div v-if="msg.role === 'user'" class="chat-msg__text">{{ msg.content }}</div>
              <div v-else class="chat-msg__text">
                <pre v-if="msg.content">{{ msg.content }}</pre>
                <img v-if="msg.imageUrl" :src="msg.imageUrl" class="chat-msg__image" />
                <div v-if="msg.error" class="chat-msg__error">{{ msg.error }}</div>
              </div>
            </div>
          </div>

          <!-- AI 正在生成 -->
          <div v-if="generating" class="chat-msg chat-msg--assistant">
            <div class="chat-msg__avatar">🤖</div>
            <div class="chat-msg__body">
              <div class="chat-msg__role">{{ modelName }}</div>
              <div class="chat-msg__loading">
                <span class="dot"></span><span class="dot"></span><span class="dot"></span>
                <span class="loading-text">{{ taskType === 'image_gen' ? '图片生成中...' : '思考中...' }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区 -->
        <div class="chat-input">
          <el-input
            v-model="inputText"
            type="textarea"
            :rows="2"
            :placeholder="generating ? '等待 AI 回复中...' : '输入你的消息...'"
            :disabled="generating"
            @keydown.enter.exact.prevent="handleSend"
            resize="none"
          />
          <NeonButton
            type="primary"
            :loading="generating"
            :disabled="!inputText.trim() || generating"
            @click="handleSend"
          >
            发送
          </NeonButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useAIStore } from '@/store/ai'
import { useConversationStore } from '@/store/conversation'
import type { Conversation } from '@/api/conversation'
import { connectWebSocket, disconnectWebSocket } from '@/utils/websocket'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import ModelSelector from '@/components/common/ModelSelector.vue'

interface DisplayMessage {
  role: 'user' | 'assistant'
  content: string
  imageUrl?: string
  error?: string
  taskId?: number
}

const props = defineProps<{ id: string; pid: string }>()
const aiStore = useAIStore()
const convStore = useConversationStore()

const taskType = ref('text_gen')
const modelName = ref('zhipu')
const inputText = ref('')
const generating = ref(false)
const displayMessages = ref<DisplayMessage[]>([])
const chatContainer = ref<HTMLElement>()
const pendingTaskId = ref<number | null>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  connectWebSocket()
  await convStore.fetchConversations(Number(props.pid))
  aiStore.fetchTasks(Number(props.pid))
})

onUnmounted(() => {
  disconnectWebSocket()
  stopPolling()
})

// 监听 store 中任务状态变化，更新对话
watch(
  () => aiStore.tasks.map((t) => `${t.id}:${t.status}:${t.result}`),
  () => {
    if (pendingTaskId.value === null) return
    const task = aiStore.tasks.find((t) => t.id === pendingTaskId.value)
    if (!task) return

    if (task.status === 'completed' || task.status === 'failed') {
      generating.value = false
      const aiMsg: DisplayMessage = { role: 'assistant', content: '', taskId: task.id }

      if (task.status === 'failed') {
        aiMsg.error = task.error_msg || '任务失败'
      } else if (task.result) {
        try {
          const parsed = JSON.parse(task.result)
          aiMsg.content = parsed.content || ''
          aiMsg.imageUrl = parsed.image_url || ''
        } catch {
          aiMsg.content = task.result
        }

        // 保存 AI 回复到服务端会话
        if (convStore.currentConversation && aiMsg.content) {
          convStore.appendAssistantMessage(aiMsg.content, task.id)
        }
      }

      displayMessages.value.push(aiMsg)
      pendingTaskId.value = null
      stopPolling()
      scrollToBottom()
    }
  },
  { deep: true }
)

async function handleSend() {
  const text = inputText.value.trim()
  if (!text || generating.value) return

  const portfolioId = Number(props.pid)

  try {
    // 如果没有当前会话，先创建一个
    if (!convStore.currentConversation) {
      await convStore.createConversation(portfolioId, modelName.value)
    }

    const convId = convStore.currentConversation!.id

    // 添加用户消息到显示列表
    displayMessages.value.push({ role: 'user', content: text })
    inputText.value = ''
    generating.value = true
    scrollToBottom()

    // 通过 Conversation API 发送消息
    const taskId = await convStore.sendMessage(convId, text)
    pendingTaskId.value = taskId
    startPolling()
  } catch (e: any) {
    generating.value = false
    displayMessages.value.push({ role: 'assistant', content: '', error: e.message || '提交失败' })
    scrollToBottom()
  }
}

function clearChat() {
  if (convStore.currentConversation) {
    convStore.archiveConversation(convStore.currentConversation.id).catch(() => {})
  }
  convStore.clearCurrent()
  displayMessages.value = []
  pendingTaskId.value = null
  generating.value = false
}

function scrollToBottom() {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}

// 轮询兜底：WebSocket 断开时定期拉取任务状态
function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    if (pendingTaskId.value === null) {
      stopPolling()
      return
    }
    await aiStore.fetchTasks(Number(props.pid))
  }, 3000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 从会话列表加载历史对话
async function loadHistory(conv: Conversation) {
  pendingTaskId.value = null
  generating.value = false

  await convStore.loadConversation(conv.id)
  modelName.value = conv.model_name || 'zhipu'

  // 将服务端消息转为 DisplayMessage
  displayMessages.value = convStore.messages.map((msg) => ({
    role: msg.role as 'user' | 'assistant',
    content: msg.content,
    taskId: msg.task_id ?? undefined,
  }))

  scrollToBottom()
}

// 新建对话
async function newChat() {
  convStore.clearCurrent()
  displayMessages.value = []
  pendingTaskId.value = null
  generating.value = false
}

// 截取标题前 30 字作为摘要
function truncatePrompt(text: string): string {
  if (!text) return '(空)'
  return text.length > 30 ? text.slice(0, 30) + '...' : text
}

// 格式化时间
function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<style scoped lang="scss">
.ai-studio { width: 100%; max-width: 1400px; margin: 0 auto; display: flex; flex-direction: column; height: calc(100vh - 96px - 48px); }
.page-header {
  display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; flex-shrink: 0;
  padding: 14px 20px; background: var(--color-bg-card); border-radius: 12px;
  border: 1px solid var(--border-glow); box-shadow: var(--shadow-sm);
  transition: box-shadow 0.3s ease;
  &:hover { box-shadow: var(--shadow-md); }
}
.page-title { font-size: 22px; font-weight: 700; color: var(--color-text-primary); margin-top: 8px; }
.ws-status { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--color-text-muted); }
.ws-dot { width: 8px; height: 8px; border-radius: 50%; background: #ef4444;
  &.connected { background: #22c55e; }
}
.studio-layout { display: grid; grid-template-columns: 260px 1fr; gap: 24px; flex: 1; min-height: 0; }
.left-panel { display: flex; flex-direction: column; gap: 16px; min-height: 0; overflow: hidden; }
.settings-panel { padding: 20px; height: fit-content; flex-shrink: 0; }
.panel-title { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 16px; }

// 历史对话面板
.history-panel { padding: 16px; flex: 1; min-height: 0; display: flex; flex-direction: column; overflow: hidden; }
.history-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;
  .panel-title { margin-bottom: 0; }
}
.history-list { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 6px; }
.history-empty { text-align: center; padding: 20px 8px; color: var(--color-text-muted); font-size: 13px; }
.history-item {
  padding: 10px 12px; border-radius: 8px; cursor: pointer; transition: background 0.2s;
  background: var(--color-bg-deep, #0f1117);
  &:hover { background: rgba(124, 140, 248, 0.1); }
  &--active { background: rgba(124, 140, 248, 0.15); border-left: 3px solid var(--color-primary, #7c8cf8); }
  &__summary { font-size: 13px; color: var(--color-text-primary); line-height: 1.4; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &__meta { display: flex; justify-content: space-between; align-items: center; margin-top: 6px; }
  &__status { font-size: 11px; padding: 1px 6px; border-radius: 4px; font-weight: 500; }
  &__time { font-size: 11px; color: var(--color-text-muted); }
}
.status--completed { color: #22c55e; background: rgba(34, 197, 94, 0.1); }
.status--failed { color: #ef4444; background: rgba(239, 68, 68, 0.1); }
.status--running, .status--pending { color: #f59e0b; background: rgba(245, 158, 11, 0.1); }

.chat-panel { display: flex; flex-direction: column; min-height: 0; }
.chat-messages {
  flex: 1; overflow-y: auto; padding: 16px; display: flex; flex-direction: column; gap: 16px;
  background: var(--color-bg-deep, #0f1117); border-radius: 12px 12px 0 0;
}
.chat-empty { text-align: center; padding: 60px 20px; color: var(--color-text-muted); font-size: 14px; }

.chat-msg {
  display: flex; gap: 10px; max-width: 85%;
  &--user { align-self: flex-end; flex-direction: row-reverse; }
  &--assistant { align-self: flex-start; }
  &__avatar { width: 32px; height: 32px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0;
    background: var(--color-bg-card, #1a1d2e);
  }
  &__role { font-size: 11px; color: var(--color-text-muted); margin-bottom: 4px; }
  &__body { min-width: 0; }
  &__text {
    padding: 10px 14px; border-radius: 12px; font-size: 14px; line-height: 1.6;
    pre { margin: 0; white-space: pre-wrap; word-break: break-word; font-family: inherit; }
  }
  &--user &__text { background: var(--color-primary, #7c8cf8); color: #fff; border-radius: 12px 12px 2px 12px; }
  &--assistant &__text { background: var(--color-bg-card, #1a1d2e); color: var(--color-text-secondary); border-radius: 12px 12px 12px 2px; }
  &__image { max-width: 300px; border-radius: 8px; margin-top: 8px; }
  &__error { color: #ef4444; font-size: 13px; padding: 8px 12px; background: rgba(239, 68, 68, 0.1); border-radius: 8px; margin-top: 4px; }
  &__loading { display: flex; gap: 4px; padding: 12px 14px; align-items: center;
    .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-primary, #7c8cf8);
      animation: dot-bounce 1.4s infinite ease-in-out both;
      &:nth-child(1) { animation-delay: -0.32s; }
      &:nth-child(2) { animation-delay: -0.16s; }
    }
    .loading-text { margin-left: 8px; font-size: 13px; color: var(--color-text-muted); }
  }
}

@keyframes dot-bounce {
  0%, 80%, 100% { transform: scale(0.4); opacity: 0.4; }
  40% { transform: scale(1); opacity: 1; }
}

.chat-input {
  display: flex; gap: 10px; align-items: flex-end; padding: 12px 16px;
  background: var(--color-bg-card, #1a1d2e); border-radius: 0 0 12px 12px;
  :deep(.el-textarea__inner) { background: var(--color-bg-deep, #0f1117); border: 1px solid var(--border-glow, #2a2d3e); color: var(--color-text-primary); }
}

@media (max-width: 768px) {
  .studio-layout { grid-template-columns: 1fr; height: auto; }
  .left-panel { max-height: 300px; }
}
</style>
