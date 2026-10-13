<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { adminApi } from '@/api'
import type { VisionReference, ApiResponse } from '@/types'
import { useConfirm, toast } from '@/composables/useToast'

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
  <section class="reference-library">
    <div class="reference-heading"><h3>角色参照库 <small>{{ references.length }} / 12</small></h3><a-button :disabled="busy || !hasKey || !references.some(r => r.needs_description || !r.description)" @click="describeMissing">转述缺失 / 过期描述</a-button></div>
    <p>角色名默认取文件名，可修改。图片上传后自动压缩至最长边 1024、512KB 内，仅管理员可预览。转述使用已保存的模型和提示词，首次转述会发送参照图。</p>
    <p v-if="!hasKey">先在上方保存模型 Key 即可自动转述，也可以直接手动填写文字特征。</p>
    <a-alert v-if="error" type="error" :title="error" />
    <div class="reference-upload">
      <a-input v-model="name" :disabled="busy" :max-length="30" placeholder="角色名" aria-label="新参照角色名" />
      <input ref="uploadInput" type="file" accept="image/*" :disabled="busy" aria-label="选择参照图" @change="selected" />
      <a-button type="primary" :disabled="busy || !file || !name.trim() || references.length >= 12" :loading="busy" @click="upload">上传参照图</a-button>
    </div>
    <p v-if="!references.length">尚未导入，选择角色设定图后点击上传。</p>
    <div class="reference-grid">
      <article v-for="r in references" :key="r.name" class="reference-card">
        <img v-if="previews[r.name]" :src="previews[r.name]" :alt="r.name + '角色设定图'" />
        <div v-else class="reference-preview-empty">参照图预览不可用</div>
        <div class="reference-heading"><strong>{{ r.name }}</strong><small>{{ Math.ceil(r.bytes / 1024) }} KB</small></div>
        <template v-if="drafts[r.name]">
          <a-input v-model="drafts[r.name]!.name" :disabled="busy" :max-length="30" :aria-label="r.name + '角色名'" />
          <div class="reference-actions">
            <a-button :disabled="busy || drafts[r.name]!.name === r.name" @click="run(async () => { data(await adminApi.renameVisionReference(r.name, drafts[r.name]!.name.trim(), controller.signal)) })">保存角色名</a-button>
            <label class="reference-replace">替换图片<input type="file" accept="image/*" :disabled="busy" :aria-label="'替换' + r.name + '图片'" @change="replace(r, $event)" /></label>
            <a-button status="danger" :disabled="busy" @click="remove(r)">移出</a-button>
          </div>
          <p :class="{ 'reference-stale': r.needs_description }">{{ r.needs_description ? '文字描述缺失或已过期，请重新转述 / 编辑并保存' : '文字描述已就绪' }}</p>
          <a-textarea v-model="drafts[r.name]!.description" :disabled="busy" :max-length="2000" :auto-size="{ minRows: 4, maxRows: 8 }" :aria-label="r.name + '文字特征'" placeholder="发色、发型、瞳色、标志配饰等可见特征，勿填写剧情与性格" />
          <div class="reference-actions">
            <a-button :disabled="busy || !hasKey" @click="describe(r)">用模型转述</a-button>
            <a-button :disabled="busy" @click="run(async () => { data(await adminApi.saveVisionReferenceDescription(r, drafts[r.name]!.description, controller.signal)) })">保存文字描述</a-button>
          </div>
        </template>
      </article>
    </div>
  </section>
</template>

<style scoped>
.reference-library { margin: 28px 0; padding-top: 20px; border-top: 1px solid var(--admin-border-soft); }
.reference-library p, small { color: var(--admin-text-3); font-size: 12px; line-height: 1.8; }
.reference-heading, .reference-actions, .reference-upload { display:flex; gap:10px; align-items:center; flex-wrap:wrap; margin:12px 0; }
.reference-heading { justify-content:space-between; }
.reference-upload .arco-input-wrapper { max-width:220px; }
.reference-upload input { max-width:100%; }
.reference-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(260px,1fr)); gap:16px; }
.reference-card { min-width:0; padding:16px; border:1px solid var(--admin-border-soft); border-radius:14px; background:var(--admin-surface); }
.reference-card img, .reference-preview-empty { width:100%; height:180px; object-fit:contain; border-radius:8px; background:var(--admin-canvas); }
.reference-preview-empty { display:grid; place-items:center; color:var(--admin-text-3); }
.reference-replace { cursor:pointer; font-size:12px; color:var(--admin-primary); }
.reference-replace input { display:block; width:160px; font-size:11px; }
.reference-card p.reference-stale { color:rgb(var(--warning-6)); }
@media(max-width:640px) { .reference-grid { grid-template-columns:1fr; } .reference-upload { align-items:stretch; flex-direction:column; } }
</style>
