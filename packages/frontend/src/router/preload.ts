import type { Router } from 'vue-router'

interface PreloadTask {
  loader: () => Promise<unknown>
  priority: number
  delay: number
}

const idleQueue: PreloadTask[] = [
  { loader: () => import('../views/ContentDetailView.vue'), priority: 1, delay: 2000 },
  { loader: () => import('../views/LoginView.vue'), priority: 2, delay: 4000 },
  { loader: () => import('../views/AdminView.vue'), priority: 3, delay: 6000 },
]

function scheduleIdleLoad(task: PreloadTask) {
  const run = () => {
    task.loader().catch(() => {})
  }

  if ('requestIdleCallback' in globalThis) {
    globalThis.requestIdleCallback(() => {
      setTimeout(run, task.delay)
    })
  } else {
    setTimeout(run, task.delay + 2000)
  }
}

export function scheduleIdlePreloads() {
  if (typeof globalThis === 'undefined') return
  idleQueue.sort((a, b) => a.priority - b.priority)
  idleQueue.forEach(scheduleIdleLoad)
}

export function createRoutePreloader(router: Router) {
  const preloading = new Map<string, Promise<unknown>>()

  return function preloadRoute(path: string) {
    const record = router.resolve(path).matched.at(-1)
    const loader = record?.components?.default
    if (!record || typeof loader !== 'function' || preloading.has(record.path)) return
    const loading = Promise.resolve((loader as () => Promise<unknown>)()).catch((error) => {
      preloading.delete(record.path)
      console.warn('页面预加载失败，将在打开时重试', error)
    })
    preloading.set(record.path, loading)
  }
}
