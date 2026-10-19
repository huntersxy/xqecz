<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { IconLeft, IconRight, IconSearch, IconUser } from '@arco-design/web-vue/es/icon'
import { creativeApi } from '@/api/creative'
import type { CreativeResource } from '@/types/schemas'
import CreativeImage from './CreativeImage.vue'
import { resolveCreativeResource } from '@/utils/creativeResources'
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
  applyParameters = ref(false),
  loading = ref(true)
const controller = new AbortController()
const includeReferences = ref(false)
const resolved = computed(() =>
  selected.value
    ? resolveCreativeResource(selected.value, resources.value)
    : { assets: [], characters: [], text: '' },
)
const recommendedAssets = computed(() => resolved.value.assets)
const list = computed(() =>
  resources.value.filter(
    (r) =>
      (kind.value === 'character'
        ? ['character', 'preset'].includes(r.kind)
        : r.kind === kind.value) && `${r.title} ${r.summary}`.includes(search.value),
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
    text: selected.value.kind === 'template' ? preview.value : resolved.value.text,
    assets:
      selected.value.kind === 'template'
        ? includeReferences.value
          ? recommendedAssets.value
          : []
        : resolved.value.assets,
    characters: resolved.value.characters,
    parameters: applyParameters.value ? selected.value.payload.parameters : {},
    action,
  })
}
onMounted(async () => {
  try {
    resources.value = await creativeApi.resources(undefined, undefined, controller.signal)
  } catch (e) {
    if (!controller.signal.aborted) error.value = (e as Error).message
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
})
function back() {
  selected.value = null
  error.value = ''
}
onBeforeUnmount(() => controller.abort())
</script>
<template>
  <div class="creative-library" :class="{ 'is-detail': !!selected }">
    <Transition name="lib-pane" mode="out-in">
      <section v-if="!selected" key="list" class="lib-pane lib-list">
        <p class="lib-intro">
          选择角色会同时导入外观描述和参考图；提示词模板可以填写变量后追加、替换或插入到输入框。
        </p>
        <div class="lib-kind" role="radiogroup" aria-label="资源类型" :class="{ 'is-template': kind === 'template' }">
          <span class="lib-kind-thumb" aria-hidden="true" />
          <button type="button" role="radio" :aria-checked="kind === 'character'" @click="kind = 'character'">
            角色资源
          </button>
          <button type="button" role="radio" :aria-checked="kind === 'template'" @click="kind = 'template'">
            提示词
          </button>
        </div>
        <label class="lib-search">
          <IconSearch aria-hidden="true" />
          <input v-model="search" type="search" placeholder="查找创作资源" aria-label="查找创作资源" />
        </label>
        <div v-if="error" class="creative-error">{{ error }}</div>
        <div v-if="loading" class="lib-skeletons" role="status">
          <span class="sr-only">正在读取资源…</span>
          <i v-for="n in 4" :key="n" aria-hidden="true" />
        </div>
        <ul v-else-if="list.length" class="lib-cards">
          <li v-for="(row, index) in list" :key="row.id" :style="{ '--i': Math.min(index, 8) }">
            <button type="button" class="library-row" @click="select(row)">
              <span class="lib-thumb">
                <CreativeImage v-if="row.payload.asset_ids[0]" :id="row.payload.asset_ids[0]" public-asset />
                <IconUser v-else aria-hidden="true" />
              </span>
              <span class="lib-copy">
                <strong>{{ row.title }}</strong>
                <span>{{ row.summary || row.payload.description || '暂无简介' }}</span>
              </span>
              <IconRight class="lib-chevron" aria-hidden="true" />
            </button>
          </li>
        </ul>
        <p v-else class="lib-empty">{{ search ? '没有匹配的资源' : '尚无已发布资源' }}</p>
      </section>

      <section v-else key="detail" class="lib-pane lib-detail">
        <header class="lib-detail-head">
          <button type="button" class="lib-back" @click="back"><IconLeft />返回</button>
          <span class="lib-badge">{{ selected.kind === 'template' ? '提示词模板' : '角色资源' }}</span>
        </header>
        <div class="lib-detail-body">
          <h3>{{ selected.title }}</h3>
          <p class="lib-desc">{{ selected.payload.description || selected.summary }}</p>
          <div v-if="recommendedAssets.length" class="library-images">
            <CreativeImage v-for="id in recommendedAssets" :key="id" :id="id" public-asset />
          </div>
          <label v-for="variable in selected.payload.variables" :key="variable.name" class="lib-field">
            <span>{{ variable.label || variable.name }}<em v-if="variable.required">*</em></span>
            <select v-if="variable.type === 'select'" v-model="values[variable.name]">
              <option value="">请选择</option>
              <option v-for="option in variable.options" :key="option">{{ option }}</option>
            </select>
            <input v-else v-model="values[variable.name]" :placeholder="variable.help" />
          </label>
          <button v-if="selected.kind === 'template'" type="button" class="lib-btn" @click="expand">预览提示词</button>
          <pre v-if="preview" class="lib-preview">{{ preview }}</pre>
          <label v-if="selected.kind === 'template' && recommendedAssets.length" class="lib-switch">
            <input v-model="includeReferences" type="checkbox" />
            <span>同时导入推荐参考图（{{ recommendedAssets.length }} 张）</span>
          </label>
          <label v-if="Object.keys(selected.payload.parameters).length" class="lib-switch">
            <input v-model="applyParameters" type="checkbox" />
            <span>同时应用推荐参数</span>
          </label>
          <div v-if="error" class="creative-error">{{ error }}</div>
        </div>
        <footer class="lib-actions">
          <button type="button" class="lib-btn is-primary" @click="apply('append')">追加 / 导入</button>
          <template v-if="selected.kind === 'template'">
            <button type="button" class="lib-btn" @click="apply('replace')">替换提示词</button>
            <button type="button" class="lib-btn" @click="apply('insert')">插入光标位置</button>
          </template>
          <button v-if="preview" type="button" class="lib-btn" @click="copyPreview()">复制</button>
        </footer>
      </section>
    </Transition>
  </div>
</template>
<style scoped>
.creative-library {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 100%;
}
.lib-pane {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
.lib-pane-enter-active,
.lib-pane-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.26s cubic-bezier(0.22, 1, 0.36, 1);
}
.lib-pane-enter-from {
  opacity: 0;
  transform: translateX(18px);
}
.lib-pane-leave-to {
  opacity: 0;
  transform: translateX(-12px);
}
.lib-intro {
  margin: 0 0 14px;
  color: var(--creative-muted);
  font-size: 12px;
  line-height: 1.75;
}
.lib-kind {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 3px;
  border-radius: 999px;
  background: var(--creative-soft);
}
.lib-kind-thumb {
  position: absolute;
  top: 3px;
  bottom: 3px;
  left: 3px;
  width: calc((100% - 6px) / 2);
  border-radius: 999px;
  background: var(--creative-paper);
  box-shadow: var(--shadow-sm);
  transition: transform 0.34s cubic-bezier(0.22, 1, 0.36, 1);
}
.lib-kind.is-template .lib-kind-thumb {
  transform: translateX(100%);
}
.lib-kind button {
  position: relative;
  z-index: 1;
  min-height: 34px;
  border: 0;
  background: transparent;
  color: var(--creative-muted);
  font-size: 12.5px;
  cursor: pointer;
  transition: color 0.2s ease;
}
.lib-kind button[aria-checked='true'] {
  color: var(--creative-ink);
  font-weight: 600;
}
.lib-search {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 12px 0 14px;
  padding: 0 14px;
  border: 1px solid var(--creative-line);
  border-radius: 12px;
  background: var(--creative-paper);
  color: var(--creative-muted);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
.lib-search:focus-within {
  border-color: var(--creative-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 12%, transparent);
}
.lib-search input {
  flex: 1;
  min-width: 0;
  height: 40px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--creative-ink);
  font: inherit;
  font-size: 13px;
}
.lib-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.lib-cards li {
  animation: lib-rise 0.4s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i) * 36ms);
}
@keyframes lib-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}
.library-row {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--creative-line);
  border-radius: 16px;
  background: var(--creative-paper);
  color: var(--creative-ink);
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    box-shadow 0.25s ease,
    transform 0.25s cubic-bezier(0.22, 1, 0.36, 1);
}
.library-row:hover {
  border-color: color-mix(in srgb, var(--creative-accent) 40%, var(--creative-line));
  box-shadow: 0 10px 26px color-mix(in srgb, var(--creative-ink) 10%, transparent);
  transform: translateY(-2px);
}
.library-row:active {
  transform: translateY(0) scale(0.99);
}
.lib-thumb {
  display: grid;
  flex: 0 0 56px;
  place-items: center;
  width: 56px;
  height: 56px;
  overflow: hidden;
  border-radius: 13px;
  background: var(--creative-soft);
  color: var(--creative-muted);
}
.lib-thumb :deep(.creative-image) {
  min-height: 0;
}
.lib-thumb :deep(.creative-image img) {
  object-fit: cover;
}
.lib-thumb > svg {
  width: 22px;
  height: 22px;
}
.lib-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.lib-copy strong {
  overflow: hidden;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lib-copy span {
  display: -webkit-box;
  overflow: hidden;
  color: var(--creative-muted);
  font-size: 12px;
  line-height: 1.55;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.lib-chevron {
  flex: 0 0 auto;
  color: var(--creative-muted);
  transition: transform 0.25s cubic-bezier(0.22, 1, 0.36, 1), color 0.2s ease;
}
.library-row:hover .lib-chevron {
  color: var(--creative-accent);
  transform: translateX(3px);
}
.lib-empty {
  margin: 36px 0;
  color: var(--creative-muted);
  font-size: 13px;
  text-align: center;
}
.lib-skeletons {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.lib-skeletons i {
  height: 80px;
  border-radius: 16px;
  background: linear-gradient(
      105deg,
      transparent 30%,
      color-mix(in srgb, var(--creative-paper) 70%, transparent) 50%,
      transparent 70%
    )
    0 0 / 220% 100% no-repeat,
    var(--creative-soft);
  animation: lib-sheen 1.6s linear infinite;
}
@keyframes lib-sheen {
  from {
    background-position: 160% 0, 0 0;
  }
  to {
    background-position: -60% 0, 0 0;
  }
}

/* 详情：返回条 + 可滚动内容 + 吸底操作栏 */
.lib-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}
.lib-back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px 6px 8px;
  border: 0;
  border-radius: 999px;
  background: var(--creative-soft);
  color: var(--creative-ink);
  font-size: 12.5px;
  cursor: pointer;
  transition: background 0.2s ease, transform 0.2s ease;
}
.lib-back:hover {
  background: var(--creative-accent-soft, var(--creative-soft));
  transform: translateX(-2px);
}
.lib-badge {
  padding: 3px 10px;
  border-radius: 999px;
  background: var(--creative-accent-soft, var(--creative-soft));
  color: var(--creative-accent);
  font-size: 11px;
  font-weight: 600;
}
.lib-detail-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 14px;
  max-width: 640px;
}
.lib-detail h3 {
  margin: 0;
  color: var(--creative-ink);
  font: 600 22px/1.3 var(--site-title-font, inherit);
  letter-spacing: -0.01em;
}
.lib-desc {
  margin: -6px 0 0;
  color: var(--creative-muted);
  font-size: 13px;
  line-height: 1.8;
}
.library-images {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.library-images > * {
  height: 120px;
  overflow: hidden;
  border-radius: 14px;
  background: var(--creative-soft);
}
.library-images :deep(.creative-image) {
  min-height: 0;
}
.library-images :deep(.creative-image img) {
  object-fit: cover;
}
.lib-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--creative-muted);
  font-size: 12px;
}
.lib-field em {
  margin-left: 2px;
  color: var(--creative-accent);
  font-style: normal;
}
.lib-field input,
.lib-field select {
  height: 40px;
  padding: 0 12px;
  border: 1px solid var(--creative-line);
  border-radius: 12px;
  outline: 0;
  background: var(--creative-paper);
  color: var(--creative-ink);
  font: inherit;
  font-size: 13px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
.lib-field input:focus,
.lib-field select:focus {
  border-color: var(--creative-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 12%, transparent);
}
.lib-preview {
  margin: 0;
  padding: 14px 16px;
  border-radius: 14px;
  background: var(--creative-soft);
  color: var(--creative-ink);
  font: inherit;
  font-size: 13px;
  line-height: 1.8;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
/* 开关：仍是原生 checkbox（可聚焦、可读屏），只重绘外观。 */
.lib-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--creative-ink);
  font-size: 13px;
  cursor: pointer;
}
.lib-switch input {
  position: relative;
  flex: 0 0 auto;
  width: 36px;
  height: 22px;
  margin: 0;
  border-radius: 999px;
  background: color-mix(in srgb, var(--creative-muted) 35%, transparent);
  appearance: none;
  cursor: pointer;
  transition: background 0.25s ease;
}
.lib-switch input::after {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
  content: '';
  transition: transform 0.28s cubic-bezier(0.34, 1.4, 0.64, 1);
}
.lib-switch input:checked {
  background: var(--creative-accent);
}
.lib-switch input:checked::after {
  transform: translateX(14px);
}
.lib-switch input:focus-visible {
  outline: 2px solid var(--creative-accent);
  outline-offset: 2px;
}
.lib-actions {
  position: sticky;
  bottom: -24px;
  z-index: 2;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 18px -28px -24px;
  padding: 14px 28px 18px;
  border-top: 1px solid var(--creative-line);
  background: color-mix(in srgb, var(--creative-paper) 92%, transparent);
  backdrop-filter: blur(12px);
}
.lib-btn {
  min-height: 38px;
  padding: 0 16px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
  background: var(--creative-paper);
  color: var(--creative-ink);
  font-size: 12.5px;
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    background 0.2s ease,
    transform 0.15s ease;
}
.lib-btn:hover {
  border-color: var(--creative-accent);
  background: var(--creative-soft);
}
.lib-btn:active {
  transform: scale(0.97);
}
.lib-btn.is-primary {
  border-color: transparent;
  background: var(--creative-accent);
  color: var(--creative-paper);
  box-shadow: 0 6px 16px color-mix(in srgb, var(--creative-accent) 28%, transparent);
}
.lib-btn.is-primary:hover {
  background: color-mix(in srgb, var(--creative-accent) 86%, var(--creative-ink));
}
.lib-detail-body > .lib-btn {
  align-self: flex-start;
}
.creative-library button:focus-visible {
  outline: 2px solid var(--creative-accent);
  outline-offset: 2px;
}
@media (max-width: 820px) {
  .lib-cards,
  .lib-skeletons {
    grid-template-columns: minmax(0, 1fr);
  }
  .lib-actions {
    margin: 18px -14px -20px;
    padding: 12px 14px max(14px, env(safe-area-inset-bottom));
  }
  .library-images {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
