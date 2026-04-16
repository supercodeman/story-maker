<!-- web/src/views/novel/MemoryBindingPanel.vue -->
<!-- 写作记忆绑定面板：为小说绑定/解绑各类别的写作记忆 -->
<template>
  <div class="memory-binding-panel">
    <div v-if="loading" class="memory-binding-panel__loading">
      <el-skeleton :rows="3" animated />
    </div>

    <template v-else>
      <div
        v-for="cat in categories"
        :key="cat.value"
        class="binding-slot"
      >
        <div class="binding-slot__header">
          <span class="binding-slot__icon">{{ cat.icon }}</span>
          <span class="binding-slot__label">{{ cat.label }}</span>
        </div>

        <!-- 已绑定状态 -->
        <div v-if="boundMap[cat.value]" class="binding-slot__bound">
          <div class="binding-slot__info">
            <span class="binding-slot__name">{{ boundMap[cat.value]!.title }}</span>
            <span v-if="boundMap[cat.value]!.quality > 0" class="binding-slot__quality">
              {{ boundMap[cat.value]!.quality.toFixed(0) }}分
            </span>
          </div>
          <div class="binding-slot__actions">
            <el-button type="primary" link size="small" @click="openSelector(cat.value)">更换</el-button>
            <el-button type="danger" link size="small" @click="handleUnbind(cat.value)">解绑</el-button>
          </div>
        </div>

        <!-- 未绑定状态 -->
        <button v-else class="binding-slot__empty" @click="openSelector(cat.value)">
          <span class="binding-slot__empty-text">点击绑定{{ cat.label }}</span>
        </button>
      </div>

      <!-- 空状态提示 -->
      <div v-if="!hasAnyBinding" class="binding-hint">
        绑定记忆后，AI 生成时会自动注入对应的风格、人设等约束
      </div>
    </template>

    <!-- 记忆选择弹窗 -->
    <el-dialog
      v-model="showSelector"
      :title="`选择${selectorCategoryLabel}`"
      width="520px"
      :close-on-click-modal="false"
      append-to-body
    >
      <div v-loading="selectorLoading" class="memory-selector">
        <div v-if="availableMemories.length === 0" class="memory-selector__empty">
          暂无可用的{{ selectorCategoryLabel }}记忆
        </div>
        <div
          v-for="mem in availableMemories"
          :key="mem.id"
          class="memory-selector__item"
          :class="{ 'memory-selector__item--selected': selectedMemoryId === mem.id }"
          @click="selectedMemoryId = mem.id"
        >
          <div class="memory-selector__item-main">
            <span class="memory-selector__item-title">{{ mem.title }}</span>
            <span v-if="mem.quality > 0" class="memory-selector__item-quality">{{ mem.quality.toFixed(0) }}分</span>
          </div>
          <p v-if="mem.description" class="memory-selector__item-desc">{{ mem.description }}</p>
          <div class="memory-selector__item-meta">
            <span>{{ mem.sample_len }}字样本</span>
            <span>v{{ mem.version }}</span>
            <el-tag v-if="isBound(mem.id)" size="small" type="success" effect="plain">当前绑定</el-tag>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showSelector = false">取消</el-button>
        <el-button type="primary" :disabled="!selectedMemoryId" :loading="binding" @click="handleBind">确认绑定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useMemoryStore } from '@/store/memory'
import type { WritingMemory } from '@/api/memory'
import { memoryCategoryOptions } from '@/api/memory'

const props = defineProps<{ novelId: number }>()
const memoryStore = useMemoryStore()

// 类别配置
const categories = memoryCategoryOptions.map(o => ({
  value: o.value,
  label: o.label,
  icon: { style: '✍️', character: '👤', worldview: '🌍', plot_preference: '📖' }[o.value] || '📋',
}))

const loading = ref(false)
const showSelector = ref(false)
const selectorLoading = ref(false)
const selectorCategory = ref('')
const selectedMemoryId = ref<number | null>(null)
const binding = ref(false)

// 绑定的记忆映射：category → WritingMemory
const boundMap = ref<Record<string, WritingMemory | null>>({})

const hasAnyBinding = computed(() => Object.values(boundMap.value).some(Boolean))

const selectorCategoryLabel = computed(() =>
  categories.find(c => c.value === selectorCategory.value)?.label || ''
)

// 当前选择弹窗中可用的记忆列表（已完成提取的）
const availableMemories = computed(() =>
  memoryStore.accessibleMemories.filter(m => m.extract_status === 'completed')
)

// 判断某个记忆是否是当前类别的绑定
function isBound(memoryId: number): boolean {
  return boundMap.value[selectorCategory.value]?.id === memoryId
}

// 加载绑定数据
async function loadBindings() {
  loading.value = true
  try {
    await memoryStore.fetchBindings(props.novelId)
    // 构建 boundMap
    const map: Record<string, WritingMemory | null> = {}
    for (const cat of categories) {
      const b = memoryStore.bindings.find(b => b.category === cat.value)
      if (b) {
        const mem = memoryStore.bindingMemories.find(m => m.id === b.memory_id)
        map[cat.value] = mem || null
      } else {
        map[cat.value] = null
      }
    }
    boundMap.value = map
  } finally {
    loading.value = false
  }
}

// 打开选择弹窗
async function openSelector(category: string) {
  selectorCategory.value = category
  selectedMemoryId.value = boundMap.value[category]?.id || null
  showSelector.value = true
  selectorLoading.value = true
  try {
    await memoryStore.fetchAccessible(category)
  } finally {
    selectorLoading.value = false
  }
}

// 确认绑定
async function handleBind() {
  if (!selectedMemoryId.value) return
  binding.value = true
  try {
    // 构建完整的绑定列表（保留其他类别的绑定，更新当前类别）
    const bindingList = categories.map(cat => {
      if (cat.value === selectorCategory.value) {
        return { category: cat.value, memory_id: selectedMemoryId.value! }
      }
      return { category: cat.value, memory_id: boundMap.value[cat.value]?.id || 0 }
    }).filter(b => b.memory_id > 0)

    await memoryStore.setBindings(props.novelId, bindingList)
    await loadBindings()
    showSelector.value = false
    ElMessage.success('记忆已绑定')
  } catch {
    ElMessage.error('绑定失败')
  } finally {
    binding.value = false
  }
}

// 解绑
async function handleUnbind(category: string) {
  try {
    // 构建绑定列表：要解绑的类别 memory_id 设为 0，其余保留
    const bindingList = categories
      .filter(cat => cat.value === category || boundMap.value[cat.value])
      .map(cat => ({
        category: cat.value,
        memory_id: cat.value === category ? 0 : boundMap.value[cat.value]!.id,
      }))

    await memoryStore.setBindings(props.novelId, bindingList)
    await loadBindings()
    ElMessage.success('已解绑')
  } catch {
    ElMessage.error('解绑失败')
  }
}

onMounted(() => loadBindings())

// 小说切换时重新加载
watch(() => props.novelId, () => loadBindings())
</script>

<style scoped lang="scss">
.memory-binding-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.binding-slot {
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--border-glow, rgba(124, 140, 248, 0.15));
  background: var(--color-bg-surface, rgba(30, 32, 48, 0.6));
  transition: border-color 0.2s ease;

  &:hover {
    border-color: rgba(124, 140, 248, 0.3);
  }

  &__header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  &__icon {
    font-size: 14px;
    line-height: 1;
  }

  &__label {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-secondary, #9CA3C0);
  }

  &__bound {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  &__info {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  &__name {
    font-size: 13px;
    color: var(--color-text-primary, #E8EAF6);
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 120px;
  }

  &__quality {
    font-size: 11px;
    color: var(--color-accent-amber, #F59E0B);
    flex-shrink: 0;
  }

  &__actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  &__empty {
    width: 100%;
    padding: 6px 0;
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
    transition: color 0.15s ease;
  }

  &__empty-text {
    font-size: 12px;
    color: var(--color-text-muted, #6B7280);
    transition: color 0.15s ease;

    .binding-slot__empty:hover & {
      color: var(--color-primary-light, #A5B4FC);
    }
  }
}

.binding-hint {
  font-size: 11px;
  color: var(--color-text-muted, #6B7280);
  line-height: 1.5;
  padding: 4px 0;
}

// 记忆选择弹窗
.memory-selector {
  max-height: 400px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;

  &__empty {
    text-align: center;
    padding: 32px 0;
    color: var(--color-text-muted, #6B7280);
    font-size: 13px;
  }

  &__item {
    padding: 12px;
    border-radius: 8px;
    border: 1px solid var(--border-glow, rgba(124, 140, 248, 0.15));
    background: var(--color-bg-surface, rgba(30, 32, 48, 0.6));
    cursor: pointer;
    transition: all 0.15s ease;

    &:hover {
      border-color: rgba(124, 140, 248, 0.4);
      background: var(--color-bg-hover, rgba(124, 140, 248, 0.08));
    }

    &--selected {
      border-color: var(--color-primary, #7C8CF8);
      background: rgba(124, 140, 248, 0.12);
      box-shadow: 0 0 0 1px var(--color-primary, #7C8CF8);
    }
  }

  &__item-main {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  &__item-title {
    font-size: 14px;
    font-weight: 500;
    color: var(--color-text-primary, #E8EAF6);
  }

  &__item-quality {
    font-size: 12px;
    color: var(--color-accent-amber, #F59E0B);
  }

  &__item-desc {
    font-size: 12px;
    color: var(--color-text-secondary, #9CA3C0);
    margin: 4px 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__item-meta {
    display: flex;
    gap: 8px;
    align-items: center;
    font-size: 11px;
    color: var(--color-text-muted, #6B7280);
    margin-top: 4px;
  }
}
</style>
