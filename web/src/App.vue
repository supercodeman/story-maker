<template>
  <router-view />
</template>

<script setup lang="ts">
import { useTheme } from '@/composables/useTheme'

// 初始化主题（确保 DOM 上的 class 与存储一致）
useTheme()
</script>
