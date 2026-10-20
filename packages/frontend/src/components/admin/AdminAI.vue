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
import type { Component } from 'vue'
import {
  IconBulb,
  IconDelete,
  IconEdit,
  IconFile,
  IconGift,
  IconImage,
  IconPlayArrow,
  IconPoweroff,
  IconSettings,
  IconThunderbolt,
  IconUser,
  IconUserGroup,
} from '@arco-design/web-vue/es/icon'
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
  grantForm = ref({ users: [] as number[], scope: 'selected', amount: 1, note: '' }),
  grantKey = ref(newKey()),
  fileInput = ref<HTMLInputElement>(),
  selected = ref<number[]>([])
const assetError = ref(''),
  assetSaving = ref(false)
const promptEnhancementSaving = ref(false)
const assetLicenseChoice = ref('')
const previewAsset = ref<{ id: number; title: string } | null>(null)
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

type AdminStatusType = 'success' | 'warning' | 'danger' | 'info' | 'neutral'
type LedgerRow = z.infer<typeof AILedgerSchema>

const menuIcons: Record<string, Component> = {
  providers: IconSettings,
  quota: IconGift,
  tasks: IconImage,
  ideas: IconBulb,
  sponsors: IconUserGroup,
  character: IconUser,
  template: IconFile,
}

const enabledProviderCount = computed(() => providers.value.filter((p) => p.enabled).length)
const failedTaskCount = computed(() => tasks.value.filter((t) => t.task.status === 'failed').length)
const sponsorCount = computed(() => tasks.value.filter((t) => t.task.idea_id > 0).length)
const characterCount = computed(() => resources.value.filter((r) => r.kind !== 'template').length)
const templateCount = computed(() => resources.value.filter((r) => r.kind === 'template').length)

const navItems = computed(() => {
  const counts: Record<string, number> = {
    providers: providers.value.length,
    quota: ledger.value.length,
    tasks: tasks.value.length,
    ideas: ideas.value.length,
    sponsors: sponsorCount.value,
    character: characterCount.value,
    template: templateCount.value,
  }
  return menus.map(([key, label]) => ({ key, label, icon: menuIcons[key], count: counts[key] ?? 0 }))
})

function dateText(iso: string) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleDateString()
}

function timeText(iso: string) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleTimeString()
}

function dayKey(iso: string) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '未知日期' : d.toLocaleDateString()
}

const ledgerGroups = computed(() => {
  const groups = new Map<string, LedgerRow[]>()
  for (const row of ledger.value) {
    const key = dayKey(row.created_at)
    const list = groups.get(key)
    if (list) list.push(row)
    else groups.set(key, [row])
  }
  return Array.from(groups, ([key, rows]) => ({ key, rows }))
})

function userName(id: number) {
  return grantUsers.value.find((u) => u.id === id)?.username || `用户 #${id}`
}

function taskAuthor(row: AITaskView) {
  return (
    row.user?.username ||
    grantUsers.value.find((u) => u.id === row.task.user_id)?.username ||
    '已注销用户'
  )
}

function taskType(status: string): AdminStatusType {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'unknown') return 'warning'
  if (status === 'running' || status === 'queued') return 'info'
  return 'neutral'
}

function resourceLabel(status: string) {
  if (status === 'published') return '已发布'
  if (status === 'archived') return '已归档'
  return '草稿'
}

function toggleSelected(id: number, checked: boolean) {
  if (checked) {
    if (!selected.value.includes(id)) selected.value = [...selected.value, id]
    return
  }
  selected.value = selected.value.filter((x) => x !== id)
}

function toggleProvider(row: AIProvider) {
  return run(async () => {
    const action = row.enabled ? '关闭' : '启用'
    if (!(await confirm(`${action}供应商“${row.name}”？`))) return
    await aiApi.setProviderEnabled(row.id, !row.enabled)
    await load()
  })
}

function testProviderConnection(row: AIProvider) {
  return run(async () => {
    if (!(await confirm('测试会请求生成一张图，可能产生供应商费用。继续？'))) return
    await aiApi.testProvider(row.id, newKey())
    tab.value = 'tasks'
    await load()
  })
}

function removeProvider(row: AIProvider) {
  return run(async () => {
    if (!(await confirm(`删除供应商“${row.name}”？删除后不可恢复。`))) return
    await aiApi.deleteProvider(row.id)
    await load()
  })
}

function sponsorIdea(row: AIIdeaView) {
  return run(async () => {
    if (!(await confirm(`赞助此创意，预留 ${row.current_cost} 额度？`))) return
    await aiApi.sponsor(row.idea.id, row.current_cost, newKey())
    await load()
  })
}

function hideIdea(id: number) {
  return run(async () => {
    if (!(await confirm('下架此创意与公开成果？'))) return
    await aiApi.hideIdea(id, '管理员下架')
    await load()
  })
}

const taskLabels: Record<string, string> = {
  queued: '等待生成',
  running: '正在生成',
  succeeded: '生成成功',
  failed: '生成失败',
  unknown: '历史待核查',
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
  unknown: '历史待核查',
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
    desc="供应商、生图任务、创意投稿与角色素材共用一套持久化任务与审计记录。"
    flow
  >
    <template #actions>
      <a-button size="small" :loading="loading" @click="run(load)">刷新</a-button>
    </template>

    <div class="ai-workspace">
      <nav class="ai-rail" role="tablist" aria-label="AI 创作管理分区">
        <div v-for="item in navItems" :key="item.key" class="ai-rail-row">
          <button
            type="button"
            class="ai-rail-item"
            :class="{ 'is-active': tab === item.key }"
            role="tab"
            :id="'ai-tab-' + item.key"
            :aria-selected="tab === item.key"
            :aria-controls="'ai-pane-' + item.key"
            :aria-current="tab === item.key ? 'page' : undefined"
            @click="switchTab(item.key)"
          >
            <span class="ai-rail-icon" aria-hidden="true"><component :is="item.icon" /></span>
            <span class="ai-rail-label">{{ item.label }}</span>
          </button>
          <span v-if="item.count > 0" class="ai-rail-count">{{
            item.count > 99 ? '99+' : item.count
          }}</span>
        </div>
      </nav>

      <div class="ai-main">
        <div class="adm-stat-grid adm-ai-stats">
          <div class="adm-stat">
            <span class="adm-stat-label">启用供应商</span>
            <strong class="adm-stat-value">{{ enabledProviderCount }}</strong>
            <span class="adm-stat-hint">共 {{ providers.length }} 个供应商</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">生图任务</span>
            <strong class="adm-stat-value">{{ tasks.length }}</strong>
            <span class="adm-stat-hint">{{ failedTaskCount }} 条失败已自动返还</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">创意投稿</span>
            <strong class="adm-stat-value">{{ ideas.length }}</strong>
            <span class="adm-stat-hint">{{ sponsorCount }} 条赞助关联</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">角色资源</span>
            <strong class="adm-stat-value">{{ characterCount }}</strong>
            <span class="adm-stat-hint">另有 {{ templateCount }} 个提示词模板</span>
          </div>
        </div>

        <p v-if="error" class="adm-ai-alert" role="alert">{{ error }}</p>

        <section id="ai-pane-providers" role="tabpanel" :aria-labelledby="'ai-tab-providers'" v-if="tab === 'providers'" class="adm-ai-pane">
          <header class="adm-ai-pane-head">
            <div>
              <h2>供应商</h2>
              <p>每张卡片是一个图像通道；密钥只在保存时写入，不回显。</p>
            </div>
            <a-button type="primary" size="small" @click="editProvider(null)">新建供应商</a-button>
          </header>

          <AdminSkeleton
            v-if="loading && !providers.length"
            variant="cards"
            :rows="2"
            :columns="3"
            label="加载供应商"
          />
          <AdminEmpty
            v-else-if="!providers.length"
            title="暂无供应商"
            desc="添加一个图像供应商后，用户就能在 AI 工作台提交生图任务。"
          >
            <template #actions>
              <a-button type="primary" size="small" @click="editProvider(null)">新建供应商</a-button>
            </template>
          </AdminEmpty>
          <div v-else class="adm-card-grid is-wide adm-stagger">
            <article v-for="p in providers" :key="p.id" class="adm-card adm-ai-provider">
              <div class="adm-card-content">
                <header class="adm-ai-provider-head">
                  <h3 class="adm-card-title">{{ p.name }}</h3>
                  <span
                    class="adm-ai-dot"
                    :class="p.enabled ? 'is-on' : 'is-off'"
                    aria-hidden="true"
                  />
                  <span class="adm-ai-provider-state">{{ p.enabled ? '启用' : '停用' }}</span>
                </header>
                <div class="adm-ai-chips">
                  <span class="adm-ai-chip">{{ adapterLabels[p.adapter] || p.adapter }}</span>
                  <span v-if="p.default" class="adm-ai-chip is-accent">默认通道</span>
                  <span v-if="!p.has_key" class="adm-ai-chip is-danger">缺少密钥</span>
                </div>
                <dl class="adm-kv">
                  <div class="adm-kv-row">
                    <dt>model</dt>
                    <dd :title="p.model">{{ p.model }}</dd>
                  </div>
                  <div class="adm-kv-row">
                    <dt>base_url</dt>
                    <dd :title="p.endpoint">{{ p.endpoint }}</dd>
                  </div>
                </dl>
              </div>
              <footer class="adm-card-foot">
                <div class="ai-icon-bar is-push">
                  <button
                    type="button"
                    class="admin-icon-btn"
                    title="编辑供应商"
                    aria-label="编辑供应商"
                    @click="editProvider(p)"
                  >
                    <IconEdit />
                  </button>
                  <button
                    type="button"
                    class="admin-icon-btn"
                    :title="p.enabled ? '停用供应商' : '启用供应商'"
                    :aria-label="p.enabled ? '停用供应商' : '启用供应商'"
                    @click="toggleProvider(p)"
                  >
                    <IconPoweroff v-if="p.enabled" />
                    <IconPlayArrow v-else />
                  </button>
                  <button
                    type="button"
                    class="admin-icon-btn"
                    title="连通测试"
                    aria-label="连通测试"
                    :disabled="!p.enabled"
                    @click="testProviderConnection(p)"
                  >
                    <IconThunderbolt />
                  </button>
                  <button
                    type="button"
                    class="admin-icon-btn is-danger"
                    title="删除供应商"
                    aria-label="删除供应商"
                    @click="removeProvider(p)"
                  >
                    <IconDelete />
                  </button>
                </div>
              </footer>
            </article>
          </div>

          <details class="adm-card adm-ai-fold" open>
            <summary class="adm-ai-fold-head">
              <span class="adm-ai-fold-title">提示词增强</span>
              <span class="adm-ai-fold-desc"
                >启用后把这段固定要求追加到用户提示词末尾，工作台里对所有人生效。</span
              >
              <span class="adm-ai-fold-state">{{
                promptEnhancement.enabled ? '已启用' : '已停用'
              }}</span>
            </summary>
            <div class="adm-ai-fold-body">
              <div class="adm-ai-field-inline">
                <a-switch
                  v-model="promptEnhancement.enabled"
                  size="small"
                  aria-label="启用提示词增强"
                />
                <span class="adm-ai-hint">{{
                  promptEnhancement.enabled ? '用户在工作台可见' : '仅保存文本，不追加到任务'
                }}</span>
              </div>
              <a-textarea
                v-model="promptEnhancement.prompt"
                :maxlength="4000"
                :auto-size="{ minRows: 3, maxRows: 8 }"
                aria-label="提示词增强内容"
                placeholder="例如：统一线条与光影要求、禁止出现的元素……"
              />
              <div class="adm-ai-fold-foot">
                <a-button
                  type="primary"
                  size="small"
                  :loading="promptEnhancementSaving"
                  @click="run(savePromptEnhancement)"
                  >保存</a-button
                >
              </div>
            </div>
          </details>
        </section>

        <section id="ai-pane-quota" role="tabpanel" :aria-labelledby="'ai-tab-quota'" v-else-if="tab === 'quota'" class="adm-board">
          <div class="adm-board-main">
            <header class="adm-ai-pane-head">
              <div>
                <h2>用户额度 / 账本</h2>
                <p>按日期倒序排列的额度流水：打卡、预留、结算与人工发放都在这里对账。</p>
              </div>
              <span class="adm-ai-hint">共 {{ ledger.length }} 条记录</span>
            </header>
            <AdminSkeleton
              v-if="loading && !ledger.length"
              variant="rows"
              :rows="6"
              :columns="3"
              label="加载额度记录"
            />
            <AdminEmpty
              v-else-if="!ledger.length"
              compact
              title="暂无额度记录"
              desc="用户登录、生成与管理员发放产生的记录会显示在这里。"
            />
            <div v-else class="adm-ai-timeline">
              <section v-for="group in ledgerGroups" :key="group.key" class="adm-ai-day">
                <h3 class="adm-ai-day-label">{{ group.key }}</h3>
                <ol class="adm-ai-day-rows">
                  <li v-for="row in group.rows" :key="row.id" class="adm-ai-entry">
                    <span class="adm-ai-node" aria-hidden="true" />
                    <div class="adm-ai-entry-body">
                      <div class="adm-ai-entry-head">
                        <strong class="adm-ai-entry-user">{{ userName(row.user_id) }}</strong>
                        <span class="adm-ai-chip">{{ ledgerLabels[row.kind] || row.kind }}</span>
                        <span
                          class="adm-ai-entry-amount"
                          :class="row.amount > 0 ? 'is-plus' : 'is-minus'"
                          >{{ row.amount > 0 ? '+' + row.amount : row.amount }}</span
                        >
                        <time class="adm-ai-entry-time" :datetime="row.created_at">{{
                          timeText(row.created_at)
                        }}</time>
                      </div>
                      <p v-if="row.note" class="adm-ai-entry-note">{{ row.note }}</p>
                      <span v-if="row.task_id" class="adm-ai-entry-link"
                        >关联任务 #{{ row.task_id }}</span
                      >
                    </div>
                  </li>
                </ol>
              </section>
            </div>
          </div>

          <aside class="adm-board-aside">
            <div class="adm-card adm-ai-grant">
              <div class="adm-card-content">
                <header class="adm-ai-grant-head">
                  <h3 class="adm-card-title">发放额度</h3>
                  <a-radio-group
                    v-model="grantForm.scope"
                    :disabled="grantBusy || !!pendingGrant"
                    type="button"
                    size="small"
                  >
                    <a-radio value="selected">选择用户</a-radio>
                    <a-radio value="all">全体用户</a-radio>
                  </a-radio-group>
                </header>
                <label v-if="grantForm.scope === 'selected'" class="adm-ai-field">
                  <span class="adm-ai-field-label">发给谁</span>
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
                <div v-else class="adm-ai-field">
                  <span class="adm-ai-field-label">发给谁</span>
                  <strong class="adm-ai-field-static">全体用户</strong>
                  <span class="adm-ai-hint">提交时由服务器确定完整用户列表</span>
                </div>
                <label class="adm-ai-field">
                  <span class="adm-ai-field-label">发放额度</span>
                  <a-input-number
                    v-model="grantForm.amount"
                    :input-attrs="{ 'aria-label': '发放额度' }"
                    :min="1"
                    :max="10000"
                    :precision="0"
                    :disabled="grantBusy || !!pendingGrant"
                  />
                  <span class="adm-ai-hint">每位用户本次增加的额度，1–10000</span>
                </label>
                <label class="adm-ai-field">
                  <span class="adm-ai-field-label">发放原因（选填）</span>
                  <a-input
                    v-model="grantForm.note"
                    :input-attrs="{ 'aria-label': '发放原因' }"
                    placeholder="如：活动奖励、创作支持"
                    :max-length="300"
                    :disabled="grantBusy || !!pendingGrant"
                  />
                </label>
                <div class="adm-ai-grant-foot">
                  <p v-if="grantSuccess" role="status" class="adm-ai-hint is-success">
                    {{ grantSuccess }}
                  </p>
                  <p v-else-if="pendingGrant" class="adm-ai-hint">
                    发放结果尚未确认，再次提交会核对上次操作，不会重复增加额度。
                  </p>
                  <span v-else class="adm-ai-hint">单次每位用户 1–10000 额度</span>
                  <a-button type="primary" size="small" :loading="grantBusy" @click="grant">{{
                    pendingGrant ? '确认上次发放' : '发放额度'
                  }}</a-button>
                </div>
              </div>
            </div>
          </aside>
        </section>

        <section id="ai-pane-tasks" role="tabpanel" :aria-labelledby="'ai-tab-tasks'" v-else-if="tab === 'tasks'" class="adm-ai-pane">
          <header class="adm-ai-pane-head">
            <div>
              <h2>生图任务</h2>
              <p>任务、结果图与原始参考图并排可查；失败任务会自动返还额度，原始报错仅管理员可见。</p>
            </div>
            <div class="adm-ai-filter-row">
              <a-input
                v-model="search"
                size="small"
                placeholder="搜索提示词或用户"
                class="adm-ai-search"
                @press-enter="run(() => queryRows())"
              />
              <a-select
                v-model="status"
                size="small"
                placeholder="全部生成状态"
                allow-clear
                class="adm-ai-filter"
              >
                <a-option v-for="(label, key) in taskLabels" :key="key" :value="key">{{
                  label
                }}</a-option>
              </a-select>
              <a-button size="small" @click="run(() => queryRows())">搜索</a-button>
            </div>
          </header>

          <AdminSkeleton
            v-if="loading && !tasks.length"
            variant="cards"
            :rows="3"
            :columns="3"
            label="加载生成任务"
          />
          <AdminEmpty
            v-else-if="!shownTasks.length"
            :title="tasks.length ? '没有匹配的任务' : '暂无生成任务'"
            :desc="
              tasks.length
                ? '换个关键词或状态筛选再试。'
                : '用户在 AI 工作台提交生图后，任务会出现在这里。'
            "
          />
          <div v-else class="adm-card-grid adm-stagger">
            <article
              v-for="row in shownTasks"
              :key="row.task.id"
              class="adm-card admin-task-record"
            >
              <div class="admin-task-cover adm-card-media">
                <button
                  v-if="row.assets.length"
                  type="button"
                  class="admin-image-preview"
                  :aria-label="'查看生成结果 1'"
                  @click="previewAsset = { id: row.assets[0]?.id ?? 0, title: '生成结果 1' }"
                >
                  <CreativeImage :id="row.assets[0]?.id ?? 0" />
                </button>
                <span v-else class="admin-task-cover-empty">{{
                  row.task.status === 'succeeded' ? '暂无可展示的结果图片' : '尚无生成结果'
                }}</span>
                <span class="adm-card-badge">{{
                  taskLabels[row.task.status] || row.task.status
                }}</span>
              </div>

              <div class="adm-card-content">
                <header class="admin-task-header">
                  <div class="admin-task-identity">
                    <strong class="admin-task-id">#{{ row.task.id }}</strong>
                    <span class="admin-task-author">{{ taskAuthor(row) }}</span>
                  </div>
                  <time class="admin-task-time" :datetime="row.task.created_at"
                    >{{ dateText(row.task.created_at) }} {{ timeText(row.task.created_at) }}</time
                  >
                </header>
                <p v-if="row.task.error" class="adm-ai-alert is-warning" role="alert">
                  {{ row.task.error }}
                </p>
                <div class="admin-task-meta">
                  <span>{{ row.task.model }}</span>
                  <span>{{ row.task.size }} · {{ row.task.count }} 张</span>
                  <span>{{ row.task.cost }} 额度</span>
                  <span v-if="row.task.attempts > 1">{{ row.task.attempts }} 次尝试</span>
                  <RouterLink v-if="row.task.idea_id" :to="'/ideas/' + row.task.idea_id"
                    >创意 #{{ row.task.idea_id }}</RouterLink
                  >
                </div>
                <div v-if="row.assets.length > 1" class="admin-task-results" aria-label="生成结果">
                  <button
                    v-for="(asset, index) in row.assets.slice(1)"
                    :key="asset.id"
                    type="button"
                    class="admin-image-preview"
                    :aria-label="'查看生成结果 ' + (index + 2)"
                    @click="previewAsset = { id: asset.id, title: '生成结果 ' + (index + 2) }"
                  >
                    <CreativeImage :id="asset.id" />
                  </button>
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
                  <summary>原始报错记录 · 仅管理员可见</summary>
                  <pre>{{ row.diagnostic }}</pre>
                  <a-button size="small" @click="copyDiagnostic(row.diagnostic)">复制诊断</a-button>
                </details>
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
                      :aria-label="'查看原始参考图 ' + (index + 1)"
                      @click="previewAsset = { id: asset.id, title: '原始参考图 ' + (index + 1) }"
                    >
                      <CreativeImage :id="asset.id" />
                    </button>
                  </div>
                </aside>
              </div>

              <footer class="adm-card-foot">
                <a-button
                  size="small"
                  status="danger"
                  class="is-push"
                  @click="deleteTask(row.task.id)"
                  >删除任务及结果</a-button
                >
              </footer>
            </article>
          </div>
          <div v-if="moreRows" class="adm-ai-more">
            <a-button size="small" :loading="loading" @click="run(() => queryRows(true))"
              >加载更早的任务</a-button
            >
          </div>
        </section>

        <section id="ai-pane-ideas" role="tabpanel" :aria-labelledby="'ai-tab-ideas'" v-else-if="tab === 'ideas'" class="adm-ai-pane">
          <header class="adm-ai-pane-head">
            <div>
              <h2>创意投稿</h2>
              <p>投稿、参考图与赞助状态同卡呈现，审核动作集中在卡底。</p>
            </div>
            <div class="adm-ai-filter-row">
              <a-input
                v-model="search"
                size="small"
                placeholder="搜索投稿"
                class="adm-ai-search"
                @press-enter="run(() => queryRows())"
              />
              <a-select
                v-model="status"
                size="small"
                placeholder="全部状态"
                allow-clear
                class="adm-ai-filter"
              >
                <a-option v-for="(label, value) in ideaStatusLabels" :key="value" :value="value">{{
                  label
                }}</a-option>
              </a-select>
              <a-button size="small" @click="run(() => queryRows())">搜索投稿</a-button>
            </div>
          </header>

          <AdminSkeleton
            v-if="loading && !ideas.length"
            variant="cards"
            :rows="3"
            :columns="3"
            label="加载创意投稿"
          />
          <AdminEmpty
            v-else-if="!ideas.length"
            title="暂无创意投稿"
            desc="用户提交创意并通过审核后，会在这里看到投稿、参考图与赞助状态。"
          />
          <div v-else class="adm-card-grid adm-stagger">
            <article v-for="row in ideas" :key="row.idea.id" class="adm-card adm-idea-card">
              <div class="adm-card-media">
                <CreativeImage v-if="row.references.length" :id="row.references[0]?.id ?? 0" />
                <span v-else class="admin-task-cover-empty">暂无参考图</span>
                <span class="adm-card-badge">{{
                  ideaStatusLabels[row.idea.status] || row.idea.status
                }}</span>
              </div>
              <div class="adm-card-content">
                <h3 class="adm-card-title">{{ row.idea.title || '未命名投稿' }}</h3>
                <p class="adm-idea-desc">{{ row.idea.description || '暂无说明' }}</p>
                <div class="adm-card-meta">
                  <span class="adm-ai-chip is-accent">{{ row.current_cost }} 额度</span>
                  <span>作者 {{ row.author.username }}</span>
                  <span>{{
                    row.sponsor?.username ? '赞助 ' + row.sponsor.username : '等待赞助'
                  }}</span>
                  <time :datetime="row.idea.created_at">{{ dateText(row.idea.created_at) }}</time>
                </div>
                <details class="adm-idea-snapshot">
                  <summary>提示词与参数</summary>
                  <pre>{{ row.idea.snapshot }}</pre>
                </details>
              </div>
              <footer class="adm-card-foot">
                <RouterLink class="adm-ai-link" :to="'/ideas/' + row.idea.id">预览</RouterLink>
                <a-button size="small" @click="showEvents(row.idea.id)">操作记录</a-button>
                <a-button v-if="row.sponsorable" size="small" @click="sponsorIdea(row)"
                  >赞助</a-button
                >
                <a-button size="small" @click="hideIdea(row.idea.id)">下架</a-button>
                <a-button
                  size="small"
                  status="danger"
                  class="is-push"
                  @click="deleteIdea(row.idea.id)"
                  >删除</a-button
                >
              </footer>
            </article>
          </div>
          <div v-if="moreRows" class="adm-ai-more">
            <a-button size="small" :loading="loading" @click="run(() => queryRows(true))"
              >加载更早的投稿</a-button
            >
          </div>
        </section>

        <section id="ai-pane-sponsors" role="tabpanel" :aria-labelledby="'ai-tab-sponsors'" v-else-if="tab === 'sponsors'" class="adm-ai-pane">
          <header class="adm-ai-pane-head">
            <div>
              <h2>赞助记录</h2>
              <p>每条记录对应一次创意赞助任务；金额为本次预留并结算的额度。</p>
            </div>
            <div v-if="tasks.length" class="adm-ai-filter-row">
              <a-input
                v-model="search"
                size="small"
                placeholder="搜索赞助记录或用户"
                class="adm-ai-search"
                @press-enter="run(() => queryRows())"
              />
              <a-button size="small" @click="run(() => queryRows())">搜索</a-button>
            </div>
          </header>

          <AdminSkeleton
            v-if="loading && !tasks.length"
            variant="cards"
            :rows="2"
            :columns="3"
            label="加载赞助记录"
          />
          <AdminEmpty
            v-else-if="!shownTasks.length"
            :title="tasks.length ? '没有匹配的赞助记录' : '暂无赞助记录'"
            :desc="
              tasks.length
                ? '换个关键词再试。'
                : '用户赞助创意后会在这里生成对应任务与错误记录。'
            "
          />
          <div v-else class="adm-card-grid is-wide adm-stagger">
            <article v-for="row in shownTasks" :key="row.task.id" class="adm-card adm-sponsor-card">
              <div class="adm-card-content">
                <div class="adm-sponsor-head">
                  <span class="adm-sponsor-amount">{{ row.task.cost }}<span>额度</span></span>
                  <AdminStatus
                    :type="taskType(row.task.status)"
                    :label="taskLabels[row.task.status] || row.task.status"
                  />
                </div>
                <h3 class="adm-card-title">赞助人 {{ taskAuthor(row) }}</h3>
                <div class="adm-card-meta">
                  <span>创意 #{{ row.task.idea_id }}</span>
                  <time :datetime="row.task.created_at">{{
                    dateText(row.task.created_at) + ' ' + timeText(row.task.created_at)
                  }}</time>
                  <span>{{ row.task.model }}</span>
                </div>
                <p v-if="row.task.error" class="adm-ai-alert is-warning" role="alert">
                  {{ row.task.error }}
                </p>
              </div>
              <footer class="adm-card-foot">
                <RouterLink class="adm-ai-link is-push" :to="'/ideas/' + row.task.idea_id"
                  >查看创意</RouterLink
                >
                <a-button size="small" status="danger" @click="deleteTask(row.task.id)"
                  >删除任务及结果</a-button
                >
              </footer>
            </article>
          </div>
          <div v-if="moreRows" class="adm-ai-more">
            <a-button size="small" :loading="loading" @click="run(() => queryRows(true))"
              >加载更早的赞助记录</a-button
            >
          </div>
        </section>

        <section id="ai-pane-character" role="tabpanel" :aria-labelledby="'ai-tab-character'" v-else-if="tab === 'character'" class="adm-ai-pane">
          <header class="adm-ai-pane-head">
            <div>
              <h2>角色资源</h2>
              <p>角色由名称、外观说明与参考图组成，生成时一起导入。</p>
            </div>
            <div class="adm-ai-filter-row">
              <a-input
                v-model="search"
                size="small"
                placeholder="搜索角色资源"
                class="adm-ai-search"
                @press-enter="run(() => queryRows())"
              />
              <a-button type="primary" size="small" @click="editResource(null)"
                >新建角色资源</a-button
              >
            </div>
          </header>

          <AdminSkeleton
            v-if="loading && !shownResources.length"
            variant="cards"
            :rows="3"
            label="加载角色资源"
          />
          <AdminEmpty
            v-else-if="!shownResources.length"
            title="还没有角色资源"
            desc="添加角色名称与参考图片，生成时即可一起导入。"
          >
            <template #actions>
              <a-button type="primary" size="small" @click="editResource(null)"
                >新建角色资源</a-button
              >
            </template>
          </AdminEmpty>
          <div v-else class="adm-card-grid adm-stagger">
            <article v-for="row in shownResources" :key="row.id" class="adm-card">
              <div class="adm-card-media admin-resource-cover">
                <CreativeImage
                  v-if="row.payload.asset_ids.length"
                  :id="row.payload.asset_ids[0] ?? 0"
                />
                <span v-else class="admin-task-cover-empty">暂无参考图</span>
                <span class="adm-card-badge">{{ resourceLabel(row.status) }}</span>
              </div>
              <div class="adm-card-body">
                <h3 class="adm-card-title">{{ row.title || '未命名角色' }}</h3>
                <p class="adm-resource-summary">{{ row.summary || '暂无简介' }}</p>
                <div class="adm-card-meta">
                  <span>{{ row.payload.asset_ids.length }} 张参考图</span>
                  <span>{{ row.status === 'published' ? '已公开' : '未公开' }}</span>
                </div>
              </div>
              <footer class="adm-card-foot">
                <a-button size="small" @click="editResource(row)">编辑 / 发布 / 归档</a-button>
                <a-button size="small" status="danger" class="is-push" @click="deleteResource(row)"
                  >删除</a-button
                >
              </footer>
            </article>
          </div>

          <section class="adm-card adm-ai-assets">
            <div class="adm-card-content">
              <header class="adm-ai-assets-head">
                <h3 class="adm-card-title">未归入角色的图片（{{ unassignedAssets.length }}）</h3>
                <a-button size="small" @click="fileInput?.click()">单独上传图片</a-button>
                <input
                  ref="fileInput"
                  type="file"
                  accept="image/png,image/jpeg,image/webp"
                  hidden
                  @change="upload"
                />
              </header>
              <p class="adm-ai-hint">
                文件不可原地替换。修改图像请上传新版本，旧图与历史引用继续保留。
              </p>
              <div v-if="unassignedAssets.length" class="adm-asset-grid">
                <article
                  v-for="asset in unassignedAssets"
                  :key="asset.id"
                  class="adm-card adm-asset-card"
                >
                  <div class="adm-card-media adm-asset-thumb">
                    <CreativeImage :id="asset.id" />
                  </div>
                  <div class="adm-card-body">
                    <h4 class="adm-card-title">#{{ asset.id }} {{ asset.title || '未命名图片' }}</h4>
                    <p class="adm-ai-hint">
                      {{ asset.public ? '公开' : '私人' }} ·
                      {{ asset.archived ? '已归档' : '可用' }} ·
                      {{ asset.license || '尚未填写授权' }}
                    </p>
                  </div>
                  <footer class="adm-card-foot">
                    <a-button size="small" @click="editAsset(asset)">编辑 / 发布 / 归档</a-button>
                    <a-button
                      size="small"
                      status="danger"
                      class="is-push"
                      @click="deleteAsset(asset.id)"
                      >删除图片</a-button
                    >
                  </footer>
                </article>
              </div>
              <AdminEmpty
                v-else
                compact
                title="没有待归入的图片"
                desc="上传的设定图会先留在这里，归档到角色后从这里消失。"
              />
            </div>
          </section>
        </section>

        <section id="ai-pane-template" role="tabpanel" :aria-labelledby="'ai-tab-template'" v-else class="adm-ai-pane">
          <header class="adm-ai-pane-head">
            <div>
              <h2>提示词模板</h2>
              <p>把常用提示词整理成模板，生成时可直接选用。</p>
            </div>
            <div class="adm-ai-filter-row">
              <a-input
                v-model="search"
                size="small"
                placeholder="搜索提示词模板"
                class="adm-ai-search"
                @press-enter="run(() => queryRows())"
              />
              <a-select
                v-model="status"
                size="small"
                placeholder="全部状态"
                allow-clear
                class="adm-ai-filter"
              >
                <a-option value="draft">草稿</a-option>
                <a-option value="published">发布</a-option>
                <a-option value="archived">归档</a-option>
              </a-select>
              <a-button size="small" :disabled="!selected.length" @click="archiveSelected"
                >批量归档</a-button
              >
              <a-button type="primary" size="small" @click="editResource(null)"
                >新建提示词</a-button
              >
            </div>
          </header>

          <AdminSkeleton
            v-if="loading && !shownResources.length"
            variant="cards"
            :rows="3"
            :columns="3"
            label="加载提示词模板"
          />
          <AdminEmpty
            v-else-if="!shownResources.length"
            title="还没有提示词模板"
            desc="把常用提示词整理成模板，生成时可以直接选用。"
          >
            <template #actions>
              <a-button type="primary" size="small" @click="editResource(null)"
                >新建提示词</a-button
              >
            </template>
          </AdminEmpty>
          <div v-else class="adm-card-grid is-wide adm-stagger">
            <article
              v-for="row in shownResources"
              :key="row.id"
              class="adm-card ai-template-card"
            >
              <div class="adm-card-content">
                <header class="ai-template-head">
                  <a-checkbox
                    class="adm-template-check"
                    :model-value="selected.includes(row.id)"
                    :aria-label="'选择模板 ' + (row.title || '未命名模板')"
                    @change="(checked) => toggleSelected(row.id, !!checked)"
                  />
                  <h3 class="adm-card-title">{{ row.title || '未命名模板' }}</h3>
                </header>
                <div class="adm-ai-chips">
                  <span class="adm-ai-chip">版本 {{ row.version }}</span>
                  <span class="adm-ai-chip">排序 {{ row.sort_order }}</span>
                  <span class="adm-ai-chip" :class="row.status === 'draft' ? 'is-warn' : ''">{{
                    row.status === 'draft'
                      ? '草稿'
                      : row.status === 'published'
                        ? '已发布'
                        : '已归档'
                  }}</span>
                </div>
                <p class="adm-template-summary" :title="row.summary || ''">
                  {{ row.summary || '暂无摘要' }}
                </p>
              </div>
              <footer class="adm-card-foot">
                <a-button size="small" @click="editResource(row)">编辑 / 预览</a-button>
                <a-button
                  size="small"
                  status="danger"
                  class="is-push"
                  @click="deleteResource(row)"
                  >删除</a-button
                >
              </footer>
            </article>
          </div>
        </section>
      </div>
    </div>
      <a-modal
        :visible="!!previewAsset"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        :title="previewAsset?.title"
        :width="900"
        :footer="false"
        @cancel="previewAsset = null"
        ><div v-if="previewAsset" class="ai-dialog-body admin-asset-preview">
          <CreativeImage :id="previewAsset.id" full /></div
      ></a-modal>
      <a-modal
        v-model:visible="providerOpen"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        title="供应商配置"
        :width="760"
        :footer="false"
        ><div class="ai-dialog-body">
          <AIProviderEditor v-if="providerOpen" :provider="editingProvider" @saved="saved" />
        </div></a-modal
      >
      <a-modal
        v-model:visible="eventsOpen"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        title="创意操作记录"
        :footer="false"
        ><div class="ai-dialog-body">
          <div v-for="event in events" :key="event.id" class="studio-task">
            <strong
              >{{ event.action }} · 任务 #{{ event.task_id }} · 操作人 #{{ event.actor_id }}</strong
            >
            <p>{{ event.note }}</p>
            <small>{{ event.created_at }}</small>
          </div>
        </div></a-modal
      >
      <a-modal
        v-model:visible="resourceOpen"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        title="创作资源"
        :width="800"
        :footer="false"
        ><div class="ai-dialog-body">
          <CreativeResourceEditor
            v-if="resourceOpen"
            :resource="editingResource"
            :kind="editingResource?.kind ?? kind"
            :assets="assets"
            :resources="resources"
            @saved="saved"
            @assets-changed="assets = $event"
          />
        </div></a-modal
      >
      <a-modal
        v-model:visible="assetOpen"
        class="admin-theme creative-theme creative-dialog ai-dialog"
        title="设定图信息"
        :footer="false"
        ><div v-if="editingAsset" class="ai-dialog-body"
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
          ><a-button type="primary" size="small" :loading="assetSaving" @click="saveAsset"
            >保存</a-button
          ></div
        ></a-modal
      >
  </AdminPanel>
</template>

<style scoped>
.ai-workspace {
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr);
  gap: calc(var(--adm-gap-lg, 16px) + var(--adm-gap-xs, 4px));
  align-items: start;
  min-width: 0;
}

.ai-rail {
  position: sticky;
  top: var(--adm-gap-lg, 16px);
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: var(--adm-gap-sm, 8px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-surface, var(--admin-surface));
}
.ai-rail-row {
  position: relative;
  display: flex;
  align-items: center;
  min-width: 0;
}
.ai-rail-item {
  display: flex;
  flex: 1 1 auto;
  align-items: center;
  gap: 10px;
  min-width: 0;
  min-height: 40px;
  padding: 8px 34px 8px 12px;
  border: 0;
  border-left: 2px solid transparent;
  border-radius: var(--adm-r-control, var(--admin-radius));
  background: transparent;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.4;
  text-align: left;
  cursor: pointer;
  transition:
    background-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
.ai-rail-item:hover {
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-ink, var(--admin-text));
}
.ai-rail-item.is-active {
  border-left-color: var(--adm-accent, var(--admin-primary));
  background: var(--adm-accent-soft, var(--admin-primary-soft));
  color: var(--adm-ink, var(--admin-text));
  font-weight: 600;
}
.ai-rail-item:focus-visible {
  outline: 2px solid var(--adm-accent, var(--admin-primary));
  outline-offset: 2px;
}
.ai-rail-icon {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: calc(var(--adm-r-control, 10px) - 2px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
}
.ai-rail-item.is-active .ai-rail-icon {
  background: var(--adm-accent-soft, var(--admin-primary-soft));
  color: var(--adm-accent, var(--admin-primary));
}
.ai-rail-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ai-rail-count {
  position: absolute;
  top: 50%;
  right: 8px;
  transform: translateY(-50%);
  padding: 0 6px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 11px);
  line-height: 18px;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  pointer-events: none;
}

.ai-main {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-lg, 16px);
  min-width: 0;
}
.adm-ai-stats {
  margin: 0;
}
.adm-ai-pane {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-lg, 16px);
  min-width: 0;
}
.adm-ai-pane-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
}
.adm-ai-pane-head h2 {
  margin: 0;
  font-size: var(--adm-font-lg, 17px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
}
.adm-ai-pane-head p {
  margin: 4px 0 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text));
}
.adm-ai-filter-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  min-width: 0;
}
.adm-ai-search {
  width: 200px;
}
.adm-ai-filter {
  width: 150px;
}
.adm-ai-filter-row :deep(.adm-ai-search) {
  flex: 0 0 200px;
  width: 200px;
}
.adm-ai-filter-row :deep(.adm-ai-filter) {
  flex: 0 0 150px;
  width: 150px;
}
.adm-ai-filter-row :deep(.arco-btn) {
  flex: 0 0 auto;
  white-space: nowrap;
}

.adm-ai-alert,
.admin-ai-alert {
  margin: 0;
  padding: 8px 12px;
  border: 1px solid var(--adm-danger, var(--admin-danger));
  border-radius: var(--adm-r-control, var(--admin-radius));
  background: var(--adm-danger-soft, var(--admin-danger-soft));
  color: var(--adm-danger, var(--admin-danger));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
}
.adm-ai-alert.is-warning,
.admin-ai-alert.is-warning {
  border-color: var(--adm-warning, var(--adm-accent, var(--admin-primary)));
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-ink, var(--admin-text));
}

.adm-ai-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}
.adm-ai-chip {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  padding: 1px 8px;
  overflow: hidden;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 11px);
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.adm-ai-chip.is-accent {
  background: var(--adm-accent-soft, var(--admin-primary-soft));
  color: var(--adm-accent, var(--admin-primary));
}
.adm-ai-chip.is-danger {
  background: var(--adm-danger-soft, var(--admin-danger-soft));
  color: var(--adm-danger, var(--admin-danger));
}
.adm-ai-chip.is-warn {
  color: var(--adm-warning, var(--adm-accent, var(--admin-primary)));
}

.adm-ai-provider-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.adm-ai-provider-head .adm-card-title {
  flex: 1 1 auto;
  min-width: 0;
  margin: 0;
}
.adm-ai-dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--adm-muted, var(--admin-text));
}
.adm-ai-dot.is-on {
  background: var(--adm-success, var(--adm-accent, var(--admin-primary)));
}
.adm-ai-dot.is-off {
  background: var(--adm-line, var(--admin-border-soft));
  box-shadow: inset 0 0 0 1px var(--adm-muted, var(--admin-text));
}
.adm-ai-provider-state {
  flex: none;
  font-size: var(--adm-font-xs, 11px);
  color: var(--adm-muted, var(--admin-text));
}

.adm-ai-fold {
  padding: 0;
  overflow: hidden;
}
.adm-ai-fold-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid transparent;
  cursor: pointer;
  list-style: none;
  transition: background-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
.adm-ai-fold-head::-webkit-details-marker {
  display: none;
}
.adm-ai-fold-head:hover {
  background: var(--adm-soft, var(--admin-fill));
}
.adm-ai-fold[open] .adm-ai-fold-head {
  border-bottom-color: var(--adm-line, var(--admin-border-soft));
}
.adm-ai-fold-head::after {
  content: '▾';
  margin-left: auto;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  transition: transform var(--adm-dur-base, 220ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1));
}
.adm-ai-fold[open] .adm-ai-fold-head::after {
  transform: rotate(180deg);
}
.adm-ai-fold-title {
  flex: none;
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
}
.adm-ai-fold-desc {
  flex: 1 1 240px;
  min-width: 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text));
}
.adm-ai-fold-state {
  flex: none;
  font-size: var(--adm-font-xs, 11px);
  color: var(--adm-muted, var(--admin-text));
}
.adm-ai-fold:not([open]) .adm-ai-fold-body {
  display: none;
}
.admin-task-prompt:not([open]) pre,
.admin-task-diagnostic:not([open]) > *:not(summary),
.adm-idea-snapshot:not([open]) pre {
  display: none;
}
.adm-ai-fold-body {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  padding: 14px 16px 16px;
}
.adm-ai-fold-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.adm-ai-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.adm-ai-field-label {
  font-size: var(--adm-font-xs, 11px);
  font-weight: 600;
  letter-spacing: 0.04em;
  color: var(--adm-muted, var(--admin-text));
}
.adm-ai-field-static {
  font-size: var(--adm-font-base, 14px);
  color: var(--adm-ink, var(--admin-text));
}
.adm-ai-field-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.adm-ai-hint {
  margin: 0;
  font-size: var(--adm-font-xs, 11px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text));
}
.adm-ai-hint.is-success {
  color: var(--adm-success, var(--adm-accent, var(--admin-primary)));
}

.adm-ai-grant-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
}
.adm-ai-grant-head .adm-card-title {
  margin: 0;
  font-size: var(--adm-font-base, 14px);
}
.adm-ai-grant-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-top: var(--adm-gap-md, 12px);
  border-top: 1px solid var(--adm-line, var(--admin-border-soft));
}
.adm-ai-grant-foot .adm-ai-hint {
  flex: 1 1 140px;
  min-width: 0;
}

.adm-ai-timeline {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-xl, 24px);
  min-width: 0;
}
.adm-ai-day {
  margin: 0;
}
.adm-ai-day-label {
  margin: 0 0 8px;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 11px);
  font-weight: 600;
  letter-spacing: 0.08em;
  line-height: 1.5;
  text-transform: uppercase;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}
.adm-ai-day-rows {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0 0 0 20px;
  list-style: none;
}
.adm-ai-day-rows::before {
  content: '';
  position: absolute;
  top: 8px;
  bottom: 8px;
  left: 5px;
  width: 1px;
  background: var(--adm-line, var(--admin-border-soft));
}
.adm-ai-entry {
  position: relative;
  display: flex;
  align-items: flex-start;
  min-width: 0;
  padding: 8px 10px;
  border-radius: var(--adm-r-control, var(--admin-radius));
  transition: background-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
.adm-ai-entry:hover {
  background: var(--adm-soft, var(--admin-fill));
}
.adm-ai-node {
  position: absolute;
  top: 14px;
  left: -19px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--adm-surface, var(--admin-surface));
  box-shadow: inset 0 0 0 1px var(--adm-line, var(--admin-border-soft));
}
.adm-ai-entry-body {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.adm-ai-entry-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.adm-ai-entry-user {
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-ink, var(--admin-text));
  overflow-wrap: anywhere;
}
.adm-ai-entry-amount {
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}
.adm-ai-entry-amount.is-plus {
  color: var(--adm-success, var(--adm-accent, var(--admin-primary)));
}
.adm-ai-entry-amount.is-minus {
  color: var(--adm-danger, var(--admin-danger));
}
.adm-ai-entry-time {
  margin-left: auto;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 11px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}
.adm-ai-entry-note {
  margin: 0;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.adm-ai-entry-link {
  font-size: var(--adm-font-xs, 11px);
  color: var(--adm-accent, var(--admin-primary));
}

.admin-task-record {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.admin-task-cover {
  position: relative;
  aspect-ratio: 4 / 3;
  margin: 0;
  background: var(--adm-soft, var(--admin-fill));
}
.admin-task-cover .admin-image-preview {
  display: block;
  width: 100%;
  height: 100%;
}
.admin-task-cover :deep(.creative-image img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.admin-task-cover-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 12px;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  text-align: center;
}
.admin-task-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-width: 0;
}
.admin-task-identity {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.admin-task-id {
  color: var(--adm-ink, var(--admin-text));
  font-family: ui-monospace,SFMono-Regular,Menlo,monospace;
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
}
.admin-task-author {
  max-width: 22ch;
  overflow: hidden;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.admin-task-time {
  color: var(--adm-muted, var(--admin-text));
  font-family: ui-monospace,SFMono-Regular,Menlo,monospace;
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}
.admin-task-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  overflow-wrap: anywhere;
}
.admin-task-meta > span {
  padding: 1px 8px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
}
.admin-task-meta > a {
  color: var(--adm-accent, var(--admin-primary));
  font-size: var(--adm-font-xs, 12px);
  text-decoration: underline;
  text-underline-offset: 3px;
}
.admin-task-results {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  min-width: 0;
}
.admin-task-results .admin-image-preview {
  width: 100%;
  height: auto;
  aspect-ratio: 1 / 1;
}
.admin-task-prompt {
  min-width: 0;
}
.admin-task-prompt summary {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
  cursor: pointer;
  list-style: none;
}
.admin-task-prompt summary::-webkit-details-marker {
  display: none;
}
.admin-task-prompt-excerpt {
  display: -webkit-box;
  flex: 0 1 auto;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  min-width: 0;
  overflow: hidden;
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.7;
  overflow-wrap: anywhere;
  white-space: normal;
}
.admin-prompt-expand,
.admin-prompt-collapse {
  flex: none;
  color: var(--adm-accent, var(--admin-primary));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.9;
  text-decoration: underline;
  text-underline-offset: 3px;
  white-space: nowrap;
}
.admin-prompt-collapse,
.admin-task-prompt[open] .admin-task-prompt-excerpt,
.admin-task-prompt[open] .admin-prompt-expand {
  display: none;
}
.admin-task-prompt[open] .admin-prompt-collapse {
  display: inline-block;
  margin-left: auto;
}
.admin-task-prompt pre {
  max-height: 220px;
  margin: 8px 0 0;
  padding: 0;
  overflow: auto;
  color: var(--adm-muted, var(--admin-text));
  font-family: ui-monospace,SFMono-Regular,Menlo,monospace;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.75;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.admin-task-diagnostic {
  padding: 8px 12px;
  border: 1px dashed var(--adm-line, var(--admin-border-soft));
  border-radius: 10px;
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
}
.admin-task-diagnostic summary {
  cursor: pointer;
  list-style: none;
}
.admin-task-diagnostic summary::-webkit-details-marker {
  display: none;
}
.admin-task-diagnostic pre {
  max-height: 200px;
  margin: 8px 0;
  overflow: auto;
  font-family: ui-monospace,SFMono-Regular,Menlo,monospace;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.admin-task-references {
  min-width: 0;
  margin: 0;
}
.admin-task-eyebrow {
  margin: 0;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 11px);
  font-weight: 600;
  letter-spacing: 0.08em;
  line-height: 1.5;
  text-transform: uppercase;
}
.admin-task-eyebrow span {
  margin-left: 4px;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  opacity: 0.8;
}
.admin-task-reference-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  margin-top: 6px;
}
.admin-task-reference-list .admin-image-preview {
  width: 56px;
  height: 56px;
}

.adm-idea-card .adm-card-media {
  position: relative;
  aspect-ratio: 4 / 3;
}
.adm-idea-card .adm-card-media :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.adm-idea-desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  margin: 0;
  overflow: hidden;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  overflow-wrap: anywhere;
}
.adm-idea-snapshot summary {
  color: var(--adm-accent, var(--admin-primary));
  font-size: var(--adm-font-xs, 11px);
  line-height: 1.7;
  cursor: pointer;
  list-style: none;
}
.adm-idea-snapshot summary::-webkit-details-marker {
  display: none;
}
.adm-idea-snapshot pre {
  max-height: 180px;
  margin: 8px 0 0;
  overflow: auto;
  color: var(--adm-muted, var(--admin-text));
  font-family: ui-monospace,SFMono-Regular,Menlo,monospace;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.adm-sponsor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}
.adm-sponsor-amount {
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-xl, 21px);
  font-weight: 600;
  line-height: 1.2;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}
.adm-sponsor-amount span {
  margin-left: 4px;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 11px);
  font-weight: 500;
}

.admin-resource-cover {
  aspect-ratio: 4 / 5;
}
.admin-resource-cover :deep(.creative-image img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center 22%;
}
.adm-resource-summary {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  margin: 0;
  overflow: hidden;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
}
.adm-ai-assets {
  padding: 0;
}
.adm-ai-assets-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.adm-ai-assets-head .adm-card-title {
  flex: 1 1 200px;
  margin: 0;
  font-size: var(--adm-font-base, 14px);
}
.adm-asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(196px, 1fr));
  gap: var(--adm-gap-md, 12px);
  margin-top: var(--adm-gap-md, 12px);
}
.adm-asset-card {
  min-width: 0;
}
.adm-asset-thumb {
  aspect-ratio: 4 / 5;
}
.adm-asset-thumb :deep(.creative-image img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center 22%;
}

.adm-template-summary {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.adm-ai-more {
  display: flex;
  justify-content: center;
  padding-top: 4px;
}
.adm-ai-link {
  align-self: center;
  color: var(--adm-accent, var(--admin-primary));
  font-size: var(--adm-font-xs, 12px);
  text-decoration: underline;
  text-underline-offset: 3px;
}


.ai-icon-bar {
  display: inline-flex;
  align-items: center;
  gap: var(--adm-gap-xs, 4px);
}
.ai-icon-bar .admin-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--adm-r-control, 10px);
  background: transparent;
  color: var(--adm-muted, var(--admin-text));
  font-size: var(--adm-font-base, 14px);
  cursor: pointer;
  transition:
    background-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease),
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);
}
.ai-icon-bar .admin-icon-btn:hover {
  border-color: var(--adm-line, var(--admin-border-soft));
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-ink, var(--admin-text));
}
.ai-icon-bar .admin-icon-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.ai-icon-bar .admin-icon-btn.is-danger:hover {
  border-color: var(--adm-danger, var(--admin-danger));
  background: var(--adm-danger-soft, var(--admin-danger-soft));
  color: var(--adm-danger, var(--admin-danger));
}
.ai-icon-bar .admin-icon-btn:focus-visible {
  outline: 2px solid var(--adm-accent, var(--admin-primary));
  outline-offset: 2px;
}

.ai-template-head {
  display: flex;
  align-items: flex-start;
  gap: var(--adm-gap-sm, 8px);
  min-width: 0;
}
.ai-template-head .adm-card-title {
  margin: 0;
  overflow-wrap: anywhere;
}
.ai-template-head .adm-template-check {
  flex: none;
  margin: 0;
}
.ai-template-card .adm-card-content {
  overflow-wrap: anywhere;
}
.ai-workspace .adm-card-title {
  overflow-wrap: anywhere;
}

.ai-dialog-body {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
  overflow-wrap: anywhere;
}
.ai-dialog-body :deep(.arco-btn) {
  border-radius: var(--adm-r-control, 10px);
}
.ai-dialog-body
  :deep(.arco-btn.arco-btn-secondary.arco-btn-status-normal:not(.arco-btn-disabled)) {
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  background: var(--adm-surface, var(--admin-surface));
  color: var(--adm-ink, var(--admin-text));
}
.ai-dialog-body
  :deep(.arco-btn.arco-btn-secondary.arco-btn-status-normal:not(.arco-btn-disabled):hover) {
  border-color: var(--adm-accent, var(--admin-primary));
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-accent, var(--admin-primary));
}
.ai-workspace :deep(.arco-btn) {
  border-radius: var(--adm-r-control, 10px);
}
.ai-workspace :deep(.arco-btn-primary:not(.arco-btn-disabled)),
.ai-dialog-body :deep(.arco-btn-primary:not(.arco-btn-disabled)) {
  border-color: transparent;
  background-color: var(--adm-accent, var(--admin-primary));
}
.ai-workspace :deep(.arco-btn-primary:not(.arco-btn-disabled):hover),
.ai-workspace :deep(.arco-btn-primary:not(.arco-btn-disabled):active),
.ai-dialog-body :deep(.arco-btn-primary:not(.arco-btn-disabled):hover),
.ai-dialog-body :deep(.arco-btn-primary:not(.arco-btn-disabled):active) {
  border-color: transparent;
  background-color: var(
    --adm-accent-hover,
    var(--site-accent-hover, var(--adm-accent, var(--admin-primary)))
  );
}
.ai-workspace :deep(.arco-input-wrapper),
.ai-workspace :deep(.arco-select-view-single),
.ai-dialog-body :deep(.arco-input-wrapper),
.ai-dialog-body :deep(.arco-select-view-single),
.ai-dialog-body :deep(.arco-textarea-wrapper) {
  border-radius: var(--adm-r-control, 10px);
}

@media (max-width: 1180px) {
  .ai-workspace {
    grid-template-columns: 176px minmax(0, 1fr);
  }
}

@media (max-width: 820px) {
  .ai-workspace {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--adm-gap-md, 12px);
  }
  .ai-rail {
    position: static;
    flex-direction: row;
    flex-wrap: nowrap;
    gap: 4px;
    overflow-x: auto;
    overflow-y: hidden;
    padding: 6px;
    white-space: nowrap;
    scrollbar-width: none;
  }
  .ai-rail::-webkit-scrollbar {
    display: none;
  }
  .ai-rail-row {
    flex: 0 0 auto;
  }
  .ai-rail-item {
    flex: 0 0 auto;
    min-height: 44px;
    padding: 8px 36px 8px 10px;
    border-left: 0;
    border-bottom: 2px solid transparent;
    white-space: nowrap;
  }
  .ai-rail-item.is-active {
    border-bottom-color: var(--adm-accent, var(--admin-primary));
  }
  .ai-rail-label {
    overflow: visible;
  }
}

@media (max-width: 768px) {
  .ai-rail-item {
    min-height: 44px;
  }
  .ai-icon-bar .admin-icon-btn {
    width: 44px;
    height: 44px;
  }
  .ai-workspace :deep(.arco-btn-size-small),
  .ai-dialog-body :deep(.arco-btn-size-small) {
    min-height: 44px;
  }
  .adm-ai-pane-head {
    flex-direction: column;
    align-items: stretch;
  }
  .adm-ai-search,
  .adm-ai-filter,
  .adm-ai-filter-row :deep(.adm-ai-search),
  .adm-ai-filter-row :deep(.adm-ai-filter) {
    width: 100%;
  }
  .adm-ai-entry-time {
    margin-left: 0;
  }
  .adm-ai-pane .adm-card-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .adm-asset-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .adm-ai-pane .adm-card-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .adm-asset-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .ai-rail-item,
  .adm-ai-entry,
  .ai-icon-bar .admin-icon-btn,
  .adm-ai-fold-head,
  .adm-ai-fold-head::after {
    transition: none;
  }
}
</style>
