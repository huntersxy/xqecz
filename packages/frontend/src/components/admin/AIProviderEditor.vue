<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import type { AIProvider } from '@/types/schemas'
import { aiApi } from '@/api/creative'
const props = defineProps<{ provider: AIProvider | null }>(),
  emit = defineEmits<{ saved: [] }>()
const blank = () => ({
  id: 0,
  revision: 0,
  owner: 0,
  allowed_provider_id: 0,
  name: '',
  endpoint: 'https://api.slomerex.xyz/v1/images/generations',
  model: 'gpt-image-2.5-sunburst-1K',
  enabled: false,
  default: false,
  has_key: false,
  sizes: ['1024x1024'],
  qualities: [] as string[],
  max_count: 1,
  max_references: 0,
  edit_transport: 'none' as AIProvider['edit_transport'],
  timeout_seconds: 180,
  concurrency: 1,
  cost: 1,
  quality_costs: {} as Record<string, number>,
  api_key: '',
})
const form = ref(blank()),
  error = ref(''),
  busy = ref(false),
  prices = ref('{}')
let stopped = false
onBeforeUnmount(() => {
  stopped = true
})
watch(
  () => props.provider,
  (row) => {
    form.value = row ? { ...JSON.parse(JSON.stringify(row)), api_key: '' } : blank()
    prices.value = JSON.stringify(form.value.quality_costs, null, 2)
    error.value = ''
  },
  { immediate: true },
)
const sizes = computed({
  get: () => form.value.sizes.join(', '),
  set: (v) =>
    (form.value.sizes = v
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)),
})
const qualities = computed({
  get: () => form.value.qualities.join(', '),
  set: (v) =>
    (form.value.qualities = v
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)),
})
watch(
  () => form.value.edit_transport,
  (value) => {
    if (value === 'none') form.value.max_references = 0
  },
)
async function save() {
  if (busy.value) return
  busy.value = true
  try {
    form.value.quality_costs = JSON.parse(prices.value)
    await aiApi.saveProvider(form.value)
    form.value.api_key = ''
    if (!stopped) emit('saved')
  } catch (e) {
    if (!stopped) error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <a-form :model="form" layout="vertical"
    ><div class="provider-fields">
      <a-form-item label="名称"><a-input v-model="form.name" aria-label="供应商名称" /></a-form-item
      ><a-form-item label="模型 ID"
        ><a-input v-model="form.model" aria-label="模型 ID"
      /></a-form-item>
    </div>
    <a-form-item label="生成接口地址"
      ><a-input v-model="form.endpoint" aria-label="生成接口地址" />
      <template #extra>仅支持公开 HTTPS 443 地址，并且必须以 /v1/images/generations 结尾。</template>
    </a-form-item
    ><a-form-item label="API Key（留空保留已保存密钥）"
      ><a-input-password
        v-model="form.api_key"
        aria-label="供应商 API Key"
        autocomplete="new-password"
    /></a-form-item>
    <div class="provider-fields">
      <a-form-item label="允许尺寸（逗号分隔）"><a-input v-model="sizes" /></a-form-item
      ><a-form-item label="允许画质（无此参数则留空）"><a-input v-model="qualities" /></a-form-item
      ><a-form-item label="最多输出张数"
        ><a-input-number v-model="form.max_count" :min="1" :max="2" /></a-form-item
      ><a-form-item label="每张基础额度"
        ><a-input-number v-model="form.cost" :min="1" :max="10000" /></a-form-item
      ><a-form-item label="参考图传输"
        ><a-select v-model="form.edit_transport"
          ><a-option value="none">关闭（仅文生图）</a-option
          ><a-option value="multipart">OpenAI multipart /images/edits</a-option
          ><a-option value="json">OpenAI JSON images /images/edits</a-option></a-select
        ></a-form-item
      ><a-form-item label="最多参考图"
        ><a-input-number
          v-model="form.max_references"
          :min="0"
          :max="8"
          :disabled="form.edit_transport === 'none'" /></a-form-item
      ><a-form-item label="请求超时（秒）"
        ><a-input-number v-model="form.timeout_seconds" :min="10" :max="300" /></a-form-item
      ><a-form-item label="供应商并发上限"
        ><a-input-number v-model="form.concurrency" :min="1" :max="2"
      /></a-form-item>
    </div>
    <a-form-item label='按画质覆盖单张额度（JSON，如 { "high": 3 }）'
      ><a-textarea v-model="prices"
    /></a-form-item>
    <p class="creative-note">
      只有经供应商文档或测试确认后才启用参考图接口、尺寸、画质与数量。站点不会自动重试可能已经收费的请求。
    </p>
    <a-checkbox v-model="form.enabled">启用</a-checkbox
    ><a-checkbox v-model="form.default">默认供应商</a-checkbox>
    <div v-if="error" class="creative-error">{{ error }}</div>
    <a-button type="primary" :loading="busy" @click="save">加密保存供应商</a-button></a-form
  >
</template>
<style scoped>
.provider-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 16px;
}
@media (max-width: 560px) {
  .provider-fields {
    grid-template-columns: 1fr;
  }
}
</style>
