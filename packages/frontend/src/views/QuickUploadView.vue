<script setup lang="ts">
import { useRouter } from 'vue-router'
import { toast } from '@/composables/useToast'
import { useFilePicker } from '@/composables/useFilePicker'
import { useAutoMetadata } from '@/composables/useAutoMetadata'
import AutoMetadataOption from '@/components/AutoMetadataOption.vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { CC_LICENSE_TEXT } from '@/utils/constants'
import { IconUpload, IconArrowRight, IconClose } from '@arco-design/web-vue/es/icon'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import TagCloud from '@/components/admin/TagCloud.vue'

const router = useRouter()
const userStore = useUserStore()
const isLoggedIn = computed(() => userStore.isLoggedIn)

const GUEST_STORAGE_KEY = 'xqecz_guest_identity'

const form = ref({
  nickname: '',
  email: '',
  title: '',
  content: '',
  tags: [] as string[],
})
const file = ref<File | undefined>(undefined)
const { autoMetadata, isAiGenerated, metadataOptions, canGenerate, metadataTitle, metadataTags } = useAutoMetadata(() => file.value)
const availableTags = ref<string[]>([])
const uploading = ref(false)
const progress = ref(0)
const { filePreview, fileInput, dragOver, onFileChange, onDrop, clearPreview } = useFilePicker((f) => {
  file.value = f
  if (!form.value.title) form.value.title = f.name.replace(/\.[^.]+$/, '')
}, { allowVideo: false })

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function toggleTag(tag: string) {
  const i = form.value.tags.indexOf(tag)
  if (i > -1) form.value.tags.splice(i, 1)
  else form.value.tags.push(tag)
}
function handleAddCustomTag(tag: string) {
  if (!form.value.tags.includes(tag)) form.value.tags.push(tag)
}

function removeFile() {
  file.value = undefined
  clearPreview()
}

function validate(): string | null {
  if (!isLoggedIn.value) {
    if (!form.value.nickname.trim()) return '请填写昵称'
    if (!EMAIL_RE.test(form.value.email.trim())) return '请填写正确的邮箱地址'
  }
  if (!autoMetadata.value && !form.value.title.trim()) return '请填写标题'
  if (!form.value.content.trim() && !file.value) return '请填写描述正文或上传媒体文件'
  return null
}

async function handleSubmit() {
  if (uploading.value) return
  const err = validate()
  if (err) {
    toast.warning(err)
    return
  }
  uploading.value = true
  progress.value = 0
  try {
    const res = await contentApi.quickUpload(
      {
        title: metadataTitle(form.value.title.trim()),
        auto_metadata: autoMetadata.value,
        is_ai_generated: isAiGenerated.value,
        nickname: isLoggedIn.value ? userStore.user?.username || '' : form.value.nickname.trim(),
        email: isLoggedIn.value ? userStore.user?.email || '' : form.value.email.trim(),
        content: form.value.content.trim() || undefined,
        tags: metadataTags(form.value.tags),
        file: file.value,
      },
      (p) => { progress.value = p },
    )
    if (res.code === 200) {
      localStorage.setItem(
        GUEST_STORAGE_KEY,
        JSON.stringify({ nickname: form.value.nickname.trim(), email: form.value.email.trim() }),
      )
      toast.success(autoMetadata.value ? '上传成功，标题与标签将在后台生成' : res.data.audit_status === 'approved' ? '上传成功' : '上传成功，审核通过后将进入推荐')
      router.push('/')
    } else {
      toast.error(res.message || '上传失败')
    }
  } catch (e) {
    toast.error((e as Error)?.message || '上传失败，请稍后重试')
  } finally {
    uploading.value = false
  }
}

onMounted(async () => {
  try {
    const saved = JSON.parse(localStorage.getItem(GUEST_STORAGE_KEY) || '{}')
    if (typeof saved.nickname === 'string') form.value.nickname = saved.nickname
    if (typeof saved.email === 'string') form.value.email = saved.email
  } catch {}

  try {
    const res = await contentApi.getTags()
    if (res.code === 200 && Array.isArray(res.data)) {
      availableTags.value = res.data
    }
  } catch {}
})
</script>

<template>
  <div class="qu-page creative-theme">
    <div class="qu-layout">
      <header class="qu-heading">
        <div><span class="qu-kicker">PUBLISH A PIECE <span aria-hidden="true">/</span></span><h1>发布一份作品<span>。</span></h1><p>把你的画面、文字和灵感，留在小泉作品广场。</p></div>
        <RouterLink to="/" class="qu-back">返回作品广场 <span aria-hidden="true">↗</span></RouterLink>
      </header>
      <a-form class="qu-card" layout="vertical" :model="form" @submit="handleSubmit">
        <div class="qu-body">
          <section class="qu-media" aria-labelledby="qu-media-title">
            <div class="qu-section-heading"><span class="qu-step">01</span><h2 id="qu-media-title">添加媒体</h2><span class="qu-optional">可选</span></div>
            <input ref="fileInput" type="file" class="qu-hidden" accept="image/*" :disabled="uploading" @change="onFileChange" />
            <Transition name="qu-media-state" mode="out-in">
            <button v-if="!file" key="dropzone" type="button" class="qu-dropzone" :class="{ 'is-dragover': dragOver }" :disabled="uploading" @click="fileInput?.click()" @dragover.prevent="!uploading && (dragOver = true)" @dragleave.prevent="dragOver = false" @drop.prevent="!uploading && onDrop($event)">
              <span class="qu-upload-art" aria-hidden="true"><IconUpload /><span>+</span></span>
              <strong>选择一张图片</strong><span>点击选择，或拖拽到这里</span><span class="qu-file-limit">支持图片 · 最大 20MB</span>
            </button>
            <div v-else key="preview" class="qu-preview">
              <div class="qu-preview-stage"><a-image :src="filePreview" :preview="false" class="qu-preview-img" alt="所选作品预览" /></div>
              <div class="qu-file-row"><span class="qu-file-name">{{ file.name }}</span><button type="button" class="qu-remove" aria-label="移除已选文件" :disabled="uploading" @click="removeFile"><IconClose /></button></div>
            </div>
            </Transition>
            <p class="qu-media-note"><span aria-hidden="true">/</span> 没有图片也没关系，一段文字同样可以发布。</p>
            <div class="qu-tip-card"><span class="qu-tip-kicker">A SMALL REMINDER</span><p>先把想法留下，<br>细节之后再慢慢完善<span>。</span></p><span class="qu-tip-spark" aria-hidden="true">+</span></div>
          </section>

          <section class="qu-fields" aria-labelledby="qu-story-title">
            <div class="qu-section-heading"><span class="qu-step">02</span><h2 id="qu-story-title">作品信息</h2></div>
            <div v-if="!isLoggedIn" class="qu-guest-grid"><a-form-item label="昵称" required><a-input v-model="form.nickname" placeholder="怎么称呼你？" :maxlength="50" :disabled="uploading" /></a-form-item><a-form-item label="邮箱" required><a-input v-model="form.email" placeholder="仅用于标识身份，不会公开" :maxlength="254" :disabled="uploading" /></a-form-item></div>
            <p v-else class="qu-identity"><span aria-hidden="true">/</span> 将以 {{ userStore.user?.username }} 的身份发布</p>
            <a-form-item><AutoMetadataOption v-model="autoMetadata" :options="metadataOptions" :can-generate="canGenerate" :disabled="uploading" /></a-form-item>
            <a-form-item><a-checkbox v-model="isAiGenerated" :disabled="uploading">这是 AI 生成的作品（添加 AIGC 标签）</a-checkbox></a-form-item>
            <a-form-item label="作品标题" required><a-input :model-value="metadataTitle(form.title)" @update:model-value="form.title = $event" placeholder="给这份灵感起个名字" :maxlength="200" :disabled="uploading || autoMetadata" /></a-form-item>
            <a-form-item label="描述正文（可选）"><MarkdownEditor v-model="form.content" placeholder="描述、提示词、灵感… 支持 Markdown 排版" :height="260" :disabled="uploading" /></a-form-item>
            <p class="qu-form-note">描述和媒体至少填写一项，作品审核后将进入推荐。</p>
            <a-form-item label="标签（可选）"><a-input-tag v-if="autoMetadata" :model-value="metadataTags(form.tags)" disabled /><div v-else class="qu-tags"><TagCloud :tags="availableTags" :selected-tags="form.tags" :max-tags="16" allow-custom :disabled="uploading" @toggle="toggleTag" @add="handleAddCustomTag" /></div></a-form-item>
          </section>
        </div>
        <footer class="qu-footer">
          <div class="qu-footer-copy"><div class="qu-license" v-html="CC_LICENSE_TEXT"></div><div v-if="uploading" class="qu-progress" aria-live="polite"><span>正在上传 {{ progress }}%</span><a-progress :percent="progress" :show-text="false" :stroke-width="4" /></div><p v-else><span aria-hidden="true">/</span> 发布后，作品会出现在作品广场。</p></div>
          <a-button type="primary" size="large" html-type="submit" class="qu-submit" :loading="uploading">{{ uploading ? '正在发布…' : '发布作品' }}<IconArrowRight v-if="!uploading" /></a-button>
        </footer>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
.qu-page {
  min-height: 100%;
  background: var(--creative-canvas);
  color: var(--creative-ink);
  padding: clamp(32px, 5vw, 70px) clamp(18px, 5vw, 72px) 80px;
}
.qu-layout { width: min(100%, 1180px); margin: 0 auto; }
.qu-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 28px; padding-bottom: 28px; border-bottom: 1px solid var(--creative-line); }
.qu-kicker { color: var(--creative-muted); font-size: 9px; letter-spacing: .15em; text-transform: uppercase; }
.qu-kicker > span { margin-left: 7px; color: var(--creative-accent); }
.qu-heading h1 { margin: 13px 0 10px; color: var(--creative-ink); font: 600 clamp(35px, 5vw, 56px)/1.03 var(--creative-title-font); letter-spacing: -.04em; }
.qu-heading h1 > span { color: var(--creative-accent); }
.qu-heading p { max-width: 34em; color: var(--creative-muted); font-size: 12px; line-height: 1.75; }
.qu-back { align-self: flex-end; padding: 0 0 3px; color: var(--creative-muted); font-size: 11px; text-decoration: none; white-space: nowrap; }
.qu-back:hover { color: var(--creative-accent); }
.qu-back span { margin-left: 5px; color: var(--creative-accent); }
.qu-card { margin-top: 28px; border: 1px solid var(--creative-line); border-radius: var(--radius-lg); background: var(--creative-paper); overflow: hidden; box-shadow: var(--shadow-sm); }
.qu-body { display: grid; grid-template-columns: minmax(240px, 320px) minmax(0, 1fr); gap: clamp(26px, 5vw, 72px); padding: clamp(24px, 4vw, 48px); }
.qu-section-heading { display: flex; align-items: center; gap: 10px; margin-bottom: 20px; }
.qu-step { display: grid; place-items: center; width: 25px; height: 25px; border: 1px solid var(--creative-line); border-radius: var(--radius-sm); background: var(--creative-soft); color: var(--creative-accent); font-size: 9px; font-variant-numeric: tabular-nums; }
.qu-section-heading h2 { margin: 0; color: var(--creative-ink); font: 600 21px/1.2 var(--creative-title-font); letter-spacing: -.02em; }
.qu-optional { margin-left: auto; color: var(--creative-muted); font-size: 10px; }
.qu-hidden { display: none; }
.qu-dropzone { width: 100%; min-height: 290px; padding: 30px 16px; border: 1px dashed color-mix(in srgb, var(--creative-accent) 38%, var(--creative-line)); border-radius: var(--radius-md); background: var(--creative-canvas); color: var(--creative-ink); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; cursor: pointer; transition: border-color .18s, background .18s; }
.qu-dropzone:hover:not(:disabled), .qu-dropzone.is-dragover { background: var(--creative-soft); border-color: var(--creative-accent); }
.qu-upload-art { position: relative; display: grid; place-items: center; width: 44px; height: 44px; margin: 4px 0 11px; border: 1px solid var(--creative-line); border-radius: 50%; background: var(--creative-paper); color: var(--creative-accent); }
.qu-upload-art > svg { font-size: 19px; }
.qu-upload-art > span { position: absolute; right: -8px; bottom: -8px; color: var(--creative-accent); font: 19px/1 var(--creative-title-font); }
.qu-dropzone strong { font-size: 13px; font-weight: 600; }
.qu-dropzone > span:not(.qu-upload-art) { color: var(--creative-muted); font-size: 10px; line-height: 1.8; }
.qu-file-limit { margin-top: 2px; }
.qu-media-note { display: flex; gap: 7px; margin: 13px 0 0; color: var(--creative-muted); font-size: 11px; line-height: 1.8; }
.qu-media-note > span { color: var(--creative-accent); }
.qu-tip-card { position: relative; margin-top: 32px; padding-top: 20px; border-top: 1px solid var(--creative-line); }
.qu-tip-kicker { color: var(--creative-muted); font-size: 8px; letter-spacing: .13em; }
.qu-tip-card p { margin: 11px 0 0; color: var(--creative-ink); font: 18px/1.7 var(--creative-title-font); }
.qu-tip-card p > span { color: var(--creative-accent); }
.qu-tip-spark { position: absolute; right: 4px; bottom: 4px; color: var(--creative-accent); font: 30px/1 var(--creative-title-font); }
.qu-fields { min-width: 0; }
.qu-guest-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.qu-fields :deep(.arco-form-item) { margin-bottom: 22px; }
.qu-fields :deep(.arco-form-item-label-col) { margin-bottom: 8px; }
.qu-fields :deep(.arco-form-item-label) { color: var(--creative-ink); font-size: 11px; font-weight: 600; }
.qu-fields :deep(.arco-input-wrapper) { min-height: 42px; border: 1px solid var(--creative-line); border-radius: var(--radius-sm); background: var(--creative-canvas); }
.qu-fields :deep(.arco-input-focus) { border-color: var(--creative-accent); background: var(--creative-paper); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 10%, transparent); }
.qu-fields :deep(.arco-input) { color: var(--creative-ink); font-size: 12px; }
.qu-fields :deep(.vditor) { border-color: var(--creative-line); border-radius: var(--radius-sm); background: var(--creative-paper); --toolbar-background-color: var(--creative-canvas); --textarea-background-color: var(--creative-paper); --border-color: var(--creative-line); --second-color: var(--creative-muted); }
.qu-fields :deep(.vditor-toolbar) { border-bottom-color: var(--creative-line); }
.qu-fields :deep(.vditor-toolbar button:hover) { color: var(--creative-accent); }
.qu-identity { display: flex; gap: 7px; margin: 0 0 22px; padding: 11px 13px; border: 1px solid var(--creative-line); border-radius: var(--radius-sm); background: var(--creative-soft); color: var(--creative-accent); font-size: 12px; }
.qu-form-note { margin: -10px 0 24px; color: var(--creative-muted); font-size: 10px; line-height: 1.8; }
.qu-tags { width: 100%; padding: 12px; border: 1px solid var(--creative-line); border-radius: var(--radius-sm); background: var(--creative-canvas); }
.qu-tags :deep(.tag-item) { background: var(--creative-paper); border-color: var(--creative-line); color: var(--creative-muted); font-size: 11px; padding: 5px 10px; }
.qu-tags :deep(.tag-item:hover), .qu-tags :deep(.tag-item.is-selected) { background: var(--creative-soft); border-color: var(--creative-accent); color: var(--creative-accent); }
.qu-preview { border: 1px solid var(--creative-line); border-radius: var(--radius-md); overflow: hidden; background: var(--creative-canvas); }
.qu-preview-stage { display: grid; place-items: center; height: 290px; padding: 14px; }
.qu-preview-img { width: 100%; }
.qu-preview-img :deep(.arco-image-img) { display: block; width: 100%; max-height: 262px; object-fit: contain; border-radius: var(--radius-sm); }
.qu-file-row { display: flex; align-items: center; gap: 8px; padding: 11px 12px; border-top: 1px solid var(--creative-line); background: var(--creative-paper); }
.qu-file-name { flex: 1; min-width: 0; overflow: hidden; color: var(--creative-ink); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.qu-remove { display: grid; place-items: center; width: 26px; height: 26px; padding: 0; border: 0; border-radius: 50%; background: var(--creative-canvas); color: var(--creative-muted); cursor: pointer; }
.qu-remove:hover { background: var(--creative-soft); color: var(--creative-accent); }
.qu-footer { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 20px clamp(24px, 4vw, 48px); border-top: 1px solid var(--creative-line); background: var(--creative-paper); }
.qu-footer-copy { min-width: 0; }
.qu-license { color: var(--creative-muted); font-size: 10px; line-height: 1.8; }
.qu-license :deep(a) { color: var(--creative-accent); }
.qu-footer-copy > p { margin: 8px 0 0; color: var(--creative-muted); font-size: 11px; }
.qu-footer-copy > p > span { margin-right: 6px; color: var(--creative-accent); }
.qu-submit { flex-shrink: 0; min-width: 130px; border-radius: var(--radius-full); gap: 12px; background: var(--creative-accent); color: #fff; border-color: var(--creative-accent); font-weight: 600; }
.qu-submit:hover { background: var(--creative-accent-hover); border-color: var(--creative-accent-hover); }
.qu-progress { max-width: 250px; margin-top: 9px; color: var(--creative-muted); font-size: 11px; }
button:disabled { cursor: not-allowed; }
@media (max-width: 1100px) { .qu-body { grid-template-columns: 260px minmax(0, 1fr); gap: 28px; } .qu-guest-grid { grid-template-columns: 1fr; gap: 0; } }
@media (max-width: 768px) { .qu-page { min-height: 100%; padding: 30px 20px 48px; } .qu-heading { align-items: flex-start; gap: 18px; } .qu-heading h1 { font-size: 37px; } .qu-heading p { max-width: 27em; font-size: 11px; } .qu-back { margin-top: 3px; font-size: 10px; } .qu-body { grid-template-columns: minmax(0, 1fr); gap: 30px; padding: 26px 22px; } .qu-section-heading { margin-bottom: 17px; } .qu-tip-card { display: none; } .qu-dropzone { min-height: 220px; } .qu-guest-grid { grid-template-columns: 1fr 1fr; gap: 12px; } .qu-footer { padding: 18px 22px; gap: 16px; } .qu-submit { min-width: 120px; } }
@media (max-width: 480px) { .qu-heading { flex-direction: column; gap: 14px; } .qu-heading h1 { font-size: 35px; } .qu-heading p { max-width: none; } .qu-card { border-radius: var(--radius-md); } .qu-body { padding: 22px 18px; } .qu-guest-grid { grid-template-columns: 1fr; gap: 0; } .qu-footer { flex-direction: column; align-items: stretch; padding: 18px; } .qu-submit { width: 100%; } }
.qu-media-state-enter-active, .qu-media-state-leave-active { transition: opacity .16s ease, transform .16s ease; }
.qu-media-state-enter-from { opacity: 0; transform: translateY(4px); }
.qu-media-state-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
