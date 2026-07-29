<script setup lang="ts">
import { useRouter } from 'vue-router'
import { toast } from '@/composables/useToast'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { CC_LICENSE_TEXT } from '@/utils/constants'
import { CloudUploadOutlined } from '@ant-design/icons-vue'
import MarkdownToolbar from '@/components/MarkdownToolbar.vue'
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

function beforeUpload(f: File) {
  if (!f.type.startsWith('image/') && !f.type.startsWith('video/')) {
    toast.error('仅支持图片或视频文件')
    return false
  }
  if (f.size > MAX_FILE_SIZE) {
    toast.error('文件大小不能超过 20MB')
    return false
  }
  file.value = f
  fileKind.value = f.type.startsWith('image/') ? 'image' : 'video'
  if (!form.value.title) form.value.title = f.name.replace(/\.[^.]+$/, '')
  const reader = new FileReader()
  reader.onload = (ev) => { filePreview.value = ev.target?.result as string }
  reader.readAsDataURL(f)
  // 返回 false 阻止 antd upload 自动上传，我们用自己的 xhrUpload
  return false
}

function removeFile() {
  file.value = undefined
  filePreview.value = ''
  fileKind.value = ''
}

function insertMd(prefix: string, suffix: string) {
  const ta = document.querySelector('.quick-upload-textarea textarea') as HTMLTextAreaElement
  if (!ta) return
  const s = ta.selectionStart, e = ta.selectionEnd, t = form.value.content || ''
  form.value.content = t.substring(0, s) + prefix + t.substring(s, e) + suffix + t.substring(e)
  ta.focus()
  setTimeout(
    () => ta.setSelectionRange(s + prefix.length + (e - s) + suffix.length, s + prefix.length + (e - s) + suffix.length),
    0,
  )
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
        nickname: isLoggedIn.value ? '' : form.value.nickname.trim(),
        email: isLoggedIn.value ? '' : form.value.email.trim(),
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
    <div class="theme-card rounded-xl p-6 shadow-sm theme-border">
      <a-form layout="vertical">
        <div v-if="!isLoggedIn" class="grid grid-cols-1 sm:grid-cols-2 gap-x-4">
          <a-form-item label="昵称" required>
            <a-input
              v-model:value="form.nickname"
              placeholder="用于展示的昵称"
              :maxlength="50"
              :disabled="uploading"
            />
          </a-form-item>
          <a-form-item label="邮箱" required>
            <a-input
              v-model:value="form.email"
              placeholder="仅用于标识身份，不会公开"
              :maxlength="254"
              :disabled="uploading"
            />
          </a-form-item>
        </div>

        <a-form-item label="标题" required>
          <a-input
            v-model:value="form.title"
            placeholder="给作品起个标题"
            :maxlength="200"
            :disabled="uploading"
          />
        </a-form-item>

        <a-form-item label="描述正文（可选）">
          <div class="w-full quick-upload-textarea">
            <MarkdownToolbar @insert="insertMd" @upload-image="() => {}" />
            <a-textarea
              v-model:value="form.content"
              :rows="6"
              placeholder="支持 Markdown，描述、提示词、灵感…"
              :disabled="uploading"
            />
          </div>
        </a-form-item>

        <a-form-item label="媒体文件（可选）">
          <a-upload-dragger
            v-if="!file"
            :before-upload="beforeUpload"
            accept="image/*,video/*"
            :show-upload-list="false"
            :disabled="uploading"
          >
            <p class="ant-upload-drag-icon">
              <CloudUploadOutlined />
            </p>
            <p class="ant-upload-text">点击或拖拽文件到此区域</p>
            <p class="ant-upload-hint">图片或视频，最大 20MB</p>
          </a-upload-dragger>
          <div v-else>
            <img
              v-if="fileKind === 'image'"
              :src="filePreview"
              style="max-height: 240px; border-radius: 8px; display: block"
              alt="预览"
            />
            <video
              v-else
              :src="filePreview"
              controls
              style="max-height: 240px; max-width: 100%; border-radius: 8px"
            />
            <a-button
              size="small"
              danger
              class="mt-2"
              :disabled="uploading"
              @click="removeFile"
            >移除文件</a-button>
          </div>
        </a-form-item>

        <a-form-item label="标签（可选）">
          <div class="theme-card rounded-lg p-3 theme-border">
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

        <div
          class="border-t pt-2 mb-4 text-xs theme-text-secondary"
          style="border-color: var(--theme-card-border)"
          v-html="CC_LICENSE_TEXT"
        ></div>

        <a-form-item v-if="uploading">
          <a-progress :percent="progress" status="active" :stroke-width="18" />
        </a-form-item>

        <a-button
          type="primary"
          block
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
