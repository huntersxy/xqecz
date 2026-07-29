<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, computed, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { getImageUrl, formatTime, renderMarkdown } from '@/utils'
import CommentSections from '@/components/CommentSections.vue'
import ReportModal from '@/components/ReportModal.vue'
import ClaimModal from '@/components/ClaimModal.vue'
import Viewer from 'viewerjs'
import 'viewerjs/dist/viewer.css'
import type { Content, Comment } from '@/types'

/**
 * 全屏覆盖式详情页 —— 瀑布流点击卡片后无痕路由切换到此页。
 * - 视觉上采用 fixed 全屏覆盖 + 两栏布局
 * - 保留评论 / 认领 / 举报功能
 * - 通过 router.back() 返回瀑布流（滚动位置由 homeStore 恢复）
 */

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const content = ref<Content | null>(null)
const message = ref('')
const reportTarget = ref<Comment | null>(null)
const showClaimModal = ref(false)
const commentRef = ref<InstanceType<typeof CommentSections> | null>(null)

// ── v-viewer 实例 ──
let viewerInstance: Viewer | null = null

function openViewerInline() {
  const img = document.querySelector('.cd-media-image img') as HTMLImageElement | null
  if (!img) return
  const originUrl = content.value?.origin ? getImageUrl(content.value.origin) : ''
  if (originUrl) img.dataset.origin = originUrl
  viewerInstance = new Viewer(img, {
    navbar: false,
    zIndex: 10000,
    zIndexInline: 10000,
    url(imgEl: HTMLImageElement) {
      return (imgEl as HTMLImageElement).dataset.origin || imgEl.src
    },
    hidden() {
      viewerInstance?.destroy()
      viewerInstance = null
    },
  })
  nextTick(() => {
    viewerInstance?.show()
  })
}

// ── 媒体计算 ──
const mediaUrl = computed(() => {
  if (!content.value) return ''
  if (content.value.img) return getImageUrl(content.value.img)
  if (content.value.video) return getImageUrl(content.value.video)
  return ''
})
const mediaKind = computed<'image' | 'video' | 'link' | 'text'>(() => {
  if (!content.value) return 'text'
  if (content.value.img) return 'image'
  if (content.value.video) return 'video'
  if (content.value.url) return 'link'
  return 'text'
})
const isDownloadable = computed(() => mediaKind.value === 'image' || mediaKind.value === 'video')

const noMediaReason = computed(() => {
  if (mediaKind.value !== 'text') return ''
  const t = content.value?.type
  if (t === 'image' || t === 'video') {
    if (!content.value?.img && !content.value?.video) return '原文件丢失或未生成，等待后台处理中...'
  }
  return ''
})

// ── 文本渲染 ──
const renderedText = computed(() => {
  const t = content.value?.text
  return t ? renderMarkdown(t) : ''
})

const refImages = computed(() => {
  const t = content.value?.text
  if (!t) return [] as { alt: string; url: string }[]
  const re = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g
  const out: { alt: string; url: string }[] = []
  let m: RegExpExecArray | null
  while ((m = re.exec(t)) !== null) {
    out.push({ alt: m[1] || '参考图', url: m[2] })
  }
  return out
})

const genParams = computed(() => {
  const list: { label: string; value: string }[] = []
  for (const tag of content.value?.tags || []) {
    const m = /^([a-zA-Z_]+):(.+)$/.exec(tag)
    if (m) list.push({ label: m[1].toUpperCase(), value: m[2] })
  }
  return list
})

// ── 交互占位 ──
const likeCount = ref(0)
const isLiked = ref(false)
const isFavorited = ref(false)

function toggleLike() {
  isLiked.value = !isLiked.value
  likeCount.value += isLiked.value ? 1 : -1
}

function toggleFavorite() {
  isFavorited.value = !isFavorited.value
}

async function shareContent() {
  const url = globalThis.location.href
  try {
    if (navigator.share) {
      await navigator.share({ title: content.value?.title || '', url })
    } else if (navigator.clipboard) {
      await navigator.clipboard.writeText(url)
    }
  } catch {
    /* 用户取消分享，忽略 */
  }
}

function downloadMedia() {
  if (!isDownloadable.value || !mediaUrl.value) return
  const a = document.createElement('a')
  a.href = mediaUrl.value
  a.download = content.value?.title || 'download'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

function openExternalLink() {
  if (mediaKind.value === 'link' && content.value?.url) {
    globalThis.open(content.value.url, '_blank', 'noopener')
  }
}

async function copyPrompt() {
  if (!content.value?.text) return
  try {
    await navigator.clipboard?.writeText(content.value.text)
  } catch {
    /* 剪贴板权限被拒或非安全上下文，忽略 */
  }
}

// ── 加载内容 ──
async function loadContent() {
  try {
    const id = Number(route.params.id)
    const res = await contentApi.detail(id)
    if (res.code === 200) {
      content.value = res.data
    } else {
      message.value = res.message
    }
  } catch {
    message.value = '加载内容失败'
  }
}

// ── 关闭 ──
function goBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/')
  }
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') goBack()
}

// ── 生命周期 ──
onMounted(() => {
  userStore.checkAuth()
  loadContent()
  document.addEventListener('keydown', onKeyDown)
  const scrollBarW = window.innerWidth - document.documentElement.clientWidth
  const prevOverflow = document.body.style.overflow
  const prevPadding = document.body.style.paddingRight
  document.body.style.overflow = 'hidden'
  if (scrollBarW > 0) document.body.style.paddingRight = `${scrollBarW}px`
  onBeforeUnmount(() => {
    document.body.style.overflow = prevOverflow
    if (scrollBarW > 0) document.body.style.paddingRight = prevPadding
  })
})

onBeforeUnmount(() => {
  if (viewerInstance) {
    viewerInstance.destroy()
    viewerInstance = null
  }
  document.removeEventListener('keydown', onKeyDown)
})

// 路由参数变化时重新加载
watch(() => route.params.id, (newId) => {
  if (newId) {
    content.value = null
    likeCount.value = 0
    isLiked.value = false
    isFavorited.value = false
    loadContent()
    nextTick(() => commentRef.value?.loadComments())
  }
})
</script>

<template>
  <div class="cd-root" role="dialog" aria-modal="true">
    <!-- 遮罩 -->
    <div class="cd-backdrop" @click="goBack"></div>

    <div class="cd-shell" @click.self="goBack">
      <!-- 顶部条 -->
      <header class="cd-topbar">
        <button class="cd-back-btn" type="button" aria-label="返回" @click="goBack">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
        </button>
        <div class="cd-topbar-title">
          <h2 class="cd-title">{{ content?.title || '加载中...' }}</h2>
          <div v-if="content" class="cd-meta">
            <span class="cd-meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>
              {{ formatTime(content.created_at) }}
            </span>
            <span class="cd-meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>
              {{ content.view_count }} 浏览
            </span>
          </div>
        </div>
      </header>

      <!-- 消息条 -->
      <div v-if="message" class="cd-message" :class="{ 'cd-msg-error': message.includes('失败') || message.includes('请'), 'cd-msg-success': message.includes('成功') }">
        <span>{{ message }}</span>
        <button type="button" class="cd-msg-close" @click="message = ''">×</button>
      </div>

      <!-- 主体 -->
      <div v-if="content" class="cd-body">
        <!-- 左侧：媒体 -->
        <section class="cd-media-wrap">
          <div v-if="mediaKind === 'image'" class="cd-media-image" @click="openViewerInline">
            <img :src="mediaUrl" :alt="content.title" class="cd-image" draggable="false" />
          </div>
          <video v-else-if="mediaKind === 'video'" :src="mediaUrl" controls playsinline class="cd-video">
            您的浏览器不支持视频播放。
          </video>
          <div v-else-if="mediaKind === 'link'" class="cd-link-card">
            <div class="cd-link-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
              </svg>
            </div>
            <div class="cd-link-text">
              <div class="cd-link-label">外部链接</div>
              <div class="cd-link-url">{{ content.url }}</div>
            </div>
            <button class="cd-link-open" type="button" @click="openExternalLink">打开</button>
          </div>
          <div v-else class="cd-text-only">
            <p v-if="noMediaReason" class="cd-no-media-reason">{{ noMediaReason }}</p>
            <p v-else-if="content.text" class="cd-text-content" v-html="renderedText"></p>
            <p v-else>{{ content.title }}</p>
          </div>
        </section>

        <!-- 右侧：详情 + 评论 -->
        <aside class="cd-side">
          <!-- 作者卡 -->
          <div v-if="content.user" class="cd-author">
            <div class="cd-author-left">
              <img v-if="content.avatar_url" :src="content.avatar_url" class="cd-avatar-img" :alt="content.user.username" />
              <div v-else class="cd-avatar">{{ (content.user.username || '?').slice(0, 1).toUpperCase() }}</div>
              <div class="cd-author-info">
                <span class="cd-author-name">{{ content.user.username }}</span>
                <span class="cd-author-id">ID #{{ content.user.id }}</span>
              </div>
            </div>
            <div class="cd-author-actions">
              <button class="cd-claim-btn" type="button" @click="userStore.isLoggedIn ? (showClaimModal = true) : router.push('/login')">认领</button>
              <button class="cd-follow-btn" type="button">+ 关注</button>
            </div>
          </div>

          <!-- 标签 -->
          <div v-if="(content.tags || []).filter(t => !/^[a-zA-Z_]+:.+$/.test(t)).length > 0" class="cd-section">
            <div class="cd-section-head"><span class="cd-section-title">标签</span></div>
            <div class="cd-tag-list">
              <span v-for="tag in (content.tags || []).filter(t => !/^[a-zA-Z_]+:.+$/.test(t))" :key="tag" class="cd-tag">{{ tag }}</span>
            </div>
          </div>

          <!-- 简介 -->
          <div v-if="content.text" class="cd-section">
            <div class="cd-section-head">
              <span class="cd-section-title">简介</span>
              <button class="cd-copy-btn" type="button" @click="copyPrompt">复制</button>
            </div>
            <div class="cd-prompt" v-html="renderedText"></div>
          </div>

          <!-- 参考图 -->
          <div v-if="refImages.length > 0" class="cd-section">
            <div class="cd-section-head"><span class="cd-section-title">参考图片</span></div>
            <div class="cd-ref-grid">
              <a v-for="(img, i) in refImages" :key="i" :href="img.url" target="_blank" rel="noopener" class="cd-ref-thumb">
                <img :src="img.url" :alt="img.alt" loading="lazy" />
              </a>
            </div>
          </div>

          <!-- 生成参数 -->
          <div v-if="genParams.length > 0" class="cd-section">
            <div class="cd-section-head"><span class="cd-section-title">生成参数</span></div>
            <div class="cd-gen-params">
              <div v-for="p in genParams" :key="p.label" class="cd-gen-chip">
                <span class="cd-gen-label">{{ p.label }}</span>
                <span class="cd-gen-value">{{ p.value }}</span>
              </div>
            </div>
          </div>

          <!-- 评论 -->
          <CommentSections
            ref="commentRef"
            :content-id="content?.id || 0"
            :is-logged-in="userStore.isLoggedIn"
            @message="message = $event"
            @report-comment="reportTarget = $event"
          />
        </aside>
      </div>

      <!-- 加载中 -->
      <div v-else class="cd-loading">
        <div class="cd-spinner"></div>
        <p>加载中...</p>
      </div>

      <!-- 底部交互栏 -->
      <footer v-if="content" class="cd-bottombar">
        <button :class="['cd-action', { active: isLiked }]" type="button" @click="toggleLike">
          <svg viewBox="0 0 24 24" :fill="isLiked ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path></svg>
          <span>{{ likeCount || '点赞' }}</span>
        </button>
        <button :class="['cd-action', { active: isFavorited }]" type="button" @click="toggleFavorite">
          <svg viewBox="0 0 24 24" :fill="isFavorited ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg>
          <span>收藏</span>
        </button>
        <button class="cd-action" type="button" @click="shareContent">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg>
          <span>分享</span>
        </button>
        <button v-if="isDownloadable" class="cd-action" type="button" @click="downloadMedia">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
          <span>下载</span>
        </button>
      </footer>
    </div>

    <!-- 举报弹窗 -->
    <ReportModal :target="reportTarget" @close="reportTarget = null" @success="message = $event" />

    <!-- 认领弹窗 -->
    <ClaimModal :open="showClaimModal" :content-id="content?.id || 0" @close="showClaimModal = false" @success="message = $event" />
  </div>
</template>

<style scoped>
.cd-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--theme-text);
  animation: cd-fade-in 0.2s ease-out;
}

.cd-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(8, 4, 18, 0.92);
  backdrop-filter: blur(8px);
}

.cd-shell {
  position: relative;
  width: 100vw;
  height: 100vh;
  height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--theme-surface);
  overflow: hidden;
}

@keyframes cd-fade-in {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}

/* ── 消息条 ── */
.cd-message {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
}
.cd-msg-error { color: var(--theme-danger); background: color-mix(in srgb, var(--theme-danger) 10%, transparent); }
.cd-msg-success { color: var(--theme-success); background: color-mix(in srgb, var(--theme-success) 10%, transparent); }
.cd-msg-close { background: none; border: none; color: inherit; font-size: 1.125rem; cursor: pointer; line-height: 1; }

/* ── 顶部条 ── */
.cd-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--theme-card-border);
  background: var(--theme-header-bg);
}

.cd-back-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--theme-text-secondary);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.2s, color 0.2s;
}
.cd-back-btn:hover { background: var(--theme-hover-bg); color: var(--theme-text); }
.cd-back-btn svg { width: 18px; height: 18px; }

.cd-topbar-title { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.cd-title {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.3;
  margin: 0;
  color: var(--theme-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cd-meta {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  font-size: 0.75rem;
  color: var(--theme-text-secondary);
  margin-top: 2px;
}
.cd-meta-item { display: inline-flex; align-items: center; gap: 4px; }
.cd-meta-item svg { width: 13px; height: 13px; flex-shrink: 0; }

/* ── 主体 ── */
.cd-body { flex: 1; min-height: 0; display: flex; gap: 0; }

.cd-media-wrap {
  flex: 1 1 60%;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--theme-bg-color);
  padding: 1rem;
  overflow: hidden;
}
.cd-media-image {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: zoom-in;
}
.cd-image { max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 6px; user-select: none; }
.cd-video { width: 100%; max-height: 100%; border-radius: 8px; background: #000; }

.cd-link-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  width: min(100%, 560px);
  padding: 1.25rem 1.5rem;
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  border-radius: 12px;
}
.cd-link-icon {
  width: 48px; height: 48px; border-radius: 50%;
  background: var(--theme-hover-bg);
  display: flex; align-items: center; justify-content: center;
  color: var(--theme-primary); flex-shrink: 0;
}
.cd-link-icon svg { width: 22px; height: 22px; }
.cd-link-text { flex: 1; min-width: 0; }
.cd-link-label { font-size: 0.875rem; font-weight: 600; color: var(--theme-text); margin-bottom: 4px; }
.cd-link-url { font-size: 0.8125rem; color: var(--theme-text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-link-open {
  padding: 0.5rem 1rem; font-size: 0.8125rem; font-weight: 600;
  color: var(--theme-on-primary); background: var(--theme-primary);
  border: none; border-radius: 6px; cursor: pointer; flex-shrink: 0;
}
.cd-link-open:hover { filter: brightness(0.92); }

.cd-text-only { padding: 2rem; text-align: center; font-size: 1.125rem; color: var(--theme-text-secondary); max-width: 600px; }
.cd-no-media-reason {
  padding: 3rem 1rem; font-size: 0.875rem; color: var(--theme-text-tertiary);
  background: var(--theme-placeholder-bg); border-radius: 0.5rem; border: 1px dashed var(--theme-card-border);
}
.cd-text-content { text-align: left; line-height: 1.7; }

/* ── 右侧详情 ── */
.cd-side {
  flex: 0 0 40%;
  max-width: 460px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--theme-header-bg);
  border-left: 1px solid var(--theme-card-border);
  overflow-y: scroll;
  scrollbar-gutter: stable;
}

.cd-author {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  border-radius: 10px;
}
.cd-author-left { display: flex; align-items: center; gap: 0.625rem; min-width: 0; }
.cd-avatar {
  width: 38px; height: 38px; border-radius: 50%;
  background: linear-gradient(135deg, var(--theme-primary), var(--theme-secondary));
  color: var(--theme-on-primary);
  display: flex; align-items: center; justify-content: center;
  font-size: 1rem; font-weight: 700; flex-shrink: 0;
}
.cd-avatar-img { width: 38px; height: 38px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.cd-author-info { display: flex; flex-direction: column; min-width: 0; }
.cd-author-name {
  font-size: 0.875rem; font-weight: 600; color: var(--theme-text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.cd-author-id { font-size: 0.6875rem; color: var(--theme-text-secondary); }
.cd-author-actions { display: flex; gap: 0.375rem; flex-shrink: 0; }
.cd-claim-btn {
  padding: 0.375rem 0.625rem; font-size: 0.6875rem;
  color: var(--theme-text-secondary); background: var(--theme-hover-bg);
  border: 1px solid var(--theme-card-border); border-radius: 999px; cursor: pointer;
}
.cd-claim-btn:hover { color: var(--theme-primary); border-color: var(--theme-primary); }
.cd-follow-btn {
  padding: 0.375rem 0.875rem; font-size: 0.75rem; font-weight: 600;
  color: var(--theme-on-primary); background: var(--theme-primary);
  border: none; border-radius: 999px; cursor: pointer;
}
.cd-follow-btn:hover { filter: brightness(0.92); }

.cd-section {
  display: flex; flex-direction: column; gap: 0.5rem;
  padding: 0.75rem; background: var(--theme-surface);
  border: 1px solid var(--theme-card-border); border-radius: 10px;
}
.cd-section-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.cd-section-title {
  font-size: 0.75rem; font-weight: 600; color: var(--theme-text-secondary);
  letter-spacing: 0.05em; text-transform: uppercase;
}
.cd-copy-btn {
  background: transparent; border: 1px solid var(--theme-card-border);
  color: var(--theme-text-secondary); font-size: 0.6875rem;
  padding: 2px 8px; border-radius: 4px; cursor: pointer; transition: all 0.15s;
}
.cd-copy-btn:hover { border-color: var(--theme-primary); color: var(--theme-primary); }

.cd-tag-list { display: flex; flex-wrap: wrap; gap: 0.375rem; }
.cd-tag {
  display: inline-block; padding: 0.1875rem 0.625rem; font-size: 0.75rem;
  color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--theme-primary) 25%, transparent);
  border-radius: 999px;
}

.cd-prompt {
  font-size: 0.8125rem; line-height: 1.65; color: var(--theme-text);
  max-height: 220px; overflow-y: auto; word-break: break-word;
}
.cd-prompt :deep(p) { margin: 0 0 0.5em; }
.cd-prompt :deep(p:last-child) { margin-bottom: 0; }
.cd-prompt :deep(pre) {
  background: var(--theme-hover-bg); padding: 0.5rem 0.625rem;
  border-radius: 6px; overflow-x: auto; font-size: 0.75rem;
}
.cd-prompt :deep(code) { background: var(--theme-hover-bg); padding: 1px 4px; border-radius: 3px; font-size: 0.75em; }
.cd-prompt :deep(ul), .cd-prompt :deep(ol) { padding-left: 1.25em; margin: 0.25em 0; }
.cd-prompt :deep(a) { color: var(--theme-primary); }

.cd-ref-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: 0.375rem; }
.cd-ref-thumb { aspect-ratio: 1; border-radius: 6px; overflow: hidden; background: var(--theme-placeholder-bg); display: block; }
.cd-ref-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.2s; }
.cd-ref-thumb:hover img { transform: scale(1.06); }

.cd-gen-params { display: flex; flex-wrap: wrap; gap: 0.375rem; }
.cd-gen-chip {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 0.1875rem 0.5rem; font-size: 0.6875rem;
  background: var(--theme-hover-bg); border: 1px solid var(--theme-card-border); border-radius: 4px;
}
.cd-gen-label { color: var(--theme-text-secondary); font-weight: 600; letter-spacing: 0.04em; }
.cd-gen-value { color: var(--theme-text); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }

/* ── 加载中 ── */
.cd-loading {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 1rem;
  color: var(--theme-text-secondary);
}
.cd-spinner {
  width: 2rem; height: 2rem;
  border: 3px solid var(--theme-card-border);
  border-top-color: var(--theme-primary);
  border-radius: 50%; animation: cd-spin 0.7s linear infinite;
}
@keyframes cd-spin { to { transform: rotate(360deg); } }

/* ── 底部交互栏 ── */
.cd-bottombar {
  flex-shrink: 0; display: flex; align-items: center; justify-content: center;
  gap: 0.5rem; padding: 0.625rem 1rem;
  border-top: 1px solid var(--theme-card-border); background: var(--theme-header-bg);
}
.cd-action {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 0.5rem 0.875rem; font-size: 0.8125rem;
  color: var(--theme-text-secondary); background: var(--theme-surface);
  border: 1px solid var(--theme-card-border); border-radius: 999px;
  cursor: pointer; transition: all 0.15s;
}
.cd-action svg { width: 16px; height: 16px; }
.cd-action:hover {
  color: var(--theme-primary); border-color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 8%, transparent);
}
.cd-action.active {
  color: var(--theme-primary); border-color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 14%, transparent);
}

/* ── 窄屏 ── */
@media (max-width: 768px) {
  .cd-body { flex-direction: column; overflow-y: auto; }
  .cd-media-wrap { flex: 0 0 auto; height: 50vh; min-height: 240px; padding: 0.5rem; }
  .cd-side {
    flex: 1 1 auto; max-width: none; width: 100%;
    border-left: none; border-top: 1px solid var(--theme-card-border); padding: 0.75rem;
  }
  .cd-prompt { max-height: 160px; }
  .cd-bottombar { padding: 0.5rem 0.75rem; gap: 0.375rem; }
  .cd-action { padding: 0.4375rem 0.625rem; font-size: 0.75rem; }
}
</style>
