<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { creativeApi } from '@/api/creative'
import type { CreativeResource } from '@/types/schemas'
import CreativeImage from './CreativeImage.vue'
const emit = defineEmits<{
  apply: [
    value: {
      text: string
      assets: number[]
      characters: number[]
      parameters: Record<string, unknown>
      action: 'append' | 'replace' | 'insert'
    },
  ]
}>()
const resources = ref<CreativeResource[]>([]),
  selected = ref<CreativeResource | null>(null),
  kind = ref('character'),
  search = ref(''),
  values = ref<Record<string, string>>({}),
  preview = ref(''),
  error = ref(''),
  applyParameters = ref(false)
const controller = new AbortController()
const includeReferences = ref(false)
const recommendedAssets = computed(() => [
  ...new Set([
    ...(selected.value?.payload.asset_ids ?? []),
    ...(resources.value.find(
      (r) => r.id === selected.value?.payload.preset_id && r.kind === 'preset',
    )?.payload.asset_ids ?? []),
  ]),
])
const list = computed(() =>
  resources.value.filter(
    (r) => r.kind === kind.value && `${r.title} ${r.summary}`.includes(search.value),
  ),
)
function select(row: CreativeResource) {
  selected.value = row
  error.value = ''
  values.value = Object.fromEntries(row.payload.variables.map((v) => [v.name, v.default]))
  preview.value = row.payload.body
  applyParameters.value = false
  includeReferences.value = false
}
async function expand() {
  if (!selected.value) return
  const id = selected.value.id,
    signature = JSON.stringify(values.value)
  try {
    const result = await creativeApi.expand(
      id,
      { ...values.value },
      selected.value.version,
      controller.signal,
    )
    if (controller.signal.aborted || selected.value?.id !== id) return
    if (signature !== JSON.stringify(values.value)) {
      error.value = '变量已经修改，请重新预览'
      return
    }
    preview.value = result.prompt
    error.value = ''
    return result.prompt
  } catch (e) {
    if (!controller.signal.aborted) error.value = (e as Error).message
  }
}
async function copyPreview() {
  try {
    await navigator.clipboard.writeText(preview.value)
  } catch {
    error.value = '复制失败，请手动选择提示词复制'
  }
}
async function apply(action: 'append' | 'replace' | 'insert') {
  if (!selected.value) return
  const id = selected.value.id
  if (selected.value.kind === 'template') {
    const text = await expand()
    if (typeof text !== 'string' || error.value || selected.value?.id !== id) return
  }
  emit('apply', {
    text: selected.value.kind === 'template' ? preview.value : selected.value.payload.appearance,
    assets:
      selected.value.kind === 'template'
        ? includeReferences.value
          ? recommendedAssets.value
          : []
        : selected.value.payload.asset_ids,
    characters:
      selected.value.kind === 'character'
        ? [selected.value.id]
        : selected.value.payload.character_ids,
    parameters: applyParameters.value ? selected.value.payload.parameters : {},
    action,
  })
}
onMounted(async () => {
  try {
    resources.value = await creativeApi.resources(undefined, undefined, controller.signal)
  } catch (e) {
    if (!controller.signal.aborted) error.value = (e as Error).message
  }
})
onBeforeUnmount(() => controller.abort())
</script>
<template>
  <div class="creative-library">
    <div>
      <a-radio-group v-model="kind" type="button"
        ><a-radio value="character">角色</a-radio><a-radio value="preset">参考组合</a-radio
        ><a-radio value="template">提示词</a-radio></a-radio-group
      ><a-input v-model="search" placeholder="查找创作资源" class="library-search" /><button
        v-for="row in list"
        :key="row.id"
        class="library-row"
        :class="{ active: selected?.id === row.id }"
        @click="select(row)"
      >
        <strong>{{ row.title }}</strong
        ><span>{{ row.summary }}</span>
      </button>
      <p v-if="!list.length">尚无已发布资源</p>
    </div>
    <div v-if="selected">
      <h3>{{ selected.title }}</h3>
      <p>{{ selected.payload.description || selected.summary }}</p>
      <div class="library-images">
        <CreativeImage v-for="id in selected.payload.asset_ids" :key="id" :id="id" public-asset />
      </div>
      <label
        v-for="variable in selected.payload.variables"
        :key="variable.name"
        class="creative-field"
        >{{ variable.label || variable.name }}{{ variable.required ? ' *' : ''
        }}<select v-if="variable.type === 'select'" v-model="values[variable.name]">
          <option value="">请选择</option>
          <option v-for="option in variable.options" :key="option">{{ option }}</option></select
        ><input v-else v-model="values[variable.name]" :placeholder="variable.help" /></label
      ><a-button v-if="selected.kind === 'template'" size="small" @click="expand"
        >预览提示词</a-button
      >
      <pre v-if="preview">{{ preview }}</pre>
      <a-checkbox
        v-if="selected.kind === 'template' && recommendedAssets.length"
        v-model="includeReferences"
        >同时导入推荐参考图（{{ recommendedAssets.length }} 张）</a-checkbox
      ><a-checkbox v-if="Object.keys(selected.payload.parameters).length" v-model="applyParameters"
        >同时应用推荐参数</a-checkbox
      >
      <div v-if="error" class="creative-error">{{ error }}</div>
      <div class="creative-actions">
        <a-button type="primary" @click="apply('append')">追加 / 导入</a-button
        ><a-button v-if="selected.kind === 'template'" @click="apply('replace')"
          >替换提示词</a-button
        ><a-button v-if="selected.kind === 'template'" @click="apply('insert')"
          >插入光标位置</a-button
        ><a-button v-if="preview" @click="copyPreview()">复制</a-button>
      </div>
    </div>
    <div v-else class="creative-empty">选择角色、参考组合或提示词模板</div>
  </div>
</template>
<style scoped>
.creative-library {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 24px;
}
.library-search {
  margin: 14px 0;
}
.library-row {
  display: grid;
  gap: 6px;
  text-align: left;
  width: 100%;
  border: 0;
  border: 1px solid var(--creative-line);
  border-radius: 12px;
  margin-bottom: 10px;
  background: var(--creative-paper);
  padding: 15px;
  color: var(--creative-ink);
  cursor: pointer;
}
.library-row span {
  font-size: 12px;
  color: var(--creative-muted);
}
.library-row.active {
  background: var(--creative-soft);
  border-color: var(--creative-accent);
}
.library-row:hover { background: var(--creative-soft); }
.creative-library > div { min-width: 0; }
.creative-library p { color: var(--creative-muted); line-height: 1.8; }
.creative-library h3 { color: var(--creative-ink); font-size: 20px; margin-bottom: 12px; }
.creative-library .arco-checkbox { display: flex; margin-top: 12px; }
.library-images {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.library-images > * {
  height: 110px;
}
.creative-library pre {
  white-space: pre-wrap;
  padding: 14px;
  background: var(--creative-canvas);
  border-radius: 8px;
  margin: 16px 0;
  color: var(--creative-ink);
  overflow-wrap: anywhere;
  font: inherit;
  line-height: 1.8;
}
.creative-library .creative-actions {
  margin-top: 16px;
}
@media (max-width: 650px) {
  .creative-library {
    grid-template-columns: 1fr;
  }
  .library-images {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
