<!-- web/src/components/layout/AppHeader.vue -->
<template>
  <header class="app-header">
    <div class="app-header__left">
      <div class="app-header__logo">
        <span class="logo-icon">✨</span>
        <span class="logo-text">Ai-Curton</span>
      </div>

      <el-select
        v-if="workspaces.length > 0"
        v-model="currentWorkspaceId"
        placeholder="选择工作空间"
        class="workspace-selector"
        @change="handleWorkspaceChange"
      >
        <el-option
          v-for="ws in workspaces"
          :key="ws.id"
          :label="ws.name"
          :value="ws.id"
        />
      </el-select>
    </div>

    <div class="app-header__right">
      <el-dropdown @command="handleUserCommand">
        <div class="user-avatar">
          <span>{{ userInitial }}</span>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item disabled>{{ userStore.profile?.username }}</el-dropdown-item>
            <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import { useWorkspaceStore } from '@/store/workspace'

const router = useRouter()
const userStore = useUserStore()
const workspaceStore = useWorkspaceStore()

const currentWorkspaceId = ref<number | null>(null)

const workspaces = computed(() => workspaceStore.workspaces)
const userInitial = computed(() => {
  const username = userStore.profile?.username || 'U'
  return username.charAt(0).toUpperCase()
})

onMounted(async () => {
  await workspaceStore.fetchWorkspaces()
  // 优先从 localStorage 恢复上次选中的空间
  const savedId = localStorage.getItem('currentWorkspaceId')
  const savedWs = savedId ? workspaces.value.find(w => w.id === Number(savedId)) : null
  if (savedWs) {
    currentWorkspaceId.value = savedWs.id
    workspaceStore.setCurrentWorkspace(savedWs)
  } else if (workspaceStore.currentWorkspace) {
    currentWorkspaceId.value = workspaceStore.currentWorkspace.id
  } else if (workspaces.value.length > 0) {
    currentWorkspaceId.value = workspaces.value[0].id
    workspaceStore.setCurrentWorkspace(workspaces.value[0])
  }
})

// 保持 dropdown 与 store 同步
watch(() => workspaceStore.currentWorkspace, (ws) => {
  currentWorkspaceId.value = ws?.id ?? null
})

// 当列表从空变为有值且没有 currentWorkspace 时，自动选中第一个
// 优先恢复 localStorage 中保存的空间
watch(workspaces, (list) => {
  if (list.length > 0 && !workspaceStore.currentWorkspace) {
    const savedId = localStorage.getItem('currentWorkspaceId')
    const savedWs = savedId ? list.find(w => w.id === Number(savedId)) : null
    const target = savedWs || list[0]
    currentWorkspaceId.value = target.id
    workspaceStore.setCurrentWorkspace(target)
  }
})

function handleWorkspaceChange(id: number) {
  const ws = workspaces.value.find((w) => w.id === id)
  if (ws) {
    workspaceStore.setCurrentWorkspace(ws)
    router.push(`/workspace/${id}`)
  }
}

function handleUserCommand(command: string) {
  if (command === 'logout') {
    userStore.logout()
  }
}
</script>

<style scoped lang="scss">
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  padding: 0 24px;
  background-color: var(--color-bg-surface);
  border-bottom: 1px solid var(--border-glow);

  &__left {
    display: flex;
    align-items: center;
    gap: 24px;
  }

  &__logo {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 20px;
    font-weight: 600;
    color: var(--color-text-primary);

    .logo-icon {
      font-size: 24px;
    }
  }

  &__right {
    display: flex;
    align-items: center;
    gap: 16px;
  }
}

.workspace-selector {
  width: 200px;
}

.user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 0 15px rgba(124, 140, 248, 0.4);
  }
}
</style>
