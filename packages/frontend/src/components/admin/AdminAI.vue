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
import { creativeLicenseOptions } from '@/utils/creativeLicenses'
const tab = ref('providers'),
  providers = ref<AIProvider[]>([]),
  assets = ref<AIAsset[]>([]),
  tasks = ref<AITaskView[]>([]),
  ideas = ref<AIIdeaView[]>([]),
  resources = ref<CreativeResource[]>([]),
  promptEnhancement = ref({ enabled: true, prompt: '' }),
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
  grantForm = ref({ users: [] as number[], scope: 'selected', amount: 1, note: '' }),
  grantKey = ref(newKey()),
  fileInput = ref<HTMLInputElement>(),
  selected = ref<number[]>([])
const assetError = ref(''),
  assetSaving = ref(false)
const promptEnhancementSaving = ref(false)
const assetLicenseChoice = ref('')
const previewAsset = ref<{ id: number; title: string } | null>(null)
const resolveError = ref(''),
  resolveBusy = ref(false)
watch(assetLicenseChoice, (value) => {
  if (editingAsset.value && value && value !== 'other') editingAsset.value.license = value
})
const grantBusy = ref(false)
const grantUsers = ref<{ id: number; username: string }[]>([])
const pendingGrant = ref<{ users: number[]; all: boolean; amount: number; note: string } | null>(
  null,
)
const grantSuccess = ref('')
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
  ['character', '角色资源'],
  ['template', '提示词模板'],
]
const taskLabels: Record<string, string> = {
  queued: '等待生成',
  running: '正在生成',
  succeeded: '生成成功',
  failed: '生成失败',
  unknown: '结果待核查',
  cancelled: '已取消',
}
const adapterLabels: Record<AIProvider['adapter'], string> = {
  openai: 'Images',
  openai_chat: 'Chat',
  gemini: 'Gemini',
  codex_responses: 'Codex Responses',
  volcengine_ark: '火山方舟',
  gitee_ai: 'Gitee AI',
  siliconflow_adapter: 'SiliconFlow',
  agnes_ai: 'Agnes AI',
  grok: 'Grok',
}
const ledgerLabels: Record<string, string> = {
  grant: '额度发放',
  reserve: '生成预留',
  settle: '生成扣减',
  release: '退回额度',
}
const kind = computed(() =>
  ['character', 'preset', 'template'].includes(tab.value)
    ? (tab.value as CreativeResource['kind'])
    : 'character',
)
const shownResources = computed(() =>
  resources.value.filter(
    (r) =>
      (kind.value === 'character'
        ? ['character', 'preset'].includes(r.kind)
        : r.kind === kind.value) &&
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
const unassignedAssets = computed(() =>
  assets.value.filter(
    (a) =>
      !a.archived &&
      !resources.value.some((r) => r.kind !== 'template' && r.payload.asset_ids.includes(a.id)),
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
    const [p, a, t, i, r, l, users, enhancement] = await Promise.all([
      aiApi.providers(),
      aiApi.adminAssets(),
      aiApi.adminTasks(),
      aiApi.adminIdeas(),
      creativeApi.adminResources(),
      aiApi.adminLedger(),
      aiApi.grantUsers(),
      aiApi.adminPromptEnhancement(),
    ])
    if (stopped) return
    providers.value = p
    assets.value = a
    tasks.value = t
    ideas.value = i
    resources.value = r
    ledger.value = l
    grantUsers.value = users
    promptEnhancement.value = enhancement
    await queryRows()
  } finally {
    loading.value = false
  }
}
async function savePromptEnhancement() {
  if (promptEnhancementSaving.value) return
  promptEnhancementSaving.value = true
  await run(async () => {
    promptEnhancement.value = await aiApi.savePromptEnhancement(promptEnhancement.value)
  })
  promptEnhancementSaving.value = false
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
  assetError.value = ''
  editingAsset.value = { ...row }
  assetLicenseChoice.value = creativeLicenseOptions.some((option) => option.value === row.license)
    ? row.license
    : row.license
      ? 'other'
      : ''
  assetOpen.value = true
}
function startResolve(id: number) {
  resolveId.value = id
  consumed.value = false
  resolveNote.value = ''
  resolveError.value = ''
  resolveOpen.value = true
}
async function saved() {
  providerOpen.value = false
  resourceOpen.value = false
  assetOpen.value = false
  await run(load)
}
async function grant() {
  if (grantBusy.value) return
  const { users, scope, amount, note } = grantForm.value
  if (!pendingGrant.value && scope === 'selected' && !users.length) {
    error.value = '请选择要发放额度的用户'
    return
  }
  if (!Number.isSafeInteger(amount) || amount < 1 || amount > 10000) {
    error.value = '发放额度须为1–10000的整数'
    return
  }
  grantBusy.value = true
  grantSuccess.value = ''
  try {
    error.value = ''
    pendingGrant.value ??= {
      users: scope === 'all' ? [] : [...users],
      all: scope === 'all',
      amount,
      note,
    }
    const target = pendingGrant.value
    const result = await aiApi.grantBatch(
      target.users,
      target.all,
      target.amount,
      target.note,
      grantKey.value,
    )
    if (!stopped) {
      grantKey.value = newKey()
      pendingGrant.value = null
      grantSuccess.value = `已为 ${result.count} 位用户各增加 ${target.amount} 额度`
      await load()
    }
  } catch (e) {
    if (!stopped) error.value = (e as Error).message
    if ((e as Error).name === 'BusinessError') pendingGrant.value = null
  } finally {
    grantBusy.value = false
  }
}
async function upload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  await run(async () => {
    const row = await aiApi.upload(file)
    editAsset({ ...row, title: file.name, license: '', public: false })
    await load()
  })
  if (fileInput.value) fileInput.value.value = ''
}
async function saveAsset() {
  if (!editingAsset.value || assetSaving.value) return
  assetError.value = ''
  const {
    id,
    title,
    description,
    license,
    purpose,
    public: isPublic,
    download_allowed,
    archived,
  } = editingAsset.value
  if (isPublic && !license.trim()) {
    assetError.value = '公开素材前请填写来源与授权说明'
    return
  }
  assetSaving.value = true
  try {
    // Submit editable metadata only; ownership and task provenance remain server-owned.
    await aiApi.metadata(id, {
      title,
      description,
      license,
      purpose,
      public: isPublic,
      download_allowed,
      archived,
    })
    if (!stopped) await saved()
  } catch (e) {
    if (!stopped) assetError.value = (e as Error).message
  } finally {
    assetSaving.value = false
  }
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
async function deleteTask(id: number) {
  if (!(await confirm('删除这条任务及其生成结果？删除后仅保留后台审计记录，不能恢复。'))) return
  await run(async () => {
    await aiApi.deleteAdminTask(id)
    await load()
  })
}
async function deleteIdea(id: number) {
  if (!(await confirm('永久删除这条创意投稿及其操作记录？关联的生成任务会保留为独立审计记录。'))) return
  await run(async () => {
    await aiApi.deleteIdea(id)
    await load()
  })
}
async function deleteResource(row: CreativeResource) {
  if (!(await confirm(`删除资源“${row.title}”？历史任务快照会保留，操作不能恢复。`))) return
  await run(async () => {
    await creativeApi.delete(row.id)
    await load()
  })
}
async function deleteAsset(id: number) {
  if (!(await confirm('删除这张图片？历史任务快照会保留，图片将从素材库移除。'))) return
  await run(async () => {
    await aiApi.deleteAdminAsset(id)
    await load()
  })
}
async function resolve() {
  if (resolveBusy.value) return
  const note = resolveNote.value.trim()
  resolveError.value = ''
  if (Array.from(note).length < 3 || Array.from(note).length > 300) {
    resolveError.value = '核查依据与说明须填写 3–300 个字'
    return
  }
  resolveBusy.value = true
  try {
    await aiApi.resolve(resolveId.value, consumed.value, note)
    if (stopped) return
    resolveOpen.value = false
    await run(load)
  } catch (e) {
    if (!stopped) resolveError.value = (e as Error).message
  } finally {
    resolveBusy.value = false
  }
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
        ><div class="creative-box prompt-enhancement-admin">
          <div class="creative-actions">
            <div>
              <h3>提示词增强</h3>
              <p class="creative-note">勾选后会把这段固定要求追加到用户提示词末尾。所有用户可在工作台悬浮查看。</p>
            </div>
            <a-button type="primary" :loading="promptEnhancementSaving" @click="savePromptEnhancement">保存提示词</a-button>
          </div>
          <a-checkbox v-model="promptEnhancement.enabled">允许用户使用提示词增强（工作台默认勾选）</a-checkbox>
          <a-textarea v-model="promptEnhancement.prompt" :maxlength="4000" :auto-size="{ minRows: 3, maxRows: 8 }" aria-label="提示词增强内容" />
        </div>
        <a-button type="primary" @click="editProvider(null)">添加供应商</a-button>
        <div v-for="p in providers" :key="p.id" class="admin-ai-row">
          <div>
            <strong>{{ p.name }}</strong>
            <p>
              {{ adapterLabels[p.adapter] }} · {{ p.model }} · {{ p.enabled ? '启用' : '停用' }} ·
              {{ p.has_key ? '密钥已加密保存' : '未配置密钥' }}
            </p>
            <small>{{ p.endpoint }}</small>
          </div>
          <div class="creative-actions">
            <a-button @click="editProvider(p)">编辑</a-button
            ><a-button
              :status="p.enabled ? 'warning' : 'success'"
              @click="
                run(async () => {
                  const action = p.enabled ? '关闭' : '启用'
                  if (await confirm(`${action}供应商“${p.name}”？`)) {
                    await aiApi.setProviderEnabled(p.id, !p.enabled)
                    await load()
                  }
                })
              "
              >{{ p.enabled ? '关闭' : '启用' }}</a-button
            ><a-button
              status="danger"
              @click="
                run(async () => {
                  if (await confirm(`删除供应商“${p.name}”？删除后不可恢复。`)) {
                    await aiApi.deleteProvider(p.id)
                    await load()
                  }
                })
              "
              >删除</a-button
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
          <p class="creative-note">
            选择用户或全体用户。以下数量是每位用户本次增加的可用额度；管理员生成不消耗站点额度。
          </p>
          <a-radio-group
            v-model="grantForm.scope"
            :disabled="grantBusy || !!pendingGrant"
            type="button"
            ><a-radio value="selected">选择用户</a-radio
            ><a-radio value="all">全体用户</a-radio></a-radio-group
          >
          <div class="grant-fields">
            <label v-if="grantForm.scope === 'selected'" class="creative-field"
              >发放给谁
              <a-select
                v-model="grantForm.users"
                multiple
                allow-search
                placeholder="搜索并选择用户"
                :disabled="grantBusy || !!pendingGrant"
                ><a-option v-for="u in grantUsers" :key="u.id" :value="u.id"
                  >{{ u.username }}（#{{ u.id }}）</a-option
                ></a-select
              >
            </label>
            <div v-else class="creative-field">
              <span>发放给谁</span><strong>全体用户</strong
              ><span class="creative-note">提交时由服务器确定完整用户列表</span>
            </div>
            <label class="creative-field"
              >发放额度
              <a-input-number
                v-model="grantForm.amount"
                :input-attrs="{ 'aria-label': '发放额度' }"
                :min="1"
                :max="10000"
                :precision="0"
                :disabled="grantBusy || !!pendingGrant"
              />
              <span class="creative-note">本次增加的额度，1–10000</span>
            </label>
            <label class="creative-field"
              >发放原因（选填）
              <a-input
                v-model="grantForm.note"
                :input-attrs="{ 'aria-label': '发放原因' }"
                placeholder="如：活动奖励、创作支持"
                :max-length="300"
                :disabled="grantBusy || !!pendingGrant"
              />
            </label>
            <a-button type="primary" :loading="grantBusy" @click="grant">{{
              pendingGrant ? '确认上次发放' : '发放额度'
            }}</a-button>
          </div>
          <p v-if="grantSuccess" role="status" class="creative-note">{{ grantSuccess }}</p>
          <p v-if="pendingGrant" class="creative-note">
            发放结果尚未确认，再次提交会核对上次操作，不会重复增加额度。
          </p>
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
                <td>
                  {{
                    grantUsers.find((user) => user.id === row.user_id)?.username ||
                    `用户 #${row.user_id}`
                  }}
                </td>
                <td>{{ row.task_id || '—' }}</td>
                <td>{{ ledgerLabels[row.kind] || row.kind }}</td>
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
            placeholder="全部生成状态"
            allow-clear
            ><a-option
              v-for="s in ['queued', 'running', 'succeeded', 'failed', 'unknown', 'cancelled']"
              :key="s"
              :value="s"
              >{{ taskLabels[s] }}</a-option
            ></a-select
          >
          <a-button @click="run(() => queryRows())">搜索全部任务</a-button>
        </div>
        <p class="creative-note">
          {{
            tab === 'sponsors'
              ? '这里显示赞助产生的生成任务。状态表示生成进度；“结果待核查”表示执行或费用尚未确认，需要管理员核查。'
              : '状态表示图片生成进度；失败与结果待核查的任务会显示具体原因。'
          }}
        </p>
        <article v-for="row in shownTasks" :key="row.task.id" class="admin-task-record">
          <header class="admin-task-header">
            <div class="admin-task-identity">
              <strong>#{{ row.task.id }}</strong>
              <a-tag>{{ taskLabels[row.task.status] }}</a-tag>
              <span>{{
                row.user?.username ||
                grantUsers.find((user) => user.id === row.task.user_id)?.username ||
                '已注销用户'
              }}</span>
            </div>
            <time :datetime="row.task.created_at">{{
              new Date(row.task.created_at).toLocaleString()
            }}</time>
          </header>
          <div class="admin-task-layout">
            <div class="admin-task-main">
              <div class="admin-task-meta">
                <span>{{ row.task.model }}</span
                ><span>{{ row.task.size }} · {{ row.task.count }} 张</span>
                <span>{{ row.task.cost }} 额度</span>
                <RouterLink v-if="row.task.idea_id" :to="`/ideas/${row.task.idea_id}`"
                  >创意 #{{ row.task.idea_id }}</RouterLink
                >
              </div>
              <details class="admin-task-prompt">
                <summary>
                  <span class="admin-task-prompt-excerpt">{{ row.task.prompt }}</span
                  ><span class="admin-prompt-expand">展开提示词</span
                  ><span class="admin-prompt-collapse">收起提示词</span>
                </summary>
                <pre>{{ row.task.prompt }}</pre>
              </details>
              <p v-if="row.task.error" class="creative-error" role="alert">{{ row.task.error }}</p>
              <div v-if="row.assets.length" class="admin-task-results" aria-label="生成结果">
                <button
                  v-for="(asset, index) in row.assets"
                  :key="asset.id"
                  type="button"
                  class="admin-image-preview"
                  :aria-label="`查看生成结果 ${index + 1}`"
                  @click="previewAsset = { id: asset.id, title: `生成结果 ${index + 1}` }"
                >
                  <CreativeImage :id="asset.id" />
                  <span>结果 {{ index + 1 }} · 查看大图</span>
                </button>
              </div>
              <p v-else class="admin-task-no-results">
                {{ row.task.status === 'succeeded' ? '暂无可展示的结果图片' : '尚无生成结果' }}
              </p>
              <div class="creative-actions">
                <a-button v-if="row.task.status === 'unknown'" @click="startResolve(row.task.id)"
                  >核查结算</a-button
                ><a-button status="danger" @click="deleteTask(row.task.id)">删除任务及结果</a-button>
              </div>
            </div>
            <aside
              v-if="row.references?.length"
              class="admin-task-references"
              aria-label="原始参考图"
            >
              <h3>
                原始参考图 <span>{{ row.references.length }}</span>
              </h3>
              <div class="admin-task-reference-list">
                <button
                  v-for="(asset, index) in row.references"
                  :key="asset.id"
                  type="button"
                  class="admin-image-preview"
                  :aria-label="`查看原始参考图 ${index + 1}`"
                  @click="previewAsset = { id: asset.id, title: `原始参考图 ${index + 1}` }"
                >
                  <CreativeImage :id="asset.id" /><span>参考图 {{ index + 1 }}</span>
                </button>
              </div>
            </aside>
          </div>
        </article>
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
            ><a-button status="danger" @click="deleteIdea(row.idea.id)">删除</a-button>
          </div>
        </div>
        <a-button v-if="moreRows" @click="run(() => queryRows(true))"
          >加载更早的投稿</a-button
        ></template
      >
      <template v-else-if="tab === 'character'">
        <div class="creative-actions">
          <a-button type="primary" @click="editResource(null)">新建角色资源</a-button
          ><a-input v-model="search" placeholder="搜索角色资源" />
        </div>
        <p class="creative-note">
          一个角色资源包含名称、外观说明和参考图片。在同一处管理，生成时一起导入。
        </p>
        <div v-for="row in shownResources" :key="row.id" class="admin-ai-row">
          <div>
            <strong>{{ row.title }}</strong>
            <p>
              {{ row.summary || '暂无简介' }} ·
              {{ row.status === 'published' ? '已公开展示' : '未公开展示' }} ·
              {{ row.payload.asset_ids.length }} 张参考图
            </p>
            <div class="admin-result-images">
              <CreativeImage v-for="id in row.payload.asset_ids.slice(0, 4)" :key="id" :id="id" />
            </div>
          </div>
          <div class="creative-actions">
            <a-button @click="editResource(row)">管理角色与图片</a-button
            ><a-button status="danger" @click="deleteResource(row)">删除</a-button>
          </div>
        </div>
        <p v-if="!shownResources.length" class="creative-empty">
          添加角色名称与图片，就可以开始建立角色资源。
        </p>
        <details class="resource-advanced">
          <summary>未归入角色的图片（{{ unassignedAssets.length }}）</summary>
          <a-button @click="fileInput?.click()">单独上传图片</a-button
          ><input
            ref="fileInput"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            hidden
            @change="upload"
          />
          <p class="creative-note">
            文件不可原地替换。修改图像请上传新版本，旧图与历史引用继续保留。
          </p>
          <div class="creative-grid">
            <article v-for="asset in unassignedAssets" :key="asset.id" class="creative-box">
              <div class="creative-image-tile"><CreativeImage :id="asset.id" /></div>
              <h3>#{{ asset.id }} {{ asset.title || '未命名图片' }}</h3>
              <p>
                {{ asset.public ? '公开' : '私人' }} · {{ asset.archived ? '已归档' : '可用' }} ·
                {{ asset.license || '尚未填写授权' }}
              </p>
              <div class="creative-actions">
                <a-button @click="editAsset(asset)">编辑 / 发布 / 归档</a-button
                ><a-button status="danger" @click="deleteAsset(asset.id)">删除图片</a-button>
              </div>
            </article>
          </div>
        </details></template
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
          <div class="creative-actions">
            <a-button @click="editResource(row)">编辑 / 预览</a-button
            ><a-button status="danger" @click="deleteResource(row)">删除</a-button>
          </div>
        </div></template
      >
      <a-modal
        :visible="!!previewAsset"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        :title="previewAsset?.title"
        :width="900"
        :footer="false"
        @cancel="previewAsset = null"
        ><div v-if="previewAsset" class="admin-asset-preview">
          <CreativeImage :id="previewAsset.id" full /></div
      ></a-modal>
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
          :kind="editingResource?.kind ?? kind"
          :assets="assets"
          :resources="resources"
          @saved="saved"
          @assets-changed="assets = $event"
      /></a-modal>
      <a-modal
        v-model:visible="assetOpen"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        title="设定图信息"
        :footer="false"
        ><template v-if="editingAsset"
          ><div v-if="assetError" class="creative-error" role="alert">{{ assetError }}</div>
          <label class="creative-field"
            >标题<a-input v-model="editingAsset.title" :max-length="150" /></label
          ><label class="creative-field"
            >描述<a-textarea v-model="editingAsset.description" :max-length="3000" /></label
          ><label class="creative-field"
            >用途<a-input
              v-model="editingAsset.purpose"
              :max-length="80"
              placeholder="标准设定、服装、表情、场景…" /></label
          ><label class="creative-field"
            >来源与授权{{ editingAsset.public ? '（公开素材必选）' : ''
            }}<select v-model="assetLicenseChoice">
              <option value="">请选择（私人图片可留空）</option>
              <option
                v-for="option in creativeLicenseOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
              <option value="other">其他授权说明</option></select
            ><a-textarea
              v-if="assetLicenseChoice === 'other'"
              v-model="editingAsset.license"
              :max-length="300"
              placeholder="说明素材来源及允许使用的范围" /></label
          ><a-checkbox
            v-model="editingAsset.public"
            @change="
              (checked) => {
                if (checked && editingAsset) editingAsset.download_allowed = true
              }
            "
            >公开素材（允许生成、二创及下载）</a-checkbox
          >
          <p class="creative-note">
            只有自己上传的设定图可设为公开素材，并需填写来源与授权说明。生成结果请在 AI
            工作台发布到站内。
          </p>
          <a-checkbox v-model="editingAsset.archived">归档</a-checkbox
          ><a-button type="primary" :loading="assetSaving" @click="saveAsset"
            >保存</a-button
          ></template
        ></a-modal
      >
      <a-modal
        v-model:visible="resolveOpen"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        title="核查结果不明的任务"
        :footer="false"
        :mask-closable="!resolveBusy"
        :esc-to-close="!resolveBusy"
        :closable="!resolveBusy"
        ><p>先在供应商侧核查实际执行与计费情况，再释放或结算预留额度。此操作不会重新请求生图。</p>
        <a-checkbox v-model="consumed" :disabled="resolveBusy"
          >已确认产生供应商费用，结算预留额度</a-checkbox
        ><label class="creative-field"
          >核查依据与说明（必填，3–300 个字）<a-textarea
            v-model="resolveNote"
            :textarea-attrs="{ 'aria-label': '核查依据与说明', 'aria-invalid': !!resolveError }"
            :disabled="resolveBusy"
            :auto-size="{ minRows: 3, maxRows: 8 }"
            placeholder="请填写供应商侧核查到的执行与计费情况"
          /><span class="creative-note"
            >{{ Array.from(resolveNote.trim()).length }} / 300 字</span
          ></label
        >
        <p v-if="resolveError" role="alert" class="creative-error">{{ resolveError }}</p>
        <a-button type="primary" :loading="resolveBusy" @click="resolve">{{
          consumed ? '确认结算' : '确认未消耗，释放预留'
        }}</a-button></a-modal
      >
    </div>
  </AdminPanel>
</template>
<style scoped>
.admin-ai-body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.admin-ai-body > .creative-note {
  margin: -8px 0 0;
}
.admin-ai-body > .arco-btn {
  align-self: flex-start;
}
.resource-advanced {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--creative-line);
}
.resource-advanced summary {
  color: var(--creative-muted);
  cursor: pointer;
  margin-bottom: 16px;
}
.grant-fields {
  display: grid;
  grid-template-columns: minmax(140px, 1fr) minmax(140px, 1fr) minmax(180px, 1.5fr) auto;
  align-items: start;
  gap: 16px;
}
.grant-fields .creative-field {
  margin: 12px 0 0;
}
.grant-fields > .arco-btn {
  margin-top: 36px;
}
.grant-fields :deep(.arco-input-number) {
  width: 100%;
}
@media (max-width: 1100px) {
  .grant-fields {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .grant-fields > .arco-btn {
    align-self: end;
    margin: 0 0 0 auto;
  }
}
.admin-ai-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 0;
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
  margin-top: 0;
  border: 1px solid var(--creative-line);
  border-radius: 16px;
  background: var(--creative-paper);
}
.admin-ai-row > div:first-child {
  min-width: 0;
  flex: 1;
}
.admin-ai-row p {
  margin: 8px 0;
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
  flex-wrap: wrap;
  margin-top: 12px;
}
.admin-result-images > * {
  width: 100px;
  height: 100px;
}
.creative-actions > .arco-input-wrapper {
  max-width: 280px;
}
.creative-actions :deep(.arco-select-view) {
  width: 180px;
  max-width: 100%;
}
.creative-grid {
  margin-top: 20px;
}
@media (max-width: 650px) {
  .grant-fields {
    grid-template-columns: minmax(0, 1fr);
  }
  .grant-fields > .arco-btn {
    width: 100%;
    margin-top: 8px;
  }
  .admin-ai-body {
    padding: 16px;
  }
  .admin-ai-row {
    flex-direction: column;
  }
}
</style>
