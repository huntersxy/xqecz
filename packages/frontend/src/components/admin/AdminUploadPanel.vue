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
      <a-form layout="vertical" :model="form" class="upload-board">
        <section class="upload-card upload-card-base" aria-labelledby="up-base">
          <header class="upload-card-head">
            <span class="upload-card-num" aria-hidden="true">01</span>
            <div class="upload-card-lead">
              <h3 id="up-base" class="upload-card-title">基本信息</h3>
              <p class="upload-card-desc">标题与标签决定内容在站内的归类；图片可交给视觉模型自动生成。</p>
            </div>
          </header>
          <div class="upload-card-body">
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
                <div class="upload-tagcloud">
                  <TagCloud
                    :tags="admin.tags"
                    :selected-tags="form.tags"
                    :max-tags="50"
                    allow-custom
                    @toggle="toggleTag"
                    @add="handleAddCustomTag"
                    :disabled="admin.uploading"
                  />
                </div>
              </template>
            </a-form-item>
          </div>
        </section>

        <aside class="upload-card upload-card-media" aria-labelledby="up-media">
          <header class="upload-card-head">
            <span class="upload-card-num" aria-hidden="true">02</span>
            <div class="upload-card-lead">
              <h3 id="up-media" class="upload-card-title">媒体</h3>
              <p class="upload-card-desc">图片或视频可选；上传后由后台生成缩略图。</p>
            </div>
          </header>
          <div class="upload-card-body">
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
                <div class="upload-preview-meta">
                  <span class="upload-preview-kind">{{ fileKind === 'image' ? '图片' : '视频' }}</span>
                  <a-button size="small" class="upload-preview-remove" @click="clearFile">
                    <IconClose /> 移除
                  </a-button>
                </div>
              </div>
            </a-form-item>

            <a-form-item v-if="fileKind === 'video'">
              <a-checkbox v-model="agreeUpload">
                <span class="upload-legal">{{ VIDEO_TERMS_TEXT }}</span>
              </a-checkbox>
            </a-form-item>
            <div v-else class="upload-legal upload-legal-block" v-html="CC_LICENSE_TEXT"></div>

            <dl class="upload-check">
              <div class="upload-check-row">
                <dt>标题</dt>
                <dd :class="{ 'is-empty': !form.title.trim() && !autoMetadata }">{{ autoMetadata ? '视觉模型自动生成' : (form.title.trim() || '待填写') }}</dd>
              </div>
              <div class="upload-check-row">
                <dt>标签</dt>
                <dd :class="{ 'is-empty': !form.tags.length }">{{ form.tags.length ? form.tags.length + ' 个' : '未选择' }}</dd>
              </div>
              <div class="upload-check-row">
                <dt>媒体</dt>
                <dd :class="{ 'is-empty': !form.file }">{{ form.file ? (fileKind === 'image' ? '图片' : '视频') : '未选择' }}</dd>
              </div>
              <div class="upload-check-row">
                <dt>正文</dt>
                <dd :class="{ 'is-empty': !form.content.trim() }">{{ form.content.trim() ? form.content.trim().length + ' 字' : '未填写' }}</dd>
              </div>
            </dl>
          </div>
        </aside>

        <section class="upload-card upload-card-content" aria-labelledby="up-content">
          <header class="upload-card-head">
            <span class="upload-card-num" aria-hidden="true">03</span>
            <div class="upload-card-lead">
              <h3 id="up-content" class="upload-card-title">正文 Markdown</h3>
              <p class="upload-card-desc">支持 Markdown 排版；纯媒体投稿可留空，与媒体二选一或都填。</p>
            </div>
          </header>
          <div class="upload-card-body">
            <a-form-item label="描述正文（可选，与媒体二选一或都填）">
              <MarkdownEditor
                v-model="form.content"
                placeholder="支持 Markdown，描述、提示词、灵感…"
                :height="440"
              />
            </a-form-item>
          </div>
        </section>
      </a-form>

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

.upload-board {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(300px, 340px);
  grid-template-areas:
    'base media'
    'content media';
  gap: var(--adm-gap-lg, 16px);
  align-items: start;
  width: 100%;
  padding: var(--adm-gap-lg, 16px);
}

.upload-card-base {
  grid-area: base;
}

.upload-card-media {
  grid-area: media;
  position: sticky;
  top: var(--adm-gap-lg, 16px);
}

.upload-card-content {
  grid-area: content;
}

.upload-card {
  min-width: 0;
  padding: var(--adm-gap-lg, 16px) var(--adm-gap-xl, 20px) var(--adm-gap-xl, 20px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-surface, transparent);
}

.upload-card-head {
  display: flex;
  align-items: flex-start;
  gap: var(--adm-gap-md, 12px);
  margin-bottom: var(--adm-gap-lg, 16px);
  padding-bottom: var(--adm-gap-md, 12px);
  border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
}

.upload-card-num {
  @include numeric;
  flex: none;
  padding-top: 2px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--adm-font-xs, 12px);
  letter-spacing: 0.08em;
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-card-lead {
  min-width: 0;
}

.upload-card-title {
  margin: 0;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  line-height: 1.4;
  letter-spacing: -0.01em;
  color: var(--adm-ink, var(--admin-text));
}

.upload-card-desc {
  margin: 4px 0 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-card-body {
  :deep(.arco-form-item:last-child) {
    margin-bottom: 0;
  }
}

.upload-input-hidden {
  display: none;
}

.upload-tagcloud {
  max-height: 220px;
  overflow-y: auto;
  margin-right: -4px;
  padding-right: 4px;
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
  padding: var(--adm-gap-xl, 24px) var(--adm-gap-lg, 16px);
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
  border-radius: 50%;
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-accent, var(--admin-primary));
}

.upload-area-text {
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-ink, var(--admin-text));
}

.upload-area-hint {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-preview {
  position: relative;
}

.upload-preview-media {
  display: block;
  width: 100%;
  max-height: 320px;
  object-fit: contain;
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-soft, var(--admin-fill));
}

.upload-preview-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-sm, 8px);
  margin-top: var(--adm-gap-sm, 8px);
}

.upload-preview-kind {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-legal {
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));
}

.upload-legal-block {
  margin-top: var(--adm-gap-md, 12px);
}

.upload-check {
  margin: var(--adm-gap-lg, 16px) 0 0;
  padding-top: var(--adm-gap-md, 12px);
  border-top: 1px solid var(--adm-line, var(--admin-border-soft));
}

.upload-check-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  padding: 3px 0;

  dt {
    flex: none;
    font-size: var(--adm-font-xs, 12px);
    color: var(--adm-muted, var(--admin-text-3));
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow: hidden;
    font-size: var(--adm-font-xs, 12px);
    color: var(--adm-ink, var(--admin-text));
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  dd.is-empty {
    color: var(--adm-muted, var(--admin-text-3));
  }
}

.upload-actions {
  position: sticky;
  bottom: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  border-top: 1px solid var(--adm-line, var(--admin-border-soft));
  background: color-mix(in srgb, var(--adm-surface, var(--admin-panel)) 88%, transparent);
  backdrop-filter: blur(8px);
}

.upload-actions-progress {
  width: 100%;
}

.upload-actions-buttons {
  display: flex;
  justify-content: flex-end;
  gap: var(--adm-gap-sm, 8px);
}

@media (max-width: 1180px) {
  .upload-board {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas:
      'base'
      'media'
      'content';
  }

  .upload-card-media {
    position: static;
  }
}

@media (max-width: 768px) {
  .upload-board {
    padding: var(--adm-gap-md, 12px);
  }

  .upload-card {
    padding: var(--adm-gap-md, 12px) var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  }

  .upload-actions-buttons {
    :deep(.arco-btn) {
      flex: 1 1 0;
      height: var(--adm-touch, 44px);
    }
  }
}
</style>
