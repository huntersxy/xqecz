<script setup lang="ts">
function newKey() {
  return crypto.randomUUID()
}
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { useAIStudio } from '@/composables/useAIStudio'
import { aiApi } from '@/api/creative'
import type { AIAsset } from '@/types/schemas'
import { useConfirm } from '@/composables/useToast'
import CreativeImage from '@/components/CreativeImage.vue'
import CreativeLibrary from '@/components/CreativeLibrary.vue'
import '@/assets/creative-pages.css'
const studio = useAIStudio(),
  router = useRouter(),
  route = useRoute(),
  { confirm } = useConfirm()
const {
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
  provider,
  availableProviders,
  cost,
  error,
  saveState,
  busy,
  ready,
  pending,
  selectedAsset,
} = studio
const libraryOpen = ref(false),
  historyOpen = ref(false),
  keyOpen = ref(false),
  ideaOpen = ref(false),
  publishOpen = ref(false),
  leftOpen = ref(true),
  rightOpen = ref(true),
  mobileSection = ref('canvas'),
  zoom = ref(1),
  search = ref(''),
  status = ref('')
const promptArea = ref<HTMLTextAreaElement>(),
  fileInput = ref<HTMLInputElement>()
const previewReference = ref(0)
const generationMode = computed(() =>
  studio.parentAssetId.value ? 'edit' : references.value.length ? 'reference' : 'text',
)
const activeTask = computed(() =>
  tasks.value.find(
    (row) =>
      row.task.session_id === sessionId.value &&
      ['queued', 'running', 'unknown'].includes(row.task.status),
  ),
)
async function changeGenerationMode(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  await studio.run(async () => {
    if (value === 'text') {
      if (
        references.value.length &&
        !(await confirm('切换到文生图将清空当前参考图，历史图片仍保留。确认？'))
      )
        return
      references.value = []
      studio.parentAssetId.value = 0
    } else if (value === 'reference') {
      studio.parentAssetId.value = 0
      if (!references.value.length) historyOpen.value = true
    } else if (selectedAsset.value) studio.continueFrom(selectedAsset.value)
    else throw new Error('请先选择一张生成结果，再继续编辑')
  })
  ;(event.target as HTMLSelectElement).value = generationMode.value
}
const keyForm = ref({ name: '', allowed_provider_id: 0, api_key: '', default: true })
watch(keyOpen, (open) => {
  if (!open) return
  const approved = providers.value.filter((p) => p.owner === 0 && p.enabled)
  if (!approved.some((p) => p.id === keyForm.value.allowed_provider_id))
    keyForm.value.allowed_provider_id = approved[0]?.id ?? 0
})
const proposal = ref({ title: '', description: '', public_consent: false, reference_rights: false })
const publishForm = ref({ title: '', description: '', public_consent: false })
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
const historyScope = ref(route.query.history === 'all' ? 'all' : 'session')
let historyTimer: ReturnType<typeof setTimeout> | undefined
watch([ready, sessionId, search, status, historyScope], () => {
  if (!ready.value) return
  if (historyTimer) clearTimeout(historyTimer)
  historyTimer = setTimeout(
    () =>
      void studio.run(() =>
        studio.setHistoryQuery({
          ...(historyScope.value === 'session' ? { session_id: sessionId.value } : {}),
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
  await studio.run(async () => {
    if (
      !pending.value &&
      !(await confirm(
        `本次生成 ${count.value} 张图片，${mode.value === 'byok' ? '使用自备密钥' : `预留 ${cost.value} 额度`}。确认开始？`,
      ))
    )
      return
    await studio.generate()
  })
}
async function uploadFiles(files: FileList | null) {
  if (!files) return
  await studio.run(async () => {
    for (const file of Array.from(files)) await studio.upload(file)
  })
  if (fileInput.value) fileInput.value.value = ''
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
    if (
      typeof value.parameters.size === 'string' &&
      provider.value?.sizes.includes(value.parameters.size)
    )
      size.value = value.parameters.size
    if (
      typeof value.parameters.quality === 'string' &&
      provider.value?.qualities.includes(value.parameters.quality)
    )
      quality.value = value.parameters.quality
    libraryOpen.value = false
  })
}
async function saveKey() {
  await studio.run(async () => {
    await aiApi.saveBYOK(keyForm.value)
    keyForm.value.api_key = ''
    keyOpen.value = false
    await studio.loadProviders()
    mode.value = 'byok'
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
    )
    publishOpen.value = false
    await router.push(`/content/${row.content_id}`)
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
function selectResult(asset: AIAsset) {
  selectedAsset.value = asset
  zoom.value = 1
  mobileSection.value = 'canvas'
}
async function removeResult() {
  if (!selectedAsset.value || !(await confirm('从私人历史移除此结果？已有任务引用将继续保留。')))
    return
  await studio.run(async () => {
    const id = selectedAsset.value!.id
    await aiApi.archive(id)
    tasks.value = tasks.value.map((row) => ({
      ...row,
      assets: row.assets.filter((a) => a.id !== id),
    }))
    selectedAsset.value = null
    await studio.refresh()
  })
}
watch(ready, (value) => {
  if (!value) return
  const ids = String(route.query.assets ?? '')
    .split(',')
    .map(Number)
    .filter((n) => n > 0)
  if (ids.length)
    void studio.run(async () => {
      for (const id of ids) studio.addReference(id)
    })
})
</script>
<template>
  <div class="creative-page creative-theme ai-studio">
    <header class="creative-page-head studio-top">
      <div>
        <span class="creative-eyebrow">✧ 创作手记 / AI STUDIO</span>
        <h1>让想象，有了形状</h1>
        <p>从一段灵感开始，在这里慢慢打磨你的作品。</p>
      </div>
      <div class="creative-actions">
        <a-select
          :model-value="sessionId"
          aria-label="切换会话"
          @change="(value) => studio.run(() => studio.switchSession(Number(value)))"
          ><a-option v-for="row in sessions" :key="row.id" :value="row.id">{{
            row.name
          }}</a-option></a-select
        ><a-button @click="studio.run(studio.newSession)">新建会话</a-button
        ><a-button v-if="studio.moreSessions.value" @click="studio.run(studio.loadMoreSessions)"
          >更多会话</a-button
        ><a-button @click="keyOpen = true">API 设置</a-button
        ><RouterLink to="/characters">角色设定库</RouterLink
        ><RouterLink to="/ideas">创意广场</RouterLink>
      </div>
    </header>
    <div v-if="error" class="creative-error" role="alert">{{ error }}</div>
    <div class="studio-mobile-tabs">
      <a-radio-group v-model="mobileSection" type="button"
        ><a-radio value="parameters">参数</a-radio><a-radio value="canvas">画布</a-radio
        ><a-radio value="history">历史</a-radio></a-radio-group
      >
    </div>
    <div class="studio-toolbar creative-actions">
      <div class="studio-session">
        <a-input v-model="name" :input-attrs="{ 'aria-label': '创作会话名称' }" :disabled="!!pending" />
        <span class="creative-note" role="status">{{ saveState }}</span>
      </div>
      <a-button size="small" @click="leftOpen = !leftOpen"
        >{{ leftOpen ? '收起' : '展开' }}参数</a-button
      ><span class="creative-note">可用 {{ account.available }} · 预留 {{ account.reserved }}</span
      ><a-button size="small" @click="rightOpen = !rightOpen"
        >{{ rightOpen ? '收起' : '展开' }}历史</a-button
      >
    </div>
    <div
      class="studio-grid"
      :class="[`mobile-${mobileSection}`, { 'hide-left': !leftOpen, 'hide-right': !rightOpen }]"
    >
      <aside
        class="studio-parameters creative-box"
        @dragover.prevent
        @drop.prevent="(event) => uploadFiles(event.dataTransfer?.files ?? null)"
      >
        <h2 class="studio-panel-title"><span aria-hidden="true">01</span> 创作参数</h2>
        <fieldset :disabled="!!pending || busy">
          <label class="creative-field"
            >生成方式<select :value="generationMode" @change="changeGenerationMode">
              <option value="text">文生图</option>
              <option value="reference" :disabled="!provider?.max_references">参考生成</option>
              <option value="edit" :disabled="!provider?.max_references || !selectedAsset">
                继续编辑
              </option>
            </select></label
          >
          <label class="creative-field"
            >额度模式<select v-model="mode">
              <option value="site">站点额度</option>
              <option value="byok">自备 API Key</option>
            </select></label
          ><label class="creative-field"
            >供应商<select v-model="providerId">
              <option v-for="p in availableProviders" :key="p.id" :value="p.id">
                {{ p.name }} · {{ p.model }}
              </option>
            </select></label
          >
          <p v-if="!provider" class="creative-note">
            {{ mode === 'byok' ? '请先添加自备密钥配置' : '管理员尚未启用生图供应商' }}
          </p>
          <p v-if="provider && mode === 'site' && !provider.has_key" class="creative-note">
            站点尚未设置此供应商密钥。你仍可投稿创意或使用自备密钥。
          </p>
          <label class="creative-field"
            >提示词<textarea
              ref="promptArea"
              v-model="prompt"
              rows="6"
              maxlength="8000"
              placeholder="描述你想创作的画面…"
            /></label
          ><a-button size="small" @click="libraryOpen = true">角色 / 组合 / 提示词模板</a-button>
          <div class="reference-heading">
            <strong>参考图 {{ references.length }} / {{ provider?.max_references ?? 0 }}</strong
            ><a-button size="mini" :disabled="!provider?.max_references" @click="fileInput?.click()"
              >上传</a-button
            ><a-button size="mini" :disabled="!provider?.max_references" @click="historyOpen = true"
              >常用图片</a-button
            >
          </div>
          <p v-if="!provider?.max_references" class="creative-note">
            此供应商尚未启用参考图接口，可进行文生图。
          </p>
          <input
            ref="fileInput"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            multiple
            hidden
            @change="(event) => uploadFiles((event.target as HTMLInputElement).files)"
          />
          <div
            v-for="(reference, index) in references"
            :key="reference.asset_id"
            class="studio-reference"
          >
            <button
              class="reference-preview"
              type="button"
              :aria-label="`预览参考图 ${reference.asset_id}`"
              @click="previewReference = reference.asset_id"
            >
              <CreativeImage :id="reference.asset_id" /></button
            ><a-input v-model="reference.note" placeholder="参考备注" size="mini" />
            <div>
              <button :disabled="index === 0" @click="studio.moveReference(index, -1)">↑</button
              ><button
                :disabled="index === references.length - 1"
                @click="studio.moveReference(index, 1)"
              >
                ↓</button
              ><button @click="studio.removeReference(index)">移除</button>
            </div>
          </div>
          <div class="studio-output-fields"><label class="creative-field"
            >画面尺寸<select v-model="size">
              <option v-for="s in provider?.sizes" :key="s">{{ s }}</option>
            </select></label
          ><label v-if="provider?.qualities.length" class="creative-field"
            >画质<select v-model="quality">
              <option value="">供应商默认</option>
              <option v-for="q in provider.qualities" :key="q">{{ q }}</option>
            </select></label
          ><label class="creative-field"
            >输出数量<input
              v-model.number="count"
              type="number"
              min="1"
              :max="provider?.max_count ?? 1"
          /></label></div>
        </fieldset>
        <p class="creative-note">
          {{ mode === 'byok' ? '不消耗站点额度；供应商可能收费' : `预计消耗 ${cost} 额度` }}
        </p>
        <a-button
          type="primary"
          long
          :loading="busy"
          :disabled="
            !ready ||
            (!pending && (!provider || (mode === 'site' && !provider.has_key) || !prompt.trim()))
          "
          @click="generate"
          >{{ pending ? '确认上次提交' : '开始生成' }}</a-button
        >
        <div class="creative-actions studio-secondary">
          <a-button size="small" @click="studio.run(studio.save)">保存草稿</a-button
          ><a-button size="small" :disabled="!!pending || !prompt.trim()" @click="ideaOpen = true"
            >投稿创意</a-button
          >
        </div>
      </aside>
      <section class="studio-canvas">
        <div class="canvas-heading"><span>作品画布</span><span class="creative-note">{{ selectedAsset ? '预览与编辑' : '你的下一幅作品，从这里开始' }}</span></div>
        <p v-if="activeTask" role="status" class="creative-note">
          任务 #{{ activeTask.task.id }} · {{ labels[activeTask.task.status]
          }}{{
            activeTask.task.status === 'unknown' ? '，预留额度待管理员核查' : '，离开页面仍会继续'
          }}
        </p>
        <div class="canvas-surface">
          <div v-if="selectedAsset" class="canvas-image" :style="{ transform: `scale(${zoom})` }">
            <CreativeImage :id="selectedAsset.id" full />
          </div>
          <div v-else class="creative-empty">
            <span aria-hidden="true" class="canvas-star">✧</span>
            <h2>把想象留在这里</h2>
            <p>
              从一段提示词或一张角色设定图开始。<br />结果默认仅自己可见，离开页面后任务会继续。
            </p>
          </div>
        </div>
        <div v-if="selectedAsset" class="creative-actions canvas-actions">
          <a-button size="small" @click="zoom = Math.max(0.25, zoom - 0.25)">缩小</a-button
          ><a-button size="small" @click="zoom = 1">适应画布</a-button
          ><a-button size="small" @click="zoom = Math.min(3, zoom + 0.25)">放大</a-button
          ><a-button size="small" @click="download">下载</a-button
          ><a-button
            size="small"
            :disabled="!provider?.max_references || !!pending"
            @click="studio.run(async () => studio.continueFrom(selectedAsset!))"
            >继续编辑</a-button
          ><a-button
            size="small"
            :disabled="!provider?.max_references || !!pending"
            @click="studio.run(async () => studio.addReference(selectedAsset!.id))"
            >加入参考图</a-button
          ><a-button type="primary" size="small" @click="publishOpen = true">发布到站内</a-button
          ><a-button size="small" status="danger" @click="removeResult">移除</a-button>
        </div>
        <div class="result-strip">
          <button
            v-for="asset in shownTasks.flatMap((t) => t.assets).slice(0, 20)"
            :key="asset.id"
            @click="selectResult(asset)"
          >
            <CreativeImage :id="asset.id" />
          </button>
        </div>
      </section>
      <aside class="studio-history creative-box">
        <h2 class="studio-panel-title"><span aria-hidden="true">02</span> 任务与历史</h2>
        <label class="creative-field"
          >历史范围<select v-model="historyScope">
            <option value="session">本会话</option>
            <option value="all">全部 / 赞助任务</option>
          </select></label
        ><a-input v-model="search" placeholder="搜索历史提示词" /><label class="creative-field"
          >状态<select v-model="status">
            <option value="">全部</option>
            <option v-for="(label, key) in labels" :key="key" :value="key">{{ label }}</option>
          </select></label
        >
        <div v-for="row in shownTasks" :key="row.task.id" class="studio-task">
          <div>
            <strong>#{{ row.task.id }}</strong
            ><a-tag size="small">{{ labels[row.task.status] }}</a-tag>
          </div>
          <p>{{ row.task.prompt }}</p>
          <small>{{ row.task.model }} · {{ row.task.size }}</small>
          <p v-if="row.task.parent_task_id" class="creative-note">
            继续自任务 #{{ row.task.parent_task_id }} · 图片 #{{ row.task.parent_asset_id }}
          </p>
          <p v-if="row.task.error" class="task-error">{{ row.task.error }}</p>
          <div class="creative-actions">
            <a-button v-if="row.assets[0]" size="mini" @click="selectResult(row.assets[0]!)"
              >查看结果</a-button
            ><a-button
              v-if="row.task.status === 'queued'"
              size="mini"
              @click="
                studio.run(async () => {
                  await aiApi.cancel(row.task.id)
                  await studio.refresh()
                })
              "
              >取消排队</a-button
            ><RouterLink v-if="row.task.idea_id" :to="`/ideas/${row.task.idea_id}`"
              >赞助创意</RouterLink
            >
          </div>
        </div>
        <a-button v-if="studio.moreTasks.value" long @click="studio.run(studio.loadMoreTasks)"
          >加载更早的任务</a-button
        >
        <div v-if="!shownTasks.length" class="studio-history-empty"><span aria-hidden="true">✧</span><p>还没有生成任务</p><small>每一次尝试，都会留在这里。</small></div>
      </aside>
    </div>
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
        <button
          v-for="asset in assets"
          :key="asset.id"
          class="library-asset"
          @click="
            studio.run(async () => {
              studio.addReference(asset.id)
              historyOpen = false
            })
          "
        >
          <CreativeImage :id="asset.id" /><span>{{ asset.title || `图片 ${asset.id}` }}</span>
        </button>
      </div>
      <a-button v-if="studio.moreAssets.value" @click="studio.run(studio.loadMoreAssets)"
        >加载更早的图片</a-button
      >
      <p v-if="!assets.length">上传的参考图与生成结果会保存在这里。</p></a-modal
    >
    <a-modal
      v-model:visible="keyOpen"
      class="creative-theme creative-dialog ai-dialog"
      title="自备 API Key"
      :footer="false"
      ><p class="creative-note">密钥在服务器加密保存。仅可选择管理员批准的供应商。</p>
      <label class="creative-field">配置名称<a-input v-model="keyForm.name" /></label
      ><label class="creative-field"
        >供应商<select v-model="keyForm.allowed_provider_id">
          <option
            v-for="p in providers.filter((p) => p.owner === 0 && p.enabled)"
            :key="p.id"
            :value="p.id"
          >
            {{ p.name }}
          </option>
        </select></label
      ><label class="creative-field"
        >API Key<a-input-password v-model="keyForm.api_key" autocomplete="new-password" /></label
      ><a-checkbox v-model="keyForm.default">设为默认</a-checkbox
      ><a-button type="primary" @click="saveKey">加密保存</a-button>
      <div v-for="p in providers.filter((p) => p.owner !== 0)" :key="p.id" class="studio-task">
        <strong>{{ p.name }}</strong>
        <div class="creative-actions">
          <a-button
            size="small"
            :disabled="!p.enabled"
            @click="
              studio.run(async () => {
                if (await confirm('连通测试会生成一张图，并可能产生供应商费用。继续？')) {
                  await aiApi.testBYOK(p.id, newKey())
                  keyOpen = false
                  await studio.refresh()
                }
              })
            "
            >测试</a-button
          ><a-button
            size="small"
            status="danger"
            @click="
              studio.run(async () => {
                await aiApi.deleteBYOK(p.id)
                await studio.loadProviders()
              })
            "
            >删除</a-button
          >
        </div>
      </div></a-modal
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
      ><label class="creative-field"
        >作品标题<a-input v-model="publishForm.title" aria-label="作品标题" /></label
      ><label class="creative-field"
        >作品描述<a-textarea v-model="publishForm.description" aria-label="作品描述" /></label
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
.ai-studio {
  max-width: 1600px;
}
.studio-top {
  margin-bottom: 20px;
}
.studio-top > div:first-child {
  min-width: 0;
}
.studio-top .arco-select-view-single {
  width: 180px;
}
.studio-grid {
  --studio-left-width: 300px;
  --studio-right-width: 260px;
  display: grid;
  grid-template-columns: var(--studio-left-width) minmax(0, 1fr) var(--studio-right-width);
  grid-template-areas: 'parameters canvas history';
  gap: 18px;
  min-height: 640px;
}
.studio-grid.hide-left {
  grid-template-columns: 0 minmax(0, 1fr) var(--studio-right-width);
}
.studio-grid.hide-right {
  grid-template-columns: var(--studio-left-width) minmax(0, 1fr) 0;
}
.studio-grid.hide-left.hide-right {
  grid-template-columns: 0 minmax(0, 1fr) 0;
}
.hide-left .studio-parameters,
.hide-right .studio-history {
  visibility: hidden;
  overflow: hidden;
  padding: 0;
  border: 0;
}
.studio-parameters {
  grid-area: parameters;
  max-height: clamp(640px, calc(100dvh - 320px), 900px);
  overflow: auto;
}
.studio-history {
  grid-area: history;
  max-height: clamp(640px, calc(100dvh - 320px), 900px);
  overflow: auto;
}
.studio-canvas {
  grid-area: canvas;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.canvas-surface {
  display: grid;
  place-items: center;
  flex: 1;
  min-height: 440px;
  overflow: auto;
  border: 1px solid var(--creative-line);
  border-radius: 22px;
  background: var(--creative-grain), var(--creative-canvas);
  background-size: 8px 8px;
}
.canvas-surface .creative-empty { background: none; border: 0; padding: 40px 24px; }
.canvas-surface .creative-empty p { font-size: 13px; }
.canvas-heading { display: flex; justify-content: space-between; gap: 12px; padding: 4px 4px 16px; color: var(--creative-muted); font-size: 13px; }
.studio-panel-title { display: flex; align-items: center; gap: 10px; padding-bottom: 16px; border-bottom: 1px solid var(--creative-line); }
.studio-panel-title span { color: var(--creative-accent); font-size: 11px; font-weight: 400; letter-spacing: .1em; }
.studio-output-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 12px; }
.studio-history-empty { text-align: center; padding: 42px 0; color: var(--creative-muted); }
.studio-history-empty > span { font-size: 32px; color: var(--creative-accent); }
.studio-history-empty small { font-size: 11px; }
.studio-session { display: flex; align-items: center; gap: 12px; margin-right: auto; }
.studio-session .arco-input-wrapper { width: 190px; }
.canvas-image {
  width: 100%;
  height: 500px;
  padding: 24px;
  transition: transform 0.2s;
}
.canvas-star {
  font-size: 72px;
  color: var(--creative-accent);
  font-weight: 200;
  display: grid;
  place-items: center;
  width: 124px;
  height: 124px;
  margin: 0 auto 24px;
  border: 1px solid var(--creative-line);
  border-radius: 50%;
  background: var(--creative-paper);
  box-shadow: 12px -8px 0 -4px var(--creative-soft);
}
.canvas-actions {
  padding: 18px 0;
}
.studio-top .creative-eyebrow {
  margin-bottom: 6px;
}
.studio-task {
  padding: 18px 0;
  border-bottom: 1px solid var(--creative-line);
  font-size: 12px;
}
.studio-task > div:first-child {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}
.studio-task p {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.studio-task .task-error {
  display: block;
  color: var(--creative-accent);
}
.studio-task small {
  color: var(--creative-muted);
}
.studio-reference {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 8px;
  margin-top: 12px;
}
.studio-reference > .reference-preview {
  grid-row: span 2;
  height: 72px;
}
.reference-preview {
  padding: 0;
}
.reference-large {
  height: 60dvh;
}
.studio-reference button {
  border: 0;
  background: transparent;
  color: var(--creative-accent);
  cursor: pointer;
  font-size: 11px;
}
.reference-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 22px;
  font-size: 12px;
}
.studio-secondary {
  margin-top: 14px;
}
.result-strip {
  display: flex;
  gap: 10px;
  overflow: auto;
  padding-bottom: 10px;
}
.result-strip button {
  flex: 0 0 92px;
  height: 90px;
  border: 1px solid var(--creative-line);
  border-radius: 8px;
  background: var(--creative-paper);
  cursor: pointer;
}
.studio-mobile-tabs {
  display: none;
}
.studio-toolbar {
  margin-bottom: 20px;
  padding: 12px 16px;
  background: var(--creative-paper);
  border: 1px solid var(--creative-line);
  border-radius: 16px;
}
.studio-parameters fieldset {
  padding: 0;
  border: 0;
  min-width: 0;
}
.library-asset {
  border: 1px solid var(--creative-line);
  border-radius: 10px;
  background: var(--creative-paper);
  color: var(--creative-ink);
  height: 180px;
  padding: 10px;
  cursor: pointer;
}
.library-asset .creative-image {
  height: 130px;
}
@media (max-width: 1100px) {
  .studio-grid {
    --studio-left-width: 260px;
    --studio-right-width: 220px;
  }
  .studio-top {
    align-items: flex-start;
    flex-direction: column;
  }
  .studio-parameters,
  .studio-history {
    max-height: none;
  }
}
@media (max-width: 800px) {
  .studio-mobile-tabs {
    display: block;
    margin: 14px 0;
  }
  .studio-toolbar {
    display: flex;
  }
  .studio-toolbar > .arco-btn, .studio-toolbar > .creative-note { display: none; }
  .studio-session { width: 100%; flex-wrap: wrap; }
  .studio-session .arco-input-wrapper { flex: 1; min-width: 140px; }
  .studio-grid,
  .studio-grid.hide-left,
  .studio-grid.hide-right,
  .studio-grid.hide-left.hide-right {
    display: block;
    min-height: 0;
  }
  .studio-parameters,
  .studio-history {
    visibility: visible !important;
    padding: 20px !important;
    border: 1px solid var(--creative-line) !important;
    max-height: none;
  }
  .studio-grid > * {
    display: none;
  }
  .mobile-parameters .studio-parameters,
  .mobile-canvas .studio-canvas,
  .mobile-history .studio-history {
    display: block;
  }
  .canvas-surface {
    min-height: 400px;
  }
  .canvas-image {
    height: 420px;
  }
  .studio-top .creative-actions {
    gap: 8px;
  }
  .studio-top .arco-select-view-single {
    max-width: 160px;
  }
  .ai-studio {
    padding: 0 14px;
  }
}
</style>
