<!-- web/src/components/common/FadePanel.vue -->
<template>
  <div class="fade-panel" :class="{ 'fade-panel--blur': blur }">
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  blur?: boolean
}>()
</script>

<style scoped lang="scss">
.fade-panel {
  background: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  border-radius: 12px;
  padding: 24px;
  box-shadow: var(--shadow-sm);
  transition: background 0.3s ease, border-color 0.3s ease;

  &--blur {
    backdrop-filter: blur(10px);
  }
}

// 暗色模式下保留渐变背景
html.dark .fade-panel {
  background: linear-gradient(
    135deg,
    rgba(35, 38, 64, 0.9) 0%,
    rgba(26, 29, 46, 0.8) 100%
  );

  &--blur {
    background: linear-gradient(
      135deg,
      rgba(35, 38, 64, 0.7) 0%,
      rgba(26, 29, 46, 0.6) 100%
    );
  }
}
</style>
