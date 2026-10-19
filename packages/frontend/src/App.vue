<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { Layout } from '@arco-design/web-vue'
import { Toaster } from 'vue-sonner'
import ErrorBoundary from '@/components/ErrorBoundary.vue'
import PublicShell from '@/components/PublicShell.vue'
import { useThemeStore } from '@/stores/theme'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const userStore = useUserStore()

// 主题必须在任何路由组件挂载前落到 html/body 上：详情页不经过 PublicShell，
// 只靠 index.html 预置的 html.dark 会让 body 缺少 arco-theme，Arco 弹层与
// 详情调色板仍是浅色。App 的 setup 早于 RouterView 解析，是可靠的落点。
useThemeStore()

const isPublicShellRoute = computed(() => !route.path.startsWith('/content/'))

// RouterView 的 key 决定「路由变化时是否重建页面实例」。
// 后台切换页签只改 query（?tab=users），若继续用 route.fullPath 做 key，整壳会
// 跟着重挂载——AdminView/AdminNav 的 setup、角标、Teleport 与过渡全部重来一遍。
// 所以 /admin 用「路由记录」做稳定 key（name 为 'admin'）：同一条路由内 query 怎么
// 变都不换 key；离开 /admin（或进入别的后台子路由）时 name 变化，仍照常重建。
// 其余路由原样保留 route.fullPath，行为与改动前完全一致——只有 /admin 走这条分支。
const viewKey = computed(() =>
  route.path === '/admin' || route.path.startsWith('/admin/')
    ? `admin:${String(route.name ?? route.path)}`
    : route.fullPath,
)

// 公开整页之间做「旧页淡出 → 新页上浮淡入」的串行切换（mode=out-in，旧页离场时间很短）。
// 以下情况直接瞬时切换，不做过渡：
// - 详情层负责自身入场，且覆盖层挂载不能被延迟；
// - 后台内部切换页签频繁，过渡只会拖慢；
// - 同一路径只有 query 变化（筛选/搜索），页面骨架没变，淡出再淡入反而像闪屏。
// 路由守卫之前就要决定好过渡名，所以用 flush: 'sync' 的 watch 在路由状态变更的同一刻更新。
const pageTransition = ref<'page-swap' | ''>('')
watch(
  () => [route.path, route.name] as const,
  ([toPath, toName], [fromPath, fromName]) => {
    const skip =
      toName === 'content-detail' ||
      fromName === 'content-detail' ||
      toPath.startsWith('/admin') ||
      fromPath.startsWith('/admin') ||
      toPath === fromPath
    pageTransition.value = skip ? '' : 'page-swap'
  },
  { flush: 'sync' },
)

// 没有全局 scrollBehavior：旧页离场结束后回到顶部，避免新页沿用上一页的滚动位置。
// 首页自己用 onActivated 恢复滚动，晚于这里，所以不会被覆盖。
function onPageLeft() {
  if (typeof window !== 'undefined') window.scrollTo({ top: 0, behavior: 'instant' })
}

// 跨天回到已登录标签页时补一次签到；服务端持有真实日期与发放上限。
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
  <Layout :class="['app-layout', { 'creative-theme app-public-theme': isPublicShellRoute, 'home-gallery-mode': route.path === '/' }]">
    <main role="main">
      <ErrorBoundary>
        <RouterView v-slot="{ Component }">
          <PublicShell v-if="isPublicShellRoute && Component">
            <Transition :name="pageTransition" mode="out-in" @after-leave="onPageLeft">
              <KeepAlive :include="['HomeView']">
                <component :is="Component" :key="viewKey" />
              </KeepAlive>
            </Transition>
          </PublicShell>
          <KeepAlive v-else-if="Component" :include="['HomeView']">
            <component :is="Component" :key="viewKey" />
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
/* 页面切换：离场短而干脆，入场稍长并带一点上浮。只用 opacity 与小位移，
   不用 blur/scale——整页重绘成本高，而且 transform 祖先会改变 position:fixed 后代的定位基准。 */
.page-swap-enter-active {
  transition:
    opacity 0.32s cubic-bezier(0.22, 1, 0.36, 1),
    transform 0.4s cubic-bezier(0.22, 1, 0.36, 1);
}
.page-swap-leave-active {
  transition: opacity 0.14s ease-in;
  pointer-events: none;
}
.page-swap-enter-from {
  opacity: 0;
  transform: translateY(10px);
}
.page-swap-leave-to {
  opacity: 0;
}
</style>
