<!-- web/src/components/common/NeonButton.vue -->
<template>
  <button
    class="neon-button"
    :class="[`neon-button--${type}`, { 'neon-button--loading': loading }]"
    :disabled="disabled || loading"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="neon-button__spinner"></span>
    <slot v-else />
  </button>
</template>

<script setup lang="ts">
defineProps<{
  type?: 'primary' | 'success' | 'warning' | 'danger'
  loading?: boolean
  disabled?: boolean
}>()

defineEmits<{
  click: [event: MouseEvent]
}>()
</script>

<style scoped lang="scss">
.neon-button {
  position: relative;
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: linear-gradient(45deg, transparent, rgba(255, 255, 255, 0.1), transparent);
    transform: translateX(-100%);
    transition: transform 0.6s;
  }

  &:hover::before {
    transform: translateX(100%);
  }

  &--primary {
    background-color: var(--color-primary);
    color: white;
    box-shadow: var(--shadow-sm);

    &:hover:not(:disabled) {
      background-color: var(--color-primary-light);
      box-shadow: var(--shadow-md);
    }
  }

  &--success {
    background-color: var(--color-accent-green);
    color: #0F1117;
    box-shadow: var(--shadow-sm);
  }

  &--warning {
    background-color: var(--color-accent-amber);
    color: #0F1117;
    box-shadow: var(--shadow-sm);
  }

  &--danger {
    background-color: #EF4444;
    color: white;
    box-shadow: var(--shadow-sm);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &__spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
