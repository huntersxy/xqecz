<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { adminApi } from '@/api'
import { toast } from '@/composables/useToast'
import { useSiteStore } from '@/stores/site'
import type { ApiResponse, SiteConfig, SiteSettings, SiteTextElement } from '@/types'
import { DEFAULT_SITE_TEXTS, renderSiteText } from '@/utils/siteText'
import { IconDelete, IconPlus, IconUpload } from '@arco-design/web-vue/es/icon'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminStatus from './AdminStatus.vue'
import AdminDataToolbar from './AdminDataToolbar.vue'

const friendLinkPlaceholder = 'https://example.com/'
const settings = ref<SiteSettings>()
const busy = ref(false)
const uploading = ref<'' | 'logo' | 'favicon'>('')
const error = ref('')
const textFilter = ref('')
const savedSnapshot = ref('')
const logoInput = ref<HTMLInputElement | null>(null)
const faviconInput = ref<HTMLInputElement | null>(null)
const controller = new AbortController()
const siteStore = useSiteStore()

const groups = computed(() => {
  const grouped = new Map<string, SiteTextElement[]>()
  const keyword = textFilter.value.trim().toLowerCase()
  for (const element of settings.value?.elements ?? []) {
    if (keyword && !`${element.label} ${element.key} ${element.group}`.toLowerCase().includes(keyword)) continue
    const items = grouped.get(element.group) ?? []
    items.push(element)
    grouped.set(element.group, items)
  }
  return [...grouped.entries()].map(([label, items]) => ({ label, items }))
})

const dirty = computed(() => {
  if (!settings.value) return false
  return JSON.stringify(configPayload(settings.value)) !== savedSnapshot.value
})

const summary = computed(() => {
  if (!settings.value) return '正在载入站点配置'
  return `${settings.value.elements.length} 个文本元素 · 友链 ${settings.value.friend_links.length} 条`
})

const visibleTextCount = computed(() => groups.value.reduce((total, group) => total + group.items.length, 0))

function configPayload(value: SiteSettings): SiteConfig {
  return {
    name: value.name,
    short_name: value.short_name,
    tagline: value.tagline,
    browser_title: value.browser_title,
    description: value.description,
    logo_url: value.logo_url,
    favicon_url: value.favicon_url,
    friend_links: value.friend_links.map((link) => ({ ...link })),
    texts: { ...value.texts },
  }
}

function data<T>(response: ApiResponse<T>): T {
  if (response.code !== 200) throw new Error(response.message || '操作失败')
  return response.data
}

async function run(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await action()
  } catch (e) {
    if (!controller.signal.aborted) error.value = (e as Error).message || '操作失败'
  } finally {
    if (!controller.signal.aborted) busy.value = false
  }
}

async function load() {
  await run(async () => {
    const response = await adminApi.siteSettings(controller.signal)
    if (controller.signal.aborted) return
    settings.value = data(response)
    savedSnapshot.value = JSON.stringify(configPayload(settings.value))
  })
}

async function save() {
  if (!settings.value) return
  const response = await adminApi.saveSiteSettings(configPayload(settings.value), controller.signal)
  if (controller.signal.aborted) return
  settings.value = data(response)
  savedSnapshot.value = JSON.stringify(configPayload(settings.value))
  siteStore.applyConfig(settings.value)
}

async function handleSave() {
  await run(async () => {
    await save()
    if (!controller.signal.aborted) toast.success('站点外观已保存')
  })
}

async function uploadAsset(kind: 'logo' | 'favicon', event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !settings.value) return
  if (file.size > 4 << 20) {
    toast.error('图片不能超过 4MB')
    return
  }
  uploading.value = kind
  try {
    const response = await adminApi.uploadSiteAsset(kind, file, controller.signal)
    if (controller.signal.aborted) return
    const asset = data(response)
    if (kind === 'logo') settings.value.logo_url = asset.url
    else settings.value.favicon_url = asset.url
    toast.success('图片已上传，保存后生效')
  } catch (e) {
    if (!controller.signal.aborted) error.value = (e as Error).message || '图片上传失败'
  } finally {
    if (!controller.signal.aborted) uploading.value = ''
  }
}

function addFriend() {
  if (!settings.value || settings.value.friend_links.length >= 12) return
  settings.value.friend_links.push({ name: '', url: '', icon_url: '' })
}

function removeFriend(index: number) {
  settings.value?.friend_links.splice(index, 1)
}

function resetText(element: SiteTextElement) {
  if (!settings.value) return
  settings.value.texts[element.key] = element.default
}

function resetAllTexts() {
  if (!settings.value) return
  for (const element of settings.value.elements) settings.value.texts[element.key] = element.default
}

function preview(template: string) {
  if (!settings.value) return ''
  return renderSiteText(template || '', settings.value)
}

function previewText(key: string) {
  return preview(settings.value?.texts[key] ?? DEFAULT_SITE_TEXTS[key] ?? '')
}

onMounted(load)
onBeforeUnmount(() => controller.abort())
</script>

<template>
  <AdminPanel
    title="站点外观"
    desc="集中管理站点名称、Logo、图标、友链与前台文案；保存后访客看到的内容立即更新。"
    flow
  >
    <template #actions>
      <AdminStatus
        :type="dirty ? 'warning' : 'success'"
        :label="dirty ? '有未保存修改' : '与线上一致'"
      />
      <a-button type="primary" :loading="busy" :disabled="!settings" @click="handleSave">保存更改</a-button>
    </template>

    <template #toolbar>
      <AdminDataToolbar :summary="summary" :loading="busy" refreshable @refresh="load" />
    </template>

    <a-alert v-if="error" type="error" :title="error" class="site-error" />

    <AdminSkeleton v-if="!settings" variant="rows" :rows="6" :columns="2" label="正在载入站点外观" />

    <div v-else class="adm-board site-board">
      <div class="adm-board-main">
        <section class="adm-card site-card" aria-labelledby="site-brand-title">
          <div class="site-card-head">
            <div>
              <h3 id="site-brand-title" class="site-card-title">品牌与 SEO</h3>
              <p class="site-card-desc">站点名称与标语会出现在侧边栏；浏览器标题支持 {name}、{short_name}、{tagline} 占位符。</p>
            </div>
          </div>
          <a-form :model="settings" layout="vertical" class="site-form">
            <div class="site-grid">
              <a-form-item label="站点名称" required>
                <a-input v-model="settings.name" :maxlength="40" show-word-limit />
              </a-form-item>
              <a-form-item label="短名称">
                <a-input v-model="settings.short_name" :maxlength="20" show-word-limit placeholder="留空跟随站点名称" />
              </a-form-item>
              <a-form-item label="站点标语">
                <a-input v-model="settings.tagline" :maxlength="40" show-word-limit placeholder="显示在站点名称下方" />
              </a-form-item>
              <a-form-item label="浏览器标题">
                <a-input v-model="settings.browser_title" :maxlength="120" show-word-limit placeholder="{name} · 站点副标题" />
              </a-form-item>
              <a-form-item class="is-full" label="站点描述">
                <a-textarea
                  v-model="settings.description"
                  :maxlength="300"
                  :auto-size="{ minRows: 2, maxRows: 4 }"
                  show-word-limit
                  placeholder="用于搜索引擎与社交分享"
                />
              </a-form-item>
            </div>
          </a-form>
        </section>

        <section class="adm-card site-card" aria-labelledby="site-assets-title">
          <div class="site-card-head">
            <div>
              <h3 id="site-assets-title" class="site-card-title">Logo 与图标</h3>
              <p class="site-card-desc">支持 PNG、JPEG、WebP、GIF，单张不超过 4MB；图标会自动裁切为 256×256。</p>
            </div>
          </div>
          <div class="site-assets">
            <article class="site-asset">
              <div class="site-asset-preview" :class="{ 'is-empty': !settings.logo_url }">
                <img v-if="settings.logo_url" :src="settings.logo_url" alt="站点 Logo 预览" />
                <span v-else>暂无 Logo</span>
              </div>
              <div class="site-asset-body">
                <strong>站点 Logo</strong>
                <p>显示在侧边栏品牌区。留空则只展示站点名称。</p>
                <div class="site-asset-actions">
                  <a-button size="small" :loading="uploading === 'logo'" @click="logoInput?.click()">
                    <template #icon><IconUpload /></template>
                    上传 Logo
                  </a-button>
                  <a-button v-if="settings.logo_url" size="small" type="text" @click="settings.logo_url = ''">
                    移除
                  </a-button>
                </div>
                <input
                  ref="logoInput"
                  class="site-hidden-input"
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  @change="uploadAsset('logo', $event)"
                />
              </div>
            </article>

            <article class="site-asset">
              <div class="site-asset-preview is-icon" :class="{ 'is-empty': !settings.favicon_url }">
                <img :src="settings.favicon_url || '/favicon.webp'" alt="站点图标预览" />
              </div>
              <div class="site-asset-body">
                <strong>站点图标 / Favicon</strong>
                <p>用于浏览器标签页、PWA 与手机桌面图标。</p>
                <div class="site-asset-actions">
                  <a-button size="small" :loading="uploading === 'favicon'" @click="faviconInput?.click()">
                    <template #icon><IconUpload /></template>
                    上传图标
                  </a-button>
                  <a-button v-if="settings.favicon_url && settings.favicon_url !== '/favicon.webp'" size="small" type="text" @click="settings.favicon_url = '/favicon.webp'">
                    恢复默认
                  </a-button>
                </div>
                <input
                  ref="faviconInput"
                  class="site-hidden-input"
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  @change="uploadAsset('favicon', $event)"
                />
              </div>
            </article>
          </div>
        </section>

        <section class="adm-card site-card" aria-labelledby="site-friends-title">
          <div class="site-card-head">
            <div>
              <h3 id="site-friends-title" class="site-card-title">友情链接</h3>
              <p class="site-card-desc">最多 12 条，显示在侧边栏底部；图标可填写站内路径或 HTTP(S) 图片地址。</p>
            </div>
            <a-button size="small" :disabled="settings.friend_links.length >= 12" @click="addFriend">
              <template #icon><IconPlus /></template>
              添加链接
            </a-button>
          </div>
          <div v-if="settings.friend_links.length" class="site-friends">
            <div v-for="(link, index) in settings.friend_links" :key="index" class="site-friend-row">
              <a-input v-model="link.name" :maxlength="40" placeholder="站点名称" />
              <a-input v-model="link.url" :placeholder="friendLinkPlaceholder" />
              <a-input v-model="link.icon_url" placeholder="图标地址（可选）" />
              <a-button type="text" status="danger" aria-label="删除这条友链" @click="removeFriend(index)">
                <template #icon><IconDelete /></template>
              </a-button>
            </div>
          </div>
          <p v-else class="site-empty">还没有友情链接，点击「添加链接」创建第一条。</p>
        </section>

        <section class="adm-card site-card" aria-labelledby="site-text-title">
          <div class="site-card-head">
            <div>
              <h3 id="site-text-title" class="site-card-title">文本元素</h3>
              <p class="site-card-desc">按分组编辑前台文案；支持 {name}、{short_name}、{tagline} 占位符，留空或恢复默认即使用内置文案。</p>
            </div>
            <a-button size="small" type="text" @click="resetAllTexts">全部恢复默认</a-button>
          </div>
          <div class="site-text-tools">
            <a-input v-model="textFilter" allow-clear size="small" placeholder="搜索文案元素、分组或 key" aria-label="搜索文案元素" />
            <span v-if="textFilter.trim()" class="site-text-count">匹配 {{ visibleTextCount }} 项</span>
          </div>
          <a-form layout="vertical" :model="settings" class="site-text-groups">
            <section v-for="group in groups" :key="group.label" class="site-text-group">
              <h4>{{ group.label }}</h4>
              <div class="site-grid">
                <a-form-item
                  v-for="element in group.items"
                  :key="element.key"
                  :label="element.label"
                  :class="{ 'is-full': element.multiline }"
                >
                  <div class="site-text-control">
                    <a-textarea
                      v-if="element.multiline"
                      v-model="settings.texts[element.key]"
                      :maxlength="element.max_length"
                      :auto-size="{ minRows: 2, maxRows: 5 }"
                      show-word-limit
                    />
                    <a-input v-else v-model="settings.texts[element.key]" :maxlength="element.max_length" />
                    <span class="site-field-hint">
                      <code>{{ element.key }}</code>
                      <button type="button" @click="resetText(element)">恢复默认</button>
                    </span>
                  </div>
                </a-form-item>
              </div>
            </section>
          </a-form>
        </section>
      </div>

      <aside class="adm-board-aside site-aside">
        <section class="site-preview" aria-label="站点外观实时预览">
          <div class="site-preview-chrome" aria-hidden="true">
            <span></span><span></span><span></span>
            <i>PREVIEW</i>
          </div>
          <div class="site-preview-tab">
            <img :src="settings.favicon_url || '/favicon.webp'" alt="" width="16" height="16" />
            <span>{{ preview(settings.browser_title) }}</span>
          </div>
          <div class="site-preview-body">
            <div class="site-preview-brand">
              <img v-if="settings.logo_url" :src="settings.logo_url" alt="" width="34" height="34" />
              <div>
                <strong>{{ settings.name }}</strong>
                <span v-if="settings.tagline">{{ settings.tagline }}</span>
              </div>
            </div>
            <div class="site-preview-lines">
              <small>{{ previewText('home.featuredLabel') }}</small>
              <h4>{{ previewText('home.featuredTitle') }}</h4>
              <p>{{ previewText('login.storyText') }}</p>
            </div>
            <div class="site-preview-footer">{{ previewText('footer.copyright') }}</div>
          </div>
        </section>
        <p class="site-aside-hint">预览随输入实时更新；点击「保存更改」后对访客生效。</p>
      </aside>
    </div>
  </AdminPanel>
</template>

<style scoped>
.site-error { margin: 0 var(--adm-gap-xl, 24px); }

.site-card { padding: 20px var(--adm-gap-xl, 24px); }
.site-card + .site-card { margin-top: var(--adm-gap-lg, 16px); }
.site-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--admin-border-soft);
}
.site-card-title { margin: 0; color: var(--adm-ink); font: 600 16px/1.3 var(--site-title-font); }
.site-card-desc { margin: 5px 0 0; color: var(--adm-muted); font-size: 12px; line-height: 1.7; }
.site-form { padding-top: 16px; }
.site-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; padding-top: 16px; }
.site-grid > .is-full { grid-column: 1 / -1; }
.site-field-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
  margin-top: 5px;
  color: var(--adm-muted);
  font-size: 11px;
}
.site-text-control { display: grid; width: 100%; min-width: 0; }
.site-text-group :deep(.arco-form-item) { width: 100%; }
.site-text-group :deep(.arco-form-item-content) { flex: 1 1 auto; width: 100%; min-width: 0; }
.site-text-control :deep(.arco-input-wrapper),
.site-text-control :deep(.arco-textarea-wrapper) { width: 100%; }
.site-field-hint code { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.site-field-hint button {
  flex: 0 0 auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--adm-accent);
  font: inherit;
  cursor: pointer;
}
.site-field-hint button:hover { text-decoration: underline; }

.site-assets { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; padding-top: 16px; }
.site-asset {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--admin-border-soft);
  border-radius: 14px;
  background: color-mix(in srgb, var(--adm-soft) 45%, var(--adm-surface));
}
.site-asset-preview {
  display: grid;
  flex: 0 0 84px;
  place-items: center;
  width: 84px;
  height: 72px;
  overflow: hidden;
  padding: 8px;
  border: 1px dashed var(--admin-border-soft);
  border-radius: 10px;
  background: var(--adm-surface);
  color: var(--adm-muted);
  font-size: 11px;
}
.site-asset-preview img { display: block; max-width: 100%; max-height: 100%; object-fit: contain; }
.site-asset-preview.is-icon { width: 72px; height: 72px; }
.site-asset-preview.is-icon img { width: 48px; height: 48px; object-fit: contain; }
.site-asset-preview.is-empty { font-size: 10px; }
.site-asset-body { min-width: 0; }
.site-asset-body strong { display: block; color: var(--adm-ink); font-size: 13px; }
.site-asset-body p { margin: 4px 0 9px; color: var(--adm-muted); font-size: 11px; line-height: 1.6; }
.site-asset-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; }
.site-hidden-input { display: none; }

.site-friends { display: grid; gap: 10px; padding-top: 16px; }
.site-friend-row { display: grid; grid-template-columns: 1fr 1.5fr 1.5fr auto; gap: 8px; align-items: center; }
.site-empty { margin: 0; padding: 20px 0 6px; color: var(--adm-muted); font-size: 12px; text-align: center; }

.site-text-groups { display: grid; gap: 8px; }
.site-text-tools { display: flex; align-items: center; gap: 10px; padding-top: 14px; }
.site-text-tools :deep(.arco-input-wrapper) { max-width: 340px; }
.site-text-count { color: var(--adm-muted); font-size: 11px; white-space: nowrap; }
.site-text-group { padding-top: 16px; }
.site-text-group + .site-text-group { border-top: 1px solid var(--admin-border-soft); }
.site-text-group h4 {
  margin: 0;
  color: var(--adm-accent);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: .08em;
}
.site-text-group .site-grid { padding-top: 10px; }

.site-aside { position: sticky; top: 16px; }
.site-preview {
  overflow: hidden;
  border: 1px solid var(--admin-border-soft);
  border-radius: 14px;
  background: var(--adm-surface);
  box-shadow: var(--admin-shadow);
}
.site-preview-chrome {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 9px 12px;
  border-bottom: 1px solid var(--admin-border-soft);
  background: var(--adm-soft);
}
.site-preview-chrome span { width: 7px; height: 7px; border-radius: 50%; background: color-mix(in srgb, var(--adm-muted) 45%, transparent); }
.site-preview-chrome i {
  margin-left: auto;
  color: var(--adm-muted);
  font-size: 8px;
  font-style: normal;
  letter-spacing: .18em;
}
.site-preview-tab {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  padding: 8px 12px;
  border-bottom: 1px solid var(--admin-border-soft);
  color: var(--adm-muted);
  font-size: 10px;
}
.site-preview-tab img { flex: 0 0 auto; object-fit: contain; }
.site-preview-tab span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.site-preview-body { padding: 18px 16px 14px; }
.site-preview-brand { display: flex; align-items: center; gap: 9px; min-width: 0; }
.site-preview-brand img { flex: 0 0 auto; object-fit: contain; border-radius: 6px; }
.site-preview-brand > div { display: flex; flex-direction: column; min-width: 0; }
.site-preview-brand strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--adm-ink); font: 600 17px/1.2 var(--site-title-font); }
.site-preview-brand span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-top: 3px; color: var(--adm-muted); font-size: 9px; letter-spacing: .12em; }
.site-preview-lines { display: grid; gap: 5px; margin-top: 26px; padding-top: 16px; border-top: 1px solid var(--admin-border-soft); }
.site-preview-lines small { color: var(--adm-muted); font-size: 8px; letter-spacing: .14em; }
.site-preview-lines h4 { margin: 0; color: var(--adm-ink); font: 600 19px/1.25 var(--site-title-font); }
.site-preview-lines p { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 3; overflow: hidden; margin: 0; color: var(--adm-muted); font-size: 10px; line-height: 1.8; white-space: pre-line; }
.site-preview-footer { margin-top: 20px; padding-top: 12px; border-top: 1px solid var(--admin-border-soft); color: var(--adm-muted); font-size: 9px; text-align: center; }
.site-aside-hint { margin: 10px 2px 0; color: var(--adm-muted); font-size: 11px; line-height: 1.7; }

@media (max-width: 1180px) {
  .site-aside { position: static; max-width: 420px; }
}
@media (max-width: 760px) {
  .site-card { padding: 16px; }
  .site-grid { grid-template-columns: minmax(0, 1fr); }
  .site-assets { grid-template-columns: minmax(0, 1fr); }
  .site-friend-row { grid-template-columns: minmax(0, 1fr); }
  .site-error { margin: 0 12px; }
}
</style>
