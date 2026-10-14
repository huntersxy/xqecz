import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { aiApi, type GenerationInput } from '@/api/creative'
import type { AIAccount, AIProvider, AISession, AIAsset, AITaskView } from '@/types/schemas'
import { useUserStore } from '@/stores/user'

export function useAIStudio() {
  const user = useUserStore()
  const isQuotaExempt = computed(() => !!user.user?.is_admin)
  const providers = ref<AIProvider[]>([]),
    sessions = ref<AISession[]>([]),
    tasks = ref<AITaskView[]>([]),
    assets = ref<AIAsset[]>([])
  const account = ref<AIAccount>({ user_id: 0, available: 0, reserved: 0 })
  const sessionId = ref(0),
    name = ref('未命名创作'),
    prompt = ref(''),
    mode = ref<'site' | 'byok'>('site'),
    providerId = ref(0),
    size = ref(''),
    quality = ref(''),
    count = ref(1),
    references = ref<{ asset_id: number; note: string }[]>([]),
    parentAssetId = ref(0)
  const error = ref(''),
    saveState = ref(''),
    busy = ref(false),
    ready = ref(false),
    pending = ref<GenerationInput | null>(null),
    selectedAsset = ref<AIAsset | null>(null)
  const characterIds = ref<number[]>([])
  const taskQuery = ref<Record<string, unknown>>({}),
    moreTasks = ref(false),
    moreAssets = ref(false),
    moreSessions = ref(false)
  let queryFingerprint = '',
    requestEpoch = 0,
    pagedHistory = false,
    sessionCursor = 0
  const availableProviders = computed(() =>
    providers.value.filter(
      (p) => p.enabled && (mode.value === 'site' ? p.owner === 0 : p.owner === user.user?.id),
    ),
  )
  const provider = computed(() => availableProviders.value.find((p) => p.id === providerId.value))
  const session = computed(() => sessions.value.find((s) => s.id === sessionId.value))
  const cost = computed(() =>
    mode.value === 'byok' || user.user?.is_admin
      ? 0
      : (provider.value?.quality_costs[quality.value] ?? provider.value?.cost ?? 0) * count.value,
  )
  const input = (): GenerationInput => ({
    session_id: sessionId.value,
    idempotency_key: '',
    prompt: prompt.value,
    provider_id: providerId.value,
    mode: mode.value,
    size: size.value,
    quality: quality.value,
    count: count.value,
    references: references.value.map((r) => ({ ...r })),
    parent_asset_id: parentAssetId.value,
    expected_cost: cost.value,
    provider_revision: provider.value?.revision ?? 0,
    character_ids: [...characterIds.value],
  })
  let stopped = false,
    restoring = false,
    saving = false,
    timer: ReturnType<typeof setTimeout> | undefined,
    pollTimer: ReturnType<typeof setTimeout> | undefined
  const controller = new AbortController()
  let saveQueue: Promise<void> = Promise.resolve()
  async function run(action: () => Promise<unknown>) {
    error.value = ''
    try {
      await action()
    } catch (e) {
      if (!stopped) error.value = (e as Error).message || '操作失败'
    }
  }
  async function save() {
    if (!session.value || !ready.value || restoring || stopped) return
    const id = session.value.id,
      savedName = name.value || '未命名创作',
      draft = JSON.parse(JSON.stringify({ ...input(), pending: pending.value }))
    const operation = saveQueue.then(async () => {
      if (stopped) return
      const current = sessions.value.find((s) => s.id === id)
      if (!current) return
      saving = true
      saveState.value = '保存中…'
      try {
        const row = await aiApi.saveSession(
          id,
          savedName,
          draft,
          current.version,
          controller.signal,
        )
        if (stopped) return
        const index = sessions.value.findIndex((s) => s.id === id)
        if (index >= 0) sessions.value[index] = row
        saveState.value = '已保存'
      } catch (e) {
        if (!stopped) {
          saveState.value = '保存失败'
          error.value = (e as Error).message
        }
        throw e
      } finally {
        saving = false
      }
    })
    // A failed save is reported to its owner but must not poison all later saves.
    saveQueue = operation.then(
      () => undefined,
      () => undefined,
    )
    return operation
  }
  function restore(row: AISession) {
    restoring = true
    sessionId.value = row.id
    name.value = row.name
    let draft: Partial<GenerationInput> & { pending?: GenerationInput } = {}
    try {
      draft = JSON.parse(row.draft)
    } catch {
      error.value = '草稿格式不完整，已恢复会话名称'
    }
    prompt.value = draft.prompt ?? ''
    mode.value = draft.mode === 'byok' ? 'byok' : 'site'
    providerId.value = draft.provider_id ?? 0
    size.value = draft.size ?? ''
    quality.value = draft.quality ?? ''
    count.value = draft.count ?? 1
    references.value = draft.references ?? []
    parentAssetId.value = draft.parent_asset_id ?? 0
    pending.value = draft.pending ?? null
    characterIds.value = draft.character_ids ?? []
    taskQuery.value = { session_id: row.id }
    const chosen =
      availableProviders.value.find((p) => p.id === providerId.value) ?? availableProviders.value[0]
    providerId.value = chosen?.id ?? 0
    if (chosen && !chosen.sizes.includes(size.value)) size.value = chosen.sizes[0] ?? ''
    selectedAsset.value =
      tasks.value.filter((t) => t.task.session_id === row.id).flatMap((t) => t.assets)[0] ?? null
    localStorage.setItem(`xqecz:ai-session:${user.user?.id}`, String(row.id))
    saveState.value = '已恢复'
    queueMicrotask(() => {
      restoring = false
    })
  }
  async function switchSession(id: number) {
    if (timer) clearTimeout(timer)
    await save()
    const row = sessions.value.find((s) => s.id === id)
    if (row) restore(row)
  }
  async function newSession() {
    await save()
    const row = await aiApi.createSession('新的创作')
    sessions.value.unshift(row)
    restore(row)
  }
  async function refresh() {
    const signature = JSON.stringify(taskQuery.value),
      epoch = ++requestEpoch
    const [nextTasks, nextAccount] = await Promise.all([
      aiApi.tasks(taskQuery.value, controller.signal),
      aiApi.account(controller.signal),
    ])
    if (stopped || epoch !== requestEpoch || signature !== JSON.stringify(taskQuery.value)) return
    const oldest = nextTasks.at(-1)?.task.id ?? Infinity
    const older =
      signature === queryFingerprint && pagedHistory && nextTasks.length === 100
        ? tasks.value.filter((t) => t.task.id < oldest)
        : []
    if (signature !== queryFingerprint) pagedHistory = false
    tasks.value = [...nextTasks, ...older]
    account.value = nextAccount
    queryFingerprint = signature
    if (!pagedHistory) moreTasks.value = nextTasks.length === 100
    if (
      selectedAsset.value &&
      !nextTasks.some((t) => t.assets.some((a) => a.id === selectedAsset.value!.id))
    )
      selectedAsset.value = null
    if (!selectedAsset.value)
      selectedAsset.value =
        nextTasks
          .filter((t) => t.task.session_id === sessionId.value)
          .flatMap((t) => t.assets)[0] ?? null
  }
  async function setHistoryQuery(query: Record<string, unknown>) {
    if (JSON.stringify(query) === JSON.stringify(taskQuery.value)) return
    taskQuery.value = query
    pagedHistory = false
    await refresh()
  }
  async function loadMoreTasks() {
    const before = tasks.value.at(-1)?.task.id
    if (!before) return
    const signature = JSON.stringify(taskQuery.value)
    const batch = await aiApi.tasks({ ...taskQuery.value, before_id: before }, controller.signal)
    if (stopped || signature !== JSON.stringify(taskQuery.value)) return
    const ids = new Set(tasks.value.map((t) => t.task.id))
    tasks.value.push(...batch.filter((t) => !ids.has(t.task.id)))
    moreTasks.value = batch.length === 100
    pagedHistory = true
  }
  async function loadMoreAssets() {
    const before = assets.value.at(-1)?.id
    if (!before) return
    const batch = await aiApi.assets(controller.signal, before)
    if (stopped) return
    const ids = new Set(assets.value.map((a) => a.id))
    assets.value.push(...batch.filter((a) => !ids.has(a.id)))
    moreAssets.value = batch.length === 100
  }
  async function loadMoreSessions() {
    if (!sessionCursor) return
    const batch = await aiApi.sessions(controller.signal, sessionCursor)
    if (stopped) return
    const ids = new Set(sessions.value.map((s) => s.id))
    sessions.value.push(...batch.filter((s) => !ids.has(s.id)))
    sessionCursor = batch.at(-1)?.id ?? 0
    moreSessions.value = batch.length === 100
  }
  async function poll() {
    if (stopped) return
    if (!document.hidden) {
      try {
        await refresh()
      } catch (e) {
        if (!stopped && !error.value) error.value = `任务状态刷新失败：${(e as Error).message}`
      }
    }
    if (!stopped)
      pollTimer = setTimeout(
        poll,
        tasks.value.some((t) => ['queued', 'running'].includes(t.task.status)) ? 3000 : 10000,
      )
  }
  async function generate() {
    if (busy.value) return
    busy.value = true
    try {
      if (!pending.value) {
        const draft = input()
        const quote = await aiApi.quote(draft)
        if (quote.cost !== draft.expected_cost)
          throw new Error('生成价格已变化，请刷新供应商后重新确认')
        if (quote.cost > account.value.available) throw new Error('生成额度不足')
        pending.value = { ...draft, idempotency_key: crypto.randomUUID() }
      }
      await save()
      const task = await aiApi.submit(pending.value!, controller.signal)
      pending.value = null
      await save()
      await refresh()
      return task
    } catch (e) {
      if ((e as Error).name === 'BusinessError') {
        pending.value = null
        await save()
      }
      throw e
    } finally {
      busy.value = false
    }
  }
  async function upload(file: File) {
    const originSession = sessionId.value
    const row = await aiApi.upload(file, controller.signal)
    if (stopped) return
    if (!assets.value.some((a) => a.id === row.id)) assets.value.unshift(row)
    // Uploaded assets stay in the library; only their originating draft receives the reference.
    if (sessionId.value === originSession) addReference(row.id)
  }
  function addReference(id: number) {
    if (references.value.some((r) => r.asset_id === id)) return
    if (references.value.length >= (provider.value?.max_references ?? 0))
      throw new Error('参考图数量已达供应商上限')
    references.value.push({ asset_id: id, note: '' })
  }
  function removeReference(index: number) {
    const removed = references.value.splice(index, 1)[0]
    if (removed?.asset_id === parentAssetId.value) parentAssetId.value = 0
  }
  function moveReference(index: number, direction: number) {
    const next = index + direction
    if (
      index < 0 ||
      index >= references.value.length ||
      next < 0 ||
      next >= references.value.length ||
      next === index
    )
      return
    // Move into the target slot instead of swapping: intermediate images keep
    // their relative order when a drag crosses more than one reference.
    const reference = references.value.splice(index, 1)[0]!
    references.value.splice(next, 0, reference)
  }
  function continueFrom(asset: AIAsset) {
    addReference(asset.id)
    parentAssetId.value = asset.id
  }
  async function loadProviders() {
    const cfg = await aiApi.config(controller.signal)
    if (!stopped) providers.value = cfg.providers
  }
  watch(
    [prompt, name, references, mode, providerId, size, quality, count, parentAssetId, characterIds],
    () => {
      if (!ready.value || restoring || pending.value) return
      saveState.value = '未保存'
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => void run(save), 900)
    },
    { deep: true },
  )
  watch(availableProviders, (rows) => {
    if (!rows.some((p) => p.id === providerId.value)) providerId.value = rows[0]?.id ?? 0
  })
  watch(provider, (p) => {
    if (!p) return
    if (!p.sizes.includes(size.value)) size.value = p.sizes[0] ?? ''
    if (quality.value && !p.qualities.includes(quality.value)) quality.value = ''
    count.value = Math.max(1, Math.min(count.value, p.max_count))
  })
  const visibility = () => {
    if (!document.hidden) void run(refresh)
  }
  onMounted(
    () =>
      void run(async () => {
        const [cfg, rows, library] = await Promise.all([
          aiApi.config(controller.signal),
          aiApi.sessions(controller.signal),
          aiApi.assets(controller.signal),
        ])
        if (stopped) return
        providers.value = cfg.providers
        sessions.value = rows
        assets.value = library
        sessionCursor = rows.at(-1)?.id ?? 0
        moreSessions.value = rows.length === 100
        moreAssets.value = library.length === 100
        const stored = Number(localStorage.getItem(`xqecz:ai-session:${user.user?.id}`))
        let row = rows.find((r) => r.id === stored)
        if (!row && stored) {
          try {
            row = await aiApi.session(stored, controller.signal)
            if (!stopped) sessions.value.push(row)
          } catch (e) {
            if (stopped) return
            error.value = `先前会话无法恢复：${(e as Error).message}`
          }
        }
        row ??= rows[0]
        if (row) restore(row)
        else {
          const fresh = await aiApi.createSession('第一次创作')
          if (stopped) return
          sessions.value.unshift(fresh)
          restore(fresh)
        }
        await refresh()
        ready.value = true
        pollTimer = setTimeout(poll, 3000)
        document.addEventListener('visibilitychange', visibility)
      }),
  )
  onBeforeUnmount(() => {
    stopped = true
    controller.abort()
    if (timer) clearTimeout(timer)
    if (pollTimer) clearTimeout(pollTimer)
    document.removeEventListener('visibilitychange', visibility)
  })
  return {
    isQuotaExempt,
    providers,
    sessions,
    tasks,
    assets,
    account,
    sessionId,
    name,
    prompt,
    mode,
    providerId,
    size,
    quality,
    count,
    references,
    parentAssetId,
    characterIds,
    provider,
    availableProviders,
    cost,
    input,
    error,
    saveState,
    busy,
    ready,
    pending,
    selectedAsset,
    moreTasks,
    moreAssets,
    moreSessions,
    setHistoryQuery,
    loadMoreTasks,
    loadMoreAssets,
    loadMoreSessions,
    run,
    save,
    switchSession,
    newSession,
    generate,
    upload,
    addReference,
    removeReference,
    moveReference,
    continueFrom,
    refresh,
    loadProviders,
  }
}
