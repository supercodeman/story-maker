<!-- web/src/views/novel/ButlerRelationGraph.vue -->
<!-- 管家步骤3：只读人物关系图（基于 CharacterGraph.vue 简化） -->
<template>
  <div class="butler-graph">
    <div v-if="characters.length === 0" class="butler-graph__empty">暂无人物数据</div>

    <div v-else ref="viewportRef" class="butler-graph__viewport"
      @wheel.prevent="onWheel"
      @mousedown="onPanStart">

      <!-- 缩放控制栏 -->
      <div class="butler-graph__controls">
        <button class="bg-ctrl-btn" @click="zoomIn" title="放大">＋</button>
        <span class="bg-ctrl-pct">{{ Math.round(scale * 100) }}%</span>
        <button class="bg-ctrl-btn" @click="zoomOut" title="缩小">－</button>
        <button class="bg-ctrl-btn" @click="fitAll" title="适应画布">⊡</button>
      </div>

      <!-- 变换层 -->
      <div class="butler-graph__transform"
        :style="{ transform: `translate(${panX}px, ${panY}px) scale(${scale})` }">

        <!-- 连线 SVG 层 -->
        <svg class="butler-graph__svg" :viewBox="`0 0 ${svgW} ${svgH}`"
          :width="svgW" :height="svgH">
          <defs>
            <marker v-for="color in uniqueEdgeColors" :key="color" :id="'bg-arrow-' + color.replace('#', '')"
              markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
              <path d="M0,0 L8,3 L0,6 Z" :fill="color" />
            </marker>
          </defs>
          <g v-for="edge in edgeLines" :key="edge.key">
            <line :x1="edge.x1" :y1="edge.y1" :x2="edge.x2" :y2="edge.y2"
              :stroke="edge.color" stroke-width="2"
              :stroke-dasharray="edge.dashed ? '6,3' : 'none'"
              :marker-end="'url(#bg-arrow-' + edge.color.replace('#', '') + ')'"
              class="butler-graph__edge"
              @click.stop="handleEdgeClick(edge.raw)" />
            <rect :x="edge.labelX - edge.labelW / 2 - 4" :y="edge.labelY - 8"
              :width="edge.labelW + 8" height="16" rx="3"
              fill="white" fill-opacity="0.9" stroke="none"
              class="butler-graph__edge-label-bg"
              @click.stop="handleEdgeClick(edge.raw)" />
            <text :x="edge.labelX" :y="edge.labelY + 3"
              text-anchor="middle" font-size="11" :fill="edge.color"
              class="butler-graph__edge-label"
              @click.stop="handleEdgeClick(edge.raw)">
              {{ edge.label }}
            </text>
          </g>
        </svg>

        <!-- 人物节点 -->
        <div v-for="node in nodeList" :key="node.name"
          class="butler-graph__node"
          :class="`butler-graph__node--${node.roleClass}`"
          :style="{ left: node.x + 'px', top: node.y + 'px' }"
          @mousedown.stop.prevent="startDrag($event, node)"
          @click.stop="handleNodeClick(node)">
          <div class="butler-graph__node-name">{{ node.name }}</div>
          <div class="butler-graph__node-identity">{{ node.identity }}</div>
        </div>
      </div>
    </div>

    <!-- 人物详情弹窗（只读） -->
    <el-dialog v-model="showNodeDetail" :title="selectedNode?.name || ''" width="380px" destroy-on-close>
      <div v-if="selectedNode" class="butler-graph__detail">
        <div class="detail-row"><span class="detail-label">身份</span>{{ selectedNode.identity }}</div>
        <div class="detail-row"><span class="detail-label">类型</span>
          <el-tag :type="roleTagType(selectedNode.role_type)" size="small">{{ selectedNode.role_type }}</el-tag>
        </div>
        <div v-if="selectedNode.appearance" class="detail-row"><span class="detail-label">外在</span>{{ selectedNode.appearance }}</div>
        <div v-if="selectedNode.personality_surface" class="detail-row"><span class="detail-label">表层</span>{{ selectedNode.personality_surface }}</div>
        <div v-if="selectedNode.personality_deep" class="detail-row"><span class="detail-label">内核</span>{{ selectedNode.personality_deep }}</div>
        <div v-if="selectedNode.speech_style" class="detail-row"><span class="detail-label">语言</span>{{ selectedNode.speech_style }}</div>
        <div v-if="selectedNode.motivation" class="detail-row"><span class="detail-label">动机</span>{{ selectedNode.motivation }}</div>
        <div v-if="selectedNode.arc" class="detail-row"><span class="detail-label">弧光</span>{{ selectedNode.arc }}</div>
      </div>
    </el-dialog>

    <!-- 关系详情弹窗（只读） -->
    <el-dialog v-model="showEdgeDetail" title="关系详情" width="380px" destroy-on-close>
      <div v-if="selectedEdge" class="butler-graph__detail">
        <div class="detail-row">
          <span class="detail-label">人物</span>
          <strong>{{ selectedEdge.from }}</strong> → <strong>{{ selectedEdge.to }}</strong>
        </div>
        <div class="detail-row"><span class="detail-label">关系</span>{{ selectedEdge.relation }}</div>
        <div class="detail-row"><span class="detail-label">描述</span>{{ selectedEdge.detail }}</div>
        <div class="detail-row"><span class="detail-label">张力</span>
          <el-tag :type="tensionTagType(selectedEdge.tension)" size="small">{{ selectedEdge.tension }}</el-tag>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, watch, nextTick, onMounted } from 'vue'

// 人物卡片类型
interface CharacterCard {
  name: string
  identity: string
  role_type: string
  appearance?: string
  personality_surface?: string
  personality_deep?: string
  speech_style?: string
  motivation?: string
  arc?: string
}

// 关系矩阵类型
interface RelationItem {
  from: string
  to: string
  relation: string
  detail: string
  tension: string
}

const props = defineProps<{
  characters: CharacterCard[]
  relations: RelationItem[]
}>()

const viewportRef = ref<HTMLElement | null>(null)
const showNodeDetail = ref(false)
const selectedNode = ref<CharacterCard | null>(null)
const showEdgeDetail = ref(false)
const selectedEdge = ref<RelationItem | null>(null)

// 节点尺寸
const NODE_W = 140
const NODE_H = 56

// ========== 颜色映射 ==========

// 节点颜色：按 role_type
function roleClass(roleType: string): string {
  if (roleType === '主角') return 'lead'
  if (roleType === '核心配角') return 'core'
  return 'support'
}

function roleTagType(roleType: string): '' | 'danger' | 'warning' | 'info' {
  if (roleType === '主角') return 'danger'
  if (roleType === '核心配角') return 'warning'
  return 'info'
}

// 连线颜色：按 tension
function tensionColor(tension: string): string {
  if (tension === 'high') return '#F56C6C'
  if (tension === 'medium') return '#E6A23C'
  return '#909399'
}

function tensionTagType(tension: string): '' | 'danger' | 'warning' | 'info' {
  if (tension === 'high') return 'danger'
  if (tension === 'medium') return 'warning'
  return 'info'
}

// ========== 缩放 & 平移 ==========

const scale = ref(1)
const panX = ref(0)
const panY = ref(0)
const MIN_SCALE = 0.2
const MAX_SCALE = 2

function onWheel(e: WheelEvent) {
  const delta = e.deltaY > 0 ? -0.06 : 0.06
  const newScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale.value + delta))
  if (newScale === scale.value) return
  const rect = viewportRef.value!.getBoundingClientRect()
  const mx = e.clientX - rect.left
  const my = e.clientY - rect.top
  const ratio = newScale / scale.value
  panX.value = mx - ratio * (mx - panX.value)
  panY.value = my - ratio * (my - panY.value)
  scale.value = newScale
}

function zoomIn() {
  const newScale = Math.min(MAX_SCALE, scale.value + 0.12)
  const rect = viewportRef.value?.getBoundingClientRect()
  if (rect) {
    const cx = rect.width / 2, cy = rect.height / 2
    const ratio = newScale / scale.value
    panX.value = cx - ratio * (cx - panX.value)
    panY.value = cy - ratio * (cy - panY.value)
  }
  scale.value = newScale
}

function zoomOut() {
  const newScale = Math.max(MIN_SCALE, scale.value - 0.12)
  const rect = viewportRef.value?.getBoundingClientRect()
  if (rect) {
    const cx = rect.width / 2, cy = rect.height / 2
    const ratio = newScale / scale.value
    panX.value = cx - ratio * (cx - panX.value)
    panY.value = cy - ratio * (cy - panY.value)
  }
  scale.value = newScale
}

function fitAll() {
  if (nodeList.length === 0 || !viewportRef.value) return
  const vw = viewportRef.value.clientWidth
  const vh = viewportRef.value.clientHeight
  if (vw === 0 || vh === 0) return
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity
  for (const n of nodeList) {
    minX = Math.min(minX, n.x)
    minY = Math.min(minY, n.y)
    maxX = Math.max(maxX, n.x + NODE_W)
    maxY = Math.max(maxY, n.y + NODE_H)
  }
  const contentW = maxX - minX + 80
  const contentH = maxY - minY + 80
  const s = Math.min(vw / contentW, vh / contentH, 1) * 0.8
  scale.value = Math.max(MIN_SCALE, Math.min(MAX_SCALE, s))
  panX.value = (vw - contentW * scale.value) / 2 - minX * scale.value + 40 * scale.value
  panY.value = (vh - contentH * scale.value) / 2 - minY * scale.value + 40 * scale.value
}

// 画布平移
let isPanning = false
let panStartX = 0
let panStartY = 0
let panStartPanX = 0
let panStartPanY = 0

function onPanStart(e: MouseEvent) {
  isPanning = true
  panStartX = e.clientX
  panStartY = e.clientY
  panStartPanX = panX.value
  panStartPanY = panY.value
  document.addEventListener('mousemove', onPanMove)
  document.addEventListener('mouseup', onPanEnd)
}

function onPanMove(e: MouseEvent) {
  if (!isPanning) return
  panX.value = panStartPanX + (e.clientX - panStartX)
  panY.value = panStartPanY + (e.clientY - panStartY)
}

function onPanEnd() {
  isPanning = false
  document.removeEventListener('mousemove', onPanMove)
  document.removeEventListener('mouseup', onPanEnd)
}

// ========== 节点位置 ==========

interface NodePos {
  name: string
  identity: string
  role_type: string
  roleClass: string
  x: number
  y: number
  cx: number
  cy: number
  raw: CharacterCard
}

const nodeList = reactive<NodePos[]>([])

const svgW = computed(() => {
  if (nodeList.length === 0) return 800
  return Math.max(800, ...nodeList.map(n => n.x + NODE_W + 100))
})
const svgH = computed(() => {
  if (nodeList.length === 0) return 600
  return Math.max(600, ...nodeList.map(n => n.y + NODE_H + 100))
})

// 监听 characters 变化，圆形布局
watch(() => props.characters, (chars) => {
  const count = chars.length
  if (count === 0) { nodeList.length = 0; return }

  const centerX = 400
  const centerY = 280
  const radius = Math.max(140, count * 32)

  // 重建节点列表
  nodeList.length = 0
  chars.forEach((ch, i) => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2
    const x = centerX + radius * Math.cos(angle) - NODE_W / 2
    const y = centerY + radius * Math.sin(angle) - NODE_H / 2
    nodeList.push({
      name: ch.name,
      identity: ch.identity,
      role_type: ch.role_type,
      roleClass: roleClass(ch.role_type),
      x, y,
      cx: x + NODE_W / 2,
      cy: y + NODE_H / 2,
      raw: ch,
    })
  })

  nextTick(() => requestAnimationFrame(() => fitAll()))
}, { immediate: true, deep: true })

onMounted(() => {
  if (nodeList.length > 0) {
    requestAnimationFrame(() => fitAll())
  }
})

// ========== 节点拖拽 ==========

let dragging: NodePos | null = null
let dragStartX = 0
let dragStartY = 0
let dragNodeStartX = 0
let dragNodeStartY = 0
let hasDragged = false

function startDrag(e: MouseEvent, node: NodePos) {
  dragging = node
  dragStartX = e.clientX
  dragStartY = e.clientY
  dragNodeStartX = node.x
  dragNodeStartY = node.y
  hasDragged = false
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

function onDrag(e: MouseEvent) {
  if (!dragging) return
  const dx = (e.clientX - dragStartX) / scale.value
  const dy = (e.clientY - dragStartY) / scale.value
  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) hasDragged = true
  dragging.x = dragNodeStartX + dx
  dragging.y = dragNodeStartY + dy
  dragging.cx = dragging.x + NODE_W / 2
  dragging.cy = dragging.y + NODE_H / 2
}

function stopDrag() {
  dragging = null
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

// ========== 连线计算 ==========

function edgePoint(cx: number, cy: number, tx: number, ty: number): { x: number; y: number } {
  const dx = tx - cx, dy = ty - cy
  if (dx === 0 && dy === 0) return { x: cx, y: cy }
  const hw = NODE_W / 2 + 4, hh = NODE_H / 2 + 4
  const s = Math.abs(dx) / hw > Math.abs(dy) / hh ? hw / Math.abs(dx) : hh / Math.abs(dy)
  return { x: cx + dx * s, y: cy + dy * s }
}

const edgeLines = computed(() => {
  const nameMap = new Map(nodeList.map(n => [n.name, n]))
  return props.relations.map((r, i) => {
    const from = nameMap.get(r.from)
    const to = nameMap.get(r.to)
    if (!from || !to) return null
    const color = tensionColor(r.tension)
    const dashed = r.tension === 'high'
    const p1 = edgePoint(from.cx, from.cy, to.cx, to.cy)
    const p2 = edgePoint(to.cx, to.cy, from.cx, from.cy)
    const midX = (p1.x + p2.x) / 2, midY = (p1.y + p2.y) / 2
    const len = Math.sqrt((p2.x - p1.x) ** 2 + (p2.y - p1.y) ** 2) || 1
    const nx = -(p2.y - p1.y) / len, ny = (p2.x - p1.x) / len
    const labelX = midX + nx * 14, labelY = midY + ny * 14
    const label = r.relation.length > 8 ? r.relation.substring(0, 8) + '…' : r.relation
    return {
      key: `${r.from}-${r.to}-${i}`,
      x1: p1.x, y1: p1.y, x2: p2.x, y2: p2.y,
      color, dashed, label, labelX, labelY,
      labelW: label.length * 12,
      raw: r,
    }
  }).filter(Boolean) as any[]
})

const uniqueEdgeColors = computed(() => [...new Set(edgeLines.value.map((e: any) => e.color))])

// ========== 交互 ==========

function handleNodeClick(node: NodePos) {
  if (hasDragged) return
  selectedNode.value = node.raw
  showNodeDetail.value = true
}

function handleEdgeClick(r: RelationItem) {
  selectedEdge.value = r
  showEdgeDetail.value = true
}
</script>

<style scoped lang="scss">
.butler-graph {
  display: flex;
  flex-direction: column;

  &__empty {
    text-align: center;
    color: var(--el-text-color-secondary);
    padding: 40px 0;
  }

  &__viewport {
    position: relative;
    min-height: 420px;
    overflow: hidden;
    cursor: grab;
    border-radius: 8px;
    background:
      radial-gradient(circle, var(--el-border-color-extra-light) 1px, transparent 1px);
    background-size: 20px 20px;

    &:active { cursor: grabbing; }
  }

  &__controls {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 10;
    display: flex;
    align-items: center;
    gap: 4px;
    background: rgba(255, 255, 255, 0.9);
    border-radius: 6px;
    padding: 4px 6px;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  }

  &__transform {
    position: absolute;
    top: 0;
    left: 0;
    transform-origin: 0 0;
  }

  &__svg {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
  }

  &__edge,
  &__edge-label-bg,
  &__edge-label {
    pointer-events: auto;
    cursor: pointer;
  }

  &__edge:hover { stroke-width: 3; }

  &__node {
    position: absolute;
    width: 140px;
    padding: 8px 10px;
    border-radius: 8px;
    cursor: pointer;
    user-select: none;
    text-align: center;
    transition: box-shadow 0.15s;
    border: 2px solid;

    &:hover { box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15); }

    // 主角 - 红色系
    &--lead {
      background: #fef0f0;
      border-color: #F56C6C;
      .butler-graph__node-name { color: #F56C6C; }
    }

    // 核心配角 - 橙色系
    &--core {
      background: #fdf6ec;
      border-color: #E6A23C;
      .butler-graph__node-name { color: #E6A23C; }
    }

    // 功能配角 - 蓝色系
    &--support {
      background: #ecf5ff;
      border-color: #409EFF;
      .butler-graph__node-name { color: #409EFF; }
    }
  }

  &__node-name {
    font-size: 13px;
    font-weight: 600;
    line-height: 1.3;
  }

  &__node-identity {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    margin-top: 2px;
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  &__detail {
    .detail-row {
      padding: 6px 0;
      font-size: 13px;
      line-height: 1.6;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child { border-bottom: none; }
    }

    .detail-label {
      display: inline-block;
      width: 40px;
      color: var(--el-text-color-secondary);
      font-size: 12px;
      flex-shrink: 0;
    }
  }
}

.bg-ctrl-btn {
  width: 26px;
  height: 26px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);

  &:hover {
    background: var(--el-fill-color-light);
    color: var(--el-color-primary);
  }
}

.bg-ctrl-pct {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  min-width: 32px;
  text-align: center;
}
</style>
