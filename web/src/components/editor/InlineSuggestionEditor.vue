<!-- web/src/components/editor/InlineSuggestionEditor.vue -->
<template>
  <div class="inline-suggestion-editor" :class="{ 'is-readonly': readonly }">
    <editor-content :editor="editor" class="editor-content" />
    <div v-if="loading" class="suggestion-loading">
      <span class="suggestion-loading__dot" />
      <span class="suggestion-loading__text">思考中...</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount, computed, toRef } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import { SuggestionExtension } from './SuggestionExtension'
import { useSuggestion } from './useSuggestion'

interface Props {
  modelValue: string
  placeholder?: string
  readonly?: boolean
  suggestionEnabled?: boolean
  novelId?: number
  fontFamily?: string
  fontSize?: number
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '开始写作...',
  readonly: false,
  suggestionEnabled: true,
  novelId: 0,
  fontFamily: 'default',
  fontSize: 16,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'selection-change': [text: string, start: number, end: number]
  'suggest-accept': [suggestion: string]
  'suggest-reject': [suggestion: string]
}>()

const novelIdRef = toRef(props, 'novelId')

const editor = useEditor({
  content: props.modelValue ? `<p>${props.modelValue.split('\n').join('</p><p>')}</p>` : '',
  editable: !props.readonly,
  extensions: [
    StarterKit.configure({
      // 小说编辑器只需要基础段落，不需要 heading 等
      heading: false,
      codeBlock: false,
      blockquote: false,
      bulletList: false,
      orderedList: false,
      horizontalRule: false,
    }),
    Placeholder.configure({
      placeholder: props.placeholder,
    }),
    SuggestionExtension,
  ],
  onUpdate: ({ editor: ed }) => {
    // 将 HTML 转为纯文本（段落用换行分隔）
    const text = ed.getText('\n')
    emit('update:modelValue', text)
    // 触发联想检测
    if (props.suggestionEnabled) {
      suggestionCtrl.onUpdate()
    }
  },
  onSelectionUpdate: ({ editor: ed }) => {
    const { from, to } = ed.state.selection
    if (from !== to) {
      const selectedText = ed.state.doc.textBetween(from, to, '\n')
      emit('selection-change', selectedText, from, to)
    } else {
      // 选区取消时通知父组件清空
      emit('selection-change', '', from, from)
    }
    // 选区变化时清除联想并重置输入标记
    if (suggestionCtrl.suggestion.value) {
      suggestionCtrl.rejectSuggestion()
    }
    suggestionCtrl.resetInputFlag()
  },
})

// 联想控制
const editorRef = computed(() => editor.value ?? null)
const suggestionCtrl = useSuggestion(editorRef as any, novelIdRef as any)
const loading = suggestionCtrl.loading

// 注册快捷键
watch(editor, (ed) => {
  if (!ed || ed.isDestroyed) return
  ed.view.dom.addEventListener('keydown', handleKeydown)
}, { immediate: true })

function handleKeydown(e: KeyboardEvent) {
  // 标记真实键盘输入（排除功能键、快捷键、撤销/重做）
  const isModifier = e.ctrlKey || e.metaKey || e.altKey
  const isFunctionKey = e.key.length > 1 && !['Backspace', 'Delete', 'Enter'].includes(e.key)
  if (!isModifier && !isFunctionKey) {
    suggestionCtrl.markInput()
  }

  // Tab 键处理
  if (e.key === 'Tab') {
    // 判断光标是否在段首 — 段首时执行缩进而非联想
    const ed = editor.value
    if (ed) {
      const { from, to } = ed.state.selection
      if (from === to) {
        const $pos = ed.state.doc.resolve(from)
        const offsetInPara = from - $pos.start($pos.depth)
        const paraText = $pos.parent.textContent || ''
        // 光标在段落开头或段落内容为空 → 插入缩进
        if (offsetInPara === 0 || paraText.trim() === '') {
          e.preventDefault()
          ed.commands.insertContent('\u3000\u3000') // 插入两个全角空格作为段首缩进
          return
        }
      }
    }

    e.preventDefault()
    if (suggestionCtrl.suggestion.value) {
      // 有联想内容 → 采纳
      if (suggestionCtrl.acceptSuggestion()) {
        emit('suggest-accept', suggestionCtrl.suggestion.value)
      }
    } else if (!suggestionCtrl.loading.value) {
      // 无联想且未在加载中 → 主动触发联想
      suggestionCtrl.onManualTrigger()
    }
    return
  }

  // Escape 忽略联想
  if (e.key === 'Escape' && suggestionCtrl.suggestion.value) {
    e.preventDefault()
    const text = suggestionCtrl.suggestion.value
    suggestionCtrl.rejectSuggestion()
    emit('suggest-reject', text)
    return
  }

  // Ctrl+Space 手动触发联想
  if (e.key === ' ' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    suggestionCtrl.onManualTrigger()
    return
  }
}

// 同步外部 modelValue 变化到编辑器
watch(() => props.modelValue, (newVal) => {
  if (!editor.value || editor.value.isDestroyed) return
  const currentText = editor.value.getText('\n')
  if (newVal !== currentText) {
    const html = newVal ? `<p>${newVal.split('\n').join('</p><p>')}</p>` : ''
    editor.value.commands.setContent(html, false)
  }
})

// 同步字体样式
watch([() => props.fontFamily, () => props.fontSize], () => {
  // 通过 CSS 变量传递
}, { immediate: true })

onBeforeUnmount(() => {
  try {
    if (editor.value?.view?.dom) {
      editor.value.view.dom.removeEventListener('keydown', handleKeydown)
    }
    editor.value?.destroy()
  } catch {
    // 编辑器可能已被销毁
  }
})

// 暴露方法供父组件调用
defineExpose({
  toggleSuggestion: suggestionCtrl.toggleEnabled,
  suggestionEnabled: suggestionCtrl.enabled,
  editor,
})
</script>

<style scoped>
.inline-suggestion-editor {
  position: relative;
  width: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.editor-content {
  width: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
}

/* Tiptap 编辑器 */
.editor-content :deep(.tiptap) {
  outline: none;
  flex: 1;
  min-height: 300px;
  padding: 28px 36px;
  font-family: v-bind("props.fontFamily === 'default' ? 'inherit' : props.fontFamily");
  font-size: v-bind("props.fontSize + 'px'");
  line-height: 2;
  letter-spacing: 0.02em;
  color: var(--color-text-primary);
  background: var(--color-bg-editor);
  border: 1px solid var(--border-glow);
  border-radius: 10px;
  transition: border-color 0.25s ease, box-shadow 0.25s ease;
  overflow-y: auto;
  caret-color: var(--color-primary);

  /* 自定义滚动条 */
  &::-webkit-scrollbar { width: 6px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb {
    background: var(--color-text-muted);
    opacity: 0.3;
    border-radius: 3px;
  }
}

.editor-content :deep(.tiptap:focus) {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.08);
}

.editor-content :deep(.tiptap p) {
  margin: 0 0 0.6em 0;
  text-indent: 2em;
}

.editor-content :deep(.tiptap p:last-child) {
  margin-bottom: 0;
}

/* Placeholder */
.editor-content :deep(.tiptap p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  color: var(--color-text-muted);
  pointer-events: none;
  height: 0;
  font-style: italic;
}

/* 联想文本 */
.editor-content :deep(.inline-suggestion) {
  color: var(--color-primary-light);
  opacity: 0.45;
  font-style: italic;
  pointer-events: none;
  user-select: none;
}

/* 只读模式 */
.is-readonly .editor-content :deep(.tiptap) {
  background: var(--color-bg-surface);
  cursor: default;
  opacity: 0.85;
}

/* 联想加载指示器 */
.suggestion-loading {
  position: absolute;
  bottom: 14px;
  right: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.suggestion-loading__dot {
  display: inline-block;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--color-primary, #7C8CF8);
  animation: suggestion-pulse 1.2s ease-in-out infinite;
}

.suggestion-loading__text {
  font-size: 12px;
  color: var(--color-text-muted, #5a6178);
  animation: suggestion-pulse 1.2s ease-in-out infinite;
}

@keyframes suggestion-pulse {
  0%, 100% { opacity: 0.2; transform: scale(0.8); }
  50% { opacity: 0.8; transform: scale(1); }
}
</style>
