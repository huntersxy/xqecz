import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { createAsyncComponent } from '@/utils/asyncComponent'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
  }
}

// 路由懒加载 - 使用 createAsyncComponent 提供更好的加载体验
const HomeView = createAsyncComponent(() => import('../views/HomeView.vue'))
const EasterEggView = createAsyncComponent(() => import('../views/EasterEggView.vue'))
const LoginView = createAsyncComponent(() => import('../views/LoginView.vue'))
const ContentDetailView = createAsyncComponent(() => import('../views/ContentDetailView.vue'))
const UploadView = createAsyncComponent(() => import('../views/UploadView.vue'))
const AdminView = createAsyncComponent(() => import('../views/AdminView.vue'))
const AdminReportsView = createAsyncComponent(() => import('../views/AdminReportsView.vue'))

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

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()

  // 不阻塞路由，直接跳转
  // 认证检查在后台异步进行
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    // 先触发检查，但不等待结果
    userStore.checkAuth().then(() => {
      // 检查完成后，如果仍然未登录且在需要认证的页面，重定向
      if (!userStore.isLoggedIn) {
        next({ name: 'login', query: { redirect: to.fullPath } })
      } else {
        next()
      }
    })
  } else if (to.meta.requiresAdmin && !userStore.user?.is_admin && !userStore.user?.IsAdmin) {
    next({ name: 'home' })
  } else {
    next()
  }
})

// 预加载策略：首页加载完成后，空闲时预加载详情页组件
if (typeof window !== 'undefined') {
  const preloadComponent = () => {
    import('../views/ContentDetailView.vue')
  }

  if ('requestIdleCallback' in window) {
    window.requestIdleCallback(() => {
      setTimeout(preloadComponent, 3000)
    })
  } else {
    setTimeout(preloadComponent, 5000)
  }
}

export default router
