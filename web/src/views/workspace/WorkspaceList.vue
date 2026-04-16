<!-- web/src/views/workspace/WorkspaceList.vue -->
<template>
  <div class="workspace-list">
    <div class="page-header">
      <h1 class="page-title">工作空间</h1>
      <NeonButton type="primary" @click="showCreateDialog = true">
        + 新建工作空间
      </NeonButton>
    </div>

    <div v-loading="loading" class="workspace-grid">
      <GlowCard
        v-for="ws in workspaces"
        :key="ws.id"
        hoverable
        class="workspace-card"
        @click="goToWorkspace(ws.id)"
      >
        <div class="workspace-card__header">
          <h3 class="workspace-card__title">{{ ws.name }}</h3>
          <el-tag :type="ws.type === 'personal' ? 'info' : 'success'" size="small">
            {{ ws.type }}
          </el-tag>
        </div>
        <p class="workspace-card__desc">{{ ws.description || '暂无描述' }}</p>
        <div class="workspace-card__footer">
          <span class="workspace-card__date">{{ formatDate(ws.created_at) }}</span>
        </div>
      </GlowCard>
    </div>

    <el-dialog
      v-model="showCreateDialog"
      title="创建工作空间"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="输入工作空间名称" />
        </el-form-item>

        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="选择类型">
            <el-option label="个人" value="personal" />
            <el-option label="团队" value="team" />
          </el-select>
        </el-form-item>

        <el-form-item label="描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="输入描述（可选）"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleCreate">
          创建
        </NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { workspaceApi } from '@/api/workspace'
import { useWorkspaceStore } from '@/store/workspace'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const router = useRouter()
const workspaceStore = useWorkspaceStore()

const loading = ref(false)
const showCreateDialog = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const workspaces = computed(() => workspaceStore.workspaces)

const form = reactive({
  name: '',
  type: 'personal',
  description: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入工作空间名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
}

onMounted(async () => {
  await fetchWorkspaces()
})

async function fetchWorkspaces() {
  loading.value = true
  try {
    await workspaceStore.fetchWorkspaces()
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      await workspaceApi.create(form)
      ElMessage.success('工作空间创建成功')
      showCreateDialog.value = false
      // 刷新 store 列表并设置 currentWorkspace
      await workspaceStore.fetchWorkspaces()
      const created = workspaceStore.workspaces.find(w => w.name === form.name)
      if (created) {
        workspaceStore.setCurrentWorkspace(created)
      }
      Object.assign(form, { name: '', type: 'personal', description: '' })
    } catch (error: any) {
      ElMessage.error(error.message || '创建工作空间失败')
    } finally {
      submitting.value = false
    }
  })
}

function goToWorkspace(id: number) {
  router.push(`/workspace/${id}`)
}

function formatDate(date: string) {
  return new Date(date).toLocaleDateString()
}
</script>

<style scoped lang="scss">
.workspace-list {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.workspace-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 24px;
}

.workspace-card {
  cursor: pointer;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  &__title {
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__desc {
    font-size: 14px;
    color: var(--color-text-secondary);
    margin-bottom: 16px;
    min-height: 40px;
  }

  &__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 12px;
    border-top: 1px solid var(--border-glow);
  }

  &__date {
    font-size: 12px;
    color: var(--color-text-muted);
  }
}
</style>
