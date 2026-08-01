<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from '@/composables/useToast'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { IconUpload } from '@arco-design/web-vue/es/icon'

interface Props {
  open: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()

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
const filePreview = ref('')
const uploading = ref(false)
const progress = ref(0)

const MAX_FILE_SIZE = 20 * 1024 * 1024
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s]+$/

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
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
  if (!form.value.title) form.value.title = f.name.replace(/\.[^.]+$/, '')
  const reader = new FileReader()
  reader.onload = (ev) => { filePreview.value = ev.target?.result as string }
  reader.readAsDataURL(f)
}

function removeFile() {
  file.value = undefined
  filePreview.value = ''
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
      toast.success('上传成功')
      emit('close')
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
    filePreview.value = ''
  }
})
</script>

<template>
  <a-drawer
    v-model:visible="visible"
    title="快速上传"
    placement="bottom"
    :height="'auto'"
    class="qu-sheet"
    @close="emit('close')"
    :z-index="2000"
  >
    <div class="qus-stack">
      <div v-if="!isLoggedIn" class="qus-grid">
        <a-input v-model="form.nickname" placeholder="昵称" :maxlength="50" />
        <a-input v-model="form.email" placeholder="邮箱" :maxlength="254" />
      </div>

      <a-input v-model="form.title" placeholder="标题" :maxlength="200" />

      <a-textarea v-model="form.content" placeholder="描述（可选）" :auto-size="{ minRows: 3, maxRows: 6 }" />

      <div>
        <div v-if="!file" class="qus-dropzone" @click="($refs.fileInput as HTMLInputElement).click()">
          <input ref="fileInput" type="file" accept="image/*,video/*" class="qus-hidden" @change="onFileChange" />
          <IconUpload class="qus-upload-icon" />
          <div class="qus-dropzone-hint">选择图片/视频</div>
        </div>
        <div v-else class="qus-preview">
          <img v-if="file.type.startsWith('image/')" :src="filePreview" alt="预览" class="qus-preview-img" />
          <video v-else :src="filePreview" controls class="qus-preview-video" />
          <a-button size="small" status="danger" @click="removeFile">移除文件</a-button>
        </div>
      </div>

      <a-progress v-if="uploading" :percent="progress" :stroke-width="10" />

      <a-button type="primary" size="large" long :loading="uploading" @click="handleSubmit">
        {{ uploading ? '上传中...' : '上传' }}
      </a-button>
    </div>
  </a-drawer>
</template>

<style scoped>
.qu-sheet :deep(.arco-drawer-body) {
  padding: 0;
}

.qus-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 480px;
  margin: 0 auto;
  padding: 16px 20px;
  max-height: 80vh;
  overflow-y: auto;
}

.qus-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.qus-dropzone {
  border: 2px dashed var(--color-border);
  border-radius: 12px;
  padding: 24px;
  text-align: center;
  cursor: pointer;
  transition: border-color 0.2s;

  &:hover {
    border-color: var(--color-primary);
  }
}

.qus-hidden {
  display: none;
}

.qus-upload-icon {
  font-size: 28px;
  color: var(--color-text-secondary);
}

.qus-dropzone-hint {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: 8px;
}

.qus-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.qus-preview-img {
  max-height: 180px;
  border-radius: 8px;
  object-fit: contain;
}

.qus-preview-video {
  max-height: 180px;
  border-radius: 8px;
}
</style>
