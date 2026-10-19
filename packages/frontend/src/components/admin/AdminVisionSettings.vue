<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { adminApi } from '@/api'
import { toast } from '@/composables/useToast'
import type { VisionSettings, VisionJob, VisionResult, VisionReference, ApiResponse } from '@/types'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminStatus from './AdminStatus.vue'
import AdminReferenceLibrary from './AdminReferenceLibrary.vue'

type StatusType = 'success' | 'warning' | 'danger' | 'info' | 'neutral'

const settings = ref<VisionSettings>()
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

// 页头状态点：把「生成开关」这一最常被问的状态放到标题旁，省得滚到表单里找复选框
const enabledState = computed<{ type: StatusType; label: string }>(() => {
  if (!settings.value) return { type: 'neutral', label: '配置载入中' }
  return settings.value.enabled ? { type: 'success', label: '后台生成已启用' } : { type: 'neutral', label: '后台生成已停用' }
})

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
    settings.value = data(config); jobs.value = data(tasks); references.value = data(refs)
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
  <AdminPanel title="视觉模型" desc="为勾选自动生成的图片补全标题与标签，使用 OpenAI Chat Completions 兼容接口。">
    <template #actions>
      <AdminStatus :type="enabledState.type" :label="enabledState.label" />
    </template>

    <div class="vision-settings">
      <a-alert v-if="error" type="error" :title="error" />

      <AdminSkeleton v-if="!settings" variant="rows" :rows="6" :columns="2" label="正在载入视觉配置" />

      <template v-else>
        <!-- 区一：接口与生成（配置表单） -->
        <section class="vision-zone" aria-labelledby="vision-api-title">
          <header class="vision-zone-head">
            <div class="vision-zone-text">
              <h3 id="vision-api-title" class="vision-zone-title">接口与生成</h3>
              <p class="vision-zone-desc">未启用或未配置 Key 时，任务保留在队列中等待。生成失败不会删除作品，也不改变审核状态。</p>
            </div>
            <a-button size="small" :disabled="busy" @click="preset">使用硅基流动 Qwen 4B 预设</a-button>
          </header>

          <a-form :model="settings" layout="vertical" class="vision-form">
            <div class="vision-switch">
              <a-checkbox v-model="settings.enabled" :disabled="busy">启用后台生成</a-checkbox>
            </div>

            <fieldset class="vision-group">
              <legend class="vision-group-title">接口</legend>
              <div class="vision-grid">
                <a-form-item label="API 基址 / 完整端点" required><a-input v-model="settings.base_url" :disabled="busy" placeholder="https://api.siliconflow.cn/v1" /></a-form-item>
                <a-form-item label="视觉模型 ID" required><a-input v-model="settings.model" :disabled="busy" /></a-form-item>
              </div>
              <a-form-item label="API Key"><a-input-password v-model="apiKey" :disabled="busy || clearKey" autocomplete="new-password" :placeholder="settings.has_api_key ? '已配置，留空保持原 Key' : '待配置 API Key'" /></a-form-item>
              <a-checkbox v-model="clearKey" :disabled="busy">清除已保存的 Key（需同时关闭后台生成）</a-checkbox>
            </fieldset>

            <fieldset class="vision-group">
              <legend class="vision-group-title">编目</legend>
              <a-form-item label="编目提示词" required><a-textarea v-model="settings.prompt" :disabled="busy" :auto-size="{ minRows: 7, maxRows: 14 }" /></a-form-item>
              <div class="vision-grid">
                <a-form-item label="标签上限"><a-input-number v-model="settings.max_tags" :disabled="busy" :min="1" :max="12" /></a-form-item>
                <a-form-item label="请求超时（秒）"><a-input-number v-model="settings.timeout_seconds" :disabled="busy" :min="10" :max="180" /></a-form-item>
                <a-form-item label="输出 token 参数"><a-select v-model="settings.token_parameter" :disabled="busy" :options="['max_tokens', 'max_completion_tokens']" /></a-form-item>
                <a-form-item label="兼容设置"><a-checkbox v-model="settings.disable_thinking" :disabled="busy">发送 enable_thinking=false（硅基流动）</a-checkbox></a-form-item>
              </div>
            </fieldset>

            <fieldset class="vision-group">
              <legend class="vision-group-title">角色参照</legend>
              <a-checkbox v-model="settings.use_references" :disabled="busy">使用角色设定图辨认角色</a-checkbox>
              <p class="vision-note">参照库：{{ references.length ? references.map(ref => ref.name).join('、') : '尚未导入' }}。只有明确匹配才生成角色名；无法确认时使用主体标签。</p>
              <a-checkbox :model-value="settings.reference_mode === 'text'" :disabled="busy" @update:model-value="settings.reference_mode = $event ? 'text' : 'images'">参照库无图模式（使用预转述的文字特征）</a-checkbox>
              <p class="vision-note">待识别作品仍发送图片。无图模式只发送角色文字描述；每张参照图需先完成转述或手动填写描述。普通模式会发送参照图片。</p>
              <a-form-item label="参照图转述提示词"><a-textarea v-model="settings.reference_prompt" :disabled="busy" :auto-size="{ minRows: 3, maxRows: 8 }" /></a-form-item>
            </fieldset>

            <div class="vision-actions">
              <a-button type="primary" :loading="busy" :disabled="busy" @click="handleSave">保存配置</a-button>
              <a-button :disabled="busy" @click="test">保存并测试视觉接口</a-button>
              <span v-if="busy" class="vision-busy" role="status">请求中…</span>
            </div>
            <p class="vision-note">测试使用内置色块图片，验证视觉请求及标题、标签解析；角色识别效果需用实际作品验证。其他兼容服务不支持思考参数时可取消上方选项。</p>
            <a-alert v-if="result" type="success" :title="result.title"><template #default>{{ result.tags.join(' · ') }}</template></a-alert>
          </a-form>
        </section>

        <!-- 区二：参考库（子组件自带分区外框） -->
        <AdminReferenceLibrary :references="references" :has-key="!!settings?.has_api_key" @update="references = $event" />

        <!-- 区三：生成任务队列 -->
        <section class="vision-zone" aria-labelledby="vision-jobs-title">
          <header class="vision-zone-head">
            <div class="vision-zone-text">
              <h3 id="vision-jobs-title" class="vision-zone-title">最近 30 条任务</h3>
              <p class="vision-zone-desc">失败任务会保留错误信息，修好配置后可逐条重试。</p>
            </div>
            <a-button size="small" :disabled="busy" @click="load">刷新</a-button>
          </header>
          <p v-if="!jobs.length" class="vision-note">暂无自动生成任务</p>
          <ul v-else class="vision-jobs adm-stagger">
            <li v-for="job in jobs" :key="job.content_id">
              <div class="vision-job-main">
                <strong class="vision-job-id">作品 #{{ job.content_id }}</strong>
                <AdminStatus :type="jobStatusType[job.status]" :label="labels[job.status]" />
                <span class="vision-job-meta">已尝试 {{ job.attempts }} 次</span>
              </div>
              <p v-if="job.error" class="vision-job-error">{{ job.error }}</p>
              <a-button v-if="job.status === 'failed'" size="small" :disabled="busy" @click="retry(job.content_id)">重试</a-button>
            </li>
          </ul>
        </section>
      </template>
    </div>
  </AdminPanel>
</template>

<style lang="scss" scoped>
// 令牌一律 var(--adm-*, 后备值)：__adm-tokens 由 AdminView 的非 scoped 样式注入 body，
// 单测或首屏极早期可能还没挂上，后备值取旧的 --admin-* 保证观感不跳。
.vision-settings {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-xl, 24px);
  padding: var(--adm-gap-xl, 24px);
}

.vision-zone {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
}

.vision-zone-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  padding-bottom: var(--adm-gap-md, 12px);
  border-bottom: 1px solid var(--admin-border-soft);
}

.vision-zone-text { min-width: 0; }

.vision-zone-title {
  margin: 0;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-lg, 17px);
  font-weight: 600;
  letter-spacing: -.01em;
  color: var(--adm-ink, var(--admin-text));
}

.vision-zone-desc {
  margin: 4px 0 0;
  max-width: 68ch;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  color: var(--adm-muted, var(--admin-text-3));
}

.vision-form :deep(.arco-form-item) { margin-bottom: 14px; }

.vision-form :deep(.arco-form-item-label) {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
}

.vision-switch {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
}

.vision-group {
  min-width: 0;
  margin: 0;
  padding: 0;
  border: 0;
}

.vision-group + .vision-group {
  margin-top: var(--adm-gap-lg, 16px);
  padding-top: var(--adm-gap-lg, 16px);
  border-top: 1px solid var(--admin-border-soft);
}

.vision-group-title {
  display: block;
  padding: 0;
  margin-bottom: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-xs, 12px);
  font-weight: 600;
  letter-spacing: .08em;
  color: var(--adm-muted, var(--admin-text-3));
}

.vision-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 20px;
}

.vision-actions {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  margin-top: var(--adm-gap-lg, 16px);
  padding-top: var(--adm-gap-lg, 16px);
  border-top: 1px solid var(--admin-border-soft);
}

.vision-note {
  margin: var(--adm-gap-sm, 8px) 0 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  color: var(--adm-muted, var(--admin-text-3));
}

.vision-actions + .vision-note { margin-bottom: var(--adm-gap-md, 12px); }

// 保存 / 试连都是几秒级的网络请求，光靠按钮禁用看不出还在跑，补一句明确的进行中文案。
.vision-busy {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}

.vision-jobs {
  list-style: none;
  margin: 0;
  padding: 0;
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-control, 10px);
  overflow: hidden;

  li {
    display: flex;
    align-items: center;
    gap: var(--adm-gap-md, 12px);
    flex-wrap: wrap;
    padding: 12px 16px;
    border-bottom: 1px solid var(--admin-border-soft);
    transition: background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));

    &:last-child { border-bottom: 0; }
    &:hover { background: var(--adm-soft, var(--admin-fill)); }
  }
}

.vision-job-main {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  min-width: 0;
}

.vision-job-id {
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}

.vision-job-meta {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}

.vision-job-error {
  flex-basis: 100%;
  margin: 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-danger, var(--admin-danger));
  overflow-wrap: anywhere;
}

.vision-jobs li > .arco-btn { margin-left: auto; }

@media (max-width: 768px) {
  .vision-settings { padding: var(--adm-gap-lg, 16px); }
  .vision-grid { grid-template-columns: 1fr; }
  .vision-zone-head { align-items: stretch; }
  .vision-zone-head > .arco-btn { align-self: flex-start; }
}
</style>
