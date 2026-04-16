// web/src/components/editor/SuggestionExtension.ts
// Tiptap 自定义扩展：在光标位置后显示灰色联想文本（Decoration）
import { Extension } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

export const suggestionPluginKey = new PluginKey('inlineSuggestion')

export interface SuggestionPluginState {
  suggestion: string
  position: number // 联想文本插入位置
}

export const SuggestionExtension = Extension.create({
  name: 'inlineSuggestion',

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: suggestionPluginKey,
        state: {
          init(): SuggestionPluginState {
            return { suggestion: '', position: -1 }
          },
          apply(tr, prev): SuggestionPluginState {
            const meta = tr.getMeta(suggestionPluginKey)
            if (meta !== undefined) {
              return meta as SuggestionPluginState
            }
            // 如果文档发生变化（用户输入），清除联想
            if (tr.docChanged) {
              return { suggestion: '', position: -1 }
            }
            return prev
          },
        },
        props: {
          decorations(state) {
            const pluginState = suggestionPluginKey.getState(state) as SuggestionPluginState
            if (!pluginState?.suggestion || pluginState.position < 0) {
              return DecorationSet.empty
            }

            // 在光标位置创建 widget decoration 显示灰色联想文本
            const widget = Decoration.widget(pluginState.position, () => {
              const span = document.createElement('span')
              span.className = 'inline-suggestion'
              span.textContent = pluginState.suggestion
              span.style.color = '#9ca3af'
              span.style.opacity = '0.7'
              span.style.pointerEvents = 'none'
              return span
            }, { side: 1 })

            return DecorationSet.create(state.doc, [widget])
          },
        },
      }),
    ]
  },
})
