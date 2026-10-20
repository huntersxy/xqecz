<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { adminApi } from '@/api'
import type { VisionReference, ApiResponse } from '@/types'
import { useConfirm, toast } from '@/composables/useToast'
import AdminEmpty from './AdminEmpty.vue'
import AdminStatus from './AdminStatus.vue'

const props = defineProps<{ references: VisionReference[]; hasKey: boolean }>()
const emit = defineEmits<{ update: [VisionReference[]] }>()
const { confirm } = useConfirm()
const controller = new AbortController()
const busy = ref(false), error = ref(''), name = ref(''), file = ref<File>()
const uploadInput = ref<HTMLInputElement>()
const previews = ref<Record<string, string>>({})
const drafts = ref<Record<string, { name: string; description: string }>>({})
const selectedName = ref('')
const viewing = ref<VisionReference>()
let previewGeneration = 0

const active = computed<VisionReference | undefined>(() => props.references.find(ref => ref.name === selectedName.value) ?? props.references[0])
const canAdd = computed(() => props.references.length < 12)
const staleCount = computed(() => props.references.filter(ref => ref.needs_description || !ref.description).length)

function data<T>(res: ApiResponse<T>): T { if (res.code !== 200) throw new Error(res.message); return res.data }
async function refresh() {
  const res = data(await adminApi.visionReferences(controller.signal))
  if (!controller.signal.aborted) emit('update', res)
}
async function run(action: () => Promise<void>) {
  if (busy.value || controller.signal.aborted) return
  busy.value = true; error.value = ''
  try { await action(); if (!controller.signal.aborted) await refresh() }
  catch (e) { if (!controller.signal.aborted) error.value = (e as Error).message || '操作失败' }
  finally { if (!controller.signal.aborted) busy.value = false }
}
function selected(event: Event) {
  file.value = (event.target as HTMLInputElement).files?.[0]
  if (file.value && !name.value) name.value = file.value.name.replace(/\.[^.]+$/, '')
}
async function upload() {
  if (!file.value || !name.value.trim()) return
  await run(async () => {
    data(await adminApi.uploadVisionReference(name.value.trim(), file.value!, false, controller.signal))
    if (!controller.signal.aborted) { name.value = ''; file.value = undefined; if (uploadInput.value) uploadInput.value.value = ''; toast.success('参照图已压缩并保存') }
  })
}
async function replace(ref: VisionReference, event: Event) {
  const input = event.target as HTMLInputElement, selectedFile = input.files?.[0]; input.value = ''
  if (!selectedFile) return
  await run(async () => { data(await adminApi.uploadVisionReference(ref.name, selectedFile, true, controller.signal)) })
}
async function remove(ref: VisionReference) {
  if (!await confirm(`将“${ref.name}”移出参照库？服务器会保留图片备份。`) || controller.signal.aborted) return
  await run(async () => { data(await adminApi.deleteVisionReference(ref.name, controller.signal)) })
}
async function describe(ref: VisionReference) {
  await run(async () => { data(await adminApi.describeVisionReference(ref, controller.signal)) })
}
async function describeMissing() {
  await run(async () => {
    for (const ref of props.references.filter(r => r.needs_description || !r.description)) {
      if (controller.signal.aborted) return
      data(await adminApi.describeVisionReference(ref, controller.signal))
      await refresh()
    }
  })
}
async function saveName(ref: VisionReference) {
  await run(async () => { data(await adminApi.renameVisionReference(ref.name, drafts.value[ref.name]!.name.trim(), controller.signal)) })
}
async function saveDescription(ref: VisionReference) {
  await run(async () => { data(await adminApi.saveVisionReferenceDescription(ref, drafts.value[ref.name]!.description, controller.signal)) })
}
function select(ref: VisionReference) { selectedName.value = ref.name }
function openPreview(ref: VisionReference) { viewing.value = ref }
function closePreview() { viewing.value = undefined }
function onKeydown(event: KeyboardEvent) { if (event.key === 'Escape' && viewing.value) closePreview() }
watch(() => props.references, async refs => {
  const generation = ++previewGeneration
  for (const url of Object.values(previews.value)) URL.revokeObjectURL(url)
  previews.value = {}
  drafts.value = Object.fromEntries(refs.map(r => [r.name, { name: r.name, description: r.description || '' }]))
  if (!refs.some(r => r.name === selectedName.value)) selectedName.value = refs[0]?.name ?? ''
  await Promise.all(refs.map(async r => {
    try {
      const blob = await adminApi.visionReferenceImage(r.name, controller.signal)
      if (controller.signal.aborted || generation !== previewGeneration) return
      previews.value[r.name] = URL.createObjectURL(blob)
    } catch { }
  }))
}, { immediate: true })
onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  controller.abort(); ++previewGeneration
  for (const url of Object.values(previews.value)) URL.revokeObjectURL(url)
})
</script>

<template>
  <section class="reference-library" aria-labelledby="reference-title">
    <header class="reference-head">
      <div class="reference-head-text">
        <h3 id="reference-title" class="reference-title">
          角色参照库
          <span class="reference-count">{{ references.length }} / 12</span>
        </h3>
        <p class="reference-desc">角色名默认取文件名，可修改。图片上传后自动压缩至最长边 1024、512KB 内，仅管理员可预览。转述使用已保存的模型和提示词，首次转述会发送参照图。</p>
        <p v-if="!hasKey" class="reference-desc is-warn">先在上方保存模型 Key 即可自动转述，也可以直接手动填写文字特征。</p>
      </div>
      <a-button :disabled="busy || !hasKey || !staleCount" @click="describeMissing">转述缺失 / 过期描述</a-button>
    </header>

    <a-alert v-if="error" type="error" :title="error" />

    <div class="reference-uploader">
      <a-input v-model="name" :disabled="busy" :max-length="30" placeholder="角色名（默认取文件名）" aria-label="新参照角色名" />
      <a-button type="primary" :disabled="busy || !file || !name.trim() || !canAdd" :loading="busy" @click="upload">上传参照图</a-button>
      <span class="reference-hint">选图后角色名自动取文件名，可先改名再上传；最多 12 张。</span>
    </div>

    <AdminEmpty v-if="!references.length" title="尚未导入参照图" desc="选择角色设定图后点击上传，模型就能据此辨认角色。" compact />

    <div class="reference-grid adm-stagger">
      <article
        v-for="r in references"
        :key="r.name"
        class="reference-tile"
        :class="{ 'is-active': active?.name === r.name }"
      >
        <div class="reference-frame">
          <button type="button" class="reference-thumb" :aria-label="`预览${r.name}的角色设定图`" @click="openPreview(r)">
            <img v-if="previews[r.name]" :src="previews[r.name]" :alt="r.name + '角色设定图'" />
            <span v-else class="reference-thumb-empty">预览不可用</span>
          </button>

          <div class="reference-tools">
            <label class="reference-tool">
              替换
              <input type="file" accept="image/*" :disabled="busy" :aria-label="'替换' + r.name + '图片'" @change="replace(r, $event)" />
            </label>
            <button type="button" class="reference-tool is-danger" :disabled="busy" :aria-label="`将${r.name}移出参照库`" @click="remove(r)">移出</button>
            <button type="button" class="reference-tool" :disabled="busy || !hasKey" :aria-label="`用模型转述${r.name}`" @click="describe(r)">转述</button>
          </div>
        </div>

        <div class="reference-meta">
          <button
            type="button"
            class="reference-name"
            :class="{ 'is-active': active?.name === r.name }"
            :aria-pressed="active?.name === r.name"
            @click="select(r)"
          >{{ r.name }}</button>
          <span class="reference-size">{{ Math.ceil(r.bytes / 1024) }} KB</span>
        </div>
        <AdminStatus :type="r.needs_description ? 'warning' : 'success'" :label="r.needs_description ? '描述待更新' : '描述已就绪'" />
      </article>

      <label v-if="canAdd" class="reference-add">
        <input ref="uploadInput" type="file" accept="image/*" :disabled="busy" aria-label="选择参照图" @change="selected" />
        <span class="reference-add-mark" aria-hidden="true">＋</span>
        <span class="reference-add-text">上传参考图</span>
        <span class="reference-add-hint">3:4 竖图 · 12 张上限</span>
      </label>
    </div>

    <section v-if="active" class="reference-editor" :aria-label="active.name + '的文字特征'">
      <header class="reference-editor-head">
        <h4 class="reference-editor-title">{{ active.name }} · 文字特征</h4>
        <span class="reference-size">{{ Math.ceil(active.bytes / 1024) }} KB</span>
        <AdminStatus :type="active.needs_description ? 'warning' : 'success'" :label="active.needs_description ? '描述待更新' : '描述已就绪'" />
      </header>

      <div class="reference-editor-row">
        <div class="reference-field">
          <label class="reference-label" :for="'reference-name-' + active.name">角色名</label>
          <a-input :id="'reference-name-' + active.name" v-model="drafts[active.name]!.name" :disabled="busy" :max-length="30" :aria-label="active.name + '角色名'" />
        </div>
        <a-button size="small" :disabled="busy || drafts[active.name]!.name === active.name" @click="saveName(active)">保存角色名</a-button>
      </div>

      <p class="reference-note" :class="{ 'reference-stale': active.needs_description }">{{ active.needs_description ? '文字描述缺失或已过期，请重新转述 / 编辑并保存' : '文字描述已就绪' }}</p>

      <div class="reference-field">
        <label class="reference-label" :for="'reference-desc-' + active.name">文字特征</label>
        <a-textarea
          :id="'reference-desc-' + active.name"
          v-model="drafts[active.name]!.description"
          :disabled="busy"
          :max-length="2000"
          :auto-size="{ minRows: 4, maxRows: 8 }"
          :aria-label="active.name + '文字特征'"
          placeholder="发色、发型、瞳色、标志配饰等可见特征，勿填写剧情与性格"
        />
      </div>

      <div class="reference-editor-actions">
        <a-button size="small" :disabled="busy || !hasKey" @click="describe(active)">用模型转述</a-button>
        <a-button size="small" :disabled="busy" @click="saveDescription(active)">保存文字描述</a-button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="viewing" class="reference-viewer" role="dialog" aria-modal="true" :aria-label="viewing.name + '角色设定图预览'">
        <button type="button" class="reference-viewer-backdrop" aria-label="关闭预览" @click="closePreview" />
        <figure class="reference-viewer-panel">
          <img v-if="previews[viewing.name]" :src="previews[viewing.name]" :alt="viewing.name + '角色设定图'" />
          <span v-else class="reference-thumb-empty">参照图预览不可用</span>
          <figcaption class="reference-viewer-meta">
            <strong class="reference-viewer-name">{{ viewing.name }}</strong>
            <span class="reference-size">{{ Math.ceil(viewing.bytes / 1024) }} KB</span>
            <AdminStatus :type="viewing.needs_description ? 'warning' : 'success'" :label="viewing.needs_description ? '描述待更新' : '描述已就绪'" />
            <a-button size="small" class="reference-viewer-close" @click="closePreview">关闭</a-button>
          </figcaption>
        </figure>
      </div>
    </Teleport>
  </section>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.reference-library {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px) var(--adm-gap-lg, 16px);
}

.reference-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
}

.reference-head-text { min-width: 0; }

.reference-title {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-family: var(--site-title-font);
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  letter-spacing: -.01em;
}

.reference-count {
  padding: 1px 8px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-3));
  font-family: var(--site-body-font);
  font-size: var(--adm-font-xs, 12px);
  font-weight: 500;
  @include numeric;
}

.reference-desc {
  margin: 4px 0 0;
  max-width: 72ch;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;

  &.is-warn { color: var(--adm-warning, var(--admin-warning)); }
}

.reference-uploader {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-sm, 8px) var(--adm-gap-md, 12px);
  border: 1px dashed var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-surface, var(--admin-surface));

  :deep(.arco-input-wrapper) { max-width: 220px; }
}

.reference-hint {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
}

.reference-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: var(--adm-gap-md, 12px);
}

.reference-tile {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.reference-frame {
  position: relative;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--admin-fill));
  transition:
    border-color var(--adm-dur-base, 220ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)),
    box-shadow var(--adm-dur-base, 220ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1));
}

.reference-tile.is-active .reference-frame {
  border-color: var(--adm-accent, var(--admin-primary));
  box-shadow: 0 0 0 1px var(--adm-accent, var(--admin-primary));
}

.reference-thumb {
  display: grid;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  background: none;
  place-items: center;
  cursor: zoom-in;
  @include focus-ring(2px);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.reference-thumb-empty {
  padding: 0 6px;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.5;
  text-align: center;
}

.reference-tools {
  position: absolute;
  right: 6px;
  bottom: 6px;
  left: 6px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  opacity: 0;
  transform: translateY(4px);
  transition:
    opacity var(--adm-dur-fast, 120ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1)),
    transform var(--adm-dur-fast, 120ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1));
}

.reference-frame:hover .reference-tools,
.reference-frame:focus-within .reference-tools {
  opacity: 1;
  transform: none;
}

.reference-tool {
  position: relative;
  display: inline-flex;
  flex: 1 1 auto;
  align-items: center;
  justify-content: center;
  min-height: 26px;
  padding: 2px 6px;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-pill, 999px);
  background: color-mix(in srgb, var(--adm-surface, var(--admin-surface)) 90%, transparent);
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.4;
  white-space: nowrap;
  cursor: pointer;
  @include focus-ring(2px);

  &:disabled {
    color: var(--adm-muted, var(--admin-text-3));
    cursor: not-allowed;
  }

  &.is-danger { color: var(--adm-danger, var(--admin-danger)); }

  input[type='file'] {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    margin: 0;
    opacity: 0;
    cursor: pointer;
  }
}

.reference-add {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  aspect-ratio: 3 / 4;
  padding: var(--adm-gap-sm, 8px);
  border: 1px dashed var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  background: transparent;
  color: var(--adm-muted, var(--admin-text-3));
  text-align: center;
  cursor: pointer;
  transition: border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));

  &:hover {
    border-color: var(--adm-accent, var(--admin-primary));
    color: var(--adm-accent, var(--admin-primary));
  }

  input[type='file'] {
    position: absolute;
    width: 1px;
    height: 1px;
    opacity: 0;
    pointer-events: none;
  }
}

.reference-add-mark {
  font-size: 20px;
  line-height: 1;
}

.reference-add-text {
  font-size: var(--adm-font-sm, 13px);
  font-weight: 500;
}

.reference-add-hint {
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.5;
  opacity: .8;
}

.reference-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
  min-width: 0;
}

.reference-name {
  padding: 0;
  border: 0;
  background: none;
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  overflow-wrap: anywhere;
  @include focus-ring(2px);

  &.is-active { color: var(--adm-accent, var(--admin-primary)); }
}

.reference-size {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.reference-editor {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-md, 12px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--admin-fill));
}

.reference-editor-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
}

.reference-editor-title {
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-family: var(--site-title-font);
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
}

.reference-editor-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: var(--adm-gap-sm, 8px);
}

.reference-field {
  display: flex;
  flex: 1 1 200px;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.reference-label {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

.reference-note {
  margin: 0;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;

  &.reference-stale { color: var(--adm-warning, rgb(var(--warning-6))); }
}

.reference-editor-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-sm, 8px);
}

.reference-viewer {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: grid;
  place-items: center;
  padding: var(--adm-gap-lg, 16px);
}

.reference-viewer-backdrop {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  background: color-mix(in srgb, var(--adm-ink, var(--admin-text)) 62%, transparent);
  cursor: zoom-out;
}

.reference-viewer-panel {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);
  max-width: min(92vw, 720px);
  max-height: 88vh;
  margin: 0;
  padding: var(--adm-gap-md, 12px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-surface, var(--admin-surface));

  img {
    max-width: 100%;
    max-height: 72vh;
    border-radius: var(--adm-r-control, 10px);
    object-fit: contain;
  }
}

.reference-viewer-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
}

.reference-viewer-name {
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
}

.reference-viewer-close { margin-left: auto; }

@media (hover: none) {
  .reference-tools { opacity: 1; transform: none; }
}

@media (max-width: 768px) {
  .reference-library { padding: var(--adm-gap-md, 12px); }
  .reference-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .reference-uploader { align-items: stretch; flex-direction: column; }
  .reference-uploader :deep(.arco-input-wrapper) { max-width: none; }

  .reference-tools {
    opacity: 1;
    transform: none;
  }

  .reference-tool { min-height: var(--adm-touch, 44px); }
  .reference-name { min-height: var(--adm-touch, 44px); }
  .reference-viewer-close { min-height: var(--adm-touch, 44px); }
}

@media (max-width: 560px) {
  .reference-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .reference-editor-row { align-items: stretch; flex-direction: column; }
  .reference-viewer { padding: var(--adm-gap-sm, 8px); }
}
</style>
