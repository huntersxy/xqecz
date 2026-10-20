<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { creativeApi, aiApi } from '@/api/creative'
import type { AIAsset, CreativeResource } from '@/types/schemas'
import { previewCreativeTemplate } from '@/utils/creativeTemplate'
import CreativeImage from '@/components/CreativeImage.vue'
import { creativeLicenseOptions } from '@/utils/creativeLicenses'
const props = defineProps<{
    resource: CreativeResource | null
    kind: CreativeResource['kind']
    assets: AIAsset[]
    resources: CreativeResource[]
  }>(),
  emit = defineEmits<{ saved: []; assetsChanged: [assets: AIAsset[]] }>()
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
const availableAssets = ref<AIAsset[]>([]),
  uploading = ref(false),
  uploadInput = ref<HTMLInputElement>()
const licensePreset = ref(creativeLicenseOptions[0]!.value),
  customLicense = ref('')
watch(
  () => props.assets,
  (rows) => {
    availableAssets.value = [...rows]
  },
  { immediate: true },
)
const displayed = computed({
  get: () => form.value.status === 'published',
  set: (value: boolean) => {
    form.value.status = value ? 'published' : 'draft'
  },
})
async function uploadImages(event: Event) {
  const files = [...((event.target as HTMLInputElement).files ?? [])]
  if (uploading.value) return
  const license = licensePreset.value === 'other' ? customLicense.value.trim() : licensePreset.value
  if (!license) {
    error.value = '请选择图片来源与授权，或填写其他授权说明'
    return
  }
  uploading.value = true
  error.value = ''
  try {
    for (const file of files) {
      const asset = await aiApi.upload(file)
      if (stopped) return
      const metadata = {
        title: file.name.replace(/\.[^.]+$/, ''),
        description: '',
        purpose: '角色参考',
        license,
        public: true,
        download_allowed: true,
        archived: false,
      }
      await aiApi.metadata(asset.id, metadata)
      if (stopped) return
      const ready = { ...asset, ...metadata }
      availableAssets.value.unshift(ready)
      form.value.payload.asset_ids = [...new Set([...form.value.payload.asset_ids, asset.id])]
      emit('assetsChanged', [...availableAssets.value])
    }
  } catch (e) {
    if (!stopped) error.value = (e as Error).message
  } finally {
    uploading.value = false
    if (uploadInput.value) uploadInput.value.value = ''
  }
}
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
    ><a-form-item label="简介（选填）"
      ><a-textarea v-model="form.summary" aria-label="资源简介" /></a-form-item
    ><a-form-item label="详细介绍（选填）"
      ><a-textarea v-model="form.payload.description" aria-label="详细介绍" /></a-form-item
    ><a-form-item v-if="kind === 'character'" label="外观特征（选填，导入生成提示词）"
      ><a-textarea v-model="form.payload.appearance" aria-label="外观特征"
    /></a-form-item>
    <section v-if="kind !== 'template'" class="resource-upload">
      <h3>角色图片</h3>
      <p class="creative-note">
        上传后自动关联到当前角色；这组图片就是生成时的参考图，无需另建组合。图片将作为公开参考素材保存。
      </p>
      <label class="creative-field"
        >图片来源与授权<select v-model="licensePreset">
          <option
            v-for="option in creativeLicenseOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </option>
          <option value="other">其他授权说明</option>
        </select></label
      >
      <a-input
        v-if="licensePreset === 'other'"
        v-model="customLicense"
        placeholder="填写授权来源与允许使用的范围"
        :max-length="300"
      />
      <a-button size="small" :loading="uploading" :disabled="busy" @click="uploadInput?.click()"
        >添加角色图片</a-button
      >
      <input
        ref="uploadInput"
        type="file"
        accept="image/png,image/jpeg,image/webp"
        multiple
        hidden
        @change="uploadImages"
      />
    </section>
    <a-form-item v-if="kind !== 'character'" label="关联角色资源（选填，自动带入参考图）"
      ><a-select v-model="form.payload.character_ids" multiple
        ><a-option
          v-for="resource in resources.filter((r) => r.kind === 'character')"
          :key="resource.id"
          :value="resource.id"
          >{{ resource.title }}</a-option
        ></a-select
      ></a-form-item
    >
    <div v-if="form.payload.asset_ids.length" class="resource-previews">
      <figure v-for="(id, index) in form.payload.asset_ids" :key="id">
        <div class="creative-image-tile"><CreativeImage :id="id" /></div>
        <figcaption>
          <span>参考图 {{ index + 1 }}</span
          ><a-button
            size="small"
            status="danger"
            :disabled="busy || uploading"
            :aria-label="`移除参考图 ${index + 1}`"
            @click="
              form.payload.asset_ids = form.payload.asset_ids.filter((assetId) => assetId !== id)
            "
            >移除</a-button
          >
        </figcaption>
      </figure>
    </div>
    <template v-if="kind === 'template'"
      ><a-form-item label="提示词正文，变量写为 {变量名}"
        ><a-textarea
          v-model="form.payload.body"
          aria-label="提示词正文"
          :auto-size="{ minRows: 5 }"
      /></a-form-item>
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
    <a-checkbox v-model="displayed">{{
      kind === 'template' ? '在提示词库公开展示' : '在角色资源库公开展示'
    }}</a-checkbox>
    <details class="resource-advanced">
      <summary>高级选项（通常无需填写）</summary>
      <a-form-item label="链接标识（留空自动生成）"><a-input v-model="form.slug" /></a-form-item>
      <div class="creative-actions">
        <a-form-item label="排序"><a-input-number v-model="form.sort_order" /></a-form-item
        ><a-form-item label="状态"
          ><a-select v-model="form.status"
            ><a-option value="draft">草稿</a-option><a-option value="published">发布</a-option
            ><a-option value="archived">归档</a-option></a-select
          ></a-form-item
        >
      </div>
    </details>
    <div v-if="error" class="creative-error">{{ error }}</div>
    <a-button type="primary" size="small" :loading="busy" :disabled="uploading" @click="save"
      >保存资源</a-button
    ></a-form
  >
</template>
<style scoped>
.resource-upload {
  padding: var(--adm-gap-lg, 16px);
  margin-bottom: var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--creative-line));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-canvas, var(--creative-canvas));
}
.resource-previews {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: var(--adm-gap-lg, 16px);
  margin-bottom: var(--adm-gap-xl, 24px);
}
.resource-previews figure {
  margin: 0;
  min-width: 0;
}
.resource-previews .creative-image-tile {
  aspect-ratio: 1;
}
.resource-previews figcaption {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-sm, 8px);
  margin-top: var(--adm-gap-sm, 8px);
  color: var(--adm-muted, var(--creative-muted));
  font-size: var(--adm-font-xs, 12px);
}
.resource-advanced {
  margin: var(--adm-gap-lg, 16px) 0;
  color: var(--adm-muted, var(--creative-muted));
}
.resource-advanced summary {
  cursor: pointer;
  margin-bottom: var(--adm-gap-lg, 16px);
}
.resource-advanced summary:focus-visible {
  outline: 2px solid var(--adm-accent, var(--creative-accent));
  outline-offset: 3px;
}
.variable-editor {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-lg, 16px);
  background: var(--adm-canvas, var(--creative-canvas));
  border-radius: var(--adm-r-control, 10px);
  margin: var(--adm-gap-md, 12px) 0;
}
.template-preview {
  white-space: pre-wrap;
  background: var(--adm-canvas, var(--creative-canvas));
  padding: var(--adm-gap-lg, 16px);
  border-radius: var(--adm-r-control, 10px);
  line-height: 1.7;
}
.creative-actions > .arco-form-item {
  flex: 1;
  min-width: 130px;
}
@media (max-width: 768px) {
  .variable-editor {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
