import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { aiApi, type GenerationInput } from '@/api/creative'
import type { AIAccount, AIProvider, AISession, AIAsset, AITaskView } from '@/types/schemas'
import { useUserStore } from '@/stores/user'
import { AUTO_IMAGE_SIZE, normalizeImageSize, parseImageSize } from '@/utils/imageSize'
import { remapReferenceMentions } from '@/utils/creativeMentions'

const REMEMBERED_SIZE_KEY = 'xqecz:ai-studio-size'

function readRememberedSize() {
  try {
    const value = localStorage.getItem(REMEMBERED_SIZE_KEY)?.trim() ?? ''
    if (value.toLowerCase() === AUTO_IMAGE_SIZE) return AUTO_IMAGE_SIZE
    return parseImageSize(value) ? normalizeImageSize(value) : AUTO_IMAGE_SIZE
  } catch {
    return AUTO_IMAGE_SIZE
  }
}

function rememberSize(value: string) {
  if (value !== AUTO_IMAGE_SIZE && !parseImageSize(value)) return
  try {
    localStorage.setItem(REMEMBERED_SIZE_KEY, value)
  } catch {
  }
}

export function useAIStudio() {
  const user = useUserStore()
  const isQuotaExempt = computed(() => !!user.user?.is_admin)
  const providers = ref<AIProvider[]>([]),
    sessions = ref<AISession[]>([]),
    tasks = ref<AITaskView[]>([]),
    assets = ref<AIAsset[]>([]),
    likedAssets = ref<AIAsset[]>([])
  const account = ref<AIAccount>({ user_id: 0, available: 0, reserved: 0 })
  const promptEnhancement = ref({ enabled: true, prompt: '' })
  const dailyCreditCap = ref(5)
  const sessionId = ref(0),
    name = ref('未命名创作'),
    prompt = ref(''),
    size = ref(readRememberedSize()),
    quality = ref(''),
    count = ref(1),
    references = ref<{ asset_id: number; note: string }[]>([]),
    parentAssetId = ref(0),
    enhancePrompt = ref(true)
  const error = ref(''),
    saveState = ref(''),
    busy = ref(false),
    ready = ref(false),
    switching = ref(false),
    pending = ref<GenerationInput | null>(null),
    selectedAsset = ref<AIAsset | null>(null)
  const characterIds = ref<number[]>([])
  const taskQuery = ref<Record<string, unknown>>({}),
    moreTasks = ref(false),
    moreAssets = ref(false),
    moreLiked = ref(false)
  let queryFingerprint = '',
    requestEpoch = 0,
    pagedHistory = false
  const availableProviders = computed(() =>
    providers.value.filter(
      (p) => p.enabled && p.owner === 0,
    ),
  )
  const provider = computed(() => availableProviders.value[0])
  const session = computed(() => sessions.value.find((s) => s.id === sessionId.value))
  const cost = computed(() =>
    user.user?.is_admin
      ? 0
      : (provider.value?.quality_costs[quality.value] ?? provider.value?.cost ?? 0) * count.value,
  )
  const input = (): GenerationInput => ({
    session_id: sessionId.value,
    idempotency_key: '',
    prompt: prompt.value,
    provider_id: 0,
    mode: 'site',
    size: size.value,
    quality: quality.value,
    count: count.value,
    references: references.value.map((r) => ({ ...r })),
    parent_asset_id: parentAssetId.value,
    expected_cost: cost.value,
    provider_revision: 0,
    character_ids: [...characterIds.value],
    enhance_prompt: enhancePrompt.value,
  })
  let stopped = false,
    restoring = false,
    saving = false,
    timer: ReturnType<typeof setTimeout> | undefined,
    pollTimer: ReturnType<typeof setTimeout> | undefined
  const controller = new AbortController()
  let saveQueue: Promise<void> = Promise.resolve()
  let savedDraft = ''
  const draftSignature = () => JSON.stringify({ name: name.value || '未命名创作', draft: { ...input(), pending: pending.value } })
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
    const signature = JSON.stringify({ name: savedName, draft })
    const operation = saveQueue.then(async () => {
      if (stopped) return
      if (signature === savedDraft) return
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
        savedDraft = signature
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
    size.value = draft.size?.trim() || readRememberedSize()
    quality.value = draft.quality ?? ''
    count.value = draft.count ?? 1
    references.value = draft.references ?? []
    parentAssetId.value = draft.parent_asset_id ?? 0
    enhancePrompt.value = draft.enhance_prompt ?? true
    pending.value = draft.pending ?? null
    characterIds.value = draft.character_ids ?? []
    taskQuery.value = {}
    selectedAsset.value = null
    saveState.value = '已恢复'
    queueMicrotask(() => {
      restoring = false
      savedDraft = draftSignature()
    })
  }
  async function refresh() {
    const query = { ...taskQuery.value, session_id: sessionId.value }
    const signature = JSON.stringify(query),
      epoch = ++requestEpoch
    const [nextTasks, nextAccount] = await Promise.all([
      aiApi.tasks(query, controller.signal),
      aiApi.account(controller.signal),
    ])
    if (stopped || epoch !== requestEpoch || signature !== JSON.stringify({ ...taskQuery.value, session_id: sessionId.value })) return
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
    const all = nextTasks.flatMap((t) => t.assets)
    const current = selectedAsset.value
    if (current) {
      const fresh = all.find((a) => a.id === current.id)
      if (fresh) selectedAsset.value = fresh
    } else {
      selectedAsset.value = all[0] ?? null
    }
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
    const query = { ...taskQuery.value, session_id: sessionId.value, before_id: before }
    const signature = JSON.stringify({ ...taskQuery.value, session_id: sessionId.value })
    const batch = await aiApi.tasks(query, controller.signal)
    if (stopped || signature !== JSON.stringify({ ...taskQuery.value, session_id: sessionId.value })) return
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
  async function loadLikedAssets() {
    const rows = await aiApi.assets(controller.signal, undefined, true)
    if (stopped) return
    likedAssets.value = rows
    moreLiked.value = rows.length === 100
  }
  async function loadMoreLikedAssets() {
    const before = likedAssets.value.at(-1)?.id
    if (!before) return
    const batch = await aiApi.assets(controller.signal, before, true)
    if (stopped) return
    const ids = new Set(likedAssets.value.map((a) => a.id))
    likedAssets.value.push(...batch.filter((a) => !ids.has(a.id)))
    moreLiked.value = batch.length === 100
  }
  function applyLiked(id: number, liked: boolean, source?: AIAsset) {
    tasks.value = tasks.value.map((row) =>
      row.assets.some((a) => a.id === id)
        ? { ...row, assets: row.assets.map((a) => (a.id === id ? { ...a, liked } : a)) }
        : row,
    )
    assets.value = assets.value.map((a) => (a.id === id ? { ...a, liked } : a))
    if (selectedAsset.value?.id === id) selectedAsset.value = { ...selectedAsset.value, liked }
    if (liked) {
      const existing = likedAssets.value.find((a) => a.id === id)
      if (existing) likedAssets.value = likedAssets.value.map((a) => (a.id === id ? { ...a, liked } : a))
      else {
        const found =
          source ??
          selectedAsset.value ??
          tasks.value.flatMap((row) => row.assets).find((a) => a.id === id) ??
          assets.value.find((a) => a.id === id)
        if (found) likedAssets.value = [{ ...found, liked: true }, ...likedAssets.value]
      }
    } else {
      likedAssets.value = likedAssets.value.filter((a) => a.id !== id)
    }
  }
  async function toggleLike(asset: AIAsset) {
    const next = !asset.liked
    applyLiked(asset.id, next, asset)
    try {
      const row = await aiApi.setLiked(asset.id, next)
      applyLiked(row.id, row.liked, row)
    } catch (e) {
      applyLiked(asset.id, !next, asset)
      throw e
    }
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
    if (sessionId.value === originSession) addReference(row.id)
  }
  async function archiveAsset(id: number) {
    await aiApi.archive(id)
    const before = references.value.map((reference) => reference.asset_id)
    assets.value = assets.value.filter((asset) => asset.id !== id)
    likedAssets.value = likedAssets.value.filter((asset) => asset.id !== id)
    references.value = references.value.filter((reference) => reference.asset_id !== id)
    prompt.value = remapReferenceMentions(prompt.value, before, references.value.map((reference) => reference.asset_id))
    tasks.value = tasks.value.map((row) => ({ ...row, assets: row.assets.filter((asset) => asset.id !== id) }))
    if (parentAssetId.value === id) parentAssetId.value = 0
    if (selectedAsset.value?.id === id) selectedAsset.value = null
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
    const reference = references.value.splice(index, 1)[0]!
    references.value.splice(next, 0, reference)
  }
  function continueFrom(asset: AIAsset) {
    addReference(asset.id)
    parentAssetId.value = asset.id
  }
  async function loadProviders() {
    const cfg = await aiApi.config(controller.signal)
    if (!stopped) {
      providers.value = cfg.providers
      promptEnhancement.value = cfg.prompt_enhancement
      dailyCreditCap.value = cfg.daily_credit_cap ?? 5
    }
  }
  async function switchSession(id: number) {
    if (id === sessionId.value || switching.value || !ready.value) return
    switching.value = true
    try {
      await save()
      const row = await aiApi.session(id, controller.signal)
      if (stopped) return
      const index = sessions.value.findIndex((s) => s.id === id)
      if (index >= 0) sessions.value[index] = row
      restore(row)
      await refresh()
    } finally {
      switching.value = false
    }
  }
  async function createWorkspace(workspaceName = '新工作区') {
    if (switching.value || !ready.value) return
    switching.value = true
    try {
      await save()
      const row = await aiApi.createSession(workspaceName.trim().slice(0, 100) || '新工作区', true)
      if (stopped) return
      sessions.value.unshift(row)
      restore(row)
      await refresh()
    } finally {
      switching.value = false
    }
  }
  async function renameWorkspace(next: string) {
    const trimmed = next.trim().slice(0, 100)
    if (!trimmed || trimmed === name.value) return
    name.value = trimmed
    await save()
  }
  watch(
    [prompt, name, references, size, quality, count, parentAssetId, characterIds, enhancePrompt],
    () => {
      if (!ready.value || restoring || pending.value) return
      saveState.value = '未保存'
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => void run(save), 900)
    },
    { deep: true },
  )
  watch(size, (value) => {
    if (!restoring) rememberSize(value)
  })
  watch(provider, (p) => {
    if (!p) return
    if (quality.value && !p.qualities.includes(quality.value)) quality.value = ''
    count.value = Math.max(1, Math.min(count.value, p.max_count))
    const maxReferences = p.edit_transport === 'none' ? 0 : p.max_references
    if (references.value.length > maxReferences) {
      references.value = references.value.slice(0, maxReferences)
      if (parentAssetId.value && !references.value.some((ref) => ref.asset_id === parentAssetId.value)) {
        parentAssetId.value = 0
      }
    }
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
        promptEnhancement.value = cfg.prompt_enhancement
        dailyCreditCap.value = cfg.daily_credit_cap ?? 5
        sessions.value = rows
        assets.value = library
        moreAssets.value = library.length === 100
        const row = rows[0]
        if (row) restore(row)
        else {
          const fresh = await aiApi.createSession('创作工作台')
          if (stopped) return
          sessions.value.unshift(fresh)
          restore(fresh)
        }
        await Promise.all([refresh(), loadLikedAssets()])
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
    likedAssets,
    account,
    sessionId,
    name,
    prompt,
    size,
    quality,
    count,
    references,
    parentAssetId,
    characterIds,
    enhancePrompt,
    promptEnhancement,
    dailyCreditCap,
    provider,
    availableProviders,
    cost,
    input,
    error,
    saveState,
    busy,
    ready,
    switching,
    pending,
    selectedAsset,
    moreTasks,
    moreAssets,
    moreLiked,
    setHistoryQuery,
    loadMoreTasks,
    loadMoreAssets,
    loadMoreLikedAssets,
    loadLikedAssets,
    run,
    save,
    generate,
    upload,
    archiveAsset,
    toggleLike,
    addReference,
    removeReference,
    moveReference,
    continueFrom,
    refresh,
    loadProviders,
    switchSession,
    createWorkspace,
    renameWorkspace,
  }
}
