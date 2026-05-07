// web/src/router/index.ts
import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/store/user'

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
        path: 'workspace/:id/portfolio/:pid/novels',
        name: 'NovelList',
        component: () => import('@/views/novel/NovelList.vue'),
        meta: { title: 'Novels' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid/butler',
        name: 'NovelButler',
        component: () => import('@/views/novel/NovelButlerPage.vue'),
        meta: { title: 'Novel Butler' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid/novel/:nid/overview',
        name: 'NovelOverview',
        component: () => import('@/views/novel/NovelOverview.vue'),
        meta: { title: 'Novel Overview' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid/novel/:nid/world-builder',
        name: 'WorldBuilder',
        component: () => import('@/views/novel/WorldBuilderPage.vue'),
        meta: { title: 'World Builder' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid/novel/:nid',
        name: 'NovelWorkshop',
        component: () => import('@/views/novel/NovelWorkshop.vue'),
        meta: { title: 'Novel Workshop' },
        props: true,
      },
      {
        path: 'workspace/:id/portfolio/:pid/outline',
        name: 'OutlineWorkshop',
        component: () => import('@/views/novel/OutlineWorkshop.vue'),
        meta: { title: 'Outline Workshop' },
        props: true,
      },
      {
        path: 'settings/apikeys',
        name: 'APIKeyManage',
        component: () => import('@/views/settings/APIKeyManage.vue'),
        meta: { title: 'API Keys' },
      },
      // 记忆管理
      {
        path: 'memories',
        name: 'MyMemories',
        component: () => import('@/views/memory/MyMemories.vue'),
        meta: { title: 'My Memories' },
      },
      // 用户风格库
      {
        path: 'my-styles',
        name: 'MyStyles',
        component: () => import('@/views/style/MyStyles.vue'),
        meta: { title: 'My Styles' },
      },
      // 记忆市场
      {
        path: 'market',
        name: 'MemoryMarket',
        component: () => import('@/views/market/MemoryMarket.vue'),
        meta: { title: 'Memory Market' },
      },
      {
        path: 'market/:mid',
        name: 'MemoryDetail',
        component: () => import('@/views/market/MemoryDetail.vue'),
        meta: { title: 'Memory Detail' },
        props: true,
      },
      // 钱包
      {
        path: 'wallet',
        name: 'WalletPage',
        component: () => import('@/views/wallet/WalletPage.vue'),
        meta: { title: 'Wallet' },
      },
      // 写手等级
      {
        path: 'writer-level',
        name: 'WriterLevel',
        component: () => import('@/views/settings/WriterLevel.vue'),
        meta: { title: 'Writer Level' },
      },
      // 管理员审核
      {
        path: 'admin/memory-review',
        name: 'MemoryReview',
        component: () => import('@/views/admin/MemoryReview.vue'),
        meta: { title: 'Memory Review', requiresAdmin: true },
      },
      // 用户管理
      {
        path: 'admin/users',
        name: 'UserManage',
        component: () => import('@/views/admin/UserManage.vue'),
        meta: { title: 'User Management', requiresAdmin: true },
      },
      // 模型调试面板
      {
        path: 'admin/model-debug',
        name: 'ModelDebugPanel',
        component: () => import('@/views/admin/ModelDebugPanel.vue'),
        meta: { title: 'Model Debug', requiresAdmin: true },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue'),
    meta: { requiresAuth: false, title: '404' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫：未登录跳转 /login，非 admin 拦截管理页面
router.beforeEach(async (to, _from, next) => {
  const token = localStorage.getItem('access_token')
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth !== false)

  if (requiresAuth && !token) {
    next({ path: '/login', query: { redirect: to.fullPath } })
    return
  }

  if ((to.path === '/login' || to.path === '/register') && token) {
    next({ path: '/workspaces' })
    return
  }

  // admin 路由守卫
  const requiresAdmin = to.matched.some((record) => record.meta.requiresAdmin)
  if (requiresAdmin && token) {
    const userStore = useUserStore()
    // 若 profile 尚未加载，先拉取
    if (!userStore.profile) {
      try {
        await userStore.fetchProfile()
      } catch {
        next({ path: '/login' })
        return
      }
    }
    if (userStore.profile?.role !== 'admin') {
      next({ path: '/workspaces' })
      return
    }
  }

  next()

  // 设置页面标题
  if (to.meta.title) {
    document.title = `${to.meta.title} | Story-Maker`
  }
})

export default router
