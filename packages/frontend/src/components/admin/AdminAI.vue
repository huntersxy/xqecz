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
const ideaStatusLabels: Record<string, string> = {
  waiting: '等待审核',
  generating: '生成中',
  realized: '已实现',
  unknown: '结果待核查',
  unavailable: '不可用',
  hidden: '已下架',
  withdrawn: '已撤回',
}
const ledgerLabels: Record<string, string> = {
  grant: '额度发放',
  reserve: '生成预留',
  settle: '生成扣减',
  release: '退回额度',
  daily_login: '每日登录',
  content_approved: '作品通过奖励',
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
/** 复制管理员诊断文本，供贴进排查记录或日志检索。 */
async function copyDiagnostic(text: string) {
  try {
    await navigator.clipboard?.writeText(text)
  } catch {
    error.value = '复制失败，请手动选择文本复制'
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
  >
    <template #actions>
      <a-button size="small" :loading="loading" @click="run(load)">刷新</a-button>
    </template>
    <div class="admin-ai-body">
      <div class="admin-ai-tabs">
        <button
          v-for="menu in menus"
          :key="menu[0]"
          :class="{ 'is-active': tab === menu[0] }"
          :aria-current="tab === menu[0] ? 'page' : undefined"
          @click="switchTab(menu[0]!)"
        >
          {{ menu[1] }}
        </button>
      </div>
      <p v-if="error" class="admin-ai-alert">{{ error }}</p>

      <!-- A. 供应商：提示词增强说明卡 + 一行一个的结构化供应商行 -->
      <template v-if="tab === 'providers'">
        <section class="admin-ai-card admin-prompt-card">
          <header class="admin-ai-card-head">
            <div class="admin-ai-card-heading">
              <h3>提示词增强</h3>
              <p>勾选后会把这段固定要求追加到用户提示词末尾，所有用户可在工作台悬浮查看。</p>
            </div>
            <a-switch v-model="promptEnhancement.enabled" size="small" aria-label="启用提示词增强" />
          </header>
          <a-textarea
            v-model="promptEnhancement.prompt"
            class="admin-prompt-input"
            :maxlength="4000"
            :auto-size="{ minRows: 3, maxRows: 8 }"
            aria-label="提示词增强内容"
            placeholder="例如：统一线条与光影要求、禁止出现的元素……"
          />
          <footer class="admin-ai-card-foot">
            <span class="admin-ai-hint">
              {{ promptEnhancement.enabled ? '当前已启用，用户在工作台可见' : '当前已停用，仅保存文本' }}
            </span>
            <a-button
              type="primary"
              size="small"
              :loading="promptEnhancementSaving"
              @click="savePromptEnhancement"
              >保存提示词</a-button
            >
          </footer>
        </section>

        <AdminDataToolbar
          :summary="`共 ${providers.length} 个供应商 · ${providers.filter((p) => p.enabled).length} 个启用`"
        >
          <template #actions>
            <a-button type="primary" size="small" @click="editProvider(null)">添加供应商</a-button>
          </template>
        </AdminDataToolbar>

        <AdminSkeleton
          v-if="loading && !providers.length"
          variant="rows"
          :rows="4"
          :columns="3"
          label="加载供应商"
        />
        <AdminEmpty
          v-else-if="!providers.length"
          title="还没有供应商"
          desc="添加图片生成供应商，填写接口地址与密钥后即可在 AI 工作台使用。"
        >
          <template #actions>
            <a-button type="primary" size="small" @click="editProvider(null)">添加供应商</a-button>
          </template>
        </AdminEmpty>
        <div v-else class="admin-ai-list adm-stagger">
          <article v-for="p in providers" :key="p.id" class="provider-row">
            <div class="provider-main">
              <div class="provider-head">
                <span class="provider-state" :class="{ 'is-on': p.enabled }" aria-hidden="true" />
                <strong class="provider-name">{{ p.name }}</strong>
                <span class="admin-ai-pill">{{ adapterLabels[p.adapter] || p.adapter }}</span>
                <span class="admin-ai-pill is-mono">{{ p.model }}</span>
                <span class="admin-ai-pill" :class="p.has_key ? 'is-quiet' : 'is-warn'">
                  {{ p.has_key ? '密钥已加密保存' : '未配置密钥' }}
                </span>
                <span v-if="p.default" class="admin-ai-pill is-accent">默认</span>
              </div>
              <p class="provider-endpoint" :title="p.endpoint">{{ p.endpoint }}</p>
            </div>
            <div class="provider-actions">
              <a-switch
                :model-value="p.enabled"
                size="small"
                :aria-label="`切换供应商 ${p.name} 的启用状态`"
                @change="
                  run(async () => {
                    const action = p.enabled ? '关闭' : '启用'
                    if (await confirm(`${action}供应商“${p.name}”？`)) {
                      await aiApi.setProviderEnabled(p.id, !p.enabled)
                      await load()
                    }
                  })
                "
              />
              <a-button size="small" @click="editProvider(p)">编辑</a-button>
              <a-button
                size="small"
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
              <a-button
                size="small"
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
              >
            </div>
          </article>
        </div>
      </template>

      <!-- B. 用户额度 / 账本：发放表单收进卡 + 紧凑账本行列表 -->
      <template v-else-if="tab === 'quota'">
        <section class="admin-ai-card admin-grant-card">
          <header class="admin-ai-card-head">
            <div class="admin-ai-card-heading">
              <h3>发放额度</h3>
              <p>选择用户或全体用户。以下数量是每位用户本次增加的可用额度；管理员生成不消耗站点额度。</p>
            </div>
            <a-radio-group
              v-model="grantForm.scope"
              :disabled="grantBusy || !!pendingGrant"
              type="button"
            >
              <a-radio value="selected">选择用户</a-radio>
              <a-radio value="all">全体用户</a-radio>
            </a-radio-group>
          </header>
          <div class="grant-fields">
            <label v-if="grantForm.scope === 'selected'" class="admin-ai-field">
              <span class="admin-ai-field-label">发给谁</span>
              <a-select
                v-model="grantForm.users"
                multiple
                allow-search
                placeholder="搜索并选择用户"
                :disabled="grantBusy || !!pendingGrant"
              >
                <a-option v-for="u in grantUsers" :key="u.id" :value="u.id">
                  {{ u.username }}（#{{ u.id }}）
                </a-option>
              </a-select>
            </label>
            <div v-else class="admin-ai-field">
              <span class="admin-ai-field-label">发给谁</span>
              <strong class="admin-ai-field-static">全体用户</strong>
              <span class="admin-ai-hint">提交时由服务器确定完整用户列表</span>
            </div>
            <label class="admin-ai-field">
              <span class="admin-ai-field-label">发放额度</span>
              <a-input-number
                v-model="grantForm.amount"
                :input-attrs="{ 'aria-label': '发放额度' }"
                :min="1"
                :max="10000"
                :precision="0"
                :disabled="grantBusy || !!pendingGrant"
              />
              <span class="admin-ai-hint">本次增加的额度，1–10000</span>
            </label>
            <label class="admin-ai-field">
              <span class="admin-ai-field-label">发放原因（选填）</span>
              <a-input
                v-model="grantForm.note"
                :input-attrs="{ 'aria-label': '发放原因' }"
                placeholder="如：活动奖励、创作支持"
                :max-length="300"
                :disabled="grantBusy || !!pendingGrant"
              />
            </label>
          </div>
          <div class="admin-grant-foot">
            <p v-if="grantSuccess" role="status" class="admin-ai-hint is-success">
              {{ grantSuccess }}
            </p>
            <p v-else-if="pendingGrant" class="admin-ai-hint">
              发放结果尚未确认，再次提交会核对上次操作，不会重复增加额度。
            </p>
            <span v-else class="admin-ai-hint">单次每位用户 1–10000 额度</span>
            <a-button type="primary" size="small" :loading="grantBusy" @click="grant">{{
              pendingGrant ? '确认上次发放' : '发放额度'
            }}</a-button>
          </div>
        </section>

        <AdminDataToolbar :summary="`共 ${ledger.length} 条记录`" />
        <AdminSkeleton
          v-if="loading && !ledger.length"
          variant="rows"
          :rows="6"
          :columns="5"
          label="加载额度记录"
        />
        <AdminEmpty
          v-else-if="!ledger.length"
          compact
          title="暂无额度记录"
          desc="用户登录、生成与管理员发放产生的记录会显示在这里。"
        />
        <ul v-else class="ledger-list">
          <li v-for="row in ledger" :key="row.id" class="ledger-row">
            <time class="ledger-time" :datetime="row.created_at">
              {{ new Date(row.created_at).toLocaleString() }}
            </time>
            <span class="ledger-user">
              {{
                grantUsers.find((user) => user.id === row.user_id)?.username ||
                `用户 #${row.user_id}`
              }}
            </span>
            <span class="ledger-kind">{{ ledgerLabels[row.kind] || row.kind }}</span>
            <span
              class="ledger-amount"
              :class="{ 'is-plus': row.amount > 0, 'is-minus': row.amount < 0 }"
              >{{ row.amount > 0 ? `+${row.amount}` : row.amount }}</span
            >
            <span class="ledger-note" :title="row.note">
              {{ row.note || '—' }}
              <span v-if="row.task_id" class="ledger-task">任务 #{{ row.task_id }}</span>
            </span>
          </li>
        </ul>
      </template>

      <!-- C/D. 生图任务 / 赞助记录：卡头一行 + 卡身两列（正文 / 图集） -->
      <template v-else-if="tab === 'tasks' || tab === 'sponsors'">
        <AdminDataToolbar
          :summary="
            search || status
              ? `结果 ${shownTasks.length} · 共 ${tasks.length}`
              : `共 ${tasks.length} 条${tab === 'sponsors' ? '赞助记录' : '生成任务'}`
          "
        >
          <template #actions>
            <a-input
              v-model="search"
              class="admin-ai-search"
              size="small"
              :placeholder="tab === 'sponsors' ? '搜索赞助记录或用户' : '搜索提示词或用户'"
              @press-enter="run(() => queryRows())"
            />
            <a-select
              v-model="status"
              class="admin-ai-filter"
              size="small"
              placeholder="全部生成状态"
              allow-clear
            >
              <a-option
                v-for="s in ['queued', 'running', 'succeeded', 'failed', 'unknown', 'cancelled']"
                :key="s"
                :value="s"
                >{{ taskLabels[s] }}</a-option
              >
            </a-select>
            <a-button size="small" @click="run(() => queryRows())">搜索</a-button>
          </template>
        </AdminDataToolbar>

        <AdminSkeleton
          v-if="loading && !tasks.length"
          variant="rows"
          :rows="3"
          :columns="4"
          label="加载生成任务"
        />
        <AdminEmpty
          v-else-if="!shownTasks.length"
          :title="
            tasks.length
              ? tab === 'sponsors'
                ? '没有匹配的赞助记录'
                : '没有匹配的任务'
              : tab === 'sponsors'
                ? '暂无赞助记录'
                : '暂无生成任务'
          "
          :desc="
            tasks.length
              ? '换个关键词或状态筛选再试。'
              : tab === 'sponsors'
                ? '用户赞助创意后会在这里生成对应任务与核查记录。'
                : '用户在 AI 工作台提交生图后，任务会出现在这里。'
          "
        />
        <div v-else class="admin-task-list adm-stagger">
          <article v-for="row in shownTasks" :key="row.task.id" class="admin-task-record">
            <header class="admin-task-header">
              <div class="admin-task-identity">
                <strong class="admin-task-id">#{{ row.task.id }}</strong>
                <AdminStatus
                  :type="
                    row.task.status === 'succeeded'
                      ? 'success'
                      : row.task.status === 'failed'
                        ? 'danger'
                        : row.task.status === 'unknown'
                          ? 'warning'
                          : row.task.status === 'running'
                            ? 'info'
                            : 'neutral'
                  "
                  :label="taskLabels[row.task.status] || row.task.status"
                />
                <span class="admin-task-author">{{
                  row.user?.username ||
                  grantUsers.find((user) => user.id === row.task.user_id)?.username ||
                  '已注销用户'
                }}</span>
              </div>
              <time class="admin-task-time" :datetime="row.task.created_at">{{
                new Date(row.task.created_at).toLocaleString()
              }}</time>
            </header>
            <p v-if="row.task.error" class="admin-ai-alert is-warning" role="alert">
              {{ row.task.error }}
            </p>
            <div class="admin-task-body">
              <div class="admin-task-main">
                <div class="admin-task-meta">
                  <span>{{ row.task.model }}</span>
                  <span>{{ row.task.size }} · {{ row.task.count }} 张</span>
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
                <details v-if="row.diagnostic" class="admin-task-diagnostic">
                  <summary>排查诊断 · 仅管理员可见</summary>
                  <pre>{{ row.diagnostic }}</pre>
                  <a-button size="mini" @click="copyDiagnostic(row.diagnostic)">复制诊断</a-button>
                </details>
                <div class="admin-task-actions">
                  <a-button
                    v-if="row.task.status === 'unknown'"
                    size="small"
                    @click="startResolve(row.task.id)"
                    >核查结算</a-button
                  >
                  <a-button size="small" status="danger" @click="deleteTask(row.task.id)"
                    >删除任务及结果</a-button
                  >
                </div>
              </div>
              <div
                v-if="row.assets.length || row.references?.length"
                class="admin-task-side"
              >
                <section v-if="row.assets.length || row.task.status === 'succeeded'" class="admin-task-gallery-block">
                  <h3 class="admin-task-eyebrow">
                    结果<span v-if="row.assets.length">{{ row.assets.length }}</span>
                  </h3>
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
                    </button>
                  </div>
                  <p v-else class="admin-task-no-results">
                    {{ row.task.status === 'succeeded' ? '暂无可展示的结果图片' : '尚无生成结果' }}
                  </p>
                </section>
                <aside
                  v-if="row.references?.length"
                  class="admin-task-references"
                  aria-label="原始参考图"
                >
                  <h3 class="admin-task-eyebrow">
                    参考图<span>{{ row.references.length }}</span>
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
                      <CreativeImage :id="asset.id" />
                    </button>
                  </div>
                </aside>
              </div>
            </div>
          </article>
        </div>
        <div v-if="moreRows" class="admin-ai-more">
          <a-button size="small" :loading="loading" @click="run(() => queryRows(true))"
            >加载更早的任务</a-button
          >
        </div>
      </template>

      <!-- E. 创意投稿：左竖图 + 右侧正文 + 底部操作 -->
      <template v-else-if="tab === 'ideas'">
        <AdminDataToolbar :summary="`共 ${ideas.length} 条投稿`">
          <template #actions>
            <a-input
              v-model="search"
              class="admin-ai-search"
              size="small"
              placeholder="搜索投稿"
              @press-enter="run(() => queryRows())"
            />
            <a-select
              v-model="status"
              class="admin-ai-filter"
              size="small"
              placeholder="全部状态"
              allow-clear
            >
              <a-option v-for="(label, value) in ideaStatusLabels" :key="value" :value="value">
                {{ label }}</a-option
              >
            </a-select>
            <a-button size="small" @click="run(() => queryRows())">搜索投稿</a-button>
          </template>
        </AdminDataToolbar>

        <AdminSkeleton
          v-if="loading && !ideas.length"
          variant="rows"
          :rows="4"
          :columns="4"
          label="加载创意投稿"
        />
        <AdminEmpty
          v-else-if="!ideas.length"
          title="暂无创意投稿"
          desc="用户提交创意并通过审核后，会在这里看到投稿、参考图与赞助状态。"
        />
        <div v-else class="idea-admin-list adm-stagger">
          <article v-for="row in ideas" :key="row.idea.id" class="idea-admin-card">
            <div class="idea-admin-media">
              <CreativeImage v-if="row.references.length" :id="row.references[0]?.id ?? 0" />
              <span v-else class="idea-admin-media-empty">暂无参考图</span>
            </div>
            <div class="idea-admin-main">
              <div class="idea-admin-head">
                <h3 class="idea-admin-title">{{ row.idea.title || '未命名投稿' }}</h3>
                <span
                  class="admin-ai-pill"
                  :class="row.idea.status === 'unknown' ? 'is-warn' : ''"
                  >{{ ideaStatusLabels[row.idea.status] || row.idea.status }}</span
                >
                <span class="admin-ai-pill is-accent">{{ row.current_cost }} 额度</span>
              </div>
              <p class="idea-admin-desc">{{ row.idea.description || '暂无说明' }}</p>
              <details class="idea-admin-snapshot">
                <summary>提示词与参数</summary>
                <pre>{{ row.idea.snapshot }}</pre>
              </details>
              <div class="idea-admin-meta">
                <span>作者 {{ row.author.username }}</span>
                <span>{{ row.sponsor?.username ? `赞助 ${row.sponsor.username}` : '等待赞助' }}</span>
                <time :datetime="row.idea.created_at">{{ new Date(row.idea.created_at).toLocaleString() }}</time>
              </div>
            </div>
            <div class="idea-admin-actions">
              <RouterLink class="idea-admin-link" :to="`/ideas/${row.idea.id}`">预览</RouterLink>
              <a-button size="small" @click="showEvents(row.idea.id)">操作与核查记录</a-button>
              <a-button
                v-if="row.sponsorable"
                size="small"
                @click="
                  run(async () => {
                    if (await confirm(`赞助此创意，预留 ${row.current_cost} 额度？`)) {
                      await aiApi.sponsor(row.idea.id, row.current_cost, newKey())
                      await load()
                    }
                  })
                "
                >赞助</a-button
              >
              <a-button
                size="small"
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
              <a-button size="small" status="danger" @click="deleteIdea(row.idea.id)"
                >删除</a-button
              >
            </div>
          </article>
        </div>
        <div v-if="moreRows" class="admin-ai-more">
          <a-button size="small" :loading="loading" @click="run(() => queryRows(true))"
            >加载更早的投稿</a-button
          >
        </div>
      </template>

      <!-- F. 角色资源：4:3 封面卡片栅格 + 未归入角色的图片折叠区 -->
      <template v-else-if="tab === 'character'">
        <AdminDataToolbar :summary="`共 ${shownResources.length} 个角色资源`">
          <template #actions>
            <a-input
              v-model="search"
              class="admin-ai-search"
              size="small"
              placeholder="搜索角色资源"
              @press-enter="run(() => queryRows())"
            />
            <a-button type="primary" size="small" @click="editResource(null)">新建角色资源</a-button>
          </template>
        </AdminDataToolbar>
        <p class="admin-ai-hint">
          一个角色资源包含名称、外观说明和参考图片。在同一处管理，生成时一起导入。
        </p>

        <AdminSkeleton
          v-if="loading && !shownResources.length"
          variant="cards"
          :rows="4"
          label="加载角色资源"
        />
        <AdminEmpty
          v-else-if="!shownResources.length"
          title="还没有角色资源"
          desc="添加角色名称与参考图片，生成时即可一起导入。"
        >
          <template #actions>
            <a-button type="primary" size="small" @click="editResource(null)">新建角色资源</a-button>
          </template>
        </AdminEmpty>
        <div v-else class="adm-card-grid resource-grid">
          <article v-for="row in shownResources" :key="row.id" class="adm-card">
            <div class="adm-card-media resource-cover">
              <CreativeImage v-if="row.payload.asset_ids.length" :id="row.payload.asset_ids[0] ?? 0" />
              <span v-else class="resource-cover-empty">暂无参考图</span>
            </div>
            <div class="adm-card-body">
              <h3 class="adm-card-title">{{ row.title || '未命名角色' }}</h3>
              <p class="resource-summary">{{ row.summary || '暂无简介' }}</p>
              <div class="adm-card-meta">
                <span>{{ row.payload.asset_ids.length }} 张参考图</span>
                <span>{{ row.status === 'published' ? '已公开' : '未公开' }}</span>
              </div>
            </div>
            <div class="adm-card-actions">
              <a-button size="small" @click="editResource(row)">管理角色与图片</a-button>
              <a-button size="small" status="danger" @click="deleteResource(row)">删除</a-button>
            </div>
          </article>
        </div>

        <details class="resource-advanced">
          <summary>未归入角色的图片（{{ unassignedAssets.length }}）</summary>
          <a-button size="small" @click="fileInput?.click()">单独上传图片</a-button
          ><input
            ref="fileInput"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            hidden
            @change="upload"
          />
          <p class="admin-ai-hint">文件不可原地替换。修改图像请上传新版本，旧图与历史引用继续保留。</p>
          <div class="admin-asset-grid">
            <article v-for="asset in unassignedAssets" :key="asset.id" class="admin-asset-row">
              <div class="admin-asset-thumb"><CreativeImage :id="asset.id" /></div>
              <h3 class="admin-asset-name">#{{ asset.id }} {{ asset.title || '未命名图片' }}</h3>
              <p class="admin-asset-meta">
                {{ asset.public ? '公开' : '私人' }} · {{ asset.archived ? '已归档' : '可用' }} ·
                {{ asset.license || '尚未填写授权' }}
              </p>
              <div class="admin-asset-actions">
                <a-button size="small" @click="editAsset(asset)">编辑 / 发布 / 归档</a-button>
                <a-button size="small" status="danger" @click="deleteAsset(asset.id)">删除图片</a-button>
              </div>
            </article>
          </div>
        </details>
      </template>

      <!-- G. 提示词模板：行卡（标题 + 胶囊 + 操作） -->
      <template v-else>
        <AdminDataToolbar :summary="`共 ${shownResources.length} 个提示词模板`">
          <template #actions>
            <a-input
              v-model="search"
              class="admin-ai-search"
              size="small"
              placeholder="搜索提示词模板"
              @press-enter="run(() => queryRows())"
            />
            <a-select
              v-model="status"
              class="admin-ai-filter"
              size="small"
              placeholder="全部状态"
              allow-clear
            >
              <a-option value="draft">草稿</a-option>
              <a-option value="published">发布</a-option>
              <a-option value="archived">归档</a-option>
            </a-select>
            <a-button size="small" :disabled="!selected.length" @click="archiveSelected"
              >批量归档</a-button
            >
            <a-button type="primary" size="small" @click="editResource(null)">新建提示词</a-button>
          </template>
        </AdminDataToolbar>

        <AdminSkeleton
          v-if="loading && !shownResources.length"
          variant="rows"
          :rows="5"
          :columns="3"
          label="加载提示词模板"
        />
        <AdminEmpty
          v-else-if="!shownResources.length"
          title="还没有提示词模板"
          desc="把常用提示词整理成模板，生成时可以直接选用。"
        >
          <template #actions>
            <a-button type="primary" size="small" @click="editResource(null)">新建提示词</a-button>
          </template>
        </AdminEmpty>
        <div v-else class="prompt-list adm-stagger">
          <article v-for="row in shownResources" :key="row.id" class="prompt-row">
            <div class="prompt-main">
              <a-checkbox
                class="prompt-check"
                :model-value="selected.includes(row.id)"
                @change="
                  (checked) =>
                    (selected = checked
                      ? [...selected, row.id]
                      : selected.filter((id) => id !== row.id))
                "
                >{{ row.title || '未命名模板' }}</a-checkbox
              >
              <div class="prompt-pills">
                <span class="admin-ai-pill">版本 {{ row.version }}</span>
                <span class="admin-ai-pill">排序 {{ row.sort_order }}</span>
                <span class="admin-ai-pill" :class="row.status === 'draft' ? 'is-warn' : ''">
                  {{ row.status === 'draft' ? '草稿' : row.status === 'published' ? '已发布' : '已归档' }}
                </span>
              </div>
              <p class="prompt-summary" :title="row.summary || ''">{{ row.summary || '暂无摘要' }}</p>
            </div>
            <div class="prompt-actions">
              <a-button size="small" @click="editResource(row)">编辑 / 预览</a-button>
              <a-button size="small" status="danger" @click="deleteResource(row)">删除</a-button>
            </div>
          </article>
        </div>
      </template>

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
          ><div v-if="assetError" class="admin-ai-alert" role="alert">{{ assetError }}</div>
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
        <p v-if="resolveError" role="alert" class="admin-ai-alert">{{ resolveError }}</p>
        <a-button type="primary" :loading="resolveBusy" @click="resolve">{{
          consumed ? '确认结算' : '确认未消耗，释放预留'
        }}</a-button></a-modal
      >
    </div>
  </AdminPanel>
</template>
<style scoped>
.admin-ai-body {
  padding: var(--adm-gap-xl, 24px);
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-lg, 16px);
  min-width: 0;
}
/* 工具条在 body 内不再自带上下留白，靠 body 的 gap 控制节奏 */
.admin-ai-body :deep(.admin-data-toolbar) {
  padding: 0;
}
.admin-ai-body :deep(.admin-data-toolbar-actions) {
  min-width: 0;
}

/* ── 页签：沿用上一轮的胶囊分段控件 ── */
.admin-ai-tabs {
  display: flex;
  gap: var(--adm-gap-sm, 8px);
  flex-wrap: wrap;
  padding-bottom: var(--adm-gap-md, 12px);
  border-bottom: 1px solid var(--adm-line, var(--creative-line));
}
.admin-ai-tabs button {
  min-height: var(--adm-touch, 36px);
  padding: 0 14px;
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-surface, var(--creative-paper));
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-sm, 13px);
  cursor: pointer;
  transition:
    background var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    transform var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
.admin-ai-tabs button:hover:not(:disabled) {
  border-color: var(--adm-accent, var(--creative-accent));
  background: var(--adm-soft, var(--creative-soft));
  color: var(--adm-accent, var(--creative-accent));
}
.admin-ai-tabs button:focus-visible {
  outline: 2px solid var(--adm-accent, var(--creative-accent));
  outline-offset: 2px;
}
.admin-ai-tabs button:active:not(:disabled) {
  transform: scale(.98);
}
.admin-ai-tabs button.is-active {
  border-color: var(--adm-accent, var(--creative-accent));
  background: var(--adm-accent-soft, var(--creative-soft));
  color: var(--adm-accent, var(--creative-accent));
}

/* ── 通用块：提示条 / 卡片 / 胶囊 / 提示文字 ── */
.admin-ai-alert {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--adm-danger) 30%, transparent);
  border-radius: var(--adm-r-control, 10px);
  background: color-mix(in srgb, var(--adm-danger) 8%, var(--adm-surface, var(--creative-paper)));
  color: color-mix(in srgb, var(--adm-danger) 70%, var(--adm-ink, var(--creative-ink)));
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.65;
  overflow-wrap: anywhere;
}
.admin-ai-alert.is-warning {
  border-color: color-mix(in srgb, var(--adm-warning) 30%, transparent);
  background: color-mix(in srgb, var(--adm-warning) 10%, var(--adm-surface, var(--creative-paper)));
  color: color-mix(in srgb, var(--adm-warning) 68%, var(--adm-ink, var(--creative-ink)));
}

.admin-ai-card {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
  padding: var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-surface, var(--creative-paper));
}
.admin-ai-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--adm-gap-lg, 16px);
  flex-wrap: wrap;
}
.admin-ai-card-heading {
  min-width: 0;
  flex: 1;
}
.admin-ai-card-heading h3 {
  margin: 0;
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  line-height: 1.4;
}
.admin-ai-card-heading p {
  max-width: 68ch;
  margin: var(--adm-gap-xs, 4px) 0 0;
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-base, 14px);
  line-height: 1.65;
}
.admin-ai-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
}
.admin-ai-hint {
  margin: 0;
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
}
.admin-ai-hint.is-success {
  color: color-mix(in srgb, var(--adm-success) 78%, var(--adm-ink, var(--creative-ink)));
}

.admin-ai-pill {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--creative-soft));
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  white-space: nowrap;
}
.admin-ai-pill.is-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: -.01em;
}
.admin-ai-pill.is-quiet {
  background: transparent;
  border-color: var(--adm-line, var(--creative-line));
}
.admin-ai-pill.is-warn {
  background: color-mix(in srgb, var(--adm-warning) 12%, transparent);
  color: color-mix(in srgb, var(--adm-warning) 70%, var(--adm-ink, var(--creative-ink)));
}
.admin-ai-pill.is-accent {
  background: var(--adm-accent-soft, var(--creative-soft));
  color: var(--adm-accent, var(--creative-accent));
}

.admin-ai-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.admin-ai-more {
  display: flex;
  justify-content: center;
}

/* ── A. 提示词增强说明卡 ── */
.admin-prompt-card {
  gap: var(--adm-gap-md, 12px);
}
.admin-prompt-input :deep(.arco-textarea) {
  min-height: 96px;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.7;
}

/* ── A. 供应商行卡 ── */
.provider-row {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-lg, 16px);
  min-width: 0;
  padding: 14px var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-surface, var(--creative-paper));
  transition: border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
@media (hover: hover) and (pointer: fine) {
  .provider-row:hover {
    border-color: color-mix(in srgb, var(--adm-accent, var(--creative-accent)) 45%, var(--adm-line, var(--creative-line)));
  }
}
.provider-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.provider-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--adm-gap-sm, 8px);
  min-width: 0;
}
.provider-state {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-muted, var(--creative-muted));
  opacity: .45;
}
.provider-state.is-on {
  opacity: 1;
  background: var(--adm-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--adm-success) 18%, transparent);
}
.provider-name {
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  line-height: 1.4;
}
.provider-endpoint {
  margin: 0;
  overflow: hidden;
  color: var(--adm-muted, var(--creative-muted));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.5;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.provider-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  flex: none;
  gap: var(--adm-gap-sm, 8px);
}

/* ── B. 发放额度 ── */
.admin-grant-card {
  gap: var(--adm-gap-lg, 16px);
}
.grant-fields {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(120px, .55fr) minmax(0, 1fr);
  align-items: start;
  gap: var(--adm-gap-lg, 16px);
}
.admin-ai-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.admin-ai-field-label {
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.5;
}
.admin-ai-field-static {
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
}
.grant-fields :deep(.arco-select),
.grant-fields :deep(.arco-input-number),
.grant-fields :deep(.arco-input-wrapper) {
  width: 100%;
}
.admin-grant-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
}

/* ── B. 账本紧凑行 ── */
.ledger-list {
  margin: 0;
  padding: 0;
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-surface, var(--creative-paper));
  list-style: none;
  overflow: hidden;
}
.ledger-row {
  display: grid;
  grid-template-columns: minmax(128px, 160px) minmax(72px, 120px) 92px 76px minmax(0, 1fr);
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  min-height: 44px;
  padding: 0 var(--adm-gap-lg, 16px);
  border-bottom: 1px solid var(--adm-line, var(--creative-line));
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-sm, 13px);
}
.ledger-row:last-child {
  border-bottom: 0;
}
.ledger-time {
  color: var(--adm-muted, var(--creative-muted));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}
.ledger-user {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.ledger-kind {
  justify-self: start;
  padding: 1px 8px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--creative-soft));
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
  white-space: nowrap;
}
.ledger-amount {
  justify-self: end;
  color: var(--adm-muted, var(--creative-muted));
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  font-weight: 600;
  text-align: right;
}
.ledger-amount.is-plus {
  color: var(--adm-success);
}
.ledger-amount.is-minus {
  color: var(--adm-danger);
}
.ledger-note {
  overflow: hidden;
  color: var(--adm-muted, var(--creative-muted));
  white-space: nowrap;
  text-overflow: ellipsis;
}
.ledger-task {
  margin-left: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-xs, 12px);
  opacity: .85;
}

/* ── E. 创意投稿横向卡 ── */
.idea-admin-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.idea-admin-card {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
  padding: var(--adm-gap-md, 12px);
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-surface, var(--creative-paper));
  transition: border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
@media (hover: hover) and (pointer: fine) {
  .idea-admin-card:hover {
    border-color: color-mix(in srgb, var(--adm-accent, var(--creative-accent)) 45%, var(--adm-line, var(--creative-line)));
  }
}
.idea-admin-media {
  flex: none;
  width: 88px;
  height: 110px;
  overflow: hidden;
  border-radius: 10px;
  background: var(--adm-canvas, var(--creative-canvas));
}
.idea-admin-media :deep(.creative-image),
.idea-admin-media :deep(.creative-image img) {
  width: 100%;
  height: 100%;
  min-height: 0;
  object-fit: cover;
}
.idea-admin-media-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
}
.idea-admin-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.idea-admin-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--adm-gap-sm, 8px);
}
.idea-admin-title {
  margin: 0;
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  line-height: 1.45;
  overflow-wrap: anywhere;
}
.idea-admin-desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin: 0;
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.65;
  overflow-wrap: anywhere;
}
.idea-admin-snapshot summary {
  color: var(--adm-accent, var(--creative-accent));
  font-size: var(--adm-font-xs, 12px);
  cursor: pointer;
}
.idea-admin-snapshot pre {
  max-height: 180px;
  margin: 6px 0 0;
  overflow: auto;
  color: var(--adm-muted, var(--creative-muted));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.65;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.idea-admin-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px var(--adm-gap-md, 12px);
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}
/* 投稿卡的操作按钮排成底部一行，不再在右侧竖着堆四五个 */
.idea-admin-actions {
  display: flex;
  flex: 1 1 100%;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: var(--adm-gap-sm, 8px);
}
.idea-admin-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-surface, var(--creative-paper));
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-sm, 13px);
  text-decoration: none;
  transition:
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
.idea-admin-link:hover {
  border-color: var(--adm-accent, var(--creative-accent));
  color: var(--adm-accent, var(--creative-accent));
}
.idea-admin-link:focus-visible {
  outline: 2px solid var(--adm-accent, var(--creative-accent));
  outline-offset: 2px;
}

/* ── F. 角色资源卡片 / 未归入角色的图片 ── */
.resource-grid {
  min-width: 0;
}
.resource-cover {
  display: flex;
  align-items: center;
  justify-content: center;
  /* 角色立绘多是竖版，4:3 会把头脚裁掉，改用 4:5 并偏上取景 */
  aspect-ratio: 4 / 5;
}
.resource-cover :deep(.creative-image),
.resource-cover :deep(.creative-image img) {
  width: 100%;
  height: 100%;
  min-height: 0;
  object-fit: cover;
  object-position: center 22%;
}
.resource-cover-empty {
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
}
.resource-summary {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin: 0;
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.resource-advanced {
  margin-top: var(--adm-gap-sm, 8px);
  padding-top: var(--adm-gap-lg, 16px);
  border-top: 1px solid var(--adm-line, var(--creative-line));
}
.resource-advanced > summary {
  margin-bottom: var(--adm-gap-md, 12px);
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-sm, 13px);
  cursor: pointer;
}
.resource-advanced > summary:focus-visible {
  outline: 2px solid var(--adm-accent, var(--creative-accent));
  outline-offset: 3px;
}
.admin-asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--adm-gap-md, 12px);
  margin-top: var(--adm-gap-md, 12px);
}
.admin-asset-row {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);
  min-width: 0;
  padding: var(--adm-gap-md, 12px);
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-surface, var(--creative-paper));
}
.admin-asset-thumb {
  aspect-ratio: 1 / 1;
  overflow: hidden;
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-canvas, var(--creative-canvas));
}
.admin-asset-thumb :deep(.creative-image),
.admin-asset-thumb :deep(.creative-image img) {
  width: 100%;
  height: 100%;
  min-height: 0;
  object-fit: cover;
}
.admin-asset-name {
  margin: 0;
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  overflow-wrap: anywhere;
}
.admin-asset-meta {
  margin: 0;
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
}
.admin-asset-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-sm, 8px);
  margin-top: auto;
}

/* ── G. 提示词模板行卡 ── */
.prompt-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.prompt-row {
  display: flex;
  align-items: flex-start;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-surface, var(--creative-paper));
  transition: border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
@media (hover: hover) and (pointer: fine) {
  .prompt-row:hover {
    border-color: color-mix(in srgb, var(--adm-accent, var(--creative-accent)) 45%, var(--adm-line, var(--creative-line)));
  }
}
.prompt-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.prompt-check :deep(.arco-checkbox-label) {
  color: var(--adm-ink, var(--creative-ink));
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  line-height: 1.45;
}
.prompt-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.prompt-summary {
  margin: 0;
  overflow: hidden;
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.prompt-actions {
  display: flex;
  flex: none;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  margin-left: auto;
}

/* ── 工具条：搜索、筛选、按钮排成一行 ──
   Arco 的 Select 根节点不继承 scoped 作用域属性，宽度只能走 :deep。 */
.admin-ai-body :deep(.admin-data-toolbar-actions) {
  flex: 1 1 auto;
  flex-wrap: nowrap;
  justify-content: flex-end;
  gap: var(--adm-gap-sm, 8px);
  min-width: 0;
}
.admin-ai-body :deep(.admin-ai-search) {
  flex: 0 1 220px;
  width: 220px;
}
.admin-ai-body :deep(.admin-ai-filter) {
  flex: 0 0 160px;
  width: 160px;
}
/* 输入框、下拉、按钮等高，否则 38px 的 Select 会把工具条撑成三层 */
.admin-ai-body
  :deep(.admin-data-toolbar-actions :is(.arco-input-wrapper, .arco-select-view-single)) {
  height: 32px;
  min-height: 32px;
  font-size: var(--adm-font-sm, 13px);
}

@media (max-width: 820px) {
  .provider-row {
    flex-direction: column;
    align-items: stretch;
    gap: var(--adm-gap-md, 12px);
  }
  .provider-actions {
    justify-content: flex-start;
  }
  .grant-fields {
    grid-template-columns: minmax(0, 1fr);
  }
  .admin-grant-foot .arco-btn {
    width: 100%;
  }
}

@media (max-width: 768px) {
  .admin-ai-body {
    padding: var(--adm-gap-lg, 16px);
  }
  /* 手机档页签横向滚动，避免 7 个入口堆成三行挤压内容 */
  .admin-ai-tabs {
    flex-wrap: nowrap;
    overflow-x: auto;
    padding: 2px 0 var(--adm-gap-sm, 8px);
    scrollbar-width: thin;
  }
  .admin-ai-tabs button {
    flex: 0 0 auto;
  }
  .ledger-list {
    border-radius: var(--adm-r-card, 16px);
  }
  .ledger-row {
    display: flex;
    flex-wrap: wrap;
    gap: 4px var(--adm-gap-md, 12px);
    padding: 10px var(--adm-gap-md, 12px);
  }
  .ledger-time {
    flex-basis: 100%;
  }
  .ledger-user {
    flex: 1;
    min-width: 0;
  }
  .ledger-amount {
    justify-self: auto;
  }
  .ledger-note {
    flex-basis: 100%;
    white-space: normal;
  }
  .idea-admin-card {
    flex-wrap: wrap;
    padding: var(--adm-gap-sm, 8px);
  }
  .idea-admin-media {
    width: 72px;
    height: 90px;
  }
  .idea-admin-actions {
    flex-direction: row;
    flex-wrap: wrap;
    width: 100%;
  }
  .prompt-row {
    flex-wrap: wrap;
  }
  .prompt-actions {
    width: 100%;
    margin-left: 0;
  }
  .admin-ai-body :deep(.admin-ai-search),
  .admin-ai-body :deep(.admin-ai-filter) {
    flex: 1 1 100%;
    width: 100%;
  }
  .admin-ai-body :deep(.admin-data-toolbar-actions) {
    width: 100%;
    margin-left: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .admin-ai-tabs button,
  .provider-row,
  .idea-admin-card,
  .idea-admin-link,
  .prompt-row {
    transition: none;
  }
  .admin-ai-tabs button:active:not(:disabled) {
    transform: none;
  }
}
</style>
