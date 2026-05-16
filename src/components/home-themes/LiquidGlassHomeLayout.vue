<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { motion } from 'motion-v'
import logoImg from '@/assets/logo.webp'
import bgImg from '@/assets/bg.webp'
import PollComponent from '@/components/PollComponent.vue'
import { useHomeLogic } from '@/composables/useHomeLogic'
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

  const glassElements = document.querySelectorAll('.glass-card, .poll-card, .poll-component, .glass-search-bar')
  glassElements.forEach(el => {
    const htmlEl = el as HTMLElement
    ;(htmlEl.style as any).backdropFilter = 'url(#liquid-glass) blur(4px) contrast(1.05) brightness(1.08)'
    ;(htmlEl.style as any).webkitBackdropFilter = 'url(#liquid-glass) blur(4px) contrast(1.05) brightness(1.08)'
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
})

onUnmounted(() => {
  glassObserver?.disconnect()
})
</script>

<template>
  <div class="glass-home-wrapper">
    <svg style="position: absolute; width: 0; height: 0;">
      <filter id="liquid-glass">
        <feImage id="liquid-glass-map" result="map" />
        <feDisplacementMap in="SourceGraphic" in2="map" xChannelSelector="R" yChannelSelector="G" scale="0" />
      </filter>
    </svg>

    <div class="glass-home-container">
      <header class="glass-header">
        <div class="window-controls">
          <span class="control control-close"></span>
          <span class="control control-minimize"></span>
          <span class="control control-maximize"></span>
        </div>
        <span class="header-text">小泉动漫二创站</span>
      </header>

      <div class="glass-content">
        <section class="hero-glass">
          <img :src="logoImg" alt="小泉动漫" class="hero-img max-w-full h-auto max-h-[80px] sm:max-h-[120px] mx-auto mb-2" />
          <button @click="goToEasterEgg" class="glass-btn-primary">
            <span class="btn-icon">🎉</span>
            <span>发现精彩</span>
          </button>
        </section>

        <section class="search-glass">
          <div class="glass-search-bar">
            <input
              v-model="searchKeyword"
              type="text"
              placeholder="搜索..."
              @keyup.enter="handleSearch"
              class="glass-input"
            />
            <button @click="handleSearch" class="glass-search-btn">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="m21 21-4.35-4.35"/>
              </svg>
            </button>
          </div>
        </section>

        <section class="poll-glass">
          <PollComponent />
        </section>

        <section class="filter-glass glass-card">
          <div class="filter-row">
            <span class="filter-title">类型</span>
            <div class="filter-pills">
              <button
                v-for="type in contentTypes"
                :key="type"
                @click="selectType(type)"
                class="pill"
                :class="{ 'pill-active': selectedTypes.includes(type) }"
              >
                {{ contentTypeLabels[type] }}
              </button>
            </div>
          </div>
          <div class="filter-row">
            <span class="filter-title">标签</span>
            <div class="filter-pills">
              <button
                v-for="tag in sortedTags"
                :key="tag"
                @click="selectTag(tag)"
                class="pill"
                :class="{ 'pill-active': selectedTags.includes(tag) }"
              >
                {{ tag }}
              </button>
            </div>
          </div>
        </section>

        <motion.div
          class="main-content"
          :animate="{ height: 'auto', transition: { duration: 0.3, ease: 'easeOut' } }"
        >
          <motion.div
            id="recommend-section"
            class="content-section"
            :initial="{ opacity: 1, y: 0 }"
            :animate="{
              opacity: swapSections ? 0 : 1,
              y: swapSections ? -80 : 0,
              height: swapSections ? 0 : 'auto',
              transition: { duration: 0.3, ease: 'easeOut' },
            }"
            style="overflow: hidden"
          >
            <div class="section-head">
              <h2 class="section-name">推荐内容</h2>
              <div class="section-controls">
                <span class="section-desc">精选推荐</span>
                <button @click="refreshRecommend" class="glass-btn-sm" :disabled="isRecommendLoading">
                  <svg class="spin-icon" :class="{ 'spinning': isRecommendLoading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="23 4 23 10 17 10" />
                    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
                  </svg>
                  {{ isRecommendLoading ? '加载中' : '刷新' }}
                </button>
              </div>
              <span v-if="recommendHint" class="hint">{{ recommendHint }}</span>
            </div>

            <div class="grid-recommend" :class="{ 'dimmed': isRecommendLoading }">
              <div
                v-for="content in recommendContents"
                :key="content.id"
                @click="goToDetail({ ...content, type: content.type, audit_status: 'approved' } as unknown as Content)"
                class="glass-card recommend-item"
              >
                <div class="img-wrap">
                  <img :src="getImageUrl(content.thumb)" :alt="content.title" loading="lazy" @load="onImageLoad" />
                  <div v-if="content.type === 'video'" class="play-btn">
                    <svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>
                  </div>
                </div>
                <div class="info-wrap">
                  <h3 class="item-title">{{ content.title }}</h3>
                  <div class="item-meta">
                    <span class="author">{{ content.user.username }}</span>
                    <div class="tag-list">
                      <span v-for="tag in content.tags.slice(0, 2)" :key="tag" class="tag">{{ tag }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="isRecommendLoading" class="loader">
              <div class="loader-ring"></div>
              <p>加载中...</p>
            </div>
          </motion.div>

          <div id="content-section" class="content-section" style="scroll-margin-top: 120px">
            <div class="section-head">
              <h2 class="section-name">最近上传</h2>
              <span class="section-desc">共 {{ total }} 条</span>
            </div>

            <div class="grid-content" :class="{ 'dimmed': isLoading }">
              <div
                v-for="content in contents"
                :key="content.id"
                @click="goToDetail(content)"
                class="glass-card content-item"
              >
                <div class="img-wrap">
                  <template v-if="content.type === 'image' || content.type === 'video' || content.type === 'link'">
                    <img :src="getImageUrl(content.thumb)" :alt="content.title" loading="lazy" @load="onImageLoad" />
                  </template>
                  <div v-if="content.type === 'video'" class="play-btn">
                    <svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>
                  </div>
                  <div v-if="content.type === 'link'" class="link-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                    </svg>
                  </div>
                  <template v-if="content.type === 'text'">
                    <div class="text-preview">
                      <p>{{ getPreviewText(content.text || '暂无内容') }}</p>
                    </div>
                  </template>
                </div>
                <div class="info-wrap">
                  <h3 class="item-title">{{ content.title }}</h3>
                  <div class="item-meta">
                    <span class="author">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                        <circle cx="12" cy="7" r="4" />
                      </svg>
                      {{ content.user?.username }}
                    </span>
                    <div class="tag-list">
                      <span v-for="tag in content.tags" :key="tag" class="tag tag-outline">{{ tag }}</span>
                    </div>
                  </div>
                  <div class="item-footer">
                    <span class="type-tag" :class="`type-${content.type}`">
                      {{ content.type === 'video' ? '视频' : content.type === 'image' ? '图片' : content.type === 'link' ? '链接' : '文字' }}
                    </span>
                    <span class="view-count">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                        <circle cx="12" cy="12" r="3"/>
                      </svg>
                      {{ content.view_count }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="isLoading" class="loader">
              <div class="loader-ring"></div>
              <p>加载中...</p>
            </div>
            <div v-if="!isLoading && contents.length === 0" class="empty">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
              <p>暂无内容</p>
            </div>
          </div>
        </motion.div>

        <div v-if="totalPages > 1" class="pagination-glass">
          <button @click="goToPage(1)" :disabled="page <= 1 || isLoading" class="page-btn-glass">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 19l-7-7 7-7" /></svg>
          </button>
          <button @click="goToPage(page - 1)" :disabled="page <= 1 || isLoading" class="page-btn-glass">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 19l-7-7 7-7" /></svg>
          </button>
          <span class="page-text">第 {{ page }} / {{ totalPages }} 页</span>
          <button @click="goToPage(page + 1)" :disabled="page >= totalPages || isLoading" class="page-btn-glass">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 5l7 7-7 7" /></svg>
          </button>
          <button @click="goToPage(totalPages)" :disabled="page >= totalPages || isLoading" class="page-btn-glass">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 5l7 7-7 7" /></svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.glass-home-wrapper {
  min-height: 100vh;
  background: url('@/assets/bg.webp') center/cover no-repeat fixed;
  background-color: #082f49;
  padding: 16px;
  display: flex;
  justify-content: center;
}

.glass-home-container {
  width: 100%;
  max-width: 1200px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.glass-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 20px;
  background: linear-gradient(to bottom, rgba(255,255,255,0.1), rgba(255,255,255,0.04));
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.window-controls {
  display: flex;
  gap: 8px;
}

.control {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.control-close { background: #ff5f57; }
.control-minimize { background: #febc2e; }
.control-maximize { background: #28c840; }

.header-text {
  font-size: 13px;
  color: #ffffff;
  font-weight: 500;
}

.glass-content {
  padding: 24px;
  color: rgba(255, 255, 255, 0.9);
  text-shadow: 
    -1px -1px 0 rgba(0, 0, 0, 0.5),
    1px -1px 0 rgba(0, 0, 0, 0.5),
    -1px 1px 0 rgba(0, 0, 0, 0.5),
    1px 1px 0 rgba(0, 0, 0, 0.5);
}

.hero-glass {
  text-align: center;
  margin-bottom: 32px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.hero-img {
  display: block;
}

.glass-btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  color: #ffffff;
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 24px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
}

.glass-btn-primary:hover {
  transform: scale(1.05);
  background: rgba(255, 255, 255, 0.25);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
}

.search-glass {
  margin-bottom: 24px;
}

.glass-search-bar {
  display: flex;
  max-width: 600px;
  margin: 0 auto;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  overflow: hidden;
}

.glass-input {
  flex: 1;
  padding: 12px 16px;
  background: transparent;
  border: none;
  color: #ffffff;
  font-size: 14px;
  outline: none;
}

.glass-input::placeholder {
  color: rgba(255, 255, 255, 0.85);
}

.glass-search-btn {
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: none;
  color: #ffffff;
  cursor: pointer;
  transition: all 0.2s ease;
}

.glass-search-btn:hover {
  background: rgba(255, 255, 255, 0.25);
}

.glass-search-btn svg {
  width: 18px;
  height: 18px;
}

.poll-glass {
  margin-bottom: 24px;
}

.filter-glass {
  padding: 16px;
  border-radius: 12px;
  margin-bottom: 32px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-title {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.9);
  font-weight: 500;
  min-width: 40px;
}

.filter-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pill {
  padding: 6px 14px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 20px;
  color: rgba(255, 255, 255, 0.95);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.pill:hover {
  border-color: #34d399;
  color: #34d399;
}

.pill-active {
  background: rgba(52, 211, 153, 0.2);
  border-color: rgba(52, 211, 153, 0.4);
  color: #34d399;
  font-weight: 500;
}

.main-content {
  min-height: 100px;
}

.content-section {
  margin-bottom: 32px;
}

.section-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.section-name {
  font-size: 18px;
  color: #ffffff;
  margin: 0;
  font-weight: 600;
}

.section-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
}

.section-desc {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.9);
}

.glass-btn-sm {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.95);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.glass-btn-sm:hover {
  border-color: #22d3ee;
  color: #22d3ee;
}

.spin-icon {
  width: 14px;
  height: 14px;
}

.spinning {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.hint {
  font-size: 13px;
  color: #f87171;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.grid-recommend,
.grid-content {
  display: grid;
  gap: 16px;
  transition: opacity 0.3s ease;
}

.grid-recommend {
  grid-template-columns: repeat(2, 1fr);
}

.grid-content {
  grid-template-columns: repeat(1, 1fr);
}

@media (min-width: 640px) {
  .grid-content { grid-template-columns: repeat(2, 1fr); }
}

@media (min-width: 1024px) {
  .grid-recommend { grid-template-columns: repeat(4, 1fr); }
  .grid-content { grid-template-columns: repeat(3, 1fr); }
}

@media (min-width: 1280px) {
  .grid-content { grid-template-columns: repeat(4, 1fr); }
}

.dimmed {
  opacity: 0.3;
  pointer-events: none;
}

.glass-card {
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 12px;
}

.recommend-item,
.content-item {
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s ease;
}

.recommend-item:hover,
.content-item:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.3);
  border-color: rgba(255, 255, 255, 0.3);
}

.img-wrap {
  position: relative;
  width: 100%;
  padding-top: 75%;
  background: rgba(255, 255, 255, 0.05);
  overflow: hidden;
}

.img-wrap img {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.play-btn {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 48px;
  height: 48px;
  background: rgba(0, 0, 0, 0.6);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.play-btn svg {
  width: 20px;
  height: 20px;
  color: white;
  margin-left: 2px;
}

.link-icon {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #22d3ee, #38bdf8);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.link-icon svg {
  width: 16px;
  height: 16px;
  color: #082f49;
}

.text-preview {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.text-preview p {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.85);
  line-clamp: 4;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.info-wrap {
  padding: 12px 16px;
}

.item-title {
  font-size: 14px;
  color: #ffffff;
  margin: 0 0 8px 0;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.author {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.9);
}

.author svg {
  width: 14px;
  height: 14px;
}

.tag-list {
  display: flex;
  gap: 4px;
}

.tag {
  padding: 2px 8px;
  background: rgba(251, 191, 36, 0.15);
  border-radius: 4px;
  font-size: 11px;
  color: #fbbf24;
}

.tag-outline {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.85);
}

.item-footer {
  display: flex;
  align-items: center;
  gap: 8px;
}

.type-tag {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
}

.type-video {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
}

.type-image {
  background: rgba(52, 211, 153, 0.15);
  color: #34d399;
}

.type-link {
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
}

.type-text {
  background: rgba(34, 211, 238, 0.15);
  color: #22d3ee;
}

.view-count {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.9);
}

.view-count svg {
  width: 14px;
  height: 14px;
}

.loader {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(8, 47, 73, 0.8);
  z-index: 10;
}

.loader-ring {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(34, 211, 238, 0.2);
  border-top-color: #22d3ee;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 12px;
}

.loader p {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.9);
}

.empty {
  text-align: center;
  padding: 48px 16px;
  color: rgba(255, 255, 255, 0.85);
}

.empty svg {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
}

.pagination-glass {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding-top: 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.page-btn-glass {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.95);
  cursor: pointer;
  transition: all 0.2s ease;
}

.page-btn-glass:hover:not(:disabled) {
  border-color: #22d3ee;
  color: #22d3ee;
}

.page-btn-glass:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-btn-glass svg {
  width: 16px;
  height: 16px;
}

.page-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.9);
}
</style>
