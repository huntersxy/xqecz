<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount, nextTick } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { useAIStudio } from '@/composables/useAIStudio'
import { useStudioDock } from '@/composables/useStudioDock'
import { aiApi } from '@/api/creative'
import { contentApi } from '@/api'
import type { AIAsset, AITaskView } from '@/types/schemas'
import {
  IconImage,
  IconPlus,
  IconSettings,
  IconRefresh,
  IconSend,
  IconExpand,
  IconFile,
  IconLoading,
  IconDelete,
} from '@arco-design/web-vue/es/icon'
import { useConfirm } from '@/composables/useToast'
import CreativeImage from '@/components/CreativeImage.vue'
import CreativeLibrary from '@/components/CreativeLibrary.vue'
import StudioReferences from '@/components/StudioReferences.vue'
import StudioSizePicker from '@/components/StudioSizePicker.vue'
import AutoMetadataOption from '@/components/AutoMetadataOption.vue'
import '@/assets/creative-pages.css'
import { referenceMentionAt, remapReferenceMentions } from '@/utils/creativeMentions'
import { formatImageSize, normalizeImageSize, parseImageSize, AUTO_IMAGE_SIZE } from '@/utils/imageSize'
const studio = useAIStudio(),
  router = useRouter(),
  route = useRoute(),
  { confirm } = useConfirm()
const {
  tasks,
  assets,
  account,
  sessionId,
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
  pending,
  selectedAsset,
  archiveAsset,
  enhancePrompt,
  promptEnhancement,
} = studio
const libraryOpen = ref(false),
  historyOpen = ref(false),
  ideaOpen = ref(false),
  publishOpen = ref(false),
  resultOpen = ref(false),
  sizePickerOpen = ref(false),
  promptExpanded = ref(false),
  mobileSettingsOpen = ref(false),
  composerCollapsed = ref(false),
  uploading = ref(false),
  confirming = ref(false),
  dragging = ref(false),
  zoom = ref(1),
  search = ref(''),
  status = ref('')
const promptArea = ref<HTMLTextAreaElement>(),
  fileInput = ref<HTMLInputElement>()
const composerElement = ref<HTMLElement>()
const dockStyle = useStudioDock(composerElement)
function toggleMobileSettings() {
  mobileSettingsOpen.value = !mobileSettingsOpen.value
  composerCollapsed.value = false
}
const previewReference = ref(0)
const awaitingTask = ref(0)
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
const quotaNote = computed(() =>
  `本次 ${cost.value} 额度 · 可用 ${account.value.available}`,
)
const dailyCreditTotal = computed(
  () => (account.value.daily_available ?? 0) + (account.value.daily_reserved ?? 0),
)
const otherCreditTotal = computed(() =>
  Math.max(0, account.value.available + account.value.reserved - dailyCreditTotal.value),
)
const quotaVisible = computed(() => !studio.isQuotaExempt.value)
const mention = ref<ReturnType<typeof referenceMentionAt>>(null),
  mentionIndex = ref(0)
const mentionChoices = computed(() =>
  references.value
    .map((reference, index) => ({ ...reference, index, label: `@图片${index + 1}` }))
    .filter(
      (item) => !mention.value?.query || String(item.index + 1).startsWith(mention.value.query),
    ),
)
watch(sessionId, () => {
  mention.value = null
  awaitingTask.value = 0
  resultOpen.value = false
})
watch([tasks, awaitingTask], () => {
  const row = tasks.value.find((row) => row.task.id === awaitingTask.value)
  if (row?.assets[0]) {
    selectedAsset.value = row.assets[0]
    awaitingTask.value = 0
  } else if (row && ['failed', 'cancelled', 'unknown'].includes(row.task.status)) {
    awaitingTask.value = 0
  }
})
function onPromptInput() {
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
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
    event.preventDefault()
    if (canGenerate.value) void generate()
    return
  }
  if (!mention.value || !mentionChoices.value.length || event.isComposing) return
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
  const before = references.value.map((r) => r.asset_id)
  studio.moveReference(index, direction)
  prompt.value = remapReferenceMentions(
    prompt.value,
    before,
    references.value.map((r) => r.asset_id),
  )
  mention.value = null
}
async function removeReference(index: number) {
  if (composerLocked.value) return
  const before = references.value.map((r) => r.asset_id)
  if (
    prompt.value.includes(`@图片${index + 1}`) &&
    !(await confirm(`移除 @图片${index + 1}？提示词中的对应引用也会移除。`))
  )
    return
  studio.removeReference(index)
  prompt.value = remapReferenceMentions(
    prompt.value,
    before,
    references.value.map((r) => r.asset_id),
  )
  mention.value = null
}
const activeTask = computed(() =>
  tasks.value.find(
    (row) => ['queued', 'running', 'unknown'].includes(row.task.status),
  ),
)
const proposal = ref({ title: '', description: '', public_consent: false, reference_rights: false })
const publishForm = ref({ title: '', description: '', tags: [] as string[], auto_metadata: true, public_consent: false })
const publishMetadata = ref({ available: false, ready: false, placeholder_title: '待命名作品', placeholder_tag: '待生成标签' })
const labels: Record<string, string> = {
  queued: '排队中',
  running: '生成中',
  succeeded: '已完成',
  failed: '失败',
  cancelled: '已取消',
  unknown: '结果待核查',
}
const shownTasks = computed(() =>
  tasks.value.filter(
    (t) =>
      (!search.value || t.task.prompt.includes(search.value)) &&
      (!status.value || t.task.status === status.value),
  ),
)
const failedTaskRows = computed(() =>
  shownTasks.value.filter((row) => ['failed', 'cancelled'].includes(row.task.status)),
)
function clearFilters() {
  search.value = ''
  status.value = ''
}
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
onBeforeUnmount(() => {
  if (historyTimer) clearTimeout(historyTimer)
})
onBeforeRouteLeave(async () => {
  try {
    await studio.save()
    return true
  } catch {
    return false
  }
})
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
async function focusPrompt() {
  composerCollapsed.value = false
  await nextTick()
  promptArea.value?.focus()
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
async function removeLibraryAsset(asset: AIAsset) {
  if (!(await confirm(`移除图片“${asset.title || `图片 ${asset.id}`}”？`))) return
  await studio.run(() => archiveAsset(asset.id))
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
async function archiveTask(row: AITaskView) {
  if (!['failed', 'cancelled'].includes(row.task.status)) return
  if (!(await confirm(`删除任务 #${row.task.id}？失败记录会从工作台移除，已生成的素材仍保留。`)))
    return
  await studio.run(async () => {
    await aiApi.archiveTask(row.task.id)
    await studio.refresh()
  })
}
async function archiveFailedTasks() {
  const rows = failedTaskRows.value
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
    await studio.refresh()
  })
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
      new Set([...references.value.map((r) => r.asset_id), ...value.assets]).size >
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
    libraryOpen.value = false
  })
}
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
  publishForm.value = { title: '', description: '', tags: [], auto_metadata: true, public_consent: false }
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
function selectResult(asset: AIAsset) {
  awaitingTask.value = 0
  selectedAsset.value = asset
  zoom.value = 1
  resultOpen.value = true
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
  <div class="creative-page creative-theme ai-studio" :style="dockStyle">
    <header class="studio-head">
      <div class="studio-identity">
        <div class="studio-identity-copy">
          <span class="studio-kicker">AI CREATIVE STUDIO</span>
          <h1>创作工作台</h1>
          <p class="studio-lede">把一个画面写下来，参考图、生成结果与每次尝试都会留在这里。</p>
        </div>
        <div v-if="ready" class="studio-credit-summary" aria-label="积分来源">
          <span class="studio-credit-item" title="每日登录后获得一点积分，上限5点">
            每日登录 <strong>{{ dailyCreditTotal }}/5</strong>
          </span>
          <span class="studio-credit-item" title="上传作品通过审核后奖励1点积分；其他奖励以账本记录为准">
            其他积分 <strong>{{ otherCreditTotal }}</strong>
          </span>
          <span class="studio-credit-item" title="当前可用积分；生成中的额度会暂时计入预留">
            可用 <strong>{{ account.available }}</strong>
          </span>
        </div>
      </div>
      <nav class="studio-head-nav" aria-label="工作台导航">
        <RouterLink to="/">作品广场</RouterLink>
        <RouterLink to="/ideas">创意接力</RouterLink>
      </nav>
    </header>
    <div class="studio-workspace">
      <section class="studio-gallery" aria-label="作品与任务记录">
        <div class="studio-gallery-toolbar">
          <div class="studio-gallery-heading">
            <div>
              <span class="studio-section-kicker">YOUR WORK</span>
              <h2>我的生成</h2>
            </div>
            <a-button
              v-if="failedTaskRows.length"
              size="small"
              status="danger"
              aria-label="清除失败任务"
              @click="archiveFailedTasks"
              ><IconDelete />清除失败 {{ failedTaskRows.length }}</a-button
            >
          </div>
          <div class="studio-gallery-filters">
            <a-input
              v-model="search"
              allow-clear
              placeholder="搜索提示词"
              :input-attrs="{ 'aria-label': '搜索历史提示词' }"
            />
            <select v-model="status" aria-label="任务状态">
              <option value="">全部状态</option>
              <option v-for="(label, key) in labels" :key="key" :value="key">{{ label }}</option>
            </select>
          </div>
        </div>
        <div class="studio-gallery-scroll" tabindex="0" aria-label="滚动浏览任务">
          <div v-if="!ready && !error" class="studio-gallery-loading" role="status">
            <IconLoading />正在恢复你的创作…
          </div>
          <div v-else-if="!ready" class="studio-gallery-empty">
            <h2>工作台暂时无法恢复</h2>
            <p>请检查连接后重试，已保存的创作仍会保留。</p>
            <a-button @click="router.go(0)">重试加载</a-button>
          </div>
          <div v-else-if="!shownTasks.length" class="studio-gallery-empty">
            <IconImage aria-hidden="true" />
            <h2>{{ search || status ? '没有匹配的任务' : '你的下一幅作品，从这里开始' }}</h2>
            <p>
              {{
                search || status
                  ? '调整搜索词或状态，找回之前的创作。'
                  : '写下画面，或添加一张参考图。每次尝试都会留在这里。'
              }}
            </p>
            <span v-if="!search && !status" class="creative-note"
              >结果默认仅自己可见 · 离开页面后任务会继续</span
            >
            <a-button v-if="search || status" @click="clearFilters">清除筛选</a-button>
            <a-button v-else @click="focusPrompt"><IconPlus />开始创作</a-button>
          </div>
          <div v-else class="studio-task-grid">
            <article
              v-for="row in shownTasks"
              :key="row.task.id"
              class="studio-task-card"
              :class="{ 'is-selected': row.assets.some((asset) => asset.id === selectedAsset?.id) }"
            >
              <div
                v-if="row.assets.length"
                class="task-images"
                :class="{ 'has-multiple': row.assets.length > 1 }"
              >
                <button
                  v-for="(asset, index) in row.assets"
                  :key="asset.id"
                  type="button"
                  :aria-label="`查看任务 ${row.task.id} 的结果 ${index + 1}`"
                  :aria-pressed="selectedAsset?.id === asset.id"
                  @click="selectResult(asset)"
                >
                  <CreativeImage :id="asset.id" /><span class="task-image-open"
                    ><IconExpand
                  /></span>
                </button>
              </div>
              <div v-else class="task-placeholder" :data-status="row.task.status">
                <IconLoading
                  v-if="['queued', 'running'].includes(row.task.status)"
                  class="task-loading"
                /><IconImage v-else />
                <span>{{ labels[row.task.status] }}</span>
              </div>
              <div class="task-body">
                <div class="task-meta">
                  <span>#{{ row.task.id }}</span
                  ><span class="task-status" :data-status="row.task.status">{{
                    labels[row.task.status]
                  }}</span>
                </div>
                <p class="task-prompt" :title="row.task.prompt">{{ row.task.prompt }}</p>
                <span class="task-parameters"
                  >{{ row.task.model }} · {{ formatImageSize(row.task.size) }} ·
                  {{ row.task.count }} 张</span
                >
                <span v-if="row.task.parent_task_id" class="creative-note"
                  >继续自 #{{ row.task.parent_task_id }}</span
                >
                <p v-if="row.task.error" class="task-error" :title="row.task.error">
                  {{ row.task.error }}
                </p>
                <p v-if="row.task.status === 'unknown'" class="task-error">
                  结果待核查，预留额度待管理员确认。
                </p>
                <div class="task-actions">
                  <a-button size="small" :disabled="composerLocked" @click="reusePrompt(row)"
                    ><IconRefresh />复用提示词</a-button
                  >
                  <a-button
                    v-if="row.task.status === 'succeeded' && row.assets.length"
                    size="small"
                    :disabled="composerLocked"
                    @click="continueFromTask(row)"
                    >继续编辑</a-button
                  >
                  <a-button
                    v-if="row.task.status === 'queued'"
                    size="small"
                    @click="
                      studio.run(async () => {
                        await aiApi.cancel(row.task.id)
                        await studio.refresh()
                      })
                    "
                    >取消排队</a-button
                  >
                  <a-button
                    v-if="['failed', 'cancelled'].includes(row.task.status)"
                    size="small"
                    status="danger"
                    @click="archiveTask(row)"
                    >删除记录</a-button
                  >
                  <RouterLink v-if="row.task.idea_id" :to="`/ideas/${row.task.idea_id}`"
                    >赞助创意</RouterLink
                  >
                </div>
              </div>
            </article>
          </div>
          <a-button
            v-if="studio.moreTasks.value"
            class="studio-load-more"
            @click="studio.run(studio.loadMoreTasks)"
            >加载更早的任务</a-button
          >
        </div>
      </section>
      <section
        ref="composerElement"
        class="studio-composer"
        :class="{
          'is-dragging': dragging,
          'is-expanded': promptExpanded,
          'is-collapsed': composerCollapsed,
        }"
        aria-label="创作输入区"
        @dragover.prevent="dragging = !composerLocked && !!provider?.max_references"
        @dragleave.self="dragging = false"
        @drop.prevent="dropImages"
      >
        <div class="composer-heading">
          <span class="studio-section-kicker">NEW PIECE</span>
          <h2>构想一张新作品</h2>
          <p>输入画面描述，或从参考图开始。</p>
        </div>
        <div class="composer-mobile-bar">
          <span class="composer-model-summary"
            >{{ provider?.name || '创作输入' }} · {{ size }}</span
          >
          <button
            type="button"
            aria-controls="studio-generation-controls"
            :aria-expanded="mobileSettingsOpen && !composerCollapsed"
            @click="toggleMobileSettings"
          >
            <IconSettings />生成设置
          </button>
          <button
            type="button"
            aria-controls="studio-composer-fields"
            :aria-expanded="!composerCollapsed"
            :aria-label="composerCollapsed ? '展开创作输入区' : '收起创作输入区'"
            @click="composerCollapsed = !composerCollapsed"
          >
            {{ composerCollapsed ? '展开' : '收起' }}
          </button>
        </div>
        <div v-if="error" class="creative-error composer-error" role="alert">{{ error }}</div>
        <div v-if="activeTask" class="composer-task-note" role="status">
          <IconLoading v-if="activeTask.task.status !== 'unknown'" class="task-loading" />任务 #{{
            activeTask.task.id
          }}
          · {{ labels[activeTask.task.status]
          }}{{
            activeTask.task.status === 'unknown' ? '，预留额度待管理员核查' : '，离开页面仍会继续'
          }}
        </div>
        <fieldset id="studio-composer-fields" :disabled="composerLocked">
          <StudioReferences
            :references="references"
            :disabled="composerLocked"
            @preview="(id) => (previewReference = id)"
            @mention="insertReferenceMention"
            @remove="removeReference"
            @reorder="(from, to) => moveReference(from, to - from)"
          />
          <div v-if="studio.parentAssetId.value" class="composer-context">
            <span
              >继续编辑 @图片{{
                references.findIndex(
                  (reference) => reference.asset_id === studio.parentAssetId.value,
                ) + 1
              }}</span
            >
          </div>
          <div class="composer-prompt">
            <label class="sr-only" for="studio-prompt">创作提示词</label>
            <textarea
              id="studio-prompt"
              ref="promptArea"
              v-model="prompt"
              :rows="promptExpanded ? 8 : 3"
              maxlength="8000"
              placeholder="描述你想创作的画面… 输入 @ 引用图片"
              aria-label="创作提示词"
              @input="onPromptInput"
              @click="onPromptInput"
              @keydown="onPromptKeydown"
              @paste="pasteImages"
            />
            <button
              class="prompt-expand"
              type="button"
              :aria-expanded="promptExpanded"
              :aria-label="promptExpanded ? '收起提示词' : '展开提示词'"
              @click="promptExpanded = !promptExpanded"
            >
              <IconExpand />
            </button>
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
                <span>{{ item.label }}</span
                ><span class="creative-note">插入提示词</span>
              </button>
            </div>
          </div>
          <div class="composer-attachments">
            <a-button
              size="small"
              :disabled="
                !provider?.max_references || references.length >= (provider?.max_references ?? 0)
              "
              @click="fileInput?.click()"
              ><IconPlus />参考图<span
                >{{ references.length }}/{{ provider?.max_references ?? 0 }}</span
              ></a-button
            >
            <a-button size="small" :disabled="!provider?.max_references" @click="historyOpen = true"
              ><IconImage />图片库</a-button
            >
            <a-button size="small" @click="libraryOpen = true"><IconFile />角色 / 模板</a-button>
            <span class="attachment-hint creative-note">{{
              uploading
                ? '正在上传参考图…'
                : provider?.max_references
                  ? '可拖入或粘贴图片 · PNG / JPEG / WebP'
                  : '当前模型仅支持文生图'
            }}</span>
          </div>
          <input
            ref="fileInput"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            multiple
            hidden
            @change="(event) => uploadFiles((event.target as HTMLInputElement).files)"
          />
          <div
            id="studio-generation-controls"
            class="composer-controls"
            :class="{ 'is-mobile-open': mobileSettingsOpen }"
          >
            <label class="composer-control composer-model"
              ><span>模型</span
              ><select v-model="providerId" aria-label="供应商与模型">
                <option v-if="!availableProviders.length" :value="0">暂无可用模型</option>
                <option v-for="p in availableProviders" :key="p.id" :value="p.id">
                  {{ p.name }} · {{ p.model }}
                </option>
              </select></label
            >
            <div class="composer-control composer-size-control">
              <span>尺寸</span
              ><button
                type="button"
                class="studio-size-trigger"
                aria-label="画面尺寸"
                :disabled="composerLocked"
                @click="sizePickerOpen = true"
              >
                {{ formatImageSize(size) }}
              </button>
            </div>
            <label v-if="provider?.qualities.length" class="composer-control"
              ><span>画质</span
              ><select v-model="quality" aria-label="画质">
                <option value="">默认</option>
                <option v-for="q in provider.qualities" :key="q">{{ q }}</option>
              </select></label
            >
            <label class="composer-control composer-count"
              ><span>数量</span
              ><select v-model.number="count" aria-label="输出数量">
                <option v-for="n in provider?.max_count ?? 1" :key="n" :value="n">
                  {{ n }} 张
                </option>
              </select></label
            >
            <label class="composer-control composer-enhancement"
              ><span>提示词增强</span
              ><span class="prompt-enhancement-toggle" :title="promptEnhancement.prompt || '暂无增强提示词'">
                <input v-model="enhancePrompt" type="checkbox" :disabled="!promptEnhancement.enabled || !promptEnhancement.prompt" />
                <strong>{{ enhancePrompt && promptEnhancement.enabled ? '已开启' : '未开启' }}</strong>
              </span>
            </label>
          </div>
        </fieldset>
        <div class="composer-footer">
          <div class="composer-feedback">
            <span v-if="!provider" class="creative-note">管理员尚未启用生图供应商</span>
            <span v-else-if="!provider.has_key" class="creative-note"
              >此模型尚未配置站点密钥，请联系管理员。</span
            >
            <span v-else-if="quotaVisible" class="creative-note"
              >{{ quotaNote }}<span class="composer-shortcut"> · Ctrl / ⌘ + Enter 生成</span></span
            >
            <span v-else class="creative-note composer-shortcut">Ctrl / ⌘ + Enter 生成</span>
            <span v-if="pending" class="creative-note"
              >上次提交结果尚未确认，草稿已锁定。请用原请求继续确认。</span
            >
          </div>
          <div class="composer-quick-actions">
            <a-button size="small" :disabled="composerLocked" @click="studio.run(studio.save)">保存草稿</a-button>
            <a-button size="small" :disabled="composerLocked || !prompt.trim()" @click="ideaOpen = true">投稿创意</a-button>
          <a-button
            type="primary"
            class="composer-submit"
            :loading="busy || uploading"
            :disabled="!canGenerate"
            @click="generate"
            ><IconSend v-if="!busy && !uploading" />{{
              pending ? '确认上次提交' : '开始生成'
            }}</a-button
          >
          </div>
        </div>
      </section>
    </div>
    <StudioSizePicker
      v-model:visible="sizePickerOpen"
      :model-value="size"
      @select="size = $event"
    />
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
            ><a-button @click="zoom = 1">{{
              zoom === 1 ? '适应画布' : `${Math.round(zoom * 100)}%`
            }}</a-button
            ><a-button aria-label="放大结果" @click="zoom = Math.min(3, zoom + 0.25)"
              ><IconPlus
            /></a-button>
          </div>
        </div>
        <div class="result-preview-scroll">
          <div
            class="result-preview-image"
            :style="{
              width: `${zoom * 100}%`,
              height: `${zoom * 50}dvh`,
              marginInline: zoom < 1 ? 'auto' : '0',
            }"
          >
            <CreativeImage :id="selectedAsset.id" full />
          </div>
        </div>
        <div class="result-actions">
          <a-button
            type="primary"
            :disabled="!provider?.max_references || composerLocked"
            @click="useResult(true)"
            >继续编辑</a-button
          ><a-button
            :disabled="!provider?.max_references || composerLocked"
            @click="useResult(false)"
            >加入参考图</a-button
          ><a-button @click="download">下载原图</a-button
          ><a-button @click="openPublish">发布到站内</a-button
          ><a-button type="text" status="danger" @click="removeResult">移除结果</a-button>
        </div>
      </template>
    </a-modal>
    <a-modal
      :visible="!!previewReference"
      class="creative-theme creative-dialog ai-dialog"
      title="参考图预览"
      :footer="false"
      @cancel="previewReference = 0"
      ><div class="reference-large">
        <CreativeImage v-if="previewReference" :id="previewReference" full /></div
    ></a-modal>
    <a-modal
      v-model:visible="libraryOpen"
      class="creative-theme creative-dialog ai-dialog"
      title="创作资源"
      :width="900"
      :footer="false"
      ><CreativeLibrary v-if="libraryOpen" @apply="applyResource"
    /></a-modal>
    <a-modal
      v-model:visible="historyOpen"
      class="creative-theme creative-dialog ai-dialog"
      title="个人常用参考图"
      :footer="false"
      ><div class="creative-grid">
        <div v-for="asset in assets" :key="asset.id" class="library-asset">
          <button
            type="button"
            class="library-asset-select"
            @click="
              studio.run(async () => {
                studio.addReference(asset.id)
                historyOpen = false
              })
            "
          >
            <CreativeImage :id="asset.id" /><span>{{ asset.title || `图片 ${asset.id}` }}</span>
          </button>
          <button type="button" class="library-asset-delete" :disabled="composerLocked" @click="removeLibraryAsset(asset)">
            删除
          </button>
        </div>
      </div>
      <a-button v-if="studio.moreAssets.value" @click="studio.run(studio.loadMoreAssets)"
        >加载更早的图片</a-button
      >
      <p v-if="!assets.length">上传的参考图与生成结果会保存在这里。</p></a-modal
    >
    <a-modal
      v-model:visible="ideaOpen"
      class="creative-theme creative-dialog ai-dialog"
      title="投稿到创意广场"
      :footer="false"
      ><p>
        投稿将公开标题、描述、完整提示词、选中的参考图、模型与生成参数。赞助成功的结果会公开展示。投稿本身不消耗额度。
      </p>
      <label class="creative-field"
        >标题<a-input v-model="proposal.title" :max-length="150" /></label
      ><label class="creative-field">描述<a-textarea v-model="proposal.description" /></label
      ><a-checkbox v-model="proposal.public_consent">我同意上述公开范围和成果展示规则</a-checkbox
      ><br /><a-checkbox v-model="proposal.reference_rights"
        >我确认选中参考图有公开展示授权</a-checkbox
      >
      <p class="creative-note">赞助额度不代表取得作品著作权。</p>
      <a-button
        type="primary"
        :disabled="!proposal.public_consent || !proposal.reference_rights"
        @click="submitIdea"
        >公开投稿</a-button
      ></a-modal
    >
    <a-modal
      v-model:visible="publishOpen"
      class="creative-theme creative-dialog ai-dialog"
      title="发布生成作品"
      :footer="false"
      ><AutoMetadataOption v-model="publishForm.auto_metadata" :options="publishMetadata" :can-generate="true" />
      <label class="creative-field"
        >作品标题<a-input :model-value="publishForm.auto_metadata ? publishMetadata.placeholder_title : publishForm.title" aria-label="作品标题" :disabled="publishForm.auto_metadata" @update:model-value="publishForm.title = $event" /></label
      ><label class="creative-field"
        >作品描述<a-textarea v-model="publishForm.description" aria-label="作品描述" /></label
      ><label class="creative-field"
        >标签<a-input-tag :model-value="publishForm.auto_metadata ? [publishMetadata.placeholder_tag] : publishForm.tags" aria-label="作品标签" placeholder="输入标签后按回车" :disabled="publishForm.auto_metadata" @update:model-value="publishForm.tags = $event.filter((tag): tag is string => typeof tag === 'string')" /></label
      ><a-checkbox v-model="publishForm.public_consent"
        >将此结果公开发布到站内，遵循现有作品审核与授权规则</a-checkbox
      >
      <p class="creative-note">同一结果只关联一条站内内容。</p>
      <a-button type="primary" :disabled="!publishForm.public_consent" @click="publish"
        >确认发布</a-button
      ></a-modal
    >
  </div>
</template>

<style scoped>
/*
 * The studio keeps the same warm editorial language as the gallery home:
 * a quiet canvas, cream paper surfaces and restrained wine accents. The
 * rules live with this view so the generation contract and other creative
 * pages keep their existing layout and interaction affordances.
 */
.ai-studio {
  --studio-canvas: #ebe4e1;
  --studio-paper: #fffdf9;
  --studio-ink: #201b1c;
  --studio-muted: #817779;
  --studio-line: rgba(52, 40, 42, 0.14);
  --studio-soft: #f1e9e6;
  --studio-accent: #6b173b;
  --studio-accent-soft: #efe1e5;
  position: relative;
  isolation: isolate;
  width: 100%;
  max-width: none;
  min-height: calc(100dvh - 72px);
  margin: 0;
  padding: clamp(28px, 4vw, 58px) clamp(20px, 5vw, 76px) 64px;
  overflow: hidden;
  color: var(--studio-ink);
  background:
    radial-gradient(circle at 80% -10%, color-mix(in srgb, var(--studio-paper) 72%, transparent), transparent 35rem),
    var(--studio-canvas);
}
.ai-studio::before {
  position: absolute;
  z-index: -1;
  inset: 0;
  pointer-events: none;
  content: '';
  opacity: 0.28;
  background-image: radial-gradient(circle, color-mix(in srgb, var(--studio-muted) 20%, transparent) 0.6px, transparent 0.8px);
  background-size: 7px 7px;
  mask-image: linear-gradient(to bottom, black, transparent 76%);
}
body[arco-theme='dark'] .ai-studio {
  --studio-canvas: #211b1e;
  --studio-paper: #2b2427;
  --studio-ink: #f8f0ed;
  --studio-muted: #c0b1b4;
  --studio-line: rgba(250, 240, 237, 0.16);
  --studio-soft: #342b2f;
  --studio-accent: #f0a8bb;
  --studio-accent-soft: #4a303a;
}
.studio-head,
.studio-workspace {
  position: relative;
  z-index: 1;
  width: min(100%, 1280px);
  margin-inline: auto;
}
.studio-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 28px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--studio-line);
}
.studio-identity {
  display: flex;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 22px 30px;
  min-width: 0;
}
.studio-identity-copy {
  min-width: min(100%, 320px);
}
.studio-kicker,
.studio-section-kicker {
  display: block;
  color: var(--studio-muted);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.17em;
  line-height: 1.2;
}
.ai-studio .studio-head h1 {
  margin: 8px 0 0;
  color: var(--studio-ink);
  font: 600 clamp(28px, 4vw, 43px)/1.05 Georgia, 'Songti SC', 'SimSun', serif;
  letter-spacing: -0.04em;
}
.studio-lede {
  max-width: 42em;
  margin: 12px 0 0;
  color: var(--studio-muted);
  font-size: 12px;
  line-height: 1.7;
}
.studio-credit-summary {
  gap: 7px;
}
.studio-credit-item {
  border-color: var(--studio-line);
  border-radius: 4px;
  background: color-mix(in srgb, var(--studio-paper) 84%, transparent);
  color: var(--studio-muted);
}
.studio-credit-item strong {
  color: var(--studio-ink);
}
.studio-head-nav {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 18px;
  padding-bottom: 3px;
  font-size: 11px;
}
.studio-head-nav a {
  color: var(--studio-muted);
  text-decoration: none;
  text-underline-offset: 4px;
}
.studio-head-nav a:hover,
.studio-head-nav a.router-link-active {
  color: var(--studio-accent);
  text-decoration: underline;
}
.studio-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(330px, 420px);
  align-items: start;
  gap: clamp(22px, 4vw, 58px);
  height: auto;
  min-height: 0;
  margin-top: 26px;
}
.studio-gallery {
  min-width: 0;
  min-height: 480px;
}
.studio-gallery-toolbar {
  align-items: flex-end;
  justify-content: space-between;
  gap: 18px;
  padding: 0 0 15px;
  border-bottom: 1px solid var(--studio-line);
}
.studio-gallery-heading {
  align-items: flex-end;
  gap: 14px;
}
.ai-studio .studio-gallery-toolbar h2 {
  margin: 5px 0 0;
  color: var(--studio-ink);
  font: 600 21px/1.2 Georgia, 'Songti SC', 'SimSun', serif;
  letter-spacing: -0.025em;
}
.studio-gallery-filters {
  align-items: center;
  flex-wrap: nowrap;
  margin-left: auto;
}
.studio-gallery-filters .arco-input-wrapper {
  width: 210px;
  border-color: var(--studio-line);
  border-radius: 4px;
  background: color-mix(in srgb, var(--studio-paper) 72%, transparent);
}
.ai-studio select {
  border-color: var(--studio-line);
  border-radius: 4px;
  background: var(--studio-paper);
  color: var(--studio-ink);
}
.studio-gallery-scroll {
  overflow: visible;
  padding: 22px 2px 18px;
}
.studio-task-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-auto-rows: auto;
  gap: 20px;
}
.ai-studio .studio-task-card {
  display: flex;
  flex-direction: column;
  height: auto;
  min-height: 0;
  border: 1px solid var(--studio-line);
  border-radius: 5px;
  background: var(--studio-paper);
  box-shadow: 0 8px 22px color-mix(in srgb, var(--studio-ink) 5%, transparent);
}
.ai-studio .studio-task-card.is-selected {
  border-color: var(--studio-accent);
  box-shadow: 0 0 0 1px var(--studio-accent), 0 8px 22px color-mix(in srgb, var(--studio-accent) 12%, transparent);
}
.task-images {
  aspect-ratio: 1.18;
  min-height: 0;
  flex: none;
  background: var(--studio-soft);
}
.task-images button,
.task-images.has-multiple button {
  min-height: 0;
}
.task-images .creative-image img {
  object-fit: cover;
}
.task-image-open {
  right: 10px;
  bottom: 10px;
  width: 27px;
  height: 27px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--studio-paper) 90%, transparent);
  color: var(--studio-ink);
}
.task-placeholder {
  aspect-ratio: 1.18;
  flex: none;
  padding: 20px;
  background: var(--studio-soft);
  color: var(--studio-muted);
}
.task-placeholder[data-status='running'] {
  background: var(--studio-accent-soft);
  color: var(--studio-accent);
}
.task-body {
  min-height: 154px;
  gap: 8px;
  padding: 15px 16px 16px;
}
.task-meta {
  font-size: 10px;
}
.task-status {
  color: var(--studio-accent);
}
.ai-studio .task-prompt {
  font-size: 13px;
  line-height: 1.55;
}
.task-parameters {
  color: var(--studio-muted);
  font-size: 10px;
}
.task-actions {
  gap: 5px;
  padding-top: 6px;
}
.task-actions .arco-btn {
  border-radius: 3px;
  color: var(--studio-muted);
  font-size: 11px;
}
.task-actions .arco-btn:hover {
  color: var(--studio-accent);
}
.studio-load-more {
  margin-top: 24px;
  border: 0;
  border-radius: 0;
  color: var(--studio-accent);
  background: transparent;
  font-size: 11px;
}
.studio-gallery-empty,
.studio-gallery-loading {
  min-height: 360px;
  color: var(--studio-muted);
}
.ai-studio .studio-gallery-empty h2 {
  color: var(--studio-ink);
  font: 600 23px/1.2 Georgia, 'Songti SC', 'SimSun', serif;
}
.studio-gallery-empty > .arco-btn {
  border-radius: 4px;
}
.studio-composer {
  position: sticky;
  top: 92px;
  width: auto;
  max-height: none;
  margin: 0;
  padding: 24px;
  overflow: visible;
  border: 1px solid var(--studio-line);
  border-radius: 5px;
  background: var(--studio-paper);
  box-shadow: 0 12px 30px color-mix(in srgb, var(--studio-ink) 6%, transparent);
}
.studio-composer.is-dragging {
  background: var(--studio-accent-soft);
}
.composer-heading {
  margin-bottom: 18px;
  padding-bottom: 17px;
  border-bottom: 1px solid var(--studio-line);
}
.composer-heading h2 {
  margin: 6px 0 0;
  color: var(--studio-ink);
  font: 600 20px/1.2 Georgia, 'Songti SC', 'SimSun', serif;
  letter-spacing: -0.02em;
}
.composer-heading p {
  margin: 8px 0 0;
  color: var(--studio-muted);
  font-size: 11px;
  line-height: 1.6;
}
.composer-prompt textarea {
  min-height: 112px;
  padding: 2px 36px 8px 0;
  color: var(--studio-ink);
  font-size: 14px;
  line-height: 1.7;
}
.composer-prompt:focus-within {
  box-shadow: 0 1px 0 var(--studio-accent);
}
.composer-attachments {
  margin-block: 8px 18px;
}
.composer-attachments .arco-btn {
  border-radius: 3px;
  color: var(--studio-muted);
  font-size: 11px;
}
.composer-controls {
  gap: 7px;
  padding-top: 15px;
  border-top-color: var(--studio-line);
}
.composer-control {
  border-radius: 3px;
  background: var(--studio-canvas);
  font-size: 11px;
}
.studio-size-trigger {
  font-size: 11px;
}
.composer-footer {
  align-items: flex-end;
  gap: 12px;
  margin-top: 18px;
}
.composer-feedback .creative-note {
  color: var(--studio-muted);
  font-size: 10px;
  line-height: 1.5;
}
.composer-quick-actions {
  gap: 6px;
}
.composer-quick-actions .arco-btn {
  border-radius: 3px;
  font-size: 11px;
}
.composer-submit.arco-btn {
  min-width: 126px;
  height: 40px;
  border-radius: 3px;
  background: var(--studio-ink);
  border-color: var(--studio-ink);
  color: var(--studio-paper);
}
.composer-submit.arco-btn:hover {
  background: var(--studio-accent);
  border-color: var(--studio-accent);
}
.composer-submit.arco-btn:disabled {
  opacity: 0.45;
}
.reference-mentions {
  border-color: var(--studio-line);
  border-radius: 4px;
  background: var(--studio-paper);
}
.reference-mentions button {
  border-radius: 3px;
}
.ai-studio button:focus-visible,
.ai-studio select:focus-visible,
.ai-studio textarea:focus-visible,
.studio-gallery-scroll:focus-visible {
  outline-color: var(--studio-accent);
}

@media (max-width: 980px) {
  .ai-studio {
    padding-inline: clamp(18px, 4vw, 40px);
  }
  .studio-workspace {
    grid-template-columns: minmax(0, 1fr) minmax(280px, 340px);
    gap: 24px;
  }
  .studio-credit-summary {
    gap: 4px;
  }
  .studio-credit-item {
    padding-inline: 7px;
  }
}
@media (max-width: 800px) {
  .ai-studio {
    min-height: 100dvh;
    padding: 22px 16px calc(var(--studio-dock-height, 280px) + 28px);
  }
  .studio-head {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }
  .studio-identity {
    display: block;
  }
  .studio-identity-copy {
    min-width: 0;
  }
  .studio-lede {
    max-width: 34em;
  }
  .studio-credit-summary {
    margin-top: 15px;
  }
  .studio-head-nav {
    gap: 15px;
    padding: 0;
  }
  .studio-workspace {
    display: block;
    margin-top: 20px;
  }
  .studio-gallery {
    min-height: 0;
  }
  .studio-gallery-scroll {
    padding-top: 18px;
  }
  .studio-composer {
    position: fixed;
    z-index: 90;
    inset-inline: max(8px, env(safe-area-inset-left)) max(8px, env(safe-area-inset-right));
    bottom: var(--studio-dock-bottom, 0px);
    width: auto;
    max-height: min(60dvh, calc(var(--studio-visible-height, 100dvh) - 16px));
    padding: 8px 16px max(12px, env(safe-area-inset-bottom));
    border-radius: 5px 5px 0 0;
    box-shadow: 0 -8px 30px color-mix(in srgb, var(--studio-ink) 13%, transparent);
  }
  .composer-heading {
    display: none;
  }
  .studio-task-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }
}
@media (max-width: 540px) {
  .ai-studio {
    padding-inline: 12px;
  }
  .ai-studio .studio-head h1 {
    font-size: 31px;
  }
  .studio-lede {
    font-size: 11px;
  }
  .studio-gallery-toolbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 11px;
  }
  .studio-gallery-filters {
    width: 100%;
    margin-left: 0;
  }
  .studio-gallery-filters .arco-input-wrapper {
    flex: 1;
    min-width: 0;
    width: auto;
  }
  .studio-task-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .studio-task-card .task-images,
  .studio-task-card .task-placeholder {
    aspect-ratio: 1.35;
  }
  .studio-composer {
    padding-inline: 12px;
  }
}
</style>
