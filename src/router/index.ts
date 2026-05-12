import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
  }
}

// 路由懒加载
const HomeView = () => import('../views/HomeView.vue')
const EasterEggView = () => import('../views/EasterEggView.vue')
const LoginView = () => import('../views/LoginView.vue')
const ContentDetailView = () => import('../views/ContentDetailView.vue')
const UploadView = () => import('../views/UploadView.vue')
const AdminView = () => import('../views/AdminView.vue')
const AdminReportsView = () => import('../views/AdminReportsView.vue')

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/easter-egg',
      name: 'easter-egg',
      component: EasterEggView,
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView,
    },
    {
      path: '/content/:id',
      name: 'content-detail',
      component: ContentDetailView,
    },
    {
      path: '/upload',
      name: 'upload',
      component: UploadView,
      meta: { requiresAuth: true }
    },
    {
      path: '/admin',
      name: 'admin',
      component: AdminView,
      meta: { requiresAuth: true, requiresAdmin: true }
    },
    {
      path: '/admin/reports',
      name: 'admin-reports',
      component: AdminReportsView,
      meta: { requiresAuth: true, requiresAdmin: true }
    },
  ],
})

router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()

  if (!userStore.isLoggedIn) {
    await userStore.checkAuth()
  }

  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else if (to.meta.requiresAdmin && !userStore.user?.is_admin && !userStore.user?.IsAdmin) {
    next({ name: 'home' })
  } else {
    next()
  }
})

// 预加载策略：首页加载完成后，空闲时预加载详情页组件
if (typeof window !== 'undefined') {
  // 使用 requestIdleCallback 或 setTimeout 在浏览器空闲时预加载
  const preloadComponent = () => {
    // 预加载详情页（用户最可能访问的页面）
    ContentDetailView()
  }

  // 页面加载完成后 3 秒，如果浏览器空闲则预加载
  if ('requestIdleCallback' in window) {
    window.requestIdleCallback(() => {
      setTimeout(preloadComponent, 3000)
    })
  } else {
    setTimeout(preloadComponent, 5000)
  }
}

export default router
