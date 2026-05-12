<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  src: string
  alt?: string
  class?: string
}>()

const isLoaded = ref(false)
const isInView = ref(false)
const imgRef = ref<HTMLImageElement | null>(null)
let observer: IntersectionObserver | null = null

onMounted(() => {
  if (!imgRef.value) return

  // 使用 Intersection Observer 检测图片是否进入视口
  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          isInView.value = true
          // 加载图片
          const img = new Image()
          img.onload = () => {
            isLoaded.value = true
          }
          img.src = props.src
          // 停止观察
          observer?.unobserve(entry.target)
        }
      })
    },
    {
      rootMargin: '50px', // 提前 50px 开始加载
      threshold: 0.01
    }
  )

  observer.observe(imgRef.value)
})

onUnmounted(() => {
  if (observer && imgRef.value) {
    observer.unobserve(imgRef.value)
    observer = null
  }
})
</script>

<template>
  <div
    ref="imgRef"
    class="lazy-image-wrapper"
    :class="{ 'is-loaded': isLoaded }"
  >
    <!-- 占位符 -->
    <div v-if="!isLoaded" class="lazy-image-placeholder">
      <svg class="placeholder-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
        <circle cx="8.5" cy="8.5" r="1.5"/>
        <polyline points="21 15 16 10 5 21"/>
      </svg>
    </div>
    <!-- 实际图片 -->
    <img
      v-show="isLoaded"
      :src="src"
      :alt="alt || ''"
      :class="class"
      class="lazy-image"
    />
  </div>
</template>

<style scoped>
.lazy-image-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
  background: #f5f5f5;
  overflow: hidden;
}

.lazy-image-placeholder {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e8ec 100%);
}

.placeholder-icon {
  width: 32px;
  height: 32px;
  color: #ccc;
}

.lazy-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.lazy-image-wrapper.is-loaded .lazy-image {
  opacity: 1;
}
</style>
