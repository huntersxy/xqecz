import { onBeforeUnmount, onDeactivated, ref, shallowRef } from 'vue'
import { pollApi } from '@/api'
import { PollListResponseSchema, type Poll } from '@/types/schemas'

export function usePollList(pageSize: number) {
  const rows = shallowRef<Poll[]>([])
  const total = ref(0)
  const totalPages = ref(0)
  const loading = ref(false)
  const error = ref('')
  let controller: AbortController | undefined
  let pendingPage: number | undefined

  async function load(page = 1) {
    if (loading.value && pendingPage === page && !controller?.signal.aborted) return
    controller?.abort()
    const owner = new AbortController()
    controller = owner
    pendingPage = page
    loading.value = true
    error.value = ''
    try {
      const response = await pollApi.list({ page, page_size: pageSize }, owner.signal)
      if (owner.signal.aborted) return
      if (response.code !== 200) throw new Error(response.message || '投票加载失败，请重试')
      const data = PollListResponseSchema.parse(response.data)
      rows.value = data.list
      total.value = data.total
      totalPages.value = data.total_page
    } catch (cause) {
      if (!owner.signal.aborted)
        error.value =
          cause instanceof Error && cause.name !== 'ZodError'
            ? cause.message
            : '投票数据暂时无法读取，请重试'
    } finally {
      if (controller === owner) {
        loading.value = false
        pendingPage = undefined
      }
    }
  }

  function stop() {
    controller?.abort()
    loading.value = false
    pendingPage = undefined
  }
  onDeactivated(stop)
  onBeforeUnmount(stop)
  return { rows, total, totalPages, loading, error, load, stop }
}
