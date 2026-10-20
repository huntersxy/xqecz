<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { useContentBrowse } from '@/composables/useContentBrowse'
import { toast } from '@/composables/useToast'
import { cachedImageSource } from '@/utils/imageSource'
import { mediaKindOf, resolveMedia } from '@/utils/contentMedia'
import SketchPlanet from '@/components/SketchPlanet.vue'
import CdIcon from '@/components/content/CdIcon.vue'
import MediaStage from '@/components/content/MediaStage.vue'
import Filmstrip from '@/components/content/Filmstrip.vue'
import InfoPanel from '@/components/content/InfoPanel.vue'
import ReportDialog from '@/components/content/ReportDialog.vue'
import ClaimDialog from '@/components/content/ClaimDialog.vue'
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

function cachedItem(id: number): Content | null {
  return cachedList.value.find((item) => item.id === id) || null
}

const preloaded = new Set<string>()
function preloadWork(item?: Content | null) {
  if (!item || mediaKindOf(item) !== 'image') return
  const source = cachedImageSource()
  if (!source) return
  const { url } = resolveMedia(item, source)
  if (!url || preloaded.has(url)) return
  preloaded.add(url)
  const img = new Image()
  img.decoding = 'async'
  img.src = url
}
function preloadById(id: number) { preloadWork(cachedItem(id)) }
function preloadNeighbours() {
  const idx = currentIndex.value
  if (idx < 0) return
  preloadWork(cachedList.value[idx - 1])
  preloadWork(cachedList.value[idx + 1])
}

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
    } else if (showLoading) {
      loadState.value = 'error'
      toast.error(res.message)
    }
  } catch {
    if (seq !== contentLoadSeq || id !== currentId.value) return
    if (showLoading) {
      loadState.value = 'error'
      toast.error('加载内容失败')
    }
  }
}

watch(() => route.params.id, () => {
  const hit = cachedItem(currentId.value)
  if (hit) {
    content.value = hit
    loadState.value = 'ready'
  }
  void loadContent(!hit)
  scrollThumbIntoView()
})

onMounted(() => {
  userStore.checkAuth()
  void loadContent()
})
</script>

<template>
  <div class="cd-root creative-theme">
    <div class="cd-shell">
      <button class="cd-back-btn" type="button" aria-label="返回" @click="goBack">
        <CdIcon name="arrow-left" />
        <span>返回</span>
      </button>

      <div v-if="loadState === 'loading'" class="cd-loading" role="status">
        <SketchPlanet class="cd-loading-planet" loading />
        <p>正在穿越小泉宇宙…</p>
      </div>

      <div v-else-if="loadState === 'error'" class="cd-loading" role="alert">
        <CdIcon name="image-broken" class="cd-error-icon" />
        <h2 class="cd-error-title">信号暂时离开了轨道</h2>
        <p>内容不存在或网络异常，请稍后再试</p>
        <button type="button" class="cd-retry-btn" @click="loadContent()">
          <CdIcon name="refresh" />重试
        </button>
      </div>

      <div v-else-if="content" class="cd-body">
        <section class="cd-stage" aria-label="作品展示">
          <MediaStage :content="content" />

          <span v-if="currentIndex >= 0" class="cd-counter">
            {{ currentIndex + 1 }} / {{ cachedList.length }}
          </span>

          <template v-if="previewItems.length > 0">
            <button v-if="hasPrev" class="cd-stage-arrow cd-stage-prev" type="button" aria-label="上一个" @click="goToPrev">
              <CdIcon name="chevron-left" />
            </button>
            <button v-if="hasNext" class="cd-stage-arrow cd-stage-next" type="button" aria-label="下一个" @click="goToNext">
              <CdIcon name="chevron-right" />
            </button>
          </template>

          <div v-if="previewItems.length > 0" ref="previewRef" class="cd-film-host">
            <Filmstrip
              :items="previewItems"
              :has-prev="hasPrev"
              :has-next="hasNext"
              @select="navigateTo"
              @prev="goToPrev"
              @next="goToNext"
              @hover="preloadById"
            />
          </div>
        </section>

        <InfoPanel :content="content" @open-claim="showClaimModal = true" @report-comment="reportTarget = $event" />
      </div>
    </div>

    <ReportDialog :target="reportTarget" @close="reportTarget = null" />
    <ClaimDialog :open="showClaimModal" :content-id="Number(content?.id) || 0" @close="showClaimModal = false" />
  </div>
</template>

<style scoped>
.cd-root { position: fixed; inset: 0; z-index: 1000; color: var(--creative-ink); background: var(--creative-canvas); }
.cd-shell { position: relative; width: 100%; height: 100%; display: flex; flex-direction: column; }

.cd-back-btn, .cd-counter, .cd-stage-arrow {
  border: 1px solid color-mix(in srgb, var(--creative-line) 80%, transparent);
  background: color-mix(in srgb, var(--creative-paper) 82%, transparent);
  backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
  box-shadow: 0 4px 16px rgba(48, 36, 39, .1);
}
.cd-back-btn {
  position: absolute; top: 16px; left: 16px; z-index: 5; display: inline-flex; align-items: center; gap: 7px;
  height: 38px; padding: 0 14px 0 11px; border-radius: 999px; color: var(--creative-ink);
  font: inherit; font-size: 12px; cursor: pointer; transition: color .18s, border-color .18s, background .18s, transform .18s;
}
.cd-back-btn:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-paper); }
.cd-back-btn:active { transform: translateY(1px); }
.cd-back-btn .cd-icon { font-size: 16px; }

.cd-body { flex: 1; min-height: 0; display: flex; }
.cd-stage { position: relative; flex: 1 1 0; min-width: 0; display: flex; flex-direction: column; }
.cd-film-host { display: contents; }
.cd-counter {
  position: absolute; top: 16px; right: 16px; z-index: 5; display: inline-flex; align-items: center; height: 30px; padding: 0 12px;
  border-radius: 999px; color: var(--creative-accent); font-size: 11px; font-variant-numeric: tabular-nums; pointer-events: none;
}
.cd-stage-arrow {
  position: absolute; top: calc((100% - 68px) / 2); z-index: 5; transform: translateY(-50%);
  display: grid; place-items: center; width: 44px; height: 44px; border-radius: 50%; color: var(--creative-ink);
  font-size: 18px; cursor: pointer; transition: color .18s, border-color .18s, background .18s, box-shadow .18s;
}
.cd-stage-prev { left: 16px; }
.cd-stage-next { right: 16px; }
.cd-stage-arrow:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-paper); box-shadow: 0 6px 20px color-mix(in srgb, var(--creative-accent) 16%, transparent); }
.cd-stage-arrow:active { transform: translateY(-50%) scale(.96); }

.cd-loading { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--creative-muted); font-size: 12px; }
.cd-loading p { margin: 0; }
.cd-loading-planet { width: 44px; height: 44px; color: var(--creative-accent); }
.cd-error-icon { font-size: 44px; }
.cd-error-title { margin: 0; color: var(--creative-ink); font: 600 18px/1.3 var(--creative-title-font); }
.cd-retry-btn { display: inline-flex; align-items: center; gap: 6px; height: 36px; padding: 0 18px; border: 0; border-radius: 999px; background: var(--creative-accent); color: var(--color-on-primary, #fff); font: inherit; font-size: 13px; cursor: pointer; }
.cd-retry-btn:hover { background: var(--creative-accent-hover); }

@media (max-width: 768px) {
  .cd-shell { height: 100dvh; }
  .cd-back-btn { top: 10px; left: 10px; width: 38px; padding: 0; justify-content: center; }
  .cd-back-btn span { display: none; }
  .cd-counter { top: 10px; right: 10px; }
  .cd-body { flex-direction: column; overflow-y: auto; }
  .cd-stage { flex: 0 0 auto; }
  .cd-stage-arrow { display: none; }
}
</style>
