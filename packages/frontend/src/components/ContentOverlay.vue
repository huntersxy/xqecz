<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, computed, watch, nextTick } from 'vue'
import { getImageUrl, formatTime, renderMarkdown } from '@/utils'
import type { Content } from '@/types'
import Viewer from 'viewerjs'
import 'viewerjs/dist/viewer.css'

/**
 * 全页面覆盖层 —— 瀑布流点击触发的"详情"展示。
 * - 不改变路由（仅 emit close 让父组件清除状态）
 * - 内部用 v-viewer 触发全屏看图
 * - 顶部 ← / ESC / 点遮罩 都能关闭
 * - 窄屏（<768px）改为上下堆叠：媒体在上，详情在下
 */

const props = defineProps<{
  content: Content
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

// 手动创建 viewer 实例（不在模板用 v-viewer 指令，避免 Teleport 下重复创建导致卡顿）
let viewerInstance: Viewer | null = null

function openViewerInline() {
  const img = document.querySelector('.co-media-image img') as HTMLImageElement | null
  if (!img) return
  // 标记原图 URL 到 dataset，viewer 通过 url 函数读取
  const originUrl = props.content.origin ? getImageUrl(props.content.origin) : ''
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

onBeforeUnmount(() => {
  if (viewerInstance) {
    viewerInstance.destroy()
    viewerInstance = null
  }
  document.removeEventListener('keydown', onKeyDown)
})
const mediaUrl = computed(() => {
  // 优先用全图（img），其次视频（video），最次外链
  if (props.content.img) return getImageUrl(props.content.img)
  if (props.content.video) return getImageUrl(props.content.video)
  return ''
})
const mediaKind = computed<'image' | 'video' | 'link' | 'text'>(() => {
  if (props.content.img) return 'image'
  if (props.content.video) return 'video'
  if (props.content.url) return 'link'
  return 'text'
})
const isDownloadable = computed(() => mediaKind.value === 'image' || mediaKind.value === 'video')

// 当原始 type 是 image/video 但没有实际媒体 URL 时，生成缺失原因
const noMediaReason = computed(() => {
  if (mediaKind.value !== 'text') return ''
  const t = props.content.type
  if (t === 'image' || t === 'video') {
    // 原文件丢失 or worker 还没处理完
    if (!props.content.img && !props.content.video) return '原文件丢失或未生成，等待后台处理中...'
  }
  return ''
})

// ── 文本渲染 ──
const renderedText = computed(() => {
  const t = props.content.text
  return t ? renderMarkdown(t) : ''
})

// ── 提取 text 中的参考图（markdown 格式 ![alt](url)）──
const refImages = computed(() => {
  const t = props.content.text
  if (!t) return [] as { alt: string; url: string }[]
  const re = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g
  const out: { alt: string; url: string }[] = []
  let m: RegExpExecArray | null
  while ((m = re.exec(t)) !== null) {
    out.push({ alt: m[1] || '参考图', url: m[2] })
  }
  return out
})

// ── "生成参数"占位：从 tags 识别特殊 token，如 model:SDXL / size:1024x1024 / ratio:1:1 ──
const genParams = computed(() => {
  const list: { label: string; value: string }[] = []
  for (const tag of props.content.tags || []) {
    const m = /^([a-zA-Z_]+):(.+)$/.exec(tag)
    if (m) list.push({ label: m[1].toUpperCase(), value: m[2] })
  }
  return list
})

// ── 交互占位：点赞 / 收藏 / 分享 / 下载（后续接 API）──
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
      await navigator.share({ title: props.content.title, url })
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
  a.download = props.content.title || 'download'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

function openExternalLink() {
  if (mediaKind.value === 'link' && props.content.url) {
    globalThis.open(props.content.url, '_blank', 'noopener')
  }
}

async function copyPrompt() {
  if (!props.content.text) return
  try {
    await navigator.clipboard?.writeText(props.content.text)
  } catch {
    /* 剪贴板权限被拒或非安全上下文，忽略 */
  }
}

// ── 关闭交互 ──
function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

function onBackdropClick() {
  emit('close')
}

onMounted(() => {
  document.addEventListener('keydown', onKeyDown)
  // 锁滚动 + 补偿滚动条宽度，避免 body 宽度跳变
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
  document.removeEventListener('keydown', onKeyDown)
})

// 切换内容时重置 like/fav
watch(() => props.content.id, () => {
  likeCount.value = 0
  isLiked.value = false
  isFavorited.value = false
})
</script>

<template>
  <Teleport to="body">
    <div class="co-root" :key="content.id" role="dialog" aria-modal="true" :aria-label="content.title">
      <!-- 遮罩 -->
      <div class="co-backdrop" @click="onBackdropClick"></div>

      <div class="co-shell" @click.self="onBackdropClick">
        <!-- 顶部条 -->
        <header class="co-topbar">
          <div class="co-topbar-title">
            <h2 class="co-title">{{ content.title }}</h2>
            <div class="co-meta">
              <span class="co-meta-item">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>
                {{ formatTime(content.created_at) }}
              </span>
              <span class="co-meta-item">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>
                {{ content.view_count }} 浏览
              </span>
            </div>
          </div>
          <button
            class="co-close-btn"
            type="button"
            aria-label="关闭"
            @click="emit('close')"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </header>

        <!-- 主体 -->
        <div class="co-body">
          <!-- 左侧：媒体 -->
          <section class="co-media-wrap">
            <div
              v-if="mediaKind === 'image'"
              class="co-media-image"
              @click="openViewerInline"
            >
              <img
                :src="mediaUrl"
                :alt="content.title"
                class="co-image"
                draggable="false"
              />
            </div>
            <video
              v-else-if="mediaKind === 'video'"
              :src="mediaUrl"
              controls
              playsinline
              class="co-video"
            >
              您的浏览器不支持视频播放。
            </video>
            <div v-else-if="mediaKind === 'link'" class="co-link-card">
              <div class="co-link-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                  <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                </svg>
              </div>
              <div class="co-link-text">
                <div class="co-link-label">外部链接</div>
                <div class="co-link-url">{{ content.url }}</div>
              </div>
              <button class="co-link-open" type="button" @click="openExternalLink">
                打开
              </button>
            </div>
            <div v-else class="co-text-only">
              <p v-if="noMediaReason" class="co-no-media-reason">{{ noMediaReason }}</p>
              <p v-else-if="content.text" class="co-text-content" v-html="renderedText"></p>
              <p v-else>{{ content.title }}</p>
            </div>
          </section>

          <!-- 右侧：详情 -->
          <aside class="co-side">
            <!-- 作者卡 -->
            <div v-if="content.user" class="co-author">
              <div class="co-author-left">
                <img
                  v-if="content.avatar_url"
                  :src="content.avatar_url"
                  class="co-avatar-img"
                  :alt="content.user.username"
                />
                <div v-else class="co-avatar">
                  {{ (content.user.username || '?').slice(0, 1).toUpperCase() }}
                </div>
                <div class="co-author-info">
                  <span class="co-author-name">{{ content.user.username }}</span>
                  <span class="co-author-id">ID #{{ content.user.id }}</span>
                </div>
              </div>
              <button class="co-follow-btn" type="button">+ 关注</button>
            </div>

            <!-- 说明 / tags -->
            <div v-if="(content.tags || []).filter(t => !/^[a-zA-Z_]+:.+$/.test(t)).length > 0" class="co-section">
              <div class="co-section-head">
                <span class="co-section-title">标签</span>
              </div>
              <div class="co-tag-list">
                <span
                  v-for="tag in (content.tags || []).filter(t => !/^[a-zA-Z_]+:.+$/.test(t))"
                  :key="tag"
                  class="co-tag"
                >{{ tag }}</span>
              </div>
            </div>

            <!-- 提示词 / 正文 -->
            <div v-if="content.text" class="co-section">
              <div class="co-section-head">
                <span class="co-section-title">简介</span>
                <button class="co-copy-btn" type="button" @click="copyPrompt">
                  复制
                </button>
              </div>
              <div class="co-prompt" v-html="renderedText"></div>
            </div>

            <!-- 参考图 -->
            <div v-if="refImages.length > 0" class="co-section">
              <div class="co-section-head">
                <span class="co-section-title">参考图片</span>
              </div>
              <div class="co-ref-grid">
                <a
                  v-for="(img, i) in refImages"
                  :key="i"
                  :href="img.url"
                  target="_blank"
                  rel="noopener"
                  class="co-ref-thumb"
                >
                  <img :src="img.url" :alt="img.alt" loading="lazy" />
                </a>
              </div>
            </div>

            <!-- 生成参数 -->
            <div v-if="genParams.length > 0" class="co-section">
              <div class="co-section-head">
                <span class="co-section-title">生成参数</span>
              </div>
              <div class="co-gen-params">
                <div v-for="p in genParams" :key="p.label" class="co-gen-chip">
                  <span class="co-gen-label">{{ p.label }}</span>
                  <span class="co-gen-value">{{ p.value }}</span>
                </div>
              </div>
            </div>
          </aside>
        </div>

        <!-- 底部交互栏 -->
        <footer class="co-bottombar">
          <button
            :class="['co-action', { active: isLiked }]"
            type="button"
            @click="toggleLike"
          >
            <svg viewBox="0 0 24 24" :fill="isLiked ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path></svg>
            <span>{{ likeCount || '点赞' }}</span>
          </button>
          <button
            :class="['co-action', { active: isFavorited }]"
            type="button"
            @click="toggleFavorite"
          >
            <svg viewBox="0 0 24 24" :fill="isFavorited ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg>
            <span>收藏</span>
          </button>
          <button class="co-action" type="button" @click="shareContent">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg>
            <span>分享</span>
          </button>
          <button
            v-if="isDownloadable"
            class="co-action"
            type="button"
            @click="downloadMedia"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
            <span>下载</span>
          </button>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.co-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--theme-text);
  animation: co-fade-in 0.15s ease-out;
}

.co-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(8, 4, 18, 0.92);
}

.co-shell {
  position: relative;
  width: 100vw;
  height: 100vh;
  height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--theme-surface);
  overflow: hidden;
}

@keyframes co-fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* ── 顶部条 ── */
.co-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--theme-card-border);
  background: var(--theme-header-bg);
}

.co-close-btn {
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
.co-close-btn:hover {
  background: var(--theme-hover-bg);
  color: var(--theme-text);
}
.co-close-btn svg {
  width: 18px;
  height: 18px;
}

.co-topbar-title {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.co-title {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.3;
  margin: 0;
  color: var(--theme-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.co-meta {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  font-size: 0.75rem;
  color: var(--theme-text-secondary);
  margin-top: 2px;
}

.co-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.co-meta-item svg {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
}

/* ── 主体 ── */
.co-body {
  flex: 1;
  min-height: 0;
  display: flex;
  gap: 0;
}

/* 左侧媒体 */
.co-media-wrap {
  flex: 1 1 60%;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--theme-bg-color);
  padding: 1rem;
  overflow: hidden;
}

.co-media-image {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: zoom-in;
}
.co-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 6px;
  user-select: none;
}

.co-video {
  width: 100%;
  max-height: 100%;
  border-radius: 8px;
  background: #000;
}

.co-link-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  width: min(100%, 560px);
  padding: 1.25rem 1.5rem;
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  border-radius: 12px;
}
.co-link-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--theme-hover-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--theme-primary);
  flex-shrink: 0;
}
.co-link-icon svg { width: 22px; height: 22px; }
.co-link-text { flex: 1; min-width: 0; }
.co-link-label { font-size: 0.875rem; font-weight: 600; color: var(--theme-text); margin-bottom: 4px; }
.co-link-url {
  font-size: 0.8125rem;
  color: var(--theme-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.co-link-open {
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--theme-on-primary);
  background: var(--theme-primary);
  border: none;
  border-radius: 6px;
  cursor: pointer;
  flex-shrink: 0;
}
.co-link-open:hover { filter: brightness(0.92); }

.co-text-only {
  padding: 2rem;
  text-align: center;
  font-size: 1.125rem;
  color: var(--theme-text-secondary);
  max-width: 600px;
}

.co-no-media-reason {
  padding: 3rem 1rem;
  font-size: 0.875rem;
  color: var(--theme-text-tertiary);
  background: var(--theme-placeholder-bg);
  border-radius: 0.5rem;
  border: 1px dashed var(--theme-card-border);
}

.co-text-content {
  text-align: left;
  line-height: 1.7;
}

/* 右侧详情 */
.co-side {
  flex: 0 0 40%;
  max-width: 420px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--theme-header-bg);
  border-left: 1px solid var(--theme-card-border);
  overflow-y: scroll;
  scrollbar-gutter: stable;
}

.co-author {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  border-radius: 10px;
}
.co-author-left {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  min-width: 0;
}
.co-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--theme-primary), var(--theme-secondary));
  color: var(--theme-on-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
  font-weight: 700;
  flex-shrink: 0;
}
.co-avatar-img {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.co-author-info { display: flex; flex-direction: column; min-width: 0; }
.co-author-name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--theme-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.co-author-id {
  font-size: 0.6875rem;
  color: var(--theme-text-secondary);
}

.co-follow-btn {
  padding: 0.375rem 0.875rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--theme-on-primary);
  background: var(--theme-primary);
  border: none;
  border-radius: 999px;
  cursor: pointer;
  flex-shrink: 0;
  transition: filter 0.2s;
}
.co-follow-btn:hover { filter: brightness(0.92); }

.co-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  border-radius: 10px;
}

.co-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.co-section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--theme-text-secondary);
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.co-copy-btn {
  background: transparent;
  border: 1px solid var(--theme-card-border);
  color: var(--theme-text-secondary);
  font-size: 0.6875rem;
  padding: 2px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}
.co-copy-btn:hover {
  border-color: var(--theme-primary);
  color: var(--theme-primary);
}

.co-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}
.co-tag {
  display: inline-block;
  padding: 0.1875rem 0.625rem;
  font-size: 0.75rem;
  color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--theme-primary) 25%, transparent);
  border-radius: 999px;
}

.co-prompt {
  font-size: 0.8125rem;
  line-height: 1.65;
  color: var(--theme-text);
  max-height: 220px;
  overflow-y: auto;
  word-break: break-word;
}
.co-prompt :deep(p) { margin: 0 0 0.5em; }
.co-prompt :deep(p:last-child) { margin-bottom: 0; }
.co-prompt :deep(pre) {
  background: var(--theme-hover-bg);
  padding: 0.5rem 0.625rem;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 0.75rem;
}
.co-prompt :deep(code) {
  background: var(--theme-hover-bg);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 0.75em;
}
.co-prompt :deep(ul), .co-prompt :deep(ol) { padding-left: 1.25em; margin: 0.25em 0; }
.co-prompt :deep(a) { color: var(--theme-primary); }

.co-ref-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
  gap: 0.375rem;
}
.co-ref-thumb {
  aspect-ratio: 1;
  border-radius: 6px;
  overflow: hidden;
  background: var(--theme-placeholder-bg);
  display: block;
}
.co-ref-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.2s;
}
.co-ref-thumb:hover img { transform: scale(1.06); }

.co-gen-params {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}
.co-gen-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0.1875rem 0.5rem;
  font-size: 0.6875rem;
  background: var(--theme-hover-bg);
  border: 1px solid var(--theme-card-border);
  border-radius: 4px;
}
.co-gen-label {
  color: var(--theme-text-secondary);
  font-weight: 600;
  letter-spacing: 0.04em;
}
.co-gen-value {
  color: var(--theme-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

/* ── 底部交互栏 ── */
.co-bottombar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  border-top: 1px solid var(--theme-card-border);
  background: var(--theme-header-bg);
}

.co-action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0.5rem 0.875rem;
  font-size: 0.8125rem;
  color: var(--theme-text-secondary);
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s;
}
.co-action svg { width: 16px; height: 16px; }
.co-action:hover {
  color: var(--theme-primary);
  border-color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 8%, transparent);
}
.co-action.active {
  color: var(--theme-primary);
  border-color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 14%, transparent);
}

/* ── 窄屏（<768px）上下堆叠 ── */
@media (max-width: 768px) {
  .co-body {
    flex-direction: column;
    overflow-y: auto;
  }
  .co-media-wrap {
    flex: 0 0 auto;
    height: 50vh;
    min-height: 240px;
    padding: 0.5rem;
  }
  .co-side {
    flex: 1 1 auto;
    max-width: none;
    width: 100%;
    border-left: none;
    border-top: 1px solid var(--theme-card-border);
    padding: 0.75rem;
  }
  .co-prompt {
    max-height: 160px;
  }
  .co-bottombar {
    padding: 0.5rem 0.75rem;
    gap: 0.375rem;
  }
  .co-action {
    padding: 0.4375rem 0.625rem;
    font-size: 0.75rem;
  }
}
</style>
