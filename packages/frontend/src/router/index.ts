import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { createRoutePreloader, scheduleIdlePreloads } from './preload'
import { routes } from './routes'

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes,
})

/** 供导航意图（hover/focus）预热页面代码；真正打开页面仍走路由懒加载。 */
export const preloadRoute = createRoutePreloader(router)

router.beforeEach(async (to) => {
  const userStore = useUserStore()

  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    await userStore.checkAuth()
    if (!userStore.isLoggedIn) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }

  // 注：/admin 同时承载「我的内容/上传/API密钥」(任意登录用户)
  // 与「审核/用户/投票等管理功能」(仅管理员)。
  // 管理员门禁已在 AdminView 组件级按 is_admin 隔离，此处不整体拦截，
  // 否则会误伤非管理员的正常功能。
})

scheduleIdlePreloads()

export default router
