import {
  computed,
  onActivated,
  onBeforeUnmount,
  onDeactivated,
  ref,
  shallowRef,
  watch,
  type Ref,
} from 'vue'
import { pollApi } from '@/api'
import { PollDetailSchema, type PollDetail } from '@/types/schemas'

/** Both inline and full-page ballots use server totals; aborting a view never undoes a submitted vote. */
export function usePollBallot(id: Ref<number>) {
  const detail = shallowRef<PollDetail | null>(null)
  const loading = ref(false)
  const error = ref('')
  const voteError = ref('')
  const selected = ref<number | null>(null)
  const submitting = ref(false)
  const resultsVisible = ref(false)
  let controller: AbortController | undefined

  const readableError = (cause: unknown, fallback: string) =>
    cause instanceof Error && cause.name !== 'ZodError' ? cause.message : fallback

  async function loadDetail() {
    controller?.abort()
    const owner = new AbortController()
    controller = owner
    detail.value = null
    error.value = ''
    voteError.value = ''
    selected.value = null
    submitting.value = false
    resultsVisible.value = false
    if (!Number.isSafeInteger(id.value) || id.value <= 0) {
      loading.value = false
      error.value = '投票不存在'
      return
    }
    loading.value = true
    try {
      const response = await pollApi.detail(id.value, owner.signal)
      if (owner.signal.aborted) return
      if (response.code !== 200) throw new Error(response.message || '投票加载失败，请重试')
      detail.value = PollDetailSchema.parse(response.data)
      selected.value = detail.value.my_vote
      resultsVisible.value = detail.value.my_vote !== null
    } catch (cause) {
      if (!owner.signal.aborted) error.value = readableError(cause, '投票暂时无法读取，请重试')
    } finally {
      if (controller === owner) loading.value = false
    }
  }

  const canVote = computed(
    () =>
      detail.value &&
      selected.value !== null &&
      selected.value !== detail.value.my_vote &&
      !submitting.value,
  )

  async function vote() {
    if (!canVote.value || !detail.value || selected.value === null || !controller) return
    const owner = controller
    submitting.value = true
    voteError.value = ''
    try {
      const response = await pollApi.vote(detail.value.poll.id, selected.value, owner.signal)
      if (owner.signal.aborted) return
      if (response.code !== 200) throw new Error(response.message || '投票未成功，请重试')
      detail.value = PollDetailSchema.parse(response.data)
      selected.value = detail.value.my_vote
      resultsVisible.value = true
    } catch (cause) {
      if (!owner.signal.aborted)
        voteError.value = readableError(cause, '投票结果暂时无法读取，请刷新确认')
    } finally {
      if (!owner.signal.aborted) submitting.value = false
    }
  }

  function percentage(index: number) {
    if (!detail.value?.total_votes) return 0
    return Math.round(
      ((detail.value.vote_counts[String(index)] ?? 0) / detail.value.total_votes) * 100,
    )
  }

  function stop() {
    controller?.abort()
  }
  watch(
    id,
    () => {
      if (id.value) void loadDetail()
      else stop()
    },
    { immediate: true },
  )
  // KeepAlive must refresh personal selection as well as totals after a cancelled/finished request.
  onActivated(() => {
    if (controller?.signal.aborted && id.value) void loadDetail()
  })
  onDeactivated(stop)
  onBeforeUnmount(stop)
  return {
    detail,
    loading,
    error,
    voteError,
    selected,
    submitting,
    resultsVisible,
    canVote,
    vote,
    percentage,
    loadDetail,
  }
}
