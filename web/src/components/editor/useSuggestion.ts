// web/src/components/editor/useSuggestion.ts
// 联想逻辑 composable：停顿自动触发 / 快捷键触发
import { ref, type Ref, onUnmounted } from 'vue'
import type { Editor } from '@tiptap/vue-3'
import { suggestionPluginKey } from './SuggestionExtension'
import { suggestionApi } from '../../api/suggestion'

export function useSuggestion(editor: Ref<Editor | null>, novelId: Ref<number>) {
  const suggestion = ref('')
  const loading = ref(false)
  const enabled = ref(localStorage.getItem('suggestion_enabled') !== 'false')

  let abortController: AbortController | null = null
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let lastRequestTime = 0
  let hasInputSinceLastFocus = false // 追踪光标聚焦后是否有实际输入
  const MIN_INTERVAL = 3000 // 全局节流：至少 3 秒间隔

  // 标记用户有实际输入（由 onUpdate 调用）
  function markInput() {
    hasInputSinceLastFocus = true
  }

  // 光标/选区变化时重置输入标记
  function resetInputFlag() {
    hasInputSinceLastFocus = false
  }

  // 停顿触发：用户停止输入 800ms 后自动请求联想
  function onUpdate() {
    if (!enabled.value || !editor.value) return
    markInput()
    clearPending()
    debounceTimer = setTimeout(() => {
      if (shouldTrigger()) {
        fetchSuggestion()
      }
    }, 800)
  }

  // 快捷键触发：Ctrl+Space 或 Tab（无联想时）立即请求联想
  function onManualTrigger() {
    clearPending()
    fetchSuggestion()
  }

  // 判断是否满足触发条件
  function shouldTrigger(): boolean {
    const ed = editor.value
    if (!ed) return false

    const { from, to } = ed.state.selection
    // 必须是光标（非选区）
    if (from !== to) return false

    // 必须有实际输入才自动触发（防止进入页面或仅移动光标时触发）
    if (!hasInputSinceLastFocus) return false

    // 获取光标所在段落的文本
    const $pos = ed.state.doc.resolve(from)
    const paraText = $pos.parent.textContent || ''

    // 当前行至少 10 个字符
    if (paraText.length < 10) return false

    // 全局节流
    if (Date.now() - lastRequestTime < MIN_INTERVAL) return false

    return true
  }

  // 请求联想 API
  async function fetchSuggestion() {
    const ed = editor.value
    if (!ed || ed.isDestroyed || !novelId.value) return

    // 取消之前的请求
    cancelPending()

    const { from } = ed.state.selection
    // 提取光标前 500 字作为上下文
    const docText = ed.state.doc.textBetween(0, from, '\n')
    const precedingText = docText.slice(-500)

    if (!precedingText.trim()) return

    loading.value = true
    lastRequestTime = Date.now()
    abortController = new AbortController()

    try {
      const res: any = await suggestionApi.fetchSuggestion(
        novelId.value,
        precedingText,
        abortController.signal,
      )
      const text = res?.suggestion || ''
      if (text && !ed.isDestroyed && ed.state.selection.from === from) {
        suggestion.value = text
        // 设置 Decoration
        ed.view.dispatch(
          ed.state.tr.setMeta(suggestionPluginKey, {
            suggestion: text,
            position: from,
          }),
        )
      }
    } catch (err: any) {
      if (err?.name !== 'AbortError' && err?.code !== 'ERR_CANCELED') {
        console.warn('[Suggestion] fetch failed:', err)
      }
    } finally {
      loading.value = false
      abortController = null
    }
  }

  // Tab 采纳联想
  function acceptSuggestion(): boolean {
    const ed = editor.value
    if (!ed || !suggestion.value) return false

    const text = suggestion.value
    // 插入联想文本
    ed.commands.insertContent(text)
    clearSuggestion()

    // 上报 suggest_accept 事件
    suggestionApi.recordBehavior({
      novel_id: novelId.value,
      event_type: 'suggest_accept',
      payload: { suggestion: text },
    }).catch(() => {})

    return true
  }

  // 拒绝/忽略联想
  function rejectSuggestion() {
    if (!suggestion.value) return

    const text = suggestion.value
    clearSuggestion()

    // 上报 suggest_reject 事件
    suggestionApi.recordBehavior({
      novel_id: novelId.value,
      event_type: 'suggest_reject',
      payload: { suggestion: text },
    }).catch(() => {})
  }

  // 清除联想显示
  function clearSuggestion() {
    suggestion.value = ''
    const ed = editor.value
    if (ed && !ed.isDestroyed) {
      ed.view.dispatch(
        ed.state.tr.setMeta(suggestionPluginKey, {
          suggestion: '',
          position: -1,
        }),
      )
    }
  }

  // 取消进行中的请求
  function cancelPending() {
    if (abortController) {
      abortController.abort()
      abortController = null
    }
  }

  // 清除定时器和请求
  function clearPending() {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
    cancelPending()
  }

  // 切换联想开关
  function toggleEnabled() {
    enabled.value = !enabled.value
    localStorage.setItem('suggestion_enabled', String(enabled.value))
    if (!enabled.value) {
      clearPending()
      clearSuggestion()
    }
  }

  onUnmounted(() => {
    clearPending()
  })

  return {
    suggestion,
    loading,
    enabled,
    onUpdate,
    onManualTrigger,
    acceptSuggestion,
    rejectSuggestion,
    clearSuggestion,
    resetInputFlag,
    toggleEnabled,
  }
}
