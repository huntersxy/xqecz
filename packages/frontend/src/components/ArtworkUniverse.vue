<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { IconClose, IconPlus, IconMinus, IconExpand, IconArrowRight, IconForward, IconSearch, IconImage } from '@arco-design/web-vue/es/icon'
import { loadPlanetImage } from '@/utils/artworkPlanet'
import { UniverseLayout, type UniverseArtwork, type UniverseNode, type UniversePhase, type PlanetOrigin } from '@/utils/artworkUniverse'
import type { createUniverseScene } from '@/utils/universeScene'

const props = defineProps<{ artworks: readonly UniverseArtwork[]; origin: PlanetOrigin }>()
const emit = defineEmits<{ closed: []; select: [id: number] }>()
const root = ref<HTMLElement>()
const stage = ref<HTMLElement>()
const sprite = ref<HTMLElement>()
const montage = ref<HTMLElement>()
const surface = ref<HTMLElement>()
const mapCanvas = ref<HTMLCanvasElement>()
const phase = ref<UniversePhase>('flight')
const selected = ref<UniverseNode>()
const query = ref('')
const failed = ref(false)
const montageFinished = ref(false)
const reduced = window.matchMedia('(prefers-reduced-motion: reduce)')
const layout = new UniverseLayout(Math.floor(Math.random() * 0x7fffffff))
const nodes = ref(layout.update(props.artworks))
const imageUrls = ref<Record<number, string>>({})
const settledImages = ref(new Set<number>())
const rowCount = innerWidth < 700 ? 4 : 5
const columns = Math.min(12, Math.max(6, Math.ceil(innerWidth / 230) + 2))
const montageItems = nodes.value.filter(node => node.artwork.thumb).slice(0, Math.min(48, rowCount * columns))
const rows = Array.from({ length: rowCount }, (_, row) => Array.from({ length: columns }, (_, col) => montageItems[(row * columns + col) % Math.max(1, montageItems.length)]).filter((node): node is UniverseNode => !!node))
const results = computed(() => query.value.trim() ? nodes.value.filter(node => `${node.artwork.title || ''} ${node.artwork.id}`.toLowerCase().includes(query.value.trim().toLowerCase())).slice(0, 8) : [])
const isMap = computed(() => phase.value === 'map')
const status = computed(() => ({ flight: '星球正在飞行', unfold: '作品正在展开', marquee: '作品阵列', collapse: '星系正在形成', map: '星图已展开', closing: '正在返回星球' })[phase.value])
const originStyle = { width: `${props.origin.width}px`, height: `${props.origin.height}px`, backgroundImage: `url("${props.origin.image}")` }
let universe: ReturnType<typeof createUniverseScene> | undefined
let enginePromise: Promise<void>
let disposed = false
// 每次跳过/关闭都会使旧动画序列失效；await 后必须核对版本，防止已关闭的星图复活。
let version = 0
const animations = new Set<Animation>()
const abort = new AbortController()
const previousFocus = document.activeElement as HTMLElement | null
const background = document.getElementById('app')
const wasInert = background?.inert

function duration(value: number) { return reduced.matches ? 1 : value }
function animate(element: HTMLElement, frames: Keyframe[], ms: number, delay = 0) {
  const animation = element.animate(frames, { duration: duration(ms), delay: duration(delay), easing: 'cubic-bezier(.22,.75,.2,1)', fill: 'forwards' })
  animations.add(animation)
  if (document.hidden) animation.pause()
  return animation
}
function wait(ms: number, current = version) {
  return new Promise<boolean>(resolve => {
    let remaining = duration(ms)
    let last = performance.now()
    function tick(now: number) {
      if (disposed || version !== current) { resolve(false); return }
      if (!document.hidden) remaining -= Math.min(now - last, 100)
      last = now
      if (remaining <= 0) resolve(true)
      else requestAnimationFrame(tick)
    }
    requestAnimationFrame(tick)
  })
}
function originTransform(scale = 1) {
  const { left, top, width, height } = props.origin
  return `translate(calc(-50% + ${left + width / 2 - innerWidth / 2}px), calc(-50% + ${top + height / 2 - innerHeight / 2}px)) scale(${scale})`
}
function contextLost() { failed.value = true; universe?.setActive(false) }
async function prepareScene() {
  try {
    const { createUniverseScene } = await import('@/utils/universeScene')
    if (disposed || !mapCanvas.value || phase.value === 'closing') return
    universe = createUniverseScene(mapCanvas.value, node => { selected.value = node })
    universe.update(nodes.value)
    universe.fit()
  } catch { failed.value = true }
}
async function preloadMontage() {
  const queue = [...new Map(montageItems.map(node => [node.artwork.id, node])).values()]
  async function worker() {
    while (queue.length && !abort.signal.aborted) {
      const node = queue.shift()!
      const image = await loadPlanetImage(node.artwork.thumb, abort.signal)
      if (image && !abort.signal.aborted) imageUrls.value[node.artwork.id] = image.src
      if (!abort.signal.aborted) settledImages.value.add(node.artwork.id)
    }
  }
  await Promise.all([worker(), worker(), worker()])
}
async function openSequence() {
  const current = version
  animate(sprite.value!, [{ transform: originTransform(), opacity: 1 }, { transform: 'translate(-50%, -50%) scale(1.8)', opacity: 1 }], 900)
  if (!await wait(920, current)) return
  phase.value = 'unfold'
  if (surface.value) animate(surface.value, [
    { width: '320px', height: '320px', borderRadius: '50%', opacity: 0, transform: 'translate(-50%, -50%) perspective(1200px) rotateX(25deg)' },
    { offset: .4, width: '76vw', height: '48vh', borderRadius: '18%', opacity: 1, transform: 'translate(-50%, -50%) perspective(1200px) rotateX(10deg)' },
    { width: '130vw', height: '110vh', borderRadius: '0', opacity: 0, transform: 'translate(-50%, -50%) perspective(1200px) rotateX(0deg)' },
  ], 1500)
  animate(sprite.value!, [{ transform: 'translate(-50%, -50%) scale(1.8)', opacity: 1 }, { transform: 'translate(-50%, -50%) scale(5.5)', opacity: 0 }], 1500)
  if (montage.value) animate(montage.value, [{ clipPath: 'circle(115px at 50% 50%)', transform: 'scale(.42)', opacity: .3 }, { clipPath: 'circle(150% at 50% 50%)', transform: 'scale(1)', opacity: 1 }], 1500)
  if (!await wait(1520, current)) return
  phase.value = 'marquee'
  if (!await wait(3400, current)) return
  await formMap()
}
async function formMap() {
  if (phase.value === 'map' || phase.value === 'collapse' || phase.value === 'closing') return
  const current = ++version
  phase.value = 'collapse'
  sprite.value?.getAnimations().forEach(animation => animation.cancel())
  if (sprite.value) sprite.value.style.opacity = '0'
  surface.value?.getAnimations().forEach(animation => animation.cancel())
  if (surface.value) surface.value.style.opacity = '0'
  await enginePromise
  if (disposed || current !== version) return
  universe?.setActive(true)
  montage.value?.getAnimations().forEach(animation => animation.cancel())
  if (montage.value) { montage.value.style.clipPath = 'none'; montage.value.style.opacity = '1'; montage.value.style.transform = 'none' }
  const tiles = [...(montage.value?.querySelectorAll<HTMLElement>('.au-tile') || [])]
  const rects = tiles.map(tile => tile.getBoundingClientRect())
  montage.value?.querySelectorAll<HTMLElement>('.au-row').forEach(row => { row.style.animationPlayState = 'paused' })
  tiles.forEach((tile, index) => {
    const rect = rects[index]!
    const node = nodes.value.find(item => item.artwork.id === Number(tile.dataset.id))
    const point = node && universe?.project(node)
    const x = point ? point.x - rect.left - rect.width / 2 : innerWidth / 2 - rect.left - rect.width / 2
    const y = point ? point.y - rect.top - rect.height / 2 : innerHeight / 2 - rect.top - rect.height / 2
    animate(tile, [{ transform: 'translate(0, 0) scale(1)', borderRadius: '6px', opacity: 1 }, { transform: `translate(${x}px, ${y}px) scale(${point ? Math.min(.25, point.diameter / rect.width) : .02})`, borderRadius: '50%', opacity: 0 }], 1150, (index % columns) * 24)
  })
  if (!await wait(1480, current)) return
  phase.value = 'map'
  montageFinished.value = true
  abort.abort()
  mapCanvas.value?.focus({ preventScroll: true })
}
async function close() {
  if (phase.value === 'closing' || disposed) return
  ++version
  phase.value = 'closing'
  abort.abort()
  universe?.setActive(false)
  animations.forEach(animation => animation.cancel())
  sprite.value!.style.opacity = '0'
  if (stage.value) animate(stage.value, [{ transform: 'scale(1) rotate(0deg)', opacity: 1, filter: 'brightness(1)' }, { transform: 'scale(.002) rotate(28deg)', opacity: 0, filter: 'brightness(4)' }], 650)
  if (!await wait(640)) return
  animate(sprite.value!, [{ transform: 'translate(-50%, -50%) scale(.04)', opacity: 1, filter: 'brightness(3)' }, { transform: originTransform(), opacity: 1, filter: 'brightness(1)' }], 760)
  animate(root.value!, [{ backgroundColor: 'rgba(17,16,26,1)' }, { backgroundColor: 'rgba(17,16,26,0)' }], 760)
  if (await wait(780)) emit('closed')
}
function focusNode(node: UniverseNode) { selected.value = node; query.value = ''; universe?.focus(node) }
function openSelected() {
  if (!selected.value || phase.value !== 'map') return
  emit('select', selected.value.artwork.id)
}
function key(event: KeyboardEvent) {
  if (event.key === 'Escape') { event.preventDefault(); void close() }
  if (event.key === 'Enter' && event.target === mapCanvas.value) openSelected()
  if (event.key !== 'Tab') return
  const focusable = [...root.value!.querySelectorAll<HTMLElement>('button:not([disabled]), input, canvas[tabindex], a[href]')].filter(el => el.getClientRects().length > 0)
  const first = focusable[0]; const last = focusable.at(-1)
  if (event.shiftKey && (document.activeElement === first || !root.value!.contains(document.activeElement))) { event.preventDefault(); last?.focus() }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() }
}
function visibility() {
  animations.forEach(animation => { if (animation.playState === 'running' && document.hidden) animation.pause(); else if (!document.hidden && animation.playState === 'paused') animation.play() })
}
function motionChange() { if (reduced.matches) void formMap() }
watch(() => props.artworks, items => { nodes.value = layout.update(items); universe?.update(nodes.value) })
onMounted(async () => {
  document.addEventListener('keydown', key)
  if (background) background.inert = true
  document.addEventListener('visibilitychange', visibility)
  reduced.addEventListener('change', motionChange)
  root.value?.querySelector<HTMLButtonElement>('.au-close')?.focus({ preventScroll: true })
  enginePromise = prepareScene()
  void preloadMontage()
  await nextTick()
  if (reduced.matches) await formMap()
  else void openSequence()
})
onBeforeUnmount(() => {
  // Teleport 的资源不随页面 DOM 自动释放；同时还原焦点和背景 inert，防止返回后无法点击。
  disposed = true; version++; abort.abort()
  animations.forEach(animation => animation.cancel())
  universe?.dispose()
  document.removeEventListener('keydown', key)
  document.removeEventListener('visibilitychange', visibility)
  reduced.removeEventListener('change', motionChange)
  if (background) background.inert = wasInert ?? false
  if (previousFocus?.isConnected) previousFocus.focus({ preventScroll: true })
})
</script>

<template>
  <Teleport to="body">
    <div ref="root" class="au-root creative-theme" :class="`au-phase-${phase}`" role="dialog" aria-modal="true" aria-label="小泉宇宙星图" :data-phase="phase">
      <span class="au-sr" role="status" aria-live="polite">{{ status }}</span>
      <div ref="stage" class="au-stage">
        <canvas ref="mapCanvas" class="au-map" :class="{ 'is-visible': montageFinished || phase === 'collapse' }" tabindex="0" role="img" aria-label="全作品星图，可拖动和缩放，点击选择星球" @dblclick="openSelected" @webglcontextlost.prevent="contextLost" />
        <div v-if="!montageFinished" ref="montage" class="au-montage" :class="{ 'is-unfolding': phase !== 'flight' }" aria-hidden="true">
          <div v-for="(row, rowIndex) in rows" :key="rowIndex" class="au-row" :class="{ reverse: rowIndex % 2 }" :style="{ '--row-duration': `${34 + rowIndex * 5}s` }">
            <div v-for="copy in 2" :key="copy" class="au-row-group">
              <div v-for="(node, index) in row" :key="index" class="au-tile" :data-id="node.artwork.id" :style="{ '--planet-color': `hsl(${node.hue} 24% 27%)` }">
                <img v-if="imageUrls[node.artwork.id]" :src="imageUrls[node.artwork.id]" alt="" draggable="false" />
                <IconImage v-else-if="settledImages.has(node.artwork.id)" class="au-tile-missing" />
                <span v-else class="au-tile-loading"></span>
                <span class="au-tile-title">{{ node.artwork.title || `星球 ${node.artwork.id}` }}</span>
              </div>
            </div>
          </div>
        </div>
        <div v-if="isMap && failed" class="au-fallback">
          <button v-for="node in nodes" :key="node.artwork.id" @click="selected = node">{{ node.artwork.title || `星球 ${node.artwork.id}` }}</button>
        </div>
      </div>
      <div ref="surface" class="au-surface" :style="{ backgroundImage: `url('${origin.atlas || origin.image}')` }" aria-hidden="true"></div>
      <div ref="sprite" class="au-flight-planet" :style="originStyle" aria-hidden="true"></div>
      <header v-if="phase !== 'closing'" class="au-header">
        <div class="au-heading"><span>XQECZ / STAR ATLAS</span><h2>小泉星系</h2><p>{{ nodes.length }} 颗星球</p></div>
        <div class="au-header-actions">
          <div v-if="isMap" class="au-search">
            <IconSearch /><input v-model="query" type="search" aria-label="搜索星球" placeholder="寻找一颗星球" />
            <div v-if="query.trim()" class="au-search-results">
              <button v-for="node in results" :key="node.artwork.id" @click="focusNode(node)">{{ node.artwork.title || `星球 ${node.artwork.id}` }}</button>
              <span v-if="!results.length">没有找到对应的星球</span>
            </div>
          </div>
          <button v-if="!isMap && phase !== 'collapse'" type="button" class="au-icon" aria-label="跳过展开动画" title="进入星图" @click="formMap"><IconForward /></button>
          <button type="button" class="au-icon au-close" aria-label="关闭星图" title="关闭星图" @click="close"><IconClose /></button>
        </div>
      </header>
      <div v-if="isMap" class="au-map-controls">
        <button class="au-icon" aria-label="放大星图" title="放大" @click="universe?.zoom(1.4)"><IconPlus /></button>
        <button class="au-icon" aria-label="缩小星图" title="缩小" @click="universe?.zoom(1 / 1.4)"><IconMinus /></button>
        <button class="au-icon" aria-label="显示全部星球" title="全景" @click="universe?.fit()"><IconExpand /></button>
      </div>
      <div v-if="isMap && selected" class="au-selection">
        <span class="au-coordinate">COORDINATE / {{ selected.artwork.id }}</span>
        <h3>{{ selected.artwork.title || `星球 ${selected.artwork.id}` }}</h3>
        <button type="button" class="au-open" @click="openSelected">探访这颗星球<IconArrowRight /></button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.au-root { position: fixed; inset: 0; width: 100vw; z-index: 1100; overflow: hidden; background: #11101a; color: #f2edf4; isolation: isolate; }
.au-stage { position: absolute; inset: 0; transform-origin: center; }
.au-map { display: block; width: 100%; height: 100%; opacity: 0; transition: opacity 1.15s ease; cursor: grab; touch-action: none; }
.au-map:active { cursor: grabbing; }
.au-map.is-visible { opacity: 1; }
.au-flight-planet { position: fixed; left: 50%; top: 50%; transform: translate(-50%, -50%); border-radius: 50%; background-size: cover; background-position: center; opacity: 0; pointer-events: none; box-shadow: 0 0 24px #d1a6b233; }
.au-surface { position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%); width: 320px; height: 320px; border-radius: 50%; background-size: 100% 100%; opacity: 0; pointer-events: none; }
.au-montage { position: absolute; inset: 0; display: flex; flex-direction: column; justify-content: center; gap: 14px; opacity: 0; pointer-events: none; }
.au-row { display: flex; width: max-content; gap: 14px; animation: au-drift var(--row-duration) linear infinite; animation-play-state: paused; }
.is-unfolding .au-row { animation-play-state: running; }
.au-row.reverse { animation-direction: reverse; }
.au-row-group { display: flex; gap: 14px; }
.au-tile { position: relative; width: 220px; height: min(19vh, 210px); min-height: 110px; border-radius: 6px; overflow: hidden; background: var(--planet-color); flex-shrink: 0; will-change: transform; }
.au-tile img { width: 100%; height: 100%; object-fit: cover; mask-image: linear-gradient(90deg, transparent, black 4%, black 96%, transparent); }
.au-tile-title { position: absolute; inset: auto 0 0; padding: 20px 12px 10px; font-size: 12px; background: linear-gradient(transparent, #0009); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.au-tile-loading { position: absolute; left: calc(50% - 10px); top: calc(50% - 10px); width: 20px; height: 20px; border: 1px solid #ffffff30; border-top-color: #e0babb; border-radius: 50%; animation: au-loading 1.4s linear infinite; }
.au-tile-missing { position: absolute; width: 26px; height: 26px; left: calc(50% - 13px); top: calc(50% - 13px); color: #c8bdc5; opacity: .6; }
.au-header { position: absolute; top: 0; left: 0; width: 100%; padding: 28px 32px; display: flex; justify-content: space-between; align-items: flex-start; gap: 20px; pointer-events: none; background: linear-gradient(#11101ab8, transparent); }
.au-heading span, .au-coordinate { font-size: 10px; color: #c6a2ac; letter-spacing: 0; }
.au-heading h2 { font: 27px/1.6 var(--creative-title-font); color: #f5ebef; margin: 2px 0; }
.au-heading p { font-size: 12px; color: #9c96a6; }
.au-header-actions { display: flex; gap: 10px; align-items: center; pointer-events: auto; }
.au-icon { display: grid; place-items: center; width: 42px; height: 42px; flex-shrink: 0; background: #25222ee8; border: 1px solid #ffffff26; color: #f1e6eb; border-radius: 50%; cursor: pointer; }
.au-icon svg { width: 18px; height: 18px; }
.au-icon:hover { background: #473341; border-color: #d6a4b4; }
.au-search { position: relative; display: flex; align-items: center; gap: 8px; width: 230px; height: 42px; padding: 0 12px; border-bottom: 1px solid #ffffff40; background: #211e29dd; }
.au-search input { min-width: 0; width: 100%; color: #eee8ee; background: transparent; border: 0; outline: 0; font-size: 13px; }
.au-search input::placeholder { color: #aca3b2; }
.au-search-results { position: absolute; top: 48px; right: 0; width: 290px; max-height: 320px; overflow: auto; border: 1px solid #ffffff26; background: #25212b; padding: 6px; border-radius: 6px; }
.au-search-results button, .au-search-results span { display: block; width: 100%; color: #ece3eb; padding: 10px; font-size: 13px; text-align: left; background: transparent; border: 0; overflow-wrap: anywhere; }
.au-search-results button:hover { background: #43313c; cursor: pointer; }
.au-map-controls { position: absolute; right: 32px; bottom: 32px; display: flex; gap: 8px; }
.au-selection { position: absolute; left: 32px; bottom: 32px; width: min(340px, calc(100% - 150px)); padding: 16px 0; background: #11101ae0; border-top: 1px solid #cf9eaa70; }
.au-selection h3 { margin: 8px 0 14px; font: 20px/1.5 var(--creative-title-font); color: #f1e4eb; overflow-wrap: anywhere; max-height: 90px; overflow: auto; }
.au-open { display: flex; align-items: center; gap: 12px; border: 0; padding: 8px 0; background: transparent; color: #eab8c5; cursor: pointer; }
.au-fallback { position: absolute; inset: 130px 24px 100px; overflow: auto; display: flex; flex-wrap: wrap; gap: 24px; align-content: start; }
.au-fallback button { width: 100px; height: 100px; border-radius: 50%; padding: 14px; font-size: 12px; color: #f7e9ef; border: 1px solid #b18b9e; background: #443347; overflow: hidden; cursor: pointer; }
.au-sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
@keyframes au-drift { to { transform: translateX(calc(-50% - 7px)); } }
@keyframes au-loading { to { transform: rotate(360deg); } }
@media (max-width: 700px) { .au-header { padding: 18px 16px; } .au-heading h2 { font-size: 23px; } .au-heading span { font-size: 8px; } .au-search { width: 130px; } .au-search-results { width: 240px; } .au-map-controls { right: 16px; bottom: 20px; flex-direction: column; } .au-selection { left: 16px; bottom: 20px; } .au-tile { width: 180px; height: 20vh; } }
</style>
