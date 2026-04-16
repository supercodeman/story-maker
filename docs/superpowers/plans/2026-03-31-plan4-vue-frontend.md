# Plan 4: Vue 前端 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现完整的 Vue 3 前端，包含淡雅科技风主题、工作空间/作品集/角色/AI创作工坊等全部页面。

**Architecture:** Vue 3 SPA，Pinia 状态管理，Axios HTTP 客户端，WebSocket 实时通信，Tailwind CSS + Element Plus 组件库。

**Tech Stack:** Vue 3, TypeScript, Vite, Element Plus, Pinia, Tailwind CSS, Axios, WebSocket

---

### Task 1: Vue 项目初始化

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/tsconfig.node.json`
- Create: `web/tailwind.config.js`
- Create: `web/postcss.config.js`
- Create: `web/index.html`
- Create: `web/src/main.ts`
- Create: `web/src/App.vue`
- Create: `web/src/env.d.ts`

- [ ] **Step 1: 使用 Vite 创建 Vue 3 + TypeScript 项目**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
npm create vite@latest web -- --template vue-ts
```

- [ ] **Step 2: 安装核心依赖**

```bash
cd /Users/sangchenglong/tmp/Ai-curton/web
npm install element-plus pinia vue-router@4 axios
npm install -D tailwindcss@3 postcss autoprefixer sass @types/node
npx tailwindcss init -p
```

- [ ] **Step 3: 配置 `web/vite.config.ts`**

```ts
// web/vite.config.ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
```

- [ ] **Step 4: 配置 `web/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "preserve",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src/**/*.ts", "src/**/*.d.ts", "src/**/*.tsx", "src/**/*.vue"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

- [ ] **Step 5: 配置 `web/tsconfig.node.json`**

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 6: 配置 `web/tailwind.config.js`**

```js
// web/tailwind.config.js
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#7C8CF8',
          light: '#A5B4FC',
          dark: '#5B6AE0',
        },
        bg: {
          deep: '#0F1117',
          surface: '#1A1D2E',
          card: '#232640',
          hover: '#2A2E4A',
        },
        text: {
          primary: '#E8EAF6',
          secondary: '#9CA3C0',
          muted: '#5C6280',
        },
        accent: {
          cyan: '#67E8F9',
          green: '#6EE7B7',
          amber: '#FCD34D',
        },
      },
      boxShadow: {
        'glow': '0 0 15px rgba(124, 140, 248, 0.1)',
        'glow-md': '0 0 25px rgba(124, 140, 248, 0.2)',
        'glow-lg': '0 0 35px rgba(124, 140, 248, 0.3)',
      },
      animation: {
        'pulse-glow': 'pulse-glow 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        'pulse-glow': {
          '0%, 100%': { boxShadow: '0 0 15px rgba(124, 140, 248, 0.1)' },
          '50%': { boxShadow: '0 0 25px rgba(124, 140, 248, 0.3)' },
        },
      },
    },
  },
  plugins: [],
}
```

- [ ] **Step 7: 配置 `web/postcss.config.js`**

```js
// web/postcss.config.js
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

- [ ] **Step 8: 创建 `web/index.html`**

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Ai-Curton | AI 漫画创作工坊</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 9: 创建 `web/src/env.d.ts`**

```ts
// web/src/env.d.ts
/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
```

- [ ] **Step 10: 创建占位 `web/src/main.ts`（后续 Task 2 完善）**

```ts
// web/src/main.ts
import { createApp } from 'vue'
import App from './App.vue'

const app = createApp(App)
app.mount('#app')
```

- [ ] **Step 11: 创建占位 `web/src/App.vue`（后续 Task 7 完善）**

```vue
<!-- web/src/App.vue -->
<template>
  <div id="app">
    <h1>Ai-Curton</h1>
  </div>
</template>

<script setup lang="ts">
</script>

<style>
#app {
  font-family: 'Inter', sans-serif;
}
</style>
```

- [ ] **Step 12: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/
git commit -m "feat: init Vue 3 project with Vite, TypeScript, Tailwind CSS"
```

---

### Task 2: 主题与全局样式

**Files:**
- Create: `web/src/assets/styles/theme.scss`
- Create: `web/src/assets/styles/global.scss`
- Create: `web/src/assets/styles/animation.scss`

- [ ] **Step 1: 创建 `web/src/assets/styles/theme.scss`**

```scss
// web/src/assets/styles/theme.scss

// 色彩变量
:root {
  --color-primary: #7C8CF8;
  --color-primary-light: #A5B4FC;
  --color-primary-dark: #5B6AE0;

  --color-bg-deep: #0F1117;
  --color-bg-surface: #1A1D2E;
  --color-bg-card: #232640;
  --color-bg-hover: #2A2E4A;

  --color-text-primary: #E8EAF6;
  --color-text-secondary: #9CA3C0;
  --color-text-muted: #5C6280;

  --color-accent-cyan: #67E8F9;
  --color-accent-green: #6EE7B7;
  --color-accent-amber: #FCD34D;

  --border-glow: rgba(124, 140, 248, 0.2);

  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
}

// 字体导入（可选，如果使用 Google Fonts）
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
```

- [ ] **Step 2: 创建 `web/src/assets/styles/global.scss`**

```scss
// web/src/assets/styles/global.scss
@import './theme.scss';

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

body {
  font-family: var(--font-sans);
  background-color: var(--color-bg-deep);
  color: var(--color-text-primary);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

#app {
  width: 100%;
  height: 100%;
}

// Element Plus 暗色主题覆盖
.el-button {
  &.el-button--primary {
    background-color: var(--color-primary);
    border-color: var(--color-primary);

    &:hover {
      background-color: var(--color-primary-light);
      border-color: var(--color-primary-light);
    }
  }
}

.el-input__wrapper {
  background-color: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  box-shadow: none;

  &:hover {
    border-color: var(--color-primary);
  }

  &.is-focus {
    border-color: var(--color-primary);
    box-shadow: 0 0 10px rgba(124, 140, 248, 0.2);
  }
}

.el-input__inner {
  color: var(--color-text-primary);

  &::placeholder {
    color: var(--color-text-muted);
  }
}

.el-card {
  background-color: var(--color-bg-card);
  border: 1px solid var(--border-glow);
  box-shadow: var(--el-box-shadow-light);
}

.el-dialog {
  background-color: var(--color-bg-surface);
  border: 1px solid var(--border-glow);
}

.el-dropdown-menu {
  background-color: var(--color-bg-card);
  border: 1px solid var(--border-glow);
}

.el-dropdown-menu__item {
  color: var(--color-text-primary);

  &:hover {
    background-color: var(--color-bg-hover);
  }
}

// 滚动条样式
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--color-bg-surface);
}

::-webkit-scrollbar-thumb {
  background: var(--color-bg-hover);
  border-radius: 4px;

  &:hover {
    background: var(--color-primary-dark);
  }
}
```

- [ ] **Step 3: 创建 `web/src/assets/styles/animation.scss`**

```scss
// web/src/assets/styles/animation.scss

// 呼吸光效动画
@keyframes pulse-glow {
  0%, 100% {
    box-shadow: 0 0 15px rgba(124, 140, 248, 0.1);
  }
  50% {
    box-shadow: 0 0 25px rgba(124, 140, 248, 0.3);
  }
}

.pulse-glow {
  animation: pulse-glow 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

// 渐入动画
@keyframes fade-in {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.fade-in {
  animation: fade-in 0.3s ease-out;
}

// 渐入（从上）
@keyframes fade-in-down {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.fade-in-down {
  animation: fade-in-down 0.3s ease-out;
}

// 缩放渐入
@keyframes scale-in {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.scale-in {
  animation: scale-in 0.2s ease-out;
}

// 毛玻璃效果
.glass-effect {
  backdrop-filter: blur(10px);
  background-color: rgba(35, 38, 64, 0.7);
}
```

- [ ] **Step 4: 更新 `web/src/main.ts` 引入样式**

```ts
// web/src/main.ts
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './assets/styles/theme.scss'
import './assets/styles/global.scss'
import './assets/styles/animation.scss'
import App from './App.vue'

const app = createApp(App)
app.use(ElementPlus)
app.mount('#app')
```

- [ ] **Step 5: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/assets/styles/ web/src/main.ts
git commit -m "feat: add theme system with dark mode and glow animations"
```

---

### Task 3: Axios 封装

**Files:**
- Create: `web/src/api/request.ts`
- Create: `web/src/api/types.ts`

- [ ] **Step 1: 创建 `web/src/api/types.ts`**

```ts
// web/src/api/types.ts

// 通用 API 响应结构
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// 分页响应
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

// 错误响应
export interface ApiError {
  code: number
  message: string
  details?: any
}
```

- [ ] **Step 2: 创建 `web/src/api/request.ts`**

```ts
// web/src/api/request.ts
import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, AxiosError } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse, ApiError } from './types'

// 创建 axios 实例
const request: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器：自动附加 JWT token
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器：统一错误处理
request.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const { code, message, data } = response.data

    // 后端返回 code !== 0 视为业务错误
    if (code !== 0) {
      ElMessage.error(message || 'Request failed')
      return Promise.reject(new Error(message || 'Request failed'))
    }

    return data
  },
  (error: AxiosError<ApiError>) => {
    // 401 未授权，跳转登录
    if (error.response?.status === 401) {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      ElMessage.error('Session expired, please login again')
      window.location.href = '/login'
      return Promise.reject(error)
    }

    // 403 无权限
    if (error.response?.status === 403) {
      ElMessage.error('Permission denied')
      return Promise.reject(error)
    }

    // 其他错误
    const message = error.response?.data?.message || error.message || 'Network error'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export default request
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/api/
git commit -m "feat: add axios wrapper with JWT interceptor and error handling"
```

---

### Task 4: 路由配置

**Files:**
- Create: `web/src/router/index.ts`

- [ ] **Step 1: 创建 `web/src/router/index.ts`**

```ts
// web/src/router/index.ts
import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/Login.vue'),
    meta: { requiresAuth: false, title: 'Login' },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/auth/Register.vue'),
    meta: { requiresAuth: false, title: 'Register' },
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppLayout.vue'),
    meta: { requiresAuth: true },
    redirect: '/workspaces',
    children: [
      {
        path: 'workspaces',
        name: 'WorkspaceList',
        component: () => import('@/views/workspace/WorkspaceList.vue'),
        meta: { title: 'Workspaces' },
      },
      {
        path: 'workspace/:id',
        name: 'WorkspaceDetail',
        component: () => import('@/views/workspace/WorkspaceDetail.vue'),
        meta: { title: 'Workspace' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid',
        name: 'PortfolioDetail',
        component: () => import('@/views/portfolio/PortfolioDetail.vue'),
        meta: { title: 'Portfolio' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid/characters',
        name: 'CharacterList',
        component: () => import('@/views/character/CharacterList.vue'),
        meta: { title: 'Characters' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid/studio',
        name: 'AIStudio',
        component: () => import('@/views/studio/AIStudio.vue'),
        meta: { title: 'AI Studio' },
        props: true,
      },
      {
        path: 'settings/apikeys',
        name: 'APIKeyManage',
        component: () => import('@/views/settings/APIKeyManage.vue'),
        meta: { title: 'API Keys' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/workspaces',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫：未登录跳转 /login
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('access_token')
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth !== false)

  if (requiresAuth && !token) {
    next({ path: '/login', query: { redirect: to.fullPath } })
  } else if ((to.path === '/login' || to.path === '/register') && token) {
    next({ path: '/workspaces' })
  } else {
    next()
  }

  // 设置页面标题
  if (to.meta.title) {
    document.title = `${to.meta.title} | Ai-Curton`
  }
})

export default router
```

- [ ] **Step 2: 更新 `web/src/main.ts` 注册路由**

```ts
// web/src/main.ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './assets/styles/theme.scss'
import './assets/styles/global.scss'
import './assets/styles/animation.scss'
import App from './App.vue'
import router from './router'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus)
app.mount('#app')
```

- [ ] **Step 3: 更新 `web/src/App.vue` 使用 RouterView**

```vue
<!-- web/src/App.vue -->
<template>
  <router-view />
</template>

<script setup lang="ts">
</script>

<style>
html.dark {
  color-scheme: dark;
}
</style>
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/router/ web/src/main.ts web/src/App.vue
git commit -m "feat: add vue-router with auth guard and nested routes"
```

---

### Task 5: Pinia Store

**Files:**
- Create: `web/src/store/user.ts`
- Create: `web/src/store/workspace.ts`
- Create: `web/src/store/ai.ts`

- [ ] **Step 1: 创建 `web/src/store/user.ts`**

```ts
// web/src/store/user.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import request from '@/api/request'

export interface UserProfile {
  id: number
  username: string
  email: string
  role: string
  created_at: string
}

export interface LoginPayload {
  email: string
  password: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
}

export const useUserStore = defineStore('user', () => {
  const profile = ref<UserProfile | null>(null)
  const isLoggedIn = ref(!!localStorage.getItem('access_token'))

  // 登录
  async function login(payload: LoginPayload) {
    const data: any = await request.post('/auth/login', payload)
    localStorage.setItem('access_token', data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
    isLoggedIn.value = true
    await fetchProfile()
  }

  // 注册
  async function register(payload: RegisterPayload) {
    await request.post('/auth/register', payload)
  }

  // 获取用户信息
  async function fetchProfile() {
    const data: any = await request.get('/user/profile')
    profile.value = data
  }

  // 登出
  function logout() {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    profile.value = null
    isLoggedIn.value = false
    window.location.href = '/login'
  }

  return {
    profile,
    isLoggedIn,
    login,
    register,
    fetchProfile,
    logout,
  }
})
```

- [ ] **Step 2: 创建 `web/src/store/workspace.ts`**

```ts
// web/src/store/workspace.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import request from '@/api/request'

export interface Workspace {
  id: number
  name: string
  type: string
  owner_id: number
  description: string
  created_at: string
  updated_at: string
}

export const useWorkspaceStore = defineStore('workspace', () => {
  const workspaces = ref<Workspace[]>([])
  const currentWorkspace = ref<Workspace | null>(null)
  const loading = ref(false)

  // 获取工作空间列表
  async function fetchWorkspaces() {
    loading.value = true
    try {
      const data: any = await request.get('/workspaces')
      workspaces.value = data.items || data || []
    } finally {
      loading.value = false
    }
  }

  // 获取单个工作空间
  async function fetchWorkspace(id: number) {
    loading.value = true
    try {
      const data: any = await request.get(`/workspaces/${id}`)
      currentWorkspace.value = data
    } finally {
      loading.value = false
    }
  }

  // 设置当前工作空间
  function setCurrentWorkspace(ws: Workspace) {
    currentWorkspace.value = ws
  }

  return {
    workspaces,
    currentWorkspace,
    loading,
    fetchWorkspaces,
    fetchWorkspace,
    setCurrentWorkspace,
  }
})
```

- [ ] **Step 3: 创建 `web/src/store/ai.ts`**

```ts
// web/src/store/ai.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface AITask {
  id: string
  type: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  progress: number
  result_url?: string
  error?: string
  created_at: string
}

export const useAIStore = defineStore('ai', () => {
  const tasks = ref<AITask[]>([])
  const wsConnected = ref(false)

  const pendingTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'pending' || t.status === 'running')
  )

  const completedTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'completed')
  )

  // 添加任务
  function addTask(task: AITask) {
    tasks.value.unshift(task)
  }

  // 更新任务状态（来自 WebSocket 推送）
  function updateTask(taskId: string, updates: Partial<AITask>) {
    const idx = tasks.value.findIndex((t) => t.id === taskId)
    if (idx !== -1) {
      tasks.value[idx] = { ...tasks.value[idx], ...updates }
    }
  }

  // 设置 WebSocket 连接状态
  function setWsConnected(connected: boolean) {
    wsConnected.value = connected
  }

  // 清空已完成任务
  function clearCompleted() {
    tasks.value = tasks.value.filter(
      (t) => t.status === 'pending' || t.status === 'running'
    )
  }

  return {
    tasks,
    wsConnected,
    pendingTasks,
    completedTasks,
    addTask,
    updateTask,
    setWsConnected,
    clearCompleted,
  }
})
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/store/
git commit -m "feat: add Pinia stores for user, workspace, and AI tasks"
```

---

### Task 6: 通用组件

**Files:**
- Create: `web/src/components/common/GlowCard.vue`
- Create: `web/src/components/common/NeonButton.vue`
- Create: `web/src/components/common/FadePanel.vue`

- [ ] **Step 1: 创建 `web/src/components/common/GlowCard.vue`**

```vue
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
  box-shadow: 0 0 15px rgba(124, 140, 248, 0.1);
  transition: all 0.3s ease;

  &--hover {
    cursor: pointer;

    &:hover {
      border-color: var(--color-primary);
      box-shadow: 0 0 25px var(--glow-color, rgba(124, 140, 248, 0.2));
      transform: translateY(-2px);
    }
  }

  &--active {
    border-color: var(--color-primary);
    box-shadow: 0 0 25px var(--glow-color, rgba(124, 140, 248, 0.3));
  }
}
</style>
```

- [ ] **Step 2: 创建 `web/src/components/common/NeonButton.vue`**

```vue
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
    box-shadow: 0 0 15px rgba(124, 140, 248, 0.3);

    &:hover:not(:disabled) {
      background-color: var(--color-primary-light);
      box-shadow: 0 0 25px rgba(124, 140, 248, 0.5);
    }
  }

  &--success {
    background-color: var(--color-accent-green);
    color: #0F1117;
    box-shadow: 0 0 15px rgba(110, 231, 183, 0.3);

    &:hover:not(:disabled) {
      box-shadow: 0 0 25px rgba(110, 231, 183, 0.5);
    }
  }

  &--warning {
    background-color: var(--color-accent-amber);
    color: #0F1117;
    box-shadow: 0 0 15px rgba(252, 211, 77, 0.3);

    &:hover:not(:disabled) {
      box-shadow: 0 0 25px rgba(252, 211, 77, 0.5);
    }
  }

  &--danger {
    background-color: #EF4444;
    color: white;
    box-shadow: 0 0 15px rgba(239, 68, 68, 0.3);

    &:hover:not(:disabled) {
      box-shadow: 0 0 25px rgba(239, 68, 68, 0.5);
    }
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &--loading {
    pointer-events: none;
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
```

- [ ] **Step 3: 创建 `web/src/components/common/FadePanel.vue`**

```vue
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
  background: linear-gradient(
    135deg,
    rgba(35, 38, 64, 0.9) 0%,
    rgba(26, 29, 46, 0.8) 100%
  );
  border: 1px solid var(--border-glow);
  border-radius: 12px;
  padding: 24px;

  &--blur {
    backdrop-filter: blur(10px);
    background: linear-gradient(
      135deg,
      rgba(35, 38, 64, 0.7) 0%,
      rgba(26, 29, 46, 0.6) 100%
    );
  }
}
</style>
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/components/common/
git commit -m "feat: add reusable UI components (GlowCard, NeonButton, FadePanel)"
```

---

### Task 7: 布局组件

**Files:**
- Create: `web/src/components/layout/AppHeader.vue`
- Create: `web/src/components/layout/AppSidebar.vue`
- Create: `web/src/components/layout/AppLayout.vue`

- [ ] **Step 1: 创建 `web/src/components/layout/AppHeader.vue`**

```vue
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
        placeholder="Select Workspace"
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
            <el-dropdown-item divided command="logout">Logout</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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
  if (workspaceStore.currentWorkspace) {
    currentWorkspaceId.value = workspaceStore.currentWorkspace.id
  } else if (workspaces.value.length > 0) {
    currentWorkspaceId.value = workspaces.value[0].id
    workspaceStore.setCurrentWorkspace(workspaces.value[0])
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
```

- [ ] **Step 2: 创建 `web/src/components/layout/AppSidebar.vue`**

```vue
<!-- web/src/components/layout/AppSidebar.vue -->
<template>
  <aside class="app-sidebar">
    <nav class="sidebar-nav">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :class="{ 'nav-item--active': isActive(item.path) }"
      >
        <span class="nav-item__icon">{{ item.icon }}</span>
        <span class="nav-item__text">{{ item.label }}</span>
      </router-link>
    </nav>

    <div class="sidebar-footer">
      <router-link to="/settings/apikeys" class="nav-item">
        <span class="nav-item__icon">⚙️</span>
        <span class="nav-item__text">Settings</span>
      </router-link>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspaceStore } from '@/store/workspace'

const route = useRoute()
const workspaceStore = useWorkspaceStore()

const currentWorkspaceId = computed(() => workspaceStore.currentWorkspace?.id)

const navItems = computed(() => {
  const wsId = currentWorkspaceId.value
  if (!wsId) return []

  return [
    {
      path: '/workspaces',
      icon: '🏠',
      label: 'Workspaces',
    },
    {
      path: `/workspace/${wsId}`,
      icon: '📁',
      label: 'Portfolios',
    },
  ]
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
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 0 12px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 8px;
  color: var(--color-text-secondary);
  text-decoration: none;
  transition: all 0.2s ease;
  cursor: pointer;

  &__icon {
    font-size: 20px;
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
}
</style>
```

- [ ] **Step 3: 创建 `web/src/components/layout/AppLayout.vue`**

```vue
<!-- web/src/components/layout/AppLayout.vue -->
<template>
  <div class="app-layout">
    <AppHeader />
    <div class="app-layout__body">
      <AppSidebar />
      <main class="app-layout__main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
    <div class="app-layout__footer">
      <span class="footer-status">
        <span class="status-dot" :class="{ 'status-dot--online': wsConnected }"></span>
        {{ wsConnected ? 'Connected' : 'Disconnected' }}
      </span>
      <span class="footer-info">Ai-Curton v1.0.0</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import AppHeader from './AppHeader.vue'
import AppSidebar from './AppSidebar.vue'
import { useUserStore } from '@/store/user'
import { useAIStore } from '@/store/ai'

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
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/components/layout/
git commit -m "feat: add app layout with header, sidebar, and footer"
```

---

### Task 8: 登录/注册页

**Files:**
- Create: `web/src/views/auth/Login.vue`
- Create: `web/src/views/auth/Register.vue`

- [ ] **Step 1: 创建 `web/src/views/auth/Login.vue`**

```vue
<!-- web/src/views/auth/Login.vue -->
<template>
  <div class="auth-page">
    <div class="auth-container">
      <GlowCard class="auth-card">
        <div class="auth-header">
          <h1 class="auth-title">✨ Ai-Curton</h1>
          <p class="auth-subtitle">AI 漫画创作工坊</p>
        </div>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          class="auth-form"
          @submit.prevent="handleSubmit"
        >
          <el-form-item prop="email">
            <el-input
              v-model="form.email"
              placeholder="Email"
              size="large"
              prefix-icon="Message"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="Password"
              size="large"
              prefix-icon="Lock"
              show-password
            />
          </el-form-item>

          <NeonButton
            type="primary"
            :loading="loading"
            class="auth-submit"
            @click="handleSubmit"
          >
            Login
          </NeonButton>
        </el-form>

        <div class="auth-footer">
          <span>Don't have an account?</span>
          <router-link to="/register" class="auth-link">Register</router-link>
        </div>
      </GlowCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { useUserStore } from '@/store/user'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  email: '',
  password: '',
})

const rules: FormRules = {
  email: [
    { required: true, message: 'Please enter email', trigger: 'blur' },
    { type: 'email', message: 'Invalid email format', trigger: 'blur' },
  ],
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' },
  ],
}

async function handleSubmit() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      await userStore.login(form)
      ElMessage.success('Login successful')
      router.push('/workspaces')
    } catch (error: any) {
      ElMessage.error(error.message || 'Login failed')
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped lang="scss">
.auth-page {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-bg-deep) 0%, #1a1a2e 100%);
  position: relative;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    width: 500px;
    height: 500px;
    background: radial-gradient(circle, rgba(124, 140, 248, 0.1) 0%, transparent 70%);
    top: -200px;
    right: -200px;
    animation: pulse-glow 3s ease-in-out infinite;
  }
}

.auth-container {
  width: 100%;
  max-width: 420px;
  padding: 24px;
  z-index: 1;
}

.auth-card {
  padding: 40px;
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;
}

.auth-title {
  font-size: 32px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}

.auth-subtitle {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.auth-form {
  margin-bottom: 24px;
}

.auth-submit {
  width: 100%;
  height: 44px;
  font-size: 16px;
  margin-top: 8px;
}

.auth-footer {
  text-align: center;
  font-size: 14px;
  color: var(--color-text-secondary);

  .auth-link {
    color: var(--color-primary);
    text-decoration: none;
    margin-left: 8px;
    font-weight: 500;

    &:hover {
      color: var(--color-primary-light);
    }
  }
}
</style>
```

- [ ] **Step 2: 创建 `web/src/views/auth/Register.vue`**

```vue
<!-- web/src/views/auth/Register.vue -->
<template>
  <div class="auth-page">
    <div class="auth-container">
      <GlowCard class="auth-card">
        <div class="auth-header">
          <h1 class="auth-title">✨ Ai-Curton</h1>
          <p class="auth-subtitle">Create your account</p>
        </div>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          class="auth-form"
          @submit.prevent="handleSubmit"
        >
          <el-form-item prop="username">
            <el-input
              v-model="form.username"
              placeholder="Username"
              size="large"
              prefix-icon="User"
            />
          </el-form-item>

          <el-form-item prop="email">
            <el-input
              v-model="form.email"
              placeholder="Email"
              size="large"
              prefix-icon="Message"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="Password"
              size="large"
              prefix-icon="Lock"
              show-password
            />
          </el-form-item>

          <el-form-item prop="confirmPassword">
            <el-input
              v-model="form.confirmPassword"
              type="password"
              placeholder="Confirm Password"
              size="large"
              prefix-icon="Lock"
              show-password
            />
          </el-form-item>

          <NeonButton
            type="primary"
            :loading="loading"
            class="auth-submit"
            @click="handleSubmit"
          >
            Register
          </NeonButton>
        </el-form>

        <div class="auth-footer">
          <span>Already have an account?</span>
          <router-link to="/login" class="auth-link">Login</router-link>
        </div>
      </GlowCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { useUserStore } from '@/store/user'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
})

const validateConfirmPassword = (_rule: any, value: string, callback: any) => {
  if (value !== form.password) {
    callback(new Error('Passwords do not match'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  username: [
    { required: true, message: 'Please enter username', trigger: 'blur' },
    { min: 3, max: 50, message: 'Username must be 3-50 characters', trigger: 'blur' },
  ],
  email: [
    { required: true, message: 'Please enter email', trigger: 'blur' },
    { type: 'email', message: 'Invalid email format', trigger: 'blur' },
  ],
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: 'Please confirm password', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' },
  ],
}

async function handleSubmit() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      await userStore.register({
        username: form.username,
        email: form.email,
        password: form.password,
      })
      ElMessage.success('Registration successful, please login')
      router.push('/login')
    } catch (error: any) {
      ElMessage.error(error.message || 'Registration failed')
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped lang="scss">
.auth-page {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-bg-deep) 0%, #1a1a2e 100%);
  position: relative;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    width: 500px;
    height: 500px;
    background: radial-gradient(circle, rgba(124, 140, 248, 0.1) 0%, transparent 70%);
    top: -200px;
    right: -200px;
    animation: pulse-glow 3s ease-in-out infinite;
  }
}

.auth-container {
  width: 100%;
  max-width: 420px;
  padding: 24px;
  z-index: 1;
}

.auth-card {
  padding: 40px;
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;
}

.auth-title {
  font-size: 32px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}

.auth-subtitle {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.auth-form {
  margin-bottom: 24px;
}

.auth-submit {
  width: 100%;
  height: 44px;
  font-size: 16px;
  margin-top: 8px;
}

.auth-footer {
  text-align: center;
  font-size: 14px;
  color: var(--color-text-secondary);

  .auth-link {
    color: var(--color-primary);
    text-decoration: none;
    margin-left: 8px;
    font-weight: 500;

    &:hover {
      color: var(--color-primary-light);
    }
  }
}
</style>
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/views/auth/
git commit -m "feat: add login and register pages with form validation"
```

---

### Task 9: 工作空间页面

**Files:**
- Create: `web/src/api/workspace.ts`
- Create: `web/src/views/workspace/WorkspaceList.vue`
- Create: `web/src/views/workspace/WorkspaceDetail.vue`
- Create: `web/src/views/workspace/MemberManage.vue`

- [ ] **Step 1: 创建 `web/src/api/workspace.ts`**

```ts
// web/src/api/workspace.ts
import request from './request'
import type { PaginatedResponse } from './types'

export interface Workspace {
  id: number
  name: string
  type: string
  owner_id: number
  description: string
  created_at: string
  updated_at: string
}

export interface WorkspaceMember {
  id: number
  workspace_id: number
  user_id: number
  role: string
  created_at: string
  username?: string
  email?: string
}

export interface CreateWorkspacePayload {
  name: string
  type: string
  description?: string
}

export interface AddMemberPayload {
  user_id: number
  role: string
}

export const workspaceApi = {
  // 获取工作空间列表
  list: () => request.get<PaginatedResponse<Workspace>>('/workspaces'),

  // 获取单个工作空间
  get: (id: number) => request.get<Workspace>(`/workspaces/${id}`),

  // 创建工作空间
  create: (data: CreateWorkspacePayload) => request.post<Workspace>('/workspaces', data),

  // 更新工作空间
  update: (id: number, data: Partial<CreateWorkspacePayload>) =>
    request.put<Workspace>(`/workspaces/${id}`, data),

  // 删除工作空间
  delete: (id: number) => request.delete(`/workspaces/${id}`),

  // 获取成员列表
  getMembers: (id: number) => request.get<WorkspaceMember[]>(`/workspaces/${id}/members`),

  // 添加成员
  addMember: (id: number, data: AddMemberPayload) =>
    request.post<WorkspaceMember>(`/workspaces/${id}/members`, data),

  // 移除成员
  removeMember: (id: number, userId: number) =>
    request.delete(`/workspaces/${id}/members/${userId}`),

  // 更新成员角色
  updateMemberRole: (id: number, userId: number, role: string) =>
    request.put(`/workspaces/${id}/members/${userId}`, { role }),
}
```

- [ ] **Step 2: 创建 `web/src/views/workspace/WorkspaceList.vue`**

```vue
<!-- web/src/views/workspace/WorkspaceList.vue -->
<template>
  <div class="workspace-list">
    <div class="page-header">
      <h1 class="page-title">Workspaces</h1>
      <NeonButton type="primary" @click="showCreateDialog = true">
        + New Workspace
      </NeonButton>
    </div>

    <div v-loading="loading" class="workspace-grid">
      <GlowCard
        v-for="ws in workspaces"
        :key="ws.id"
        hoverable
        class="workspace-card"
        @click="goToWorkspace(ws.id)"
      >
        <div class="workspace-card__header">
          <h3 class="workspace-card__title">{{ ws.name }}</h3>
          <el-tag :type="ws.type === 'personal' ? 'info' : 'success'" size="small">
            {{ ws.type }}
          </el-tag>
        </div>
        <p class="workspace-card__desc">{{ ws.description || 'No description' }}</p>
        <div class="workspace-card__footer">
          <span class="workspace-card__date">{{ formatDate(ws.created_at) }}</span>
        </div>
      </GlowCard>
    </div>

    <el-dialog
      v-model="showCreateDialog"
      title="Create Workspace"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Enter workspace name" />
        </el-form-item>

        <el-form-item label="Type" prop="type">
          <el-select v-model="form.type" placeholder="Select type">
            <el-option label="Personal" value="personal" />
            <el-option label="Team" value="team" />
          </el-select>
        </el-form-item>

        <el-form-item label="Description" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="Enter description (optional)"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showCreateDialog = false">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleCreate">
          Create
        </NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { workspaceApi } from '@/api/workspace'
import { useWorkspaceStore } from '@/store/workspace'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const router = useRouter()
const workspaceStore = useWorkspaceStore()

const loading = ref(false)
const showCreateDialog = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const workspaces = ref<any[]>([])

const form = reactive({
  name: '',
  type: 'personal',
  description: '',
})

const rules: FormRules = {
  name: [{ required: true, message: 'Please enter workspace name', trigger: 'blur' }],
  type: [{ required: true, message: 'Please select type', trigger: 'change' }],
}

onMounted(async () => {
  await fetchWorkspaces()
})

async function fetchWorkspaces() {
  loading.value = true
  try {
    const data: any = await workspaceApi.list()
    workspaces.value = data.items || data || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      await workspaceApi.create(form)
      ElMessage.success('Workspace created successfully')
      showCreateDialog.value = false
      await fetchWorkspaces()
      Object.assign(form, { name: '', type: 'personal', description: '' })
    } catch (error: any) {
      ElMessage.error(error.message || 'Failed to create workspace')
    } finally {
      submitting.value = false
    }
  })
}

function goToWorkspace(id: number) {
  router.push(`/workspace/${id}`)
}

function formatDate(date: string) {
  return new Date(date).toLocaleDateString()
}
</script>

<style scoped lang="scss">
.workspace-list {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.workspace-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 24px;
}

.workspace-card {
  cursor: pointer;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  &__title {
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__desc {
    font-size: 14px;
    color: var(--color-text-secondary);
    margin-bottom: 16px;
    min-height: 40px;
  }

  &__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 12px;
    border-top: 1px solid var(--border-glow);
  }

  &__date {
    font-size: 12px;
    color: var(--color-text-muted);
  }
}
</style>
```

- [ ] **Step 3: 创建 `web/src/views/workspace/WorkspaceDetail.vue`（占位，后续 Task 10 完善）**

```vue
<!-- web/src/views/workspace/WorkspaceDetail.vue -->
<template>
  <div class="workspace-detail">
    <h1>Workspace Detail (Placeholder)</h1>
    <p>Workspace ID: {{ id }}</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  id: string
}>()
</script>

<style scoped lang="scss">
.workspace-detail {
  padding: 24px;
}
</style>
```

- [ ] **Step 4: 创建 `web/src/views/workspace/MemberManage.vue`（占位）**

```vue
<!-- web/src/views/workspace/MemberManage.vue -->
<template>
  <div class="member-manage">
    <h2>Member Management (Placeholder)</h2>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped lang="scss">
.member-manage {
  padding: 24px;
}
</style>
```

- [ ] **Step 5: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/api/workspace.ts web/src/views/workspace/
git commit -m "feat: add workspace list page with create dialog"
```

---

### Task 10: 作品集页面

**Files:**
- Create: `web/src/api/portfolio.ts`
- Create: `web/src/views/portfolio/PortfolioList.vue`
- Create: `web/src/views/portfolio/PortfolioDetail.vue`

- [ ] **Step 1: 创建 `web/src/api/portfolio.ts`**

```ts
// web/src/api/portfolio.ts
import request from './request'

export interface Portfolio {
  id: number
  workspace_id: number
  name: string
  description: string
  cover_url: string
  created_at: string
  updated_at: string
}

export interface Resource {
  id: number
  portfolio_id: number
  name: string
  type: string
  url: string
  thumbnail_url: string
  created_at: string
}

export interface CreatePortfolioPayload {
  name: string
  description?: string
}

export const portfolioApi = {
  // 获取作品集列表
  list: (workspaceId: number) =>
    request.get(`/workspaces/${workspaceId}/portfolios`),

  // 获取单个作品集
  get: (workspaceId: number, portfolioId: number) =>
    request.get(`/workspaces/${workspaceId}/portfolios/${portfolioId}`),

  // 创建作品集
  create: (workspaceId: number, data: CreatePortfolioPayload) =>
    request.post(`/workspaces/${workspaceId}/portfolios`, data),

  // 更新作品集
  update: (workspaceId: number, portfolioId: number, data: Partial<CreatePortfolioPayload>) =>
    request.put(`/workspaces/${workspaceId}/portfolios/${portfolioId}`, data),

  // 删除作品集
  delete: (workspaceId: number, portfolioId: number) =>
    request.delete(`/workspaces/${workspaceId}/portfolios/${portfolioId}`),

  // 获取资源列表
  getResources: (workspaceId: number, portfolioId: number) =>
    request.get(`/workspaces/${workspaceId}/portfolios/${portfolioId}/resources`),

  // 上传资源
  uploadResource: (workspaceId: number, portfolioId: number, formData: FormData) =>
    request.post(`/workspaces/${workspaceId}/portfolios/${portfolioId}/resources`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),

  // 删除资源
  deleteResource: (workspaceId: number, portfolioId: number, resourceId: number) =>
    request.delete(`/workspaces/${workspaceId}/portfolios/${portfolioId}/resources/${resourceId}`),
}
```

- [ ] **Step 2: 更新 `web/src/views/workspace/WorkspaceDetail.vue` 为完整实现**

```vue
<!-- web/src/views/workspace/WorkspaceDetail.vue -->
<template>
  <div class="workspace-detail">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ workspace?.name || 'Loading...' }}</h1>
        <p class="page-subtitle">{{ workspace?.description }}</p>
      </div>
      <div class="page-actions">
        <NeonButton type="primary" @click="showCreateDialog = true">
          + New Portfolio
        </NeonButton>
        <el-button @click="showMembers = !showMembers">
          {{ showMembers ? 'Hide Members' : 'Manage Members' }}
        </el-button>
      </div>
    </div>

    <MemberManage v-if="showMembers" :workspace-id="Number(id)" />

    <div v-loading="loading" class="portfolio-grid">
      <GlowCard
        v-for="p in portfolios"
        :key="p.id"
        hoverable
        class="portfolio-card"
        @click="goToPortfolio(p.id)"
      >
        <div class="portfolio-card__cover">
          <img v-if="p.cover_url" :src="p.cover_url" :alt="p.name" />
          <div v-else class="portfolio-card__placeholder">
            <span>🎨</span>
          </div>
        </div>
        <div class="portfolio-card__info">
          <h3>{{ p.name }}</h3>
          <p>{{ p.description || 'No description' }}</p>
        </div>
      </GlowCard>
    </div>

    <el-dialog
      v-model="showCreateDialog"
      title="Create Portfolio"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Enter portfolio name" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="Enter description (optional)"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleCreate">
          Create
        </NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { portfolioApi, Portfolio } from '@/api/portfolio'
import { workspaceApi, Workspace } from '@/api/workspace'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import MemberManage from './MemberManage.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()

const workspace = ref<Workspace | null>(null)
const portfolios = ref<Portfolio[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showMembers = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({ name: '', description: '' })
const rules: FormRules = {
  name: [{ required: true, message: 'Please enter portfolio name', trigger: 'blur' }],
}

onMounted(async () => {
  await Promise.all([fetchWorkspace(), fetchPortfolios()])
})

async function fetchWorkspace() {
  const data: any = await workspaceApi.get(Number(props.id))
  workspace.value = data
}

async function fetchPortfolios() {
  loading.value = true
  try {
    const data: any = await portfolioApi.list(Number(props.id))
    portfolios.value = data.items || data || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await portfolioApi.create(Number(props.id), form)
      ElMessage.success('Portfolio created successfully')
      showCreateDialog.value = false
      await fetchPortfolios()
      Object.assign(form, { name: '', description: '' })
    } catch (error: any) {
      ElMessage.error(error.message || 'Failed to create portfolio')
    } finally {
      submitting.value = false
    }
  })
}

function goToPortfolio(pid: number) {
  router.push(`/workspace/${props.id}/portfolio/${pid}`)
}
</script>

<style scoped lang="scss">
.workspace-detail {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.page-actions {
  display: flex;
  gap: 12px;
}

.portfolio-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 24px;
}

.portfolio-card {
  padding: 0;
  overflow: hidden;
  cursor: pointer;

  &__cover {
    width: 100%;
    height: 180px;
    overflow: hidden;
    background-color: var(--color-bg-hover);

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  &__placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 48px;
    opacity: 0.3;
  }

  &__info {
    padding: 16px;

    h3 {
      font-size: 16px;
      font-weight: 600;
      color: var(--color-text-primary);
      margin-bottom: 4px;
    }

    p {
      font-size: 13px;
      color: var(--color-text-secondary);
    }
  }
}
</style>
```

- [ ] **Step 3: 更新 `web/src/views/workspace/MemberManage.vue` 为完整实现**

```vue
<!-- web/src/views/workspace/MemberManage.vue -->
<template>
  <FadePanel blur class="member-manage">
    <div class="member-header">
      <h3>Members</h3>
      <NeonButton type="primary" @click="showAddDialog = true">+ Add Member</NeonButton>
    </div>

    <el-table :data="members" style="width: 100%">
      <el-table-column prop="username" label="Username" />
      <el-table-column prop="email" label="Email" />
      <el-table-column prop="role" label="Role" width="150">
        <template #default="{ row }">
          <el-tag :type="roleTagType(row.role)">{{ row.role }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Actions" width="120">
        <template #default="{ row }">
          <el-button
            v-if="row.role !== 'owner'"
            type="danger"
            size="small"
            text
            @click="handleRemove(row.user_id)"
          >
            Remove
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="showAddDialog"
      title="Add Member"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form :model="addForm" label-width="80px">
        <el-form-item label="User ID">
          <el-input v-model.number="addForm.user_id" placeholder="Enter user ID" />
        </el-form-item>
        <el-form-item label="Role">
          <el-select v-model="addForm.role">
            <el-option label="Editor" value="editor" />
            <el-option label="Viewer" value="viewer" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">Cancel</el-button>
        <NeonButton type="primary" @click="handleAdd">Add</NeonButton>
      </template>
    </el-dialog>
  </FadePanel>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { workspaceApi, WorkspaceMember } from '@/api/workspace'
import FadePanel from '@/components/common/FadePanel.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const props = defineProps<{ workspaceId: number }>()

const members = ref<WorkspaceMember[]>([])
const showAddDialog = ref(false)
const addForm = reactive({ user_id: 0, role: 'viewer' })

onMounted(async () => {
  await fetchMembers()
})

async function fetchMembers() {
  const data: any = await workspaceApi.getMembers(props.workspaceId)
  members.value = data || []
}

async function handleAdd() {
  try {
    await workspaceApi.addMember(props.workspaceId, addForm)
    ElMessage.success('Member added')
    showAddDialog.value = false
    await fetchMembers()
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to add member')
  }
}

async function handleRemove(userId: number) {
  await ElMessageBox.confirm('Are you sure to remove this member?', 'Confirm')
  try {
    await workspaceApi.removeMember(props.workspaceId, userId)
    ElMessage.success('Member removed')
    await fetchMembers()
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to remove member')
  }
}

function roleTagType(role: string) {
  const map: Record<string, string> = { owner: 'danger', editor: 'warning', viewer: 'info' }
  return map[role] || 'info'
}
</script>

<style scoped lang="scss">
.member-manage {
  margin-bottom: 24px;
}

.member-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h3 {
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
}
</style>
```

- [ ] **Step 4: 创建 `web/src/views/portfolio/PortfolioList.vue`**

```vue
<!-- web/src/views/portfolio/PortfolioList.vue -->
<template>
  <div class="portfolio-list">
    <div class="page-header">
      <h1 class="page-title">Portfolios</h1>
    </div>
    <div class="portfolio-grid">
      <GlowCard
        v-for="p in portfolios"
        :key="p.id"
        hoverable
        @click="$emit('select', p)"
      >
        <div class="portfolio-item">
          <div class="portfolio-item__cover">
            <img v-if="p.cover_url" :src="p.cover_url" :alt="p.name" />
            <div v-else class="portfolio-item__placeholder">🎨</div>
          </div>
          <h3>{{ p.name }}</h3>
          <p>{{ p.description || 'No description' }}</p>
        </div>
      </GlowCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Portfolio } from '@/api/portfolio'
import GlowCard from '@/components/common/GlowCard.vue'

defineProps<{
  portfolios: Portfolio[]
}>()

defineEmits<{
  select: [portfolio: Portfolio]
}>()
</script>

<style scoped lang="scss">
.portfolio-list {
  width: 100%;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.portfolio-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 20px;
}

.portfolio-item {
  &__cover {
    width: 100%;
    height: 140px;
    border-radius: 8px;
    overflow: hidden;
    background-color: var(--color-bg-hover);
    margin-bottom: 12px;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  &__placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 36px;
    opacity: 0.3;
  }

  h3 {
    font-size: 15px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 4px;
  }

  p {
    font-size: 13px;
    color: var(--color-text-secondary);
  }
}
</style>
```

- [ ] **Step 5: 创建 `web/src/views/portfolio/PortfolioDetail.vue`**

```vue
<!-- web/src/views/portfolio/PortfolioDetail.vue -->
<template>
  <div class="portfolio-detail">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ portfolio?.name || 'Loading...' }}</h1>
        <p class="page-subtitle">{{ portfolio?.description }}</p>
      </div>
      <div class="page-actions">
        <NeonButton type="primary" @click="goToStudio">🎨 AI Studio</NeonButton>
        <NeonButton @click="goToCharacters">👤 Characters</NeonButton>
      </div>
    </div>

    <FadePanel blur class="resource-section">
      <div class="resource-header">
        <h3>Resources</h3>
        <el-upload
          :action="uploadUrl"
          :headers="uploadHeaders"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
          :show-file-list="false"
          accept="image/*"
        >
          <NeonButton type="primary">Upload</NeonButton>
        </el-upload>
      </div>

      <div v-loading="loading" class="resource-grid">
        <GlowCard
          v-for="res in resources"
          :key="res.id"
          hoverable
          class="resource-card"
        >
          <div class="resource-card__preview">
            <img :src="res.thumbnail_url || res.url" :alt="res.name" />
          </div>
          <div class="resource-card__info">
            <span class="resource-card__name">{{ res.name }}</span>
            <el-button
              type="danger"
              size="small"
              text
              @click="handleDeleteResource(res.id)"
            >
              Delete
            </el-button>
          </div>
        </GlowCard>
      </div>
    </FadePanel>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { portfolioApi, Portfolio, Resource } from '@/api/portfolio'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import FadePanel from '@/components/common/FadePanel.vue'

const props = defineProps<{ id: string; pid: string }>()
const router = useRouter()

const portfolio = ref<Portfolio | null>(null)
const resources = ref<Resource[]>([])
const loading = ref(false)

const uploadUrl = computed(
  () => `/api/v1/workspaces/${props.id}/portfolios/${props.pid}/resources`
)
const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem('access_token')}`,
}))

onMounted(async () => {
  await Promise.all([fetchPortfolio(), fetchResources()])
})

async function fetchPortfolio() {
  const data: any = await portfolioApi.get(Number(props.id), Number(props.pid))
  portfolio.value = data
}

async function fetchResources() {
  loading.value = true
  try {
    const data: any = await portfolioApi.getResources(Number(props.id), Number(props.pid))
    resources.value = data || []
  } finally {
    loading.value = false
  }
}

function handleUploadSuccess() {
  ElMessage.success('Resource uploaded')
  fetchResources()
}

function handleUploadError() {
  ElMessage.error('Upload failed')
}

async function handleDeleteResource(resourceId: number) {
  await ElMessageBox.confirm('Are you sure to delete this resource?', 'Confirm')
  try {
    await portfolioApi.deleteResource(Number(props.id), Number(props.pid), resourceId)
    ElMessage.success('Resource deleted')
    await fetchResources()
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to delete resource')
  }
}

function goToStudio() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/studio`)
}

function goToCharacters() {
  router.push(`/workspace/${props.id}/portfolio/${props.pid}/characters`)
}
</script>

<style scoped lang="scss">
.portfolio-detail {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.page-actions {
  display: flex;
  gap: 12px;
}

.resource-section {
  margin-top: 24px;
}

.resource-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h3 {
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
}

.resource-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}

.resource-card {
  padding: 0;
  overflow: hidden;

  &__preview {
    width: 100%;
    height: 150px;
    overflow: hidden;
    background-color: var(--color-bg-hover);

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  &__info {
    padding: 8px 12px;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__name {
    font-size: 13px;
    color: var(--color-text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
```

- [ ] **Step 6: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/api/portfolio.ts web/src/views/portfolio/ web/src/views/workspace/WorkspaceDetail.vue web/src/views/workspace/MemberManage.vue
git commit -m "feat: add portfolio pages with resource management and member management"
```

---

### Task 11: 角色管理页面

**Files:**
- Create: `web/src/api/character.ts`
- Create: `web/src/views/character/CharacterList.vue`
- Create: `web/src/views/character/CharacterEditor.vue`

- [ ] **Step 1: 创建 `web/src/api/character.ts`**

```ts
// web/src/api/character.ts
import request from './request'

export interface Character {
  id: number
  portfolio_id: number
  name: string
  description: string
  attributes: Record<string, any>
  reference_images: string[]
  created_at: string
  updated_at: string
}

export interface CreateCharacterPayload {
  name: string
  description?: string
  attributes?: Record<string, any>
}

export const characterApi = {
  // 获取角色列表
  list: (workspaceId: number, portfolioId: number) =>
    request.get(`/workspaces/${workspaceId}/portfolios/${portfolioId}/characters`),

  // 获取单个角色
  get: (workspaceId: number, portfolioId: number, characterId: number) =>
    request.get(`/workspaces/${workspaceId}/portfolios/${portfolioId}/characters/${characterId}`),

  // 创建角色
  create: (workspaceId: number, portfolioId: number, data: CreateCharacterPayload) =>
    request.post(`/workspaces/${workspaceId}/portfolios/${portfolioId}/characters`, data),

  // 更新角色
  update: (
    workspaceId: number,
    portfolioId: number,
    characterId: number,
    data: Partial<CreateCharacterPayload>
  ) =>
    request.put(
      `/workspaces/${workspaceId}/portfolios/${portfolioId}/characters/${characterId}`,
      data
    ),

  // 删除角色
  delete: (workspaceId: number, portfolioId: number, characterId: number) =>
    request.delete(
      `/workspaces/${workspaceId}/portfolios/${portfolioId}/characters/${characterId}`
    ),

  // 上传参考图
  uploadReference: (
    workspaceId: number,
    portfolioId: number,
    characterId: number,
    formData: FormData
  ) =>
    request.post(
      `/workspaces/${workspaceId}/portfolios/${portfolioId}/characters/${characterId}/references`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    ),
}
```

- [ ] **Step 2: 创建 `web/src/views/character/CharacterList.vue`**

```vue
<!-- web/src/views/character/CharacterList.vue -->
<template>
  <div class="character-list">
    <div class="page-header">
      <h1 class="page-title">Characters</h1>
      <NeonButton type="primary" @click="showCreateDialog = true">
        + New Character
      </NeonButton>
    </div>

    <div v-loading="loading" class="character-grid">
      <GlowCard
        v-for="char in characters"
        :key="char.id"
        hoverable
        class="character-card"
        @click="handleEdit(char)"
      >
        <div class="character-card__avatar">
          <img
            v-if="char.reference_images && char.reference_images.length > 0"
            :src="char.reference_images[0]"
            :alt="char.name"
          />
          <div v-else class="character-card__placeholder">
            <span>👤</span>
          </div>
        </div>
        <div class="character-card__info">
          <h3>{{ char.name }}</h3>
          <p>{{ char.description || 'No description' }}</p>
        </div>
      </GlowCard>
    </div>

    <el-dialog
      v-model="showCreateDialog"
      title="Create Character"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="form.name" placeholder="Enter character name" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="Enter description (optional)"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleCreate">
          Create
        </NeonButton>
      </template>
    </el-dialog>

    <CharacterEditor
      v-if="editingCharacter"
      :character="editingCharacter"
      :workspace-id="Number(id)"
      :portfolio-id="Number(pid)"
      @close="editingCharacter = null"
      @updated="fetchCharacters"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { characterApi, Character } from '@/api/character'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import CharacterEditor from './CharacterEditor.vue'

const props = defineProps<{ id: string; pid: string }>()

const characters = ref<Character[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const editingCharacter = ref<Character | null>(null)

const form = reactive({ name: '', description: '' })
const rules: FormRules = {
  name: [{ required: true, message: 'Please enter character name', trigger: 'blur' }],
}

onMounted(async () => {
  await fetchCharacters()
})

async function fetchCharacters() {
  loading.value = true
  try {
    const data: any = await characterApi.list(Number(props.id), Number(props.pid))
    characters.value = data || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await characterApi.create(Number(props.id), Number(props.pid), form)
      ElMessage.success('Character created successfully')
      showCreateDialog.value = false
      await fetchCharacters()
      Object.assign(form, { name: '', description: '' })
    } catch (error: any) {
      ElMessage.error(error.message || 'Failed to create character')
    } finally {
      submitting.value = false
    }
  })
}

function handleEdit(char: Character) {
  editingCharacter.value = char
}
</script>

<style scoped lang="scss">
.character-list {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.character-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 24px;
}

.character-card {
  cursor: pointer;

  &__avatar {
    width: 100%;
    height: 200px;
    border-radius: 8px;
    overflow: hidden;
    background-color: var(--color-bg-hover);
    margin-bottom: 12px;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  &__placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 64px;
    opacity: 0.3;
  }

  &__info {
    h3 {
      font-size: 16px;
      font-weight: 600;
      color: var(--color-text-primary);
      margin-bottom: 4px;
    }

    p {
      font-size: 13px;
      color: var(--color-text-secondary);
    }
  }
}
</style>
```

- [ ] **Step 3: 创建 `web/src/views/character/CharacterEditor.vue`**

```vue
<!-- web/src/views/character/CharacterEditor.vue -->
<template>
  <el-drawer
    v-model="visible"
    title="Edit Character"
    size="600px"
    :before-close="handleClose"
  >
    <div class="character-editor">
      <el-form :model="form" label-width="120px">
        <el-form-item label="Name">
          <el-input v-model="form.name" />
        </el-form-item>

        <el-form-item label="Description">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>

        <el-form-item label="Attributes">
          <el-input
            v-model="attributesJson"
            type="textarea"
            :rows="6"
            placeholder='{"age": 25, "gender": "female"}'
          />
        </el-form-item>

        <el-form-item label="Reference Images">
          <div class="reference-images">
            <div
              v-for="(img, idx) in form.reference_images"
              :key="idx"
              class="reference-image"
            >
              <img :src="img" :alt="`Reference ${idx + 1}`" />
            </div>
            <el-upload
              :action="uploadUrl"
              :headers="uploadHeaders"
              :on-success="handleUploadSuccess"
              :show-file-list="false"
              accept="image/*"
            >
              <div class="upload-placeholder">
                <span>+</span>
              </div>
            </el-upload>
          </div>
        </el-form-item>
      </el-form>

      <div class="editor-actions">
        <el-button @click="handleClose">Cancel</el-button>
        <NeonButton type="primary" :loading="saving" @click="handleSave">
          Save
        </NeonButton>
        <el-button type="danger" @click="handleDelete">Delete</el-button>
      </div>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { characterApi, Character } from '@/api/character'
import NeonButton from '@/components/common/NeonButton.vue'

const props = defineProps<{
  character: Character
  workspaceId: number
  portfolioId: number
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const visible = ref(true)
const saving = ref(false)

const form = reactive({
  name: props.character.name,
  description: props.character.description || '',
  attributes: props.character.attributes || {},
  reference_images: props.character.reference_images || [],
})

const attributesJson = ref(JSON.stringify(form.attributes, null, 2))

const uploadUrl = computed(
  () =>
    `/api/v1/workspaces/${props.workspaceId}/portfolios/${props.portfolioId}/characters/${props.character.id}/references`
)

const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem('access_token')}`,
}))

watch(attributesJson, (val) => {
  try {
    form.attributes = JSON.parse(val)
  } catch {
    // Invalid JSON, ignore
  }
})

function handleClose() {
  visible.value = false
  setTimeout(() => emit('close'), 300)
}

async function handleSave() {
  saving.value = true
  try {
    await characterApi.update(props.workspaceId, props.portfolioId, props.character.id, {
      name: form.name,
      description: form.description,
      attributes: form.attributes,
    })
    ElMessage.success('Character updated')
    emit('updated')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to update character')
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  await ElMessageBox.confirm('Are you sure to delete this character?', 'Confirm')
  try {
    await characterApi.delete(props.workspaceId, props.portfolioId, props.character.id)
    ElMessage.success('Character deleted')
    emit('updated')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to delete character')
  }
}

function handleUploadSuccess(response: any) {
  ElMessage.success('Reference image uploaded')
  form.reference_images.push(response.data.url)
}
</script>

<style scoped lang="scss">
.character-editor {
  padding: 24px;
}

.reference-images {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.reference-image {
  width: 100%;
  aspect-ratio: 1;
  border-radius: 8px;
  overflow: hidden;
  background-color: var(--color-bg-hover);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.upload-placeholder {
  width: 100%;
  aspect-ratio: 1;
  border: 2px dashed var(--border-glow);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }
}

.editor-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--border-glow);
}
</style>
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/api/character.ts web/src/views/character/
git commit -m "feat: add character management with editor and reference images"
```

---

### Task 12: AI 创作工坊

**Files:**
- Create: `web/src/api/ai.ts`
- Create: `web/src/components/ai/ModelSelector.vue`
- Create: `web/src/components/ai/PromptEditor.vue`
- Create: `web/src/components/ai/TaskProgress.vue`
- Create: `web/src/views/studio/AIStudio.vue`

- [ ] **Step 1: 创建 `web/src/api/ai.ts`**

```ts
// web/src/api/ai.ts
import request from './request'

export interface AIModel {
  id: string
  name: string
  provider: string
  capabilities: string[]
  description: string
}

export interface AITaskPayload {
  model_id: string
  prompt: string
  negative_prompt?: string
  character_ids?: number[]
  parameters?: Record<string, any>
}

export interface AITaskResult {
  id: string
  type: string
  status: string
  progress: number
  result_url?: string
  error?: string
  created_at: string
}

export const aiApi = {
  // 获取可用模型列表
  getModels: () => request.get<AIModel[]>('/ai/models'),

  // 提交 AI 任务
  submitTask: (workspaceId: number, portfolioId: number, data: AITaskPayload) =>
    request.post<AITaskResult>(
      `/workspaces/${workspaceId}/portfolios/${portfolioId}/ai/tasks`,
      data
    ),

  // 获取任务列表
  getTasks: (workspaceId: number, portfolioId: number) =>
    request.get<AITaskResult[]>(
      `/workspaces/${workspaceId}/portfolios/${portfolioId}/ai/tasks`
    ),

  // 获取单个任务
  getTask: (workspaceId: number, portfolioId: number, taskId: string) =>
    request.get<AITaskResult>(
      `/workspaces/${workspaceId}/portfolios/${portfolioId}/ai/tasks/${taskId}`
    ),
}
```

- [ ] **Step 2: 创建 `web/src/components/ai/ModelSelector.vue`**

```vue
<!-- web/src/components/ai/ModelSelector.vue -->
<template>
  <div class="model-selector">
    <h4 class="model-selector__title">Select Model</h4>
    <div class="model-list">
      <div
        v-for="model in models"
        :key="model.id"
        class="model-item"
        :class="{
          'model-item--selected': selectedId === model.id,
          'model-item--disabled': !isCapable(model),
        }"
        @click="handleSelect(model)"
      >
        <div class="model-item__header">
          <span class="model-item__name">{{ model.name }}</span>
          <span class="model-item__provider">{{ model.provider }}</span>
        </div>
        <p class="model-item__desc">{{ model.description }}</p>
        <div class="model-item__caps">
          <el-tag
            v-for="cap in model.capabilities"
            :key="cap"
            size="small"
            :type="requiredCapability && cap === requiredCapability ? 'success' : 'info'"
          >
            {{ cap }}
          </el-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { aiApi, AIModel } from '@/api/ai'

const props = defineProps<{
  requiredCapability?: string
}>()

const emit = defineEmits<{
  select: [model: AIModel]
}>()

const models = ref<AIModel[]>([])
const selectedId = ref<string>('')

onMounted(async () => {
  const data: any = await aiApi.getModels()
  models.value = data || []
})

function isCapable(model: AIModel): boolean {
  if (!props.requiredCapability) return true
  return model.capabilities.includes(props.requiredCapability)
}

function handleSelect(model: AIModel) {
  if (!isCapable(model)) return
  selectedId.value = model.id
  emit('select', model)
}
</script>

<style scoped lang="scss">
.model-selector {
  &__title {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 12px;
  }
}

.model-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.model-item {
  padding: 12px;
  border: 1px solid var(--border-glow);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover:not(.model-item--disabled) {
    border-color: var(--color-primary);
    background-color: var(--color-bg-hover);
  }

  &--selected {
    border-color: var(--color-primary);
    box-shadow: 0 0 15px rgba(124, 140, 248, 0.2);
    background-color: var(--color-bg-hover);
  }

  &--disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  &__name {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  &__provider {
    font-size: 12px;
    color: var(--color-text-muted);
  }

  &__desc {
    font-size: 12px;
    color: var(--color-text-secondary);
    margin-bottom: 8px;
  }

  &__caps {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }
}
</style>
```

- [ ] **Step 3: 创建 `web/src/components/ai/PromptEditor.vue`**

```vue
<!-- web/src/components/ai/PromptEditor.vue -->
<template>
  <div class="prompt-editor">
    <h4 class="prompt-editor__title">Prompt</h4>

    <div v-if="characters.length > 0" class="character-inject">
      <span class="character-inject__label">Inject Character:</span>
      <el-tag
        v-for="char in characters"
        :key="char.id"
        class="character-tag"
        effect="plain"
        @click="injectCharacter(char)"
      >
        {{ char.name }}
      </el-tag>
    </div>

    <el-input
      ref="promptInput"
      v-model="prompt"
      type="textarea"
      :rows="6"
      placeholder="Describe what you want to create..."
      @input="$emit('update:modelValue', prompt)"
    />

    <div class="prompt-editor__footer">
      <el-input
        v-model="negativePrompt"
        type="textarea"
        :rows="2"
        placeholder="Negative prompt (optional)"
        @input="$emit('update:negativePrompt', negativePrompt)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Character } from '@/api/character'

const props = defineProps<{
  modelValue?: string
  negativePromptValue?: string
  characters?: Character[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:negativePrompt': [value: string]
}>()

const prompt = ref(props.modelValue || '')
const negativePrompt = ref(props.negativePromptValue || '')
const promptInput = ref()

watch(
  () => props.modelValue,
  (val) => {
    prompt.value = val || ''
  }
)

function injectCharacter(char: Character) {
  const injection = `[Character: ${char.name}] `
  prompt.value += injection
  emit('update:modelValue', prompt.value)
}
</script>

<style scoped lang="scss">
.prompt-editor {
  &__title {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 12px;
  }

  &__footer {
    margin-top: 12px;
  }
}

.character-inject {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;

  &__label {
    font-size: 12px;
    color: var(--color-text-secondary);
  }
}

.character-tag {
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }
}
</style>
```

- [ ] **Step 4: 创建 `web/src/components/ai/TaskProgress.vue`**

```vue
<!-- web/src/components/ai/TaskProgress.vue -->
<template>
  <div class="task-progress">
    <div class="task-progress__header">
      <span class="task-progress__type">{{ task.type }}</span>
      <el-tag :type="statusTagType" size="small">{{ task.status }}</el-tag>
    </div>

    <el-progress
      :percentage="task.progress"
      :status="progressStatus"
      :stroke-width="6"
      :show-text="true"
    />

    <div v-if="task.result_url" class="task-progress__result">
      <img :src="task.result_url" alt="Result" class="result-preview" />
    </div>

    <div v-if="task.error" class="task-progress__error">
      {{ task.error }}
    </div>

    <div class="task-progress__footer">
      <span class="task-progress__time">{{ formatTime(task.created_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AITask } from '@/store/ai'

const props = defineProps<{
  task: AITask
}>()

const statusTagType = computed(() => {
  const map: Record<string, string> = {
    pending: 'info',
    running: 'warning',
    completed: 'success',
    failed: 'danger',
  }
  return map[props.task.status] || 'info'
})

const progressStatus = computed(() => {
  if (props.task.status === 'completed') return 'success'
  if (props.task.status === 'failed') return 'exception'
  return undefined
})

function formatTime(date: string) {
  return new Date(date).toLocaleTimeString()
}
</script>

<style scoped lang="scss">
.task-progress {
  padding: 12px;
  border: 1px solid var(--border-glow);
  border-radius: 8px;
  margin-bottom: 8px;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }

  &__type {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-primary);
  }

  &__result {
    margin-top: 12px;

    .result-preview {
      width: 100%;
      max-height: 200px;
      object-fit: contain;
      border-radius: 8px;
    }
  }

  &__error {
    margin-top: 8px;
    font-size: 12px;
    color: #EF4444;
    padding: 8px;
    background-color: rgba(239, 68, 68, 0.1);
    border-radius: 4px;
  }

  &__footer {
    margin-top: 8px;
    display: flex;
    justify-content: flex-end;
  }

  &__time {
    font-size: 11px;
    color: var(--color-text-muted);
  }
}
</style>
```

- [ ] **Step 5: 创建 `web/src/views/studio/AIStudio.vue`**

```vue
<!-- web/src/views/studio/AIStudio.vue -->
<template>
  <div class="ai-studio">
    <!-- 左侧工具栏 -->
    <aside class="studio-sidebar">
      <FadePanel blur class="sidebar-section">
        <ModelSelector
          required-capability="image-generation"
          @select="handleModelSelect"
        />
      </FadePanel>

      <FadePanel blur class="sidebar-section">
        <PromptEditor
          v-model="prompt"
          :characters="characters"
          @update:negative-prompt="negativePrompt = $event"
        />
      </FadePanel>

      <FadePanel blur class="sidebar-section">
        <h4 class="section-title">Parameters</h4>
        <el-form label-width="80px" size="small">
          <el-form-item label="Width">
            <el-input-number v-model="params.width" :min="256" :max="2048" :step="64" />
          </el-form-item>
          <el-form-item label="Height">
            <el-input-number v-model="params.height" :min="256" :max="2048" :step="64" />
          </el-form-item>
          <el-form-item label="Steps">
            <el-slider v-model="params.steps" :min="1" :max="100" />
          </el-form-item>
        </el-form>
      </FadePanel>

      <NeonButton
        type="primary"
        :loading="submitting"
        class="generate-btn"
        @click="handleGenerate"
      >
        Generate
      </NeonButton>
    </aside>

    <!-- 中间画布 -->
    <main class="studio-canvas">
      <div v-if="latestResult" class="canvas-content">
        <img :src="latestResult" alt="Generated" class="canvas-image" />
      </div>
      <div v-else class="canvas-empty">
        <span class="canvas-empty__icon">🎨</span>
        <p>Select a model and enter a prompt to start creating</p>
      </div>
    </main>

    <!-- 右侧结果面板 -->
    <aside class="studio-results">
      <FadePanel blur class="results-panel">
        <div class="results-header">
          <h4>Tasks</h4>
          <el-button size="small" text @click="aiStore.clearCompleted()">
            Clear
          </el-button>
        </div>
        <div class="results-list">
          <TaskProgress
            v-for="task in aiStore.tasks"
            :key="task.id"
            :task="task"
          />
          <p v-if="aiStore.tasks.length === 0" class="results-empty">
            No tasks yet
          </p>
        </div>
      </FadePanel>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { aiApi, AIModel } from '@/api/ai'
import { characterApi, Character } from '@/api/character'
import { useAIStore } from '@/store/ai'
import FadePanel from '@/components/common/FadePanel.vue'
import NeonButton from '@/components/common/NeonButton.vue'
import ModelSelector from '@/components/ai/ModelSelector.vue'
import PromptEditor from '@/components/ai/PromptEditor.vue'
import TaskProgress from '@/components/ai/TaskProgress.vue'

const props = defineProps<{ id: string; pid: string }>()
const aiStore = useAIStore()

const selectedModel = ref<AIModel | null>(null)
const prompt = ref('')
const negativePrompt = ref('')
const submitting = ref(false)
const characters = ref<Character[]>([])

const params = reactive({
  width: 512,
  height: 512,
  steps: 30,
})

const latestResult = computed(() => {
  const completed = aiStore.completedTasks
  return completed.length > 0 ? completed[0].result_url : null
})

onMounted(async () => {
  try {
    const data: any = await characterApi.list(Number(props.id), Number(props.pid))
    characters.value = data || []
  } catch {
    // 角色列表加载失败不阻塞主流程
  }
})

function handleModelSelect(model: AIModel) {
  selectedModel.value = model
}

async function handleGenerate() {
  if (!selectedModel.value) {
    ElMessage.warning('Please select a model')
    return
  }
  if (!prompt.value.trim()) {
    ElMessage.warning('Please enter a prompt')
    return
  }

  submitting.value = true
  try {
    const data: any = await aiApi.submitTask(Number(props.id), Number(props.pid), {
      model_id: selectedModel.value.id,
      prompt: prompt.value,
      negative_prompt: negativePrompt.value || undefined,
      parameters: params,
    })
    aiStore.addTask({
      id: data.id,
      type: 'image-generation',
      status: 'pending',
      progress: 0,
      created_at: new Date().toISOString(),
    })
    ElMessage.success('Task submitted')
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to submit task')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped lang="scss">
.ai-studio {
  display: flex;
  height: calc(100vh - 96px);
  gap: 16px;
  margin: -24px;
  padding: 16px;
}

.studio-sidebar {
  width: 320px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
}

.sidebar-section {
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 12px;
}

.generate-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
}

.studio-canvas {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--color-bg-surface);
  border: 1px solid var(--border-glow);
  border-radius: 12px;
  overflow: hidden;
}

.canvas-content {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.canvas-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
}

.canvas-empty {
  text-align: center;
  color: var(--color-text-muted);

  &__icon {
    font-size: 64px;
    display: block;
    margin-bottom: 16px;
    opacity: 0.3;
  }

  p {
    font-size: 14px;
  }
}

.studio-results {
  width: 300px;
  overflow-y: auto;
}

.results-panel {
  height: 100%;
  padding: 16px;
}

.results-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;

  h4 {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.results-empty {
  text-align: center;
  font-size: 13px;
  color: var(--color-text-muted);
  padding: 24px 0;
}
</style>
```

- [ ] **Step 6: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/api/ai.ts web/src/components/ai/ web/src/views/studio/
git commit -m "feat: add AI studio with model selector, prompt editor, and task progress"
```

---

### Task 13: WebSocket 工具

**Files:**
- Create: `web/src/utils/websocket.ts`

- [ ] **Step 1: 创建 `web/src/utils/websocket.ts`**

```ts
// web/src/utils/websocket.ts

type MessageHandler = (data: any) => void

interface WSOptions {
  url: string
  token: string
  onMessage?: MessageHandler
  onOpen?: () => void
  onClose?: () => void
  reconnectInterval?: number
  maxRetries?: number
}

export class WebSocketClient {
  private ws: WebSocket | null = null
  private options: Required<WSOptions>
  private retryCount = 0
  private handlers: Map<string, MessageHandler[]> = new Map()
  private closed = false

  constructor(options: WSOptions) {
    this.options = {
      reconnectInterval: 3000,
      maxRetries: 10,
      onMessage: () => {},
      onOpen: () => {},
      onClose: () => {},
      ...options,
    }
  }

  connect() {
    this.closed = false
    const url = `${this.options.url}?token=${this.options.token}`
    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      this.retryCount = 0
      this.options.onOpen()
    }

    this.ws.onmessage = (event: MessageEvent) => {
      try {
        const msg = JSON.parse(event.data)
        this.options.onMessage(msg)
        // 按 type 分发
        const typeHandlers = this.handlers.get(msg.type)
        if (typeHandlers) {
          typeHandlers.forEach((handler) => handler(msg.data))
        }
      } catch {
        // 非 JSON 消息忽略
      }
    }

    this.ws.onclose = () => {
      this.options.onClose()
      if (!this.closed) {
        this.reconnect()
      }
    }

    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  private reconnect() {
    if (this.retryCount >= this.options.maxRetries) {
      return
    }
    this.retryCount++
    setTimeout(() => {
      this.connect()
    }, this.options.reconnectInterval)
  }

  on(type: string, handler: MessageHandler) {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, [])
    }
    this.handlers.get(type)!.push(handler)
  }

  off(type: string, handler?: MessageHandler) {
    if (!handler) {
      this.handlers.delete(type)
      return
    }
    const list = this.handlers.get(type)
    if (list) {
      const idx = list.indexOf(handler)
      if (idx !== -1) list.splice(idx, 1)
    }
  }

  send(data: any) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    }
  }

  close() {
    this.closed = true
    this.ws?.close()
    this.ws = null
    this.handlers.clear()
  }

  get connected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}

// 单例工厂
let instance: WebSocketClient | null = null

export function getWSClient(token: string): WebSocketClient {
  if (instance && instance.connected) {
    return instance
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  instance = new WebSocketClient({
    url: `${protocol}//${host}/ws`,
    token,
  })
  instance.connect()
  return instance
}

export function closeWSClient() {
  instance?.close()
  instance = null
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/utils/websocket.ts
git commit -m "feat: add WebSocket client with auto-reconnect and type-based dispatch"
```

---

### Task 14: API Key 管理页

**Files:**
- Create: `web/src/api/apikey.ts`
- Create: `web/src/views/settings/APIKeyManage.vue`

- [ ] **Step 1: 创建 `web/src/api/apikey.ts`**

```ts
// web/src/api/apikey.ts
import request from './request'

export interface APIKeyItem {
  id: number
  provider: string
  is_default: boolean
  created_at: string
  key_masked?: string
}

export interface CreateAPIKeyReq {
  provider: string
  key_value: string
}

export interface UpdateAPIKeyReq {
  key_value?: string
}

export const apikeyApi = {
  list() {
    return request.get<APIKeyItem[]>('/apikeys')
  },
  create(data: CreateAPIKeyReq) {
    return request.post('/apikeys', data)
  },
  update(id: number, data: UpdateAPIKeyReq) {
    return request.put(`/apikeys/${id}`, data)
  },
  delete(id: number) {
    return request.delete(`/apikeys/${id}`)
  },
}
```

- [ ] **Step 2: 创建 `web/src/views/settings/APIKeyManage.vue`**

```vue
<!-- web/src/views/settings/APIKeyManage.vue -->
<template>
  <div class="apikey-manage">
    <div class="page-header">
      <h2>API Key Management</h2>
      <NeonButton type="primary" @click="showAddDialog = true">
        Add Key
      </NeonButton>
    </div>

    <div class="key-list">
      <GlowCard v-for="key in keys" :key="key.id" class="key-card">
        <div class="key-info">
          <div class="key-provider">
            <span class="provider-badge" :class="key.provider">
              {{ key.provider }}
            </span>
            <el-tag v-if="key.is_default" size="small" type="info">
              Platform Default
            </el-tag>
          </div>
          <p class="key-masked">{{ key.key_masked || '••••••••••••' }}</p>
          <p class="key-time">Added {{ formatTime(key.created_at) }}</p>
        </div>
        <div class="key-actions">
          <el-button text size="small" @click="handleEdit(key)">
            Edit
          </el-button>
          <el-popconfirm
            title="Delete this API Key?"
            @confirm="handleDelete(key.id)"
          >
            <template #reference>
              <el-button text size="small" type="danger">Delete</el-button>
            </template>
          </el-popconfirm>
        </div>
      </GlowCard>

      <div v-if="keys.length === 0 && !loading" class="empty-state">
        <p>No API keys configured</p>
        <p class="empty-hint">
          Add your own API keys or use platform defaults
        </p>
      </div>
    </div>

    <!-- 添加/编辑弹窗 -->
    <el-dialog
      v-model="showAddDialog"
      :title="editingKey ? 'Edit API Key' : 'Add API Key'"
      width="480px"
      class="dark-dialog"
      @closed="resetForm"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
      >
        <el-form-item label="Provider" prop="provider">
          <el-select
            v-model="form.provider"
            placeholder="Select provider"
            :disabled="!!editingKey"
          >
            <el-option label="Kimi" value="kimi" />
            <el-option label="Claude / Kiro" value="claude" />
            <el-option label="Copilot" value="copilot" />
          </el-select>
        </el-form-item>
        <el-form-item label="API Key" prop="key_value">
          <el-input
            v-model="form.key_value"
            type="password"
            show-password
            placeholder="Enter your API key"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">Cancel</el-button>
        <NeonButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ editingKey ? 'Update' : 'Add' }}
        </NeonButton>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { apikeyApi, type APIKeyItem } from '@/api/apikey'
import GlowCard from '@/components/common/GlowCard.vue'
import NeonButton from '@/components/common/NeonButton.vue'

const keys = ref<APIKeyItem[]>([])
const loading = ref(false)
const showAddDialog = ref(false)
const submitting = ref(false)
const editingKey = ref<APIKeyItem | null>(null)
const formRef = ref<FormInstance>()

const form = reactive({
  provider: '',
  key_value: '',
})

const rules: FormRules = {
  provider: [{ required: true, message: 'Please select a provider', trigger: 'change' }],
  key_value: [{ required: true, message: 'Please enter API key', trigger: 'blur' }],
}

async function loadKeys() {
  loading.value = true
  try {
    const data: any = await apikeyApi.list()
    keys.value = data || []
  } catch {
    ElMessage.error('Failed to load API keys')
  } finally {
    loading.value = false
  }
}

function handleEdit(key: APIKeyItem) {
  editingKey.value = key
  form.provider = key.provider
  form.key_value = ''
  showAddDialog.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    if (editingKey.value) {
      await apikeyApi.update(editingKey.value.id, { key_value: form.key_value })
      ElMessage.success('API Key updated')
    } else {
      await apikeyApi.create({ provider: form.provider, key_value: form.key_value })
      ElMessage.success('API Key added')
    }
    showAddDialog.value = false
    await loadKeys()
  } catch (error: any) {
    ElMessage.error(error.message || 'Operation failed')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await apikeyApi.delete(id)
    ElMessage.success('API Key deleted')
    await loadKeys()
  } catch {
    ElMessage.error('Failed to delete')
  }
}

function resetForm() {
  editingKey.value = null
  form.provider = ''
  form.key_value = ''
}

function formatTime(time: string): string {
  return new Date(time).toLocaleDateString()
}

onMounted(loadKeys)
</script>

<style scoped lang="scss">
.apikey-manage {
  max-width: 800px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
}

.key-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.key-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px;
}

.key-provider {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.provider-badge {
  font-size: 13px;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 4px;
  text-transform: capitalize;

  &.kimi { background: rgba(103, 232, 249, 0.15); color: #67E8F9; }
  &.claude { background: rgba(124, 140, 248, 0.15); color: #A5B4FC; }
  &.copilot { background: rgba(110, 231, 183, 0.15); color: #6EE7B7; }
}

.key-masked {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0;
}

.key-time {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 4px 0 0;
}

.key-actions {
  display: flex;
  gap: 4px;
}

.empty-state {
  text-align: center;
  padding: 48px 0;
  color: var(--color-text-muted);

  .empty-hint {
    font-size: 13px;
    margin-top: 8px;
  }
}
</style>
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/src/api/apikey.ts web/src/views/settings/APIKeyManage.vue
git commit -m "feat: add API Key management page with CRUD operations"
```

---

### Task 15: Nginx 配置 + Dockerfile

**Files:**
- Create: `web/nginx.conf`
- Create: `web/Dockerfile`

- [ ] **Step 1: 创建 `web/nginx.conf`**

```nginx
# web/nginx.conf
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # gzip 压缩
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml;
    gzip_min_length 1024;

    # 静态资源缓存
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # API 代理
    location /api/ {
        proxy_pass http://server:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # WebSocket 代理
    location /ws {
        proxy_pass http://server:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    # SPA 路由回退
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

- [ ] **Step 2: 创建 `web/Dockerfile`**

```dockerfile
# web/Dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/tmp/Ai-curton
git add web/nginx.conf web/Dockerfile
git commit -m "feat: add Nginx config and Dockerfile for frontend deployment"
```
