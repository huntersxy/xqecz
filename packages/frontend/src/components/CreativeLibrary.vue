<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { IconCheck, IconClose, IconSearch, IconUser } from '@arco-design/web-vue/es/icon'
import { creativeApi } from '@/api/creative'
import type { CreativeResource } from '@/types/schemas'
import CreativeImage from './CreativeImage.vue'
import { resolveCreativeResource } from '@/utils/creativeResources'

interface ApplyPayload {
  text: string
  assets: number[]
  characters: number[]
  parameters: Record<string, unknown>
  action: 'append' | 'replace' | 'insert'
  label?: string
}

const props = withDefaults(defineProps<{ appliedCharacters?: number[]; error?: string }>(), {
  appliedCharacters: () => [],
  error: '',
})
const emit = defineEmits<{ apply: [value: ApplyPayload] }>()

const resources = ref<CreativeResource[]>([])
const kind = ref<'character' | 'template'>('character')
const search = ref('')
const loading = ref(true)
const failure = ref('')
const busyId = ref(0)
const fillId = ref(0)
const fillValues = ref<Record<string, string>>({})
const fillPanel = ref<HTMLElement>()
const controller = new AbortController()

const shownError = computed(() => props.error || failure.value)
const applied = computed(() => new Set(props.appliedCharacters))
const list = computed(() =>
  resources.value.filter(
    (row) =>
      (kind.value === 'character'
        ? ['character', 'preset'].includes(row.kind)
        : row.kind === 'template') &&
      `${row.title} ${row.summary} ${row.payload.description}`.includes(search.value),
  ),
)
const fillRow = computed(() => resources.value.find((row) => row.id === fillId.value) ?? null)
const fillReady = computed(
  () =>
    !!fillRow.value &&
    fillRow.value.payload.variables.every(
      (variable) => !variable.required || !!(fillValues.value[variable.name] ?? '').trim(),
    ),
)

function isApplied(row: CreativeResource) {
  return row.kind === 'character' && applied.value.has(row.id)
}
function defaultsOf(row: CreativeResource) {
  return Object.fromEntries(row.payload.variables.map((variable) => [variable.name, variable.default ?? '']))
}
async function activate(row: CreativeResource) {
  if (busyId.value || isApplied(row)) return
  failure.value = ''
  if (row.kind !== 'template') {
    const resolved = resolveCreativeResource(row, resources.value)
    emit('apply', {
      text: resolved.text,
      assets: resolved.assets,
      characters: resolved.characters,
      parameters: {},
      action: 'append',
      label: row.title,
    })
    return
  }
  if (row.payload.variables.some((variable) => variable.required && !(variable.default ?? '').trim())) {
    fillId.value = row.id
    fillValues.value = defaultsOf(row)
    await nextTick()
    fillPanel.value?.scrollIntoView?.({ block: 'nearest', behavior: 'smooth' })
    return
  }
  await expandAndApply(row, defaultsOf(row))
}
async function expandAndApply(row: CreativeResource, values: Record<string, string>) {
  busyId.value = row.id
  try {
    const result = await creativeApi.expand(row.id, values, row.version, controller.signal)
    if (controller.signal.aborted) return
    fillId.value = 0
    emit('apply', {
      text: result.prompt,
      assets: [],
      characters: [],
      parameters: {},
      action: 'append',
      label: row.title,
    })
  } catch (e) {
    if (!controller.signal.aborted) failure.value = (e as Error).message
  } finally {
    if (!controller.signal.aborted) busyId.value = 0
  }
}
function closeFill() {
  fillId.value = 0
  fillValues.value = {}
}
function applyFill() {
  if (fillRow.value && fillReady.value) void expandAndApply(fillRow.value, { ...fillValues.value })
}
onMounted(async () => {
  try {
    resources.value = await creativeApi.resources(undefined, undefined, controller.signal)
  } catch (e) {
    if (!controller.signal.aborted) failure.value = (e as Error).message
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
})
onBeforeUnmount(() => controller.abort())
</script>

<template>
  <div class="creative-library">
    <div class="lib-toolbar">
      <div class="lib-kind" :class="{ 'is-template': kind === 'template' }" role="radiogroup" aria-label="素材类型">
        <span class="lib-kind-thumb" aria-hidden="true" />
        <button
          type="button"
          role="radio"
          :aria-checked="kind === 'character'"
          @click="kind = 'character'"
        >
          角色资源
        </button>
        <button
          type="button"
          role="radio"
          :aria-checked="kind === 'template'"
          @click="kind = 'template'"
        >
          提示词
        </button>
      </div>
      <label class="lib-search">
        <IconSearch aria-hidden="true" />
        <input v-model="search" type="search" placeholder="查找创作素材" aria-label="查找创作素材" />
      </label>
    </div>

    <div v-if="shownError" class="creative-error" role="alert">{{ shownError }}</div>

    <form v-if="fillRow" ref="fillPanel" class="lib-fill" @submit.prevent="applyFill">
      <header class="lib-fill-head">
        <span><b>{{ fillRow.title }}</b> 还需要填写变量</span>
        <button type="button" class="lib-fill-close" aria-label="取消填写" @click="closeFill">
          <IconClose />
        </button>
      </header>
      <div class="lib-fill-fields">
        <label v-for="variable in fillRow.payload.variables" :key="variable.name" class="lib-field">
          <span>{{ variable.label || variable.name }}<em v-if="variable.required">*</em></span>
          <select
            v-if="variable.type === 'select'"
            v-model="fillValues[variable.name]"
            :aria-label="variable.label || variable.name"
          >
            <option value="">请选择</option>
            <option v-for="option in variable.options" :key="option">{{ option }}</option>
          </select>
          <input
            v-else
            v-model="fillValues[variable.name]"
            :placeholder="variable.help"
            :aria-label="variable.label || variable.name"
          />
        </label>
      </div>
      <button type="submit" class="lib-fill-apply" :disabled="!fillReady || busyId === fillRow.id">
        {{ busyId === fillRow.id ? '导入中…' : '导入提示词' }}
      </button>
    </form>

    <div v-if="loading" class="lib-skeletons" role="status">
      <span class="sr-only">正在读取素材…</span>
      <i v-for="n in 8" :key="n" aria-hidden="true" />
    </div>
    <ul v-else-if="list.length" class="lib-grid">
      <li v-for="(row, index) in list" :key="row.id" :style="{ '--i': Math.min(index, 11) }">
        <button
          type="button"
          class="lib-card"
          :class="{ 'is-applied': isApplied(row), 'is-template': row.kind === 'template' }"
          :disabled="isApplied(row) || busyId === row.id"
          :aria-label="
            isApplied(row) ? `${row.title}，已在当前草稿中` : `${row.title}，点击导入`
          "
          @click="activate(row)"
        >
          <span class="lib-media">
            <CreativeImage
              v-if="row.payload.asset_ids[0]"
              :id="row.payload.asset_ids[0]"
              public-asset
            />
            <span v-else class="lib-media-empty">
              <span v-if="row.kind === 'template'" class="lib-media-quote">
                {{ row.payload.body }}
              </span>
              <IconUser v-else aria-hidden="true" />
            </span>
            <span v-if="row.kind === 'template'" class="lib-kind-badge">提示词</span>
            <span class="lib-import-hint" aria-hidden="true">
              <template v-if="isApplied(row)"><IconCheck />已导入</template>
              <template v-else>{{ busyId === row.id ? '导入中…' : '导入' }}</template>
            </span>
          </span>
          <span class="lib-caption">
            <strong>{{ row.title }}</strong>
            <small>{{ row.summary || row.payload.description || '暂无简介' }}</small>
          </span>
        </button>
      </li>
    </ul>
    <div v-else class="lib-empty">
      <p>
        {{
          search
            ? '没有匹配的素材'
            : kind === 'template'
              ? '尚无已发布的提示词模板'
              : '尚无已发布的角色资源'
        }}
      </p>
      <button v-if="search" type="button" class="lib-reset" @click="search = ''">清除搜索</button>
    </div>
  </div>
</template>

<style scoped>
.creative-library {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.lib-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.lib-kind {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-width: 196px;
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
  box-shadow: 0 2px 8px color-mix(in srgb, var(--creative-ink) 10%, transparent);
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
  color: color-mix(in srgb, var(--creative-ink) 56%, var(--creative-paper));
  font: inherit;
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
  flex: 1 1 200px;
  align-items: center;
  gap: 8px;
  max-width: 300px;
  margin-left: auto;
  padding: 0 14px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
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
  height: 38px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--creative-ink);
  font: inherit;
  font-size: 13px;
}
.lib-search input::placeholder {
  color: color-mix(in srgb, var(--creative-ink) 52%, var(--creative-paper));
}
.lib-kind button:focus-visible,
.lib-search:focus-within {
  outline: 2px solid var(--creative-accent);
  outline-offset: 2px;
}

.lib-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 18px 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.lib-grid li {
  animation: lib-rise 0.42s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i) * 32ms);
}
@keyframes lib-rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
}
.lib-card {
  display: flex;
  flex-direction: column;
  gap: 9px;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--creative-ink);
  text-align: left;
  cursor: pointer;
}
.lib-media {
  position: relative;
  display: grid;
  place-items: center;
  width: 100%;
  aspect-ratio: 4 / 5;
  overflow: hidden;
  border: 1px solid var(--creative-line);
  border-radius: 16px;
  background:
    radial-gradient(
      120% 90% at 50% 0%,
      color-mix(in srgb, var(--creative-paper) 90%, transparent),
      transparent 72%
    ),
    var(--creative-soft);
  transition:
    border-color 0.24s ease,
    box-shadow 0.32s ease,
    transform 0.32s cubic-bezier(0.22, 1, 0.36, 1);
}
.lib-media :deep(.creative-image) {
  min-height: 0;
}
.lib-media :deep(.creative-image img) {
  object-fit: contain;
  transition: transform 0.55s cubic-bezier(0.22, 1, 0.36, 1);
}
.lib-media-empty {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  color: color-mix(in srgb, var(--creative-ink) 46%, var(--creative-paper));
}
.lib-media-empty > svg {
  width: 30px;
  height: 30px;
}
.lib-media-quote {
  display: -webkit-box;
  padding: 12% 14%;
  overflow: hidden;
  color: color-mix(in srgb, var(--creative-ink) 70%, var(--creative-paper));
  font: italic 400 12.5px/1.75 var(--creative-title-font);
  text-align: center;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
}
.lib-kind-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  padding: 3px 9px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
  background: var(--creative-paper);
  color: var(--creative-ink);
  font-size: 10.5px;
  letter-spacing: 0.06em;
}
.lib-import-hint {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 11px;
  border-radius: 999px;
  background: var(--creative-ink);
  color: var(--creative-paper);
  font-size: 11px;
  letter-spacing: 0.04em;
  opacity: 0;
  transform: translateY(5px);
  transition:
    opacity 0.22s ease,
    transform 0.3s cubic-bezier(0.22, 1, 0.36, 1);
}
.lib-import-hint svg {
  width: 12px;
  height: 12px;
}
.lib-card.is-applied .lib-import-hint {
  background: var(--creative-accent);
}
.lib-card:hover .lib-media,
.lib-card:focus-visible .lib-media {
  border-color: color-mix(in srgb, var(--creative-accent) 45%, var(--creative-line));
  box-shadow: 0 20px 36px -22px color-mix(in srgb, var(--creative-ink) 55%, transparent);
  transform: translateY(-3px);
}
.lib-card:hover .lib-media :deep(.creative-image img),
.lib-card:focus-visible .lib-media :deep(.creative-image img) {
  transform: scale(1.035);
}
.lib-card:active .lib-media {
  transform: translateY(-1px) scale(0.995);
}
.lib-card:hover .lib-import-hint,
.lib-card:focus-visible .lib-import-hint,
.lib-card.is-applied .lib-import-hint {
  opacity: 1;
  transform: none;
}
.lib-card:disabled {
  cursor: default;
}
.lib-card:disabled:hover .lib-media {
  transform: none;
}
.lib-card.is-applied .lib-media :deep(.creative-image img) {
  filter: saturate(0.72);
}
.lib-card:focus-visible {
  outline: none;
}
.lib-card:focus-visible .lib-media {
  outline: 2px solid var(--creative-accent);
  outline-offset: 3px;
}
.lib-caption {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  padding: 0 2px;
}
.lib-caption strong {
  overflow: hidden;
  font-size: 13.5px;
  font-weight: 600;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lib-caption small {
  overflow: hidden;
  color: color-mix(in srgb, var(--creative-ink) 60%, var(--creative-paper));
  font-size: 11.5px;
  line-height: 1.5;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lib-fill {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid color-mix(in srgb, var(--creative-accent) 26%, var(--creative-line));
  border-radius: 16px;
  background: color-mix(in srgb, var(--creative-accent) 5%, var(--creative-paper));
}
.lib-fill-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: color-mix(in srgb, var(--creative-ink) 66%, var(--creative-paper));
  font-size: 12px;
}
.lib-fill-head b {
  color: var(--creative-ink);
  font-size: 13px;
}
.lib-fill-close {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--creative-muted);
  cursor: pointer;
}
.lib-fill-close:hover {
  background: var(--creative-soft);
  color: var(--creative-ink);
}
.lib-fill-fields {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 10px;
}
.lib-field {
  display: flex;
  flex-direction: column;
  gap: 5px;
  color: color-mix(in srgb, var(--creative-ink) 62%, var(--creative-paper));
  font-size: 11.5px;
}
.lib-field em {
  margin-left: 2px;
  color: var(--creative-accent);
  font-style: normal;
}
.lib-field input,
.lib-field select {
  height: 36px;
  padding: 0 11px;
  border: 1px solid var(--creative-line);
  border-radius: 10px;
  outline: 0;
  background: var(--creative-paper);
  color: var(--creative-ink);
  font: inherit;
  font-size: 12.5px;
}
.lib-field input:focus,
.lib-field select:focus {
  border-color: var(--creative-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 12%, transparent);
}
.lib-fill-apply {
  align-self: flex-start;
  min-height: 34px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--creative-accent);
  color: var(--creative-paper);
  font: inherit;
  font-size: 12.5px;
  cursor: pointer;
  transition: background 0.2s ease, transform 0.15s ease;
}
.lib-fill-apply:hover:not(:disabled) {
  background: var(--creative-accent-hover);
}
.lib-fill-apply:active:not(:disabled) {
  transform: scale(0.97);
}
.lib-fill-apply:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.lib-skeletons {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 18px 14px;
}
.lib-skeletons i {
  aspect-ratio: 4 / 5;
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
.lib-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 56px 12px;
  color: color-mix(in srgb, var(--creative-ink) 58%, var(--creative-paper));
}
.lib-empty p {
  margin: 0;
  font-size: 13px;
}
.lib-reset {
  height: 34px;
  padding: 0 16px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
  background: var(--creative-paper);
  color: var(--creative-ink);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}
.lib-reset:hover {
  border-color: var(--creative-accent);
  color: var(--creative-accent);
}
@media (hover: none) {
  .lib-import-hint {
    opacity: 1;
    transform: none;
  }
}
@media (max-width: 820px) {
  .lib-search {
    flex-basis: 100%;
    max-width: none;
    margin-left: 0;
  }
  .lib-grid,
  .lib-skeletons {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px 10px;
  }
  .lib-import-hint {
    right: 7px;
    bottom: 7px;
    padding: 3px 9px;
  }
  .lib-fill-fields {
    grid-template-columns: minmax(0, 1fr);
  }
}
@media (prefers-reduced-motion: reduce) {
  .lib-grid li,
  .lib-skeletons i {
    animation: none;
  }
  .lib-media,
  .lib-import-hint,
  .lib-media :deep(.creative-image img) {
    transition: none;
  }
}
</style>
