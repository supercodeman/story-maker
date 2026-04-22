<!-- web/src/components/common/ModelSelector.vue -->
<!-- 统一模型选择器组件，从 API 动态加载可用模型列表 -->
<template>
  <el-select
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    :size="size"
    :placeholder="placeholder"
    :loading="modelStore.loading"
    :style="selectStyle"
  >
    <!-- 分组模式：展示子模型 -->
    <template v-if="showSubModels">
      <el-option-group
        v-for="group in filteredModels"
        :key="group.value"
        :label="group.label"
      >
        <el-option
          :label="`${group.label} (默认)`"
          :value="group.value"
          :disabled="!group.available"
        >
          <span>{{ group.label }} (默认)</span>
          <el-tag v-if="group.key_source === 'user'" size="small" type="success" style="margin-left:8px">自有Key</el-tag>
          <el-tag v-if="!group.available && !group.key_source" size="small" type="warning" style="margin-left:8px">未配置Key</el-tag>
          <el-tag v-else-if="!group.available" size="small" type="danger" style="margin-left:8px">不可用</el-tag>
        </el-option>
        <el-option
          v-for="sub in (group.sub_models || [])"
          :key="sub.value"
          :label="sub.label"
          :value="sub.value"
          :disabled="!sub.available"
        >
          <span>{{ sub.label }}</span>
          <el-tag v-if="!sub.available" size="small" type="danger" style="margin-left:8px">不可用</el-tag>
        </el-option>
      </el-option-group>
    </template>

    <!-- 平铺模式：仅展示 Provider 级别 -->
    <template v-else>
      <el-option
        v-for="m in filteredModels"
        :key="m.value"
        :label="m.label"
        :value="m.value"
        :disabled="!m.available"
      >
        <span>{{ m.label }}</span>
        <el-tag v-if="m.key_source === 'user'" size="small" type="success" style="margin-left:8px">自有Key</el-tag>
        <el-tag v-if="!m.available && !m.key_source" size="small" type="warning" style="margin-left:8px">未配置Key</el-tag>
        <el-tag v-else-if="!m.available" size="small" type="danger" style="margin-left:8px">不可用</el-tag>
      </el-option>
    </template>
  </el-select>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useModelStore } from '@/store/model'

const props = withDefaults(defineProps<{
  modelValue: string
  capability?: string
  size?: '' | 'small' | 'default' | 'large'
  placeholder?: string
  showSubModels?: boolean
  style?: string
}>(), {
  capability: 'text_gen',
  size: 'small',
  placeholder: '选择模型',
  showSubModels: false,
  style: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const modelStore = useModelStore()

const selectStyle = computed(() => props.style || undefined)

const filteredModels = computed(() => {
  return modelStore.getModels(props.capability)
})

onMounted(() => {
  modelStore.fetchModels(props.capability)
})
</script>
