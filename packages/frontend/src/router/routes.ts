import type { RouteRecordRaw } from 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
  }
}

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: () => import('../views/HomeView.vue'),
  },
  {
    path: '/quick-upload',
    name: 'quick-upload',
    component: () => import('../views/QuickUploadView.vue'),
  },
  {
    path: '/studio',
    name: 'ai-studio',
    component: () => import('../views/AIStudioView.vue'),
    meta: { requiresAuth: true },
  },
  { path: '/characters', name: 'characters', component: () => import('../views/CreativeCatalogView.vue') },
  { path: '/characters/:id', name: 'character-detail', component: () => import('../views/CreativeCatalogView.vue') },
  { path: '/ideas', name: 'ideas', component: () => import('../views/IdeasView.vue') },
  { path: '/ideas/:id', name: 'idea-detail', component: () => import('../views/IdeasView.vue') },
  { path: '/polls', name: 'polls', component: () => import('../views/PollsView.vue') },
  { path: '/polls/:id', name: 'poll-detail', component: () => import('../views/PollsView.vue') },
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
    meta: { requiresAuth: true },
  },
]
