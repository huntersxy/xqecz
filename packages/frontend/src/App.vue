<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { Layout } from '@arco-design/web-vue'
import { Toaster } from 'vue-sonner'
import ErrorBoundary from '@/components/ErrorBoundary.vue'
import PublicShell from '@/components/PublicShell.vue'
import { useThemeStore } from '@/stores/theme'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

// 主题必须在任何路由组件挂载前落到 html/body 上：详情页不经过 PublicShell，
// 只靠 index.html 预置的 html.dark 会让 body 缺少 arco-theme，Arco 弹层与
// 详情调色板仍是浅色。App 的 setup 早于 RouterView 解析，是可靠的落点。
useThemeStore()

const isPublicShellRoute = computed(() => !route.path.startsWith('/content/'))

// 详情层负责自身入场；公开整页之间使用轻量过渡，且不能延迟覆盖层挂载。
const pageTransition = ref(false)
let transitionFrame: number | null = null
let transitionToken = 0

function cancelTransitionFrame() {
  if (transitionFrame === null) return
  cancelAnimationFrame(transitionFrame)
  transitionFrame = null
}

watch(() => route.fullPath, (to, from) => {
  const token = ++transitionToken
  cancelTransitionFrame()
  pageTransition.value = false

  const toName = router.resolve(to).name
  const fromName = router.resolve(from).name
  const shouldAnimate =
    toName !== 'content-detail' && fromName !== 'content-detail' && !to.startsWith('/admin')
  if (!shouldAnimate) return

  if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') {
    pageTransition.value = true
    return
  }
  transitionFrame = window.requestAnimationFrame(() => {
    transitionFrame = null
    if (token === transitionToken) pageTransition.value = true
  })
})

// 跨天回到已登录标签页时补一次签到；服务端持有真实日期与发放上限。
function onVisibilityChange() {
  if (!document.hidden) void userStore.refreshDailyLogin()
}

onMounted(() => {
  if (!userStore.isLoggedIn) void userStore.checkAuth()
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  cancelTransitionFrame()
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
