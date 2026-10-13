<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { adminApi } from '@/api'
import { toast } from '@/composables/useToast'
import type { VisionSettings, VisionJob, VisionResult, VisionReference, ApiResponse } from '@/types'
import AdminPanel from './AdminPanel.vue'

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
    <div class="vision-settings">
      <a-alert v-if="error" type="error" :title="error" />
      <a-spin :loading="busy" style="width: 100%">
        <a-form v-if="settings" :model="settings" layout="vertical">
          <div class="vision-toolbar"><a-checkbox v-model="settings.enabled" :disabled="busy">启用后台生成</a-checkbox><a-button :disabled="busy" @click="preset">使用硅基流动 Qwen 4B 预设</a-button></div>
          <p class="vision-note">未启用或未配置 Key 时，任务保留在队列中等待。生成失败不会删除作品，也不改变审核状态。</p>
          <div class="vision-grid">
            <a-form-item label="API 基址 / 完整端点" required><a-input v-model="settings.base_url" :disabled="busy" placeholder="https://api.siliconflow.cn/v1" /></a-form-item>
            <a-form-item label="视觉模型 ID" required><a-input v-model="settings.model" :disabled="busy" /></a-form-item>
          </div>
          <a-form-item label="API Key"><a-input-password v-model="apiKey" :disabled="busy || clearKey" autocomplete="new-password" :placeholder="settings.has_api_key ? '已配置，留空保持原 Key' : '待配置 API Key'" /></a-form-item>
          <a-checkbox v-model="clearKey" :disabled="busy">清除已保存的 Key（需同时关闭后台生成）</a-checkbox>
          <a-form-item label="编目提示词" required style="margin-top: 20px"><a-textarea v-model="settings.prompt" :disabled="busy" :auto-size="{ minRows: 7, maxRows: 14 }" /></a-form-item>
          <div class="vision-grid">
            <a-form-item label="标签上限"><a-input-number v-model="settings.max_tags" :disabled="busy" :min="1" :max="12" /></a-form-item>
            <a-form-item label="请求超时（秒）"><a-input-number v-model="settings.timeout_seconds" :disabled="busy" :min="10" :max="180" /></a-form-item>
            <a-form-item label="输出 token 参数"><a-select v-model="settings.token_parameter" :disabled="busy" :options="['max_tokens', 'max_completion_tokens']" /></a-form-item>
            <a-form-item label="兼容设置"><a-checkbox v-model="settings.disable_thinking" :disabled="busy">发送 enable_thinking=false（硅基流动）</a-checkbox></a-form-item>
          </div>
          <a-checkbox v-model="settings.use_references" :disabled="busy">使用角色设定图辨认角色</a-checkbox>
          <p class="vision-note">参照库：{{ references.length ? references.map(ref => ref.name).join('、') : '尚未导入' }}。只有明确匹配才生成角色名；无法确认时使用主体标签。参照图会随请求一起发送给配置的视觉服务。</p>
          <div class="vision-toolbar"><a-button type="primary" :loading="busy" :disabled="busy" @click="handleSave">保存配置</a-button><a-button :disabled="busy" @click="test">保存并测试视觉接口</a-button></div>
          <p class="vision-note">测试使用内置色块图片，验证视觉请求及标题、标签解析；角色识别效果需用实际作品验证。其他兼容服务不支持思考参数时可取消上方选项。</p>
          <a-alert v-if="result" type="success" :title="result.title"><template #default>{{ result.tags.join(' · ') }}</template></a-alert>
        </a-form>
      </a-spin>
      <div class="vision-toolbar"><h3>最近 30 条任务</h3><a-button :disabled="busy" @click="load">刷新</a-button></div>
      <p v-if="!jobs.length" class="vision-note">暂无自动生成任务</p>
      <ul v-else class="vision-jobs">
        <li v-for="job in jobs" :key="job.content_id">
          <div><strong>作品 #{{ job.content_id }}</strong><span>{{ labels[job.status] }} · 已尝试 {{ job.attempts }} 次</span><p v-if="job.error">{{ job.error }}</p></div>
          <a-button v-if="job.status === 'failed'" :disabled="busy" @click="retry(job.content_id)">重试</a-button>
        </li>
      </ul>
    </div>
  </AdminPanel>
</template>

<style scoped>
.vision-settings { padding: 24px; }
.vision-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 20px; }
.vision-toolbar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin: 12px 0; }
.vision-note { color: var(--admin-text-3); font-size: 12px; line-height: 1.8; margin: 12px 0 20px; }
.vision-jobs { list-style: none; padding: 0; }
.vision-jobs li { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 14px 0; border-bottom: 1px solid var(--admin-border-soft); }
.vision-jobs span { margin-left: 12px; color: var(--admin-text-3); font-size: 12px; }
.vision-jobs p { font-size: 12px; color: var(--admin-danger); }
@media (max-width: 640px) { .vision-settings { padding: 16px; } .vision-grid { grid-template-columns: 1fr; } }
</style>
