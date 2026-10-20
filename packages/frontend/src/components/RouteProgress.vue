<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'

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
