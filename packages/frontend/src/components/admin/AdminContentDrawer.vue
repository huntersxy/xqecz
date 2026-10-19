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
// 抽屉在手机上占满整屏，避免 720px 固定宽把内容挤出视口（§6.3 820 断点）。
const isMobile = useMediaQuery('(max-width: 820px)')
const drawerWidth = computed(() => (isMobile.value ? '100%' : 720))

const editFormModel: Record<string, unknown> = {}
const editTitle = ref('')
const editContent = ref('')
const editTags = ref<string[]>([])
const editFile = ref<File | undefined>()
const editFileName = ref('')
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
  editFile.value = raw
  editFileName.value = raw.name
}
</script>

<template>
  <a-drawer
    class="content-drawer admin-theme creative-theme creative-dialog"
    v-model:visible="admin.drawerOpen"
    :title="admin.drawerContent?.title || '内容详情'"
    :width="drawerWidth"
    @close="admin.closeDrawer"
  >
    <template v-if="admin.drawerContent">
      <div class="drawer-body">
        <a-tabs v-model:active-key="tabKey" class="drawer-tabs">
          <a-tab-pane key="preview" title="预览">
            <div class="drawer-meta">
              <AdminStatus
                v-if="admin.drawerContent.audit_status"
                :type="admin.drawerContent.audit_status === 'approved' ? 'success' : admin.drawerContent.audit_status === 'pending' ? 'warning' : 'danger'"
                :label="{ approved: '已通过', pending: '审核中', rejected: '已拒绝' }[admin.drawerContent.audit_status]"
              />
              <span class="drawer-meta-info">
                {{ admin.drawerContent.like_count || 0 }} 点赞 · {{ admin.drawerContent.user?.username }}
              </span>
            </div>

            <div v-if="(admin.drawerContent.tags || []).length" class="drawer-tags">
              <Tag v-for="tag in admin.drawerContent.tags" :key="tag" size="small" :bordered="false">{{ tag }}</Tag>
            </div>

            <div v-if="admin.drawerContent.video" class="drawer-media-wrap">
              <video controls class="drawer-media">
                <source :src="getImageUrl(admin.drawerContent.video)" />
                <track kind="captions" />
              </video>
            </div>

            <div v-else-if="admin.drawerContent.img || admin.drawerContent.thumb" class="drawer-media-wrap">
              <MediaImage :src="admin.drawerContent.img" class="drawer-media" :preview="false" alt="" />
            </div>

            <div v-if="admin.drawerContent.text" class="drawer-markdown" v-html="renderedContent"></div>
          </a-tab-pane>

          <a-tab-pane key="edit" title="编辑">
            <a-form :model="editFormModel" layout="vertical" class="drawer-form">
              <a-form-item label="标题" required>
                <a-input v-model="editTitle" :maxlength="200" show-word-limit placeholder="输入标题" />
              </a-form-item>

              <a-form-item label="描述（Markdown，可留空为纯媒体内容）">
                <MarkdownEditor
                  v-model="editContent"
                  placeholder="支持 Markdown，可留空（纯媒体内容）"
                  :height="360"
                />
              </a-form-item>

              <a-form-item label="媒体文件">
                <a-upload
                  :auto-upload="false"
                  :show-file-list="false"
                  :limit="1"
                  accept="image/*,video/*"
                  @change="onFileChange"
                >
                  <template #upload-button>
                    <a-button>
                      <IconUpload />
                      {{ editFileName || '选择图片 / 视频' }}
                    </a-button>
                  </template>
                </a-upload>
                <a-typography-text
                  v-if="editFileName"
                  type="secondary"
                  class="drawer-file-name"
                >
                  {{ editFileName }}
                </a-typography-text>
                <a-typography-text
                  v-else-if="admin.drawerContent.img || admin.drawerContent.video || admin.drawerContent.thumb"
                  type="secondary"
                  class="drawer-file-name"
                >
                  当前已有媒体文件，可在预览页查看；选择新文件可替换
                </a-typography-text>
              </a-form-item>

              <a-form-item label="标签">
                <div class="drawer-field-stack">
                  <a-input-tag v-model="editTags" placeholder="输入标签后回车" allow-clear class="drawer-tags-input" />
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
          </a-tab-pane>

          <a-tab-pane key="author" title="修改作者">
            <div class="author-current">
              <span class="author-current-label">当前作者：</span>
              <Tag :bordered="false" class="drawer-tag-inline admin-tag-accent">{{ admin.drawerContent.user?.username }}</Tag>
            </div>
            <a-input v-model="authorKeyword" placeholder="搜索用户名..." allow-clear class="author-search">
              <template #prefix><IconSearch /></template>
            </a-input>
            <a-spin :loading="usersLoading">
              <a-list v-if="filteredUsers.length" :data="filteredUsers" :bordered="false" :split="false" class="user-list">
                <template #item="{ item }">
                  <a-list-item class="user-item" @click="handleChangeAuthor(item.id, item.username)">
                    <UserAvatar :size="24" :email="item.email" :username="item.username" />
                    <span class="admin-cell-title">{{ item.username }}</span>
                    <Tag v-if="item.is_admin" size="small" :bordered="false" class="drawer-tag-inline admin-tag-accent">管理员</Tag>
                  </a-list-item>
                </template>
              </a-list>
              <AdminEmpty v-else compact title="无匹配用户" />
            </a-spin>
          </a-tab-pane>
        </a-tabs>
      </div>
    </template>

    <template #footer>
      <div class="drawer-footer">
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
    </template>
  </a-drawer>
</template>

<style lang="scss" scoped>
@use './admin' as *;

// 头部与底栏由 Arco 固定，正文区独立滚动；页签栏在滚动区内吸顶（§10 内容抽屉）。
.content-drawer {
  :deep(.arco-drawer-header) {
    padding: 16px var(--adm-gap-xl, 24px);
    border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
  }

  :deep(.arco-drawer-title) {
    font-family: var(--site-title-font);
    font-size: var(--adm-font-lg, 17px);
    font-weight: 600;
    letter-spacing: -.01em;
    color: var(--adm-ink, var(--admin-text));
  }

  :deep(.arco-drawer-content) {
    padding: 0;
    overflow-y: auto;
  }

  :deep(.arco-drawer-body) {
    padding: 0;
  }

  :deep(.arco-drawer-footer) {
    padding: 12px var(--adm-gap-xl, 24px);
    border-top: 1px solid var(--adm-line, var(--admin-border-soft));
  }
}

.drawer-body {
  display: flex;
  flex-direction: column;
  min-height: 100%;
}

.drawer-tabs {
  :deep(.arco-tabs-header) {
    position: sticky;
    top: 0;
    z-index: 3;
    padding: 8px var(--adm-gap-xl, 24px) 0;
    background: var(--adm-surface, var(--admin-surface));
    border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
  }

  :deep(.arco-tabs-content) {
    padding: var(--adm-gap-xl, 24px);
  }
}

.drawer-meta {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  margin-bottom: var(--adm-gap-md, 12px);
}

.drawer-meta-info {
  @include numeric;
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted, var(--admin-text-3));
}

.drawer-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-xs, 4px);
  margin-bottom: var(--adm-gap-lg, 16px);
}

.drawer-media-wrap {
  display: flex;
  justify-content: center;
  margin-bottom: var(--adm-gap-lg, 16px);
}

.drawer-media {
  display: block;
  max-width: 100%;
  max-height: 460px;
  border-radius: var(--adm-r-card, 12px);

  :deep(.arco-image-img) {
    max-width: 100%;
    height: auto;
    display: block;
  }
}

.drawer-markdown {
  font-size: var(--adm-font-base, 14px);
  line-height: 1.8;
  color: var(--adm-text-2, var(--admin-text-2));

  :deep(h1), :deep(h2), :deep(h3) {
    margin: var(--adm-gap-lg, 16px) 0 var(--adm-gap-sm, 8px);
    font-family: var(--site-title-font);
    font-weight: 600;
    letter-spacing: -.01em;
    color: var(--adm-ink, var(--admin-text));
  }

  :deep(h1) { font-size: var(--adm-font-xl, 21px); }
  :deep(h2) { font-size: var(--adm-font-lg, 17px); }
  :deep(h3) { font-size: var(--adm-font-md, 15px); }

  :deep(p) { margin-bottom: var(--adm-gap-md, 12px); }
  :deep(ul), :deep(ol) { margin-bottom: var(--adm-gap-md, 12px); padding-left: 24px; }
  :deep(blockquote) {
    margin: var(--adm-gap-md, 12px) 0;
    padding-left: var(--adm-gap-md, 12px);
    border-left: 2px solid var(--adm-line, var(--admin-border));
    color: var(--adm-muted, var(--admin-text-3));
    font-style: italic;
  }

  :deep(code) {
    padding: 2px 6px;
    border-radius: 6px;
    background: var(--adm-soft, var(--admin-fill));
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: var(--adm-font-xs, 12px);
  }

  :deep(pre) {
    margin: var(--adm-gap-md, 12px) 0;
    padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
    overflow-x: auto;
    border: 1px solid var(--adm-line, var(--admin-border-soft));
    border-radius: var(--adm-r-card, 12px);
    background: var(--adm-soft, var(--admin-fill));
    color: var(--adm-ink, var(--admin-text));
  }

  :deep(pre code) {
    padding: 0;
    background: none;
    color: inherit;
    font-size: var(--adm-font-sm, 13px);
  }
}

.drawer-form {
  :deep(.arco-form-item) {
    margin-bottom: var(--adm-gap-lg, 16px);
  }

  :deep(.arco-form-item-label) {
    font-size: var(--adm-font-sm, 13px);
    color: var(--adm-muted, var(--admin-text-2));
  }

  :deep(.arco-form-item-content),
  .drawer-field-stack {
    display: block;
    width: 100%;
  }
}

.drawer-file-name {
  display: block;
  margin-top: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-xs, 12px);
}

.drawer-tags-input {
  width: 100%;
}

.tag-pool {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-xs, 4px);
  margin-top: var(--adm-gap-md, 12px);
  padding: var(--adm-gap-md, 12px);
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--admin-fill));
}

.tag-pool-item,
.drawer-tag-inline {
  margin: 0;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--adm-gap-sm, 8px);
}

.author-current {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-xs, 4px);
  margin-bottom: var(--adm-gap-md, 12px);
}

.author-current-label {
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted, var(--admin-text-3));
}

.author-search {
  margin-bottom: var(--adm-gap-md, 12px);
}

.user-list {
  max-height: 320px;
  overflow-y: auto;
}

.user-item {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-sm, 8px) var(--adm-gap-md, 12px);
  border-radius: var(--adm-r-control, 10px);
  cursor: pointer;
  transition: background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));

  &:hover {
    background: var(--adm-soft, var(--admin-fill));
  }
}
</style>
