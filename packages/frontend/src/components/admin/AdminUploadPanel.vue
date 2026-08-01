<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import { CC_LICENSE_TEXT, VIDEO_TERMS_TEXT } from '@/utils/constants'
import { Tag } from '@arco-design/web-vue'
import { IconUpload, IconClose } from '@arco-design/web-vue/es/icon'
import TagCloud from './TagCloud.vue'
import MarkdownToolbar from '@/components/MarkdownToolbar.vue'

const userStore = useUserStore()
const admin = useAdminStore()

const form = ref({
  title: '',
  content: '',
  tags: [] as string[],
  file: undefined as File | undefined,
})
const filePreview = ref('')
const fileKind = ref<'' | 'image' | 'video'>('')
const agreeUpload = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)

function toggleTag(tag: string) {
  const i = form.value.tags.indexOf(tag)
  if (i > -1) form.value.tags.splice(i, 1)
  else form.value.tags.push(tag)
}

function handleAddCustomTag(tag: string) {
  form.value.tags.push(tag)
}

function pickFile(f: File | undefined) {
  if (!f) return
  if (!f.type.startsWith('image/') && !f.type.startsWith('video/')) {
    // 静默拒绝非媒体文件
    return
  }
  form.value.file = f
  fileKind.value = f.type.startsWith('image/') ? 'image' : 'video'
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

function clearFile() {
  form.value.file = undefined
  filePreview.value = ''
  fileKind.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

function clearForm() {
  form.value = { title: '', content: '', tags: [], file: undefined }
  filePreview.value = ''
  fileKind.value = ''
  agreeUpload.value = false
}

function insertMd(prefix: string, suffix: string) {
  const ta = document.querySelector('.upload-textarea textarea') as HTMLTextAreaElement
  if (!ta) return
  const s = ta.selectionStart, e = ta.selectionEnd, t = form.value.content || ''
  form.value.content = t.substring(0, s) + prefix + t.substring(s, e) + suffix + t.substring(e)
  ta.focus()
  setTimeout(
    () => ta.setSelectionRange(s + prefix.length + (e - s) + suffix.length, s + prefix.length + (e - s) + suffix.length),
    0,
  )
}

// 视频阈值保持原 15MB
const VIDEO_SIZE_LIMIT = 15 * 1024 * 1024

function validate(): string | null {
  if (!form.value.title.trim()) return '请填写标题'
  if (!form.value.content.trim() && !form.value.file) return '请填写描述正文或上传媒体文件'
  if (fileKind.value === 'video' && form.value.file && form.value.file.size > VIDEO_SIZE_LIMIT && !agreeUpload.value) {
    return '视频超过 15MB，请勾选同意视频上传条款'
  }
  return null
}

async function handleSubmit() {
  if (!userStore.user) return
  const err = validate()
  if (err) {
    // admin store 内部有 toast，这里通过 console.warn 提示（避免重复 toast）
    console.warn('[admin upload]', err)
    return
  }
  // 类型由后端根据 file 推断；admin.uploadContent 已不再需要 type 字段
  const ok = await admin.uploadContent({
    title: form.value.title.trim(),
    content: form.value.content.trim() || undefined,
    tags: form.value.tags,
    file: form.value.file,
    userId: userStore.user.id,
  })
  if (ok) clearForm()
}
</script>

<template>
  <AdminPanel title="上传内容" desc="发布图片、视频或文字内容，正文支持 Markdown 排版。">
    <div class="upload-wrap">
      <a-form layout="vertical" :model="form">
        <a-form-item label="标题" required>
          <a-input v-model="form.title" placeholder="输入标题" :maxlength="200" />
        </a-form-item>

        <a-form-item label="描述正文（可选，与媒体二选一或都填）">
          <div class="w-full upload-textarea">
            <MarkdownToolbar @insert="insertMd" @upload-image="() => {}" />

            <a-textarea
              v-model="form.content"
              :auto-size="{ minRows: 8, maxRows: 8 }"
              placeholder="支持 Markdown"
            />
          </div>
        </a-form-item>

        <a-form-item label="媒体文件（可选，与正文二选一或都填）">
          <input ref="fileInput" type="file" class="upload-input-hidden" accept="image/*,video/*" @change="onFileChange" />
          <div
            v-if="!form.file"
            class="upload-area"
            :class="{ 'is-dragover': dragOver }"
            @click="fileInput?.click()"
            @dragover.prevent="dragOver = true"
            @dragleave.prevent="dragOver = false"
            @drop.prevent="onDrop"
          >
            <div class="upload-area-icon">
              <IconUpload />
            </div>
            <span class="upload-area-text">点击选择，或拖拽文件到此处</span>
            <span class="upload-area-hint">支持图片 / 视频，视频不超过 15MB</span>
          </div>
          <div v-else class="upload-preview">
            <img v-if="fileKind === 'image'" :src="filePreview" class="upload-preview-media" alt="" />
            <video v-else :src="filePreview" controls class="upload-preview-media" />
            <a-button size="small" class="mt-2" @click="clearFile">
              <IconClose /> 移除文件
            </a-button>
          </div>
        </a-form-item>

        <a-form-item v-if="fileKind === 'video'">
          <a-checkbox v-model="agreeUpload">
            <span class="upload-legal">{{ VIDEO_TERMS_TEXT }}</span>
          </a-checkbox>
        </a-form-item>
        <div v-else class="upload-legal upload-legal-block" v-html="CC_LICENSE_TEXT"></div>

        <a-form-item label="标签">
          <div v-if="form.tags.length" class="mb-2 flex flex-wrap gap-1">
            <Tag
              v-for="tag in form.tags"
              :key="tag"
              color="arcoblue"
              closable
              @close.prevent="toggleTag(tag)"
            >{{ tag }}</Tag>
          </div>
          <TagCloud
            :tags="admin.tags"
            :selected-tags="form.tags"
            :max-tags="50"
            allow-custom
            @toggle="toggleTag"
            @add="handleAddCustomTag"
          />
        </a-form-item>

        <a-form-item v-if="admin.uploading">
          <a-progress :percent="admin.uploadProgress" :stroke-width="18" />
        </a-form-item>

        <div class="upload-submit">
          <a-button @click="clearForm" :disabled="admin.uploading">清空</a-button>
          <a-button
            type="primary"
            :loading="admin.uploading"
            :disabled="admin.uploading"
            @click="handleSubmit"
          >提交发布</a-button>
        </div>
      </a-form>
    </div>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.upload-wrap {
  width: 100%;
  padding: 24px;
}

.upload-input-hidden {
  display: none;
}

.upload-area {
  width: 100%;
  padding: 32px 24px;
  border: 1.5px dashed $admin-border;
  border-radius: $admin-radius;
  background: transparent;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  transition: border-color 0.2s ease, background 0.2s ease;

  &:hover,
  &.is-dragover {
    border-color: $admin-primary;
    background: $admin-primary-soft;
  }
}

.upload-area-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  font-size: 20px;
  margin-bottom: 4px;
  background: $admin-fill;
  color: $admin-text-3;
  transition: background 0.2s ease, color 0.2s ease;

  .upload-area:hover &,
  .upload-area.is-dragover & {
    background: $admin-primary-soft;
    color: $admin-primary;
  }
}

.upload-area-text {
  font-size: 13px;
  color: $admin-text-2;
}

.upload-area-hint {
  font-size: 12px;
  color: $admin-text-3;
}

.upload-preview-media {
  max-height: 220px;
  max-width: 100%;
  border-radius: 10px;
  border: 1px solid $admin-border-soft;
}

.upload-legal {
  font-size: 12px;
  line-height: 1.6;
  color: $admin-text-3;
}

.upload-legal-block {
  padding-top: 12px;
  margin-bottom: 20px;
  border-top: 1px solid $admin-border-soft;
}

.upload-submit {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 4px;
}
</style>
