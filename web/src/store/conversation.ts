// web/src/store/conversation.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { conversationApi } from '@/api/conversation'
import type { Conversation, Message } from '@/api/conversation'

export type { Conversation, Message } from '@/api/conversation'

export const useConversationStore = defineStore('conversation', () => {
  const conversations = ref<Conversation[]>([])
  const currentConversation = ref<Conversation | null>(null)
  const messages = ref<Message[]>([])
  const loading = ref(false)

  const activeConvId = computed(() => currentConversation.value?.id ?? null)

  // 加载会话列表
  async function fetchConversations(portfolioId?: number) {
    loading.value = true
    try {
      const data: any = await conversationApi.list({
        portfolio_id: portfolioId,
        page: 1,
        page_size: 50,
      })
      conversations.value = data.conversations || []
    } finally {
      loading.value = false
    }
  }

  // 创建新会话
  async function createConversation(portfolioId: number, modelName?: string, title?: string) {
    const data: any = await conversationApi.create({
      portfolio_id: portfolioId,
      model_name: modelName,
      title,
    })
    const conv = data as Conversation
    conversations.value.unshift(conv)
    currentConversation.value = conv
    messages.value = []
    return conv
  }

  // 加载会话详情（含消息）
  async function loadConversation(convId: number) {
    loading.value = true
    try {
      const data: any = await conversationApi.get(convId)
      currentConversation.value = data.conversation
      messages.value = data.messages || []
    } finally {
      loading.value = false
    }
  }

  // 发送消息
  async function sendMessage(convId: number, content: string) {
    // 先在本地追加用户消息
    const userMsg: Message = {
      id: 0,
      conversation_id: convId,
      role: 'user',
      content,
      token_count: 0,
      task_id: null,
      created_at: new Date().toISOString(),
    }
    messages.value.push(userMsg)

    const data: any = await conversationApi.sendMessage(convId, { content })
    return data.task_id as number
  }

  // AI 回复完成后追加 assistant 消息
  function appendAssistantMessage(content: string, taskId: number) {
    messages.value.push({
      id: 0,
      conversation_id: currentConversation.value?.id ?? 0,
      role: 'assistant',
      content,
      token_count: 0,
      task_id: taskId,
      created_at: new Date().toISOString(),
    })

    // 更新会话列表中的消息计数
    if (currentConversation.value) {
      currentConversation.value.message_count += 1
      currentConversation.value.updated_at = new Date().toISOString()
    }
  }

  // 归档会话
  async function archiveConversation(convId: number) {
    await conversationApi.archive(convId)
    conversations.value = conversations.value.filter((c) => c.id !== convId)
    if (currentConversation.value?.id === convId) {
      currentConversation.value = null
      messages.value = []
    }
  }

  // 清空当前会话状态
  function clearCurrent() {
    currentConversation.value = null
    messages.value = []
  }

  return {
    conversations,
    currentConversation,
    messages,
    loading,
    activeConvId,
    fetchConversations,
    createConversation,
    loadConversation,
    sendMessage,
    appendAssistantMessage,
    archiveConversation,
    clearCurrent,
  }
})
