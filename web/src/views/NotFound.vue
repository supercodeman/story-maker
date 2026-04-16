<!-- web/src/views/NotFound.vue -->
<template>
  <div class="not-found">
    <h1>404</h1>
    <p>页面未找到</p>
    <router-link to="/workspaces" class="back-link">返回工作空间</router-link>
  </div>
</template>

<style scoped lang="scss">
.not-found {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  height: 100vh; background: var(--color-bg-deep);
  h1 { font-size: 72px; font-weight: 700; color: var(--color-primary); margin-bottom: 8px; }
  p { font-size: 18px; color: var(--color-text-secondary); margin-bottom: 24px; }
  .back-link { color: var(--color-primary); text-decoration: none; font-size: 14px;
    &:hover { color: var(--color-primary-light); }
  }
}
</style>
