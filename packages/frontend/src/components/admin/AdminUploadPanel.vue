<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import { CC_LICENSE_TEXT, VIDEO_TERMS_TEXT } from '@/utils/constants'
import { SECONDARY_STYLE } from './adminColumns'
import { UploadOutlined } from '@ant-design/icons-vue'
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

function toggleTag(tag: string) {
  const i = form.value.tags.indexOf(tag)
  if (i > -1) form.value.tags.splice(i, 1)
  else form.value.tags.push(tag)
}

function handleAddCustomTag(tag: string) {
  form.value.tags.push(tag)
}

function onFileChange(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
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
  <a-card :bordered="false" title="上传内容">
    <a-form layout="vertical">
      <a-form-item label="标题" required>
        <a-input v-model:value="form.title" placeholder="输入标题" :maxlength="200" />
      </a-form-item>

      <a-form-item label="描述正文（可选，与媒体二选一或都填）">
        <div class="w-full upload-textarea">
          <MarkdownToolbar @insert="insertMd" @upload-image="() => {}" />
          <a-textarea v-model:value="form.content" :rows="8" placeholder="支持Markdown" />
        </div>
      </a-form-item>

      <a-form-item label="媒体文件（可选，与正文二选一或都填）">
        <input ref="fileInput" type="file" style="display:none" accept="image/*,video/*" @change="onFileChange" />
        <div v-if="!form.file" class="upload-area" @click="fileInput?.click()">
          <UploadOutlined class="text-3xl" style="color: var(--admin-text-tertiary)" />
          <span class="text-xs" :style="{ color: SECONDARY_STYLE.split(': ')[1] }">
            点击选择图片或视频
          </span>
        </div>
        <div v-else class="mt-1">
          <img
            v-if="fileKind === 'image'"
            :src="filePreview"
            style="max-height: 200px; border-radius: 8px"
            alt=""
          />
          <video
            v-else
            :src="filePreview"
            controls
            style="max-height: 200px; max-width: 100%; border-radius: 8px"
          />
          <div>
            <a-button size="small" class="mt-2" @click="clearFile">移除文件</a-button>
          </div>
        </div>
      </a-form-item>

      <a-form-item v-if="fileKind === 'video'">
        <a-checkbox v-model:checked="agreeUpload">
          <span class="text-xs leading-relaxed" :style="{ color: SECONDARY_STYLE.split(': ')[1] }">{{ VIDEO_TERMS_TEXT }}</span>
        </a-checkbox>
      </a-form-item>
      <div
        v-else
        class="border-t pt-1 pb-0 mb-4 text-xs"
        :style="{ borderColor: 'var(--theme-card-border)', color: SECONDARY_STYLE.split(': ')[1] }"
        v-html="CC_LICENSE_TEXT"
      ></div>

      <a-form-item label="标签">
        <div v-if="form.tags.length" class="mb-2 flex flex-wrap gap-1">
          <Tag
            v-for="tag in form.tags"
            :key="tag"
            color="blue"
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
        <a-progress :percent="admin.uploadProgress" status="active" :stroke-width="18" />
      </a-form-item>

      <div class="flex justify-end gap-3">
        <a-button @click="clearForm" :disabled="admin.uploading">清空</a-button>
        <a-button
          type="primary"
          :loading="admin.uploading"
          :disabled="admin.uploading"
          @click="handleSubmit"
        >提交</a-button>
      </div>
    </a-form>
  </a-card>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.upload-area {
  width: 100%;
  padding: 24px;
  border: 2px dashed var(--theme-card-border);
  border-radius: 8px;
  background: var(--theme-hover-bg);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  transition: border-color 0.2s;

  &:hover {
    border-color: $admin-primary;
  }
}
</style>
