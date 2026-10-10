<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from '@/composables/useToast'
import { contentApi } from '@/api'
import { formatFileSize } from '@/utils'
import { useUserStore } from '@/stores/user'
import { useFilePicker } from '@/composables/useFilePicker'
import { IconUpload, IconClose, IconArrowRight, IconCheckCircle } from '@arco-design/web-vue/es/icon'

interface Props {
  open: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: []; uploaded: [] }>()

const router = useRouter()
const userStore = useUserStore()
const isLoggedIn = computed(() => userStore.isLoggedIn)

const visible = ref(props.open)
watch(
  () => props.open,
  v => {
    if (v !== visible.value) visible.value = v
  },
)
watch(visible, v => {
  if (!v) emit('close')
})

const GUEST_STORAGE_KEY = 'xqecz_guest_identity'

const form = ref({
  nickname: '',
  email: '',
  title: '',
  content: '',
})
const file = ref<File | undefined>(undefined)
const uploading = ref(false)
const progress = ref(0)
const { filePreview, fileInput, dragOver, onFileChange, onDrop, clearPreview } = useFilePicker((f) => {
  file.value = f
  if (!form.value.title) form.value.title = f.name.replace(/\.[^.]+$/, '')
})

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s]+$/

function removeFile() {
  if (uploading.value) return
  file.value = undefined
  clearPreview()
}

function validate(): string | null {
  if (!isLoggedIn.value) {
    if (!form.value.nickname.trim()) return '请填写昵称'
    if (!EMAIL_RE.test(form.value.email.trim())) return '请填写正确的邮箱地址'
  }
  if (!form.value.title.trim()) return '请填写标题'
  if (!form.value.content.trim() && !file.value) return '请填写描述或上传文件'
  return null
}

async function handleSubmit() {
  if (uploading.value) return
  const err = validate()
  if (err) { toast.warning(err); return }

  uploading.value = true
  progress.value = 0
  try {
    const res = await contentApi.quickUpload(
      {
        title: form.value.title.trim(),
        nickname: isLoggedIn.value ? userStore.user?.username || '' : form.value.nickname.trim(),
        email: isLoggedIn.value ? userStore.user?.email || '' : form.value.email.trim(),
        content: form.value.content.trim() || undefined,
        file: file.value,
      },
      (p) => { progress.value = p },
    )
    if (res.code === 200) {
      localStorage.setItem(
        GUEST_STORAGE_KEY,
        JSON.stringify({ nickname: form.value.nickname.trim(), email: form.value.email.trim() }),
      )
      toast.success(res.data.audit_status === 'approved' ? '上传成功' : '上传成功，审核通过后将进入推荐')
      emit('uploaded')
      visible.value = false
      router.push('/')
    } else {
      toast.error(res.message || '上传失败')
    }
  } catch (e) {
    toast.error((e as Error)?.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

function loadGuestInfo() {
  try {
    const saved = JSON.parse(localStorage.getItem(GUEST_STORAGE_KEY) || '{}')
    if (typeof saved.nickname === 'string') form.value.nickname = saved.nickname
    if (typeof saved.email === 'string') form.value.email = saved.email
  } catch {}
}

watch(() => props.open, (val) => {
  if (val) {
    loadGuestInfo()
    form.value.title = ''
    form.value.content = ''
    file.value = undefined
    clearPreview()
  }
}, { immediate: true })
</script>

<template>
  <a-drawer
    v-model:visible="visible"
    title="发布新作品"
    placement="bottom"
    height="auto"
    :header="false"
    :footer="false"
    :mask-closable="!uploading"
    :esc-to-close="!uploading"
    class="qu-sheet creative-theme"
    :style="{ zIndex: 2000 }"
  >
    <form class="qus-panel" aria-labelledby="qus-title" @submit.prevent="handleSubmit">
      <header class="qus-heading">
        <div>
          <span class="qus-kicker">CREATE & SHARE <span aria-hidden="true">✦</span></span>
          <h2 id="qus-title">让灵感，在这里落笔<span class="qus-title-dot">。</span></h2>
          <p>一段文字，一张画面，都可以是故事的开始。</p>
        </div>
        <button type="button" class="qus-close" aria-label="关闭上传窗口" :disabled="uploading" @click="visible = false"><IconClose /></button>
      </header>

      <div class="qus-body">
        <section class="qus-media-column" aria-label="作品媒体">
          <div class="qus-field-heading"><span>作品画面</span><span class="qus-optional">可选</span></div>
          <input ref="fileInput" type="file" accept="image/*,video/*" class="qus-hidden" :disabled="uploading" @change="onFileChange" />
          <button
            v-if="!file"
            type="button"
            class="qus-dropzone"
            :class="{ 'is-dragover': dragOver }"
            :disabled="uploading"
            @click="fileInput?.click()"
            @dragover.prevent="!uploading && (dragOver = true)"
            @dragleave.prevent="dragOver = false"
            @drop.prevent="!uploading && onDrop($event)"
          >
            <span class="qus-upload-art" aria-hidden="true"><IconUpload /><span class="qus-art-spark">✦</span></span>
            <span class="qus-dropzone-title">把你的灵感放进来</span>
            <span class="qus-dropzone-hint">点击选择，或将图片 / 视频拖到这里</span>
            <span class="qus-file-limit">原格式上传 · 单个文件不超过 20 MB</span>
          </button>
          <div v-else class="qus-preview">
            <div class="qus-preview-stage">
              <img v-if="file.type.startsWith('image/')" :src="filePreview" :alt="file.name" class="qus-preview-img" />
              <video v-else :src="filePreview" controls class="qus-preview-video" />
            </div>
            <div class="qus-file-row">
              <IconCheckCircle class="qus-file-check" />
              <span class="qus-file-name">{{ file.name }}</span>
              <span class="qus-file-size">{{ formatFileSize(file.size) }}</span>
              <button type="button" class="qus-remove" aria-label="移除已选文件" :disabled="uploading" @click="removeFile"><IconClose /></button>
            </div>
          </div>
          <p class="qus-media-note"><span aria-hidden="true">✧</span> 没有图片也没关系，让文字成为主角。</p>
        </section>

        <section class="qus-fields" aria-label="作品信息">
          <div v-if="!isLoggedIn" class="qus-grid">
            <label class="qus-field"><span>你的昵称 <span class="qus-required">*</span></span><a-input v-model="form.nickname" aria-label="你的昵称" placeholder="怎么称呼你？" :maxlength="50" :disabled="uploading" /></label>
            <label class="qus-field"><span>联系邮箱 <span class="qus-required">*</span></span><a-input v-model="form.email" aria-label="联系邮箱" placeholder="用于后续认领作品" :maxlength="254" :disabled="uploading" /></label>
          </div>
          <p v-else class="qus-identity"><span class="qus-identity-dot"></span> 以 {{ userStore.user?.username }} 的身份发布</p>
          <label class="qus-field"><span>作品标题 <span class="qus-required">*</span></span><a-input v-model="form.title" aria-label="作品标题" placeholder="给这份灵感起个名字" :maxlength="200" :disabled="uploading" /></label>
          <label class="qus-field"><span class="qus-field-heading"><span>写一点关于它的故事</span><span class="qus-optional">可选</span></span><a-textarea v-model="form.content" aria-label="作品描述" placeholder="创作的起点、想说的话，或只是此刻的心情…" :auto-size="{ minRows: 5, maxRows: 8 }" :disabled="uploading" /></label>
          <p class="qus-form-note">描述和媒体至少填写一项，作品审核后将进入推荐。</p>
        </section>
      </div>

      <footer class="qus-footer">
        <div class="qus-footer-note" aria-live="polite">
          <template v-if="uploading"><span>正在上传 {{ progress }}%</span><a-progress :percent="progress" :show-text="false" :stroke-width="4" /></template>
          <template v-else><span class="qus-footer-spark" aria-hidden="true">✦</span> 每一份创作，都值得被看见。</template>
        </div>
        <a-button type="primary" size="large" html-type="submit" class="qus-submit" :loading="uploading">{{ uploading ? '正在发布' : '发布作品' }}<IconArrowRight v-if="!uploading" /></a-button>
      </footer>
    </form>
  </a-drawer>
</template>

<style scoped>
:global(.qu-sheet .arco-drawer-mask) { background: rgba(47, 32, 43, .3); backdrop-filter: blur(6px); }
:global(.qu-sheet .arco-drawer) {
  width: min(920px, calc(100% - 48px)); left: 0; right: 0; bottom: 24px !important; margin-inline: auto;
  border: 1px solid var(--creative-line); border-radius: 24px; overflow: hidden;
  background: var(--creative-paper); box-shadow: 0 24px 100px rgba(51, 28, 43, .18);
}
:global(.qu-sheet .arco-drawer-body) { padding: 0; overflow: auto; }
.qus-panel { max-height: calc(100dvh - 64px); overflow-y: auto; color: var(--creative-ink); scrollbar-width: thin; scrollbar-color: var(--creative-line) var(--creative-paper); }
.qus-heading { position: relative; display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; padding: 30px 36px 24px; }
.qus-heading::after { content: ''; position: absolute; bottom: 0; left: 36px; width: 48px; height: 2px; background: var(--creative-accent); opacity: .5; }
.qus-kicker { font-size: 10px; letter-spacing: .16em; font-weight: 600; color: var(--creative-muted); }
.qus-kicker span { margin-left: 8px; color: var(--creative-accent); }
.qus-heading h2 { margin: 10px 0 8px; font-size: 25px; font-weight: 600; letter-spacing: -.035em; line-height: 1.4; }
.qus-title-dot { color: var(--creative-accent); }
.qus-heading p { color: var(--creative-muted); font-size: 12px; line-height: 1.7; margin: 0; }
.qus-close, .qus-remove { display: grid; place-items: center; flex-shrink: 0; border: 1px solid var(--creative-line); border-radius: 50%; background: var(--creative-paper); color: var(--creative-muted); cursor: pointer; transition: color .2s, background .2s; }
.qus-close { width: 34px; height: 34px; }
.qus-close:hover:not(:disabled), .qus-remove:hover:not(:disabled) { color: var(--creative-accent); background: var(--creative-soft); }
.qus-body { display: grid; grid-template-columns: .95fr 1.05fr; gap: 32px; padding: 28px 36px 30px; }
.qus-field-heading { display: flex; align-items: center; justify-content: space-between; gap: 8px; font-size: 12px; font-weight: 600; }
.qus-optional { font-size: 10px; font-weight: 400; color: var(--creative-muted); }
.qus-dropzone { width: 100%; min-height: 258px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; margin-top: 10px; padding: 28px 14px; border: 1px dashed color-mix(in srgb, var(--creative-accent) 32%, var(--creative-line)); border-radius: 16px; background: linear-gradient(145deg, var(--creative-soft), var(--creative-paper) 75%); color: var(--creative-ink); cursor: pointer; transition: background .2s, border-color .2s; }
.qus-dropzone:hover:not(:disabled), .qus-dropzone.is-dragover { border-color: var(--creative-accent); background: var(--creative-soft); }
.qus-upload-art { position: relative; display: grid; place-items: center; width: 66px; height: 74px; margin-bottom: 4px; border: 1px solid color-mix(in srgb, var(--creative-accent) 25%, var(--creative-line)); border-radius: 12px 24px 12px 12px; background: var(--creative-paper); color: var(--creative-accent); transform: rotate(-7deg); box-shadow: 7px 7px 0 color-mix(in srgb, var(--creative-accent) 8%, transparent); }
.qus-upload-art > svg { font-size: 25px; transform: rotate(7deg); }
.qus-art-spark { position: absolute; top: -14px; right: -13px; font-size: 24px; }
.qus-dropzone-title { font-size: 14px; font-weight: 600; }
.qus-dropzone-hint, .qus-file-limit { font-size: 11px; color: var(--creative-muted); line-height: 1.7; }
.qus-file-limit { margin-top: 4px; font-size: 10px; }
.qus-hidden { display: none; }
.qus-media-note { display: flex; gap: 7px; margin-top: 13px; font-size: 11px; color: var(--creative-muted); line-height: 1.7; }
.qus-media-note > span { color: var(--creative-accent); }
.qus-fields { display: flex; flex-direction: column; gap: 18px; min-width: 0; }
.qus-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.qus-field { display: flex; flex-direction: column; gap: 9px; min-width: 0; font-size: 12px; font-weight: 600; }
.qus-required { color: var(--creative-accent); font-size: 11px; }
.qus-field :deep(.arco-input-wrapper), .qus-field :deep(.arco-textarea-wrapper) { border: 1px solid var(--creative-line); border-radius: 9px; background: var(--creative-canvas); transition: border-color .2s, box-shadow .2s; }
.qus-field :deep(.arco-input-wrapper) { min-height: 40px; }
.qus-field :deep(.arco-input-focus), .qus-field :deep(.arco-textarea-focus) { border-color: var(--creative-accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 9%, transparent); background: var(--creative-paper); }
.qus-field :deep(input), .qus-field :deep(textarea) { font-size: 12px; font-weight: 400; color: var(--creative-ink); }
.qus-field :deep(textarea) { padding: 10px 12px; line-height: 1.8; }
.qus-form-note { margin: -7px 0 0; font-size: 10px; color: var(--creative-muted); line-height: 1.7; }
.qus-identity { display: flex; align-items: center; gap: 7px; margin: 0; padding: 10px 12px; border-radius: 8px; background: var(--creative-soft); color: var(--creative-accent); font-size: 12px; }
.qus-identity-dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.qus-preview { margin-top: 10px; overflow: hidden; border: 1px solid var(--creative-line); border-radius: 16px; background: var(--creative-canvas); }
.qus-preview-stage { height: 258px; display: grid; place-items: center; padding: 12px; }
.qus-preview-img, .qus-preview-video { width: 100%; height: 100%; max-height: 234px; object-fit: contain; border-radius: 8px; }
.qus-file-row { display: flex; align-items: center; gap: 8px; min-width: 0; padding: 12px; border-top: 1px solid var(--creative-line); background: var(--creative-paper); }
.qus-file-check { flex-shrink: 0; color: var(--creative-accent); }
.qus-file-name { flex: 1; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-size: 11px; }
.qus-file-size { font-size: 10px; color: var(--creative-muted); white-space: nowrap; }
.qus-remove { width: 24px; height: 24px; }
.qus-footer { position: sticky; bottom: 0; display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 18px 36px; border-top: 1px solid var(--creative-line); background: var(--creative-paper); z-index: 1; }
.qus-footer-note { display: flex; align-items: center; gap: 8px; color: var(--creative-muted); font-size: 11px; }
.qus-footer-note:has(.arco-progress) { display: block; flex: 1; max-width: 280px; }
.qus-footer-note :deep(.arco-progress) { margin-top: 5px; }
.qus-footer-spark { color: var(--creative-accent); }
.qus-submit { min-width: 144px; gap: 14px; border-radius: 999px; }
button:disabled { cursor: not-allowed; }
@media (max-width: 768px) {
  :global(.qu-sheet .arco-drawer) { width: 100%; bottom: 0 !important; border-radius: 22px 22px 0 0; border-bottom: 0; }
  .qus-panel { max-height: 92dvh; }
  .qus-heading { padding: 24px 22px 18px; }
  .qus-heading::after { left: 22px; }
  .qus-heading h2 { font-size: 21px; }
  .qus-heading p { max-width: 240px; }
  .qus-body { grid-template-columns: 1fr; gap: 20px; padding: 22px; }
  .qus-dropzone { min-height: 180px; padding: 18px 12px; gap: 8px; }
  .qus-upload-art { width: 48px; height: 52px; margin: 5px 0 7px; }
  .qus-upload-art > svg { font-size: 20px; }
  .qus-footer { gap: 12px; padding: 16px 22px calc(16px + env(safe-area-inset-bottom)); }
  .qus-footer-note { font-size: 10px; max-width: 140px; line-height: 1.7; }
  .qus-submit { min-width: 124px; }
}
@media (max-width: 360px) { .qus-grid { grid-template-columns: 1fr; } }
</style>
