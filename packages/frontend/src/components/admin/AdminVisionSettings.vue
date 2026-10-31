<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { adminApi } from '@/api'
import { toast } from '@/composables/useToast'
import { formatTime } from '@/utils'
import type { VisionSettings, VisionJob, VisionResult, VisionReference, ApiResponse } from '@/types'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminStatus from './AdminStatus.vue'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminReferenceLibrary from './AdminReferenceLibrary.vue'

type StatusType = 'success' | 'warning' | 'danger' | 'info' | 'neutral'

const settings = ref<VisionSettings>({
  reference_mode: 'images',
  reference_prompt: '',
  enabled: false,
  base_url: 'https://api.openai.com/v1',
  model: 'gpt-4o-mini',
  prompt: '请根据图片生成标题和标签。',
  max_tags: 8,
  timeout_seconds: 60,
  disable_thinking: false,
  token_parameter: 'max_tokens',
  use_references: false,
  has_api_key: false,
})
const loaded = ref(false)
const apiKey = ref('')
const clearKey = ref(false)
const jobs = ref<VisionJob[]>([])
const references = ref<VisionReference[]>([])
const result = ref<VisionResult>()
const busy = ref(false)
const error = ref('')
const controller = new AbortController()
const labels: Record<VisionJob['status'], string> = { pending: '等待生成', running: '生成中', succeeded: '已完成', failed: '失败', cancelled: '人工已修改 / 已删除' }
const jobStatusType: Record<VisionJob['status'], StatusType> = { pending: 'warning', running: 'info', succeeded: 'success', failed: 'danger', cancelled: 'neutral' }

const enabledState = computed<{ type: StatusType; label: string }>(() => {
  if (!settings.value) return { type: 'neutral', label: '配置载入中' }
  return settings.value.enabled ? { type: 'success', label: '后台生成已启用' } : { type: 'neutral', label: '后台生成已停用' }
})

const jobSuccess = computed(() => jobs.value.filter(job => job.status === 'succeeded').length)
const jobFailed = computed(() => jobs.value.filter(job => job.status === 'failed').length)
const staleRefs = computed(() => references.value.filter(ref => ref.needs_description || !ref.description).length)
const toolbarSummary = computed(() => `最近 ${jobs.value.length} 条任务 · 参照图 ${references.value.length} / 12`)

function data<T>(response: ApiResponse<T>): T {
  if (response.code !== 200) throw new Error(response.message || '操作失败')
  return response.data
}
async function run(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try { await action() } catch (e) {
    if (!controller.signal.aborted) error.value = (e as Error).message || '操作失败'
  } finally { if (!controller.signal.aborted) busy.value = false }
}
async function load() {
  await run(async () => {
    const [config, tasks, refs] = await Promise.all([
      adminApi.visionSettings(controller.signal), adminApi.visionJobs(controller.signal), adminApi.visionReferences(controller.signal),
    ])
    if (controller.signal.aborted) return
    settings.value = data(config); jobs.value = data(tasks); references.value = data(refs); loaded.value = true
  })
}
async function save() {
  if (!settings.value) return
  const res = await adminApi.saveVisionSettings({ ...settings.value, ...(apiKey.value.trim() ? { api_key: apiKey.value.trim() } : {}), clear_api_key: clearKey.value }, controller.signal)
  if (controller.signal.aborted) return
  settings.value = data(res); apiKey.value = ''; clearKey.value = false; result.value = undefined
}
function preset() {
  if (!settings.value) return
  settings.value.base_url = 'https://api.siliconflow.cn/v1'
  settings.value.model = 'Qwen/Qwen3.5-4B'
  settings.value.disable_thinking = true
  settings.value.token_parameter = 'max_tokens'
}
function clearApiKey() {
  if (!settings.value) return
  apiKey.value = ''
  clearKey.value = true
}
async function handleSave() { await run(async () => { await save(); if (!controller.signal.aborted) toast.success('视觉配置已保存') }) }
async function test() {
  await run(async () => {
    await save()
    if (controller.signal.aborted) return
    const res = await adminApi.testVision(controller.signal)
    if (!controller.signal.aborted) result.value = data(res)
  })
}
async function retry(id: number) {
  await run(async () => {
    data(await adminApi.retryVision(id, controller.signal))
    if (controller.signal.aborted) return
    const res = await adminApi.visionJobs(controller.signal)
    if (!controller.signal.aborted) jobs.value = data(res)
  })
}
onMounted(load)
onBeforeUnmount(() => controller.abort())
</script>

<template>
  <AdminPanel title="视觉模型" desc="为勾选自动生成的图片补全标题与标签，使用 OpenAI Chat Completions 兼容接口。" flow>
    <template #actions>
      <AdminStatus :type="enabledState.type" :label="enabledState.label" />
    </template>

    <template #toolbar>
      <AdminDataToolbar :summary="toolbarSummary" :loading="busy" refreshable @refresh="load" />
    </template>

    <a-alert v-if="error" type="error" :title="error" class="vision-error" />

    <AdminSkeleton v-if="!loaded" variant="rows" :rows="6" :columns="2" label="正在载入视觉配置" />

    <div v-else class="adm-board vision-board">
      <div class="adm-board-main">
        <div class="adm-stat-grid vision-stats">
          <div class="adm-stat">
            <span class="adm-stat-label">启用状态</span>
            <span class="adm-stat-value">{{ settings.enabled ? '开启' : '关闭' }}</span>
            <span class="adm-stat-hint">{{ settings.enabled ? '后台生成正在运行' : '未启用，任务只会排队' }}</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">视觉模型</span>
            <span class="adm-stat-value vision-stat-text" :title="settings.model">{{ settings.model }}</span>
            <span class="adm-stat-hint" :title="settings.base_url">{{ settings.base_url }}</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">参考图</span>
            <span class="adm-stat-value">{{ references.length }} / 12</span>
            <span class="adm-stat-hint">{{ staleRefs ? `${staleRefs} 张描述待更新` : '描述均已就绪' }}</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">最近任务</span>
            <span class="adm-stat-value">{{ jobs.length }}</span>
            <span class="adm-stat-hint">成功 {{ jobSuccess }} · 失败 {{ jobFailed }}</span>
          </div>
        </div>

        <section class="adm-card vision-card" aria-labelledby="vision-api-title">
          <div class="vision-card-head">
            <div class="vision-card-head-text">
              <h3 id="vision-api-title" class="vision-card-title">接口与生成</h3>
              <p class="vision-card-desc">未启用或未配置 Key 时，任务保留在队列中等待。生成失败不会删除作品，也不改变审核状态。</p>
            </div>
            <a-button size="small" :disabled="busy" @click="preset">使用硅基流动 Qwen 4B 预设</a-button>
          </div>

          <a-form :model="settings" layout="vertical" class="vision-form">
            <div class="vision-switch-row vision-switch-top">
              <a-switch v-model="settings.enabled" :disabled="busy" aria-label="启用后台生成" />
              <span class="vision-switch-label">启用后台生成</span>
              <AdminStatus :type="settings.enabled ? 'success' : 'neutral'" :label="settings.enabled ? '已启用' : '已停用'" />
            </div>

            <div class="vision-grid">
              <a-form-item label="API 基址 / 完整端点" required>
                <a-input v-model="settings.base_url" :disabled="busy" placeholder="https://api.siliconflow.cn/v1" />
              </a-form-item>
              <a-form-item label="视觉模型 ID" required>
                <a-input v-model="settings.model" :disabled="busy" />
              </a-form-item>
              <a-form-item class="is-full" label="API Key">
                <a-input-password v-model="apiKey" :disabled="busy || clearKey" autocomplete="new-password" :placeholder="settings.has_api_key ? '已配置，留空保持原 Key' : '待配置 API Key'" />
                <span class="vision-field-hint">{{ clearKey ? '已标记清除，保存配置后移除已存 Key。' : '留空保持原 Key；替换请直接填写新的 Key。' }}</span>
              </a-form-item>
              <a-form-item label="输出 token 参数">
                <a-select v-model="settings.token_parameter" :disabled="busy" :options="['max_tokens', 'max_completion_tokens']" />
              </a-form-item>
              <a-form-item label="请求超时（秒）">
                <a-input-number v-model="settings.timeout_seconds" :disabled="busy" :min="10" :max="180" />
              </a-form-item>
              <a-form-item class="is-full" label="兼容设置">
                <a-checkbox v-model="settings.disable_thinking" :disabled="busy">发送 enable_thinking=false（硅基流动）</a-checkbox>
              </a-form-item>
            </div>
          </a-form>
        </section>

        <section class="adm-card vision-card" aria-labelledby="vision-prompt-title">
          <div class="vision-card-head">
            <div class="vision-card-head-text">
              <h3 id="vision-prompt-title" class="vision-card-title">提示词与编目</h3>
              <p class="vision-card-desc">提示词决定标题与标签的产出风格；标签上限同时约束模型返回与二次解析。</p>
            </div>
          </div>

          <a-form :model="settings" layout="vertical" class="vision-form">
            <div class="vision-grid">
              <a-form-item class="is-full" label="编目提示词" required>
                <a-textarea v-model="settings.prompt" :disabled="busy" :auto-size="{ minRows: 7, maxRows: 14 }" />
              </a-form-item>
              <a-form-item label="标签上限">
                <a-input-number v-model="settings.max_tags" :disabled="busy" :min="1" :max="12" />
              </a-form-item>
              <a-form-item class="is-full" label="参照图转述提示词">
                <a-textarea v-model="settings.reference_prompt" :disabled="busy" :auto-size="{ minRows: 3, maxRows: 8 }" />
              </a-form-item>
            </div>
          </a-form>
        </section>

        <section class="adm-card vision-card">
          <div class="vision-reference-switches">
            <div class="vision-switch-row">
              <a-switch v-model="settings.use_references" :disabled="busy" aria-label="使用角色设定图辨认角色" />
              <span class="vision-switch-label">使用角色设定图辨认角色</span>
            </div>
            <p class="vision-note">参照库：{{ references.length ? references.map(ref => ref.name).join('、') : '尚未导入' }}。只有明确匹配才生成角色名；无法确认时使用主体标签。</p>
            <div class="vision-switch-row">
              <a-switch :model-value="settings.reference_mode === 'text'" :disabled="busy" aria-label="参照库无图模式" @update:model-value="settings.reference_mode = $event ? 'text' : 'images'" />
              <span class="vision-switch-label">参照库无图模式（使用预转述的文字特征）</span>
            </div>
            <p class="vision-note">待识别作品仍发送图片。无图模式只发送角色文字描述；每张参照图需先完成转述或手动填写描述。普通模式会发送参照图片。</p>
          </div>

          <AdminReferenceLibrary :references="references" :has-key="!!settings.has_api_key" @update="references = $event" />
        </section>

        <section class="adm-card vision-card" aria-labelledby="vision-jobs-title">
          <div class="vision-card-head">
            <div class="vision-card-head-text">
              <h3 id="vision-jobs-title" class="vision-card-title">任务队列</h3>
              <p class="vision-card-desc">最近 30 条自动生成任务；失败任务会保留错误信息，修好配置后可逐条重试。</p>
            </div>
            <span class="vision-card-count">{{ jobs.length }} 条</span>
          </div>

          <AdminEmpty v-if="!jobs.length" compact title="暂无自动生成任务" desc="勾选自动生成的作品生成后会出现在这里。" />

          <ul v-else class="vision-jobs adm-stagger">
            <li v-for="job in jobs" :key="job.content_id" class="vision-job">
              <span class="vision-job-thumb" aria-hidden="true">{{ job.content_id }}</span>
              <div class="vision-job-main">
                <div class="vision-job-line">
                  <AdminStatus :type="jobStatusType[job.status]" :label="labels[job.status]" />
                  <strong class="vision-job-id">作品 #{{ job.content_id }}</strong>
                  <span class="vision-job-meta">已尝试 {{ job.attempts }} 次</span>
                  <time class="vision-job-time" :datetime="job.updated_at">{{ formatTime(job.updated_at) }}</time>
                </div>
                <p v-if="job.error" class="vision-job-error">{{ job.error }}</p>
              </div>
              <a-button v-if="job.status === 'failed'" size="small" :disabled="busy" @click="retry(job.content_id)">重试</a-button>
            </li>
          </ul>
        </section>
      </div>

      <aside class="adm-board-aside" aria-label="视觉模型运行状态">
        <section class="adm-card vision-side">
          <div class="adm-card-content">
            <h3 class="vision-side-title">运行状态</h3>
            <dl class="adm-kv">
              <div class="adm-kv-row"><dt>后台生成</dt><dd>{{ settings.enabled ? '开启' : '关闭' }}</dd></div>
              <div class="adm-kv-row"><dt>API Key</dt><dd>{{ settings.has_api_key ? '已配置' : '未配置' }}</dd></div>
              <div class="adm-kv-row"><dt>参照模式</dt><dd>{{ settings.reference_mode === 'text' ? '文字特征' : '发送参照图' }}</dd></div>
              <div class="adm-kv-row"><dt>任务成功</dt><dd>{{ jobSuccess }}</dd></div>
              <div class="adm-kv-row"><dt>任务失败</dt><dd>{{ jobFailed }}</dd></div>
            </dl>

            <div class="vision-result">
              <span class="vision-side-label">最近测试结果</span>
              <a-alert v-if="result" type="success" :title="result.title"><template #default>{{ result.tags.join(' · ') }}</template></a-alert>
              <p v-else class="vision-note">尚未测试。测试使用内置色块图片，验证视觉请求及标题、标签解析；角色识别效果需用实际作品验证。</p>
            </div>
          </div>
        </section>

        <section class="adm-card vision-side">
          <div class="adm-card-content">
            <h3 class="vision-side-title">操作</h3>
            <div class="vision-actions-vert">
              <a-button type="primary" long :loading="busy" :disabled="busy" @click="handleSave">保存配置</a-button>
              <a-button long :disabled="busy" @click="test">保存并测试视觉接口</a-button>
              <a-button long status="danger" :disabled="busy || clearKey" @click="clearApiKey">清除 API Key</a-button>
            </div>
            <p v-if="clearKey" class="vision-pending">
              已标记清除已保存的 Key，保存配置后生效。
              <button type="button" class="vision-undo" @click="clearKey = false">撤销</button>
            </p>
            <p v-if="busy" class="vision-busy" role="status">请求中…</p>
            <p class="vision-note">其他兼容服务不支持思考参数时，可在「接口与生成」里取消该选项。</p>
          </div>
        </section>
      </aside>
    </div>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

:deep(.admin-panel-toolbar .admin-data-toolbar) {
  padding: 0;
}

.vision-error {
  margin: var(--adm-gap-md, 12px) var(--adm-gap-xl, 24px) 0;
}

.vision-board {
  padding-top: var(--adm-gap-md, 12px);
}

.vision-stat-text {
  display: block;
  overflow: hidden;
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.vision-card {
  min-width: 0;
}

.vision-card-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
}

.vision-card-head-text {
  flex: 1 1 240px;
  min-width: 0;
}

.vision-card-title {
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-family: var(--site-title-font);
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  letter-spacing: -.01em;
}

.vision-card-desc {
  margin: 4px 0 0;
  max-width: 66ch;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
}

.vision-card-count {
  flex-shrink: 0;
  padding-top: 2px;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.vision-form {
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px) var(--adm-gap-lg, 16px);

  :deep(.arco-form-item) {
    margin-bottom: 0;
    min-width: 0;
  }

  :deep(.arco-form-item-label-col) {
    padding-bottom: 4px;
  }

  :deep(.arco-form-item-label) {
    color: var(--adm-muted, var(--admin-text-3));
    font-size: var(--adm-font-xs, 12px);
  }

  :deep(.arco-input-wrapper),
  :deep(.arco-select-view),
  :deep(.arco-input-number),
  :deep(.arco-textarea-wrapper) {
    width: 100%;
  }
}

.vision-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.vision-grid > .is-full {
  grid-column: 1 / -1;
}

.vision-switch-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
}

.vision-switch-top {
  padding-bottom: var(--adm-gap-md, 12px);
}

.vision-switch-label {
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
}

.vision-field-hint {
  display: block;
  margin-top: 4px;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
}

.vision-reference-switches {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
}

.vision-note {
  margin: 0;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
}

.vision-jobs {
  margin: 0;
  padding: 0;
  list-style: none;
}

.vision-job {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  padding: 10px var(--adm-gap-lg, 16px);
  border-top: 1px solid var(--adm-line, var(--admin-border-soft));
  transition: background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));

  &:first-child {
    border-top: 0;
  }

  &:hover {
    background: var(--adm-soft, var(--admin-fill));
  }
}

.vision-job-thumb {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 36px;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: 6px;
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.vision-job-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.vision-job-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 10px;
  min-width: 0;
}

.vision-job-id {
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  @include numeric;
}

.vision-job-meta,
.vision-job-time {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.vision-job-time {
  flex-shrink: 0;
  margin-left: auto;
}

.vision-job-error {
  margin: 0;
  color: var(--adm-danger, var(--admin-danger));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.vision-job > :deep(.arco-btn) {
  flex: 0 0 auto;
}

.vision-side .adm-card-content {
  gap: 10px;
}

.vision-side-title {
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-family: var(--site-title-font);
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  letter-spacing: -.01em;
}

.vision-result {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 10px;
  border-top: 1px solid var(--adm-line, var(--admin-border-soft));
}

.vision-side-label {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

.vision-result :deep(.arco-alert) {
  margin: 0;
}

.vision-actions-vert {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);

  :deep(.arco-btn) {
    width: 100%;
  }
}

.vision-pending {
  margin: 0;
  color: var(--adm-warning, rgb(var(--warning-6)));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
}

.vision-undo {
  padding: 0 2px;
  border: 0;
  background: none;
  color: var(--adm-accent, var(--admin-primary));
  font-size: inherit;
  text-decoration: underline;
  cursor: pointer;
  @include focus-ring(2px);
}

.vision-busy {
  margin: 0;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

@media (max-width: 1180px) {
  .vision-job-time {
    margin-left: 0;
  }
}

@media (max-width: 768px) {
  .vision-error {
    margin: var(--adm-gap-md, 12px) var(--adm-gap-md, 12px) 0;
  }

  .vision-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .vision-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .vision-card-head,
  .vision-form,
  .vision-reference-switches,
  .vision-job {
    padding-left: var(--adm-gap-md, 12px);
    padding-right: var(--adm-gap-md, 12px);
  }

  .vision-actions-vert :deep(.arco-btn) {
    min-height: var(--adm-touch, 44px);
  }
}

@media (max-width: 560px) {
  .vision-job-time {
    flex-basis: 100%;
  }
}
</style>
