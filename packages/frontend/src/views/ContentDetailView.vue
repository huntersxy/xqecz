<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { useContentBrowse } from '@/composables/useContentBrowse'
import { formatTime } from '@/utils'
import MediaImage from '@/components/MediaImage.vue'
import SketchPlanet from '@/components/SketchPlanet.vue'
import { IconArrowLeft, IconCalendar, IconHeart, IconLeft, IconRight, IconRefresh } from '@arco-design/web-vue/es/icon'
import ContentMedia from '@/components/ContentMedia.vue'
import ContentSidebar from '@/components/ContentSidebar.vue'
import ContentActions from '@/components/ContentActions.vue'
import ReportModal from '@/components/ReportModal.vue'
import ClaimModal from '@/components/ClaimModal.vue'
import type { Content, Comment } from '@/types'

const route = useRoute()
const userStore = useUserStore()
const browse = useContentBrowse()
const {
  cachedList, currentId, currentIndex, hasPrev, hasNext, previewItems, previewRef,
  scrollThumbIntoView, navigateTo, goToPrev, goToNext, goBack,
} = browse

const content = ref<Content | null>(null)
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
let contentLoadSeq = 0
const reportTarget = ref<Comment | null>(null)
const showClaimModal = ref(false)

/** 列表缓存里同 id 的条目：切作品时先用它顶上，不等接口。 */
function cachedItem(id: number): Content | null {
  return cachedList.value.find((item) => item.id === id) || null
}

/**
 * 预取整图。走马灯点击前先在悬停/聚焦时把图拉进浏览器缓存，
 * 点下去就是解码已完成的图，不会先闪一下骨架。
 */
const preloaded = new Set<string>()
function preloadWork(item?: Content | null) {
  if (!item) return
  const url = item.img || item.thumb
  if (!url || preloaded.has(url)) return
  preloaded.add(url)
  const img = new Image()
  img.decoding = 'async'
  img.src = url
}
function preloadNeighbours() {
  const idx = currentIndex.value
  if (idx < 0) return
  preloadWork(cachedList.value[idx - 1])
  preloadWork(cachedList.value[idx + 1])
}

// ── 加载内容 ──
// showLoading=false 表示手里已经有可显示的内容（列表缓存），这次只是补全详情，
// 静默替换即可，绝不能把整屏切回骨架。
async function loadContent(showLoading = true) {
  const seq = ++contentLoadSeq
  const id = currentId.value
  if (showLoading) loadState.value = 'loading'
  try {
    const res = await contentApi.detail(id)
    if (seq !== contentLoadSeq || id !== currentId.value) return
    if (res.code === 200) {
      content.value = res.data
      loadState.value = 'ready'
      preloadNeighbours()
    } else {
      if (showLoading) {
        loadState.value = 'error'
        Message.error(res.message)
      }
    }
  } catch {
    if (seq !== contentLoadSeq || id !== currentId.value) return
    if (showLoading) {
      loadState.value = 'error'
      Message.error('加载内容失败')
    }
  }
}

// 路由参数变化（走马灯点击、上一/下一个、浏览器导航）：
// 先用列表缓存把标题/作者/媒体原地换掉，浏览器缓存里的整图立刻能画；
// 详情请求回来后再静默补全（评论、精确计数等）。整个过程不重建组件。
watch(() => route.params.id, () => {
  const id = currentId.value
  const hit = cachedItem(id)
  if (hit) {
    content.value = hit
    loadState.value = 'ready'
  }
  void loadContent(!hit)
  scrollThumbIntoView()
})

onMounted(() => {
  userStore.checkAuth()
  loadContent()
  // 背景滚动锁定改由 main.css 的 `html:has(.cd-root)` 声明式接管：
  // 覆盖层挂载即锁、卸载即解，无需存旧值再恢复。
})
</script>

<template>
  <div class="cd-root creative-theme">
    <!-- 遮罩 -->
    <div class="cd-backdrop" @click="goBack"></div>

    <div class="cd-shell" @click.self="goBack">
      <!-- 顶部条 -->
      <header class="cd-topbar">
        <button class="cd-back-btn" type="button" aria-label="返回" @click="goBack">
          <IconArrowLeft />
          <span>返回</span>
        </button>
        <div class="cd-topbar-title">
          <span class="cd-eyebrow">作品详情 <span aria-hidden="true">/</span> WORK</span>
          <h2 class="cd-title">{{ content?.title || '正在穿越小泉宇宙…' }}</h2>
          <div v-if="content" class="cd-meta">
            <span class="cd-meta-item">
              <IconCalendar />
              {{ formatTime(content.created_at) }}
            </span>
            <span class="cd-meta-item">
              <IconHeart />
              {{ content.like_count || 0 }} 点赞
            </span>
            <span
              v-if="content.ai_enhanced"
              class="cd-meta-item cd-meta-ai"
              title="发布时启用了提示词增强"
            >
              <span class="cd-meta-ai-dot" aria-hidden="true"></span>
              提示词增强
            </span>
            <span v-if="currentIndex >= 0" class="cd-meta-item cd-meta-index">
              {{ currentIndex + 1 }} / {{ cachedList.length }}
            </span>
          </div>
        </div>
        <span class="cd-topbar-note">来自小泉作品广场</span>
      </header>

      <!-- 加载中 -->
      <div v-if="loadState === 'loading'" class="cd-loading" role="status">
        <SketchPlanet class="cd-loading-planet" loading />
        <p>正在穿越小泉宇宙…</p>
      </div>

      <!-- 加载失败 -->
      <div v-else-if="loadState === 'error'" class="cd-loading">
        <a-result
          status="error"
          title="信号暂时离开了轨道"
          subtitle="内容不存在或网络异常，请稍后再试"
        >
          <template #extra>
            <a-button type="primary" @click="loadContent()">
              <IconRefresh />
              重试
            </a-button>
          </template>
        </a-result>
      </div>

      <!-- 主体 -->
      <div v-else-if="content" class="cd-body">
        <div class="cd-main">
          <ContentMedia :content="content" />
          <ContentSidebar :content="content" @open-claim="showClaimModal = true" @report-comment="reportTarget = $event" />
        </div>
      </div>

      <!-- 底部预览走马灯（内容不足一行时居中） -->
      <div
        v-if="content && previewItems.length > 0"
        ref="previewRef"
        class="cd-carousel"
      >
        <div class="cd-carousel-group">
          <span class="cd-carousel-label" aria-hidden="true">继续看</span>
          <button
            v-if="hasPrev"
            class="cd-carousel-nav cd-carousel-prev"
            type="button"
            aria-label="上一个"
            @click="goToPrev"
          >
            <IconLeft />
          </button>

          <div class="cd-carousel-track">
            <button
              v-for="item in previewItems"
              :key="item.id"
              :class="['cd-carousel-thumb', { 'cd-thumb-current': item.isCurrent }]"
              :aria-label="item.title || '查看作品'"
              :aria-current="item.isCurrent ? 'true' : undefined"
              @pointerenter="preloadWork(item)"
              @focus="preloadWork(item)"
              @click="navigateTo(item.id)"
            >
              <MediaImage :src="item.thumb" :alt="item.title" :preview="false" loading="lazy" />
            </button>
          </div>

          <button
            v-if="hasNext"
            class="cd-carousel-nav cd-carousel-next"
            type="button"
            aria-label="下一个"
            @click="goToNext"
          >
            <IconRight />
          </button>
        </div>
      </div>

      <!-- 底部交互栏 -->
      <ContentActions v-if="content" :content="content" />
    </div>

    <!-- 举报弹窗 -->
    <ReportModal :target="reportTarget" @close="reportTarget = null" />
    <!-- 认领弹窗 -->
    <ClaimModal :open="showClaimModal" :content-id="Number(content?.id) || 0" @close="showClaimModal = false" />
  </div>
</template>

<style scoped>
/* The detail view is a quiet reading surface over the gallery canvas. Keep the
   overlay mounted for keyboard/backdrop behaviour, but let the content panel
   carry the paper, line and serif hierarchy used by the home gallery.
   --creative-* 已统一为 --site-* 别名，无需在此重声明。 */
.cd-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: clamp(10px, 2vw, 28px);
  color: var(--creative-ink);
  background: var(--creative-canvas);
}
.cd-backdrop { position: absolute; inset: 0; background: color-mix(in srgb, var(--creative-ink) 5%, transparent); }
.cd-shell {
  position: relative;
  width: 100%;
  max-width: 1560px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--creative-paper);
  border: 1px solid var(--creative-line);
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 28px 64px rgba(48, 36, 39, .16), 0 2px 10px rgba(48, 36, 39, .06);
  animation: cd-fade-in .2s ease-out;
}
@keyframes cd-fade-in { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }
.cd-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 18px;
  min-height: 78px;
  padding: 14px clamp(18px, 3vw, 34px);
  border-bottom: 1px solid var(--creative-line);
  background: var(--creative-paper);
  position: relative;
  z-index: 2;
}
.cd-back-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-width: 38px;
  height: 36px;
  padding: 0 12px 0 9px;
  border: 1px solid var(--creative-line);
  border-radius: 9px;
  background: transparent;
  color: var(--creative-muted);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  flex-shrink: 0;
  transition: color .18s, border-color .18s, background .18s;
}
.cd-back-btn:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-soft); }
.cd-back-btn:active { transform: translateY(1px); }
.cd-back-btn svg { width: 16px; height: 16px; }
.cd-topbar-title { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.cd-eyebrow { margin-bottom: 6px; color: var(--creative-muted); font-size: 9px; letter-spacing: .14em; text-transform: uppercase; }
.cd-eyebrow > span { margin-left: 5px; color: var(--creative-accent); }
.cd-title { margin: 0; color: var(--creative-ink); font: 600 clamp(19px, 1.9vw, 26px)/1.18 var(--creative-title-font); letter-spacing: -.025em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin-top: 7px; color: var(--creative-muted); font-size: 10px; }
.cd-meta-item { display: inline-flex; align-items: center; gap: 4px; padding-right: 10px; border-right: 1px solid var(--creative-line); }
.cd-meta-item:last-child { padding-right: 0; border-right: 0; }
.cd-meta-ai { padding: 1px 8px; border-right: 0; }
.cd-meta-index { border-right: 0; }
.cd-meta-item svg { width: 12px; height: 12px; flex-shrink: 0; }
.cd-meta-index { color: var(--creative-accent); font-variant-numeric: tabular-nums; }
.cd-meta-ai {
  gap: 5px;
  padding: 1px 8px;
  border: 1px solid var(--creative-line);
  border-radius: 999px;
  color: var(--creative-accent);
}
.cd-meta-ai-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.cd-topbar-note { flex-shrink: 0; color: var(--creative-muted); font-size: 10px; letter-spacing: .05em; }

.cd-body { flex: 1; min-height: 0; display: flex; scrollbar-width: thin; scrollbar-color: var(--creative-line) transparent; }
.cd-main { flex: 1; min-width: 0; display: flex; min-height: 0; }
.cd-carousel {
  position: relative;
  flex-shrink: 0; display: flex; overflow-x: auto;
  padding: 11px clamp(14px, 2.5vw, 30px);
  background: color-mix(in srgb, var(--creative-canvas) 40%, var(--creative-paper));
  border-top: 1px solid var(--creative-line);
  scrollbar-width: none;
  /* 两侧渐隐：滚动到尽头时不像被硬切掉 */
  mask-image: linear-gradient(90deg, transparent 0, #000 26px, #000 calc(100% - 26px), transparent 100%);
}
.cd-carousel-label {
  flex-shrink: 0;
  margin-right: 2px;
  color: var(--creative-muted);
  font-size: 10px;
  letter-spacing: .18em;
  text-transform: uppercase;
}
.cd-carousel::-webkit-scrollbar { display: none; }
.cd-carousel-group { display: flex; align-items: center; gap: 10px; width: max-content; margin-inline: auto; }
.cd-carousel-track { display: flex; gap: 7px; }
.cd-carousel-nav { flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border: 1px solid var(--creative-line); border-radius: 8px; background: transparent; color: var(--creative-muted); cursor: pointer; transition: background .15s, color .15s, border-color .15s; }
.cd-carousel-nav:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-soft); }
.cd-carousel-nav:active { transform: translateY(1px); }
.cd-carousel-thumb { flex-shrink: 0; width: 60px; height: 46px; border-radius: 8px; overflow: hidden; border: 1px solid transparent; cursor: pointer; padding: 0; background: var(--creative-canvas); transition: border-color .18s, opacity .18s, transform .18s; }
.cd-carousel-thumb:hover { border-color: color-mix(in srgb, var(--creative-accent) 60%, transparent); opacity: .9; transform: translateY(-1px); }
.cd-thumb-current { border-color: var(--creative-accent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--creative-accent) 22%, transparent); }
.cd-carousel-thumb :deep(.arco-image) { display: block; width: 100%; height: 100%; border-radius: 5px; overflow: hidden; }
.cd-carousel-thumb :deep(.arco-image-img) { width: 100%; height: 100%; object-fit: cover; display: block; }
.cd-loading-planet { width: 44px; height: 44px; color: var(--creative-accent); }
.cd-loading { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--creative-muted); background: var(--creative-canvas); font-size: 12px; }
@media (max-width: 1024px) { .cd-topbar-note { display: none; } }
@media (max-width: 768px) {
  .cd-root { padding: 0; }
  .cd-shell { border: 0; border-radius: 0; height: 100dvh; }
  .cd-topbar { min-height: 72px; padding: 12px 16px; gap: 11px; }
  .cd-back-btn { width: 34px; min-width: 34px; padding-inline: 8px; }
  .cd-back-btn span { display: none; }
  .cd-title { font-size: 18px; }
  .cd-eyebrow { font-size: 8px; }
  .cd-meta { gap: 10px; margin-top: 5px; }
  .cd-body { flex-direction: column; overflow-y: auto; }
  .cd-main { flex: 0 0 auto; flex-direction: column; }
  .cd-carousel { padding: 9px 12px; }
  .cd-carousel-group { gap: 8px; }
  .cd-carousel-nav { width: 26px; height: 26px; }
  .cd-carousel-thumb { width: 52px; height: 40px; }
  .cd-carousel-label { display: none; }
  :deep(.cd-media-wrap) { flex: 0 0 auto; min-height: 200px; height: 43dvh; max-height: 52dvh; padding: 14px; }
  :deep(.cd-media-wrap:has(.cd-text-only)) { height: auto; max-height: none; }
  :deep(.cd-side) { flex: 0 0 auto; width: 100%; max-width: 100%; border-left: none; border-top: 1px solid var(--creative-line); overflow-y: visible; scrollbar-gutter: auto; }
}
</style>
