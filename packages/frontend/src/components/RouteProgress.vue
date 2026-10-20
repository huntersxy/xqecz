<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'

/**
 * 顶部细进度条：点击跳转后页面代码（懒加载 chunk）与路由守卫还没完成时，
 * 给一个即时的"正在前往"反馈。
 * - 延迟 SHOW_DELAY_MS 才出现：预热命中的跳转几乎瞬时完成，不该闪一下进度条；
 * - 只在 path 变化时出现：同页改 query（筛选、搜索）不算页面切换；
 * - 完成后冲到 100% 再淡出，失败/被取消的导航同样收尾，不会卡在半路。
 */
const SHOW_DELAY_MS = 120

const router = useRouter()
const progress = ref(0)
const visible = ref(false)
let showTimer: ReturnType<typeof setTimeout> | undefined
let trickleTimer: ReturnType<typeof setInterval> | undefined
let hideTimer: ReturnType<typeof setTimeout> | undefined

function clearTimers() {
  clearTimeout(showTimer)
  clearInterval(trickleTimer)
  showTimer = trickleTimer = undefined
}

function start() {
  clearTimers()
  clearTimeout(hideTimer)
  showTimer = setTimeout(() => {
    visible.value = true
    progress.value = 0.12
    // 越接近终点涨得越慢，永远不会自己涨满。
    trickleTimer = setInterval(() => {
      progress.value += (0.9 - progress.value) * 0.12
    }, 180)
  }, SHOW_DELAY_MS)
}

function finish() {
  const wasShown = visible.value
  clearTimers()
  if (!wasShown) return
  progress.value = 1
  hideTimer = setTimeout(() => {
    visible.value = false
    progress.value = 0
  }, 260)
}

// 必须是 beforeEach：路由懒加载的 chunk 在它之后、beforeResolve 之前才解析，
// 用 beforeResolve 就错过了最需要反馈的那段等待。
const removeBefore = router.beforeEach((to, from) => {
  if (to.path !== from.path) start()
})
const removeAfter = router.afterEach(finish)
const removeError = router.onError(finish)

onBeforeUnmount(() => {
  removeBefore()
  removeAfter()
  removeError()
  clearTimers()
  clearTimeout(hideTimer)
})
</script>

<template>
  <div class="route-progress" :class="{ 'is-visible': visible }" role="presentation" aria-hidden="true">
    <div class="route-progress-bar" :style="{ transform: `scaleX(${progress})` }"></div>
  </div>
</template>

<style scoped>
.route-progress {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 3000;
  height: 2px;
  opacity: 0;
  pointer-events: none;
  transition: opacity .24s ease;
}
.route-progress.is-visible { opacity: 1; }
.route-progress-bar {
  height: 100%;
  background: var(--site-accent, #6b173b);
  box-shadow: 0 0 8px color-mix(in srgb, var(--site-accent, #6b173b) 55%, transparent);
  transform-origin: left center;
  transition: transform .2s ease-out;
}
</style>
