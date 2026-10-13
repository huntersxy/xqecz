<script setup lang="ts">
function newKey() {
  return crypto.randomUUID()
}
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { aiApi, creativeApi } from '@/api/creative'
import type { AIProvider, AIAsset, AITaskView, AIIdeaView, CreativeResource } from '@/types/schemas'
import type { z } from 'zod'
import type { AILedgerSchema } from '@/types/schemas'
import { useConfirm } from '@/composables/useToast'
import CreativeImage from '@/components/CreativeImage.vue'
import AIProviderEditor from './AIProviderEditor.vue'
import CreativeResourceEditor from './CreativeResourceEditor.vue'
import '@/assets/creative-pages.css'
const tab = ref('providers'),
  providers = ref<AIProvider[]>([]),
  assets = ref<AIAsset[]>([]),
  tasks = ref<AITaskView[]>([]),
  ideas = ref<AIIdeaView[]>([]),
  resources = ref<CreativeResource[]>([]),
  ledger = ref<z.infer<typeof AILedgerSchema>[]>([]),
  error = ref(''),
  loading = ref(false),
  search = ref(''),
  status = ref('')
const providerOpen = ref(false),
  editingProvider = ref<AIProvider | null>(null),
  resourceOpen = ref(false),
  editingResource = ref<CreativeResource | null>(null),
  assetOpen = ref(false),
  editingAsset = ref<AIAsset | null>(null),
  resolveOpen = ref(false),
  resolveId = ref(0),
  consumed = ref(false),
  resolveNote = ref(''),
  grantForm = ref({ userId: 0, amount: 1, note: '' }),
  grantKey = ref(newKey()),
  fileInput = ref<HTMLInputElement>(),
  selected = ref<number[]>([])
const { confirm } = useConfirm()
let stopped = false
let queryController: AbortController | undefined
const moreRows = ref(false),
  eventsOpen = ref(false),
  events = ref<Awaited<ReturnType<typeof aiApi.ideaEvents>>>([])
const eventController = new AbortController()
async function queryRows(append = false) {
  if (!['tasks', 'sponsors', 'ideas'].includes(tab.value)) return
  queryController?.abort()
  queryController = new AbortController()
  const signal = queryController.signal,
    currentTab = tab.value
  const query = {
    search: search.value,
    status: status.value,
    sponsored: currentTab === 'sponsors' ? '1' : undefined,
    before_id: append
      ? currentTab === 'ideas'
        ? ideas.value.at(-1)?.idea.id
        : tasks.value.at(-1)?.task.id
      : undefined,
  }
  try {
    if (currentTab === 'ideas') {
      const rows = await aiApi.adminIdeas(query, signal)
      if (stopped || signal.aborted) return
      ideas.value = append ? [...ideas.value, ...rows] : rows
      moreRows.value = rows.length === 100
    } else {
      const rows = await aiApi.adminTasks(query, signal)
      if (stopped || signal.aborted) return
      tasks.value = append ? [...tasks.value, ...rows] : rows
      moreRows.value = rows.length === 100
    }
  } catch (e) {
    if (!signal.aborted && !stopped) throw e
  }
}
async function showEvents(id: number) {
  await run(async () => {
    const rows = await aiApi.ideaEvents(id, eventController.signal)
    if (stopped) return
    events.value = rows
    eventsOpen.value = true
  })
}
watch([tab, status], () => void run(() => queryRows()))
const menus = [
  ['providers', '供应商'],
  ['quota', '用户额度 / 账本'],
  ['tasks', '生图任务'],
  ['ideas', '创意投稿'],
  ['sponsors', '赞助记录'],
  ['character', '角色'],
  ['assets', '设定图'],
  ['preset', '参考图组合'],
  ['template', '提示词模板'],
]
const kind = computed(() =>
  ['character', 'preset', 'template'].includes(tab.value)
    ? (tab.value as CreativeResource['kind'])
    : 'character',
)
const shownResources = computed(() =>
  resources.value.filter(
    (r) =>
      r.kind === kind.value &&
      `${r.title} ${r.summary}`.includes(search.value) &&
      (!status.value || r.status === status.value),
  ),
)
const shownTasks = computed(() =>
  tasks.value.filter(
    (t) =>
      (tab.value !== 'sponsors' || t.task.idea_id > 0) &&
      (!status.value || t.task.status === status.value) &&
      t.task.prompt.includes(search.value),
  ),
)
async function run(fn: () => Promise<unknown>) {
  error.value = ''
  try {
    await fn()
  } catch (e) {
    if (!stopped) error.value = (e as Error).message
  }
}
async function load() {
  loading.value = true
  try {
    const [p, a, t, i, r, l] = await Promise.all([
      aiApi.providers(),
      aiApi.adminAssets(),
      aiApi.adminTasks(),
      aiApi.adminIdeas(),
      creativeApi.adminResources(),
      aiApi.adminLedger(),
    ])
    if (stopped) return
    providers.value = p
    assets.value = a
    tasks.value = t
    ideas.value = i
    resources.value = r
    ledger.value = l
    await queryRows()
  } finally {
    loading.value = false
  }
}
function editResource(row: CreativeResource | null) {
  editingResource.value = row
  resourceOpen.value = true
}
function switchTab(value: string) {
  tab.value = value
  search.value = ''
  status.value = ''
  selected.value = []
}
function editProvider(row: AIProvider | null) {
  editingProvider.value = row
  providerOpen.value = true
}
function editAsset(row: AIAsset) {
  editingAsset.value = { ...row }
  assetOpen.value = true
}
function startResolve(id: number) {
  resolveId.value = id
  consumed.value = false
  resolveNote.value = ''
  resolveOpen.value = true
}
async function saved() {
  providerOpen.value = false
  resourceOpen.value = false
  assetOpen.value = false
  await run(load)
}
async function grant() {
  await run(async () => {
    await aiApi.grant(
      grantForm.value.userId,
      grantForm.value.amount,
      grantForm.value.note,
      grantKey.value,
    )
    grantKey.value = newKey()
    await load()
  })
}
async function upload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  await run(async () => {
    const row = await aiApi.upload(file)
    editingAsset.value = { ...row, title: file.name, license: '', public: false }
    assetOpen.value = true
    await load()
  })
  if (fileInput.value) fileInput.value.value = ''
}
async function saveAsset() {
  if (!editingAsset.value) return
  await run(async () => {
    await aiApi.metadata(editingAsset.value!.id, editingAsset.value!)
    await saved()
  })
}
async function archiveSelected() {
  if (!selected.value.length || !(await confirm('将选中资源归档？历史生成任务会保留原快照。')))
    return
  await run(async () => {
    for (const row of resources.value.filter((r) => selected.value.includes(r.id))) {
      await creativeApi.save({ ...row, status: 'archived' })
    }
    selected.value = []
    await load()
  })
}
async function resolve() {
  await run(async () => {
    await aiApi.resolve(resolveId.value, consumed.value, resolveNote.value)
    resolveOpen.value = false
    await load()
  })
}
onMounted(() => void run(load))
onBeforeUnmount(() => {
  stopped = true
  queryController?.abort()
  eventController.abort()
})
</script>
<template>
  <AdminPanel
    title="AI 创作管理"
    desc="资源、供应商、额度和社区赞助使用同一套持久化任务与审计记录。"
    ><template #actions
      ><a-button size="small" :loading="loading" @click="run(load)">刷新</a-button></template
    >
    <div class="admin-ai-body">
    <div class="admin-ai-tabs">
      <button
        v-for="menu in menus"
        :key="menu[0]"
        :class="{ active: tab === menu[0] }"
        @click="switchTab(menu[0]!)"
      >
        {{ menu[1] }}
      </button>
    </div>
    <div v-if="error" class="creative-error">{{ error }}</div>
    <template v-if="tab === 'providers'"
      ><a-button type="primary" @click="editProvider(null)">添加供应商</a-button>
      <div v-for="p in providers" :key="p.id" class="admin-ai-row">
        <div>
          <strong>{{ p.name }}</strong>
          <p>
            {{ p.model }} · {{ p.enabled ? '启用' : '停用' }} ·
            {{ p.has_key ? '密钥已加密保存' : '未配置密钥' }}
          </p>
          <small>{{ p.endpoint }}</small>
        </div>
        <div class="creative-actions">
          <a-button @click="editProvider(p)">编辑</a-button
          ><a-button
            :disabled="!p.enabled"
            @click="
              run(async () => {
                if (await confirm('测试会请求生成一张图，可能产生供应商费用。继续？')) {
                  await aiApi.testProvider(p.id, newKey())
                  tab = 'tasks'
                  await load()
                }
              })
            "
            >生成连通测试</a-button
          >
        </div>
      </div></template
    >
    <template v-else-if="tab === 'quota'"
      ><div class="creative-box">
        <h3>发放额度</h3>
        <div class="creative-actions">
          <a-input-number
            v-model="grantForm.userId"
            placeholder="用户 ID"
            :min="1"
          /><a-input-number v-model="grantForm.amount" :min="1" :max="10000" /><a-input
            v-model="grantForm.note"
            placeholder="发放原因"
          /><a-button type="primary" @click="grant">发放</a-button>
        </div>
      </div>
      <div class="admin-ai-table">
        <table>
          <thead>
            <tr>
              <th>时间</th>
              <th>用户</th>
              <th>任务</th>
              <th>类型</th>
              <th>可用额度变化</th>
              <th>记录</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in ledger" :key="row.id">
              <td>{{ new Date(row.created_at).toLocaleString() }}</td>
              <td>{{ row.user_id }}</td>
              <td>{{ row.task_id || '—' }}</td>
              <td>{{ row.kind }}</td>
              <td>{{ row.amount }}</td>
              <td>{{ row.note }}</td>
            </tr>
          </tbody>
        </table>
      </div></template
    >
    <template v-else-if="tab === 'tasks' || tab === 'sponsors'"
      ><div class="creative-actions">
        <a-input v-model="search" placeholder="搜索提示词" /><a-select
          v-model="status"
          placeholder="全部状态"
          allow-clear
          ><a-option
            v-for="s in ['queued', 'running', 'succeeded', 'failed', 'unknown', 'cancelled']"
            :key="s"
            >{{ s }}</a-option
          ></a-select
        >
        <a-button @click="run(() => queryRows())">搜索全部任务</a-button>
      </div>
      <div v-for="row in shownTasks" :key="row.task.id" class="admin-ai-row">
        <div>
          <strong>#{{ row.task.id }} · {{ row.task.status }}</strong>
          <p>{{ row.task.prompt }}</p>
          <small
            >用户 {{ row.task.user_id }} · {{ row.task.model }} · {{ row.task.cost }} 额度 · 创意
            {{ row.task.idea_id || '—' }}</small
          >
          <p v-if="row.task.error">{{ row.task.error }}</p>
          <div class="admin-result-images">
            <CreativeImage v-for="asset in row.assets" :key="asset.id" :id="asset.id" />
          </div>
          <details v-if="row.references?.length">
            <summary>原始参考图</summary>
            <div class="admin-result-images">
              <CreativeImage v-for="asset in row.references" :key="asset.id" :id="asset.id" />
            </div>
          </details>
        </div>
        <a-button v-if="row.task.status === 'unknown'" @click="startResolve(row.task.id)"
          >核查结算</a-button
        >
      </div>
      <a-button v-if="moreRows" @click="run(() => queryRows(true))"
        >加载更早的任务</a-button
      ></template
    >
    <template v-else-if="tab === 'ideas'"
      ><div class="creative-actions">
        <a-input
          v-model="search"
          placeholder="搜索投稿"
          @press-enter="run(() => queryRows())"
        /><a-select v-model="status" placeholder="全部状态" allow-clear
          ><a-option
            v-for="s in [
              'waiting',
              'generating',
              'realized',
              'unknown',
              'unavailable',
              'hidden',
              'withdrawn',
            ]"
            :key="s"
            :value="s"
            >{{ s }}</a-option
          ></a-select
        ><a-button @click="run(() => queryRows())">搜索投稿</a-button>
      </div>
      <div v-for="row in ideas" :key="row.idea.id" class="admin-ai-row">
        <div>
          <strong>{{ row.idea.title }} · {{ row.idea.status }}</strong>
          <p>{{ row.idea.description }}</p>
          <details>
            <summary>提示词与参数</summary>
            <pre>{{ row.idea.snapshot }}</pre>
          </details>
          <div class="admin-result-images">
            <CreativeImage v-for="asset in row.references" :key="asset.id" :id="asset.id" />
          </div>
          <p class="creative-note">
            创意 {{ row.author.username }} · 赞助 {{ row.sponsor?.username || '等待赞助' }} ·
            {{ row.current_cost }} 额度
          </p>
        </div>
        <div class="creative-actions">
          <a-button @click="showEvents(row.idea.id)">操作与核查记录</a-button>
          <RouterLink :to="`/ideas/${row.idea.id}`">预览</RouterLink
          ><a-button
            v-if="row.sponsorable"
            @click="
              run(async () => {
                if (await confirm(`赞助此创意，预留 ${row.current_cost} 额度？`)) {
                  await aiApi.sponsor(row.idea.id, row.current_cost, newKey())
                  await load()
                }
              })
            "
            >赞助</a-button
          ><a-button
            status="danger"
            @click="
              run(async () => {
                if (await confirm('下架此创意与公开成果？')) {
                  await aiApi.hideIdea(row.idea.id, '管理员下架')
                  await load()
                }
              })
            "
            >下架</a-button
          >
        </div>
      </div>
      <a-button v-if="moreRows" @click="run(() => queryRows(true))"
        >加载更早的投稿</a-button
      ></template
    >
    <template v-else-if="tab === 'assets'"
      ><a-button type="primary" @click="fileInput?.click()">上传新的设定图版本</a-button
      ><input
        ref="fileInput"
        type="file"
        accept="image/png,image/jpeg,image/webp"
        hidden
        @change="upload"
      />
      <p class="creative-note">文件不可原地替换。修改图像请上传新版本，旧图与历史引用继续保留。</p>
      <div class="creative-grid">
        <article v-for="asset in assets" :key="asset.id" class="creative-box">
          <div class="creative-image-tile"><CreativeImage :id="asset.id" /></div>
          <h3>#{{ asset.id }} {{ asset.title || '未命名图片' }}</h3>
          <p>
            {{ asset.public ? '公开' : '私人' }} · {{ asset.archived ? '已归档' : '可用' }} ·
            {{ asset.license || '尚未填写授权' }}
          </p>
          <a-button @click="editAsset(asset)">编辑 / 发布 / 归档</a-button>
        </article>
      </div></template
    >
    <template v-else
      ><div class="creative-actions">
        <a-button type="primary" @click="editResource(null)"
          >新建{{
            kind === 'character' ? '角色' : kind === 'preset' ? '参考组合' : '提示词'
          }}</a-button
        ><a-button :disabled="!selected.length" @click="archiveSelected">批量归档</a-button
        ><a-input v-model="search" placeholder="搜索资源" /><a-select
          v-model="status"
          placeholder="全部状态"
          allow-clear
          ><a-option value="draft">草稿</a-option><a-option value="published">发布</a-option
          ><a-option value="archived">归档</a-option></a-select
        >
      </div>
      <div v-for="row in shownResources" :key="row.id" class="admin-ai-row">
        <div>
          <a-checkbox
            :model-value="selected.includes(row.id)"
            @change="
              (checked) =>
                (selected = checked
                  ? [...selected, row.id]
                  : selected.filter((id) => id !== row.id))
            "
            >{{ row.title }}</a-checkbox
          >
          <p>
            {{ row.summary }} · {{ row.status }} · 版本 {{ row.version }} · 排序
            {{ row.sort_order }}
          </p>
        </div>
        <a-button @click="editResource(row)">编辑 / 预览</a-button>
      </div></template
    >
    <a-modal
      v-model:visible="providerOpen"
      class="admin-theme creative-theme creative-dialog ai-dialog"
      title="供应商配置"
      :width="760"
      :footer="false"
      ><AIProviderEditor v-if="providerOpen" :provider="editingProvider" @saved="saved"
    /></a-modal>
    <a-modal
      v-model:visible="eventsOpen"
      class="admin-theme creative-theme creative-dialog ai-dialog"
      title="创意操作与核查记录"
      :footer="false"
      ><div v-for="event in events" :key="event.id" class="studio-task">
        <strong
          >{{ event.action }} · 任务 #{{ event.task_id }} · 操作人 #{{ event.actor_id }}</strong
        >
        <p>{{ event.note }}</p>
        <small>{{ event.created_at }}</small>
      </div></a-modal
    >
    <a-modal
      v-model:visible="resourceOpen"
      class="admin-theme creative-theme creative-dialog ai-dialog"
      title="创作资源"
      :width="800"
      :footer="false"
      ><CreativeResourceEditor
        v-if="resourceOpen"
        :resource="editingResource"
        :kind="kind"
        :assets="assets"
        :resources="resources"
        @saved="saved"
    /></a-modal>
    <a-modal
      v-model:visible="assetOpen"
      class="admin-theme creative-theme creative-dialog ai-dialog"
      title="设定图信息"
      :footer="false"
      ><template v-if="editingAsset"
        ><label class="creative-field">标题<a-input v-model="editingAsset.title" /></label
        ><label class="creative-field">描述<a-textarea v-model="editingAsset.description" /></label
        ><label class="creative-field"
          >用途<a-input
            v-model="editingAsset.purpose"
            placeholder="标准设定、服装、表情、场景…" /></label
        ><label class="creative-field"
          >来源与授权说明<a-textarea v-model="editingAsset.license" /></label
        ><a-checkbox v-model="editingAsset.public">公开素材</a-checkbox
        ><a-checkbox v-model="editingAsset.download_allowed">允许下载与公开投稿参考</a-checkbox
        ><a-checkbox v-model="editingAsset.archived">归档</a-checkbox
        ><a-button type="primary" @click="saveAsset">保存</a-button></template
      ></a-modal
    >
    <a-modal
      v-model:visible="resolveOpen"
      class="admin-theme creative-theme creative-dialog ai-dialog"
      title="核查结果不明的任务"
      :footer="false"
      ><p>先在供应商侧核查实际执行与计费情况，再释放或结算预留额度。此操作不会重新请求生图。</p>
      <a-checkbox v-model="consumed">已确认产生供应商费用，结算预留额度</a-checkbox
      ><label class="creative-field"
        >核查依据与说明（必填）<a-textarea v-model="resolveNote" /></label
      ><a-button type="primary" @click="resolve">{{
        consumed ? '确认结算' : '确认未消耗，释放预留'
      }}</a-button></a-modal
    >
    </div>
  </AdminPanel>
</template>
<style scoped>
.admin-ai-body { padding: 24px; }
.admin-ai-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 22px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--creative-line);
}
.admin-ai-tabs button {
  padding: 9px 16px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
  background: var(--creative-paper);
  color: var(--creative-muted);
  cursor: pointer;
}
.admin-ai-tabs button.active {
  background: var(--creative-soft);
  color: var(--creative-accent);
  border-color: var(--creative-accent);
}
.admin-ai-row {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  align-items: flex-start;
  padding: 22px;
  margin-top: 14px;
  border: 1px solid var(--creative-line);
  border-radius: 16px;
  background: var(--creative-paper);
}
.admin-ai-row > div:first-child {
  min-width: 0;
  flex: 1;
}
.admin-ai-row p {
  line-height: 1.6;
  color: var(--creative-muted);
  overflow-wrap: anywhere;
}
.admin-ai-row small {
  overflow-wrap: anywhere;
}
.admin-ai-row pre {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.admin-ai-table {
  overflow: auto;
  margin-top: 24px;
}
.admin-ai-table table {
  width: 100%;
  min-width: 650px;
  text-align: left;
  border-collapse: collapse;
}
.admin-ai-table td,
.admin-ai-table th {
  padding: 14px;
  border-bottom: 1px solid var(--creative-line);
}
.admin-result-images {
  display: flex;
  gap: 10px;
}
.admin-result-images > * {
  width: 100px;
  height: 100px;
}
.creative-actions > .arco-input-wrapper {
  max-width: 280px;
}
.creative-actions > .arco-select-view-single {
  max-width: 180px;
}
.creative-grid {
  margin-top: 20px;
}
@media (max-width: 650px) {
  .admin-ai-body { padding: 16px; }
  .admin-ai-row {
    flex-direction: column;
  }
}
</style>
