<!-- web/src/views/workspace/WorkspaceDetail.vue -->
<template>
  <div class="workspace-detail">
    <div class="page-header">
      <div class="page-title-row">
        <h1 class="page-title">{{ workspace?.name || '加载中...' }}</h1>
        <el-button link @click="handleEdit" style="margin-left: 8px; font-size: 18px;">
          <el-icon><Edit /></el-icon>
        </el-button>
        <p class="page-desc" style="width: 100%;">{{ workspace?.description || '' }}</p>
      </div>
      <div class="header-actions">
        <NeonButton @click="showMemberDialog = true">成员</NeonButton>
        <NeonButton type="primary" @click="showCreateDialog = true">+ 新建作品集</NeonButton>
      </div>
    </div>

    <div v-loading="loading" class="portfolio-grid">
      <GlowCard
        v-for="p in portfolios"
        :key="p.id"
        hoverable
        class="portfolio-card"
        @click="goToPortfolio(p.id)"
      >
        <div v-if="p.cover_image" class="portfolio-card__cover">
          <img :src="p.cover_image" :alt="p.name" />
        </div>
        <div class="portfolio-card__header">
          <h3>{{ p.name }}</h3>
          <el-tag :type="p.status === 'published' ? 'success' : 'info'" size="small">
            {{ p.status || 'draft' }}
          </el-tag>
        </div>
        <p class="portfolio-card__desc">{{ p.description || '暂无描述' }}</p>
        <div class="portfolio-card__footer">
          <span>{{ formatDate(p.created_at) }}</span>
          <el-dropdown trigger="click" @command="handlePortfolioAction($event, p)">
            <span class="el-dropdown-link" @click.stop>...</span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit">编辑</el-dropdown-item>
                <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </GlowCard>
    </div>

    <!-- 创建作品集弹窗 -->
    <el-dialog v-model="showCreateDialog" title="创建作品集" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="作品集名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleCreate">创建</NeonButton>
      </template>
    </el-dialog>

    <!-- 成员管理弹窗 -->
    <el-dialog v-model="showMemberDialog" title="成员管理" width="600px">
      <MemberManage v-if="showMemberDialog" :workspace-id="Number(id)" />
    </el-dialog>

    <!-- 编辑空间弹窗 -->
    <el-dialog v-model="showEditDialog" title="编辑空间" width="500px">
      <el-form ref="editFormRef" :model="editForm" :rules="editRules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="editForm.name" placeholder="空间名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <NeonButton type="primary" :loading="editSubmitting" @click="handleEditSubmit">保存</NeonButton>
      </template>
    </el-dialog>

    <!-- 编辑作品集弹窗 -->
    <el-dialog v-model="showEditPortfolioDialog" title="编辑作品集" width="520px">
      <el-form ref="editPortfolioFormRef" :model="editPortfolioForm" :rules="editPortfolioRules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="editPortfolioForm.name" placeholder="作品集名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editPortfolioForm.description" type="textarea" :rows="3" placeholder="作品集描述" />
        </el-form-item>
        <el-form-item label="封面图">
          <div class="cover-upload">
            <div v-if="editPortfolioForm.cover_image" class="cover-preview">
              <img :src="editPortfolioForm.cover_image" alt="封面" />
              <el-button class="cover-remove" type="danger" size="small" circle @click="editPortfolioForm.cover_image = ''">
                <el-icon><Close /></el-icon>
              </el-button>
            </div>
            <el-upload
              v-else
              :show-file-list="false"
              :before-upload="handleCoverUpload"
              accept="image/jpeg,image/png,image/webp,image/gif"
            >
              <div class="cover-trigger">
                <el-icon :size="24"><Plus /></el-icon>
                <span>上传封面</span>
              </div>
            </el-upload>
            <div v-if="coverUploading" class="cover-loading">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>上传中...</span>
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditPortfolioDialog = false">取消</el-button>
        <NeonButton type="primary" :loading="editPortfolioSubmitting" @click="handleEditPortfolioSubmit">保存</NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import { Edit, Close, Plus, Loading } from '@element-plus/icons-vue'
import { workspaceApi } from '@/api/workspace'
import { portfolioApi } from '@/api/portfolio'
import type { Portfolio } from '@/api/portfolio'
import { useWorkspaceStore } from '@/store/workspace'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import MemberManage from './MemberManage.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()
const workspaceStore = useWorkspaceStore()

const workspace = ref<any>(null)
const portfolios = ref<Portfolio[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showMemberDialog = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({ name: '', description: '' })
const rules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

// 编辑空间相关状态
const showEditDialog = ref(false)
const editSubmitting = ref(false)
const editFormRef = ref<FormInstance>()
const editForm = reactive({ name: '', description: '' })
const editRules: FormRules = {
  name: [{ required: true, message: '请输入空间名称', trigger: 'blur' }],
}

onMounted(async () => {
  await Promise.all([fetchWorkspace(), fetchPortfolios()])
})

// 监听路由参数变化，组件复用时重新加载数据
watch(() => props.id, async () => {
  await Promise.all([fetchWorkspace(), fetchPortfolios()])
})

async function fetchWorkspace() {
  try {
    workspace.value = await workspaceApi.get(Number(props.id))
  } catch (e: any) {
    ElMessage.error('加载工作空间失败')
  }
}

async function fetchPortfolios() {
  loading.value = true
  try {
    const data: any = await portfolioApi.list(Number(props.id))
    portfolios.value = Array.isArray(data) ? data : data.items || []
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
      await portfolioApi.create({
        workspace_id: Number(props.id),
        name: form.name,
        description: form.description,
      })
      ElMessage.success('作品集已创建')
      showCreateDialog.value = false
      Object.assign(form, { name: '', description: '' })
      await fetchPortfolios()
    } finally {
      submitting.value = false
    }
  })
}

function goToPortfolio(pid: number) {
  router.push(`/workspace/${props.id}/portfolio/${pid}`)
}

async function handlePortfolioAction(action: string, p: Portfolio) {
  if (action === 'edit') {
    editingPortfolio.value = p
    editPortfolioForm.name = p.name
    editPortfolioForm.description = p.description || ''
    editPortfolioForm.cover_image = p.cover_image || ''
    showEditPortfolioDialog.value = true
  } else if (action === 'delete') {
    await ElMessageBox.confirm('确定删除该作品集？', '确认')
    await portfolioApi.delete(p.id)
    ElMessage.success('已删除')
    await fetchPortfolios()
  }
}

// 编辑作品集相关状态
const showEditPortfolioDialog = ref(false)
const editPortfolioSubmitting = ref(false)
const editPortfolioFormRef = ref<FormInstance>()
const editingPortfolio = ref<Portfolio | null>(null)
const coverUploading = ref(false)
const editPortfolioForm = reactive({ name: '', description: '', cover_image: '' })
const editPortfolioRules: FormRules = {
  name: [{ required: true, message: '请输入作品集名称', trigger: 'blur' }],
}

async function handleCoverUpload(file: File) {
  if (!editingPortfolio.value) return false
  coverUploading.value = true
  try {
    const res: any = await portfolioApi.uploadCover(editingPortfolio.value.id, file)
    editPortfolioForm.cover_image = res.file_path || res.url || ''
    ElMessage.success('封面已上传')
  } catch {
    ElMessage.error('封面上传失败')
  } finally {
    coverUploading.value = false
  }
  return false // prevent el-upload default behavior
}

async function handleEditPortfolioSubmit() {
  if (!editPortfolioFormRef.value || !editingPortfolio.value) return
  await editPortfolioFormRef.value.validate(async (valid) => {
    if (!valid) return
    editPortfolioSubmitting.value = true
    try {
      await portfolioApi.update(editingPortfolio.value!.id, {
        name: editPortfolioForm.name,
        description: editPortfolioForm.description,
        cover_image: editPortfolioForm.cover_image,
      })
      showEditPortfolioDialog.value = false
      ElMessage.success('作品集已更新')
      await fetchPortfolios()
    } catch {
      ElMessage.error('更新失败')
    } finally {
      editPortfolioSubmitting.value = false
    }
  })
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString()
}

// 打开编辑弹窗，用当前空间数据填充
function handleEdit() {
  if (workspace.value) {
    editForm.name = workspace.value.name
    editForm.description = workspace.value.description || ''
  }
  showEditDialog.value = true
}

// 提交编辑
async function handleEditSubmit() {
  if (!editFormRef.value) return
  await editFormRef.value.validate(async (valid) => {
    if (!valid) return
    editSubmitting.value = true
    try {
      await workspaceStore.updateWorkspace(Number(props.id), {
        name: editForm.name,
        description: editForm.description,
      })
      // 同步本地页面数据
      workspace.value = { ...workspace.value, name: editForm.name, description: editForm.description }
      showEditDialog.value = false
      ElMessage.success('空间信息已更新')
    } catch {
      ElMessage.error('更新失败')
    } finally {
      editSubmitting.value = false
    }
  })
}
</script>

<style scoped lang="scss">
.workspace-detail { width: 100%; max-width: 1200px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 32px; }
.page-title-row { display: flex; flex-wrap: wrap; align-items: center; }
.page-title { font-size: 28px; font-weight: 700; color: var(--color-text-primary); }
.page-desc { font-size: 14px; color: var(--color-text-secondary); margin-top: 4px; }
.header-actions { display: flex; gap: 12px; }
.portfolio-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 24px; }
.portfolio-card {
  cursor: pointer;
  &__cover {
    margin: -16px -16px 12px; height: 120px; overflow: hidden; border-radius: 8px 8px 0 0;
    img { width: 100%; height: 100%; object-fit: cover; }
  }
  &__header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;
    h3 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); }
  }
  &__desc { font-size: 13px; color: var(--color-text-secondary); min-height: 36px; margin-bottom: 12px; }
  &__footer { display: flex; justify-content: space-between; align-items: center; padding-top: 12px; border-top: 1px solid var(--border-glow); font-size: 12px; color: var(--color-text-muted); }
}
.el-dropdown-link { cursor: pointer; padding: 4px 8px; }
.cover-upload { display: flex; align-items: center; gap: 12px; }
.cover-preview {
  position: relative; width: 120px; height: 80px; border-radius: 6px; overflow: hidden; border: 1px solid var(--border-glow);
  img { width: 100%; height: 100%; object-fit: cover; }
}
.cover-remove { position: absolute; top: 4px; right: 4px; }
.cover-trigger {
  width: 120px; height: 80px; border: 1px dashed var(--border-glow); border-radius: 6px;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px;
  cursor: pointer; color: var(--color-text-muted); font-size: 12px;
  &:hover { border-color: var(--color-primary); color: var(--color-primary); }
}
.cover-loading { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-text-muted); }
</style>
