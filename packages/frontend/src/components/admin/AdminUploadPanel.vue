<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import { CC_LICENSE_TEXT, VIDEO_TERMS_TEXT } from '@/utils/constants'
import { Tag } from '@arco-design/web-vue'
import { IconUpload, IconClose } from '@arco-design/web-vue/es/icon'
import { useFilePicker } from '@/composables/useFilePicker'
import { useAutoMetadata } from '@/composables/useAutoMetadata'
import AutoMetadataOption from '@/components/AutoMetadataOption.vue'
import { toast } from '@/composables/useToast'
import TagCloud from './TagCloud.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'

const userStore = useUserStore()
const admin = useAdminStore()

const form = ref({
  title: '',
  content: '',
  tags: [] as string[],
  file: undefined as File | undefined,
})
const fileKind = ref<'' | 'image' | 'video'>('')
const { autoMetadata, isAiGenerated, metadataOptions, canGenerate, metadataTitle, metadataTags } = useAutoMetadata(() => form.value.file)
const agreeUpload = ref(false)
const { filePreview, fileInput, dragOver, onFileChange, onDrop, clearPreview } = useFilePicker((f) => {
  form.value.file = f
  fileKind.value = f.type.startsWith('image/') ? 'image' : 'video'
})

function toggleTag(tag: string) {
  const i = form.value.tags.indexOf(tag)
  if (i > -1) form.value.tags.splice(i, 1)
  else form.value.tags.push(tag)
}

function handleAddCustomTag(tag: string) {
  form.value.tags.push(tag)
}

function clearFile() {
  form.value.file = undefined
  fileKind.value = ''
  clearPreview()
}

function clearForm() {
  form.value = { title: '', content: '', tags: [], file: undefined }
  filePreview.value = ''
  fileKind.value = ''
  agreeUpload.value = false
  autoMetadata.value = false
  isAiGenerated.value = false
}

// 视频阈值保持原 15MB
const VIDEO_SIZE_LIMIT = 15 * 1024 * 1024

function validate(): string | null {
  if (!autoMetadata.value && !form.value.title.trim()) return '请填写标题'
  if (!form.value.content.trim() && !form.value.file) return '请填写描述正文或上传媒体文件'
  if (fileKind.value === 'video' && form.value.file && form.value.file.size > VIDEO_SIZE_LIMIT && !agreeUpload.value) {
    return '视频超过 15MB，请勾选同意视频上传条款'
  }
  return null
}

async function handleSubmit() {
  if (admin.uploading) return
  if (!userStore.user) return
  const err = validate()
  if (err) {
    toast.warning(err)
    return
  }
  // 类型由后端根据 file 推断；admin.uploadContent 已不再需要 type 字段
  const ok = await admin.uploadContent({
    title: metadataTitle(form.value.title.trim()),
    auto_metadata: autoMetadata.value,
    is_ai_generated: isAiGenerated.value,
    content: form.value.content.trim() || undefined,
    tags: metadataTags(form.value.tags),
    file: form.value.file,
    userId: userStore.user.id,
  })
  if (ok) clearForm()
}
</script>

<template>
  <AdminPanel flat title="上传内容" desc="发布图片、视频或文字内容，正文支持 Markdown 排版。">
    <div class="upload-wrap">
      <a-form layout="vertical" :model="form" class="upload-form">
        <!-- 01 基本信息 -->
        <section class="upload-section">
          <header class="upload-section-head">
            <span class="upload-section-num" aria-hidden="true">01</span>
            <div class="upload-section-lead">
              <h3 class="upload-section-title">基本信息</h3>
              <p class="upload-section-desc">标题与标签决定内容在站内的归类；图片可交给视觉模型自动生成。</p>
            </div>
          </header>
          <div class="upload-section-body">
            <a-form-item><AutoMetadataOption v-model="autoMetadata" :options="metadataOptions" :can-generate="canGenerate" :disabled="admin.uploading" /></a-form-item>
            <a-form-item><a-checkbox v-model="isAiGenerated" :disabled="admin.uploading">这是 AI 生成的作品（添加 AIGC 标签）</a-checkbox></a-form-item>
            <a-form-item label="标题" required>
              <a-input :model-value="metadataTitle(form.title)" @update:model-value="form.title = $event" placeholder="输入标题" :maxlength="200" :disabled="admin.uploading || autoMetadata" />
            </a-form-item>
            <a-form-item label="标签">
              <a-input-tag v-if="autoMetadata" :model-value="metadataTags(form.tags)" disabled />
              <template v-else>
                <div v-if="form.tags.length" class="upload-tag-list">
                  <Tag
                    v-for="tag in form.tags"
                    :key="tag"
                    class="admin-tag-accent"
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
                  :disabled="admin.uploading"
                />
              </template>
            </a-form-item>
          </div>
        </section>

        <!-- 02 媒体 -->
        <section class="upload-section">
          <header class="upload-section-head">
            <span class="upload-section-num" aria-hidden="true">02</span>
            <div class="upload-section-lead">
              <h3 class="upload-section-title">媒体</h3>
              <p class="upload-section-desc">图片或视频可选填；本地上传后由后台生成缩略图。</p>
            </div>
          </header>
          <div class="upload-section-body">
            <a-form-item label="媒体文件（可选，与正文二选一或都填）">
              <input ref="fileInput" type="file" class="upload-input-hidden" accept="image/*,video/*" @change="onFileChange" />
              <button
                v-if="!form.file"
                type="button"
                class="upload-area"
                :class="{ 'is-dragover': dragOver }"
                @click="fileInput?.click()"
                @dragover.prevent="dragOver = true"
                @dragleave.prevent="dragOver = false"
                @drop.prevent="onDrop"
              >
                <span class="upload-area-icon" aria-hidden="true">
                  <IconUpload />
                </span>
                <span class="upload-area-text">点击选择，或拖拽文件到此处</span>
                <span class="upload-area-hint">图片 / 视频，最大 20MB；视频超 15MB 需勾选条款</span>
              </button>
              <div v-else class="upload-preview">
                <img v-if="fileKind === 'image'" :src="filePreview" class="upload-preview-media" alt="" />
                <video v-else :src="filePreview" controls class="upload-preview-media" />
                <a-button size="small" class="upload-preview-remove" @click="clearFile">
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
          </div>
        </section>

        <!-- 03 正文 Markdown -->
        <section class="upload-section">
          <header class="upload-section-head">
            <span class="upload-section-num" aria-hidden="true">03</span>
            <div class="upload-section-lead">
              <h3 class="upload-section-title">正文 Markdown</h3>
              <p class="upload-section-desc">支持 Markdown 排版；纯媒体投稿可留空，与媒体二选一或都填。</p>
            </div>
          </header>
          <div class="upload-section-body">
            <a-form-item label="描述正文（可选，与媒体二选一或都填）">
              <MarkdownEditor
                v-model="form.content"
                placeholder="支持 Markdown，描述、提示词、灵感…"
                :height="360"
              />
            </a-form-item>
          </div>
        </section>
      </a-form>

      <!-- 底部吸附操作条 -->
      <div class="upload-actions">
        <a-progress
          v-if="admin.uploading"
          :percent="admin.uploadProgress"
          :stroke-width="10"
          class="upload-actions-progress"
        />
        <div class="upload-actions-buttons">
          <a-button @click="clearForm" :disabled="admin.uploading">清空</a-button>
          <a-button
            type="primary"
            :loading="admin.uploading"
            :disabled="admin.uploading"
            @click="handleSubmit"
          >提交发布</a-button>
        </div>
      </div>
    </div>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.upload-wrap {
  width: 100%;
}

.upload-section {
  padding: var(--adm-gap-xl, 24px);
  border-bottom: 1px solid var(--admin-border-soft);

  &:last-of-type {
    border-bottom: 0;
  }
}

.upload-section-head {
  display: flex;
  align-items: flex-start;
  gap: var(--adm-gap-md, 12px);
  margin-bottom: var(--adm-gap-lg, 16px);
}

.upload-section-num {
  @include numeric;
  flex: none;
  padding-top: 2px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--adm-font-xs, 12px);
  letter-spacing: 0.08em;
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-section-lead {
  min-width: 0;
}

.upload-section-title {
  margin: 0;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  line-height: 1.4;
  letter-spacing: -0.01em;
  color: var(--adm-ink, var(--admin-text));
}

.upload-section-desc {
  margin: 4px 0 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-section-body {
  :deep(.arco-form-item:last-child) {
    margin-bottom: 0;
  }
}

.upload-input-hidden {
  display: none;
}

.upload-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-xs, 4px);
  margin-bottom: var(--adm-gap-sm, 8px);
}

.upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--adm-gap-xs, 4px);
  width: 100%;
  padding: var(--adm-gap-2xl, 32px) var(--adm-gap-xl, 24px);
  font: inherit;
  color: inherit;
  text-align: center;
  cursor: pointer;
  border: 1.5px dashed color-mix(in srgb, var(--adm-line, var(--admin-border)) 70%, var(--adm-accent, var(--admin-primary)));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: color-mix(in srgb, var(--adm-soft, var(--admin-fill)) 55%, transparent);
  transition:
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1)),
    background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1));

  @include focus-ring(2px);

  &:hover,
  &.is-dragover {
    border-color: var(--adm-accent, var(--admin-primary));
    background: var(--adm-accent-soft, var(--admin-primary-soft));
  }
}

.upload-area-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  margin-bottom: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-lg, 17px);
  color: var(--adm-accent, var(--admin-primary));
  background: var(--adm-accent-soft, var(--admin-primary-soft));
  border-radius: var(--adm-r-pill, 999px);
}

.upload-area-text {
  font-size: var(--adm-font-sm, 13px);
  font-weight: 500;
  color: var(--adm-ink, var(--admin-text));
}

.upload-area-hint {
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-preview {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--adm-gap-md, 12px);
}

.upload-preview-media {
  max-width: 100%;
  max-height: 220px;
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-control, var(--admin-radius));
  object-fit: contain;
  background: var(--adm-soft, var(--admin-fill));
}

.upload-preview-remove {
  min-height: var(--adm-touch, 36px);
}

.upload-legal {
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-legal-block {
  margin-top: var(--adm-gap-md, 12px);
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-control, var(--admin-radius));
  background: color-mix(in srgb, var(--adm-soft, var(--admin-fill)) 45%, transparent);
}

.upload-actions {
  position: sticky;
  bottom: 0;
  z-index: 3;
  display: flex;
  align-items: center;
  gap: var(--adm-gap-lg, 16px);
  padding: var(--adm-gap-md, 12px) var(--adm-gap-xl, 24px);
  border-top: 1px solid var(--admin-border-soft);
  background: color-mix(in srgb, var(--adm-canvas, var(--admin-surface)) 92%, var(--adm-surface, var(--admin-surface)));
}

.upload-actions-progress {
  flex: 1;
  min-width: 120px;
}

.upload-actions-buttons {
  display: flex;
  gap: var(--adm-gap-md, 12px);
  margin-left: auto;

  :deep(.arco-btn) {
    min-height: var(--adm-touch, 36px);
  }
}

@media (max-width: 820px) {
  .upload-section {
    padding: var(--adm-gap-lg, 16px);
  }

  .upload-actions {
    flex-wrap: wrap;
    gap: var(--adm-gap-sm, 8px);
    padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  }

  .upload-actions-buttons {
    width: 100%;
    margin-left: 0;

    :deep(.arco-btn) {
      flex: 1;
    }
  }
}

@media (max-width: 768px) {
  .upload-area {
    padding: var(--adm-gap-xl, 24px) var(--adm-gap-lg, 16px);
  }

  .upload-actions-buttons :deep(.arco-btn) {
    min-height: 44px;
  }
}
</style>
