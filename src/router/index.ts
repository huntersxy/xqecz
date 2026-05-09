import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import { useUserStore } from '@/stores/user'

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
      component: () => import('../views/EasterEggView.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
    },
    {
      path: '/content/:id',
      name: 'content-detail',
      component: () => import('../views/ContentDetailView.vue'),
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('../views/AdminView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/admin/reports',
      name: 'admin-reports',
      component: () => import('../views/AdminReportsView.vue'),
      meta: { requiresAuth: true }
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
  } else {
    next()
  }
})

export default router
