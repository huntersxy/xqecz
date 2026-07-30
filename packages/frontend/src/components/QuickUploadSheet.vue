<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from '@/composables/useToast'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { CloudUploadOutlined, CloseOutlined } from '@ant-design/icons-vue'

interface Props {
  open: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const userStore = useUserStore()
const isLoggedIn = computed(() => userStore.isLoggedIn)

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
  if (val) loadGuestInfo()
})
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet">
      <div v-if="open" class="upload-sheet-mask" @click.self="emit('close')">
        <div class="upload-sheet">
          <div class="upload-sheet-header">
            <span class="upload-sheet-title">快速上传</span>
            <button class="upload-sheet-close" @click="emit('close')">
              <CloseOutlined />
            </button>
          </div>

          <div class="upload-sheet-body">
            <div v-if="!isLoggedIn" class="upload-row-2">
              <input v-model="form.nickname" placeholder="昵称" :maxlength="50" class="upload-input" />
              <input v-model="form.email" placeholder="邮箱" :maxlength="254" class="upload-input" />
            </div>

            <input v-model="form.title" placeholder="标题" :maxlength="200" class="upload-input" />

            <textarea v-model="form.content" placeholder="描述（可选）" :rows="3" class="upload-textarea" />

            <div class="upload-file-area">
              <label v-if="!file" class="upload-file-label">
                <input type="file" accept="image/*,video/*" class="upload-file-input" @change="onFileChange" />
                <CloudUploadOutlined />
                <span>选择图片/视频</span>
              </label>
              <div v-else class="upload-file-preview">
                <img v-if="file.type.startsWith('image/')" :src="filePreview" alt="预览" />
                <video v-else :src="filePreview" controls />
                <button class="upload-file-remove" @click="removeFile">移除</button>
              </div>
            </div>

            <div v-if="uploading" class="upload-progress">
              <div class="upload-progress-bar" :style="{ width: progress + '%' }"></div>
            </div>

            <button class="upload-submit" :disabled="uploading" @click="handleSubmit">
              {{ uploading ? '上传中...' : '上传' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.upload-sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 2000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.upload-sheet {
  width: 100%;
  max-width: 480px;
  max-height: 85vh;
  background: var(--theme-surface);
  border-radius: 16px 16px 0 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.upload-sheet-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--theme-card-border);
}

.upload-sheet-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--theme-text);
}

.upload-sheet-close {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--theme-text-secondary);
  cursor: pointer;
  border-radius: 50%;
  font-size: 16px;
}

.upload-sheet-close:hover {
  background: var(--theme-hover-bg);
}

.upload-sheet-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.upload-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.upload-input {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid var(--theme-card-border);
  border-radius: 8px;
  background: var(--theme-bg);
  color: var(--theme-text);
  outline: none;
  transition: border-color 0.2s;
}

.upload-input:focus {
  border-color: var(--theme-primary);
}

.upload-textarea {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid var(--theme-card-border);
  border-radius: 8px;
  background: var(--theme-bg);
  color: var(--theme-text);
  outline: none;
  resize: vertical;
  min-height: 60px;
  transition: border-color 0.2s;
}

.upload-textarea:focus {
  border-color: var(--theme-primary);
}

.upload-file-area {
  width: 100%;
}

.upload-file-label {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  border: 2px dashed var(--theme-card-border);
  border-radius: 12px;
  cursor: pointer;
  color: var(--theme-text-secondary);
  font-size: 14px;
  transition: border-color 0.2s, color 0.2s;
}

.upload-file-label:hover {
  border-color: var(--theme-primary);
  color: var(--theme-primary);
}

.upload-file-label :deep(.anticon) {
  font-size: 28px;
}

.upload-file-input {
  display: none;
}

.upload-file-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.upload-file-preview img,
.upload-file-preview video {
  max-height: 180px;
  border-radius: 8px;
  object-fit: contain;
}

.upload-file-remove {
  padding: 4px 12px;
  font-size: 12px;
  color: var(--theme-danger);
  background: transparent;
  border: 1px solid var(--theme-danger);
  border-radius: 6px;
  cursor: pointer;
}

.upload-progress {
  width: 100%;
  height: 6px;
  background: var(--theme-card-border);
  border-radius: 3px;
  overflow: hidden;
}

.upload-progress-bar {
  height: 100%;
  background: var(--theme-primary);
  border-radius: 3px;
  transition: width 0.2s;
}

.upload-submit {
  width: 100%;
  padding: 12px;
  font-size: 15px;
  font-weight: 600;
  color: var(--theme-on-primary);
  background: var(--theme-primary);
  border: none;
  border-radius: 10px;
  cursor: pointer;
  transition: filter 0.2s;
}

.upload-submit:hover {
  filter: brightness(0.92);
}

.upload-submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 入场/退场动画 */
.sheet-enter-active,
.sheet-leave-active {
  transition: opacity 0.25s ease;
}
.sheet-enter-active .upload-sheet,
.sheet-leave-active .upload-sheet {
  transition: transform 0.25s ease;
}
.sheet-enter-from,
.sheet-leave-to {
  opacity: 0;
}
.sheet-enter-from .upload-sheet,
.sheet-leave-to .upload-sheet {
  transform: translateY(100%);
}
</style>
