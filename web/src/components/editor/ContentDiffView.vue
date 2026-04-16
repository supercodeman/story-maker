<!-- 对比视图组件：左原文、右AI新内容，段落级别对齐 + 字级差异高亮 -->
<template>
  <div class="content-diff-view" :class="{ 'content-diff-view--compact': compact }">
    <div class="diff-panels" ref="panelsRef">
      <div class="diff-panel diff-panel--original" ref="leftRef" @scroll="syncScroll('left')">
        <div class="diff-panel__header">原文</div>
        <div class="diff-panel__body" :style="bodyStyle">
          <div
            v-for="(para, i) in diffParagraphs"
            :key="'l-' + i"
            class="diff-para"
            :class="{ 'diff-para--removed': para.type === 'removed', 'diff-para--changed': para.type === 'changed' }"
          >
            <span v-if="para.type === 'changed'" class="diff-para__text" v-html="para.originalHtml"></span>
            <span v-else class="diff-para__text">{{ para.original }}</span>
          </div>
        </div>
      </div>
      <div class="diff-divider" />
      <div class="diff-panel diff-panel--modified" ref="rightRef" @scroll="syncScroll('right')">
        <div class="diff-panel__header">AI 新内容</div>
        <div class="diff-panel__body" :style="bodyStyle">
          <div
            v-for="(para, i) in diffParagraphs"
            :key="'r-' + i"
            class="diff-para"
            :class="{ 'diff-para--changed': para.type === 'changed', 'diff-para--added': para.type === 'added' }"
          >
            <span v-if="para.type === 'changed'" class="diff-para__text" v-html="para.modifiedHtml"></span>
            <span v-else class="diff-para__text">{{ para.modified }}</span>
          </div>
        </div>
      </div>
    </div>
    <div class="diff-actions">
      <el-button type="primary" @click="$emit('accept')">采纳</el-button>
      <el-button @click="$emit('discard')">丢弃</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, nextTick } from 'vue'

interface DiffPara {
  original: string
  modified: string
  originalHtml?: string
  modifiedHtml?: string
  type: 'equal' | 'changed' | 'added' | 'removed'
}

const props = defineProps<{
  original: string
  modified: string
  fontFamily?: string
  fontSize?: number
  compact?: boolean
}>()

defineEmits<{
  accept: []
  discard: []
}>()

const leftRef = ref<HTMLElement | null>(null)
const rightRef = ref<HTMLElement | null>(null)
const panelsRef = ref<HTMLElement | null>(null)

// 字体样式
const bodyStyle = computed(() => ({
  fontFamily: props.fontFamily && props.fontFamily !== 'default' ? props.fontFamily : 'inherit',
  fontSize: props.fontSize ? `${props.fontSize}px` : '15px',
}))

// 按空行分段
function splitParagraphs(text: string): string[] {
  if (!text) return []
  return text.split(/\n\n+/).map(p => p.trim()).filter(Boolean)
}

// 标准化段落文本用于比较（去除空白差异）
function normalize(s: string): string {
  return s.replace(/\s+/g, '')
}

// LCS 段落对齐：用最长公共子序列匹配相同/相似段落，避免错位
function alignParagraphs(origParas: string[], modParas: string[]): DiffPara[] {
  const n = origParas.length
  const m = modParas.length

  // 相似度判断：标准化后相同视为 equal，否则计算字符重叠率
  function similarity(a: string, b: string): number {
    const na = normalize(a), nb = normalize(b)
    if (na === nb) return 1
    // 简单 bigram 重叠率
    if (na.length < 2 || nb.length < 2) return 0
    const bigramsA = new Set<string>()
    for (let i = 0; i < na.length - 1; i++) bigramsA.add(na.slice(i, i + 2))
    let overlap = 0
    for (let i = 0; i < nb.length - 1; i++) {
      if (bigramsA.has(nb.slice(i, i + 2))) overlap++
    }
    return (2 * overlap) / (na.length - 1 + nb.length - 1)
  }

  // DP: LCS with similarity threshold (>= 0.5 视为可匹配)
  const THRESHOLD = 0.5
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0))
  for (let i = 1; i <= n; i++) {
    for (let j = 1; j <= m; j++) {
      if (similarity(origParas[i - 1], modParas[j - 1]) >= THRESHOLD) {
        dp[i][j] = dp[i - 1][j - 1] + 1
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1])
      }
    }
  }

  // 回溯构建对齐结果
  const result: DiffPara[] = []
  let i = n, j = m
  const aligned: Array<[number, number]> = [] // [origIdx, modIdx] pairs

  while (i > 0 && j > 0) {
    if (similarity(origParas[i - 1], modParas[j - 1]) >= THRESHOLD && dp[i][j] === dp[i - 1][j - 1] + 1) {
      aligned.unshift([i - 1, j - 1])
      i--; j--
    } else if (dp[i - 1][j] >= dp[i][j - 1]) {
      i--
    } else {
      j--
    }
  }

  // 根据对齐结果生成 diff 行
  let oi = 0, mi = 0
  for (const [ai, aj] of aligned) {
    // 对齐点之前的未匹配段落
    while (oi < ai) {
      result.push({ original: origParas[oi], modified: '', type: 'removed' })
      oi++
    }
    while (mi < aj) {
      result.push({ original: '', modified: modParas[mi], type: 'added' })
      mi++
    }
    // 对齐的段落
    const o = origParas[ai]
    const mod = modParas[aj]
    if (normalize(o) === normalize(mod)) {
      result.push({ original: o, modified: mod, type: 'equal' })
    } else {
      const { oldHtml, newHtml } = charDiff(o, mod)
      result.push({ original: o, modified: mod, originalHtml: oldHtml, modifiedHtml: newHtml, type: 'changed' })
    }
    oi = ai + 1
    mi = aj + 1
  }
  // 尾部未匹配段落
  while (oi < n) {
    result.push({ original: origParas[oi], modified: '', type: 'removed' })
    oi++
  }
  while (mi < m) {
    result.push({ original: '', modified: modParas[mi], type: 'added' })
    mi++
  }

  return result
}

// 字级 diff：Myers 算法简化版，生成带 <mark> 标签的 HTML
function charDiff(oldStr: string, newStr: string): { oldHtml: string; newHtml: string } {
  const oldChars = [...oldStr]
  const newChars = [...newStr]

  // 简化 LCS 用于字级 diff（O(nm) 但段落通常不长）
  const n = oldChars.length, m = newChars.length
  // 优化：如果段落太长，退回段落级高亮
  if (n > 3000 || m > 3000) {
    return {
      oldHtml: escapeHtml(oldStr),
      newHtml: `<mark>${escapeHtml(newStr)}</mark>`,
    }
  }

  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0))
  for (let i = 1; i <= n; i++) {
    for (let j = 1; j <= m; j++) {
      if (oldChars[i - 1] === newChars[j - 1]) {
        dp[i][j] = dp[i - 1][j - 1] + 1
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1])
      }
    }
  }

  // 回溯得到公共字符索引
  const oldKeep = new Set<number>()
  const newKeep = new Set<number>()
  let ci = n, cj = m
  while (ci > 0 && cj > 0) {
    if (oldChars[ci - 1] === newChars[cj - 1]) {
      oldKeep.add(ci - 1)
      newKeep.add(cj - 1)
      ci--; cj--
    } else if (dp[ci - 1][cj] >= dp[ci][cj - 1]) {
      ci--
    } else {
      cj--
    }
  }

  // 生成 HTML：非公共字符用 <del>/<mark> 包裹
  let oldHtml = '', newHtml = ''
  let inDel = false, inIns = false

  for (let i = 0; i < n; i++) {
    const kept = oldKeep.has(i)
    if (!kept && !inDel) { oldHtml += '<del>'; inDel = true }
    if (kept && inDel) { oldHtml += '</del>'; inDel = false }
    oldHtml += escapeHtml(oldChars[i])
  }
  if (inDel) oldHtml += '</del>'

  for (let j = 0; j < m; j++) {
    const kept = newKeep.has(j)
    if (!kept && !inIns) { newHtml += '<mark>'; inIns = true }
    if (kept && inIns) { newHtml += '</mark>'; inIns = false }
    newHtml += escapeHtml(newChars[j])
  }
  if (inIns) newHtml += '</mark>'

  return { oldHtml, newHtml }
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// 段落级别对比（LCS 对齐 + 字级 diff）
const diffParagraphs = computed<DiffPara[]>(() => {
  const origParas = splitParagraphs(props.original)
  const modParas = splitParagraphs(props.modified)
  return alignParagraphs(origParas, modParas)
})

// 同步滚动
let scrolling = false
function syncScroll(source: 'left' | 'right') {
  if (scrolling) return
  scrolling = true
  const from = source === 'left' ? leftRef.value : rightRef.value
  const to = source === 'left' ? rightRef.value : leftRef.value
  if (from && to) {
    to.scrollTop = from.scrollTop
  }
  nextTick(() => { scrolling = false })
}
</script>

<style scoped lang="scss">
.content-diff-view {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.diff-panels {
  display: flex;
  flex: 1;
  min-height: 0;
  border: 1px solid var(--border-glow, rgba(124, 140, 248, 0.12));
  border-radius: 10px;
  overflow: hidden;
}

.diff-divider {
  width: 1px;
  background: var(--border-glow, rgba(124, 140, 248, 0.12));
  flex-shrink: 0;
}

.diff-panel {
  flex: 1;
  overflow-y: auto;
  min-width: 0;

  &::-webkit-scrollbar { width: 4px; }
  &::-webkit-scrollbar-track { background: transparent; }
  &::-webkit-scrollbar-thumb { background: rgba(124, 140, 248, 0.12); border-radius: 2px; }

  &__header {
    position: sticky;
    top: 0;
    z-index: 1;
    padding: 8px 16px;
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-muted, #999);
    background: var(--color-bg-surface, #fafafa);
    border-bottom: 1px solid var(--border-glow, rgba(124, 140, 248, 0.08));
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  &__body {
    padding: 16px;
    line-height: 1.8;
  }
}

.diff-para {
  padding: 6px 10px;
  border-radius: 6px;
  border-left: 3px solid transparent;
  margin-bottom: 8px;
  white-space: pre-wrap;
  word-break: break-word;
  min-height: 1.8em;
  transition: background 0.2s;

  // 改动段落（左右都加淡底色）
  &--changed {
    background: rgba(255, 213, 79, 0.08);
    border-left-color: rgba(255, 183, 0, 0.3);
  }

  // 右侧新增段落
  &--added {
    background: rgba(102, 187, 106, 0.12);
    border-left-color: rgba(76, 175, 80, 0.5);
  }

  // 左侧被删除段落
  &--removed {
    background: rgba(239, 83, 80, 0.08);
    border-left-color: rgba(239, 83, 80, 0.4);
    text-decoration: line-through;
    opacity: 0.7;
  }

  &__text {
    color: var(--color-text-secondary, #555);

    // 字级 diff 高亮
    :deep(mark) {
      background: rgba(102, 187, 106, 0.3);
      color: inherit;
      border-radius: 2px;
      padding: 0 1px;
    }
    :deep(del) {
      background: rgba(239, 83, 80, 0.15);
      color: var(--color-text-muted, #999);
      text-decoration: line-through;
      border-radius: 2px;
      padding: 0 1px;
    }
  }
}

.diff-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  padding: 8px 0;
}

// 紧凑模式（概要对比）
.content-diff-view--compact {
  height: auto;

  .diff-panels {
    flex: none;
    max-height: 200px;
  }

  .diff-panel__header {
    padding: 4px 12px;
    font-size: 12px;
  }

  .diff-panel__body {
    padding: 8px 12px;
  }

  .diff-para {
    padding: 2px 0;
  }

  .diff-actions {
    padding: 4px 0;
  }
}
</style>
