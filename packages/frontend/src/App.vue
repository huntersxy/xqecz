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

// 站点外壳（左侧栏 + 内容槽）在所有路由下都常驻。
// 旧写法把 /content/:id 排除在外：v-if 在进入详情的瞬间会把 PublicShell 连同其中
// 的 KeepAlive 一起销毁，首页缓存实例随之丢失，返回时只能重新挂载、重新排版，
// 滚动位置回不到原处。现在详情是一层盖在外壳上的沉浸式浮层（自身 fixed inset:0
// 且不透明），外壳不动，KeepAlive 有了稳定宿主，返回时实例、瀑布流布局与滚动
// 位置都能原样保留。后台同样依赖这层外壳承载侧栏导航的 Teleport 落点。

// RouterView 的 key 决定「路由变化时是否重建页面实例」。
// 后台切换页签只改 query（?tab=users），若继续用 route.fullPath 做 key，整壳会
// 跟着重挂载——AdminView/AdminNav 的 setup、角标、Teleport 与过渡全部重来一遍。
// 所以 /admin 用「路由记录」做稳定 key（name 为 'admin'）：同一条路由内 query 怎么
// 变都不换 key；离开 /admin（或进入别的后台子路由）时 name 变化，仍照常重建。
// 其余路由原样保留 route.fullPath，行为与改动前完全一致——只有 /admin 走这条分支。
const viewKey = computed(() => {
  if (route.path === '/admin' || route.path.startsWith('/admin/')) {
    return `admin:${String(route.name ?? route.path)}`
  }
  // 详情页内部切作品（走马灯、上一/下一个、键盘）只改 params，页面骨架没变。
  // 若继续用 fullPath 做 key，每次点击都会重建整个浮层：骨架屏、入场动画、
  // 走马灯滚动位置全部重来一遍，观感就是「点一下重载一次」。用稳定 key 后同一个
  // 实例原地换内容，组件内部的 watch(route.params.id) 接管数据刷新。
  if (route.name === 'content-detail') return 'content-detail'
  return route.fullPath
})

// 整页只做轻淡入，旧页立即退出：不让离场动画挡住新页挂载、请求与首帧。
// 以下情况直接瞬时切换，不做过渡：
// - 详情层负责自身入场，且覆盖层挂载不能被延迟；
// - 后台内部切换页签频繁，过渡只会拖慢；
// - 同一路径只有 query 变化（筛选/搜索），页面骨架没变，淡出再淡入反而像闪屏。
// 路由守卫之前就要决定好过渡名，所以用 flush: 'sync' 的 watch 在路由状态变更的同一刻更新。
const pageTransition = ref<'page-swap' | ''>('')
// 只有真正的整页切换（page-swap）才需要在离场结束后回到顶部。
// 详情层进出、后台切页签、同路径换 query 都用空 name 瞬时切换，它们同样会触发
// leave 钩子；若无条件回到顶部，会把首页恢复的滚动位置清成 0。
let swapAnimated = false
watch(
  () => [route.path, route.name] as const,
  ([toPath, toName], [fromPath, fromName]) => {
    const skip =
      toName === 'content-detail' ||
      fromName === 'content-detail' ||
      toPath.startsWith('/admin') ||
      fromPath.startsWith('/admin') ||
      toPath === fromPath
    swapAnimated = !skip
    pageTransition.value = skip ? '' : 'page-swap'
  },
  { flush: 'sync' },
)

// 没有全局 scrollBehavior：移除旧页时回到顶部，避免新页沿用上一页的滚动位置。
// 首页自己用 onActivated 恢复滚动，晚于这里，所以不会被覆盖。
function onPageLeave(_element: Element, done: () => void) {
  if (swapAnimated && typeof window !== 'undefined') {
    window.scrollTo({ top: 0, behavior: 'instant' })
  }
  done()
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
  <Layout :class="['app-layout', 'creative-theme app-public-theme', { 'home-gallery-mode': route.path === '/' }]">
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
/* 首帧就可读、可点击。整页不做 transform，避免改变 fixed 弹层的定位基准。 */
.page-swap-enter-active {
  transition: opacity 180ms cubic-bezier(0.22, 1, 0.36, 1);
}
.page-swap-enter-from {
  opacity: .72;
}
@media (prefers-reduced-motion: reduce) {
  .page-swap-enter-active { transition-duration: 80ms; }
  .page-swap-enter-from { opacity: .9; }
}
</style>
