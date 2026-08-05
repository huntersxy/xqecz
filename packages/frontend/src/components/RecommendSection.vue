<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { IconRefresh } from '@arco-design/web-vue/es/icon'
import MediaImage from '@/components/MediaImage.vue'
import type { RecommendContent } from '@/types'
import type { RecommendLoader } from '@/composables/useRecommendLoader'

const props = defineProps<{ loader: RecommendLoader }>()
const emit = defineEmits<{ click: [item: RecommendContent] }>()
const scrollRef = ref<HTMLElement | null>(null)

// 解构出 loader 中的原始 ref（props.loader 是普通对象，解构得到共享的 ref 实例）
const {
  recommendContents,
  isRecommendLoading,
  loadedPage,
  poolGeneration,
  maxRecommendPages,
  pageSize,
  displayStart,
  loadNextPageIntoPool,
  refreshRecommend,
} = props.loader

// 运行时登记的破图 id：破图卡片整体从列表移除，正常图片左移补位
const brokenIds = ref<(string | number)[]>([])
function markBroken(id: string | number) {
  if (!brokenIds.value.includes(id)) brokenIds.value = [...brokenIds.value, id]
}

// 不显示无封面、不显示破图
const visibleContents = computed(() =>
  recommendContents.value.filter(
    (it) => it.thumb && !brokenIds.value.includes(it.id)
  )
)

// 展示窗口：从可见池中取一整批（坏图 / 文字类已剔除，不占名额）。
const displayList = computed(() =>
  visibleContents.value.slice(displayStart.value, displayStart.value + pageSize)
)

const filling = ref(false)
const fillQueued = ref(false)
async function ensureFilled() {
  // 加载中或已在填充：标记排队，待当前批次结束后再次补偿，
  // 保证"坏几张就补几张"（坏图 / 文字类一律不算进可见数量）。
  if (isRecommendLoading.value) {
    fillQueued.value = true
    return
  }
  if (filling.value) {
    fillQueued.value = true
    return
  }
  filling.value = true
  try {
    do {
      fillQueued.value = false
      while (
        visibleContents.value.length < displayStart.value + pageSize &&
        loadedPage.value < maxRecommendPages
      ) {
        const added = await loadNextPageIntoPool()
        if (!added) break
      }
      // 池已全部加载且窗口已越过末尾 → 回卷从头循环（"换一批"仍可见变化）
      if (loadedPage.value >= maxRecommendPages && visibleContents.value.length <= displayStart.value) {
        displayStart.value = 0
      }
    } while (fillQueued.value)
  } finally {
    filling.value = false
  }
}

// 初始加载完成 / 内容变化 / 破图登记后，尝试补齐数量
watch(
  () => [recommendContents.value.length, brokenIds.value.length, displayStart.value],
  () => {
    if (loadedPage.value > 0) ensureFilled()
  }
)

// 整池替换（刷新 / 初始加载）后清空破图记录
watch(
  () => poolGeneration.value,
  () => {
    brokenIds.value = []
  }
)

function onRefresh() {
  refreshRecommend()
}

// 电脑端：推荐条可横向滚动时，滚轮纵向位移转为横向滚动；放得下时正常滚页面。
function onWheel(e: WheelEvent) {
  const el = scrollRef.value
  if (!el) return
  if (el.scrollWidth > el.clientWidth && Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
    e.preventDefault()
    el.scrollLeft += e.deltaY
  }
}

onMounted(() => {
  scrollRef.value?.addEventListener('wheel', onWheel, { passive: false })
})

onBeforeUnmount(() => {
  scrollRef.value?.removeEventListener('wheel', onWheel)
})
</script>

<template>
  <section id="recommend-section" class="wf-recommend">
    <div class="wf-recommend-head">
      <h2>✨ 精选推荐</h2>
      <button
        class="wf-rec-refresh"
        type="button"
        :disabled="isRecommendLoading"
        :title="isRecommendLoading ? '刷新中…' : '换一批'"
        @click="onRefresh"
      >
        <IconRefresh :spin="isRecommendLoading" />
      </button>
    </div>
    <div ref="scrollRef" class="wf-recommend-scroll">
      <div
        v-for="item in displayList"
        :key="item.id"
        class="wf-recommend-card"
        @click="emit('click', item)"
      >
        <MediaImage
          :src="item.thumb"
          :alt="item.title"
          :preview="false"
          footer-position="inner"
          @error="markBroken(item.id)"
        >
          <template #extra>
            <span class="wf-rec-caption">{{ item.title }}</span>
          </template>
        </MediaImage>
      </div>
    </div>
  </section>
</template>

<style scoped>
.wf-recommend {
  max-width: 1600px; margin: 0 auto; padding: 1rem 1rem 0;
}
.wf-recommend-head {
  display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.625rem;
}
.wf-recommend-head h2 {
  margin: 0; font-size: 0.9375rem; font-weight: 700; color: var(--color-text-1);
}

.wf-rec-refresh {
  display: inline-flex; align-items: center; justify-content: center;
  width: 28px; height: 28px; border-radius: 8px;
  border: 1px solid var(--color-border-2);
  background: var(--color-bg-2);
  color: var(--color-text-2);
  cursor: pointer;
  transition: color 0.2s, background 0.2s, border-color 0.2s;
}
.wf-rec-refresh:hover:not(:disabled) {
  color: rgb(var(--primary-6)); border-color: rgb(var(--primary-6));
}
.wf-rec-refresh:disabled { opacity: 0.6; cursor: default; }
.wf-rec-refresh :deep(svg) { width: 16px; height: 16px; }

.wf-recommend-scroll {
  display: flex; gap: 0.625rem; overflow-x: auto;
  padding-bottom: 0.75rem; scrollbar-width: none;
}
.wf-recommend-scroll::-webkit-scrollbar { display: none; }

.wf-recommend-card {
  flex-shrink: 0; width: 140px; height: 180px; border-radius: 0.75rem;
  overflow: hidden; position: relative; cursor: pointer;
  background: var(--color-fill-2);
}
/* 暗色下给推荐缩略图叠一层灰色半透明遮罩，与壁纸/瀑布流卡片压暗保持一致 */
body[arco-theme='dark'] .wf-recommend-card::after {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 1;
  background: rgba(0, 0, 0, 0.35);
  pointer-events: none;
}
/* 暗色下 --color-fill-2 为 rgba(255,255,255,0.08) 半透明，卡片会透出壁纸 → 改用实色深色 token */
body[arco-theme='dark'] .wf-recommend-card { background: var(--color-bg-3); }
/* Arco <Image> 包裹层需填满卡片，内层 .arco-image-img 的 cover 才生效 */
.wf-recommend-card :deep(.arco-image) { display: block; width: 100%; height: 100%; }
.wf-recommend-card :deep(.arco-image-img) {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}
.wf-recommend-card:hover :deep(.arco-image-img) { transform: scale(1.08); }

.wf-rec-caption {
  display: -webkit-box; -webkit-line-clamp: 2;
  -webkit-box-orient: vertical; overflow: hidden;
  font-size: 0.6875rem; font-weight: 500; line-height: 1.3;
}
</style>
