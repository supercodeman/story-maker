<!-- web/src/views/workspace/MemberManage.vue -->
<template>
  <div class="member-manage">
    <div v-loading="loading" class="member-list">
      <div v-for="m in members" :key="m.id" class="member-item">
        <div class="member-info">
          <span class="member-name">{{ m.username || `用户 #${m.user_id}` }}</span>
          <el-tag :type="m.role === 'owner' ? 'warning' : 'info'" size="small">{{ m.role }}</el-tag>
        </div>
        <el-button
          v-if="m.role !== 'owner'"
          size="small"
          type="danger"
          @click="handleRemove(m.user_id)"
        >移除</el-button>
      </div>
    </div>

    <el-divider />

    <div class="add-member">
      <el-input v-model="newUserId" placeholder="用户 ID" style="width: 200px" />
      <el-select v-model="newRole" style="width: 120px">
        <el-option label="编辑者" value="editor" />
        <el-option label="查看者" value="viewer" />
      </el-select>
      <el-button type="primary" @click="handleAdd">添加</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { workspaceApi } from '@/api/workspace'
import type { WorkspaceMember } from '@/api/workspace'

const props = defineProps<{ workspaceId: number }>()

const members = ref<WorkspaceMember[]>([])
const loading = ref(false)
const newUserId = ref('')
const newRole = ref('editor')

onMounted(() => fetchMembers())

async function fetchMembers() {
  loading.value = true
  try {
    const data: any = await workspaceApi.getMembers(props.workspaceId)
    members.value = Array.isArray(data) ? data : data.items || []
  } finally { loading.value = false }
}

async function handleAdd() {
  const uid = Number(newUserId.value)
  if (!uid) { ElMessage.warning('请输入有效的用户 ID'); return }
  await workspaceApi.addMember(props.workspaceId, { user_id: uid, role: newRole.value })
  ElMessage.success('成员已添加')
  newUserId.value = ''
  await fetchMembers()
}

async function handleRemove(userId: number) {
  await workspaceApi.removeMember(props.workspaceId, userId)
  ElMessage.success('成员已移除')
  await fetchMembers()
}
</script>

<style scoped lang="scss">
.member-list { display: flex; flex-direction: column; gap: 8px; }
.member-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; }
.member-info { display: flex; align-items: center; gap: 8px; }
.member-name { font-size: 14px; color: var(--color-text-primary); }
.add-member { display: flex; gap: 8px; align-items: center; }
</style>
