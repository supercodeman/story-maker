<!-- web/src/components/common/GlowCard.vue -->
<template>
  <div
    class="glow-card"
    :class="{ 'glow-card--hover': hoverable, 'glow-card--active': active }"
    :style="{ '--glow-color': glowColor }"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  hoverable?: boolean
  active?: boolean
  glowColor?: string
}>()
</script>

<style scoped lang="scss">
.glow-card {
  background-color: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  border-radius: 12px;
  padding: 20px;
  box-shadow: var(--shadow-glow, none), var(--shadow-sm);
  transition: all 0.3s ease;

  &--hover {
    cursor: pointer;

    &:hover {
      border-color: var(--color-primary);
      box-shadow: var(--shadow-md);
      transform: translateY(-2px);
    }
  }

  &--active {
    border-color: var(--color-primary);
    box-shadow: var(--shadow-md);
  }
}
</style>
