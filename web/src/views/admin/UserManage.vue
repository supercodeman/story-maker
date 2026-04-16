<!-- web/src/views/admin/UserManage.vue -->
<template>
  <div class="user-manage">
    <div class="page-header">
      <h1 class="page-title">用户管理</h1>
      <p class="page-desc">管理系统用户及角色分配</p>
    </div>

    <FadePanel class="table-panel">
      <div v-loading="loading" class="custom-table">
        <div class="custom-table__header">
          <span class="col col--id">ID</span>
          <span class="col col--name">用户名</span>
          <span class="col col--email">邮箱</span>
          <span class="col col--role">角色</span>
          <span class="col col--level">写手等级</span>
          <span class="col col--time">注册时间</span>
        </div>
        <div v-for="user in users" :key="user.id" class="custom-table__row">
          <span class="col col--id">{{ user.id }}</span>
          <span class="col col--name">
            <span class="user-avatar">{{ user.username.charAt(0).toUpperCase() }}</span>
            {{ user.username }}
          </span>
          <span class="col col--email">{{ user.email }}</span>
          <span class="col col--role">
            <select
              class="inline-select"
              :class="`inline-select--${user.role}`"
              :value="user.role"
              :disabled="user.id === currentUserId"
              @change="(e) => handleRoleChange(user, (e.target as HTMLSelectElement).value)"
            >
              <option value="admin">Admin</option>
              <option value="creator">Creator</option>
              <option value="viewer">Viewer</option>
            </select>
          </span>
          <span class="col col--level">
            <select
              class="inline-select"
              :class="`inline-select--${user.writer_level || 'beginner'}`"
              :value="user.writer_level || 'beginner'"
              @change="(e) => handleWriterLevelChange(user, (e.target as HTMLSelectElement).value)"
            >
              <option value="beginner">小白写手</option>
              <option value="advanced">大神写手</option>
            </select>
          </span>
          <span class="col col--time">{{ formatTime(user.created_at) }}</span>
        </div>
        <div v-if="!loading && users.length === 0" class="custom-table__empty">
          暂无用户数据
        </div>
      </div>
    </FadePanel>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userAdminApi } from '@/api/user'
import { useUserStore } from '@/store/user'
import FadePanel from '@/components/common/FadePanel.vue'

interface UserItem {
  id: number
  username: string
  email: string
  role: string
  writer_level: string
  created_at: string
}

const userStore = useUserStore()
const currentUserId = computed(() => userStore.profile?.id)
const users = ref<UserItem[]>([])
const loading = ref(false)

onMounted(() => loadUsers())

async function loadUsers() {
  loading.value = true
  try {
    const data: any = await userAdminApi.listUsers()
    users.value = Array.isArray(data) ? data : []
  } finally {
    loading.value = false
  }
}

async function handleRoleChange(user: UserItem, newRole: string) {
  const oldRole = user.role
  try {
    await ElMessageBox.confirm(
      `确定将用户「${user.username}」的角色从 ${oldRole} 修改为 ${newRole}？`,
      '确认修改角色'
    )
    await userAdminApi.updateRole(user.id, { role: newRole })
    user.role = newRole
    ElMessage.success('角色更新成功')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '操作失败')
    }
  }
}

async function handleWriterLevelChange(user: UserItem, newLevel: string) {
  const label = newLevel === 'advanced' ? '大神写手' : '小白写手'
  try {
    await ElMessageBox.confirm(
      `确定将用户「${user.username}」的写手等级设置为「${label}」？`,
      '确认修改写手等级'
    )
    await userAdminApi.setWriterLevel(user.id, { writer_level: newLevel })
    user.writer_level = newLevel
    ElMessage.success('写手等级更新成功')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '操作失败')
    }
  }
}

function formatTime(t: string) {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}
</script>

<style scoped lang="scss">
.user-manage {
  width: 100%;
  max-width: 1100px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.page-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.table-panel {
  padding: 0;
  overflow: hidden;
}

.custom-table {
  width: 100%;

  &__header {
    display: flex;
    align-items: center;
    padding: 14px 24px;
    background: var(--color-bg-hover);
    border-bottom: 1px solid var(--border-glow);

    .col {
      font-size: 12px;
      font-weight: 600;
      color: var(--color-text-muted);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
  }

  &__row {
    display: flex;
    align-items: center;
    padding: 14px 24px;
    border-bottom: 1px solid var(--border-light);
    transition: background 0.15s;

    &:hover {
      background: rgba(124, 140, 248, 0.03);
    }

    &:last-child {
      border-bottom: none;
    }

    .col {
      font-size: 13px;
      color: var(--color-text-secondary);
    }
  }

  &__empty {
    padding: 48px 24px;
    text-align: center;
    color: var(--color-text-muted);
    font-size: 14px;
  }
}

.col {
  &--id { width: 60px; }
  &--name { flex: 1; min-width: 120px; display: flex; align-items: center; gap: 10px; }
  &--email { flex: 1.5; min-width: 180px; }
  &--role { width: 130px; }
  &--level { width: 130px; }
  &--time { width: 170px; text-align: right; }
}

.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: white;
  flex-shrink: 0;
}

.inline-select {
  appearance: none;
  -webkit-appearance: none;
  padding: 5px 28px 5px 10px;
  border-radius: 6px;
  border: 1px solid var(--border-glow);
  background-color: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239299b0' stroke-width='2'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 8px center;

  &:hover:not(:disabled) {
    border-color: var(--color-primary);
    background-color: rgba(124, 140, 248, 0.05);
  }

  &:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  &--admin {
    color: var(--color-accent-amber);
    border-color: rgba(252, 211, 77, 0.2);
  }

  &--advanced {
    color: var(--color-accent-cyan);
    border-color: rgba(103, 232, 249, 0.2);
  }
}
</style>
