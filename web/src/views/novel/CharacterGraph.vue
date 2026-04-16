<!-- web/src/views/novel/CharacterGraph.vue -->
<template>
  <div class="character-graph">
    <div class="character-graph__banner">
      <span class="character-graph__banner-icon">👥</span>
      <div class="character-graph__banner-text">
        <div class="character-graph__banner-title">人物关系</div>
        <div class="character-graph__banner-desc">可视化展示小说中的人物关系网络，包括盟友、敌人、师徒、恋人等关系类型。点击节点查看人物详情，点击连线查看关系描述。支持 AI 自动提取或手动添加。</div>
      </div>
      <el-button size="small" type="primary" text @click="handleAddRelation">+ 新增关系</el-button>
    </div>

    <div v-if="overviewStore.characters.length === 0" class="character-graph__empty">
      暂无人物数据，可通过 AI 提取或在知识库中添加
    </div>

    <div v-else ref="viewportRef" class="character-graph__viewport"
      @wheel.prevent="onWheel"
      @mousedown="onPanStart">

      <!-- 缩放控制栏 -->
      <div class="character-graph__controls">
        <button class="cg-ctrl-btn" @click="zoomIn" title="放大">＋</button>
        <span class="cg-ctrl-pct">{{ Math.round(scale * 100) }}%</span>
        <button class="cg-ctrl-btn" @click="zoomOut" title="缩小">－</button>
        <button class="cg-ctrl-btn" @click="fitAll" title="适应画布">⊡</button>
      </div>

      <!-- 变换层：缩放 + 平移 -->
      <div class="character-graph__transform"
        :style="{ transform: `translate(${panX}px, ${panY}px) scale(${scale})` }">

        <!-- 关系连线 SVG 层 -->
        <svg class="character-graph__svg" :viewBox="`0 0 ${svgW} ${svgH}`"
          :width="svgW" :height="svgH">
          <defs>
            <marker v-for="color in uniqueEdgeColors" :key="color" :id="'arrow-' + color.replace('#', '')"
              markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
              <path d="M0,0 L8,3 L0,6 Z" :fill="color" />
            </marker>
          </defs>
          <g v-for="edge in edgeLines" :key="edge.id">
            <line :x1="edge.x1" :y1="edge.y1" :x2="edge.x2" :y2="edge.y2"
              :stroke="edge.color" stroke-width="2"
              :stroke-dasharray="edge.dashed ? '6,3' : 'none'"
              :marker-end="'url(#arrow-' + edge.color.replace('#', '') + ')'"
              class="character-graph__edge"
              @click.stop="handleEdgeClick(edge.relation)" />
            <rect :x="edge.labelX - edge.labelW / 2 - 4" :y="edge.labelY - 8"
              :width="edge.labelW + 8" height="16" rx="3"
              fill="white" fill-opacity="0.9" stroke="none"
              class="character-graph__edge-label-bg"
              @click.stop="handleEdgeClick(edge.relation)" />
            <text :x="edge.labelX" :y="edge.labelY + 3"
              text-anchor="middle" font-size="11" :fill="edge.color"
              class="character-graph__edge-label"
              @click.stop="handleEdgeClick(edge.relation)">
              {{ edge.label }}
            </text>
          </g>
        </svg>

        <!-- 人物节点 div 层 -->
        <div v-for="node in nodeList" :key="node.id"
          class="character-graph__node"
          :style="{ left: node.x + 'px', top: node.y + 'px' }"
          @mousedown.stop.prevent="startDrag($event, node)"
          @click.stop="handleNodeClick(node.character)">
          <div class="character-graph__node-name">{{ node.character.title }}</div>
          <div class="character-graph__node-desc">{{ (node.character.content || '').substring(0, 30) }}</div>
        </div>
      </div>
    </div>

    <!-- 人物详情弹窗 -->
    <el-dialog v-model="showNodeDetail" :title="isEditingNode ? '编辑人物' : (selectedNode?.title || '')" width="400px" destroy-on-close @close="cancelEditNode">
      <div v-if="selectedNode && !isEditingNode">
        <p>{{ selectedNode.content }}</p>
        <div v-if="selectedNode.tags" style="margin-top: 8px;">
          <el-tag v-for="tag in selectedNode.tags.split(',')" :key="tag" size="small" style="margin: 2px;">{{ tag.trim() }}</el-tag>
        </div>
        <div v-if="selectedNode.chapter_ref" style="margin-top: 8px; color: var(--el-text-color-secondary); font-size: 12px;">
          关联章节: {{ selectedNode.chapter_ref }}
        </div>
      </div>
      <el-form v-if="selectedNode && isEditingNode" label-width="60px">
        <el-form-item label="名称">
          <el-input v-model="editForm.title" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.content" type="textarea" :rows="4" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="editForm.tags" placeholder="多个标签用逗号分隔" />
        </el-form-item>
      </el-form>
      <template #footer>
        <template v-if="!isEditingNode">
          <el-button type="primary" @click="startEditNode">编辑</el-button>
        </template>
        <template v-else>
          <el-button @click="cancelEditNode">取消</el-button>
          <el-button type="primary" :loading="editSaving" @click="saveEditNode">保存</el-button>
        </template>
      </template>
    </el-dialog>

    <!-- 关系详情弹窗 -->
    <el-dialog v-model="showEdgeDetail" title="关系详情" width="400px" destroy-on-close>
      <div v-if="selectedEdge">
        <p><strong>{{ getCharacterName(selectedEdge.from_knowledge_id) }}</strong> → <strong>{{ getCharacterName(selectedEdge.to_knowledge_id) }}</strong></p>
        <p>类型: {{ getRelationLabel(selectedEdge.relation_type) }}</p>
        <p v-if="selectedEdge.label">描述: {{ selectedEdge.label }}</p>
      </div>
      <template #footer>
        <el-button size="small" type="danger" @click="handleDeleteRelation">删除关系</el-button>
      </template>
    </el-dialog>

    <!-- 新增关系弹窗 -->
    <el-dialog v-model="showRelationDialog" title="新增人物关系" width="500px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="人物 A">
          <el-select v-model="relationForm.from_knowledge_id" placeholder="选择人物" filterable>
            <el-option v-for="ch in overviewStore.characters" :key="ch.id" :label="ch.title" :value="ch.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="人物 B">
          <el-select v-model="relationForm.to_knowledge_id" placeholder="选择人物" filterable>
            <el-option v-for="ch in overviewStore.characters" :key="ch.id" :label="ch.title" :value="ch.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="关系类型">
          <el-select v-model="relationForm.relation_type" placeholder="选择关系类型">
            <el-option v-for="rt in relationTypes" :key="rt.value" :label="rt.label" :value="rt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="relationForm.label" placeholder="关系描述（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRelationDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateRelation">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, reactive, nextTick, onMounted } from 'vue'
import { useOverviewStore } from '@/store/overview'
import { relationTypes } from '@/api/overview'
import type { CharacterRelation } from '@/api/overview'
import { knowledgeApi } from '@/api/knowledge'
import type { NovelKnowledge } from '@/api/knowledge'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps<{ novelId: number }>()
const overviewStore = useOverviewStore()

const viewportRef = ref<HTMLElement | null>(null)
const showNodeDetail = ref(false)
const selectedNode = ref<NovelKnowledge | null>(null)
const showEdgeDetail = ref(false)
const selectedEdge = ref<CharacterRelation | null>(null)
const showRelationDialog = ref(false)
const relationForm = ref({
  from_knowledge_id: 0,
  to_knowledge_id: 0,
  relation_type: 'ally',
  label: '',
})

// 人物编辑状态
const isEditingNode = ref(false)
const editForm = ref({ title: '', content: '', tags: '' })
const editSaving = ref(false)

const relationColorMap: Record<string, string> = {
  ally: '#67C23A',
  enemy: '#F56C6C',
  mentor: '#409EFF',
  lover: '#E6A23C',
  family: '#909399',
  rival: '#F56C6C',
  custom: '#606266',
}

const NODE_W = 160
const NODE_H = 64

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

  // 以鼠标位置为中心缩放
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
  // viewport 尚未渲染完成时跳过
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
  // 缩放比例为适配值的 80%，确保内容不会太小或太挤
  const s = Math.min(vw / contentW, vh / contentH, 1) * 0.8
  scale.value = Math.max(MIN_SCALE, Math.min(MAX_SCALE, s))
  // 居中显示
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
  // 只在空白区域拖拽时平移（节点拖拽由 startDrag 处理）
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

// ========== 节点位置管理 ==========

interface NodePos {
  id: number
  x: number
  y: number
  cx: number
  cy: number
  character: NovelKnowledge
}

const nodeList = reactive<NodePos[]>([])

// SVG 尺寸跟随节点范围
const svgW = computed(() => {
  if (nodeList.length === 0) return 800
  return Math.max(800, ...nodeList.map(n => n.x + NODE_W + 100))
})
const svgH = computed(() => {
  if (nodeList.length === 0) return 600
  return Math.max(600, ...nodeList.map(n => n.y + NODE_H + 100))
})

watch(() => overviewStore.characters, (chars) => {
  const count = chars.length
  if (count === 0) { nodeList.length = 0; return }

  const centerX = 420
  const centerY = 300
  const radius = Math.max(160, count * 35)

  const existingIds = new Set(nodeList.map(n => n.id))
  const newIds = new Set(chars.map(c => c.id))

  for (let i = nodeList.length - 1; i >= 0; i--) {
    if (!newIds.has(nodeList[i].id)) nodeList.splice(i, 1)
  }

  chars.forEach((ch, i) => {
    if (existingIds.has(ch.id)) {
      const existing = nodeList.find(n => n.id === ch.id)!
      existing.character = ch
      return
    }
    const angle = (2 * Math.PI * i) / count - Math.PI / 2
    const x = centerX + radius * Math.cos(angle) - NODE_W / 2
    const y = centerY + radius * Math.sin(angle) - NODE_H / 2
    nodeList.push({ id: ch.id, x, y, cx: x + NODE_W / 2, cy: y + NODE_H / 2, character: ch })
  })

  // 初始加载时自动适应，延迟确保 viewport 已渲染
  nextTick(() => {
    // 使用 requestAnimationFrame 确保浏览器已完成布局
    requestAnimationFrame(() => fitAll())
  })
}, { immediate: true, deep: true })

// 组件挂载后兜底执行 fitAll（immediate watch 时 viewport 可能还未渲染）
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
  const scale = Math.abs(dx) / hw > Math.abs(dy) / hh ? hw / Math.abs(dx) : hh / Math.abs(dy)
  return { x: cx + dx * scale, y: cy + dy * scale }
}

const edgeLines = computed(() => {
  const posMap = new Map(nodeList.map(n => [n.id, n]))
  return overviewStore.relations.map(r => {
    const from = posMap.get(r.from_knowledge_id)
    const to = posMap.get(r.to_knowledge_id)
    if (!from || !to) return null
    const color = relationColorMap[r.relation_type] || '#909399'
    const dashed = r.relation_type === 'enemy' || r.relation_type === 'rival'
    const p1 = edgePoint(from.cx, from.cy, to.cx, to.cy)
    const p2 = edgePoint(to.cx, to.cy, from.cx, from.cy)
    const midX = (p1.x + p2.x) / 2, midY = (p1.y + p2.y) / 2
    const len = Math.sqrt((p2.x - p1.x) ** 2 + (p2.y - p1.y) ** 2) || 1
    const nx = -(p2.y - p1.y) / len, ny = (p2.x - p1.x) / len
    const labelX = midX + nx * 14, labelY = midY + ny * 14
    const label = r.label
      ? (r.label.length > 10 ? r.label.substring(0, 10) + '…' : r.label)
      : getRelationLabel(r.relation_type)
    return {
      id: r.id, x1: p1.x, y1: p1.y, x2: p2.x, y2: p2.y,
      color, dashed, label, labelX, labelY, labelW: label.length * 11, relation: r,
    }
  }).filter(Boolean) as any[]
})

const uniqueEdgeColors = computed(() => [...new Set(edgeLines.value.map(e => e.color))])

// ========== 交互 ==========

function getRelationLabel(type: string): string {
  return relationTypes.find(rt => rt.value === type)?.label || type
}

function getCharacterName(knowledgeId: number): string {
  return overviewStore.characters.find(c => c.id === knowledgeId)?.title || '未知'
}

function handleNodeClick(ch: NovelKnowledge) {
  if (hasDragged) return
  selectedNode.value = ch
  showNodeDetail.value = true
}

function handleEdgeClick(r: CharacterRelation) {
  selectedEdge.value = r
  showEdgeDetail.value = true
}

function handleAddRelation() {
  relationForm.value = { from_knowledge_id: 0, to_knowledge_id: 0, relation_type: 'ally', label: '' }
  showRelationDialog.value = true
}

async function handleCreateRelation() {
  const f = relationForm.value
  if (!f.from_knowledge_id || !f.to_knowledge_id) { ElMessage.warning('请选择两个人物'); return }
  if (f.from_knowledge_id === f.to_knowledge_id) { ElMessage.warning('不能选择同一个人物'); return }
  try {
    await overviewStore.createRelation(props.novelId, f)
    showRelationDialog.value = false
    ElMessage.success('关系创建成功')
  } catch { ElMessage.error('创建失败') }
}

async function handleDeleteRelation() {
  if (!selectedEdge.value) return
  try {
    await ElMessageBox.confirm('确定删除该关系？', '提示', { type: 'warning' })
    await overviewStore.deleteRelation(props.novelId, selectedEdge.value.id)
    showEdgeDetail.value = false
    selectedEdge.value = null
    ElMessage.success('已删除')
  } catch { /* cancelled */ }
}

// ========== 人物编辑 ==========

function startEditNode() {
  if (!selectedNode.value) return
  editForm.value = {
    title: selectedNode.value.title,
    content: selectedNode.value.content || '',
    tags: selectedNode.value.tags || '',
  }
  isEditingNode.value = true
}

async function saveEditNode() {
  if (!selectedNode.value) return
  editSaving.value = true
  try {
    await knowledgeApi.update(selectedNode.value.id, editForm.value)
    // 更新本地 store 中的数据
    const idx = overviewStore.characters.findIndex(c => c.id === selectedNode.value!.id)
    if (idx !== -1) {
      Object.assign(overviewStore.characters[idx], editForm.value)
    }
    selectedNode.value = { ...selectedNode.value, ...editForm.value }
    isEditingNode.value = false
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    editSaving.value = false
  }
}

function cancelEditNode() {
  isEditingNode.value = false
}
</script>

<style scoped lang="scss">
.character-graph {
  height: 100%;
  display: flex;
  flex-direction: column;

  &__banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: var(--el-color-primary-light-9);
    border-radius: 6px;
    margin: 8px 0 8px;

    .el-button { margin-left: auto; flex-shrink: 0; }
  }

  &__banner-text { flex: 1; min-width: 0; }

  &__banner-icon { font-size: 18px; line-height: 1; flex-shrink: 0; }

  &__banner-title {
    font-weight: 600;
    font-size: 13px;
    color: var(--el-text-color-primary);
    margin-bottom: 2px;
  }

  &__banner-desc {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
  }

  &__toolbar {
    display: none;
  }

  &__empty {
    text-align: center;
    color: var(--el-text-color-secondary);
    padding: 60px 0;
  }

  &__viewport {
    position: relative;
    flex: 1;
    min-height: 500px;
    overflow: hidden;
    cursor: grab;
    background:
      radial-gradient(circle, var(--el-border-color-extra-light) 1px, transparent 1px);
    background-size: 20px 20px;

    &:active { cursor: grabbing; }
  }

  &__controls {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 10;
    display: flex;
    align-items: center;
    gap: 4px;
    background: #fff;
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    padding: 4px 6px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.06);
    user-select: none;
  }

  &__transform {
    position: absolute;
    top: 0;
    left: 0;
    transform-origin: 0 0;
    will-change: transform;
  }

  &__svg {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
    overflow: visible;
  }

  &__edge {
    pointer-events: stroke;
    cursor: pointer;
    stroke-linecap: round;
    &:hover { stroke-width: 3; }
  }

  &__edge-label-bg {
    pointer-events: all;
    cursor: pointer;
  }

  &__edge-label {
    pointer-events: all;
    cursor: pointer;
    user-select: none;
    font-size: 11px;
  }

  &__node {
    position: absolute;
    width: 160px;
    background: #fff;
    border: 2px solid var(--el-color-primary);
    border-radius: 10px;
    padding: 12px 14px;
    cursor: grab;
    transition: box-shadow 0.15s;
    z-index: 1;

    &:active {
      cursor: grabbing;
      box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
    }
    &:hover {
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
    }
  }

  &__node-name {
    font-weight: 600;
    font-size: 14px;
    color: var(--el-text-color-primary);
  }

  &__node-desc {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    margin-top: 4px;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
}

.cg-ctrl-btn {
  width: 28px;
  height: 28px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  font-size: 16px;
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

.cg-ctrl-pct {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  min-width: 36px;
  text-align: center;
}
</style>
