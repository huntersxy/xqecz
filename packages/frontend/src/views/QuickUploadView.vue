<script setup lang="ts">
import { useRouter } from 'vue-router'
import { toast } from '@/composables/useToast'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { CC_LICENSE_TEXT } from '@/utils/constants'
import { IconUpload } from '@arco-design/web-vue/es/icon'
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
const filePreview = ref('')
const fileKind = ref<'' | 'image' | 'video'>('')
const availableTags = ref<string[]>([])
const uploading = ref(false)
const progress = ref(0)
const fileInput = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)

const MAX_FILE_SIZE = 20 * 1024 * 1024
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function toggleTag(tag: string) {
  const i = form.value.tags.indexOf(tag)
  if (i > -1) form.value.tags.splice(i, 1)
  else form.value.tags.push(tag)
}
function handleAddCustomTag(tag: string) {
  if (!form.value.tags.includes(tag)) form.value.tags.push(tag)
}

function pickFile(f: File | undefined) {
  if (!f) return
  if (!f.type.startsWith('image/') && !f.type.startsWith('video/')) {
    toast.error('仅支持图片或视频文件')
    return
  }
  if (f.size > MAX_FILE_SIZE) {
    toast.error('文件大小不能超过 20MB')
    return
  }
  file.value = f
  fileKind.value = f.type.startsWith('image/') ? 'image' : 'video'
  if (!form.value.title) form.value.title = f.name.replace(/\.[^.]+$/, '')
  const reader = new FileReader()
  reader.onload = (ev) => { filePreview.value = ev.target?.result as string }
  reader.readAsDataURL(f)
}

function onFileChange(e: Event) {
  pickFile((e.target as HTMLInputElement).files?.[0])
}

function onDrop(e: DragEvent) {
  dragOver.value = false
  pickFile(e.dataTransfer?.files?.[0])
}

function removeFile() {
  file.value = undefined
  filePreview.value = ''
  fileKind.value = ''
  if (fileInput.value) fileInput.value.value = ''
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
      toast.success('上传成功，可在首页查看')
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
  <div class="max-w-[640px] mx-auto px-4 py-8">
    <div class="bg-[var(--color-card)] rounded-xl p-6 shadow-sm border-[var(--color-border)]">
      <a-form layout="vertical" :model="form">
        <div v-if="!isLoggedIn" class="grid grid-cols-1 sm:grid-cols-2 gap-x-4">
          <a-form-item label="昵称" required>
            <a-input
              v-model="form.nickname"
              placeholder="用于展示的昵称"
              :maxlength="50"
              :disabled="uploading"
            />
          </a-form-item>
          <a-form-item label="邮箱" required>
            <a-input
              v-model="form.email"
              placeholder="仅用于标识身份，不会公开"
              :maxlength="254"
              :disabled="uploading"
            />
          </a-form-item>
        </div>

        <a-form-item label="标题" required>
          <a-input
            v-model="form.title"
            placeholder="给作品起个标题"
            :maxlength="200"
            :disabled="uploading"
          />
        </a-form-item>

        <a-form-item label="描述正文（可选）">
          <MarkdownEditor
            v-model="form.content"
            placeholder="支持 Markdown，描述、提示词、灵感…"
            :height="320"
            :disabled="uploading"
          />
        </a-form-item>

        <a-form-item label="媒体文件（可选）">
          <input
            ref="fileInput"
            type="file"
            class="qu-upload-input-hidden"
            accept="image/*,video/*"
            :disabled="uploading"
            @change="onFileChange"
          />
          <div
            v-if="!file"
            class="qu-upload-area"
            :class="{ 'is-dragover': dragOver }"
            @click="fileInput?.click()"
            @dragover.prevent="dragOver = true"
            @dragleave.prevent="dragOver = false"
            @drop.prevent="onDrop"
          >
            <IconUpload class="qu-upload-icon" />
            <span class="qu-upload-hint">点击选择，或拖拽文件到此区域</span>
            <span class="qu-upload-sub">图片或视频，最大 20MB</span>
          </div>
          <div v-else class="qu-upload-preview">
            <a-image
              v-if="fileKind === 'image'"
              :src="filePreview"
              :preview="false"
              class="qu-preview-img"
              alt="预览"
            />
            <video
              v-else
              :src="filePreview"
              controls
              class="qu-preview-video"
            />
            <a-button
              size="small"
              status="danger"
              class="mt-2"
              :disabled="uploading"
              @click="removeFile"
            >移除文件</a-button>
          </div>
        </a-form-item>

        <a-form-item label="标签（可选）">
          <div class="bg-[var(--color-card)] rounded-lg p-3 border-[var(--color-border)]">
            <TagCloud
              :tags="availableTags"
              :selected-tags="form.tags"
              :max-tags="50"
              allow-custom
              @toggle="toggleTag"
              @add="handleAddCustomTag"
            />
          </div>
        </a-form-item>

        <div class="qu-license" v-html="CC_LICENSE_TEXT"></div>

        <a-form-item v-if="uploading">
          <a-progress :percent="progress" status="normal" :stroke-width="18" />
        </a-form-item>

        <a-button
          type="primary"
          long
          size="large"
          :loading="uploading"
          @click="handleSubmit"
        >
          {{ uploading ? '上传中...' : '上传' }}
        </a-button>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
.qu-preview-img :deep(.arco-image-img) {
  max-height: 240px;
  border-radius: 8px;
  display: block;
  object-fit: contain;
}

.qu-upload-icon {
  font-size: 40px;
  color: var(--color-primary);
}

.qu-upload-input-hidden {
  display: none;
}

.qu-upload-area {
  width: 100%;
  padding: 32px 24px;
  border: 1.5px dashed var(--color-border-2);
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.qu-upload-area:hover,
.qu-upload-area.is-dragover {
  border-color: rgb(var(--primary-6));
  background: var(--color-fill-1);
}

.qu-upload-hint {
  display: block;
  margin: 6px 0;
}

.qu-upload-sub {
  display: block;
  margin: 0;
  font-size: 12px;
}

.qu-upload-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.qu-preview-video {
  max-height: 240px;
  max-width: 100%;
  border-radius: 8px;
}

.qu-license {
  padding-top: 8px;
  margin-bottom: 16px;
  border-top: 1px solid var(--color-border);
  font-size: 12px;
  color: var(--color-text-secondary);
}
</style>
