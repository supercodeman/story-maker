<!-- web/src/views/character/CharacterList.vue -->
<template>
  <div class="character-list">
    <div class="page-header">
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: `/workspace/${id}` }">工作空间</el-breadcrumb-item>
          <el-breadcrumb-item :to="{ path: `/workspace/${id}/portfolio/${pid}` }">作品集</el-breadcrumb-item>
          <el-breadcrumb-item>角色管理</el-breadcrumb-item>
        </el-breadcrumb>
        <h1 class="page-title">角色管理</h1>
      </div>
      <NeonButton type="primary" @click="showCreateDialog = true">+ 新建角色</NeonButton>
    </div>

    <div v-loading="loading" class="character-grid">
      <div v-if="characters.length === 0 && !loading" class="empty-state">
        暂无角色，创建一个开始吧。
      </div>
      <GlowCard v-for="ch in characters" :key="ch.id" hoverable class="character-card">
        <div class="character-card__avatar">🎭</div>
        <h3 class="character-card__name">{{ ch.name }}</h3>
        <p class="character-card__desc">{{ ch.description || '暂无描述' }}</p>
        <div class="character-card__actions">
          <el-button size="small" @click="editCharacter(ch)">编辑</el-button>
          <el-button size="small" type="danger" @click="deleteCharacter(ch.id)">删除</el-button>
        </div>
      </GlowCard>
    </div>

    <el-dialog v-model="showCreateDialog" :title="editingId ? '编辑角色' : '创建角色'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="角色名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeDialog">取消</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ editingId ? '保存' : '创建' }}
        </NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import { characterApi } from '@/api/character'
import type { Character } from '@/api/character'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const props = defineProps<{ id: string; pid: string }>()

const characters = ref<Character[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const form = reactive({ name: '', description: '' })
const rules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

onMounted(() => fetchCharacters())

async function fetchCharacters() {
  loading.value = true
  try {
    const data: any = await characterApi.list(Number(props.pid))
    characters.value = Array.isArray(data) ? data : data.items || []
  } finally { loading.value = false }
}

function editCharacter(ch: Character) {
  editingId.value = ch.id
  form.name = ch.name
  form.description = ch.description || ''
  showCreateDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingId.value = null
  Object.assign(form, { name: '', description: '' })
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await characterApi.update(editingId.value, form)
        ElMessage.success('角色已更新')
      } else {
        await characterApi.create(Number(props.pid), form)
        ElMessage.success('角色已创建')
      }
      closeDialog()
      await fetchCharacters()
    } finally { submitting.value = false }
  })
}

async function deleteCharacter(id: number) {
  await ElMessageBox.confirm('确定删除该角色？', '确认')
  await characterApi.delete(id)
  ElMessage.success('已删除')
  await fetchCharacters()
}
</script>

<style scoped lang="scss">
.character-list { width: 100%; max-width: 1200px; margin: 0 auto; }
.page-header {
  display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px;
  padding: 14px 20px; background: var(--color-bg-card); border-radius: 12px;
  border: 1px solid var(--border-glow); box-shadow: var(--shadow-sm);
  transition: box-shadow 0.3s ease;
  &:hover { box-shadow: var(--shadow-md); }
}
.page-title { font-size: 22px; font-weight: 700; color: var(--color-text-primary); margin-top: 8px; }
.character-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 20px; }
.empty-state { grid-column: 1 / -1; text-align: center; padding: 60px; color: var(--color-text-muted); }
.character-card {
  text-align: center; padding: 24px;
  &__avatar { font-size: 48px; margin-bottom: 12px; }
  &__name { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
  &__desc { font-size: 13px; color: var(--color-text-secondary); min-height: 36px; margin-bottom: 12px; }
  &__actions { display: flex; justify-content: center; gap: 8px; }
}
</style>
