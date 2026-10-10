<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { useContentBrowse } from '@/composables/useContentBrowse'
import { formatTime } from '@/utils'
import MediaImage from '@/components/MediaImage.vue'
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

// ── 加载内容 ──
async function loadContent() {
  const seq = ++contentLoadSeq
  const id = currentId.value
  loadState.value = 'loading'
  try {
    const res = await contentApi.detail(id)
    if (seq !== contentLoadSeq || id !== currentId.value) return
    if (res.code === 200) {
      content.value = res.data
      loadState.value = 'ready'
    } else {
      loadState.value = 'error'
      Message.error(res.message)
    }
  } catch {
    if (seq !== contentLoadSeq || id !== currentId.value) return
    loadState.value = 'error'
    Message.error('加载内容失败')
  }
}

// 路由参数变化（上一/下一个、浏览器导航）时重新加载
watch(() => route.params.id, () => {
  loadContent()
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
          <span class="cd-eyebrow">CREATIVE ARCHIVE <span aria-hidden="true">✦</span></span>
          <h2 class="cd-title">{{ content?.title || '加载中...' }}</h2>
          <div v-if="content" class="cd-meta">
            <span class="cd-meta-item">
              <IconCalendar />
              {{ formatTime(content.created_at) }}
            </span>
            <span class="cd-meta-item">
              <IconHeart />
              {{ content.like_count || 0 }} 点赞
            </span>
            <span v-if="currentIndex >= 0" class="cd-meta-item cd-meta-index">
              {{ currentIndex + 1 }} / {{ cachedList.length }}
            </span>
          </div>
        </div>
        <span class="cd-topbar-note">每一份热爱，都有回响<span aria-hidden="true"> ✧</span></span>
      </header>

      <!-- 加载中 -->
      <div v-if="loadState === 'loading'" class="cd-loading">
        <a-spin :loading="true" :size="36" />
        <p>加载中...</p>
      </div>

      <!-- 加载失败 -->
      <div v-else-if="loadState === 'error'" class="cd-loading">
        <a-result
          status="error"
          title="加载失败"
          subtitle="内容不存在或网络异常"
        >
          <template #extra>
            <a-button type="primary" @click="loadContent">
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
/* 底色从首帧就覆盖背景，只有内容面板执行入场动画。 */
.cd-root { position: fixed; inset: 0; z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 20px; color: var(--creative-ink); background: var(--creative-canvas); }
.cd-backdrop { position: absolute; inset: 0; }
.cd-shell { position: relative; width: 100%; max-width: 1600px; height: 100%; display: flex; flex-direction: column; background: var(--creative-paper); border: 1px solid var(--creative-line); border-radius: 20px; overflow: hidden; box-shadow: var(--creative-shadow); animation: cd-fade-in .22s ease-out; }
@keyframes cd-fade-in { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }
.cd-topbar { flex-shrink: 0; display: flex; align-items: center; gap: 18px; padding: 17px 24px; border-bottom: 1px solid var(--creative-line); background: var(--creative-paper); }
.cd-back-btn { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-width: 38px; height: 38px; padding: 0 12px 0 9px; border: 1px solid var(--creative-line); border-radius: 999px; background: var(--creative-paper); color: var(--creative-muted); font: inherit; font-size: 11px; letter-spacing: .04em; cursor: pointer; flex-shrink: 0; transition: background .2s, color .2s, border-color .2s, box-shadow .2s, transform .2s; }
.cd-back-btn:hover { background: var(--creative-soft); color: var(--creative-accent); border-color: color-mix(in srgb, var(--creative-accent) 30%, var(--creative-line)); box-shadow: 0 5px 16px color-mix(in srgb, var(--creative-accent) 9%, transparent); }
.cd-back-btn:active { transform: translateY(1px); }
.cd-back-btn svg { width: 17px; height: 17px; }
.cd-topbar-title { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.cd-eyebrow { font-size: 9px; letter-spacing: .16em; color: var(--creative-muted); margin-bottom: 5px; }
.cd-eyebrow > span { margin-left: 6px; color: var(--creative-accent); }
.cd-title { font-size: 18px; font-weight: 600; line-height: 1.4; margin: 0; color: var(--creative-ink); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 14px; font-size: 10px; color: var(--creative-muted); margin-top: 7px; }
.cd-meta-item { display: inline-flex; align-items: center; gap: 4px; }
.cd-meta-item svg { width: 12px; height: 12px; flex-shrink: 0; }
.cd-meta-index { color: var(--creative-accent); font-variant-numeric: tabular-nums; }
.cd-topbar-note { color: var(--creative-muted); font-size: 11px; letter-spacing: .08em; flex-shrink: 0; }
.cd-topbar-note > span { font-size: 18px; margin-left: 5px; color: var(--creative-accent); }
.cd-body { flex: 1; min-height: 0; display: flex; scrollbar-width: thin; scrollbar-color: var(--creative-line) var(--creative-paper); }
.cd-main { flex: 1; min-width: 0; display: flex; min-height: 0; }
.cd-carousel { flex-shrink: 0; display: flex; overflow-x: auto; padding: 12px 20px; background: var(--creative-paper); border-top: 1px solid var(--creative-line); scrollbar-width: none; }
.cd-carousel::-webkit-scrollbar { display: none; }
.cd-carousel-group { display: flex; align-items: center; gap: 12px; width: max-content; margin-inline: auto; }
.cd-carousel-track { display: flex; gap: 8px; }
.cd-carousel-nav { flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; width: 32px; height: 32px; border: 1px solid var(--creative-line); border-radius: 50%; background: var(--creative-paper); color: var(--creative-muted); cursor: pointer; transition: background .15s, color .15s, border-color .15s, box-shadow .15s, transform .15s; }
.cd-carousel-nav:hover { background: var(--creative-soft); color: var(--creative-accent); border-color: color-mix(in srgb, var(--creative-accent) 28%, var(--creative-line)); box-shadow: 0 4px 12px color-mix(in srgb, var(--creative-accent) 8%, transparent); }
.cd-carousel-nav:active { transform: translateY(1px); }
.cd-carousel-thumb { flex-shrink: 0; width: 50px; height: 50px; border-radius: 9px; overflow: hidden; border: 2px solid transparent; cursor: pointer; padding: 2px; background: var(--creative-canvas); transition: border-color .2s, box-shadow .2s; }
.cd-carousel-thumb:hover { border-color: color-mix(in srgb, var(--creative-accent) 45%, transparent); }
.cd-thumb-current { border-color: var(--creative-accent); box-shadow: 0 0 0 2px var(--creative-soft); }
.cd-carousel-thumb :deep(.arco-image) { display: block; width: 100%; height: 100%; border-radius: 5px; overflow: hidden; }
.cd-carousel-thumb :deep(.arco-image-img) { width: 100%; height: 100%; object-fit: cover; display: block; }
.cd-loading { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 1rem; color: var(--creative-muted); background: var(--creative-canvas); }
@media (max-width: 1024px) { .cd-topbar-note { display: none; } }
@media (max-width: 768px) {
  .cd-root { padding: 0; }
  .cd-shell { border: 0; border-radius: 0; height: 100dvh; }
  .cd-topbar { padding: 14px 16px; gap: 12px; }
  .cd-back-btn { min-width: 38px; padding-inline: 9px; }
  .cd-back-btn span { display: none; }
  .cd-title { font-size: 16px; }
  .cd-eyebrow { font-size: 8px; }
  .cd-meta { gap: 10px; margin-top: 5px; }
  .cd-body { flex-direction: column; overflow-y: auto; }
  .cd-main { flex: 0 0 auto; flex-direction: column; }
  .cd-carousel { padding: 9px 12px; }
  .cd-carousel-group { gap: 8px; }
  .cd-carousel-nav { width: 26px; height: 26px; }
  .cd-carousel-thumb { width: 42px; height: 42px; }
  :deep(.cd-media-wrap) { flex: 0 0 auto; min-height: 200px; height: 44dvh; max-height: 50dvh; padding: 16px; }
  :deep(.cd-media-wrap:has(.cd-text-only)) { height: auto; max-height: none; }
  :deep(.cd-side) { flex: 0 0 auto; width: 100%; max-width: 100%; border-left: none; border-top: 1px solid var(--creative-line); overflow-y: visible; scrollbar-gutter: auto; }
}
</style>
