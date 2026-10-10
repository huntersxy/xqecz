<script setup lang="ts">
import { useRouter } from 'vue-router'
import { toast } from '@/composables/useToast'
import { useFilePicker } from '@/composables/useFilePicker'
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
const fileKind = ref<'' | 'image' | 'video'>('')
const availableTags = ref<string[]>([])
const uploading = ref(false)
const progress = ref(0)
const { filePreview, fileInput, dragOver, onFileChange, onDrop, clearPreview } = useFilePicker((f) => {
  file.value = f
  fileKind.value = f.type.startsWith('image/') ? 'image' : 'video'
  if (!form.value.title) form.value.title = f.name.replace(/\.[^.]+$/, '')
})

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
  fileKind.value = ''
  clearPreview()
}

function validate(): string | null {
  if (!isLoggedIn.value) {
    if (!form.value.nickname.trim()) return '请填写昵称'
    if (!EMAIL_RE.test(form.value.email.trim())) return '请填写正确的邮箱地址'
  }
  if (!form.value.title.trim()) return '请填写标题'
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
        title: form.value.title.trim(),
        nickname: isLoggedIn.value ? userStore.user?.username || '' : form.value.nickname.trim(),
        email: isLoggedIn.value ? userStore.user?.email || '' : form.value.email.trim(),
        content: form.value.content.trim() || undefined,
        tags: form.value.tags,
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
        <div><span class="qu-kicker">CREATE & SHARE <span aria-hidden="true">✦</span></span><h1>让灵感，在这里落笔<span>。</span></h1><p>一段文字，一张画面，都可以是故事的开始。</p></div>
        <RouterLink to="/" class="qu-back">← 返回首页</RouterLink>
      </header>
      <a-form class="qu-card" layout="vertical" :model="form" @submit="handleSubmit">
        <div class="qu-body">
          <section class="qu-media" aria-labelledby="qu-media-title">
            <div class="qu-section-heading"><span class="qu-step">01</span><h2 id="qu-media-title">作品画面</h2><span class="qu-optional">可选</span></div>
            <input ref="fileInput" type="file" class="qu-hidden" accept="image/*,video/*" :disabled="uploading" @change="onFileChange" />
            <Transition name="qu-media-state" mode="out-in">
            <button v-if="!file" key="dropzone" type="button" class="qu-dropzone" :class="{ 'is-dragover': dragOver }" :disabled="uploading" @click="fileInput?.click()" @dragover.prevent="!uploading && (dragOver = true)" @dragleave.prevent="dragOver = false" @drop.prevent="!uploading && onDrop($event)">
              <span class="qu-upload-art" aria-hidden="true"><IconUpload /><span>✦</span></span>
              <strong>把你的灵感放进来</strong><span>点击选择，或拖拽图片 / 视频到这里</span><span class="qu-file-limit">图片或视频 · 最大 20MB</span>
            </button>
            <div v-else key="preview" class="qu-preview">
              <div class="qu-preview-stage"><a-image v-if="fileKind === 'image'" :src="filePreview" :preview="false" class="qu-preview-img" alt="所选作品预览" /><video v-else :src="filePreview" controls class="qu-preview-video" /></div>
              <div class="qu-file-row"><span class="qu-file-name">{{ file.name }}</span><button type="button" class="qu-remove" aria-label="移除已选文件" :disabled="uploading" @click="removeFile"><IconClose /></button></div>
            </div>
            </Transition>
            <p class="qu-media-note"><span aria-hidden="true">✧</span> 没有图片也没关系，一段文字同样可以成为作品。</p>
            <div class="qu-tip-card"><span class="qu-tip-kicker">A SMALL REMINDER</span><p>创作不必完美，<br>热爱自有回响<span>。</span></p><span class="qu-tip-spark" aria-hidden="true">✧</span></div>
          </section>

          <section class="qu-fields" aria-labelledby="qu-story-title">
            <div class="qu-section-heading"><span class="qu-step">02</span><h2 id="qu-story-title">作品的故事</h2></div>
            <div v-if="!isLoggedIn" class="qu-guest-grid"><a-form-item label="昵称" required><a-input v-model="form.nickname" placeholder="怎么称呼你？" :maxlength="50" :disabled="uploading" /></a-form-item><a-form-item label="邮箱" required><a-input v-model="form.email" placeholder="仅用于标识身份，不会公开" :maxlength="254" :disabled="uploading" /></a-form-item></div>
            <p v-else class="qu-identity"><span aria-hidden="true">✦</span> 以 {{ userStore.user?.username }} 的身份发布</p>
            <a-form-item label="作品标题" required><a-input v-model="form.title" placeholder="给这份灵感起个名字" :maxlength="200" :disabled="uploading" /></a-form-item>
            <a-form-item label="描述正文（可选）"><MarkdownEditor v-model="form.content" placeholder="描述、提示词、灵感… 支持 Markdown 排版" :height="260" :disabled="uploading" /></a-form-item>
            <p class="qu-form-note">描述和媒体至少填写一项，作品审核后将进入推荐。</p>
            <a-form-item label="标签（可选）"><div class="qu-tags"><TagCloud :tags="availableTags" :selected-tags="form.tags" :max-tags="16" allow-custom @toggle="toggleTag" @add="handleAddCustomTag" /></div></a-form-item>
          </section>
        </div>
        <footer class="qu-footer">
          <div class="qu-footer-copy"><div class="qu-license" v-html="CC_LICENSE_TEXT"></div><div v-if="uploading" class="qu-progress" aria-live="polite"><span>正在上传 {{ progress }}%</span><a-progress :percent="progress" :show-text="false" :stroke-width="4" /></div><p v-else><span aria-hidden="true">✦</span> 每一份创作，都值得被看见。</p></div>
          <a-button type="primary" size="large" html-type="submit" class="qu-submit" :loading="uploading">{{ uploading ? '正在发布' : '发布作品' }}<IconArrowRight v-if="!uploading" /></a-button>
        </footer>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
.qu-page { background: var(--creative-canvas); padding: 44px 28px 64px; }
.qu-layout { max-width: 1080px; margin: 0 auto; }
.qu-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin-bottom: 28px; }
.qu-kicker { font-size: 9px; letter-spacing: .16em; color: var(--creative-muted); }
.qu-kicker > span { margin-left: 8px; color: var(--creative-accent); }
.qu-heading h1 { margin: 14px 0 10px; font-size: 30px; font-weight: 650; letter-spacing: .02em; color: var(--creative-ink); line-height: 1.5; }
.qu-heading h1 > span { color: var(--creative-accent); }
.qu-heading p { color: var(--creative-muted); font-size: 12px; line-height: 1.8; }
.qu-back { padding-bottom: 3px; text-decoration: none; color: var(--creative-muted); font-size: 11px; white-space: nowrap; }
.qu-back:hover { color: var(--creative-accent); }
.qu-card { border: 1px solid var(--creative-line); border-radius: 22px; background: var(--creative-paper); box-shadow: var(--creative-shadow); overflow: hidden; }
.qu-body { display: grid; grid-template-columns: 300px minmax(0, 1fr); gap: 42px; padding: 32px; }
.qu-section-heading { display: flex; align-items: center; gap: 10px; margin-bottom: 22px; }
.qu-step { display: grid; place-items: center; width: 26px; height: 26px; border: 1px solid var(--creative-line); border-radius: 50%; background: var(--creative-canvas); color: var(--creative-accent); font-size: 9px; }
.qu-section-heading h2 { font-size: 13px; font-weight: 600; color: var(--creative-ink); margin: 0; }
.qu-optional { margin-left: auto; font-size: 10px; color: var(--creative-muted); }
.qu-hidden { display: none; }
.qu-dropzone { width: 100%; min-height: 286px; padding: 30px 16px; border: 1px dashed color-mix(in srgb, var(--creative-accent) 28%, var(--creative-line)); border-radius: 16px; background: var(--creative-canvas); color: var(--creative-ink); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; cursor: pointer; transition: border-color .2s, background .2s; }
.qu-dropzone:hover:not(:disabled), .qu-dropzone.is-dragover { background: var(--creative-soft); border-color: var(--creative-accent); }
.qu-upload-art { position: relative; display: grid; place-items: center; width: 62px; height: 72px; margin: 4px 0 15px; border: 1px solid color-mix(in srgb, var(--creative-accent) 25%, var(--creative-line)); border-radius: 12px 24px 12px 12px; background: var(--creative-paper); color: var(--creative-accent); transform: rotate(-7deg); box-shadow: 7px 7px 0 color-mix(in srgb, var(--creative-accent) 8%, transparent); }
.qu-upload-art > svg { font-size: 25px; transform: rotate(7deg); }
.qu-upload-art > span { position: absolute; top: -17px; right: -16px; font-size: 27px; }
.qu-dropzone strong { font-size: 13px; font-weight: 600; }
.qu-dropzone > span:not(.qu-upload-art) { font-size: 10px; color: var(--creative-muted); line-height: 1.8; }
.qu-file-limit { margin-top: 3px; }
.qu-media-note { display: flex; gap: 7px; margin: 14px 0 0; font-size: 11px; line-height: 1.9; color: var(--creative-muted); }
.qu-media-note > span { color: var(--creative-accent); }
.qu-tip-card { position: relative; margin-top: 30px; border-top: 1px solid var(--creative-line); padding-top: 24px; }
.qu-tip-kicker { font-size: 8px; letter-spacing: .13em; color: var(--creative-muted); }
.qu-tip-card p { margin: 12px 0 0; color: var(--creative-ink); font-size: 18px; line-height: 1.8; }
.qu-tip-card p > span { color: var(--creative-accent); }
.qu-tip-spark { position: absolute; right: 10px; bottom: 10px; font-size: 35px; color: var(--creative-accent); }
.qu-fields { min-width: 0; }
.qu-guest-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.qu-fields :deep(.arco-form-item) { margin-bottom: 22px; }
.qu-fields :deep(.arco-form-item-label-col) { margin-bottom: 9px; }
.qu-fields :deep(.arco-form-item-label) { color: var(--creative-ink); font-size: 12px; font-weight: 600; }
.qu-fields :deep(.arco-input-wrapper) { min-height: 42px; border: 1px solid var(--creative-line); border-radius: 10px; background: var(--creative-canvas); }
.qu-fields :deep(.arco-input-focus) { border-color: var(--creative-accent); background: var(--creative-paper); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 9%, transparent); }
.qu-fields :deep(.arco-input) { font-size: 12px; color: var(--creative-ink); }
.qu-fields :deep(.vditor) { border-color: var(--creative-line); border-radius: 10px; background: var(--creative-paper); --toolbar-background-color: var(--creative-canvas); --textarea-background-color: var(--creative-paper); --border-color: var(--creative-line); --second-color: var(--creative-muted); }
.qu-fields :deep(.vditor-toolbar) { border-bottom-color: var(--creative-line); }
.qu-fields :deep(.vditor-toolbar button:hover) { color: var(--creative-accent); }
.qu-identity { display: flex; gap: 7px; margin: 0 0 22px; padding: 11px 13px; border: 1px solid var(--creative-line); border-radius: 10px; background: var(--creative-soft); color: var(--creative-accent); font-size: 12px; }
.qu-form-note { margin: -10px 0 24px; color: var(--creative-muted); font-size: 10px; line-height: 1.8; }
.qu-tags { width: 100%; padding: 12px; border: 1px solid var(--creative-line); border-radius: 10px; background: var(--creative-canvas); }
.qu-tags :deep(.tag-item) { background: var(--creative-paper); border-color: var(--creative-line); color: var(--creative-muted); font-size: 11px; padding: 5px 10px; }
.qu-tags :deep(.tag-item:hover) { background: var(--creative-soft); border-color: var(--creative-accent); color: var(--creative-accent); }
.qu-tags :deep(.tag-item.is-selected) { background: var(--creative-soft); border-color: var(--creative-accent); color: var(--creative-accent); }
.qu-preview { border: 1px solid var(--creative-line); border-radius: 16px; overflow: hidden; background: var(--creative-canvas); }
.qu-preview-stage { display: grid; place-items: center; height: 286px; padding: 14px; }
.qu-preview-img { width: 100%; }
.qu-preview-img :deep(.arco-image-img), .qu-preview-video { display: block; width: 100%; max-height: 258px; object-fit: contain; border-radius: 8px; }
.qu-file-row { display: flex; align-items: center; gap: 8px; padding: 12px; border-top: 1px solid var(--creative-line); background: var(--creative-paper); }
.qu-file-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--creative-ink); font-size: 11px; }
.qu-remove { display: grid; place-items: center; width: 26px; height: 26px; padding: 0; border: 0; border-radius: 50%; background: var(--creative-canvas); color: var(--creative-muted); cursor: pointer; }
.qu-remove:hover { background: var(--creative-soft); color: var(--creative-accent); }
.qu-footer { display: flex; align-items: center; justify-content: space-between; gap: 24px; border-top: 1px solid var(--creative-line); padding: 22px 32px; background: var(--creative-paper); }
.qu-footer-copy { min-width: 0; }
.qu-license { color: var(--creative-muted); font-size: 10px; line-height: 1.8; }
.qu-license :deep(a) { color: var(--creative-accent); }
.qu-footer-copy > p { margin: 8px 0 0; color: var(--creative-muted); font-size: 11px; }
.qu-footer-copy > p > span { margin-right: 6px; color: var(--creative-accent); }
.qu-submit { flex-shrink: 0; min-width: 144px; border-radius: 999px; gap: 14px; }
.qu-progress { max-width: 250px; margin-top: 9px; font-size: 11px; color: var(--creative-muted); }
button:disabled { cursor: not-allowed; }
@media (max-width: 900px) { .qu-body { grid-template-columns: 260px minmax(0, 1fr); gap: 28px; padding: 26px; } .qu-guest-grid { grid-template-columns: 1fr; gap: 0; } }
@media (max-width: 768px) { .qu-page { padding: 30px 20px 44px; } .qu-heading { align-items: flex-start; gap: 14px; } .qu-heading h1 { font-size: 24px; } .qu-heading p { font-size: 11px; max-width: 250px; } .qu-back { font-size: 10px; margin-top: 4px; } .qu-body { grid-template-columns: minmax(0, 1fr); gap: 26px; padding: 24px; } .qu-section-heading { margin-bottom: 18px; } .qu-tip-card { display: none; } .qu-dropzone { min-height: 210px; } .qu-guest-grid { grid-template-columns: 1fr 1fr; gap: 12px; } .qu-footer { padding: 20px 24px; gap: 16px; } .qu-submit { min-width: 120px; } }
@media (max-width: 480px) { .qu-heading { flex-direction: column; gap: 12px; } .qu-heading h1 { font-size: 25px; } .qu-heading p { max-width: none; } .qu-card { border-radius: 18px; } .qu-body { padding: 20px; } .qu-guest-grid { grid-template-columns: 1fr; gap: 0; } .qu-footer { flex-direction: column; align-items: stretch; padding: 20px; } .qu-submit { width: 100%; } }
.qu-media-state-enter-active, .qu-media-state-leave-active { transition: opacity .16s ease, transform .16s ease; }
.qu-media-state-enter-from { opacity: 0; transform: translateY(4px); }
.qu-media-state-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
