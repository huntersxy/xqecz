<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
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
let previewGeneration = 0
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
      await refresh() // Persist and show each successful conversion even if a later role fails.
    }
  })
}
watch(() => props.references, async refs => {
  const generation = ++previewGeneration
  for (const url of Object.values(previews.value)) URL.revokeObjectURL(url)
  previews.value = {}
  drafts.value = Object.fromEntries(refs.map(r => [r.name, { name: r.name, description: r.description || '' }]))
  await Promise.all(refs.map(async r => {
    try {
      const blob = await adminApi.visionReferenceImage(r.name, controller.signal)
      if (controller.signal.aborted || generation !== previewGeneration) return
      previews.value[r.name] = URL.createObjectURL(blob)
    } catch { /* Private previews are optional; the editable reference remains available. */ }
  }))
}, { immediate: true })
onBeforeUnmount(() => {
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
      <a-button :disabled="busy || !hasKey || !references.some(r => r.needs_description || !r.description)" @click="describeMissing">转述缺失 / 过期描述</a-button>
    </header>

    <a-alert v-if="error" type="error" :title="error" />

    <div class="reference-upload">
      <a-input v-model="name" :disabled="busy" :max-length="30" placeholder="角色名" aria-label="新参照角色名" />
      <input ref="uploadInput" type="file" accept="image/*" :disabled="busy" aria-label="选择参照图" @change="selected" />
      <a-button type="primary" :disabled="busy || !file || !name.trim() || references.length >= 12" :loading="busy" @click="upload">上传参照图</a-button>
    </div>
    <p class="reference-hint">选图后角色名自动取文件名，可先改名再上传；最多 12 张。</p>

    <AdminEmpty v-if="!references.length" title="尚未导入参照图" desc="选择角色设定图后点击上传，模型就能据此辨认角色。" compact />

    <div v-else class="reference-grid adm-stagger">
      <article v-for="r in references" :key="r.name" class="reference-card">
        <div class="reference-thumb">
          <img v-if="previews[r.name]" :src="previews[r.name]" :alt="r.name + '角色设定图'" />
          <span v-else class="reference-preview-empty">参照图预览不可用</span>
        </div>

        <div class="reference-meta">
          <strong class="reference-name">{{ r.name }}</strong>
          <span class="reference-size">{{ Math.ceil(r.bytes / 1024) }} KB</span>
          <AdminStatus :type="r.needs_description ? 'warning' : 'success'" :label="r.needs_description ? '描述待更新' : '描述已就绪'" />
        </div>

        <template v-if="drafts[r.name]">
          <a-input v-model="drafts[r.name]!.name" :disabled="busy" :max-length="30" :aria-label="r.name + '角色名'" />
          <div class="reference-actions">
            <a-button size="small" :disabled="busy || drafts[r.name]!.name === r.name" @click="run(async () => { data(await adminApi.renameVisionReference(r.name, drafts[r.name]!.name.trim(), controller.signal)) })">保存角色名</a-button>
            <label class="reference-replace">替换图片<input type="file" accept="image/*" :disabled="busy" :aria-label="'替换' + r.name + '图片'" @change="replace(r, $event)" /></label>
            <a-button size="small" status="danger" :disabled="busy" @click="remove(r)">移出</a-button>
          </div>
          <p class="reference-note" :class="{ 'reference-stale': r.needs_description }">{{ r.needs_description ? '文字描述缺失或已过期，请重新转述 / 编辑并保存' : '文字描述已就绪' }}</p>
          <a-textarea v-model="drafts[r.name]!.description" :disabled="busy" :max-length="2000" :auto-size="{ minRows: 4, maxRows: 8 }" :aria-label="r.name + '文字特征'" placeholder="发色、发型、瞳色、标志配饰等可见特征，勿填写剧情与性格" />
          <div class="reference-actions">
            <a-button size="small" :disabled="busy || !hasKey" @click="describe(r)">用模型转述</a-button>
            <a-button size="small" :disabled="busy" @click="run(async () => { data(await adminApi.saveVisionReferenceDescription(r, drafts[r.name]!.description, controller.signal)) })">保存文字描述</a-button>
          </div>
        </template>
      </article>
    </div>
  </section>
</template>

<style lang="scss" scoped>
// 分区容器：本组件嵌在「视觉模型」页的区二里，外框由本组件自己给（不依赖 AdminPanel）。
.reference-library {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  padding: var(--adm-gap-lg, 16px);
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-card, 16px);
  background: var(--adm-canvas, var(--admin-canvas));
  min-width: 0;
}

.reference-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  padding-bottom: var(--adm-gap-md, 12px);
  border-bottom: 1px solid var(--admin-border-soft);
}

.reference-head-text { min-width: 0; }

.reference-title {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  margin: 0;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-lg, 17px);
  font-weight: 600;
  letter-spacing: -.01em;
  color: var(--adm-ink, var(--admin-text));
}

.reference-count {
  padding: 1px 8px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  font-family: var(--site-body-font);
  font-size: var(--adm-font-xs, 12px);
  font-weight: 500;
  color: var(--adm-muted, var(--admin-text-3));
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}

.reference-desc {
  margin: 4px 0 0;
  max-width: 72ch;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  color: var(--adm-muted, var(--admin-text-3));

  &.is-warn { color: var(--adm-warning, var(--admin-warning)); }
}

.reference-hint {
  margin: 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  color: var(--adm-muted, var(--admin-text-3));
}

.reference-upload {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  padding: var(--adm-gap-md, 12px);
  border: 1px dashed var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-surface, var(--admin-surface));

  .arco-input-wrapper { max-width: 220px; }
  input { max-width: 100%; }
}

// 原生文件选择器在各家浏览器观感不一，统一成胶囊按钮，避免与 Arco 按钮并排时高低不齐。
.reference-library input[type='file'] {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

.reference-library input[type='file']::file-selector-button {
  min-height: 32px;
  margin-right: var(--adm-gap-sm, 8px);
  padding: 6px 12px;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-surface, var(--admin-surface));
  color: var(--adm-accent, var(--admin-primary));
  font-size: var(--adm-font-xs, 12px);
  cursor: pointer;
  transition: background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)),
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));

  &:hover {
    border-color: var(--adm-accent, var(--admin-primary));
    background: var(--adm-accent-soft, var(--admin-primary-soft));
  }
}

.reference-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--adm-gap-md, 12px);
}

.reference-card {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);
  min-width: 0;
  padding: var(--adm-gap-md, 12px);
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-surface, var(--admin-surface));
}

// 固定比例的画框：预览图 contain 不裁切，换图也不会引起卡片高度跳动。
.reference-thumb {
  display: grid;
  place-items: center;
  overflow: hidden;
  aspect-ratio: 4 / 3;
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-canvas, var(--admin-canvas));

  img { width: 100%; height: 100%; object-fit: contain; }
}

.reference-preview-empty {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
}

.reference-meta {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  flex-wrap: wrap;
  min-width: 0;
}

.reference-name {
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
  overflow-wrap: anywhere;
}

.reference-size {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}

.reference-actions {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  flex-wrap: wrap;
}

.reference-replace {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-accent, var(--admin-primary));
  cursor: pointer;

  input { width: 132px; font-size: var(--adm-font-xs, 12px); }
}

.reference-note {
  margin: 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));

  &.reference-stale { color: var(--adm-warning, rgb(var(--warning-6))); }
}

@media (max-width: 768px) {
  .reference-library { padding: var(--adm-gap-md, 12px); }
  .reference-grid { grid-template-columns: 1fr; }
  .reference-upload { align-items: stretch; flex-direction: column; }
  .reference-upload .arco-input-wrapper { max-width: none; }
}
</style>
