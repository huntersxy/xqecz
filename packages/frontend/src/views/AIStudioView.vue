<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { useAIStudio } from '@/composables/useAIStudio'
import { useStudioDock } from '@/composables/useStudioDock'
import { aiApi } from '@/api/creative'
import { contentApi } from '@/api'
import { defaultPublishDescription } from '@/utils/studioPublish'
import type { AIAsset, AITaskView } from '@/types/schemas'
import {
  IconImage,
  IconPlus,
  IconSend,
  IconFile,
  IconLoading,
  IconHeart,
  IconHeartFill,
  IconDownload,
  IconUpload,
  IconDelete,
  IconClose,
  IconEdit,
  IconCheck,
  IconHistory,
  IconMinus,
  IconInfoCircle,
} from '@arco-design/web-vue/es/icon'
import { toast, useConfirm } from '@/composables/useToast'
import CreativeImage from '@/components/CreativeImage.vue'
import CreativeLibrary from '@/components/CreativeLibrary.vue'
import StudioReferences from '@/components/StudioReferences.vue'
import StudioGallery from '@/components/StudioGallery.vue'
import StudioInspector from '@/components/StudioInspector.vue'
import StudioSizePicker from '@/components/StudioSizePicker.vue'
import AutoMetadataOption from '@/components/AutoMetadataOption.vue'
import '@/assets/creative-pages.css'
import { referenceMentionAt, remapReferenceMentions } from '@/utils/creativeMentions'
import { closeShellSidebar } from '@/utils/shellSidebar'
import { formatImageSize, normalizeImageSize, parseImageSize, AUTO_IMAGE_SIZE } from '@/utils/imageSize'
import {
  TASK_LABELS as labels,
  clockOf,
  dayOf,
  isFailedStatus,
  statusClass,
  type GalleryItem,
} from '@/utils/studioTask'

const studio = useAIStudio(),
  router = useRouter(),
  route = useRoute(),
  { confirm } = useConfirm()
const {
  tasks,
  likedAssets,
  sessions,
  account,
  sessionId,
  name,
  prompt,
  size,
  quality,
  count,
  references,
  characterIds,
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
  dailyCreditCap,
  moreTasks,
  moreLiked,
} = studio

const sidebarTab = ref<'workspace' | 'liked'>('workspace')
const galleryOpen = ref(true)
const activeTaskId = ref(0)
const workspaceMenuOpen = ref(false)
const renaming = ref(false)
const renameDraft = ref('')
const sizePickerOpen = ref(false)
const publishOpen = ref(false)
const previewReference = ref(0)
const resourceOpen = ref(false)
const ideaOpen = ref(false)
const resourceError = ref('')
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

const maxCount = computed(() => Math.max(1, provider.value?.max_count ?? 1))

const composerLocked = computed(
  () => !ready.value || !!pending.value || busy.value || uploading.value || confirming.value,
)
const hasDraftContent = computed(
  () =>
    !!prompt.value.trim() ||
    references.value.length > 0 ||
    studio.characterIds.value.length > 0,
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
  tasks.value.find((row) => ['queued', 'running'].includes(row.task.status)),
)
const stageTask = computed(() => {
  const chosen = tasks.value.find((row) => row.task.id === activeTaskId.value)
  if (chosen) return chosen
  if (!selectedAsset.value) return null
  return (
    tasks.value.find((row) => row.assets.some((asset) => asset.id === selectedAsset.value!.id)) ??
    null
  )
})
const stageTaskAssets = computed(() => {
  if (stageTask.value?.assets.length) return stageTask.value.assets
  return selectedAsset.value ? [selectedAsset.value] : []
})
const stageAssets = computed(() => (selectedAsset.value ? [selectedAsset.value] : []))
const stageTaskFailed = computed(() =>
  !!stageTask.value && ['failed', 'cancelled', 'unknown'].includes(stageTask.value.task.status),
)
const selectedIsLiked = computed(() => !!selectedAsset.value?.liked)
const galleryTasks = computed(() => shownTasks.value)
const galleryItems = computed<GalleryItem[]>(() => {
  const items: GalleryItem[] = []
  for (const row of galleryTasks.value) {
    const total = row.assets.length
    if (total) {
      row.assets.forEach((asset, index) =>
        items.push({ key: `asset-${asset.id}`, row, asset, index, total }),
      )
    } else {
      items.push({ key: `task-${row.task.id}`, row, asset: null, index: 0, total: 0 })
    }
  }
  return items
})
const galleryImageCount = computed(() => galleryItems.value.filter((item) => item.asset).length)
const filtered = computed(() => !!search.value || !!status.value)

const mention = ref<ReturnType<typeof referenceMentionAt>>(null),
  mentionIndex = ref(0)
const mentionChoices = computed(() =>
  references.value
    .map((reference, index) => ({ ...reference, index, label: `@图片${index + 1}` }))
    .filter(
      (item) => !mention.value?.query || String(item.index + 1).startsWith(mention.value.query),
    ),
)

function clearFilters() {
  search.value = ''
  status.value = ''
}
function setSidebarTab(tab: 'workspace' | 'liked') {
  sidebarTab.value = tab
}
function selectTask(row: AITaskView) {
  activeTaskId.value = row.task.id
  selectedAsset.value = row.assets[0] ?? null
  galleryOpen.value = false
  awaitingTask.value = 0
  closeShellSidebar()
}
function showGallery() {
  if (!stageTask.value) {
    const row = galleryTasks.value[0]
    if (row) selectTask(row)
  }
  galleryOpen.value = true
  closeShellSidebar()
}
function showTaskView() {
  galleryOpen.value = false
  if (!selectedAsset.value) {
    const row = galleryTasks.value[0]
    if (row) selectTask(row)
  }
}
function openGalleryAsset(row: AITaskView, asset: AIAsset) {
  selectTask(row)
  selectAsset(asset)
}
function selectAsset(asset: AIAsset) {
  selectedAsset.value = asset
  const row = tasks.value.find((item) => item.assets.some((own) => own.id === asset.id))
  activeTaskId.value = row?.task.id ?? 0
}
function selectLiked(asset: AIAsset) {
  activeTaskId.value = 0
  selectedAsset.value = asset
  galleryOpen.value = false
  closeShellSidebar()
}
function isActiveRow(row: AITaskView) {
  if (row.task.id === activeTaskId.value) return true
  return !!selectedAsset.value && row.assets.some((asset) => asset.id === selectedAsset.value!.id)
}
function chooseSession(id: number) {
  workspaceMenuOpen.value = false
  closeShellSidebar()
  void studio.run(() => studio.switchSession(id))
}
async function newWorkspace() {
  workspaceMenuOpen.value = false
  closeShellSidebar()
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
function clearDraft() {
  if (composerLocked.value || !hasDraftContent.value) return
  prompt.value = ''
  references.value = []
  studio.parentAssetId.value = 0
  studio.characterIds.value = []
  mention.value = null
  void focusPrompt()
}
async function generate() {
  if (!canGenerate.value) return
  confirming.value = true
  try {
    await studio.run(async () => {
      if (
        !pending.value &&
        !(await confirm(
          `本次生成 ${count.value} 张图片，将消耗 ${cost.value} 额度，失败自动返还。确认开始？`,
        ))
      )
        return
      const task = await studio.generate()
      if (task) awaitingTask.value = task.id
    })
  } finally {
    confirming.value = false
  }
}
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
  if (!files.length) return
  event.preventDefault()
  void uploadFiles(files)
}
function dropImages(event: DragEvent) {
  dragging.value = false
  void uploadFiles(event.dataTransfer?.files ?? null)
}

async function likeSelected() {
  if (!selectedAsset.value) return
  await studio.run(() => studio.toggleLike(selectedAsset.value!))
}
async function continueFromSelected() {
  if (!selectedAsset.value || composerLocked.value) return
  await studio.run(async () => {
    studio.continueFrom(selectedAsset.value!)
    await focusPrompt()
  })
}
async function cancelQueued(row: AITaskView) {
  await studio.run(async () => {
    await aiApi.cancel(row.task.id)
    await studio.refresh()
  })
}
async function reusePrompt(row: AITaskView): Promise<boolean> {
  if (composerLocked.value) return false
  if (
    prompt.value &&
    prompt.value !== row.task.prompt &&
    !(await confirm('用此任务的提示词替换当前草稿？参考图与参数保持当前设置。'))
  )
    return false
  prompt.value = row.task.prompt
  enhancePrompt.value = false
  await focusPrompt()
  return true
}
async function retryGalleryTask(row: AITaskView) {
  if (await reusePrompt(row)) galleryOpen.value = false
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
    await studio.refresh()
  })
}

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
    ideaOpen.value = false
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
  publishForm.value.description = defaultPublishDescription(
    studio.tasks.value,
    selectedAsset.value?.id,
    studio.prompt.value,
  )
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
function openResource() {
  resourceError.value = ''
  resourceOpen.value = true
}
function openIdea() {
  ideaOpen.value = true
}
async function applyResource(value: {
  text: string
  assets: number[]
  characters: number[]
  parameters: Record<string, unknown>
  action: 'append' | 'replace' | 'insert'
  label?: string
}) {
  resourceError.value = ''
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
  if (error.value) resourceError.value = error.value
  else if (value.text || value.characters.length || value.assets.length)
    toast.success(`已导入${value.label ? `「${value.label}」` : '素材'}`)
}

watch(sessionId, () => {
  mention.value = null
  awaitingTask.value = 0
  activeTaskId.value = 0
  galleryOpen.value = true
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
  () => closeShellSidebar(),
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
    :class="{ 'is-switching': switching }"
  >
    <Teleport defer to="#public-shell-workspace">
      <div class="ws-sidebar-content" aria-label="工作区导航">
        <div class="ws-sidebar-top">
          <span class="ws-sidebar-kicker">创作工作台</span>
          <small>{{ studio.saveState.value || '已同步' }}</small>
        </div>

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
          <small v-if="quotaVisible"
            >每日 {{ dailyCreditTotal }}/{{ dailyCreditCap }} · 其他 {{ otherCreditTotal }}</small
          >
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
      </div>
    </Teleport>

    <section class="ws-main" aria-label="创作舞台">
      <aside class="ws-notice" role="note" aria-label="积分说明">
        <IconInfoCircle class="ws-notice-icon" aria-hidden="true" />
        <p class="ws-notice-copy">
          <strong>积分说明</strong>
          <span>每日登录把当日额度补满至 <b>{{ dailyCreditCap }}</b></span>
          <span>作品审核通过额外 <b>+1</b> 额度，不占每日上限</span>
          <span>生成失败或取消自动退还额度</span>
        </p>
      </aside>

      <header class="ws-stage-head">
        <div class="ws-stage-title">
          <strong>{{ name }}</strong>
          <span v-if="galleryOpen" class="ws-stage-sub">
            <span class="ws-stage-sub-lead">全部结果<i aria-hidden="true" /></span
            >{{ galleryTasks.length }} 个任务<i aria-hidden="true" />{{ galleryImageCount }} 张作品
          </span>
          <span v-else-if="stageTask" class="ws-stage-sub">
            {{ dayOf(stageTask.task.created_at) }} {{ clockOf(stageTask.task.created_at) }}<i aria-hidden="true" /><span
              :class="statusClass(stageTask.task.status)"
              >{{ labels[stageTask.task.status] }}</span
            ><i aria-hidden="true" />{{ stageTask.assets.length }} 张
          </span>
          <span v-else class="ws-stage-sub">把画面写下来</span>
        </div>
        <div v-if="activeTask" class="ws-live" role="status">
          <IconLoading class="ws-spin" />
          <span>
            任务 #{{ activeTask.task.id }} · {{ labels[activeTask.task.status] }}，离开页面仍会继续
          </span>
        </div>
        <div class="ws-view-toggle" role="radiogroup" aria-label="查看方式" :class="{ 'is-task': !galleryOpen }">
          <span class="ws-view-thumb" aria-hidden="true" />
          <button
            type="button"
            role="radio"
            :aria-checked="galleryOpen"
            :class="{ 'is-active': galleryOpen }"
            @click="showGallery"
          >
            全部结果
          </button>
          <button
            type="button"
            role="radio"
            :aria-checked="!galleryOpen"
            :class="{ 'is-active': !galleryOpen }"
            @click="showTaskView"
          >
            单个任务
          </button>
        </div>
      </header>

      <div v-if="error" class="ws-error" role="alert">{{ error }}</div>

      <Transition name="ws-view" mode="out-in">
        <StudioGallery
          v-if="galleryOpen"
          key="gallery"
          :items="galleryItems"
          :ready="ready"
          :filtered="filtered"
          @open="openGalleryAsset"
          @retry="retryGalleryTask"
          @remove="archiveTask"
          @like="(asset) => studio.run(() => studio.toggleLike(asset))"
          @create="focusPrompt"
          @clear-filters="clearFilters"
        />

        <div v-else key="task" class="ws-task">
          <div class="ws-task-main">
            <div class="ws-canvas" :class="{ 'is-dragging': dragging }">
              <div v-if="stageAssets.length" :key="stageAssets[0]?.id" class="ws-canvas-image">
                <CreativeImage :id="stageAssets[0]?.id ?? 0" full />
              </div>

              <div v-else-if="stageTaskFailed && stageTask" class="ws-canvas-state is-error">
                <IconImage />
                <strong>{{ labels[stageTask.task.status] }}</strong>
                <p v-if="stageTask.task.error">{{ stageTask.task.error }}</p>
                <p v-else-if="stageTask.task.status === 'unknown'">生成失败，额度已自动返还。</p>
                <div class="ws-state-actions">
                  <button type="button" class="ws-text-btn" @click="reusePrompt(stageTask)">复用提示词重试</button>
                  <button
                    v-if="isFailedStatus(stageTask.task.status)"
                    type="button"
                    class="ws-text-btn"
                    @click="archiveTask(stageTask)"
                  >
                    删除记录
                  </button>
                </div>
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

            <div v-if="stageTaskAssets.length > 1" class="ws-result-strip" aria-label="本任务结果">
              <button
                v-for="(asset, index) in stageTaskAssets"
                :key="asset.id"
                type="button"
                :aria-label="`查看第 ${index + 1} 张`"
                :aria-pressed="selectedAsset?.id === asset.id"
                @click="selectAsset(asset)"
              >
                <CreativeImage :id="asset.id" />
              </button>
            </div>

            <div v-if="selectedAsset" class="ws-actions" aria-label="结果操作">
              <button
                type="button"
                :disabled="!provider?.max_references || composerLocked"
                @click="continueFromSelected"
              >
                <IconEdit />
                <span>继续编辑</span>
              </button>
              <button
                v-if="stageTask"
                type="button"
                :disabled="composerLocked"
                @click="reusePrompt(stageTask)"
              >
                <IconHistory />
                <span>复用提示词</span>
              </button>
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
                <IconUpload />
                <span>发布</span>
              </button>
              <button
                type="button"
                class="is-danger"
                aria-label="移除结果"
                title="移除结果"
                @click="removeResult"
              >
                <IconDelete />
              </button>
            </div>
          </div>

          <StudioInspector
            :task="stageTask"
            :asset="selectedAsset"
            :liked="selectedIsLiked"
            @preview="(id) => (previewReference = id)"
            @cancel="cancelQueued"
          />
        </div>
      </Transition>

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
                aria-label="上传参考图"
                :disabled="!provider?.max_references || references.length >= (provider?.max_references ?? 0)"
                @click="fileInput?.click()"
              >
                <IconPlus /><span>上传参考图</span>
              </button>
              <button type="button" class="ws-tool-btn" aria-label="角色与模板" @click="openResource">
                <IconFile /><span>角色与模板</span>
              </button>
            </div>
            <div class="ws-composer-options">
              <button
                type="button"
                class="ws-option"
                aria-label="画面尺寸"
                :disabled="composerLocked"
                @click="sizePickerOpen = true"
              >
                {{ formatImageSize(size) }}
              </button>
              <div class="ws-stepper" role="group" aria-label="输出数量" :class="{ 'is-single': maxCount <= 1 }">
                <button
                  type="button"
                  aria-label="减少数量"
                  :disabled="composerLocked || count <= 1"
                  @click="count = Math.max(1, count - 1)"
                >
                  <IconMinus />
                </button>
                <output aria-live="polite">{{ count }} 张</output>
                <button
                  type="button"
                  aria-label="增加数量"
                  :disabled="composerLocked || count >= maxCount"
                  @click="count = Math.min(maxCount, count + 1)"
                >
                  <IconPlus />
                </button>
              </div>
              <label
                class="ws-option ws-option-toggle"
                :title="promptEnhancement.prompt || '暂无增强提示词'"
              >
                <input
                  v-model="enhancePrompt"
                  type="checkbox"
                  role="switch"
                  :disabled="!promptEnhancement.enabled || !promptEnhancement.prompt"
                />
                <span>提示词增强</span>
              </label>
            </div>
          </div>
          <div class="ws-composer-footer">
            <span v-if="uploading" class="ws-composer-hint" role="status">正在上传参考图…</span>
            <span v-else-if="pending" class="ws-composer-hint" role="status">
              上次提交结果尚未确认，草稿已锁定。请用原请求继续确认。
            </span>
            <span v-else class="ws-composer-tip">
              Enter 换行 · Shift + Enter 生成
              <span v-if="provider?.max_references">· 可拖入或粘贴 PNG / JPEG / WebP</span>
            </span>
            <span class="ws-composer-secondary">
              <button
                type="button"
                class="is-clear"
                :disabled="composerLocked || !hasDraftContent"
                title="清空提示词和参考图"
                @click="clearDraft"
              >
                清空
              </button>
              <button type="button" :disabled="composerLocked" @click="studio.run(studio.save)">
                保存草稿
              </button>
              <button
                type="button"
                :disabled="composerLocked || !prompt.trim()"
                @click="openIdea"
              >
                投稿创意
              </button>
            </span>
            <button
              type="button"
              class="ws-send-btn"
              :disabled="!canGenerate"
              :aria-label="pending ? '确认上次提交' : '开始生成'"
              :title="quotaVisible ? quotaNote : '管理员生成不消耗额度'"
              @click="generate"
            >
              <IconLoading v-if="busy || uploading" class="ws-spin" />
              <template v-else>
                <IconSend />
                <span>生成</span>
              </template>
            </button>
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
      </section>
    </section>

  <StudioSizePicker v-model:visible="sizePickerOpen" :model-value="size" @select="size = $event" />

  <a-modal
    v-model:visible="resourceOpen"
    class="creative-theme creative-dialog studio-picker"
    :footer="false"
    :closable="false"
    hide-title
    unmount-on-close
    :width="980"
  >
    <div class="picker">
      <section class="picker-main">
        <header class="picker-head">
          <div class="picker-heading">
            <h2>角色 / 模板</h2>
            <p class="picker-note">点选即导入角色参考图或提示词，无需二次确认。</p>
          </div>
          <button type="button" class="picker-close" aria-label="关闭" @click="resourceOpen = false">
            <IconClose />
          </button>
        </header>
        <div class="picker-body">
          <CreativeLibrary
            v-if="resourceOpen"
            :applied-characters="characterIds"
            :applied-assets="references.map((reference) => reference.asset_id)"
            :error="resourceError"
            @apply="applyResource"
          />
        </div>
      </section>
    </div>
  </a-modal>

  <a-modal
    v-model:visible="ideaOpen"
    class="creative-theme creative-dialog"
    title="投稿创意"
    :footer="false"
    :width="560"
    unmount-on-close
  >
    <div class="idea-form">
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
  </a-modal>

  <a-modal
    :visible="!!previewReference"
    class="creative-theme creative-dialog ai-dialog"
    title="参考图预览"
    :footer="false"
    @cancel="previewReference = 0"
    ><div class="reference-large"><CreativeImage v-if="previewReference" :id="previewReference" full /></div
  ></a-modal>

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
  </div>
</template>

<style scoped>
@media (min-width: 821px) {
  :global(.public-shell.is-workspace .public-shell-content) {
    max-width: min(2240px, 100%);
    padding: 20px clamp(20px, 2.2vw, 40px) 24px;
  }
}
.ai-studio.ws-shell {
  position: relative;
  isolation: isolate;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: calc(100dvh - 76px - 44px);
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
.ai-studio.is-switching .ws-main {
  opacity: 0.55;
  pointer-events: none;
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

.ws-sidebar-content {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.ws-sidebar-top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 7px;
}
.ws-sidebar-kicker {
  color: var(--site-muted);
  font-size: 10px;
  letter-spacing: 0.14em;
}
.ws-sidebar-top small {
  overflow: hidden;
  color: var(--site-muted);
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ws-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  margin: 0;
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
  padding: 8px 0 12px;
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
  padding: 10px 0 12px;
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

.ws-main {
  position: relative;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.ws-notice {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 9px;
  margin: 0 0 10px;
  padding: 8px 13px;
  border: 1px solid var(--site-line);
  border-radius: 12px;
  background: color-mix(in srgb, var(--site-soft) 82%, transparent);
  color: var(--site-muted);
  font-size: 11.5px;
  line-height: 1.5;
}
.ws-notice-icon {
  flex: 0 0 auto;
  width: 14px;
  height: 14px;
  color: rgb(var(--primary-6));
  opacity: 0.9;
}
.ws-notice-copy {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 0;
  min-width: 0;
  margin: 0;
}
.ws-notice-copy strong {
  margin-right: 8px;
  color: var(--site-ink);
  font-weight: 600;
  letter-spacing: 0.01em;
}
.ws-notice-copy span {
  display: inline-flex;
  align-items: center;
  white-space: nowrap;
}
.ws-notice-copy span + span::before {
  margin: 0 8px;
  color: var(--site-line);
  content: '/';
}
.ws-notice-copy b {
  margin: 0 3px;
  color: var(--site-ink);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}
.ws-stage-head {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 56px;
  padding: 0 6px 6px;
}
.ws-stage-title {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.ws-stage-title strong {
  overflow: hidden;
  color: var(--site-ink);
  font: 600 21px/1.2 var(--site-title-font);
  letter-spacing: -0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ws-stage-sub {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
  color: var(--site-muted);
  font-size: 11.5px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.ws-stage-sub i {
  flex: 0 0 auto;
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.5;
}
.ws-stage-sub .is-success {
  color: rgb(var(--success-6));
}
.ws-stage-sub .is-danger {
  color: rgb(var(--danger-6));
}
.ws-stage-sub .is-accent {
  color: var(--site-accent);
}
.ws-live {
  display: inline-flex;
  flex: 0 1 auto;
  align-items: center;
  gap: 7px;
  min-width: 0;
  max-width: 380px;
  padding: 6px 13px;
  border-radius: 999px;
  background: var(--site-accent-soft);
  color: var(--site-accent);
  font-size: 11.5px;
}
.ws-live > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ws-live svg {
  flex: 0 0 auto;
  width: 13px;
  height: 13px;
}
.ws-spin {
  animation: ws-spin 1s linear infinite;
}
@keyframes ws-spin {
  to {
    transform: rotate(360deg);
  }
}
.ws-view-toggle {
  position: relative;
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: 1fr 1fr;
  padding: 3px;
  border: 1px solid var(--site-line);
  border-radius: 999px;
  background: var(--site-paper);
  box-shadow: var(--shadow-sm);
}
.ws-view-thumb {
  position: absolute;
  top: 3px;
  bottom: 3px;
  left: 3px;
  width: calc((100% - 6px) / 2);
  border-radius: 999px;
  background: var(--site-action);
  box-shadow: 0 4px 12px color-mix(in srgb, var(--site-action) 28%, transparent);
  transition: transform 0.36s cubic-bezier(0.22, 1, 0.36, 1);
}
.ws-view-toggle.is-task .ws-view-thumb {
  transform: translateX(100%);
}
.ws-view-toggle button {
  position: relative;
  z-index: 1;
  min-height: 30px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--site-muted);
  font-size: 12px;
  white-space: nowrap;
  cursor: pointer;
  transition: color 0.25s ease;
}
.ws-view-toggle button:hover:not(.is-active) {
  color: var(--site-ink);
}
.ws-view-toggle button.is-active {
  color: var(--site-on-action);
  font-weight: 600;
}
.ws-error {
  margin: 0 6px 10px;
  padding: 10px 14px;
  border: 1px solid color-mix(in srgb, rgb(var(--danger-6)) 28%, transparent);
  border-radius: 14px;
  background: color-mix(in srgb, rgb(var(--danger-6)) 8%, var(--site-paper));
  color: rgb(var(--danger-6));
  font-size: 12px;
  line-height: 1.5;
}
.ws-view-enter-active,
.ws-view-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
}
.ws-view-enter-from {
  opacity: 0;
  transform: translateY(10px);
}
.ws-view-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

.ws-task {
  display: grid;
  flex: 1;
  grid-template-columns: minmax(0, 1fr) 300px;
  grid-template-rows: minmax(0, 1fr);
  gap: clamp(12px, 1.4vw, 22px);
  min-height: 0;
}
.ws-task-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.ws-task > .ws-detail {
  align-self: start;
  max-height: 100%;
}
.ws-canvas {
  position: relative;
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  min-height: 0;
  padding: 18px;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--site-line) 70%, transparent);
  border-radius: 26px;
  background: radial-gradient(
    120% 90% at 50% 0%,
    color-mix(in srgb, var(--site-accent) 7%, var(--site-paper)) 0%,
    color-mix(in srgb, var(--site-paper) 55%, transparent) 72%
  );
  transition: border-color 0.2s ease, background 0.2s ease;
}
.ws-canvas.is-dragging {
  border-color: var(--site-accent);
  border-style: dashed;
  background: var(--site-accent-soft);
}
.ws-canvas-image {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  border-radius: 18px;
  animation: ws-pop 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes ws-pop {
  from {
    opacity: 0;
    transform: scale(0.97);
  }
}
.ws-canvas-image .creative-image {
  min-height: 0;
}
.ws-canvas-image :deep(.creative-image img) {
  max-height: 100%;
  border-radius: 18px;
  box-shadow: 0 24px 56px color-mix(in srgb, var(--site-ink) 18%, transparent);
  object-fit: contain;
}
.ws-canvas-state {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  align-self: stretch;
  padding: 28px 20px;
  color: var(--site-muted);
  text-align: center;
}
.ws-canvas-state > svg {
  width: 32px;
  height: 32px;
}
.ws-canvas-state strong {
  color: var(--site-ink);
  font: 600 18px/1.3 var(--site-title-font);
}
.ws-canvas-state p {
  max-width: 42ch;
  margin: 0;
  font-size: 12px;
  line-height: 1.7;
  overflow-wrap: anywhere;
}
.ws-canvas-state.is-error {
  border-radius: 18px;
  background: color-mix(in srgb, rgb(var(--danger-6)) 5%, transparent);
}
.ws-state-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}
.ws-text-btn {
  margin-top: 4px;
  padding: 8px 16px;
  border: 1px solid var(--site-line);
  border-radius: 999px;
  background: var(--site-paper);
  color: var(--site-ink);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}
.ws-text-btn:hover {
  border-color: var(--site-accent);
  background: var(--site-accent-soft);
}
.ws-result-strip {
  display: flex;
  justify-content: center;
  gap: 8px;
  padding: 14px 4px 0;
}
.ws-result-strip button {
  overflow: hidden;
  width: 52px;
  height: 52px;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 14px;
  background: var(--site-soft);
  opacity: 0.68;
  cursor: pointer;
  transition:
    opacity 0.2s ease,
    transform 0.25s cubic-bezier(0.22, 1, 0.36, 1),
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}
.ws-result-strip button:hover {
  opacity: 1;
}
.ws-result-strip button[aria-pressed='true'] {
  border-color: var(--site-accent);
  box-shadow: 0 8px 18px color-mix(in srgb, var(--site-accent) 24%, transparent);
  opacity: 1;
  transform: translateY(-2px);
}
.ws-result-strip .creative-image {
  min-height: 0;
}
.ws-result-strip :deep(.creative-image img) {
  object-fit: cover;
}
.ws-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 2px;
  width: fit-content;
  max-width: 100%;
  margin: 14px auto 0;
  padding: 4px;
  border: 1px solid var(--site-line);
  border-radius: 999px;
  background: color-mix(in srgb, var(--site-paper) 90%, transparent);
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(12px);
}
.ws-actions > button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--site-muted);
  font-size: 12px;
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
}
.ws-actions > button:hover:not(:disabled) {
  background: var(--site-soft);
  color: var(--site-ink);
}
.ws-actions > button.is-danger:hover:not(:disabled) {
  background: color-mix(in srgb, rgb(var(--danger-6)) 10%, transparent);
  color: rgb(var(--danger-6));
}
.ws-actions > button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.ws-actions > button[aria-pressed='true'] {
  color: var(--site-accent);
}
.ws-actions svg {
  width: 15px;
  height: 15px;
}
.ws-actions .is-liked {
  color: var(--site-accent);
}

.ws-composer {
  position: relative;
  width: 100%;
  max-width: 800px;
  margin: auto auto 0;
  padding-top: 6px;
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
  padding: 12px 12px 8px;
  border: 1px solid color-mix(in srgb, var(--site-line) 80%, transparent);
  border-radius: 24px;
  background: color-mix(in srgb, var(--site-paper) 94%, transparent);
  box-shadow: 0 1px 2px color-mix(in srgb, var(--site-ink) 6%, transparent),
    0 20px 48px color-mix(in srgb, var(--site-ink) 12%, transparent);
  backdrop-filter: blur(16px) saturate(1.2);
  transition: border-color var(--transition-fast), background var(--transition-fast), box-shadow var(--transition-base);
}
.ws-composer-box:focus-within {
  border-color: color-mix(in srgb, var(--site-accent) 45%, var(--site-line));
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--site-accent) 11%, transparent),
    0 20px 48px color-mix(in srgb, var(--site-ink) 14%, transparent);
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
  border-top: 1px solid color-mix(in srgb, var(--site-line) 70%, transparent);
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
.ws-composer-options {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.ws-option {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 30px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: var(--site-soft);
  color: var(--site-ink);
  font-size: 11px;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color var(--transition-fast), background var(--transition-fast);
}
button.ws-option:hover:not(:disabled),
.ws-option:has(select:hover),
.ws-option-toggle:hover {
  border-color: color-mix(in srgb, var(--site-accent) 35%, transparent);
  background: var(--site-accent-soft);
}
.ws-option:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.ws-option-toggle {
  gap: 8px;
  cursor: pointer;
}
.ws-option-toggle input {
  position: relative;
  flex: 0 0 auto;
  width: 30px;
  height: 18px;
  margin: 0;
  border-radius: 999px;
  background: color-mix(in srgb, var(--site-muted) 38%, transparent);
  appearance: none;
  cursor: pointer;
  transition: background 0.25s ease;
}
.ws-option-toggle input::after {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
  content: '';
  transition: transform 0.28s cubic-bezier(0.34, 1.4, 0.64, 1);
}
.ws-option-toggle input:checked {
  background: var(--site-accent);
}
.ws-option-toggle input:checked::after {
  transform: translateX(12px);
}
.ws-option-toggle input:disabled {
  cursor: not-allowed;
}
.ws-stepper {
  display: inline-flex;
  align-items: center;
  height: 30px;
  padding: 2px;
  border-radius: 999px;
  background: var(--site-soft);
}
.ws-stepper button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--site-ink);
  cursor: pointer;
  transition: background 0.15s ease, transform 0.15s ease, opacity 0.15s ease;
}
.ws-stepper button:hover:not(:disabled) {
  background: var(--site-paper);
  box-shadow: var(--shadow-sm);
}
.ws-stepper button:active:not(:disabled) {
  transform: scale(0.9);
}
.ws-stepper button:disabled {
  opacity: 0.32;
  cursor: not-allowed;
}
.ws-stepper svg {
  width: 12px;
  height: 12px;
}
.ws-stepper output {
  min-width: 38px;
  color: var(--site-ink);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  text-align: center;
}
.ws-stepper.is-single button {
  display: none;
}
.ws-stepper.is-single output {
  padding: 0 12px;
}
@media (max-width: 560px) {
  .ws-composer-bar { flex-wrap: wrap; }
  .ws-composer-options {
    width: 100%;
    justify-content: flex-end;
  }
}
.ws-tool-btn {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px 0 10px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--site-muted);
  font-size: 12px;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background var(--transition-fast),
    color var(--transition-fast),
    border-color var(--transition-fast),
    transform 0.15s ease;
}
.ws-tool-btn:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--site-accent) 30%, transparent);
  background: var(--site-accent-soft);
  color: var(--site-accent);
}
.ws-tool-btn:active:not(:disabled) {
  transform: scale(0.97);
}
.ws-tool-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.ws-tool-btn svg {
  width: 15px;
  height: 15px;
}
@media (max-width: 560px) {
  .ws-tool-btn {
    height: 30px;
    padding: 0 9px 0 8px;
    font-size: 11px;
  }
}
.ws-send-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 92px;
  height: 40px;
  padding: 0 20px;
  border: 0;
  border-radius: 999px;
  background: var(--site-accent);
  color: var(--site-paper);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--site-accent) 30%, transparent);
  transition: background var(--transition-fast), opacity var(--transition-fast), transform var(--transition-fast), box-shadow var(--transition-fast);
}
.ws-send-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--site-accent) 86%, var(--site-ink));
  box-shadow: 0 12px 26px color-mix(in srgb, var(--site-accent) 38%, transparent);
  transform: translateY(-1px);
}
.ws-send-btn:active:not(:disabled) {
  transform: translateY(1px);
}
.ws-send-btn:disabled {
  opacity: 0.38;
  cursor: not-allowed;
  box-shadow: none;
}
.ws-send-btn svg {
  width: 16px;
  height: 16px;
}
.ws-composer-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: 5px 6px 0;
}
.ws-composer-tip,
.ws-composer-hint {
  flex: 1 1 auto;
  min-width: 0;
  color: var(--site-muted);
  font-size: 9.5px;
  line-height: 1.4;
}
.ws-composer-secondary {
  display: flex;
  flex: 0 0 auto;
  gap: 6px;
  margin-left: auto;
}
.ws-composer-secondary button {
  min-height: 30px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--site-muted);
  font-size: 11px;
  cursor: pointer;
  transition: border-color var(--transition-fast), color var(--transition-fast), background var(--transition-fast);
}
.ws-composer-secondary button:hover:not(:disabled) {
  border-color: var(--site-accent);
  background: var(--site-accent-soft);
  color: var(--site-accent);
}
.ws-composer-secondary button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.ws-composer-secondary button.is-clear {
  color: var(--site-muted);
}
.ws-composer-secondary button.is-clear:hover:not(:disabled) {
  border-color: color-mix(in srgb, rgb(var(--danger-6)) 35%, var(--site-line));
  background: color-mix(in srgb, rgb(var(--danger-6)) 8%, transparent);
  color: rgb(var(--danger-6));
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

:global(.studio-picker .arco-modal) {
  max-width: calc(100vw - 32px);
  overflow: hidden;
  border-radius: 24px;
  background: var(--site-paper);
  box-shadow: 0 32px 80px rgba(24, 18, 20, 0.28);
}
:global(.studio-picker .arco-modal-body) {
  padding: 0;
}
.picker {
  display: flex;
  flex-direction: column;
  height: min(680px, calc(100dvh - 64px));
}
.picker-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
.picker-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 24px 28px 16px;
}
.picker-heading {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.picker-head h2 {
  margin: 0;
  color: var(--site-ink);
  font: 600 25px/1.2 var(--site-title-font);
  letter-spacing: -0.02em;
}
.picker-note {
  margin: 0;
  color: color-mix(in srgb, var(--site-ink) 62%, var(--site-paper));
  font-size: 12px;
  line-height: 1.6;
}
.picker-close {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--site-line);
  border-radius: 50%;
  background: var(--site-paper);
  color: var(--site-ink);
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    background 0.2s ease,
    transform 0.2s ease;
}
.picker-close:hover {
  border-color: var(--site-accent);
  background: var(--site-accent-soft);
  transform: rotate(90deg);
}
.picker-close svg {
  width: 15px;
  height: 15px;
}
.picker-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 6px 28px 26px;
  scrollbar-width: thin;
  scrollbar-color: var(--site-line) transparent;
}
.picker-body::-webkit-scrollbar {
  width: 8px;
}
.picker-body::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: var(--site-line);
}
.picker button:focus-visible {
  outline: 2px solid var(--site-accent);
  outline-offset: 2px;
}

.idea-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.idea-form .creative-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.idea-form .arco-checkbox {
  margin-top: 4px;
}

@media (max-width: 820px) {
  :global(.studio-picker .arco-modal) {
    width: calc(100vw - 16px) !important;
    border-radius: 20px;
  }
  .picker {
    height: min(640px, calc(100dvh - 40px));
  }
  .picker-head {
    padding: 16px 14px 12px 18px;
  }
  .picker-head h2 {
    font-size: 21px;
  }
  .picker-body {
    padding: 4px 14px max(20px, env(safe-area-inset-bottom));
  }
}
.ai-studio button:focus-visible,
.ai-studio select:focus-visible,
.ai-studio input:focus-visible {
  outline: 2px solid var(--site-accent);
  outline-offset: 2px;
}
.ai-studio .ws-composer-prompt textarea:focus,
.ai-studio .ws-composer-prompt textarea:focus-visible {
  outline: none;
}

@media (max-width: 1320px) {
  .ws-task {
    grid-template-columns: minmax(0, 1fr) 272px;
  }
}
@media (max-width: 1100px) {
  .ws-task {
    display: block;
    overflow-y: auto;
    scrollbar-width: thin;
  }
  .ws-task-main {
    min-height: min(560px, 62vh);
  }
  .ws-task > .ws-detail {
    max-height: none;
    margin-top: 16px;
    overflow: visible;
  }
}
@media (max-width: 820px) {
  .ai-studio.ws-shell {
    display: block;
    height: auto;
    min-height: 0;
    padding: 0 0 calc(var(--studio-dock-height, 150px) + 28px);
    overflow: visible;
  }
  .ai-studio.ws-shell::before {
    inset: 0;
  }
  .ws-main {
    display: flex;
    min-height: 0;
  }
  .ws-stage-head {
    position: relative;
    z-index: 12;
    flex-wrap: wrap;
    gap: 10px;
    min-height: 0;
    padding: 4px 2px 10px;
  }
  .ws-stage-title {
    flex: 1 1 140px;
  }
  .ws-stage-title strong {
    font-size: 18px;
  }
  .ws-stage-sub {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ws-stage-sub i {
    display: inline-block;
    margin: 0 8px;
    vertical-align: middle;
  }
  .ws-stage-sub-lead {
    display: none;
  }
  .ws-live {
    order: 3;
    flex: 1 0 100%;
    max-width: none;
  }
  .ws-view-toggle button {
    padding: 0 12px;
  }
  .ws-task {
    display: block;
    overflow: visible;
  }
  .ws-task-main {
    min-height: 0;
  }
  .ws-task > .ws-detail {
    margin-top: 14px;
    padding: 14px;
    border-radius: 18px;
  }
  .ws-canvas {
    flex: none;
    width: 100%;
    min-height: 0;
    padding: 10px;
    overflow: visible;
    border-radius: 20px;
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
  .ws-actions {
    width: 100%;
    flex-wrap: nowrap;
    justify-content: space-between;
    gap: 0;
    margin-top: 10px;
    padding: 4px 2px;
  }
  .ws-actions > button {
    flex: 1 1 0;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
    min-height: 46px;
    padding: 4px 2px;
    font-size: 10px;
    white-space: nowrap;
  }
  .ws-actions svg {
    width: 17px;
    height: 17px;
  }
  .ws-composer {
    position: fixed;
    z-index: 90;
    right: max(8px, env(safe-area-inset-right));
    bottom: calc(var(--studio-dock-bottom, 0px) + 10px);
    left: max(8px, env(safe-area-inset-left));
    width: auto;
    max-width: none;
    margin: 0;
    padding: 8px 8px max(10px, env(safe-area-inset-bottom));
    border-radius: 22px;
    border: 1px solid color-mix(in srgb, var(--site-line) 72%, transparent);
    background: var(--site-paper);
    box-shadow: 0 -8px 30px color-mix(in srgb, var(--site-ink) 13%, transparent);
  }
  .ws-composer.is-dragging {
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--site-accent) 55%, transparent);
  }
  .ws-composer-box {
    padding: 0 2px;
    border: 0;
    border-radius: 0;
    background: transparent;
    box-shadow: none;
    backdrop-filter: none;
  }
  .ws-composer-prompt textarea {
    min-height: 60px;
    max-height: 240px;
    padding: 4px 2px 6px;
    font-size: 13px;
  }
  .ws-composer-bar {
    gap: 4px;
    margin-top: 4px;
    padding-top: 7px;
  }
  .ws-send-btn {
    width: 100%;
    min-width: 0;
    height: 44px;
    padding: 0 8px;
    font-size: 12.5px;
  }
  .ws-composer-footer {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
    padding-top: 4px;
  }
  .ws-composer-tip {
    display: none;
  }
  .ws-composer-hint {
    display: none;
  }
  .ws-composer-secondary {
    display: contents;
  }
  .ws-composer-secondary button {
    min-height: 44px;
    padding: 0 8px;
    border-radius: 12px;
    background: var(--site-soft);
    color: var(--site-ink);
    font-size: 12px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .ws-composer-box,
  .ws-send-btn {
    transition: none;
  }
  .ws-canvas-image,
  .ws-view-thumb {
    animation: none;
    transition: none;
  }
}
</style>
