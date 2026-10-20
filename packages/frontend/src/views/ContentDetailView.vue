<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { useContentBrowse } from '@/composables/useContentBrowse'
import MediaImage from '@/components/MediaImage.vue'
import SketchPlanet from '@/components/SketchPlanet.vue'
import { IconArrowLeft, IconLeft, IconRight, IconRefresh } from '@arco-design/web-vue/es/icon'
import ContentMedia from '@/components/ContentMedia.vue'
import ContentSidebar from '@/components/ContentSidebar.vue'
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
    <!-- 整屏分两栏：左侧是作品舞台，右侧是信息面板；返回键悬浮在舞台左上角 -->
    <div class="cd-shell">
      <button class="cd-back-btn" type="button" aria-label="返回" @click="goBack">
        <IconArrowLeft />
        <span>返回</span>
      </button>

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

      <!-- 主体：舞台 + 信息面板 -->
      <div v-else-if="content" class="cd-body">
        <section class="cd-stage" aria-label="作品展示">
          <ContentMedia :content="content" />

          <span v-if="currentIndex >= 0" class="cd-counter" aria-label="当前位置">
            {{ currentIndex + 1 }} / {{ cachedList.length }}
          </span>

          <!-- 宽屏：舞台两侧的翻页键；窄屏改用走马灯自带的前后键 -->
          <template v-if="previewItems.length > 0">
            <button
              v-if="hasPrev"
              class="cd-stage-arrow cd-stage-prev"
              type="button"
              aria-label="上一个"
              @click="goToPrev"
            >
              <IconLeft />
            </button>
            <button
              v-if="hasNext"
              class="cd-stage-arrow cd-stage-next"
              type="button"
              aria-label="下一个"
              @click="goToNext"
            >
              <IconRight />
            </button>
          </template>

          <!-- 底部预览走马灯（内容不足一行时居中） -->
          <div
            v-if="previewItems.length > 0"
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
        </section>

        <ContentSidebar :content="content" @open-claim="showClaimModal = true" @report-comment="reportTarget = $event" />
      </div>
    </div>

    <!-- 举报弹窗 -->
    <ReportModal :target="reportTarget" @close="reportTarget = null" />
    <!-- 认领弹窗 -->
    <ClaimModal :open="showClaimModal" :content-id="Number(content?.id) || 0" @close="showClaimModal = false" />
  </div>
</template>

<style scoped>
/* 详情页 = 左侧作品舞台 + 右侧信息面板，整屏铺满，不再有顶栏/底栏占用竖向空间。
   舞台只负责把作品放大看清；标题、作者、操作、正文、评论全部归到右侧面板。
   --creative-* 已统一为 --site-* 别名，无需在此重声明。 */
.cd-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  color: var(--creative-ink);
  background: var(--creative-canvas);
}
.cd-shell {
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  animation: cd-fade-in .2s ease-out;
}
@keyframes cd-fade-in { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }

/* 悬浮控件共用的半透明胶囊：压在作品上也读得清，又不抢作品本身 */
.cd-back-btn,
.cd-counter,
.cd-stage-arrow {
  border: 1px solid color-mix(in srgb, var(--creative-line) 80%, transparent);
  background: color-mix(in srgb, var(--creative-paper) 82%, transparent);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  box-shadow: 0 4px 16px rgba(48, 36, 39, .1);
}
.cd-back-btn {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 5;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 38px;
  padding: 0 14px 0 11px;
  border-radius: 999px;
  color: var(--creative-ink);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
  transition: color .18s, border-color .18s, background .18s, transform .18s;
}
.cd-back-btn:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-paper); }
.cd-back-btn:active { transform: translateY(1px); }
.cd-back-btn svg { width: 16px; height: 16px; }

.cd-body { flex: 1; min-height: 0; display: flex; }

/* 舞台：作品居中，走马灯贴底，翻页键悬在两侧 */
.cd-stage {
  position: relative;
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.cd-counter {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 5;
  padding: 0 12px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  color: var(--creative-accent);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  pointer-events: none;
}
.cd-stage-arrow {
  position: absolute;
  top: calc((100% - 68px) / 2);
  z-index: 5;
  transform: translateY(-50%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  color: var(--creative-ink);
  cursor: pointer;
  transition: color .18s, border-color .18s, background .18s, box-shadow .18s;
}
.cd-stage-arrow svg { width: 18px; height: 18px; }
.cd-stage-prev { left: 16px; }
.cd-stage-next { right: 16px; }
.cd-stage-arrow:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-paper); box-shadow: 0 6px 20px color-mix(in srgb, var(--creative-accent) 16%, transparent); }
.cd-stage-arrow:active { transform: translateY(-50%) scale(.96); }

.cd-carousel {
  position: relative;
  flex-shrink: 0; display: flex; overflow-x: auto;
  padding: 11px clamp(14px, 2.5vw, 30px);
  background: color-mix(in srgb, var(--creative-paper) 70%, transparent);
  border-top: 1px solid var(--creative-line);
  scrollbar-width: none;
  /* 两侧渐隐：滚动到尽头时不像被硬切掉 */
  mask-image: linear-gradient(90deg, transparent 0, #000 26px, #000 calc(100% - 26px), transparent 100%);
}
.cd-carousel::-webkit-scrollbar { display: none; }
.cd-carousel-group { display: flex; align-items: center; gap: 10px; width: max-content; margin-inline: auto; }
.cd-carousel-track { display: flex; gap: 7px; }
.cd-carousel-nav { display: none; flex-shrink: 0; align-items: center; justify-content: center; width: 30px; height: 30px; border: 1px solid var(--creative-line); border-radius: 8px; background: transparent; color: var(--creative-muted); cursor: pointer; transition: background .15s, color .15s, border-color .15s; }
.cd-carousel-nav:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-soft); }
.cd-carousel-nav:active { transform: translateY(1px); }
.cd-carousel-thumb { flex-shrink: 0; width: 60px; height: 46px; border-radius: 8px; overflow: hidden; border: 1px solid transparent; cursor: pointer; padding: 0; background: var(--creative-canvas); transition: border-color .18s, opacity .18s, transform .18s; }
.cd-carousel-thumb:hover { border-color: color-mix(in srgb, var(--creative-accent) 60%, transparent); opacity: .9; transform: translateY(-1px); }
.cd-thumb-current { border-color: var(--creative-accent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--creative-accent) 22%, transparent); }
.cd-carousel-thumb :deep(.arco-image) { display: block; width: 100%; height: 100%; border-radius: 5px; overflow: hidden; }
.cd-carousel-thumb :deep(.arco-image-img) { width: 100%; height: 100%; object-fit: cover; display: block; }

.cd-loading-planet { width: 44px; height: 44px; color: var(--creative-accent); }
.cd-loading { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--creative-muted); background: var(--creative-canvas); font-size: 12px; }

@media (max-width: 768px) {
  .cd-shell { height: 100dvh; }
  .cd-back-btn { top: 10px; left: 10px; width: 38px; padding: 0; justify-content: center; }
  .cd-back-btn span { display: none; }
  .cd-counter { top: 10px; right: 10px; }
  .cd-body { flex-direction: column; overflow-y: auto; }
  .cd-stage { flex: 0 0 auto; }
  .cd-stage-arrow { display: none; }
  .cd-carousel { padding: 9px 12px; }
  .cd-carousel-group { gap: 8px; }
  .cd-carousel-nav { display: inline-flex; width: 26px; height: 26px; }
  .cd-carousel-thumb { width: 52px; height: 40px; }
  :deep(.cd-media-wrap) { flex: 0 0 auto; min-height: 200px; height: 46dvh; max-height: 56dvh; }
  :deep(.cd-media-wrap:has(.cd-text-only)) { height: auto; max-height: none; }
}

</style>
