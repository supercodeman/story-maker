<!-- web/src/components/layout/AppSidebar.vue -->
<template>
  <aside class="app-sidebar" :class="{ 'app-sidebar--collapsed': collapsed }">
    <nav class="sidebar-nav">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :class="{ 'nav-item--active': isActive(item.path) }"
        :title="collapsed ? item.label : undefined"
      >
        <span class="nav-item__icon">{{ item.icon }}</span>
        <span v-show="!collapsed" class="nav-item__text">{{ item.label }}</span>
      </router-link>
    </nav>

    <div class="sidebar-footer">
      <router-link to="/settings/apikeys" class="nav-item" :title="collapsed ? '设置' : undefined">
        <span class="nav-item__icon">⚙️</span>
        <span v-show="!collapsed" class="nav-item__text">设置</span>
      </router-link>
      <div class="nav-item" @click="toggleTheme" :title="collapsed ? (theme === 'dark' ? '切换浅色' : '切换深色') : undefined">
        <span class="nav-item__icon">{{ theme === 'dark' ? '☀️' : '🌙' }}</span>
        <span v-show="!collapsed" class="nav-item__text">{{ theme === 'dark' ? '浅色模式' : '深色模式' }}</span>
      </div>
      <div class="nav-item toggle-btn" @click="collapsed = !collapsed" :title="collapsed ? '展开侧边栏' : '收起侧边栏'">
        <span class="nav-item__icon toggle-icon" :class="{ 'toggle-icon--collapsed': collapsed }">‹</span>
        <span v-show="!collapsed" class="nav-item__text">收起</span>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspaceStore } from '@/store/workspace'
import { useUserStore } from '@/store/user'
import { useTheme } from '@/composables/useTheme'

const route = useRoute()
const workspaceStore = useWorkspaceStore()
const userStore = useUserStore()
const collapsed = ref(false)
const { theme, toggleTheme } = useTheme()

const currentWorkspaceId = computed(() => workspaceStore.currentWorkspace?.id)

const navItems = computed(() => {
  const wsId = currentWorkspaceId.value
  if (!wsId) return []

  const items = [
    {
      path: '/workspaces',
      icon: '🏠',
      label: '工作空间',
    },
    {
      path: `/workspace/${wsId}`,
      icon: '📁',
      label: '作品集',
    },
    {
      path: '/memories',
      icon: '🧠',
      label: '我的记忆',
    },
    {
      path: '/my-styles',
      icon: '🎨',
      label: '风格库',
    },
    {
      path: '/market',
      icon: '🏪',
      label: '记忆市场',
    },
    {
      path: '/wallet',
      icon: '💰',
      label: '钱包',
    },
    {
      path: '/writer-level',
      icon: '✨',
      label: '写手等级',
    },
  ]

  // 管理员追加审核入口
  if (userStore.profile?.role === 'admin') {
    items.push({
      path: '/admin/memory-review',
      icon: '🛡️',
      label: '记忆审核',
    })
    items.push({
      path: '/admin/users',
      icon: '👥',
      label: '用户管理',
    })
    items.push({
      path: '/admin/model-debug',
      icon: '🔧',
      label: '模型调试',
    })
  }

  return items
})

function isActive(path: string) {
  return route.path === path || route.path.startsWith(path + '/')
}
</script>

<style scoped lang="scss">
.app-sidebar {
  width: 240px;
  height: 100%;
  background-color: var(--color-bg-surface);
  border-right: 1px solid var(--border-glow);
  display: flex;
  flex-direction: column;
  padding: 16px 0;
  transition: width 0.25s ease, background-color 0.3s ease, border-color 0.3s ease;
  overflow: hidden;
  flex-shrink: 0;

  &--collapsed {
    width: 60px;

    .sidebar-nav,
    .sidebar-footer {
      padding: 0 4px;
    }

    .nav-item {
      justify-content: center;
      padding: 8px;
    }
  }
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 0 12px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  border-radius: 8px;
  color: var(--color-text-secondary);
  text-decoration: none;
  transition: all 0.2s ease;
  cursor: pointer;
  white-space: nowrap;

  &__icon {
    font-size: 20px;
    flex-shrink: 0;
  }

  &__text {
    font-size: 14px;
    font-weight: 500;
  }

  &:hover {
    background-color: var(--color-bg-hover);
    color: var(--color-text-primary);
  }

  &--active {
    background-color: var(--color-bg-hover);
    color: var(--color-primary);
    box-shadow: 0 0 10px rgba(124, 140, 248, 0.1);
  }
}

.sidebar-footer {
  padding: 0 12px;
  border-top: 1px solid var(--border-glow);
  padding-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.toggle-btn {
  user-select: none;
}

.toggle-icon {
  display: inline-block;
  transition: transform 0.25s ease;

  &--collapsed {
    transform: rotate(180deg);
  }
}
</style>
