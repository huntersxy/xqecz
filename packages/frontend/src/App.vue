<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { Toaster } from 'vue-sonner'
import ErrorBoundary from '@/components/ErrorBoundary.vue'
import PublicShell from '@/components/PublicShell.vue'
import RouteProgress from '@/components/RouteProgress.vue'
import { useThemeStore } from '@/stores/theme'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const userStore = useUserStore()

useThemeStore()


const viewKey = computed(() => {
  if (route.path === '/admin' || route.path.startsWith('/admin/')) {
    return `admin:${String(route.name ?? route.path)}`
  }
  if (route.name === 'content-detail') return 'content-detail'
  return route.fullPath
})

const pageTransition = ref<'page-swap' | ''>('')
let swapAnimated = false
watch(
  () => [route.path, route.name] as const,
  ([toPath], [fromPath]) => {
    const skip = toPath === fromPath
    swapAnimated = !skip
    pageTransition.value = skip ? '' : 'page-swap'
  },
  { flush: 'sync' },
)

function onPageLeave(_element: Element, done: () => void) {
  if (swapAnimated && typeof window !== 'undefined') {
    window.scrollTo({ top: 0, behavior: 'instant' })
  }
  done()
}

function onVisibilityChange() {
  if (!document.hidden) void userStore.refreshDailyLogin()
}

onMounted(() => {
  if (!userStore.isLoggedIn) void userStore.checkAuth()
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<template>
  <div :class="['app-layout', 'creative-theme app-public-theme', { 'home-gallery-mode': route.path === '/' }]">
    <RouteProgress />
    <main role="main">
      <ErrorBoundary>
        <RouterView v-slot="{ Component }">
          <PublicShell v-if="Component">
            <Transition :name="pageTransition" @leave="onPageLeave">
              <KeepAlive :include="['HomeView']">
                <component :is="Component" :key="viewKey" />
              </KeepAlive>
            </Transition>
          </PublicShell>
        </RouterView>
      </ErrorBoundary>
    </main>

    <Toaster position="top-center" />
    <ConfirmDialog />
  </div>
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
.page-swap-enter-active {
  transition:
    opacity var(--dur-enter) var(--ease-out),
    transform var(--dur-enter) var(--ease-out);
}
.page-swap-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
@media (prefers-reduced-motion: reduce) {
  .page-swap-enter-active { transition: none; }
  .page-swap-enter-from { opacity: 1; transform: none; }
}
</style>
