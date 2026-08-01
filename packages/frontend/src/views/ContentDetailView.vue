<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { useContentBrowse } from '@/composables/useContentBrowse'
import { getImageUrl, formatTime } from '@/utils'
import { IconArrowLeft, IconCalendar, IconEye, IconLeft, IconRight, IconRefresh } from '@arco-design/web-vue/es/icon'
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
const reportTarget = ref<Comment | null>(null)
const showClaimModal = ref(false)

// ── 加载内容 ──
async function loadContent() {
  loadState.value = 'loading'
  try {
    const res = await contentApi.detail(currentId.value)
    if (res.code === 200) {
      content.value = res.data
      loadState.value = 'ready'
    } else {
      loadState.value = 'error'
      Message.error(res.message)
    }
  } catch {
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
  // 全屏覆盖时锁定背景滚动，并补偿滚动条宽度
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
</script>

<template>
  <div class="cd-root">
    <!-- 遮罩 -->
    <div class="cd-backdrop" @click="goBack"></div>

    <div class="cd-shell" @click.self="goBack">
      <!-- 顶部条 -->
      <header class="cd-topbar">
        <a-button class="cd-back-btn" type="text" shape="circle" aria-label="返回" @click="goBack">
          <IconArrowLeft />
        </a-button>
        <div class="cd-topbar-title">
          <h2 class="cd-title">{{ content?.title || '加载中...' }}</h2>
          <div v-if="content" class="cd-meta">
            <span class="cd-meta-item">
              <IconCalendar />
              {{ formatTime(content.created_at) }}
            </span>
            <span class="cd-meta-item">
              <IconEye />
              {{ content.view_count }} 浏览
            </span>
            <span v-if="currentIndex >= 0" class="cd-meta-item cd-meta-index">
              {{ currentIndex + 1 }} / {{ cachedList.length }}
            </span>
          </div>
        </div>
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
              @click="navigateTo(item.id)"
            >
              <a-image :src="getImageUrl(item.thumb)" :alt="item.title" :preview="false" loading="lazy" />
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
.cd-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text);
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
  background: var(--color-surface);
  overflow: hidden;
}

@keyframes cd-fade-in {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}

/* ── 顶部条 ── */
.cd-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-header-bg);
}

.cd-back-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.2s, color 0.2s;
}
.cd-back-btn:hover { background: var(--color-hover); color: var(--color-text); }
.cd-back-btn svg { width: 18px; height: 18px; }

.cd-topbar-title { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.cd-title {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.3;
  margin: 0;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cd-meta {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 2px;
}
.cd-meta-item { display: inline-flex; align-items: center; gap: 4px; }
.cd-meta-item svg { width: 13px; height: 13px; flex-shrink: 0; }
.cd-meta-index { color: var(--color-primary); font-weight: 600; }

/* ── 主体 ── */
.cd-body { flex: 1; min-height: 0; display: flex; gap: 0; }

/* ── 底部预览走马灯 ── */
.cd-carousel {
  flex-shrink: 0;
  display: flex;
  overflow-x: auto;
  padding: 8px 12px;
  background: var(--color-header-bg);
  border-top: 1px solid var(--color-border);
  scrollbar-width: none;
}
.cd-carousel::-webkit-scrollbar { display: none; }

.cd-carousel-group {
  display: flex;
  align-items: center;
  gap: 6px;
  width: max-content;
  margin-inline: auto; /* 按钮+缩略图整体居中；溢出时贴左可滚动 */
}

.cd-carousel-track {
  display: flex;
  gap: 6px;
}

.cd-carousel-nav {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 50%;
  background: var(--color-hover);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.cd-carousel-nav:hover {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.cd-carousel-thumb {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: 6px;
  overflow: hidden;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
  background: var(--color-border);
  transition: border-color 0.2s, box-shadow 0.2s;
}
.cd-carousel-thumb:hover { border-color: var(--color-primary); }
.cd-thumb-current {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-primary) 30%, transparent);
}
.cd-carousel-thumb :deep(.arco-image) { display: block; width: 100%; height: 100%; }
.cd-carousel-thumb :deep(.arco-image-img) { width: 100%; height: 100%; object-fit: cover; display: block; }

/* ── 中间主体 ── */
.cd-main { flex: 1; min-width: 0; display: flex; }

/* ── 加载中 ── */
.cd-loading {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 1rem;
  color: var(--color-text-secondary);
}

/* ── 窄屏 ── */
@media (max-width: 768px) {
  .cd-body { flex-direction: column; overflow-y: auto; }
  .cd-main { flex-direction: column; }
  .cd-carousel { padding: 6px 8px; }
  .cd-carousel-nav { width: 24px; height: 24px; }
  .cd-carousel-thumb { width: 48px; height: 48px; }
  :deep(.cd-media-wrap) { flex: 1 1 100%; max-height: 50vh; }
  :deep(.cd-side) { flex: 1 1 100%; max-width: 100%; border-left: none; border-top: 1px solid var(--color-border); }
}
</style>
