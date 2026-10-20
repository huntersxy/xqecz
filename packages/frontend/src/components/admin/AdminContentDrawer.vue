<script setup lang="ts">
import UserAvatar from '@/components/UserAvatar.vue'
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api'
import { getImageUrl, renderMarkdown } from '@/utils'
import MediaImage from '@/components/MediaImage.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import AdminEmpty from './AdminEmpty.vue'
import { Tag, type FileItem } from '@arco-design/web-vue'
import { toast, useConfirm } from '@/composables/useToast'
import { useMediaQuery } from '@vueuse/core'
import type { User } from '@/types'
import { IconUpload, IconSearch } from '@arco-design/web-vue/es/icon'

const admin = useAdminStore()
// 桌面用居中大弹窗（1040px），手机上占满整屏，避免窄屏把双栏表单压扁。
const isMobile = useMediaQuery('(max-width: 820px)')
const dialogWidth = computed(() => (isMobile.value ? '100%' : 1040))

const editFormModel: Record<string, unknown> = {}
const editTitle = ref('')
const editContent = ref('')
const editTags = ref<string[]>([])
const editFile = ref<File | undefined>()
const editFileName = ref('')
const editFilePreview = ref('')
const editFileKind = ref<'' | 'image' | 'video'>('')
const tabKey = ref('preview')

const authorKeyword = ref('')
const allUsers = ref<User[]>([])
const usersLoading = ref(false)

const renderedContent = computed(() => {
  if (!admin.drawerContent) return ''
  return renderMarkdown(admin.drawerContent.text || '')
})

const filteredUsers = computed(() => {
  if (!authorKeyword.value.trim()) return allUsers.value
  const kw = authorKeyword.value.toLowerCase()
  return allUsers.value.filter(u => u.username.toLowerCase().includes(kw))
})

watch(() => admin.drawerOpen, async (open) => {
  if (open && admin.drawerContent) {
    const detail = await admin.fetchContentDetail(admin.drawerContent.id)
    if (detail) admin.drawerContent = detail
    editTitle.value = admin.drawerContent.title
    editContent.value = admin.drawerContent.text || ''
    editTags.value = [...(admin.drawerContent.tags || [])]
    editFile.value = undefined
    editFileName.value = ''
    editFileKind.value = ''
    if (editFilePreview.value) URL.revokeObjectURL(editFilePreview.value)
    editFilePreview.value = ''
    // 编辑入口直接从编辑页签打开，免得用户再点一次页签
    tabKey.value = admin.drawerMode === 'edit' ? 'edit' : 'preview'
    loadUsers()
  }
})

async function loadUsers() {
  usersLoading.value = true
  try {
    const r = await adminApi.getUsers({ page_size: 200 })
    if (r.code === 200) allUsers.value = r.data.list
  } catch { /* */ } finally { usersLoading.value = false }
}

async function handleSave() {
  if (!admin.drawerContent) return
  const ok = await admin.saveContent(admin.drawerContent.id, {
    title: editTitle.value, content: editContent.value,
    tags: editTags.value, file: editFile.value,
  })
  if (ok) admin.closeDrawer()
}

async function handleChangeAuthor(userId: number, username: string) {
  if (!admin.drawerContent) return
  const { confirm } = useConfirm()
  const ok = await confirm(`确定将作者改为「${username}」吗？`)
  if (!ok) return
  const changed = await admin.changeAuthor(admin.drawerContent.id, userId)
  if (changed) admin.closeDrawer()
}

function setTagChecked(tag: string, checked: boolean) {
  const i = editTags.value.indexOf(tag)
  if (checked && i === -1) editTags.value.push(tag)
  if (!checked && i > -1) editTags.value.splice(i, 1)
}

function onFileChange(_fileList: FileItem[], fileItem: FileItem) {
  const raw = fileItem?.file as File | undefined
  if (!raw) return
  if (!raw.type.startsWith('image/') && !raw.type.startsWith('video/')) {
    toast.error('仅支持图片或视频文件')
    return
  }
  // 与后端保持一致：单文件最大 20MB
  if (raw.size > 20 * 1024 * 1024) {
    toast.error('文件大小不能超过 20MB')
    return
  }
  if (editFilePreview.value) URL.revokeObjectURL(editFilePreview.value)
  editFile.value = raw
  editFileName.value = raw.name
  editFileKind.value = raw.type.startsWith('image/') ? 'image' : 'video'
  editFilePreview.value = URL.createObjectURL(raw)
}

onBeforeUnmount(() => {
  if (editFilePreview.value) URL.revokeObjectURL(editFilePreview.value)
})
</script>

<template>
  <a-modal
    class="content-dialog admin-theme creative-theme creative-dialog"
    v-model:visible="admin.drawerOpen"
    :title="admin.drawerContent?.title || '内容详情'"
    :width="dialogWidth"
    :top="isMobile ? '0' : '5vh'"
    :footer="false"
    @close="admin.closeDrawer"
  >
    <div v-if="admin.drawerContent" class="dialog-body">
      <a-tabs v-model:active-key="tabKey" class="dialog-tabs">
        <a-tab-pane key="preview" title="预览">
          <div class="preview-grid">
            <div class="preview-media-pane">
              <div class="media-frame">
                <video v-if="admin.drawerContent.video" controls class="media-el">
                  <source :src="getImageUrl(admin.drawerContent.video)" />
                  <track kind="captions" />
                </video>
                <MediaImage
                  v-else-if="admin.drawerContent.img || admin.drawerContent.thumb"
                  class="media-el"
                  :src="admin.drawerContent.img || admin.drawerContent.thumb"
                  :preview="false"
                  alt=""
                />
                <div v-else class="media-empty">纯文字内容，没有媒体文件</div>
              </div>
              <div class="media-meta">
                <AdminStatus
                  v-if="admin.drawerContent.audit_status"
                  :type="admin.drawerContent.audit_status === 'approved' ? 'success' : admin.drawerContent.audit_status === 'pending' ? 'warning' : 'danger'"
                  :label="{ approved: '已通过', pending: '审核中', rejected: '已拒绝' }[admin.drawerContent.audit_status]"
                />
                <span class="media-meta-info">
                  {{ admin.drawerContent.like_count || 0 }} 点赞 · {{ admin.drawerContent.user?.username }}
                </span>
              </div>
            </div>

            <div class="preview-text-pane">
              <div v-if="(admin.drawerContent.tags || []).length" class="dialog-tags">
                <Tag v-for="tag in admin.drawerContent.tags" :key="tag" size="small" :bordered="false">{{ tag }}</Tag>
              </div>
              <div v-if="admin.drawerContent.text" class="dialog-markdown" v-html="renderedContent"></div>
              <AdminEmpty v-else compact title="没有正文" desc="这条内容只有媒体文件。" />
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane key="edit" title="编辑">
          <div class="edit-grid">
            <div class="edit-main">
              <a-form :model="editFormModel" layout="vertical" class="dialog-form">
                <a-form-item label="标题" required>
                  <a-input v-model="editTitle" :maxlength="200" show-word-limit placeholder="输入标题" />
                </a-form-item>
                <a-form-item label="描述（Markdown，可留空为纯媒体内容）">
                  <MarkdownEditor
                    v-model="editContent"
                    placeholder="支持 Markdown，可留空（纯媒体内容）"
                    :height="380"
                  />
                </a-form-item>
              </a-form>
            </div>

            <aside class="edit-aside">
              <a-form :model="editFormModel" layout="vertical" class="dialog-form">
                <a-form-item label="媒体文件">
                  <div v-if="editFilePreview" class="edit-media-current">
                    <img v-if="editFileKind === 'image'" :src="editFilePreview" class="edit-media-thumb" alt="待上传的图片预览" />
                    <video v-else :src="editFilePreview" controls class="edit-media-thumb" />
                    <span class="edit-media-tag">待保存</span>
                  </div>
                  <div v-else-if="admin.drawerContent.img || admin.drawerContent.video || admin.drawerContent.thumb" class="edit-media-current">
                    <video v-if="admin.drawerContent.video" :src="getImageUrl(admin.drawerContent.video)" controls class="edit-media-thumb" />
                    <MediaImage
                      v-else
                      class="edit-media-thumb"
                      :src="admin.drawerContent.img || admin.drawerContent.thumb"
                      :preview="false"
                      alt=""
                    />
                    <span class="edit-media-tag">当前</span>
                  </div>
                  <a-upload
                    :auto-upload="false"
                    :show-file-list="false"
                    :limit="1"
                    accept="image/*,video/*"
                    @change="onFileChange"
                  >
                    <template #upload-button>
                      <a-button long>
                        <IconUpload />
                        {{ editFileName || '选择图片 / 视频' }}
                      </a-button>
                    </template>
                  </a-upload>
                  <a-typography-text v-if="editFileName" type="secondary" class="dialog-file-name">
                    {{ editFileName }}
                  </a-typography-text>
                </a-form-item>

                <a-form-item label="标签">
                  <div class="dialog-field-stack">
                    <a-input-tag v-model="editTags" placeholder="输入标签后回车" allow-clear class="dialog-tags-input" />
                    <div v-if="admin.tags.length" class="tag-pool">
                      <a-tag
                        v-for="tag in admin.tags"
                        :key="tag"
                        checkable
                        :checked="editTags.includes(tag)"
                        :bordered="false"
                        size="small"
                        class="tag-pool-item"
                        @check="setTagChecked(tag, $event)"
                      >
                        {{ tag }}
                      </a-tag>
                    </div>
                  </div>
                </a-form-item>
              </a-form>
            </aside>
          </div>
        </a-tab-pane>

        <a-tab-pane key="author" title="修改作者">
          <div class="author-pane">
            <div class="author-bar">
              <span class="author-current">
                <span class="author-current-label">当前作者</span>
                <Tag :bordered="false" class="dialog-tag-inline admin-tag-accent">{{ admin.drawerContent.user?.username }}</Tag>
              </span>
              <a-input v-model="authorKeyword" placeholder="搜索用户名..." allow-clear class="author-search">
                <template #prefix><IconSearch /></template>
              </a-input>
            </div>
            <a-spin :loading="usersLoading" class="author-spin">
              <div v-if="filteredUsers.length" class="author-list">
                <button
                  v-for="item in filteredUsers"
                  :key="item.id"
                  type="button"
                  class="author-item"
                  @click="handleChangeAuthor(item.id, item.username)"
                >
                  <UserAvatar :size="24" :email="item.email" :username="item.username" />
                  <span class="author-name">{{ item.username }}</span>
                  <Tag v-if="item.is_admin" size="small" :bordered="false" class="dialog-tag-inline admin-tag-accent">管理员</Tag>
                </button>
              </div>
              <AdminEmpty v-else compact title="无匹配用户" />
            </a-spin>
          </div>
        </a-tab-pane>
      </a-tabs>

      <div class="dialog-actions">
        <a-button @click="admin.closeDrawer">关闭</a-button>
        <a-button
          v-if="tabKey === 'edit'"
          type="primary"
          :loading="admin.drawerSaving"
          @click="handleSave"
        >
          保存修改
        </a-button>
      </div>
    </div>
  </a-modal>
</template>

<style lang="scss">
@use './admin' as *;

// 弹窗 Teleport 到 body，作用域属性不跟随，这里用全局类名前缀隔离（只有 .content-dialog 内生效）。
.content-dialog {
  .arco-modal {
    max-width: 96vw;
    overflow: hidden;
    border-radius: var(--adm-r-card, 16px);
  }

  .arco-modal-header {
    padding: 14px var(--adm-gap-xl, 20px);
    border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
  }

  .arco-modal-title {
    font-family: var(--site-title-font);
    font-size: var(--adm-font-lg, 17px);
    font-weight: 600;
    letter-spacing: -.01em;
    color: var(--adm-ink, var(--admin-text));
  }

  .arco-modal-body {
    padding: 0;
  }
}

// 弹窗本身不滚：头部固定、页签条固定、内容区自己滚、底部操作条常驻
.dialog-body {
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 150px);
  min-height: 0;
}

.dialog-tabs {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;

  .arco-tabs-nav {
    flex: none;
    padding: 6px var(--adm-gap-xl, 20px) 0;
    border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
  }

  .arco-tabs-content {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    padding: var(--adm-gap-xl, 20px);
  }
}

// 预览：左侧媒体固定比例，右侧文字铺满剩余宽度
.preview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr);
  gap: var(--adm-gap-xl, 20px);
  align-items: start;
}

.media-frame {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 260px;
  max-height: calc(100vh - 340px);
  overflow: hidden;
  border-radius: var(--adm-r-card, 12px);
  background: var(--adm-soft, var(--admin-fill));
}

.media-el {
  display: block;
  max-width: 100%;
  max-height: calc(100vh - 340px);

  .arco-image-img {
    max-width: 100%;
    height: auto;
    display: block;
    object-fit: contain;
  }
}

.media-empty {
  padding: var(--adm-gap-2xl, 32px);
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted, var(--admin-text-3));
}

.media-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--adm-gap-md, 12px);
  margin-top: var(--adm-gap-md, 12px);
}

.media-meta-info {
  @include numeric;
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted, var(--admin-text-3));
}

.preview-text-pane {
  min-height: 240px;
  padding: var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, 12px);
}

.dialog-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-xs, 4px);
  margin-bottom: var(--adm-gap-lg, 16px);
}

.dialog-markdown {
  font-size: var(--adm-font-base, 14px);
  line-height: 1.8;
  color: var(--adm-text-2, var(--admin-text-2));

  h1, h2, h3 {
    margin: var(--adm-gap-lg, 16px) 0 var(--adm-gap-sm, 8px);
    font-family: var(--site-title-font);
    font-weight: 600;
    letter-spacing: -.01em;
    color: var(--adm-ink, var(--admin-text));
  }

  h1 { font-size: var(--adm-font-xl, 21px); }
  h2 { font-size: var(--adm-font-lg, 17px); }
  h3 { font-size: var(--adm-font-md, 15px); }

  p { margin-bottom: var(--adm-gap-md, 12px); }
  ul, ol { margin-bottom: var(--adm-gap-md, 12px); padding-left: 24px; }
  blockquote {
    margin: var(--adm-gap-md, 12px) 0;
    padding-left: var(--adm-gap-md, 12px);
    border-left: 2px solid var(--adm-line, var(--admin-border));
    color: var(--adm-muted, var(--admin-text-3));
    font-style: italic;
  }

  code {
    padding: 2px 6px;
    border-radius: 6px;
    background: var(--adm-soft, var(--admin-fill));
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: var(--adm-font-xs, 12px);
  }

  pre {
    margin: var(--adm-gap-md, 12px) 0;
    padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
    overflow-x: auto;
    border: 1px solid var(--adm-line, var(--admin-border-soft));
    border-radius: var(--adm-r-card, 12px);
    background: var(--adm-soft, var(--admin-fill));
    color: var(--adm-ink, var(--admin-text));
  }

  pre code {
    padding: 0;
    background: none;
    color: inherit;
    font-size: var(--adm-font-sm, 13px);
  }
}

// 编辑：正文编辑器占主栏，媒体与标签放侧栏，避免整页一根长条
.edit-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 320px);
  gap: var(--adm-gap-xl, 20px);
  align-items: start;
}

.edit-aside {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  padding: var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, 12px);
  background: color-mix(in srgb, var(--adm-soft, var(--admin-fill)) 45%, transparent);
}

.dialog-form {
  .arco-form-item {
    margin-bottom: var(--adm-gap-lg, 16px);
  }

  .arco-form-item-label {
    font-size: var(--adm-font-sm, 13px);
    color: var(--adm-muted, var(--admin-text-2));
  }

  .arco-form-item-content {
    display: block;
    width: 100%;
  }
}

.dialog-field-stack {
  display: block;
  width: 100%;
}

// 编辑页签里的媒体缩略图：让用户看清将替换的是什么，而不是只有一行文字
.edit-media-current {
  position: relative;
  margin-bottom: var(--adm-gap-sm, 8px);
  overflow: hidden;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--admin-fill));
}

.edit-media-thumb {
  display: block;
  width: 100%;
  max-height: 160px;
  object-fit: cover;

  .arco-image-img {
    display: block;
    width: 100%;
    object-fit: cover;
  }
}

.edit-media-tag {
  position: absolute;
  top: var(--adm-gap-sm, 8px);
  left: var(--adm-gap-sm, 8px);
  padding: 2px 8px;
  border-radius: var(--adm-r-pill, 999px);
  background: color-mix(in srgb, var(--adm-surface, #fff) 82%, transparent);
  font-size: 11px;
  color: var(--adm-muted, var(--admin-text-3));
  backdrop-filter: blur(6px);
}

.dialog-file-name {
  display: block;
  margin-top: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-xs, 12px);
}

.dialog-tags-input {
  width: 100%;
}

.tag-pool {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-xs, 4px);
  max-height: 168px;
  overflow-y: auto;
  margin-top: var(--adm-gap-md, 12px);
  padding: var(--adm-gap-md, 12px);
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--admin-fill));
}

.tag-pool-item,
.dialog-tag-inline {
  margin: 0;
}

// 修改作者：搜索在顶部，用户按两列铺开
.author-pane {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
}

.author-bar {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-lg, 16px);
  flex-wrap: wrap;
}

.author-current {
  display: inline-flex;
  align-items: center;
  gap: var(--adm-gap-xs, 4px);
}

.author-current-label {
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted, var(--admin-text-3));
}

.author-search {
  flex: 1 1 240px;
  max-width: 320px;
}

.author-spin {
  display: block;
}

.author-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--adm-gap-xs, 4px);
  max-height: calc(100vh - 420px);
  min-height: 200px;
  overflow-y: auto;
}

.author-item {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  width: 100%;
  padding: var(--adm-gap-sm, 8px) var(--adm-gap-md, 12px);
  font: inherit;
  color: inherit;
  text-align: left;
  border: 0;
  border-radius: var(--adm-r-control, 10px);
  background: transparent;
  cursor: pointer;
  transition: background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));

  &:hover,
  &:focus-visible {
    background: var(--adm-soft, var(--admin-fill));
  }
}

.author-name {
  min-width: 0;
  overflow: hidden;
  font-size: var(--adm-font-sm, 13px);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-md, 12px) var(--adm-gap-xl, 20px);
  border-top: 1px solid var(--adm-line, var(--admin-border-soft));
}

@media (max-width: 980px) {
  .preview-grid,
  .edit-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .media-frame,
  .media-el {
    max-height: 320px;
  }
}

@media (max-width: 820px) {
  // 手机占满整屏，别在底部留一条能看到页面的缝
  .content-dialog .arco-modal {
    width: 100vw !important;
    max-width: 100vw;
    height: 100vh;
    max-height: 100vh;
    margin: 0;
    border-radius: 0;
  }

  .content-dialog .arco-modal-body {
    height: calc(100vh - 48px);
  }

  .dialog-body {
    height: 100%;
    max-height: 100%;
  }

  .dialog-tabs .arco-tabs-content {
    max-height: none;
  }

  .author-list {
    grid-template-columns: minmax(0, 1fr);
    max-height: none;
  }
}
</style>
