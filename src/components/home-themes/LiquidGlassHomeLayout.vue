<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { motion } from 'motion-v'
import logoImg from '@/assets/logo.webp'
import PollComponent from '@/components/PollComponent.vue'
import { useHomeLogic } from '@/composables/useHomeLogic'
import { useThemeStore } from '@/stores/theme'
import type { Content } from '@/types'

const {
  contents,
  recommendContents,
  sortedTags,
  searchKeyword,
  selectedTags,
  selectedTypes,
  page,
  isLoading,
  isRecommendLoading,
  total,
  totalPages,
  recommendHint,
  swapSections,
  contentTypes,
  contentTypeLabels,
  selectTag,
  selectType,
  handleSearch,
  goToDetail,
  goToPage,
  goToEasterEgg,
  refreshRecommend,
  onMountedHome,
  getImageUrl,
  onImageLoad,
  getPreviewText,
} = useHomeLogic()

const themeStore = useThemeStore()
const glassReady = ref(false)

function roundedRectSDF(x: number, y: number, hw: number, hh: number, r: number) {
  const qx = Math.abs(x) - hw + r
  const qy = Math.abs(y) - hh + r
  return Math.min(Math.max(qx, qy), 0) + Math.hypot(Math.max(qx, 0), Math.max(qy, 0)) - r
}

function smoothstep(a: number, b: number, t: number) {
  const x = Math.max(0, Math.min(1, (t - a) / (b - a)))
  return x * x * (3 - 2 * x)
}

let liquidGlassMapCache: { canvas: HTMLCanvasElement; scale: number } | null = null

function generateLiquidGlassMap() {
  if (liquidGlassMapCache) return liquidGlassMapCache

  const w = 300
  const h = 200
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')!
  const imageData = ctx.createImageData(w, h)
  const data = imageData.data

  let maxScale = 0
  const rawValues: number[] = []

  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const uvx = x / w
      const uvy = y / h
      const ix = uvx - 0.5
      const iy = uvy - 0.5
      const distanceToEdge = roundedRectSDF(ix, iy, 0.48, 0.46, 0.04)
      const edgeFactor = smoothstep(0.04, -0.03, distanceToEdge)
      const strength = edgeFactor * 0.65
      const posX = ix * (1 - strength) + 0.5
      const posY = iy * (1 - strength) + 0.5
      const dx = posX * w - x
      const dy = posY * h - y
      maxScale = Math.max(maxScale, Math.abs(dx), Math.abs(dy))
      rawValues.push(dx, dy)
    }
  }

  maxScale *= 0.5
  if (maxScale === 0) maxScale = 1

  let index = 0
  for (let i = 0; i < data.length; i += 4) {
    const r = rawValues[index++] / maxScale + 0.5
    const g = rawValues[index++] / maxScale + 0.5
    data[i] = Math.round(r * 255)
    data[i + 1] = Math.round(g * 255)
    data[i + 2] = 0
    data[i + 3] = 255
  }

  ctx.putImageData(imageData, 0, 0)
  liquidGlassMapCache = { canvas, scale: maxScale }
  return liquidGlassMapCache
}

function applyGlassEffect() {
  const { canvas, scale } = generateLiquidGlassMap()
  const dataUrl = canvas.toDataURL()

  const mapImage = document.getElementById('liquid-glass-map') as unknown as SVGImageElement | null
  const displMap = document.querySelector('#liquid-glass feDisplacementMap') as SVGFEDisplacementMapElement | null

  if (mapImage) {
    mapImage.setAttributeNS('http://www.w3.org/1999/xlink', 'href', dataUrl)
    mapImage.setAttribute('href', dataUrl)
  }
  if (displMap) {
    displMap.setAttribute('scale', scale.toFixed(2))
  }

  const blurVal = getComputedStyle(document.documentElement).getPropertyValue('--theme-blur').trim() || '2px'

  const glassElements = document.querySelectorAll('.glass-card, .poll-card, .poll-component, .glass-search-bar')
  glassElements.forEach(el => {
    const htmlEl = el as HTMLElement
    ;(htmlEl.style as any).backdropFilter = `url(#liquid-glass) blur(${blurVal}) contrast(1.05) brightness(1.08)`
    ;(htmlEl.style as any).webkitBackdropFilter = `url(#liquid-glass) blur(${blurVal}) contrast(1.05) brightness(1.08)`
    htmlEl.style.backgroundColor = 'transparent'
    htmlEl.style.backgroundImage = 'none'
  })

  glassReady.value = true
}

let glassObserver: MutationObserver | null = null

onMounted(() => {
  onMountedHome()
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      requestAnimationFrame(() => applyGlassEffect())
    }, { once: true })
  } else {
    requestAnimationFrame(() => applyGlassEffect())
  }

  glassObserver = new MutationObserver(() => {
    requestAnimationFrame(() => applyGlassEffect())
  })
  glassObserver.observe(document.body, { childList: true, subtree: true })

  watch(() => themeStore.glassBlur, () => {
    requestAnimationFrame(() => applyGlassEffect())
  })
})

onUnmounted(() => {
  glassObserver?.disconnect()
})
</script>

<template>
  <div class="min-h-screen bg-[url('@/assets/bg.webp')] bg-cover bg-center bg-fixed bg-[#082f49] p-4 flex justify-center">
    <svg style="position: absolute; width: 0; height: 0;">
      <filter id="liquid-glass">
        <feImage id="liquid-glass-map" result="map" />
        <feDisplacementMap in="SourceGraphic" in2="map" xChannelSelector="R" yChannelSelector="G" scale="0" />
      </filter>
    </svg>

    <div class="w-full max-w-[1200px] bg-white/[0.08] rounded-2xl border border-white/20 overflow-hidden shadow-[0_8px_32px_rgba(0,0,0,0.3)]">
      <header class="flex items-center gap-4 px-5 py-2.5 bg-gradient-to-b from-white/10 to-white/4 border-b border-white/10">
        <div class="flex gap-2">
          <span class="w-3 h-3 rounded-full bg-[#ff5f57]"></span>
          <span class="w-3 h-3 rounded-full bg-[#febc2e]"></span>
          <span class="w-3 h-3 rounded-full bg-[#28c840]"></span>
        </div>
        <span class="text-[13px] text-white font-medium">小泉动漫二创站</span>
      </header>

      <div class="p-6 text-white/90 [text-shadow:-1px_-1px_0_rgba(0,0,0,0.5),1px_-1px_0_rgba(0,0,0,0.5),-1px_1px_0_rgba(0,0,0,0.5),1px_1px_0_rgba(0,0,0,0.5)]">
        <section class="text-center mb-8 flex flex-col items-center">
          <img :src="logoImg" alt="小泉动漫" class="block max-w-full h-auto max-h-[80px] sm:max-h-[120px] mx-auto mb-2" />
          <button @click="goToEasterEgg" class="inline-flex items-center gap-2 px-6 py-3 bg-white/15 backdrop-blur-[12px] text-white border border-white/30 rounded-full text-sm font-semibold cursor-pointer transition-all duration-300 hover:scale-105 hover:bg-white/25 hover:shadow-[0_8px_24px_rgba(0,0,0,0.2)]">
            <span>🎉</span>
            <span>发现精彩</span>
          </button>
        </section>

        <section class="mb-6">
          <div class="glass-search-bar flex max-w-[600px] mx-auto bg-white/10 border border-white/20 rounded-xl overflow-hidden">
            <input
              v-model="searchKeyword"
              type="text"
              placeholder="搜索..."
              @keyup.enter="handleSearch"
              class="flex-1 px-4 py-3 bg-transparent border-none text-white text-sm outline-none placeholder:text-white/85"
            />
            <button @click="handleSearch" class="px-4 py-3 bg-white/15 backdrop-blur-[12px] border-none text-white cursor-pointer transition-all duration-200 hover:bg-white/25">
              <svg class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="m21 21-4.35-4.35"/>
              </svg>
            </button>
          </div>
        </section>

        <section class="mb-6">
          <PollComponent />
        </section>

        <section class="glass-card p-4 rounded-xl mb-8 flex flex-col gap-4">
          <div class="flex items-center gap-3 flex-wrap">
            <span class="text-[13px] text-white/90 font-medium min-w-[40px]">类型</span>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="type in contentTypes"
                :key="type"
                @click="selectType(type)"
                class="px-3.5 py-1.5 bg-white/8 border border-white/15 rounded-full text-white/95 text-[13px] cursor-pointer transition-all duration-200 hover:border-[#34d399] hover:text-[#34d399]"
                :class="{ 'bg-[#34d399]/20 border-[#34d399]/40 text-[#34d399] font-medium': selectedTypes.includes(type) }"
              >
                {{ contentTypeLabels[type] }}
              </button>
            </div>
          </div>
          <div class="flex items-center gap-3 flex-wrap">
            <span class="text-[13px] text-white/90 font-medium min-w-[40px]">标签</span>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="tag in sortedTags"
                :key="tag"
                @click="selectTag(tag)"
                class="px-3.5 py-1.5 bg-white/8 border border-white/15 rounded-full text-white/95 text-[13px] cursor-pointer transition-all duration-200 hover:border-[#34d399] hover:text-[#34d399]"
                :class="{ 'bg-[#34d399]/20 border-[#34d399]/40 text-[#34d399] font-medium': selectedTags.includes(tag) }"
              >
                {{ tag }}
              </button>
            </div>
          </div>
        </section>

        <motion.div
          class="min-h-[100px]"
          :animate="{ height: 'auto', transition: { duration: 0.3, ease: 'easeOut' } }"
        >
          <motion.div
            id="recommend-section"
            class="mb-8"
            :initial="{ opacity: 1, y: 0 }"
            :animate="{
              opacity: swapSections ? 0 : 1,
              y: swapSections ? -80 : 0,
              height: swapSections ? 0 : 'auto',
              transition: { duration: 0.3, ease: 'easeOut' },
            }"
            style="overflow: hidden"
          >
            <div class="flex items-center gap-3 mb-5 flex-wrap">
              <h2 class="text-lg text-white m-0 font-semibold">推荐内容</h2>
              <div class="flex items-center gap-3 ml-auto">
                <span class="text-[13px] text-white/90">精选推荐</span>
                <button @click="refreshRecommend" class="flex items-center gap-1.5 px-3.5 py-2 bg-white/10 border border-white/20 rounded-lg text-white/95 text-[13px] cursor-pointer transition-all duration-200 hover:border-[#22d3ee] hover:text-[#22d3ee]" :disabled="isRecommendLoading">
                  <svg class="w-3.5 h-3.5" :class="{ 'animate-[spin_0.8s_linear_infinite]': isRecommendLoading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="23 4 23 10 17 10" />
                    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
                  </svg>
                  {{ isRecommendLoading ? '加载中' : '刷新' }}
                </button>
              </div>
              <span v-if="recommendHint" class="text-[13px] text-[#f87171] animate-[pulse_1.5s_ease-in-out_infinite]">{{ recommendHint }}</span>
            </div>

            <div class="grid gap-4 transition-opacity duration-300 grid-cols-2 lg:grid-cols-4" :class="{ 'opacity-30 pointer-events-none': isRecommendLoading }">
              <div
                v-for="content in recommendContents"
                :key="content.id"
                @click="goToDetail({ ...content, type: content.type, audit_status: 'approved' } as unknown as Content)"
                class="glass-card overflow-hidden cursor-pointer transition-all duration-300 hover:-translate-y-1 hover:shadow-[0_12px_32px_rgba(0,0,0,0.3)] hover:border-white/30"
              >
                <div class="relative w-full pt-[75%] bg-white/5 overflow-hidden">
                  <img :src="getImageUrl(content.thumb)" :alt="content.title" loading="lazy" @load="onImageLoad" class="absolute top-0 left-0 w-full h-full object-cover" />
                  <div v-if="content.type === 'video'" class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-12 h-12 bg-black/60 rounded-full flex items-center justify-center">
                    <svg class="w-5 h-5 text-white ml-0.5" viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>
                  </div>
                </div>
                <div class="p-3">
                  <h3 class="text-sm text-white m-0 mb-2 font-semibold whitespace-nowrap overflow-hidden text-ellipsis">{{ content.title }}</h3>
                  <div class="flex items-center gap-2 flex-wrap mb-2">
                    <span class="flex items-center gap-1 text-xs text-white/90">{{ content.user.username }}</span>
                    <div class="flex gap-1">
                      <span v-for="tag in content.tags.slice(0, 2)" :key="tag" class="px-2 py-0.5 bg-[#fbbf24]/15 rounded text-[11px] text-[#fbbf24]">{{ tag }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="isRecommendLoading" class="relative inset-0 flex flex-col items-center justify-center bg-[#082f49]/80 z-10">
              <div class="w-10 h-10 border-3 border-[#22d3ee]/20 border-t-[#22d3ee] rounded-full animate-[spin_0.8s_linear_infinite] mb-3"></div>
              <p class="text-sm text-white/90">加载中...</p>
            </div>
          </motion.div>

          <div id="content-section" class="mb-8" style="scroll-margin-top: 120px">
            <div class="flex items-center gap-3 mb-5 flex-wrap">
              <h2 class="text-lg text-white m-0 font-semibold">最近上传</h2>
              <span class="text-[13px] text-white/90">共 {{ total }} 条</span>
            </div>

            <div class="grid gap-4 transition-opacity duration-300 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4" :class="{ 'opacity-30 pointer-events-none': isLoading }">
              <div
                v-for="content in contents"
                :key="content.id"
                @click="goToDetail(content)"
                class="glass-card overflow-hidden cursor-pointer transition-all duration-300 hover:-translate-y-1 hover:shadow-[0_12px_32px_rgba(0,0,0,0.3)] hover:border-white/30"
              >
                <div class="relative w-full pt-[75%] bg-white/5 overflow-hidden">
                  <template v-if="content.type === 'image' || content.type === 'video' || content.type === 'link'">
                    <img :src="getImageUrl(content.thumb)" :alt="content.title" loading="lazy" @load="onImageLoad" class="absolute top-0 left-0 w-full h-full object-cover" />
                  </template>
                  <div v-if="content.type === 'video'" class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-12 h-12 bg-black/60 rounded-full flex items-center justify-center">
                    <svg class="w-5 h-5 text-white ml-0.5" viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>
                  </div>
                  <div v-if="content.type === 'link'" class="absolute top-2 right-2 w-8 h-8 bg-gradient-to-br from-[#22d3ee] to-[#38bdf8] rounded-full flex items-center justify-center">
                    <svg class="w-4 h-4 text-[#082f49]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                    </svg>
                  </div>
                  <template v-if="content.type === 'text'">
                    <div class="absolute top-0 left-0 w-full h-full p-4 flex items-center justify-center text-center">
                      <p class="text-[13px] text-white/85 line-clamp-4">{{ getPreviewText(content.text || '暂无内容') }}</p>
                    </div>
                  </template>
                </div>
                <div class="p-3">
                  <h3 class="text-sm text-white m-0 mb-2 font-semibold whitespace-nowrap overflow-hidden text-ellipsis">{{ content.title }}</h3>
                  <div class="flex items-center gap-2 flex-wrap mb-2">
                    <span class="flex items-center gap-1 text-xs text-white/90">
                      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                        <circle cx="12" cy="7" r="4" />
                      </svg>
                      {{ content.user?.username }}
                    </span>
                    <div class="flex gap-1">
                      <span v-for="tag in content.tags" :key="tag" class="px-2 py-0.5 bg-transparent border border-white/15 rounded text-[11px] text-white/85">{{ tag }}</span>
                    </div>
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="px-2.5 py-1 rounded-full text-[11px] font-medium" :class="{
                      'bg-[#fbbf24]/15 text-[#fbbf24]': content.type === 'video',
                      'bg-[#34d399]/15 text-[#34d399]': content.type === 'image',
                      'bg-[#38bdf8]/15 text-[#38bdf8]': content.type === 'link',
                      'bg-[#22d3ee]/15 text-[#22d3ee]': content.type === 'text'
                    }">
                      {{ content.type === 'video' ? '视频' : content.type === 'image' ? '图片' : content.type === 'link' ? '链接' : '文字' }}
                    </span>
                    <span class="flex items-center gap-1 text-xs text-white/90">
                      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                        <circle cx="12" cy="12" r="3"/>
                      </svg>
                      {{ content.view_count }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="isLoading" class="relative inset-0 flex flex-col items-center justify-center bg-[#082f49]/80 z-10">
              <div class="w-10 h-10 border-3 border-[#22d3ee]/20 border-t-[#22d3ee] rounded-full animate-[spin_0.8s_linear_infinite] mb-3"></div>
              <p class="text-sm text-white/90">加载中...</p>
            </div>
            <div v-if="!isLoading && contents.length === 0" class="text-center py-12 px-4 text-white/85">
              <svg class="w-16 h-16 mx-auto mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
              <p>暂无内容</p>
            </div>
          </div>
        </motion.div>

        <div v-if="totalPages > 1" class="flex flex-wrap justify-center items-center gap-3 pt-5 border-t border-white/10">
          <button @click="goToPage(1)" :disabled="page <= 1 || isLoading" class="flex items-center justify-center px-3.5 py-2.5 bg-white/10 border border-white/20 rounded-lg text-white/95 cursor-pointer transition-all duration-200 hover:border-[#22d3ee] hover:text-[#22d3ee] disabled:opacity-50 disabled:cursor-not-allowed">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 19l-7-7 7-7" /></svg>
          </button>
          <button @click="goToPage(page - 1)" :disabled="page <= 1 || isLoading" class="flex items-center justify-center px-3.5 py-2.5 bg-white/10 border border-white/20 rounded-lg text-white/95 cursor-pointer transition-all duration-200 hover:border-[#22d3ee] hover:text-[#22d3ee] disabled:opacity-50 disabled:cursor-not-allowed">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 19l-7-7 7-7" /></svg>
          </button>
          <span class="text-[13px] text-white/90">第 {{ page }} / {{ totalPages }} 页</span>
          <button @click="goToPage(page + 1)" :disabled="page >= totalPages || isLoading" class="flex items-center justify-center px-3.5 py-2.5 bg-white/10 border border-white/20 rounded-lg text-white/95 cursor-pointer transition-all duration-200 hover:border-[#22d3ee] hover:text-[#22d3ee] disabled:opacity-50 disabled:cursor-not-allowed">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 5l7 7-7 7" /></svg>
          </button>
          <button @click="goToPage(totalPages)" :disabled="page >= totalPages || isLoading" class="flex items-center justify-center px-3.5 py-2.5 bg-white/10 border border-white/20 rounded-lg text-white/95 cursor-pointer transition-all duration-200 hover:border-[#22d3ee] hover:text-[#22d3ee] disabled:opacity-50 disabled:cursor-not-allowed">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 5l7 7-7 7" /></svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.glass-card {
  backdrop-filter: blur(var(--theme-blur, 2px));
  -webkit-backdrop-filter: blur(var(--theme-blur, 2px));
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 12px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
