<script setup lang="ts">
import { RouterView, useRoute, useRouter } from 'vue-router'
import { useUserStore } from './stores/user'
import { Toaster } from 'vue-sonner'
import { Layout } from '@arco-design/web-vue'
import ErrorBoundary from './components/ErrorBoundary.vue'
import PublicShell from '@/components/PublicShell.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const isPublicShellRoute = computed(
  () => !route.path.startsWith('/content/'),
)
// 详情层负责自身入场；公开整页之间使用轻量过渡，不延迟覆盖层挂载。
const pageTransition = ref(false)
let pageTransitionFrame: number | null = null
let pageTransitionToken = 0
watch(() => route.fullPath, (to, from) => {
  const token = ++pageTransitionToken
  if (pageTransitionFrame !== null) {
    cancelAnimationFrame(pageTransitionFrame)
    pageTransitionFrame = null
  }
  pageTransition.value = false
  const toName = router.resolve(to).name
  const fromName = router.resolve(from).name
  const shouldAnimate = toName !== 'content-detail' && fromName !== 'content-detail' && !to.startsWith('/admin')
  if (!shouldAnimate) return
  if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') {
    pageTransition.value = true
    return
  }
  pageTransitionFrame = window.requestAnimationFrame(() => {
    pageTransitionFrame = null
    if (token === pageTransitionToken) pageTransition.value = true
  })
})

function onVisibilityChange() {
  if (!document.hidden) void userStore.refreshDailyLogin()
}

onMounted(() => {
  if (!userStore.isLoggedIn) userStore.checkAuth()
  document.addEventListener('visibilitychange', onVisibilityChange)
})
onBeforeUnmount(() => {
  if (pageTransitionFrame !== null) cancelAnimationFrame(pageTransitionFrame)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<template>
  <Layout :class="['app-layout', { 'creative-theme app-public-theme': isPublicShellRoute, 'home-gallery-mode': route.path === '/' }]">
      <main role="main">
        <ErrorBoundary>
          <RouterView v-slot="{ Component }">
            <PublicShell v-if="isPublicShellRoute && Component">
              <KeepAlive :include="['HomeView']">
                <component :is="Component" :key="route.fullPath" :class="{ 'public-page-motion': pageTransition }" />
              </KeepAlive>
            </PublicShell>
            <KeepAlive v-else-if="Component" :include="['HomeView']">
              <component :is="Component" :key="route.fullPath" :class="{ 'public-page-motion': pageTransition }" />
            </KeepAlive>
          </RouterView>
        </ErrorBoundary>
      </main>

      <Toaster position="top-center" />
      <ConfirmDialog />
    </Layout>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--creative-canvas, var(--color-bg));
}
.app-layout > main {
  flex: 1 0 auto;
  min-height: 100dvh;
  padding-top: 0 !important;
  background: transparent;
}
.public-page-motion { animation: public-page-in .2s ease-out; }
@keyframes public-page-in {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) {
  .public-page-motion { animation: none; }
}
</style>
