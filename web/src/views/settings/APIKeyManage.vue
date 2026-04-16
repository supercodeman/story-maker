<!-- web/src/views/settings/APIKeyManage.vue -->
<template>
  <div class="apikey-manage">
    <div class="page-header">
      <h1 class="page-title">API Key Management</h1>
      <NeonButton type="primary" @click="showAddDialog = true">+ Add Key</NeonButton>
    </div>

    <div v-loading="loading" class="key-list">
      <div v-if="keys.length === 0 && !loading" class="empty-state">
        No API keys configured. Add one to use real AI models.
      </div>
      <GlowCard v-for="key in keys" :key="key.id" class="key-card">
        <div class="key-card__header">
          <el-tag size="small">{{ key.provider }}</el-tag>
          <span class="key-card__mask">{{ key.key_mask }}</span>
        </div>
        <div class="key-card__actions">
          <el-button size="small" type="danger" @click="handleDelete(key.id)">Delete</el-button>
        </div>
      </GlowCard>
    </div>

    <el-dialog v-model="showAddDialog" title="Add API Key" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Provider" prop="provider">
          <el-select v-model="form.provider" style="width: 100%">
            <el-option label="ZhipuAI" value="zhipu" />
            <el-option label="Kimi" value="kimi" />
            <el-option label="Claude" value="claude" />
            <el-option label="Copilot" value="copilot" />
          </el-select>
        </el-form-item>
        <el-form-item label="API Key" prop="key_value">
          <el-input v-model="form.key_value" type="password" show-password placeholder="Enter your API key" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleAdd">Add</NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import request from '@/api/request'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

interface APIKeyItem { id: number; provider: string; key_mask: string; is_default: boolean }

const keys = ref<APIKeyItem[]>([])
const loading = ref(false)
const showAddDialog = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({ provider: 'zhipu', key_value: '' })
const rules: FormRules = {
  provider: [{ required: true, message: 'Select provider', trigger: 'change' }],
  key_value: [{ required: true, message: 'Enter API key', trigger: 'blur' }],
}

onMounted(() => fetchKeys())

async function fetchKeys() {
  loading.value = true
  try {
    const data: any = await request.get('/apikeys')
    keys.value = Array.isArray(data) ? data : []
  } finally { loading.value = false }
}

async function handleAdd() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await request.post('/apikeys', form)
      ElMessage.success('API Key added')
      showAddDialog.value = false
      form.key_value = ''
      await fetchKeys()
    } finally { submitting.value = false }
  })
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm('Delete this API key?', 'Confirm')
  await request.delete(`/apikeys/${id}`)
  ElMessage.success('Deleted')
  await fetchKeys()
}
</script>

<style scoped lang="scss">
.apikey-manage { width: 100%; max-width: 800px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.page-title { font-size: 28px; font-weight: 700; color: var(--color-text-primary); }
.key-list { display: flex; flex-direction: column; gap: 12px; }
.empty-state { text-align: center; padding: 40px; color: var(--color-text-muted); }
.key-card {
  &__header { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
  &__mask { font-family: monospace; font-size: 14px; color: var(--color-text-secondary); }
  &__actions { display: flex; justify-content: flex-end; }
}
</style>
