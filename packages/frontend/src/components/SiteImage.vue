<script setup lang="ts">
import { onMounted, ref, toRef, watch } from 'vue'
import SiteIcon from '@/components/SiteIcon.vue'
import { useImageFallback } from '@/composables/useImageFallback'

/**
 * 不依赖组件库的图片：加载前显示柔和骨架，加载完成后淡入，失败时沿用
 * MediaImage 同一套兜底链（useImageFallback），最终失败显示坏图占位。
 * 不含预览、工具栏等能力，只服务于首页卡片与精选条这类纯展示场景。
 */
const props = withDefaults(
  defineProps<{
    src?: string
    alt?: string
    fallbackSrc?: string | string[]
    loading?: 'lazy' | 'eager'
    decoding?: 'async' | 'sync' | 'auto'
  }>(),
  { src: '', alt: '', fallbackSrc: '', loading: 'lazy', decoding: 'async' },
)

const emit = defineEmits<{ error: []; fallback: [failedUrl: string] }>()

const { currentSrc, finalFailed, onLoadError } = useImageFallback({
  src: toRef(props, 'src'),
  fallbackSrc: toRef(props, 'fallbackSrc'),
  onFallback: (failed) => emit('fallback', failed),
  onFinalError: () => emit('error'),
})

const img = ref<HTMLImageElement | null>(null)
const loaded = ref(false)

// 换源（含兜底切换）后重新等待 load，骨架随之回来，避免新图加载期间露出空白。
watch(currentSrc, () => { loaded.value = false }, { flush: 'sync' })

// 浏览器缓存命中时 load 事件可能早于监听挂上，挂载后补查一次。
onMounted(() => {
  const el = img.value
  if (el && el.complete && el.naturalWidth > 0) loaded.value = true
})
</script>

<template>
  <span class="site-image" :class="{ 'is-loaded': loaded, 'is-failed': finalFailed }">
    <img
      v-if="!finalFailed"
      ref="img"
      class="site-image-img"
      :src="currentSrc"
      :alt="alt"
      :loading="loading"
      :decoding="decoding"
      referrerpolicy="no-referrer"
      @load="loaded = true"
      @error="onLoadError"
    />
    <span v-else class="site-image-error" role="img" :aria-label="alt || '图片加载失败'">
      <SiteIcon name="image-broken" />
    </span>
  </span>
</template>

<style scoped>
.site-image {
  position: relative;
  display: block;
  width: 100%;
  min-height: 80px;
  overflow: hidden;
  background: var(--site-soft, var(--creative-soft, #f1e9e6));
}
/* 骨架：只在尚未加载且未失败时出现，扫光很淡，避免瀑布流里一片闪烁。 */
.site-image:not(.is-loaded):not(.is-failed)::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(100deg, transparent 20%, color-mix(in srgb, var(--site-paper, #fff) 55%, transparent) 50%, transparent 80%);
  background-size: 220% 100%;
  animation: site-image-shimmer 1.4s ease-in-out infinite;
  pointer-events: none;
}
.site-image-img {
  display: block;
  width: 100%;
  height: auto;
  vertical-align: top;
  opacity: 0;
  transition: opacity .32s ease;
}
.site-image.is-loaded .site-image-img { opacity: 1; }
.site-image-error {
  display: grid;
  place-items: center;
  width: 100%;
  min-height: inherit;
  aspect-ratio: 4 / 3;
  color: var(--site-muted, var(--creative-muted, #817779));
  font-size: 22px;
}
@keyframes site-image-shimmer {
  from { background-position: 120% 0; }
  to { background-position: -120% 0; }
}
@media (prefers-reduced-motion: reduce) {
  .site-image:not(.is-loaded):not(.is-failed)::after { animation: none; }
  .site-image-img { transition-duration: .01ms; }
}
</style>
