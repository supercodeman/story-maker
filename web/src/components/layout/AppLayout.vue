<!-- web/src/components/layout/AppLayout.vue -->
<template>
  <div class="app-layout">
    <AppHeader />
    <div class="app-layout__body">
      <AppSidebar />
      <main class="app-layout__main">
        <router-view :key="route.fullPath" />
      </main>
    </div>
    <div class="app-layout__footer">
      <span class="footer-status">
        <span class="status-dot" :class="{ 'status-dot--online': wsConnected }"></span>
        {{ wsConnected ? '已连接' : '未连接' }}
      </span>
      <span class="footer-info">Story-Maker v1.0.0</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from './AppHeader.vue'
import AppSidebar from './AppSidebar.vue'
import { useUserStore } from '@/store/user'
import { useAIStore } from '@/store/ai'

const route = useRoute()
const userStore = useUserStore()
const aiStore = useAIStore()

const wsConnected = computed(() => aiStore.wsConnected)

onMounted(async () => {
  if (!userStore.profile) {
    await userStore.fetchProfile()
  }
})
</script>

<style scoped lang="scss">
.app-layout {
  width: 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  &__body {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  &__main {
    flex: 1;
    overflow-y: auto;
    background-color: var(--color-bg-deep);
    padding: 24px;
  }

  &__footer {
    height: 32px;
    background-color: var(--color-bg-surface);
    border-top: 1px solid var(--border-glow);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 24px;
    font-size: 12px;
    color: var(--color-text-muted);
  }
}

.footer-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #6B7280;

  &--online {
    background-color: var(--color-accent-green);
    box-shadow: 0 0 8px var(--color-accent-green);
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
