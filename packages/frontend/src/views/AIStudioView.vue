<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { useAIStudio } from '@/composables/useAIStudio'
import { useStudioDock } from '@/composables/useStudioDock'
import { useUserStore } from '@/stores/user'
import { aiApi } from '@/api/creative'
import { contentApi } from '@/api'
import type { AIAsset, AITaskView } from '@/types/schemas'
import {
  IconImage,
  IconPlus,
  IconSettings,
  IconSend,
  IconFile,
  IconLoading,
  IconHeart,
  IconHeartFill,
  IconDownload,
  IconShareExternal,
  IconMore,
  IconMenu,
  IconClose,
  IconEdit,
  IconCheck,
  IconHistory,
} from '@arco-design/web-vue/es/icon'
import { useConfirm } from '@/composables/useToast'
import CreativeImage from '@/components/CreativeImage.vue'
import CreativeLibrary from '@/components/CreativeLibrary.vue'
import StudioReferences from '@/components/StudioReferences.vue'
import StudioSizePicker from '@/components/StudioSizePicker.vue'
import AutoMetadataOption from '@/components/AutoMetadataOption.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import '@/assets/creative-pages.css'
import { referenceMentionAt, remapReferenceMentions } from '@/utils/creativeMentions'
import { formatImageSize, normalizeImageSize, parseImageSize, AUTO_IMAGE_SIZE } from '@/utils/imageSize'

const studio = useAIStudio(),
  router = useRouter(),
  route = useRoute(),
  userStore = useUserStore(),
  { confirm } = useConfirm()
const {
  tasks,
  assets,
  likedAssets,
  sessions,
  account,
  sessionId,
  name,
  prompt,
  providerId,
  size,
  quality,
  count,
  references,
  provider,
  availableProviders,
  cost,
  error,
  busy,
  ready,
  switching,
  pending,
  selectedAsset,
  archiveAsset,
  enhancePrompt,
  promptEnhancement,
  moreTasks,
  moreLiked,
} = studio

// ── Layout state ─────────────────────────────────────────────────────────────
const sidebarOpen = ref(false)
const sidebarTab = ref<'workspace' | 'liked'>('workspace')
const viewMode = ref<'single' | 'all'>('single')
const promptExpanded = ref(false)
const activeTaskId = ref(0)
const workspaceMenuOpen = ref(false)
const renaming = ref(false)
const renameDraft = ref('')
const settingsOpen = ref(false)
const resultOpen = ref(false)
const sizePickerOpen = ref(false)
const publishOpen = ref(false)
const previewReference = ref(0)
const resourceOpen = ref(false)
const resourceTab = ref<'library' | 'history' | 'idea'>('library')
const zoom = ref(1)
const search = ref('')
const status = ref('')
const uploading = ref(false)
const confirming = ref(false)
const dragging = ref(false)
const awaitingTask = ref(0)

const promptArea = ref<HTMLTextAreaElement>(),
  fileInput = ref<HTMLInputElement>()
const composerElement = ref<HTMLElement>()
const dockStyle = useStudioDock(composerElement)

const resourceTabs = [
  { key: 'library' as const, label: '角色 / 模板', icon: IconFile },
  { key: 'history' as const, label: '我的图片', icon: IconImage },
  { key: 'idea' as const, label: '投稿创意', icon: IconPlus },
]
const resourceTitle = computed(
  () => resourceTabs.find((tab) => tab.key === resourceTab.value)?.label ?? '资源',
)

const labels: Record<string, string> = {
  queued: '排队中',
  running: '生成中',
  succeeded: '完成',
  failed: '失败',
  cancelled: '已取消',
  unknown: '待核查',
}
const username = computed(() => userStore.user?.username || '创作者')
const composerLocked = computed(
  () => !ready.value || !!pending.value || busy.value || uploading.value || confirming.value,
)
const canGenerate = computed(
  () =>
    ready.value &&
    !busy.value &&
    !uploading.value &&
    !confirming.value &&
    (!!pending.value ||
      (!!provider.value &&
        provider.value.has_key &&
        !!prompt.value.trim())),
)
const quotaNote = computed(() => `本次 ${cost.value} 额度 · 可用 ${account.value.available}`)
const dailyCreditTotal = computed(
  () => (account.value.daily_available ?? 0) + (account.value.daily_reserved ?? 0),
)
const otherCreditTotal = computed(() =>
  Math.max(0, account.value.available + account.value.reserved - dailyCreditTotal.value),
)
const quotaVisible = computed(() => !studio.isQuotaExempt.value)
const shownTasks = computed(() =>
  tasks.value.filter(
    (row) =>
      (!search.value || row.task.prompt.includes(search.value)) &&
      (!status.value || row.task.status === status.value),
  ),
)
const failedTaskRows = computed(() =>
  shownTasks.value.filter((row) => ['failed', 'cancelled'].includes(row.task.status)),
)
const activeTask = computed(() =>
  tasks.value.find((row) => ['queued', 'running', 'unknown'].includes(row.task.status)),
)
/** The task currently on stage: explicit history choice, or the owner of the shown asset. */
const stageTask = computed(() => {
  const chosen = tasks.value.find((row) => row.task.id === activeTaskId.value)
  if (chosen) return chosen
  if (!selectedAsset.value) return null
  return (
    tasks.value.find((row) => row.assets.some((asset) => asset.id === selectedAsset.value!.id)) ??
    null
  )
})
/** Every result currently visible in the history list; "全部" mirrors that set. */
const visibleAssets = computed(() => {
  const rows = shownTasks.value.length ? shownTasks.value : tasks.value
  const seen = new Set<number>()
  const out: AIAsset[] = []
  for (const row of rows) {
    for (const asset of row.assets) {
      if (seen.has(asset.id)) continue
      seen.add(asset.id)
      out.push(asset)
    }
  }
  return out
})
const stageAssets = computed(() => {
  if (viewMode.value === 'all') {
    if (visibleAssets.value.length) return visibleAssets.value
    return selectedAsset.value ? [selectedAsset.value] : []
  }
  return selectedAsset.value ? [selectedAsset.value] : []
})
const stagePrompt = computed(() => stageTask.value?.task.prompt ?? '')
const detailReferences = computed(() => stageTask.value?.references ?? [])
const stageTaskFailed = computed(() =>
  !!stageTask.value && ['failed', 'cancelled', 'unknown'].includes(stageTask.value.task.status),
)
const selectedIsLiked = computed(() => !!selectedAsset.value?.liked)

const mention = ref<ReturnType<typeof referenceMentionAt>>(null),
  mentionIndex = ref(0)
const mentionChoices = computed(() =>
  references.value
    .map((reference, index) => ({ ...reference, index, label: `@图片${index + 1}` }))
    .filter(
      (item) => !mention.value?.query || String(item.index + 1).startsWith(mention.value.query),
    ),
)

// ── Time / status helpers ───────────────────────────────────────────────────
function clockOf(iso: string) {
  const date = new Date(iso)
  return Number.isNaN(date.getTime())
    ? '--:--'
    : date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })
}
function dayOf(iso: string) {
  const date = new Date(iso)
  return Number.isNaN(date.getTime())
    ? '--.--'
    : `${String(date.getMonth() + 1).padStart(2, '0')}.${String(date.getDate()).padStart(2, '0')}`
}
function statusClass(value: string) {
  return value === 'succeeded' ? 'is-success' : ['failed', 'cancelled'].includes(value) ? 'is-danger' : 'is-accent'
}

// ── Workspace navigation ────────────────────────────────────────────────────
function closeSidebar() {
  sidebarOpen.value = false
}
function openSidebar() {
  sidebarOpen.value = true
}
function setSidebarTab(tab: 'workspace' | 'liked') {
  sidebarTab.value = tab
}
function selectTask(row: AITaskView) {
  activeTaskId.value = row.task.id
  selectedAsset.value = row.assets[0] ?? null
  viewMode.value = 'single'
  promptExpanded.value = false
  resultOpen.value = false
  awaitingTask.value = 0
  closeSidebar()
}
function selectAsset(asset: AIAsset) {
  selectedAsset.value = asset
  const row = tasks.value.find((item) => item.assets.some((own) => own.id === asset.id))
  activeTaskId.value = row?.task.id ?? 0
  resultOpen.value = false
}
/** Choosing a thumbnail from the all-results grid focuses it in the single view. */
function selectGridAsset(asset: AIAsset) {
  selectAsset(asset)
  viewMode.value = 'single'
}
function selectLiked(asset: AIAsset) {
  activeTaskId.value = 0
  selectedAsset.value = asset
  viewMode.value = 'single'
  promptExpanded.value = false
  closeSidebar()
}
function isActiveRow(row: AITaskView) {
  if (row.task.id === activeTaskId.value) return true
  return !!selectedAsset.value && row.assets.some((asset) => asset.id === selectedAsset.value!.id)
}
function chooseSession(id: number) {
  workspaceMenuOpen.value = false
  closeSidebar()
  void studio.run(() => studio.switchSession(id))
}
async function newWorkspace() {
  workspaceMenuOpen.value = false
  closeSidebar()
  await studio.run(() =>
    studio.createWorkspace(`新工作区 ${Math.max(1, sessions.value.length + 1)}`),
  )
}
function startRename() {
  renameDraft.value = name.value
  renaming.value = true
  workspaceMenuOpen.value = false
}
async function commitRename() {
  const next = renameDraft.value.trim()
  renaming.value = false
  if (!next || next === name.value) return
  await studio.run(() => studio.renameWorkspace(next))
}

// ── Draft editing ───────────────────────────────────────────────────────────
/**
 * Grow the prompt with its content (the reference dock behaves this way).
 * Height is written inline so the fixed dock's ResizeObserver can reserve the
 * new footprint before the phone keyboard or the last task is covered.
 */
function resizePrompt() {
  const el = promptArea.value
  if (!el) return
  const phone =
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(max-width: 820px)').matches
  const visibleHeight = window.visualViewport?.height ?? window.innerHeight
  const max = phone ? Math.min(168, Math.round(visibleHeight * 0.26)) : 120
  const min = phone ? 72 : 64
  el.style.height = 'auto'
  el.style.height = `${Math.max(min, Math.min(el.scrollHeight, max))}px`
  el.style.overflowY = el.scrollHeight > max ? 'auto' : 'hidden'
}
function onPromptInput() {
  resizePrompt()
  mention.value = references.value.length
    ? referenceMentionAt(prompt.value, promptArea.value?.selectionStart ?? prompt.value.length)
    : null
  mentionIndex.value = 0
}
async function insertReferenceMention(index: number) {
  if (composerLocked.value) return
  const cursor = promptArea.value?.selectionStart ?? prompt.value.length
  const range = mention.value ?? { start: cursor, end: cursor }
  const token = `@图片${index + 1} `
  const text = prompt.value.slice(0, range.start) + token + prompt.value.slice(range.end)
  if (text.length > 8000) return
  prompt.value = text
  mention.value = null
  await nextTick()
  promptArea.value?.focus()
  promptArea.value?.setSelectionRange(range.start + token.length, range.start + token.length)
}
function onPromptKeydown(event: KeyboardEvent) {
  if (event.isComposing) return
  // Enter writes a new line; Shift/Ctrl/⌘ + Enter submits, matching the dock hint.
  if (event.key === 'Enter' && (event.shiftKey || event.ctrlKey || event.metaKey)) {
    event.preventDefault()
    if (canGenerate.value) void generate()
    return
  }
  if (!mention.value || !mentionChoices.value.length) return
  if (event.key === 'Escape') {
    mention.value = null
    event.preventDefault()
  }
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    mentionIndex.value =
      (mentionIndex.value + (event.key === 'ArrowDown' ? 1 : -1) + mentionChoices.value.length) %
      mentionChoices.value.length
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    void insertReferenceMention(mentionChoices.value[mentionIndex.value]!.index)
  }
}
function moveReference(index: number, direction: number) {
  if (composerLocked.value) return
  const before = references.value.map((reference) => reference.asset_id)
  studio.moveReference(index, direction)
  prompt.value = remapReferenceMentions(
    prompt.value,
    before,
    references.value.map((reference) => reference.asset_id),
  )
  mention.value = null
}
async function removeReference(index: number) {
  if (composerLocked.value) return
  const before = references.value.map((reference) => reference.asset_id)
  if (
    prompt.value.includes(`@图片${index + 1}`) &&
    !(await confirm(`移除 @图片${index + 1}？提示词中的对应引用也会移除。`))
  )
    return
  studio.removeReference(index)
  prompt.value = remapReferenceMentions(
    prompt.value,
    before,
    references.value.map((reference) => reference.asset_id),
  )
  mention.value = null
}
async function focusPrompt() {
  await nextTick()
  promptArea.value?.focus()
}
async function generate() {
  if (!canGenerate.value) return
  // One confirmation owns the submission; repeated keyboard/button input cannot
  // replace the pending confirmation or enqueue a second task.
  confirming.value = true
  try {
    await studio.run(async () => {
      if (
        !pending.value &&
        !(await confirm(
          `本次生成 ${count.value} 张图片，预留 ${cost.value} 额度。确认开始？`,
        ))
      )
        return
      const task = await studio.generate()
      if (task) {
        awaitingTask.value = task.id
        settingsOpen.value = false
      }
    })
  } finally {
    confirming.value = false
  }
}
// All entry points share one upload gate; drops/pastes cannot mutate a pending draft.
async function uploadFiles(files: FileList | File[] | null) {
  if (!files?.length || composerLocked.value) return
  uploading.value = true
  try {
    await studio.run(async () => {
      const batch = Array.from(files)
      if (references.value.length + batch.length > (provider.value?.max_references ?? 0))
        throw new Error('参考图数量超出当前模型上限，请减少图片后重试')
      if (
        batch.some(
          (file) =>
            file.size > 20 * 1024 * 1024 ||
            (file.type && !['image/png', 'image/jpeg', 'image/webp'].includes(file.type)),
        )
      )
        throw new Error('请使用不超过 20MB 的 PNG、JPEG 或 WebP 图片')
      for (const file of batch) await studio.upload(file)
    })
  } finally {
    uploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}
function pasteImages(event: ClipboardEvent) {
  const files = Array.from(event.clipboardData?.files ?? []).filter((file) =>
    file.type.startsWith('image/'),
  )
  if (!files.length) return // Plain text retains the native textarea paste behavior.
  event.preventDefault()
  void uploadFiles(files)
}
function dropImages(event: DragEvent) {
  dragging.value = false
  void uploadFiles(event.dataTransfer?.files ?? null)
}

// ── Result actions ──────────────────────────────────────────────────────────
async function likeSelected() {
  if (!selectedAsset.value) return
  await studio.run(() => studio.toggleLike(selectedAsset.value!))
}
async function useResult(asParent: boolean) {
  if (!selectedAsset.value || composerLocked.value) return
  await studio.run(async () => {
    if (asParent) studio.continueFrom(selectedAsset.value!)
    else studio.addReference(selectedAsset.value!.id)
    resultOpen.value = false
    await focusPrompt()
  })
}
async function continueFromTask(row: AITaskView) {
  const asset = row.assets[0]
  if (!asset || composerLocked.value) return
  await studio.run(async () => {
    selectedAsset.value = asset
    studio.continueFrom(asset)
    await focusPrompt()
  })
}
async function cancelQueued(row: AITaskView) {
  await studio.run(async () => {
    await aiApi.cancel(row.task.id)
    await studio.refresh()
  })
}
async function reusePrompt(row: AITaskView) {
  if (composerLocked.value) return
  if (
    prompt.value &&
    prompt.value !== row.task.prompt &&
    !(await confirm('用此任务的提示词替换当前草稿？参考图与参数保持当前设置。'))
  )
    return
  prompt.value = row.task.prompt
  enhancePrompt.value = false
  await focusPrompt()
}
async function removeLibraryAsset(asset: AIAsset) {
  if (!(await confirm(`移除图片“${asset.title || `图片 ${asset.id}`}”？`))) return
  await studio.run(() => archiveAsset(asset.id))
}
async function archiveTask(row: AITaskView) {
  if (!['failed', 'cancelled'].includes(row.task.status)) return
  if (!(await confirm(`删除任务 #${row.task.id}？失败记录会从工作台移除，已生成的素材仍保留。`)))
    return
  await studio.run(async () => {
    await aiApi.archiveTask(row.task.id)
    if (activeTaskId.value === row.task.id) activeTaskId.value = 0
    await studio.refresh()
  })
}
async function archiveFailedTasks() {  const rows = failedTaskRows.value
  if (!rows.length) return
  if (
    !(await confirm(
      `删除当前列表中的 ${rows.length} 条失败记录？已生成的素材仍会保留。`,
    ))
  )
    return
  await studio.run(async () => {
    // The API deliberately accepts only failed/cancelled tasks. Keep requests
    // serial so a single conflict leaves the remaining records diagnosable.
    for (const row of rows) await aiApi.archiveTask(row.task.id)
    activeTaskId.value = 0
    await studio.refresh()
  })
}
async function download() {
  if (!selectedAsset.value) return
  await studio.run(async () => {
    const blob = await aiApi.blob(selectedAsset.value!.id)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `creative-${selectedAsset.value!.id}.${selectedAsset.value!.mime.split('/')[1]}`
    anchor.click()
    URL.revokeObjectURL(url)
  })
}
async function removeResult() {
  if (!selectedAsset.value || !(await confirm('从私人历史移除此结果？已有任务引用将继续保留。')))
    return
  await studio.run(async () => {
    const id = selectedAsset.value!.id
    await archiveAsset(id)
    selectedAsset.value = null
    resultOpen.value = false
    await studio.refresh()
  })
}
function onMoreAction(value: string | number | Record<string, unknown> | undefined) {
  const action = String(value ?? '')
  if (action === 'publish') void openPublish()
  else if (action === 'continue' && stageTask.value) void continueFromTask(stageTask.value)
  else if (action === 'reference') void useResult(false)
  else if (action === 'reuse' && stageTask.value) void reusePrompt(stageTask.value)
  else if (action === 'idea') {
    resourceTab.value = 'idea'
    resourceOpen.value = true
  } else if (action === 'save') void studio.run(studio.save)
  else if (action === 'archive' && stageTask.value) void archiveTask(stageTask.value)
  else if (action === 'remove') void removeResult()
}

// ── Publishing / ideas / resources ──────────────────────────────────────────
const proposal = ref({ title: '', description: '', public_consent: false, reference_rights: false })
const publishForm = ref({
  title: '',
  description: '',
  tags: [] as string[],
  auto_metadata: true,
  public_consent: false,
})
const publishMetadata = ref({
  available: false,
  ready: false,
  placeholder_title: '待命名作品',
  placeholder_tag: '待生成标签',
})
async function submitIdea() {
  await studio.run(async () => {
    if (!proposal.value.public_consent || !proposal.value.reference_rights)
      throw new Error('请确认公开范围和参考图授权')
    const row = await aiApi.postIdea({
      ...proposal.value,
      input: studio.input(),
      character_ids: studio.characterIds.value,
    })
    resourceOpen.value = false
    await router.push(`/ideas/${row.id}`)
  })
}
async function publish() {
  if (!selectedAsset.value) return
  await studio.run(async () => {
    if (!publishForm.value.public_consent) throw new Error('请确认公开发布')
    const row = await aiApi.publish(
      selectedAsset.value!.id,
      publishForm.value.title,
      publishForm.value.description,
      publishForm.value.tags,
      publishForm.value.auto_metadata,
    )
    publishOpen.value = false
    resultOpen.value = false
    await router.push(`/content/${row.content_id}`)
  })
}
async function openPublish() {
  publishForm.value = {
    title: '',
    description: '',
    tags: [],
    auto_metadata: true,
    public_consent: false,
  }
  try {
    const response = await contentApi.metadataOptions()
    if (response.code === 200) {
      publishMetadata.value = response.data
      publishForm.value.auto_metadata = response.data.available
    }
  } catch {
    publishForm.value.auto_metadata = false
  }
  publishOpen.value = true
}
function openResource(tab: typeof resourceTab.value) {
  resourceTab.value = tab
  resourceOpen.value = true
}
async function applyResource(value: {
  text: string
  assets: number[]
  characters: number[]
  parameters: Record<string, unknown>
  action: 'append' | 'replace' | 'insert'
}) {
  await studio.run(async () => {
    if (pending.value) throw new Error('请先确认上次提交的结果')
    if (
      value.action === 'replace' &&
      prompt.value &&
      !(await confirm('替换当前提示词？当前草稿会被覆盖。'))
    )
      return
    const start = promptArea.value?.selectionStart ?? prompt.value.length,
      end = promptArea.value?.selectionEnd ?? start
    if (
      new Set([...references.value.map((reference) => reference.asset_id), ...value.assets]).size >
      (provider.value?.max_references ?? 0)
    )
      throw new Error('参考图数量超出供应商上限，请先调整参考图或关闭推荐参考图')
    studio.characterIds.value = [...new Set([...studio.characterIds.value, ...value.characters])]
    if (value.text)
      prompt.value =
        value.action === 'replace'
          ? value.text
          : value.action === 'insert'
            ? prompt.value.slice(0, start) + value.text + prompt.value.slice(end)
            : [prompt.value, value.text].filter(Boolean).join('\n')
    for (const id of value.assets) studio.addReference(id)
    if (typeof value.parameters.size === 'string') {
      const rawSize = value.parameters.size.trim()
      if (rawSize.toLowerCase() === AUTO_IMAGE_SIZE) size.value = AUTO_IMAGE_SIZE
      else if (parseImageSize(rawSize)) size.value = normalizeImageSize(rawSize)
    }
    if (
      typeof value.parameters.quality === 'string' &&
      provider.value?.qualities.includes(value.parameters.quality)
    )
      quality.value = value.parameters.quality
    resourceOpen.value = false
  })
}

// ── Reactive bridges ────────────────────────────────────────────────────────
watch(sessionId, () => {
  mention.value = null
  awaitingTask.value = 0
  activeTaskId.value = 0
  resultOpen.value = false
  settingsOpen.value = false
  viewMode.value = 'single'
  // Filters belong to the workspace that was open when they were typed; a
  // switch must not leave an active query silently hiding the new history.
  search.value = ''
  status.value = ''
})
watch([tasks, awaitingTask], () => {
  const row = tasks.value.find((item) => item.task.id === awaitingTask.value)
  if (row?.assets[0]) {
    selectedAsset.value = row.assets[0]
    activeTaskId.value = row.task.id
    awaitingTask.value = 0
  } else if (row && ['failed', 'cancelled', 'unknown'].includes(row.task.status)) {
    activeTaskId.value = row.task.id
    awaitingTask.value = 0
  }
})
watch(selectedAsset, (asset) => {
  if (!asset) return
  const row = tasks.value.find((item) => item.assets.some((own) => own.id === asset.id))
  if (row) activeTaskId.value = row.task.id
})
watch(prompt, () => {
  void nextTick(resizePrompt)
})
onMounted(() => {
  void nextTick(resizePrompt)
  window.addEventListener('resize', resizePrompt)
  window.visualViewport?.addEventListener('resize', resizePrompt)
})
let historyTimer: ReturnType<typeof setTimeout> | undefined
watch([ready, search, status], () => {
  if (!ready.value) return
  if (historyTimer) clearTimeout(historyTimer)
  historyTimer = setTimeout(
    () =>
      void studio.run(() =>
        studio.setHistoryQuery({
          search: search.value,
          status: status.value,
        }),
      ),
    250,
  )
})
watch(
  () => route.fullPath,
  () => closeSidebar(),
)
onBeforeUnmount(() => {
  if (historyTimer) clearTimeout(historyTimer)
  window.removeEventListener('resize', resizePrompt)
  window.visualViewport?.removeEventListener('resize', resizePrompt)
})
onBeforeRouteLeave(async () => {
  try {
    await studio.save()
    return true
  } catch {
    return false
  }
})
watch(ready, (value) => {
  if (!value) return
  const ids = String(route.query.assets ?? '')
    .split(',')
    .map(Number)
    .filter((n) => n > 0)
  const characters = String(route.query.characters ?? '')
    .split(',')
    .map(Number)
    .filter((n) => n > 0)
  if (ids.length || characters.length)
    void applyResource({
      text: String(route.query.prompt ?? ''),
      assets: ids,
      characters,
      parameters: {},
      action: 'append',
    })
})
</script>

<template>
  <div
    class="creative-page creative-theme ai-studio ws-shell"
    :style="dockStyle"
    :class="{ 'is-switching': switching, 'is-sidebar-open': sidebarOpen }"
  >
    <button
      v-if="sidebarOpen"
      type="button"
      class="ws-scrim"
      aria-label="关闭工作区侧栏"
      @click="closeSidebar"
    ></button>

    <aside class="ws-sidebar" :class="{ 'is-open': sidebarOpen }" aria-label="工作区导航">
      <header class="ws-sidebar-head">
        <div class="ws-user">
          <UserAvatar
            :size="34"
            :email="userStore.user?.email"
            :username="userStore.user?.username"
          />
          <span class="ws-user-copy">
            <strong>{{ username }}</strong>
            <small>{{ studio.saveState.value || '创作工作台' }}</small>
          </span>
        </div>
        <button class="ws-icon-btn ws-sidebar-close" type="button" aria-label="关闭工作区侧栏" @click="closeSidebar">
          <IconClose />
        </button>
      </header>

      <nav class="ws-tabs" aria-label="工作区栏目">
        <button
          type="button"
          :class="{ 'is-active': sidebarTab === 'workspace' }"
          :aria-pressed="sidebarTab === 'workspace'"
          @click="setSidebarTab('workspace')"
        >
          <IconHistory />
          <span>工作区</span>
          <b>{{ tasks.length }}</b>
        </button>
        <button
          type="button"
          :class="{ 'is-active': sidebarTab === 'liked' }"
          :aria-pressed="sidebarTab === 'liked'"
          @click="setSidebarTab('liked')"
        >
          <IconHeart />
          <span>喜欢</span>
          <b>{{ likedAssets.length }}</b>
        </button>
      </nav>

      <div v-if="sidebarTab === 'workspace'" class="ws-sidebar-body">
        <div class="ws-workspace-picker">
          <button
            type="button"
            class="ws-workspace-current"
            :aria-expanded="workspaceMenuOpen"
            aria-label="切换工作区"
            @click="workspaceMenuOpen = !workspaceMenuOpen"
          >
            <span>{{ name }}</span>
            <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M2.5 4.5 6 8l3.5-3.5" /></svg>
          </button>
          <button class="ws-icon-btn" type="button" aria-label="重命名工作区" @click="startRename">
            <IconEdit />
          </button>
          <button class="ws-icon-btn" type="button" aria-label="新建工作区" @click="newWorkspace">
            <IconPlus />
          </button>
          <form v-if="renaming" class="ws-rename" @submit.prevent="commitRename">
            <input v-model="renameDraft" maxlength="100" aria-label="工作区名称" />
            <button type="submit" aria-label="保存工作区名称"><IconCheck /></button>
            <button type="button" aria-label="取消重命名" @click="renaming = false"><IconClose /></button>
          </form>
          <div v-if="workspaceMenuOpen" class="ws-workspace-menu" role="menu" aria-label="工作区列表">
            <button
              v-for="item in sessions"
              :key="item.id"
              type="button"
              role="menuitemradio"
              :aria-checked="item.id === sessionId"
              :class="{ 'is-active': item.id === sessionId }"
              @click="chooseSession(item.id)"
            >
              <span>{{ item.name }}</span>
              <IconCheck v-if="item.id === sessionId" />
            </button>
            <button type="button" class="ws-workspace-new" @click="newWorkspace">
              <IconPlus /><span>新建工作区</span>
            </button>
          </div>
        </div>

        <div class="ws-filters">
          <input
            v-model="search"
            type="search"
            placeholder="搜索提示词"
            aria-label="搜索历史提示词"
          />
          <select v-model="status" aria-label="任务状态">
            <option value="">全部</option>
            <option v-for="(label, key) in labels" :key="key" :value="key">{{ label }}</option>
          </select>
        </div>

        <div class="ws-history" role="list" aria-label="任务历史">
          <div v-if="!ready && !error" class="ws-sidebar-note" role="status">
            <IconLoading />正在恢复创作…
          </div>
          <div v-else-if="!shownTasks.length" class="ws-sidebar-note">
            {{ search || status ? '没有匹配的任务' : '这个工作区还没有任务' }}
          </div>
          <button
            v-for="row in shownTasks"
            :key="row.task.id"
            type="button"
            role="listitem"
            class="ws-history-item"
            :class="{ 'is-active': isActiveRow(row) }"
            @click="selectTask(row)"
          >
            <span class="ws-history-thumb">
              <CreativeImage v-if="row.assets[0]" :id="row.assets[0].id" />
              <IconImage v-else />
            </span>
            <span class="ws-history-copy">
              <span class="ws-history-prompt">{{ row.task.prompt || '（无提示词）' }}</span>
              <span class="ws-history-meta">
                <span :class="statusClass(row.task.status)">{{ labels[row.task.status] }}</span>
                <span>· {{ row.assets.length }} 张</span>
                <time>{{ dayOf(row.task.created_at) }}</time>
              </span>
            </span>
          </button>
        </div>
        <button
          v-if="moreTasks"
          type="button"
          class="ws-load-more"
          @click="studio.run(studio.loadMoreTasks)"
        >
          加载更早的任务
        </button>
      </div>

      <div v-else class="ws-sidebar-body">
        <div v-if="likedAssets.length" class="ws-liked-grid">
          <div v-for="asset in likedAssets" :key="asset.id" class="ws-liked-item">
            <button type="button" class="ws-liked-open" :aria-label="`查看收藏图片 ${asset.id}`" @click="selectLiked(asset)">
              <CreativeImage :id="asset.id" />
            </button>
            <button
              type="button"
              class="ws-liked-heart"
              aria-label="取消喜欢"
              @click="studio.run(() => studio.toggleLike(asset))"
            >
              <IconHeartFill />
            </button>
          </div>
        </div>
        <p v-else class="ws-sidebar-note">
          还没有喜欢的作品。在生成结果下方点击心形，即可收纳到这里。
        </p>
        <button
          v-if="moreLiked"
          type="button"
          class="ws-load-more"
          @click="studio.run(studio.loadMoreLikedAssets)"
        >
          加载更早的喜欢
        </button>
      </div>

      <footer class="ws-sidebar-foot">
        <span class="ws-credit-ring" :title="quotaVisible ? quotaNote : '管理员生成不消耗额度'">
          {{ quotaVisible ? account.available : '∞' }}
        </span>
        <span class="ws-credit-copy">
          <strong>{{ quotaVisible ? '可用额度' : '管理员免额度' }}</strong>
          <small v-if="quotaVisible">每日 {{ dailyCreditTotal }}/5 · 其他 {{ otherCreditTotal }}</small>
          <small v-else>生成不扣除站点额度</small>
        </span>
        <button
          v-if="failedTaskRows.length"
          type="button"
          class="ws-clear-failed"
          aria-label="清除失败任务"
          @click="archiveFailedTasks"
        >
          清理失败 {{ failedTaskRows.length }}
        </button>
      </footer>
    </aside>

    <section class="ws-main" aria-label="创作舞台">
      <header class="ws-stage-head">
        <button class="ws-icon-btn ws-menu-btn" type="button" aria-label="打开工作区" @click="openSidebar">
          <IconMenu />
        </button>
        <div v-if="stageTask" class="ws-stage-meta">
          <strong>{{ clockOf(stageTask.task.created_at) }}</strong>
          <span class="ws-stage-sep" aria-hidden="true">·</span>
          <span class="ws-stage-date">{{ dayOf(stageTask.task.created_at) }}</span>
          <span class="ws-stage-sep" aria-hidden="true">·</span>
          <span v-if="viewMode === 'all'" class="ws-stage-all-count"
            >全部 {{ visibleAssets.length }} 张</span
          >
          <template v-else>
            <span class="ws-stage-status" :class="statusClass(stageTask.task.status)">{{
              labels[stageTask.task.status]
            }}</span>
            <span class="ws-stage-sep" aria-hidden="true">·</span>
            <span class="ws-stage-count">{{ stageTask.assets.length }} 张</span>
          </template>
        </div>
        <div v-else class="ws-stage-meta">
          <strong>{{ name }}</strong>
          <span>把画面写下来</span>
        </div>
        <div class="ws-view-toggle" role="radiogroup" aria-label="查看方式">
          <button
            type="button"
            role="radio"
            :aria-checked="viewMode === 'single'"
            :class="{ 'is-active': viewMode === 'single' }"
            @click="viewMode = 'single'"
          >
            单张
          </button>
          <button
            type="button"
            role="radio"
            :aria-checked="viewMode === 'all'"
            :class="{ 'is-active': viewMode === 'all' }"
            @click="viewMode = 'all'"
          >
            全部
          </button>
        </div>
      </header>

      <div v-if="stagePrompt" class="ws-prompt-line">
        <p :class="{ 'is-expanded': promptExpanded }">{{ stagePrompt }}</p>
        <button type="button" @click="promptExpanded = !promptExpanded">
          {{ promptExpanded ? '收起' : '展开' }}
        </button>
      </div>

      <div v-if="error" class="ws-error" role="alert">{{ error }}</div>
      <div v-if="activeTask" class="ws-task-note" role="status">
        <IconLoading v-if="activeTask.task.status !== 'unknown'" class="ws-spin" />
        任务 #{{ activeTask.task.id }} · {{ labels[activeTask.task.status] }}
        {{ activeTask.task.status === 'unknown' ? '，预留额度待管理员核查' : '，离开页面仍会继续' }}
      </div>

      <div
        class="ws-canvas"
        :class="{ 'is-multi': viewMode === 'all' && stageAssets.length > 1, 'is-dragging': dragging }"
      >
        <template v-if="stageAssets.length">
          <div
            v-if="viewMode === 'all'"
            class="ws-canvas-grid"
            :class="{ 'is-single': stageAssets.length === 1 }"
          >
            <button
              v-for="(asset, index) in stageAssets"
              :key="asset.id"
              type="button"
              :aria-label="`查看结果 ${index + 1}`"
              :aria-pressed="selectedAsset?.id === asset.id"
              @click="selectGridAsset(asset)"
            >
              <CreativeImage :id="asset.id" />
            </button>
          </div>
          <button
            v-else
            type="button"
            class="ws-canvas-image"
            :aria-label="`放大查看图片 ${selectedAsset?.id ?? ''}`"
            @click="resultOpen = true"
          >
            <CreativeImage :id="stageAssets[0]?.id ?? 0" full />
          </button>
        </template>

        <div v-else-if="stageTaskFailed && stageTask" class="ws-canvas-state is-error">
          <IconImage />
          <strong>{{ labels[stageTask.task.status] }}</strong>
          <p v-if="stageTask.task.error">{{ stageTask.task.error }}</p>
          <p v-else-if="stageTask.task.status === 'unknown'">结果待核查，预留额度待管理员确认。</p>
          <button type="button" class="ws-text-btn" @click="reusePrompt(stageTask)">复用提示词重试</button>
        </div>
        <div v-else-if="!ready" class="ws-canvas-state">
          <IconLoading class="ws-spin" />
          <strong>正在恢复你的创作…</strong>
        </div>
        <div v-else-if="activeTask" class="ws-canvas-state">
          <IconLoading class="ws-spin" />
          <strong>{{ labels[activeTask.task.status] }}…</strong>
          <p>任务已提交，离开页面也会继续。</p>
        </div>
        <div v-else class="ws-canvas-state">
          <IconImage />
          <strong>你的下一幅作品，从这里开始</strong>
          <p>写下画面，或添加一张参考图。每次尝试都会留在这里。</p>
          <button type="button" class="ws-text-btn" @click="focusPrompt">开始创作</button>
        </div>
      </div>

      <div v-if="selectedAsset" class="ws-actions" aria-label="结果操作">
        <button type="button" :aria-pressed="selectedIsLiked" @click="likeSelected">
          <IconHeartFill v-if="selectedIsLiked" class="is-liked" />
          <IconHeart v-else />
          <span>喜欢</span>
        </button>
        <button type="button" @click="download">
          <IconDownload />
          <span>下载</span>
        </button>
        <button type="button" @click="openPublish">
          <IconShareExternal />
          <span>分享</span>
        </button>
        <a-dropdown trigger="click" position="br" @select="onMoreAction">
          <button type="button" class="ws-more-trigger" aria-label="更多操作">
            <IconMore />
            <span>更多</span>
          </button>
          <template #content>
            <a-doption value="continue">继续编辑</a-doption>
            <a-doption value="reference">加入参考图</a-doption>
            <a-doption value="reuse">复用提示词</a-doption>
            <a-doption value="publish">发布到站内</a-doption>
            <a-doption value="idea">投稿创意</a-doption>
            <a-doption value="save">保存草稿</a-doption>
            <a-doption v-if="stageTask && ['failed', 'cancelled'].includes(stageTask.task.status)" value="archive">
              删除任务记录
            </a-doption>
            <a-doption value="remove">移除结果</a-doption>
          </template>
        </a-dropdown>
      </div>

      <section
        ref="composerElement"
        class="ws-composer"
        :class="{ 'is-dragging': dragging }"
        aria-label="创作输入区"
        @dragover.prevent="dragging = !composerLocked && !!provider?.max_references"
        @dragleave.self="dragging = false"
        @drop.prevent="dropImages"
      >
        <StudioReferences
          :references="references"
          :disabled="composerLocked"
          @preview="(id) => (previewReference = id)"
          @mention="insertReferenceMention"
          @remove="removeReference"
          @reorder="(from, to) => moveReference(from, to - from)"
        />
        <div v-if="studio.parentAssetId.value" class="ws-composer-context">
          继续编辑 @图片{{
            references.findIndex((reference) => reference.asset_id === studio.parentAssetId.value) + 1
          }}
        </div>
        <div class="ws-composer-box">
          <div class="ws-composer-prompt">
            <label class="sr-only" for="studio-prompt">创作提示词</label>
            <textarea
              id="studio-prompt"
              ref="promptArea"
              v-model="prompt"
              rows="3"
              maxlength="8000"
              :disabled="composerLocked"
              placeholder="描述你想生成的图片，或附加图片进行编辑……"
              aria-label="创作提示词"
              @input="onPromptInput"
              @click="onPromptInput"
              @keydown="onPromptKeydown"
              @paste="pasteImages"
            />
            <div
              v-if="mention && mentionChoices.length"
              class="reference-mentions"
              role="group"
              aria-label="选择参考图片"
            >
              <button
                v-for="(item, index) in mentionChoices"
                :key="item.asset_id"
                type="button"
                :class="{ active: index === mentionIndex }"
                @mousedown.prevent
                @click="insertReferenceMention(item.index)"
              >
                <span>{{ item.label }}</span><span class="creative-note">插入提示词</span>
              </button>
            </div>
          </div>
          <div class="ws-composer-bar">
            <div class="ws-composer-tools">
              <button
                type="button"
                class="ws-tool-btn"
                aria-label="生成设置"
                :aria-expanded="settingsOpen"
                @click="settingsOpen = !settingsOpen"
              >
                <IconSettings />
              </button>
              <button
                type="button"
                class="ws-tool-btn"
                aria-label="上传参考图"
                :disabled="!provider?.max_references || references.length >= (provider?.max_references ?? 0)"
                @click="fileInput?.click()"
              >
                <IconPlus />
              </button>
              <button type="button" class="ws-tool-btn" aria-label="图片库" @click="openResource('history')">
                <IconImage />
              </button>
              <button type="button" class="ws-tool-btn" aria-label="角色与模板" @click="openResource('library')">
                <IconFile />
              </button>
              <button type="button" class="ws-model-chip" @click="settingsOpen = !settingsOpen">
                {{ provider?.name || '暂无模型' }} <span>×{{ count }}</span>
              </button>
            </div>
            <div class="ws-composer-send">
              <span v-if="quotaVisible" class="ws-cost-note">{{ quotaNote }}</span>
              <button
                type="button"
                class="ws-send-btn"
                :disabled="!canGenerate"
                :aria-label="pending ? '确认上次提交' : '开始生成'"
                @click="generate"
              >
                <IconLoading v-if="busy || uploading" class="ws-spin" />
                <IconSend v-else />
              </button>
            </div>
          </div>
          <div v-if="uploading" class="ws-composer-hint" role="status">正在上传参考图…</div>
          <div v-else-if="pending" class="ws-composer-hint" role="status">
            上次提交结果尚未确认，草稿已锁定。请用原请求继续确认。
          </div>
          <div v-else class="ws-composer-hint">
            Enter 换行 · Shift + Enter 生成
            <span v-if="provider?.max_references">· 可拖入或粘贴 PNG / JPEG / WebP</span>
          </div>
        </div>
        <input
          ref="fileInput"
          type="file"
          accept="image/png,image/jpeg,image/webp"
          multiple
          hidden
          @change="(event) => uploadFiles((event.target as HTMLInputElement).files)"
        />

        <div v-if="settingsOpen" class="ws-settings" role="dialog" aria-label="生成设置">
          <header class="ws-settings-head">
            <strong>生成设置</strong>
            <button type="button" aria-label="关闭生成设置" @click="settingsOpen = false"><IconClose /></button>
          </header>
          <div class="ws-settings-grid">
            <label class="ws-field">
              <span>模型</span>
              <select v-model="providerId" aria-label="供应商与模型">
                <option v-if="!availableProviders.length" :value="0">暂无可用模型</option>
                <option v-for="item in availableProviders" :key="item.id" :value="item.id">
                  {{ item.name }} · {{ item.model }}
                </option>
              </select>
            </label>
            <div class="ws-field">
              <span>尺寸</span>
              <button type="button" class="ws-size-trigger" aria-label="画面尺寸" :disabled="composerLocked" @click="sizePickerOpen = true">
                {{ formatImageSize(size) }}
              </button>
            </div>
            <label v-if="provider?.qualities.length" class="ws-field">
              <span>画质</span>
              <select v-model="quality" aria-label="画质">
                <option value="">默认</option>
                <option v-for="item in provider.qualities" :key="item">{{ item }}</option>
              </select>
            </label>
            <label class="ws-field">
              <span>数量</span>
              <select v-model.number="count" aria-label="输出数量">
                <option v-for="n in provider?.max_count ?? 1" :key="n" :value="n">{{ n }} 张</option>
              </select>
            </label>
            <label class="ws-field ws-field-toggle">
              <span>提示词增强</span>
              <span class="ws-toggle" :title="promptEnhancement.prompt || '暂无增强提示词'">
                <input
                  v-model="enhancePrompt"
                  type="checkbox"
                  :disabled="!promptEnhancement.enabled || !promptEnhancement.prompt"
                />
                <strong>{{ enhancePrompt && promptEnhancement.enabled ? '已开启' : '未开启' }}</strong>
              </span>
            </label>
          </div>
          <div class="ws-settings-actions">
            <button type="button" :disabled="composerLocked" @click="studio.run(studio.save)">保存草稿</button>
            <button type="button" :disabled="composerLocked || !prompt.trim()" @click="openResource('idea')">
              投稿创意
            </button>
            <span v-if="!provider" class="ws-settings-note">管理员尚未启用生图供应商</span>
            <span v-else-if="!provider.has_key" class="ws-settings-note">此模型尚未配置站点密钥</span>
          </div>
        </div>
      </section>
    </section>

    <aside class="ws-detail" aria-label="任务详情">
      <header class="ws-detail-head">
        <span class="ws-kicker">DETAIL</span>
        <h2>任务详情</h2>
      </header>
      <template v-if="stageTask">
        <dl class="ws-detail-meta">
          <div><dt>状态</dt><dd :class="statusClass(stageTask.task.status)">{{ labels[stageTask.task.status] }}</dd></div>
          <div><dt>时间</dt><dd>{{ clockOf(stageTask.task.created_at) }} · {{ dayOf(stageTask.task.created_at) }}</dd></div>
          <div><dt>模型</dt><dd>{{ stageTask.task.model }}</dd></div>
          <div><dt>规格</dt><dd>{{ formatImageSize(stageTask.task.size) }} · {{ stageTask.task.count }} 张</dd></div>
        </dl>
        <div class="ws-detail-block">
          <span class="ws-kicker">PROMPT</span>
          <p class="ws-detail-prompt">{{ stageTask.task.prompt }}</p>
        </div>
        <div v-if="detailReferences.length" class="ws-detail-block">
          <span class="ws-kicker">REFERENCES</span>
          <div class="ws-detail-refs">
            <button
              v-for="(asset, index) in detailReferences"
              :key="asset.id"
              type="button"
              :aria-label="`预览参考图 ${index + 1}`"
              @click="previewReference = asset.id"
            >
              <CreativeImage :id="asset.id" />
            </button>
          </div>
        </div>
        <p v-if="stageTask.task.error" class="ws-detail-error">{{ stageTask.task.error }}</p>
        <div class="ws-detail-actions">
          <button
            v-if="stageTask.assets.length"
            type="button"
            class="is-primary"
            :disabled="!provider?.max_references || composerLocked"
            @click="continueFromTask(stageTask)"
          >
            继续编辑
          </button>
          <button type="button" :disabled="composerLocked" @click="reusePrompt(stageTask)">复用提示词</button>
          <button
            v-if="stageTask.task.status === 'queued'"
            type="button"
            @click="cancelQueued(stageTask)"
          >
            取消排队
          </button>
          <button
            v-if="['failed', 'cancelled'].includes(stageTask.task.status)"
            type="button"
            @click="archiveTask(stageTask)"
          >
            删除记录
          </button>
          <RouterLink v-if="stageTask.task.idea_id" :to="`/ideas/${stageTask.task.idea_id}`">赞助创意</RouterLink>
        </div>
      </template>
      <template v-else-if="selectedAsset">
        <dl class="ws-detail-meta">
          <div><dt>类型</dt><dd>{{ selectedIsLiked ? '已喜欢' : '私人结果' }}</dd></div>
          <div><dt>尺寸</dt><dd>{{ selectedAsset.width }} × {{ selectedAsset.height }}</dd></div>
          <div><dt>时间</dt><dd>{{ clockOf(selectedAsset.created_at) }} · {{ dayOf(selectedAsset.created_at) }}</dd></div>
        </dl>
        <div class="ws-detail-block">
          <span class="ws-kicker">ASSET</span>
          <p class="ws-detail-prompt">{{ selectedAsset.title || `图片 ${selectedAsset.id}` }}</p>
        </div>
        <div class="ws-detail-actions">
          <button type="button" class="is-primary" @click="openPublish">发布到站内</button>
          <button type="button" @click="download">下载原图</button>
        </div>
      </template>
      <div v-else class="ws-detail-empty">
        <IconImage />
        <p>选择一个任务，查看提示词、参考图与生成参数。</p>
      </div>
    </aside>
  </div>

  <StudioSizePicker v-model:visible="sizePickerOpen" :model-value="size" @select="size = $event" />

  <a-drawer
    v-model:visible="resourceOpen"
    class="creative-theme creative-dialog ai-dialog studio-side-drawer"
    :width="440"
    :footer="false"
    placement="right"
    :title="resourceTitle"
  >
    <div class="side-drawer-tabs" role="tablist" aria-label="资源抽屉切换">
      <button
        v-for="tab in resourceTabs"
        :key="tab.key"
        type="button"
        role="tab"
        :aria-selected="resourceTab === tab.key"
        :class="['side-drawer-tab', { 'is-active': resourceTab === tab.key }]"
        @click="resourceTab = tab.key"
      >
        <component :is="tab.icon" aria-hidden="true" />
        <span>{{ tab.label }}</span>
      </button>
    </div>
    <div class="side-drawer-body">
      <CreativeLibrary v-if="resourceOpen && resourceTab === 'library'" @apply="applyResource" />
      <div v-else-if="resourceTab === 'history'" class="creative-grid ws-library-grid">
        <div v-for="asset in assets" :key="asset.id" class="library-asset">
          <button
            type="button"
            class="library-asset-select"
            @click="studio.run(async () => { studio.addReference(asset.id); resourceOpen = false })"
          >
            <CreativeImage :id="asset.id" /><span>{{ asset.title || `图片 ${asset.id}` }}</span>
          </button>
          <button type="button" class="library-asset-delete" :disabled="composerLocked" @click="removeLibraryAsset(asset)">
            删除
          </button>
        </div>
        <a-button v-if="studio.moreAssets.value" @click="studio.run(studio.loadMoreAssets)">加载更早的图片</a-button>
        <p v-if="!assets.length" class="creative-note">上传的参考图与生成结果会保存在这里。</p>
      </div>
      <div v-else-if="resourceTab === 'idea'" class="side-drawer-idea">
        <p class="creative-note">
          投稿将公开标题、描述、完整提示词、选中的参考图、模型与生成参数。赞助成功的结果会公开展示。投稿本身不消耗额度。
        </p>
        <label class="creative-field">标题<a-input v-model="proposal.title" :max-length="150" /></label>
        <label class="creative-field">描述<a-textarea v-model="proposal.description" /></label>
        <a-checkbox v-model="proposal.public_consent">我同意上述公开范围和成果展示规则</a-checkbox>
        <a-checkbox v-model="proposal.reference_rights">我确认选中参考图有公开展示授权</a-checkbox>
        <p class="creative-note">赞助额度不代表取得作品著作权。</p>
        <a-button
          type="primary"
          :disabled="!proposal.public_consent || !proposal.reference_rights"
          @click="submitIdea"
        >公开投稿</a-button>
      </div>
    </div>
  </a-drawer>

  <a-modal
    :visible="!!previewReference"
    class="creative-theme creative-dialog ai-dialog"
    title="参考图预览"
    :footer="false"
    @cancel="previewReference = 0"
    ><div class="reference-large"><CreativeImage v-if="previewReference" :id="previewReference" full /></div
  ></a-modal>

  <a-modal
    v-model:visible="resultOpen"
    class="creative-theme creative-dialog ai-dialog studio-result-dialog"
    title="生成结果"
    :width="1040"
    :footer="false"
  >
    <template v-if="selectedAsset">
      <div class="result-preview-meta">
        <span>{{ selectedAsset.width }} × {{ selectedAsset.height }}</span
        ><span class="creative-note">默认仅自己可见</span>
        <div class="result-zoom">
          <a-button aria-label="缩小结果" @click="zoom = Math.max(0.25, zoom - 0.25)">−</a-button
          ><a-button @click="zoom = 1">{{ zoom === 1 ? '适应画布' : `${Math.round(zoom * 100)}%` }}</a-button
          ><a-button aria-label="放大结果" @click="zoom = Math.min(3, zoom + 0.25)"><IconPlus /></a-button>
        </div>
      </div>
      <div class="result-preview-scroll">
        <div
          class="result-preview-image"
          :style="{
            '--result-zoom': zoom,
            width: `${zoom * 100}%`,
            marginInline: zoom < 1 ? 'auto' : '0',
          }"
        >
          <CreativeImage :id="selectedAsset.id" full />
        </div>
      </div>
      <div class="result-actions">
        <a-button type="primary" :disabled="!provider?.max_references || composerLocked" @click="useResult(true)">继续编辑</a-button
        ><a-button :disabled="!provider?.max_references || composerLocked" @click="useResult(false)">加入参考图</a-button
        ><a-button @click="download">下载原图</a-button
        ><a-button @click="openPublish">发布到站内</a-button
        ><a-button type="text" status="danger" @click="removeResult">移除结果</a-button>
      </div>
    </template>
  </a-modal>

  <a-modal
    v-model:visible="publishOpen"
    class="creative-theme creative-dialog ai-dialog"
    title="发布生成作品"
    :footer="false"
    ><AutoMetadataOption v-model="publishForm.auto_metadata" :options="publishMetadata" :can-generate="true" />
    <label class="creative-field"
      >作品标题<a-input
        :model-value="publishForm.auto_metadata ? publishMetadata.placeholder_title : publishForm.title"
        aria-label="作品标题"
        :disabled="publishForm.auto_metadata"
        @update:model-value="publishForm.title = $event"
    /></label>
    <label class="creative-field"
      >作品描述<a-textarea v-model="publishForm.description" aria-label="作品描述"
    /></label>
    <label class="creative-field"
      >标签<a-input-tag
        :model-value="publishForm.auto_metadata ? [publishMetadata.placeholder_tag] : publishForm.tags"
        aria-label="作品标签"
        placeholder="输入标签后按回车"
        :disabled="publishForm.auto_metadata"
        @update:model-value="publishForm.tags = $event.filter((tag): tag is string => typeof tag === 'string')"
    /></label>
    <a-checkbox v-model="publishForm.public_consent"
      >将此结果公开发布到站内，遵循现有作品审核与授权规则</a-checkbox
    >
    <p class="creative-note">同一结果只关联一条站内内容。</p>
    <a-button type="primary" :disabled="!publishForm.public_consent" @click="publish">确认发布</a-button></a-modal
  >
</template>

<style scoped>
/*
 * Workspace shell — 参考工作台的「左历史 / 中舞台 / 右详情 + 吸底输入」结构。
 * 颜色只用全站 --site-* 与 --creative-*，保证与首页暖纸视觉同源。
 */
.ai-studio.ws-shell {
  --ws-sidebar: 262px;
  --ws-detail: 282px;
  position: relative;
  isolation: isolate;
  display: grid;
  grid-template-columns: var(--ws-sidebar) minmax(0, 1fr) var(--ws-detail);
  gap: clamp(10px, 1.1vw, 18px);
  width: 100%;
  max-width: 1480px;
  height: calc(100dvh - 76px - 92px);
  min-height: 560px;
  margin: 0 auto;
  padding: 0;
  overflow: hidden;
  color: var(--site-ink);
  background: transparent;
}
.ai-studio.ws-shell::before {
  position: fixed;
  z-index: -1;
  inset: 76px 0 0;
  pointer-events: none;
  content: '';
  opacity: 0.22;
  background-image: radial-gradient(circle, color-mix(in srgb, var(--site-muted) 22%, transparent) 0.6px, transparent 0.8px);
  background-size: 7px 7px;
  mask-image: linear-gradient(to bottom, black, transparent 74%);
}
.ai-studio.is-switching .ws-main,
.ai-studio.is-switching .ws-detail {
  opacity: 0.55;
  pointer-events: none;
}
/* The shell is an isolated stacking context; while its drawer is open it must
   paint above the global sticky topbar, not behind it. */
.ai-studio.is-sidebar-open {
  z-index: 101;
}
.ws-kicker {
  display: block;
  color: var(--site-muted);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.16em;
  line-height: 1.2;
}
.ws-icon-btn {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--site-muted);
  cursor: pointer;
  transition: color var(--transition-fast), background var(--transition-fast);
}
.ws-icon-btn:hover {
  color: var(--site-ink);
  background: var(--site-soft);
}
.ws-icon-btn svg {
  width: 16px;
  height: 16px;
}

/* ── Sidebar ─────────────────────────────────────────────────────────────── */
.ws-sidebar {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--site-line);
  border-radius: 18px;
  background: color-mix(in srgb, var(--site-paper) 92%, transparent);
  box-shadow: var(--shadow-sm);
}
.ws-sidebar-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 14px 10px;
}
.ws-user {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.ws-user-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.ws-user-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--site-ink);
  font-size: 13px;
}
.ws-user-copy small {
  margin-top: 2px;
  overflow: hidden;
  color: var(--site-muted);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ws-sidebar-close {
  display: none;
  margin-left: auto;
}
.ws-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  margin: 0 12px;
  padding: 3px;
  border-radius: 12px;
  background: var(--site-soft);
}
.ws-tabs button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 8px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--site-muted);
  font-size: 12px;
  cursor: pointer;
  transition: color var(--transition-fast), background var(--transition-fast);
}
.ws-tabs button.is-active {
  background: var(--site-paper);
  color: var(--site-ink);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}
.ws-tabs button svg {
  width: 14px;
  height: 14px;
}
.ws-tabs b {
  min-width: 16px;
  padding: 1px 4px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--site-muted) 14%, transparent);
  color: inherit;
  font-size: 9px;
  font-weight: 600;
  text-align: center;
}
.ws-sidebar-body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 10px 10px 14px;
  overscroll-behavior: contain;
  scrollbar-color: var(--site-line) transparent;
}
.ws-workspace-picker {
  position: relative;
  display: flex;
  align-items: center;
  gap: 2px;
  margin-bottom: 8px;
}
.ws-workspace-current {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
  min-height: 36px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 9px;
  background: transparent;
  color: var(--site-ink);
  font: 600 15px/1.2 var(--site-title-font);
  text-align: left;
  cursor: pointer;
}
.ws-workspace-current:hover {
  background: var(--site-soft);
}
.ws-workspace-current span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ws-workspace-current svg {
  flex: 0 0 auto;
  width: 12px;
  height: 12px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.4;
}
.ws-workspace-menu {
  position: absolute;
  z-index: 8;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  display: flex;
  flex-direction: column;
  padding: 5px;
  border: 1px solid var(--site-line);
  border-radius: 12px;
  background: var(--site-paper);
  box-shadow: var(--shadow-md);
}
.ws-workspace-menu button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 34px;
  padding: 0 9px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--site-ink);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
}
.ws-workspace-menu button span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ws-workspace-menu button:hover,
.ws-workspace-menu button.is-active {
  background: var(--site-accent-soft);
}
.ws-workspace-menu .ws-workspace-new {
  justify-content: flex-start;
  color: var(--site-accent);
  border-top: 1px solid var(--site-line);
  border-radius: 0 0 8px 8px;
  margin-top: 4px;
  padding-top: 4px;
}
.ws-workspace-menu svg {
  width: 13px;
  height: 13px;
}
.ws-rename {
  position: absolute;
  z-index: 9;
  inset: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 4px;
  border: 1px solid var(--site-accent);
  border-radius: 10px;
  background: var(--site-paper);
}
.ws-rename input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--site-ink);
  font: inherit;
  font-size: 13px;
}
.ws-rename button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--site-muted);
  cursor: pointer;
}
.ws-rename button:hover {
  background: var(--site-soft);
  color: var(--site-ink);
}
.ws-filters {
  display: flex;
  gap: 5px;
  margin-bottom: 8px;
}
.ws-filters input,
.ws-filters select {
  min-width: 0;
  min-height: 32px;
  padding: 0 8px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-canvas);
  color: var(--site-ink);
  font: inherit;
  font-size: 11px;
}
.ws-filters input {
  flex: 1;
}
.ws-filters select {
  flex: 0 0 82px;
}
.ws-history {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ws-history-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 6px;
  border: 1px solid transparent;
  border-radius: 11px;
  background: transparent;
  color: var(--site-ink);
  text-align: left;
  cursor: pointer;
  transition: background var(--transition-fast), border-color var(--transition-fast);
}
.ws-history-item:hover {
  background: var(--site-soft);
}
.ws-history-item.is-active {
  border-color: color-mix(in srgb, var(--site-accent) 26%, transparent);
  background: var(--site-accent-soft);
}
.ws-history-thumb {
  display: grid;
  flex: 0 0 40px;
  place-items: center;
  width: 40px;
  height: 40px;
  overflow: hidden;
  border-radius: 8px;
  background: var(--site-soft);
  color: var(--site-muted);
}
.ws-history-thumb .creative-image {
  min-height: 0;
}
.ws-history-thumb :deep(.creative-image img) {
  object-fit: cover;
}
.ws-history-thumb > svg {
  width: 16px;
  height: 16px;
}
.ws-history-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.ws-history-prompt {
  display: -webkit-box;
  overflow: hidden;
  color: var(--site-ink);
  font-size: 11.5px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.ws-history-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--site-muted);
  font-size: 9.5px;
  font-variant-numeric: tabular-nums;
}
.ws-history-meta time {
  margin-left: auto;
}
.ws-history-meta .is-success {
  color: rgb(var(--success-6));
}
.ws-history-meta .is-danger {
  color: rgb(var(--danger-6));
}
.ws-history-meta .is-accent {
  color: var(--site-accent);
}
.ws-load-more {
  display: block;
  width: 100%;
  margin-top: 8px;
  padding: 9px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--site-accent);
  font-size: 11px;
  cursor: pointer;
}
.ws-load-more:hover {
  background: var(--site-soft);
}
.ws-sidebar-note {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 96px;
  padding: 16px 12px;
  color: var(--site-muted);
  font-size: 11px;
  line-height: 1.6;
  text-align: center;
}
.ws-liked-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 5px;
}
.ws-liked-item {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  border-radius: 9px;
  background: var(--site-soft);
}
.ws-liked-open {
  display: block;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}
.ws-liked-open .creative-image {
  min-height: 0;
}
.ws-liked-open :deep(.creative-image img) {
  object-fit: cover;
}
.ws-liked-heart {
  position: absolute;
  top: 3px;
  right: 3px;
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: color-mix(in srgb, var(--site-paper) 88%, transparent);
  color: var(--site-accent);
  cursor: pointer;
}
.ws-liked-heart svg {
  width: 12px;
  height: 12px;
}
.ws-sidebar-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px 12px;
  border-top: 1px solid var(--site-line);
}
.ws-credit-ring {
  display: grid;
  flex: 0 0 34px;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 2px solid var(--site-accent);
  border-radius: 50%;
  color: var(--site-ink);
  font-size: 12px;
  font-weight: 650;
}
.ws-credit-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}
.ws-credit-copy strong {
  color: var(--site-ink);
  font-size: 11px;
}
.ws-credit-copy small {
  margin-top: 2px;
  color: var(--site-muted);
  font-size: 9.5px;
}
.ws-clear-failed {
  padding: 5px 7px;
  border: 1px solid var(--site-line);
  border-radius: 7px;
  background: transparent;
  color: rgb(var(--danger-6));
  font-size: 9.5px;
  cursor: pointer;
}

/* ── Stage ───────────────────────────────────────────────────────────────── */
.ws-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.ws-stage-head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0 4px;
}
.ws-menu-btn {
  display: none;
}
.ws-stage-meta {
  display: flex;
  align-items: baseline;
  gap: 5px;
  min-width: 0;
  overflow: hidden;
  color: var(--site-muted);
  font-size: 10.5px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.ws-stage-meta > span,
.ws-stage-meta > strong {
  flex: 0 0 auto;
  white-space: nowrap;
}
.ws-stage-meta > strong:first-child {
  flex: 0 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ws-stage-meta strong {
  color: var(--site-ink);
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.01em;
}
.ws-stage-sep {
  color: var(--site-muted);
  opacity: 0.55;
}
.ws-stage-date,
.ws-stage-count {
  color: var(--site-muted);
  font-size: 10px;
}
.ws-stage-status {
  color: var(--site-muted);
  font-weight: 500;
}
.ws-stage-status::before {
  display: inline-block;
  width: 4px;
  height: 4px;
  margin-right: 4px;
  border-radius: 50%;
  background: currentColor;
  content: '';
  vertical-align: 1px;
}
.ws-stage-meta .is-success {
  color: rgb(var(--success-6));
}
.ws-stage-meta .is-danger {
  color: rgb(var(--danger-6));
}
.ws-stage-meta .is-accent {
  color: var(--site-accent);
}
.ws-stage-all-count {
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--site-accent-soft);
  color: var(--site-accent);
  font-size: 10px;
  font-weight: 600;
}
.ws-view-toggle {
  display: flex;
  gap: 2px;
  margin-left: auto;
  padding: 3px;
  border: 1px solid var(--site-line);
  border-radius: 9px;
  background: var(--site-paper);
}
.ws-view-toggle button {
  min-height: 26px;
  padding: 0 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--site-muted);
  font-size: 11px;
  cursor: pointer;
}
.ws-view-toggle button.is-active {
  background: var(--site-soft);
  color: var(--site-ink);
  font-weight: 600;
}
.ws-prompt-line {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 0 4px 8px;
}
.ws-prompt-line p {
  display: -webkit-box;
  flex: 1;
  overflow: hidden;
  margin: 0;
  color: var(--site-muted);
  font-size: 11.5px;
  line-height: 1.6;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.ws-prompt-line p.is-expanded {
  display: block;
  color: var(--site-ink);
}
.ws-prompt-line button {
  flex: 0 0 auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--site-accent);
  font-size: 11px;
  cursor: pointer;
}
.ws-error {
  margin: 0 4px 8px;
  padding: 9px 11px;
  border: 1px solid color-mix(in srgb, rgb(var(--danger-6)) 30%, transparent);
  border-radius: 9px;
  background: color-mix(in srgb, rgb(var(--danger-6)) 8%, transparent);
  color: rgb(var(--danger-6));
  font-size: 11px;
  line-height: 1.5;
}
.ws-task-note {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 4px 8px;
  padding: 9px 11px;
  border: 1px solid color-mix(in srgb, var(--site-accent) 24%, transparent);
  border-radius: 9px;
  background: var(--site-accent-soft);
  color: var(--site-accent);
  font-size: 11px;
}
.ws-spin {
  animation: ws-spin 1s linear infinite;
}
@keyframes ws-spin {
  to {
    transform: rotate(360deg);
  }
}
.ws-canvas {
  position: relative;
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  min-height: 0;
  overflow: hidden;
  border-radius: 16px;
}
.ws-canvas.is-dragging {
  outline: 2px dashed var(--site-accent);
  outline-offset: -8px;
  background: var(--site-accent-soft);
}
.ws-canvas-image {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  border-radius: 16px;
  background: transparent;
  cursor: zoom-in;
}
.ws-canvas-image .creative-image {
  min-height: 0;
}
.ws-canvas-image :deep(.creative-image img) {
  max-height: 100%;
  border-radius: 16px;
  box-shadow: 0 18px 48px color-mix(in srgb, var(--site-ink) 13%, transparent);
  object-fit: contain;
}
.ws-canvas-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  width: 100%;
  max-height: 100%;
  overflow-y: auto;
  padding: 2px;
}
.ws-canvas-grid.is-single {
  grid-template-columns: minmax(0, 1fr);
  max-width: min(100%, 420px);
  margin-inline: auto;
}
.ws-canvas-grid button {
  overflow: hidden;
  aspect-ratio: 1;
  padding: 0;
  border: 1px solid var(--site-line);
  border-radius: 12px;
  background: var(--site-paper);
  cursor: zoom-in;
}
.ws-canvas-grid button[aria-pressed='true'] {
  border-color: var(--site-accent);
  box-shadow: 0 0 0 1px var(--site-accent);
}
.ws-canvas-grid .creative-image {
  min-height: 0;
}
.ws-canvas-grid :deep(.creative-image img) {
  object-fit: cover;
}
.ws-canvas-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 9px;
  width: 100%;
  min-height: 240px;
  padding: 28px 20px;
  border: 1px dashed var(--site-line);
  border-radius: 16px;
  color: var(--site-muted);
  text-align: center;
}
.ws-canvas-state > svg {
  width: 30px;
  height: 30px;
}
.ws-canvas-state strong {
  color: var(--site-ink);
  font: 600 17px/1.3 var(--site-title-font);
}
.ws-canvas-state p {
  max-width: 42ch;
  margin: 0;
  font-size: 11.5px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.ws-canvas-state.is-error {
  border-color: color-mix(in srgb, rgb(var(--danger-6)) 28%, transparent);
  background: color-mix(in srgb, rgb(var(--danger-6)) 5%, transparent);
}
.ws-text-btn {
  margin-top: 4px;
  padding: 7px 12px;
  border: 1px solid var(--site-line);
  border-radius: 999px;
  background: var(--site-paper);
  color: var(--site-ink);
  font-size: 11px;
  cursor: pointer;
}
.ws-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: clamp(8px, 2vw, 22px);
  padding: 8px 4px 6px;
  color: var(--site-muted);
}
.ws-actions > button,
.ws-more-trigger {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 7px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: inherit;
  font-size: 11px;
  cursor: pointer;
}
.ws-actions > button:hover,
.ws-more-trigger:hover {
  color: var(--site-ink);
  background: var(--site-soft);
}
.ws-actions svg {
  width: 15px;
  height: 15px;
}
.ws-actions .is-liked {
  color: var(--site-accent);
}

/* ── Composer ────────────────────────────────────────────────────────────── */
.ws-composer {
  position: relative;
  margin-top: auto;
  padding-top: 4px;
}
.ws-composer.is-dragging .ws-composer-box {
  border-color: var(--site-accent);
  background: var(--site-accent-soft);
}
.ws-composer-context {
  margin-bottom: 5px;
  color: var(--site-accent);
  font-size: 10px;
}
.ws-composer-box {
  position: relative;
  padding: 10px 10px 8px;
  border: 1px solid var(--site-line);
  border-radius: 16px;
  background: var(--site-paper);
  box-shadow: 0 10px 30px color-mix(in srgb, var(--site-ink) 9%, transparent);
  transition: border-color var(--transition-fast), background var(--transition-fast);
}
.ws-composer-prompt {
  position: relative;
}
.ws-composer-prompt textarea {
  display: block;
  width: 100%;
  min-height: 64px;
  max-height: 172px;
  padding: 6px 6px 8px;
  border: 0;
  outline: 0;
  resize: none;
  background: transparent;
  color: var(--site-ink);
  font: inherit;
  font-size: 13px;
  line-height: 1.6;
  overflow-y: hidden;
}
.ws-composer-prompt textarea::placeholder {
  color: var(--site-muted);
}
.ws-composer-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  margin-top: 6px;
  padding-top: 8px;
  border-top: 1px solid var(--site-line);
}
.ws-composer-tools {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 2px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
}
.ws-composer-tools::-webkit-scrollbar {
  display: none;
}
.ws-tool-btn {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--site-muted);
  cursor: pointer;
}
.ws-tool-btn:hover {
  background: var(--site-soft);
  color: var(--site-ink);
}
.ws-tool-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.ws-tool-btn svg {
  width: 16px;
  height: 16px;
}
.ws-model-chip {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  margin-left: 4px;
  padding: 0 9px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-canvas);
  color: var(--site-ink);
  font-size: 10.5px;
  cursor: pointer;
  white-space: nowrap;
}
.ws-model-chip span {
  color: var(--site-muted);
}
.ws-composer-send {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 7px;
}
.ws-cost-note {
  color: var(--site-muted);
  font-size: 9.5px;
  white-space: nowrap;
}
.ws-send-btn {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: var(--site-action);
  color: var(--site-on-action);
  cursor: pointer;
  transition: background var(--transition-fast), opacity var(--transition-fast);
}
.ws-send-btn:hover:not(:disabled) {
  background: var(--site-accent);
}
.ws-send-btn:disabled {
  opacity: 0.38;
  cursor: not-allowed;
}
.ws-send-btn svg {
  width: 17px;
  height: 17px;
}
.ws-composer-hint {
  padding: 5px 6px 0;
  color: var(--site-muted);
  font-size: 9.5px;
  line-height: 1.4;
}
.ws-settings {
  position: absolute;
  z-index: 12;
  right: 0;
  bottom: calc(100% + 8px);
  left: 0;
  max-height: min(58dvh, 420px);
  overflow-y: auto;
  padding: 14px;
  border: 1px solid var(--site-line);
  border-radius: 14px;
  background: var(--site-paper);
  box-shadow: var(--shadow-md);
  overscroll-behavior: contain;
}
.ws-settings-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.ws-settings-head strong {
  font: 600 15px/1.2 var(--site-title-font);
  color: var(--site-ink);
}
.ws-settings-head button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--site-muted);
  cursor: pointer;
}
.ws-settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.ws-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  color: var(--site-muted);
  font-size: 10px;
}
.ws-field select,
.ws-size-trigger {
  width: 100%;
  min-height: 34px;
  padding: 0 8px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-canvas);
  color: var(--site-ink);
  font: inherit;
  font-size: 11px;
}
.ws-size-trigger {
  text-align: left;
  cursor: pointer;
}
.ws-field-toggle {
  grid-column: 1 / -1;
}
.ws-toggle {
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 34px;
}
.ws-toggle input {
  width: 16px;
  height: 16px;
  accent-color: var(--site-accent);
}
.ws-toggle strong {
  color: var(--site-ink);
  font-size: 11px;
  font-weight: 500;
}
.ws-settings-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--site-line);
}
.ws-settings-actions button {
  min-height: 30px;
  padding: 0 10px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: transparent;
  color: var(--site-ink);
  font-size: 10.5px;
  cursor: pointer;
}
.ws-settings-actions button:hover:not(:disabled) {
  border-color: var(--site-accent);
  color: var(--site-accent);
}
.ws-settings-actions button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.ws-settings-note {
  color: var(--site-muted);
  font-size: 10px;
}
.reference-mentions {
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-paper);
  box-shadow: var(--shadow-md);
}
.reference-mentions button {
  border-radius: 8px;
}

/* ── Detail column ───────────────────────────────────────────────────────── */
.ws-detail {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow-y: auto;
  padding: 16px;
  border: 1px solid var(--site-line);
  border-radius: 18px;
  background: color-mix(in srgb, var(--site-paper) 92%, transparent);
  box-shadow: var(--shadow-sm);
  scrollbar-color: var(--site-line) transparent;
}
.ws-detail-head {
  padding-bottom: 12px;
  border-bottom: 1px solid var(--site-line);
}
.ws-detail-head h2 {
  margin: 4px 0 0;
  color: var(--site-ink);
  font: 600 18px/1.2 var(--site-title-font);
  letter-spacing: -0.02em;
}
.ws-detail-meta {
  display: grid;
  gap: 8px;
  margin: 14px 0;
}
.ws-detail-meta > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}
.ws-detail-meta dt {
  color: var(--site-muted);
  font-size: 10px;
}
.ws-detail-meta dd {
  margin: 0;
  color: var(--site-ink);
  font-size: 11px;
  text-align: right;
  overflow-wrap: anywhere;
}
.ws-detail-meta dd.is-success {
  color: rgb(var(--success-6));
}
.ws-detail-meta dd.is-danger {
  color: rgb(var(--danger-6));
}
.ws-detail-meta dd.is-accent {
  color: var(--site-accent);
}
.ws-detail-block {
  padding: 12px 0;
  border-top: 1px solid var(--site-line);
}
.ws-detail-prompt {
  max-height: 200px;
  overflow-y: auto;
  margin: 6px 0 0;
  color: var(--site-ink);
  font-size: 11px;
  line-height: 1.65;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
.ws-detail-refs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 5px;
  margin-top: 7px;
}
.ws-detail-refs button {
  overflow: hidden;
  aspect-ratio: 1;
  padding: 0;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-soft);
  cursor: zoom-in;
}
.ws-detail-refs .creative-image {
  min-height: 0;
}
.ws-detail-refs :deep(.creative-image img) {
  object-fit: cover;
}
.ws-detail-error {
  padding: 9px 10px;
  border-radius: 8px;
  background: color-mix(in srgb, rgb(var(--danger-6)) 7%, transparent);
  color: rgb(var(--danger-6));
  font-size: 10.5px;
  line-height: 1.5;
  overflow-wrap: anywhere;
}
.ws-detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--site-line);
}
.ws-detail-actions button,
.ws-detail-actions a {
  min-height: 32px;
  padding: 0 11px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: transparent;
  color: var(--site-ink);
  font-size: 10.5px;
  text-decoration: none;
  cursor: pointer;
}
.ws-detail-actions button.is-primary {
  border-color: var(--site-action);
  background: var(--site-action);
  color: var(--site-on-action);
}
.ws-detail-actions button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.ws-detail-empty {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px 10px;
  color: var(--site-muted);
  text-align: center;
}
.ws-detail-empty svg {
  width: 26px;
  height: 26px;
}
.ws-detail-empty p {
  max-width: 24ch;
  margin: 0;
  font-size: 11px;
  line-height: 1.6;
}

/* ── Drawer / modals ─────────────────────────────────────────────────────── */
.ws-scrim {
  display: none;
}
:global(.studio-side-drawer .arco-drawer-body) {
  display: flex;
  flex-direction: column;
  padding: 0;
}
.side-drawer-tabs {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  gap: 4px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--site-line);
  background: var(--site-paper);
}
.side-drawer-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--site-muted);
  font-size: 12px;
  cursor: pointer;
  transition: color var(--transition-fast), background var(--transition-fast), border-color var(--transition-fast);
}
.side-drawer-tab:hover {
  color: var(--site-ink);
  background: var(--site-soft);
}
.side-drawer-tab.is-active {
  border-color: color-mix(in srgb, var(--site-accent) 22%, transparent);
  background: var(--site-accent-soft);
  color: var(--site-accent);
  font-weight: 600;
}
.side-drawer-tab svg {
  width: 14px;
  height: 14px;
}
.side-drawer-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 20px 16px 24px;
}
.side-drawer-idea {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.side-drawer-idea .creative-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.side-drawer-idea .arco-checkbox {
  margin-top: 4px;
}
@media (max-width: 820px) {
  :global(.studio-side-drawer .arco-drawer) {
    width: 100vw !important;
    max-width: 100vw !important;
  }
  :global(.studio-side-drawer .arco-drawer-header) {
    padding: 14px 16px;
  }
  :global(.studio-side-drawer .arco-drawer-body) {
    padding: 0;
  }
  .side-drawer-tabs {
    gap: 2px;
    overflow-x: auto;
    padding: 10px 12px;
    scrollbar-width: none;
  }
  .side-drawer-tabs::-webkit-scrollbar {
    display: none;
  }
  .side-drawer-tab {
    flex: 0 0 auto;
    padding: 7px 10px;
    font-size: 11px;
  }
  .side-drawer-body {
    padding: 14px 12px max(24px, env(safe-area-inset-bottom));
  }
  .ws-library-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 8px;
  }
  .ws-library-grid .library-asset {
    display: flex;
    align-items: center;
    gap: 8px;
    height: auto;
    padding: 8px;
  }
  .ws-library-grid .library-asset-select {
    display: flex;
    flex: 1;
    align-items: center;
    gap: 10px;
    min-width: 0;
    padding: 0;
  }
  .ws-library-grid .library-asset .creative-image {
    flex: 0 0 58px;
    width: 58px;
    height: 58px;
    min-height: 0;
    overflow: hidden;
    border-radius: 8px;
  }
  .ws-library-grid :deep(.creative-image img) {
    object-fit: cover;
  }
  .ws-library-grid .library-asset-select > span {
    overflow: hidden;
    color: var(--site-ink);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ws-library-grid .library-asset-delete {
    flex: 0 0 auto;
    margin: 0;
  }
}

.ai-studio button:focus-visible,
.ai-studio select:focus-visible,
.ai-studio textarea:focus-visible,
.ai-studio input:focus-visible {
  outline: 2px solid var(--site-accent);
  outline-offset: 2px;
}

/* ── Responsive ─────────────────────────────────────────────────────────── */
@media (max-width: 1320px) {
  .ai-studio.ws-shell {
    --ws-sidebar: 244px;
    --ws-detail: 252px;
  }
}
@media (max-width: 1100px) {
  .ai-studio.ws-shell {
    grid-template-columns: 244px minmax(0, 1fr);
  }
  .ws-detail {
    display: none;
  }
}
@media (max-width: 820px) {
  .ai-studio.ws-shell {
    display: block;
    height: auto;
    min-height: 0;
    padding: 0 0 calc(var(--studio-dock-height, 150px) + 18px);
    overflow: visible;
  }
  .ai-studio.ws-shell::before {
    inset: 0;
  }
  .ws-sidebar {
    position: fixed;
    z-index: 100;
    inset: 0 auto 0 0;
    width: min(86vw, 340px);
    border-radius: 0 18px 18px 0;
    background: var(--site-paper);
    box-shadow: 18px 0 44px color-mix(in srgb, var(--site-ink) 18%, transparent);
    transform: translateX(-104%);
    transition: transform 0.24s cubic-bezier(0.22, 1, 0.36, 1);
  }
  .ws-sidebar.is-open {
    transform: translateX(0);
  }
  .ws-sidebar-close {
    display: grid;
  }
  .ws-scrim {
    position: fixed;
    inset: 0;
    z-index: 95;
    display: block;
    border: 0;
    background: color-mix(in srgb, var(--site-ink) 34%, transparent);
  }
  .ws-main {
    display: flex;
    min-height: 0;
  }
  .ws-menu-btn {
    display: grid;
    flex: 0 0 auto;
    margin-left: -4px;
    border: 1px solid var(--site-line);
    background: var(--site-paper);
  }
  .ws-stage-head {
    position: relative;
    z-index: 12;
    min-height: 48px;
    padding: 4px;
    background: color-mix(in srgb, var(--site-canvas) 92%, transparent);
  }
  .ws-stage-meta {
    flex: 1;
    min-width: 0;
  }
  .ws-stage-meta strong {
    font-size: 15px;
  }
  .ws-prompt-line {
    padding-bottom: 6px;
  }
  .ws-canvas {
    flex: none;
    width: 100%;
    min-height: 0;
    overflow: visible;
  }
  .ws-canvas-image {
    height: auto;
  }
  .ws-canvas-image :deep(.creative-image img) {
    width: auto;
    max-width: 100%;
    max-height: max(
      160px,
      calc(var(--studio-visible-height, 100dvh) - var(--studio-dock-height, 0px) - 300px)
    );
    border-radius: 14px;
    box-shadow: none;
  }
  .ws-canvas-grid {
    max-height: none;
    overflow: visible;
  }
  .ws-canvas-grid.is-single {
    max-width: min(
      100%,
      max(
        160px,
        calc(var(--studio-visible-height, 100dvh) - var(--studio-dock-height, 0px) - 300px)
      )
    );
  }
  .ws-actions {
    justify-content: space-around;
    gap: 2px;
    padding: 8px 0 10px;
  }
  .ws-actions > button,
  .ws-more-trigger {
    flex-direction: column;
    gap: 3px;
    min-width: 52px;
    padding: 4px 6px;
    font-size: 9.5px;
  }
  .ws-actions svg {
    width: 17px;
    height: 17px;
  }
  .ws-composer {
    position: fixed;
    z-index: 90;
    right: max(8px, env(safe-area-inset-right));
    bottom: var(--studio-dock-bottom, 0px);
    left: max(8px, env(safe-area-inset-left));
    margin: 0;
    padding: 0 0 max(4px, env(safe-area-inset-bottom));
  }
  .ws-composer-box {
    padding: 10px 10px 8px;
    border-radius: 16px;
    box-shadow: 0 -8px 30px color-mix(in srgb, var(--site-ink) 13%, transparent);
  }
  .ws-composer-prompt textarea {
    min-height: 72px;
    max-height: 240px;
    padding: 6px 4px 6px;
    font-size: 13px;
  }
  .ws-composer-bar {
    gap: 4px;
    margin-top: 4px;
    padding-top: 7px;
  }
  .ws-tool-btn {
    width: 30px;
    height: 30px;
  }
  .ws-send-btn {
    width: 40px;
    height: 40px;
  }
  .ws-cost-note {
    display: none;
  }
  .ws-composer-hint {
    padding-top: 4px;
    font-size: 9px;
  }
  .ws-settings {
    position: fixed;
    right: max(8px, env(safe-area-inset-right));
    bottom: calc(var(--studio-dock-height, 150px) + var(--studio-dock-bottom, 0px) + 8px);
    left: max(8px, env(safe-area-inset-left));
    max-height: min(54dvh, calc(var(--studio-visible-height, 100dvh) - var(--studio-dock-height, 150px) - 24px));
    padding: 12px;
  }
  .ws-settings-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 480px) {
  .ws-model-chip {
    max-width: 132px;
    overflow: hidden;
    text-overflow: ellipsis;
    display: inline-block;
    padding-top: 6px;
  }
  .ws-settings {
    max-height: 50dvh;
  }
}
@media (prefers-reduced-motion: reduce) {
  .ws-sidebar,
  .ws-composer-box,
  .ws-send-btn {
    transition: none;
  }
}
</style>
