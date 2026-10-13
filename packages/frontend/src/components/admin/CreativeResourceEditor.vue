<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { creativeApi } from '@/api/creative'
import type { AIAsset, CreativeResource } from '@/types/schemas'
import { previewCreativeTemplate } from '@/utils/creativeTemplate'
import CreativeImage from '@/components/CreativeImage.vue'
const props = defineProps<{
    resource: CreativeResource | null
    kind: CreativeResource['kind']
    assets: AIAsset[]
    resources: CreativeResource[]
  }>(),
  emit = defineEmits<{ saved: [] }>()
const blank = (): CreativeResource => ({
  id: 0,
  kind: props.kind,
  slug: '',
  title: '',
  summary: '',
  status: 'draft',
  sort_order: 0,
  version: 0,
  payload: {
    description: '',
    appearance: '',
    body: '',
    asset_ids: [],
    character_ids: [],
    tags: [],
    variables: [],
    preset_id: 0,
    parameters: {},
  },
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
})
const form = ref(blank()),
  values = ref<Record<string, string>>({}),
  error = ref(''),
  busy = ref(false)
let stopped = false
onBeforeUnmount(() => {
  stopped = true
})
watch(
  () => props.resource,
  (row) => {
    form.value = row ? JSON.parse(JSON.stringify(row)) : blank()
    values.value = {}
    error.value = ''
  },
  { immediate: true },
)
const preview = computed(() =>
  previewCreativeTemplate(form.value.payload.body, form.value.payload.variables, values.value),
)
const recommendedSize = computed({
  get: () => String(form.value.payload.parameters.size ?? ''),
  set: (value) => (form.value.payload.parameters.size = value),
})
const recommendedQuality = computed({
  get: () => String(form.value.payload.parameters.quality ?? ''),
  set: (value) => (form.value.payload.parameters.quality = value),
})
async function save() {
  if (busy.value) return
  busy.value = true
  try {
    await creativeApi.save(form.value)
    if (!stopped) emit('saved')
  } catch (e) {
    if (!stopped) error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
function addVariable() {
  form.value.payload.variables.push({
    name: '',
    label: '',
    type: 'text',
    default: '',
    required: true,
    options: [],
    help: '',
  })
}
</script>
<template>
  <a-form :model="form" layout="vertical"
    ><a-form-item label="标题"><a-input v-model="form.title" aria-label="资源标题" /></a-form-item
    ><a-form-item label="Slug（小写英文、数字、短横线）"
      ><a-input v-model="form.slug" aria-label="资源 Slug" /></a-form-item
    ><a-form-item label="简介"
      ><a-textarea v-model="form.summary" aria-label="资源简介" /></a-form-item
    ><a-form-item label="详细介绍"
      ><a-textarea v-model="form.payload.description" aria-label="详细介绍" /></a-form-item
    ><a-form-item v-if="kind === 'character'" label="外观特征"
      ><a-textarea v-model="form.payload.appearance" aria-label="外观特征" /></a-form-item
    ><a-form-item label="关联设定图 / 参考图（可同时用于多个角色）"
      ><a-select v-model="form.payload.asset_ids" aria-label="关联设定图" multiple allow-search
        ><a-option
          v-for="asset in assets.filter((a) => a.public && !a.archived)"
          :key="asset.id"
          :value="asset.id"
          >#{{ asset.id }} {{ asset.title || '未命名设定图' }}</a-option
        ></a-select
      ></a-form-item
    ><a-form-item v-if="kind !== 'character'" label="关联角色"
      ><a-select v-model="form.payload.character_ids" multiple
        ><a-option
          v-for="resource in resources.filter((r) => r.kind === 'character')"
          :key="resource.id"
          :value="resource.id"
          >{{ resource.title }}</a-option
        ></a-select
      ></a-form-item
    >
    <div class="creative-grid resource-previews">
      <div v-for="id in form.payload.asset_ids" :key="id" class="creative-image-tile">
        <CreativeImage :id="id" />
      </div>
    </div>
    <template v-if="kind === 'template'"
      ><a-form-item label="提示词正文，变量写为 {变量名}"
        ><a-textarea
          v-model="form.payload.body"
          aria-label="提示词正文"
          :auto-size="{ minRows: 5 }" /></a-form-item
      ><a-form-item label="推荐参考图组合"
        ><a-select v-model="form.payload.preset_id" allow-clear
          ><a-option :value="0">不指定</a-option
          ><a-option
            v-for="resource in resources.filter((r) => r.kind === 'preset')"
            :key="resource.id"
            :value="resource.id"
            >{{ resource.title }}</a-option
          ></a-select
        ></a-form-item
      >
      <h3>模板变量</h3>
      <div v-for="(variable, index) in form.payload.variables" :key="index" class="variable-editor">
        <a-input v-model="variable.name" placeholder="变量名" /><a-input
          v-model="variable.label"
          placeholder="显示标签"
        /><a-select v-model="variable.type"
          ><a-option value="text">文本</a-option><a-option value="select">选项</a-option></a-select
        ><a-input v-model="variable.default" placeholder="默认值" /><a-input
          v-model="variable.help"
          placeholder="输入说明"
        /><a-input-tag
          v-if="variable.type === 'select'"
          v-model="variable.options"
          placeholder="输入选项后回车"
        /><a-checkbox v-model="variable.required">必填</a-checkbox
        ><a-button size="small" status="danger" @click="form.payload.variables.splice(index, 1)"
          >移除变量</a-button
        ><a-input v-model="values[variable.name]" placeholder="试填变量以预览" />
      </div>
      <a-button size="small" @click="addVariable">添加变量</a-button>
      <h3>实时预览</h3>
      <pre class="template-preview">{{ preview }}</pre>
      <a-form-item label="标签"><a-input-tag v-model="form.payload.tags" /></a-form-item
      ><a-form-item label="推荐尺寸"><a-input v-model="recommendedSize" /></a-form-item
      ><a-form-item label="推荐画质"><a-input v-model="recommendedQuality" /></a-form-item
    ></template>
    <div class="creative-actions">
      <a-form-item label="排序"><a-input-number v-model="form.sort_order" /></a-form-item
      ><a-form-item label="状态"
        ><a-select v-model="form.status"
          ><a-option value="draft">草稿</a-option><a-option value="published">发布</a-option
          ><a-option value="archived">归档</a-option></a-select
        ></a-form-item
      >
    </div>
    <div v-if="error" class="creative-error">{{ error }}</div>
    <a-button type="primary" :loading="busy" @click="save">保存资源</a-button></a-form
  >
</template>
<style scoped>
.variable-editor {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  padding: 16px;
  background: var(--creative-canvas);
  border-radius: 10px;
  margin: 12px 0;
}
.template-preview {
  white-space: pre-wrap;
  background: var(--creative-canvas);
  padding: 16px;
  border-radius: 8px;
  line-height: 1.7;
}
.creative-actions > .arco-form-item {
  flex: 1;
  min-width: 130px;
}
@media (max-width: 560px) {
  .variable-editor {
    grid-template-columns: 1fr;
  }
}
</style>
