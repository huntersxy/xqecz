<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import type { AIProvider } from '@/types/schemas'
import { aiApi } from '@/api/creative'
import { formatImageRatio, parseImageSize } from '@/utils/imageSize'
const COMMON_PROVIDER_SIZES = [
  '1024x1024',
  '1024x576',
  '576x1024',
  '1024x640',
  '640x1024',
  '1024x683',
  '683x1024',
  '1024x768',
  '768x1024',
  '1536x1024',
  '1024x1536',
]
const props = defineProps<{ provider: AIProvider | null }>(),
  emit = defineEmits<{ saved: [] }>()
const blank = () => ({
  id: 0,
  revision: 0,
  owner: 0,
  allowed_provider_id: 0,
  name: '',
  adapter: 'openai' as AIProvider['adapter'],
  endpoint: 'https://api.slomerex.xyz/v1/images/generations',
  model: 'gpt-image-2.5-sunburst-1K',
  enabled: false,
  default: false,
  has_key: false,
  sizes: [...COMMON_PROVIDER_SIZES],
  qualities: [] as string[],
  max_count: 1,
  max_references: 0,
  edit_transport: 'none' as AIProvider['edit_transport'],
  timeout_seconds: 180,
  concurrency: 1,
  cost: 1,
  quality_costs: {} as Record<string, number>,
  options: {} as Record<string, unknown>,
  api_key: '',
})
const form = ref(blank()),
  error = ref(''),
  busy = ref(false),
  prices = ref('{}'),
  optionsText = ref('{}')
let stopped = false
onBeforeUnmount(() => {
  stopped = true
})
watch(
  () => props.provider,
  (row) => {
    form.value = row ? { ...JSON.parse(JSON.stringify(row)), api_key: '' } : blank()
    prices.value = JSON.stringify(form.value.quality_costs, null, 2)
    optionsText.value = JSON.stringify(form.value.options ?? {}, null, 2)
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
const sizePreview = computed(() =>
  form.value.sizes.map((value) => ({
    value,
    parsed: parseImageSize(value),
  })),
)
const qualities = computed({
  get: () => form.value.qualities.join(', '),
  set: (v) =>
    (form.value.qualities = v
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)),
})
function fillCommonSizes() {
  form.value.sizes = [...COMMON_PROVIDER_SIZES]
}
function removeSize(value: string) {
  form.value.sizes = form.value.sizes.filter((item) => item !== value)
}
watch(
  () => form.value.edit_transport,
  (value) => {
    if (value === 'none') form.value.max_references = 0
  },
)
watch(
  () => form.value.adapter,
  (value) => {
    if (value === 'gemini' || value === 'codex_responses') form.value.max_count = 1
    if (
      (value === 'openai_chat' || value === 'gemini' || value === 'codex_responses') &&
      form.value.edit_transport === 'multipart'
    ) {
      form.value.edit_transport = 'json'
    }
  },
)
async function save() {
  if (busy.value) return
  busy.value = true
  try {
    form.value.quality_costs = JSON.parse(prices.value)
    form.value.options = JSON.parse(optionsText.value)
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
      ><a-form-item label="适配器协议"
        ><a-select v-model="form.adapter" aria-label="供应商适配器">
          <a-option value="openai">OpenAI Images / 兼容 Images</a-option>
          <a-option value="openai_chat">OpenAI Chat Completions</a-option>
          <a-option value="gemini">Google Gemini 原生</a-option>
          <a-option value="codex_responses">Codex Responses</a-option>
          <a-option value="volcengine_ark">火山方舟 Images</a-option>
          <a-option value="gitee_ai">Gitee AI Images</a-option>
          <a-option value="siliconflow_adapter">SiliconFlow Images</a-option>
          <a-option value="agnes_ai">Agnes AI Images</a-option>
          <a-option value="grok">xAI Grok Images</a-option>
        </a-select></a-form-item
      ><a-form-item label="模型 ID"
        ><a-input v-model="form.model" aria-label="模型 ID"
      /></a-form-item>
    </div>
    <a-form-item label="生成接口地址"
      ><a-input v-model="form.endpoint" aria-label="生成接口地址" />
      <template #extra
        >仅支持公开 HTTPS 443 地址；Images 以 /images/generations 结尾，Chat 以 /chat/completions
        结尾，Gemini 以 :generateContent 结尾，Codex 以 /responses 结尾。</template
      > </a-form-item
    ><a-form-item label="API Key（留空保留已保存密钥）"
      ><a-input-password
        v-model="form.api_key"
        aria-label="供应商 API Key"
        autocomplete="new-password"
    /></a-form-item>
    <div class="provider-fields">
      <a-form-item label="允许尺寸（逗号分隔）"
        ><a-input v-model="sizes" />
        <template #extra
          >按供应商文档保留实际支持的值；<a-button type="text" size="mini" @click="fillCommonSizes"
            >填入常用预设</a-button
          ></template
        >
          <div v-if="sizePreview.length" class="provider-size-preview" aria-label="尺寸比例预览">
            <span v-for="item in sizePreview" :key="item.value" class="provider-size-chip">
              <span>{{ item.parsed ? `${formatImageRatio(item.parsed)} · ${item.value.replace('x', '×')}` : item.value }}</span>
              <button type="button" :aria-label="`移除尺寸 ${item.value}`" @click="removeSize(item.value)">×</button>
            </span>
          </div>
        </a-form-item
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
    <a-form-item label="适配器选项（JSON，可留空）"
      ><a-textarea v-model="optionsText" :auto-size="{ minRows: 2, maxRows: 6 }" />
      <template #extra>仅填写供应商文档明确支持的非密配置；API Key 始终单独加密保存。</template>
    </a-form-item>
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
.provider-size-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
.provider-size-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  padding: 4px 6px 4px 9px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
  color: var(--creative-ink);
  background: var(--creative-canvas);
  font-size: 11px;
  line-height: 1.3;
}
.provider-size-chip span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.provider-size-chip button {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  color: var(--creative-muted);
  background: transparent;
  cursor: pointer;
}
.provider-size-chip button:hover {
  color: var(--color-danger);
  background: var(--creative-soft);
}
@media (max-width: 560px) {
  .provider-fields {
    grid-template-columns: 1fr;
  }
}
</style>
