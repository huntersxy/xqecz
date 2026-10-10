<script setup lang="ts">
import { defineAsyncComponent, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { IconPause, IconPlayArrow } from '@arco-design/web-vue/es/icon'
import heroArt from '@/assets/bg.webp'
import { loadPlanetImage, paintPlanetTexture, planetSlotSamples, PlanetArtworkDeck, PLANET_SLOTS, PLANET_TEXTURE_HEIGHT, PLANET_TEXTURE_WIDTH, type PlanetArtwork } from '@/utils/artworkPlanet'
import type { CanvasTexture, Mesh, MeshStandardMaterial, Scene, OrthographicCamera, WebGLRenderer, Vector3 } from 'three'
import type { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import type { PlanetOrigin, UniverseArtwork } from '@/utils/artworkUniverse'

const props = defineProps<{ artworks: readonly (PlanetArtwork & UniverseArtwork)[] }>()
const emit = defineEmits<{ select: [id: number] }>()
const ArtworkUniverse = defineAsyncComponent(() => import('./ArtworkUniverse.vue'))
const universeOrigin = ref<PlanetOrigin>()
const host = ref<HTMLElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const ready = ref(false)
const paused = ref(false)
const dragging = ref(false)
let renderer: WebGLRenderer | undefined
let controls: OrbitControls | undefined
let scene: Scene
let camera: OrthographicCamera
let globe: Mesh
let texture: CanvasTexture
let material: MeshStandardMaterial
let direction: Vector3
let scratch: Vector3
let samples: Vector3[][] = []
let observer: IntersectionObserver | undefined
let desktop: MediaQueryList
let reducedMotion: MediaQueryList
let active = true
let inView = false
let initializing = false
let destroyed = false
let lostContext = false
let lastFrame = 0
let lastDraw = 0
let lastSwap = 0
let lastRefill = 0
let resumeAt = 0
let settling = 0
let pendingLoads = 0
let seeded = false
let lastLoadStart = -1000
let refillTimer: ReturnType<typeof setTimeout> | undefined
let loads = new AbortController()
const lifetime = new AbortController()
const deck = new PlanetArtworkDeck()
const resident: { url: string; image: HTMLImageElement }[] = []
const queue: { url: string; image: HTMLImageElement }[] = []
const loadingUrls = new Set<string>()
const failedUntil = new Map<string, number>()
const refreshed = new Set<number>()

function canRun() {
  // KeepAlive 停用、离屏、隐藏标签页和全屏星图均暂停渲染与补图；恢复只经 updateAnimation。
  return active && inView && desktop?.matches && !universeOrigin.value && !document.hidden && !destroyed && !lostContext
}

function explore() {
  if (universeOrigin.value || !host.value) return
  const rect = host.value.getBoundingClientRect()
  let image = heroArt
  let atlas: string | undefined
  if (renderer && ready.value) {
    renderer.render(scene, camera)
    try {
      image = renderer.domElement.toDataURL('image/png')
      atlas = (texture.image as HTMLCanvasElement).toDataURL('image/png')
    } catch { /* A packaged fallback also works without WebGL. */ }
  }
  universeOrigin.value = { left: rect.left, top: rect.top, width: rect.width, height: rect.height, image, atlas }
  updateAnimation()
}
function closeUniverse() { universeOrigin.value = undefined; updateAnimation() }
function selectUniverse(id: number) { closeUniverse(); emit('select', id) }

function refill() {
  if (!renderer || !ready.value || !canRun() || paused.value || dragging.value) return
  const delay = 800 - (performance.now() - lastLoadStart)
  if (delay > 0) {
    refillTimer ??= setTimeout(() => { refillTimer = undefined; refill() }, delay)
    return
  }
  // 两个并发请求、三个预备图、十二个球面贴片；内存与作品总数无关。
  const excluded = new Set([...resident.map(item => item.url), ...queue.map(item => item.url), ...loadingUrls])
  const now = performance.now()
  failedUntil.forEach((until, url) => { if (until > now) excluded.add(url); else failedUntil.delete(url) })
  while (pendingLoads < 2 && queue.length + pendingLoads < 3) {
    const url = deck.next(excluded)
    if (!url) break
    excluded.add(url)
    loadingUrls.add(url)
    pendingLoads++
    lastLoadStart = performance.now()
    const signal = loads.signal
    void loadPlanetImage(url, signal).then(image => {
      if (signal.aborted || destroyed) return
      if (image) {
        queue.push({ url, image })
        seedArtworks()
      }
      else {
        if (failedUntil.size >= 128) failedUntil.delete(failedUntil.keys().next().value!)
        failedUntil.set(url, performance.now() + 60000)
      }
    }).finally(() => {
      loadingUrls.delete(url)
      pendingLoads--
      if (!destroyed) refill()
    })
  }
}

function seedArtworks() {
  if (seeded || !ready.value || dragging.value || queue.length < 2 || !canRun()) return
  const first = queue.splice(0)
  for (let index = 0; index < PLANET_SLOTS; index++) resident[index] = first[index % first.length]!
  seeded = true
  paintPlanetTexture(texture.image as HTMLCanvasElement, resident.map(item => item.image))
  texture.needsUpdate = true
  renderer?.render(scene, camera)
}

function recycleBackside(time: number) {
  // 未凑齐首批作品时不局部替换，避免装饰底图失败后出现稀疏的 resident 数组。
  if (!seeded || dragging.value || paused.value || reducedMotion.matches || time < resumeAt || time - lastSwap < 1400 || !queue.length) return
  globe.updateWorldMatrix(true, false)
  camera.getWorldPosition(direction).normalize()
  for (let index = 0; index < PLANET_SLOTS; index++) {
    const hidden = samples[index]!.every(point => scratch.copy(point).transformDirection(globe.matrixWorld).dot(direction) < .02)
    if (!hidden) { refreshed.delete(index); continue }
    if (refreshed.has(index)) continue
    resident[index] = queue.shift()!
    refreshed.add(index)
    paintPlanetTexture(texture.image as HTMLCanvasElement, resident.map(item => item.image))
    texture.needsUpdate = true
    lastSwap = time
    refill()
    break
  }
}

function drawFrame(time: number) {
  if (!renderer || !canRun()) return
  const delta = lastFrame ? Math.min((time - lastFrame) / 1000, .1) : 0
  lastFrame = time
  if (!paused.value && !reducedMotion.matches && !dragging.value && time > resumeAt) globe.rotation.y += delta * Math.PI * 2 / 48
  controls?.update(delta)
  if (time - lastDraw < 1000 / 30) return
  lastDraw = time
  seedArtworks()
  recycleBackside(time)
  if (time - lastRefill > 5000) { lastRefill = time; refill() }
  renderer.render(scene, camera)
  if ((paused.value || reducedMotion.matches) && !dragging.value && time > settling) renderer.setAnimationLoop(null)
}

function updateAnimation() {
  if (!renderer) {
    if (canRun()) void initialize()
    return
  }
  lastFrame = 0
  lastDraw = 0
  renderer.setAnimationLoop(null)
  if (!canRun()) {
    clearTimeout(refillTimer)
    refillTimer = undefined
    loads.abort()
    loads = new AbortController()
    return
  }
  controls!.enableDamping = !reducedMotion.matches
  renderer.render(scene, camera)
  refill()
  if ((!paused.value && !reducedMotion.matches) || dragging.value || performance.now() < settling) renderer.setAnimationLoop(drawFrame)
}

function onDragStart() { dragging.value = true; updateAnimation() }
function onDragEnd() {
  dragging.value = false
  seedArtworks()
  resumeAt = performance.now() + 2000
  settling = performance.now() + 800
  updateAnimation()
}
function toggleRotation() { paused.value = !paused.value; updateAnimation() }
function rotateByKey(event: KeyboardEvent) {
  if (!globe || !controls || !['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(event.key)) return
  event.preventDefault()
  if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') globe.rotation.y += event.key === 'ArrowLeft' ? -.22 : .22
  else globe.rotation.x += event.key === 'ArrowUp' ? -.18 : .18
  resumeAt = performance.now() + 2000
  controls.update()
  updateAnimation()
}

async function initialize() {
  if (renderer || initializing || destroyed || !canvas.value) return
  initializing = true
  try {
    const [THREE, { OrbitControls }] = await Promise.all([import('three'), import('three/addons/controls/OrbitControls.js')])
    if (!canRun() || !canvas.value) return
    renderer = new THREE.WebGLRenderer({ canvas: canvas.value, alpha: true, antialias: true, powerPreference: 'low-power' })
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
    renderer.setSize(180, 180, false)
    renderer.outputColorSpace = THREE.SRGBColorSpace
    scene = new THREE.Scene()
    camera = new THREE.OrthographicCamera(-1.08, 1.08, 1.08, -1.08, .1, 10)
    camera.position.z = 4
    direction = new THREE.Vector3()
    scratch = new THREE.Vector3()
    samples = Array.from({ length: PLANET_SLOTS }, (_, index) => planetSlotSamples(index).map(point => new THREE.Vector3(...point)))
    const atlas = document.createElement('canvas')
    atlas.width = PLANET_TEXTURE_WIDTH
    atlas.height = PLANET_TEXTURE_HEIGHT
    texture = new THREE.CanvasTexture(atlas)
    texture.colorSpace = THREE.SRGBColorSpace
    texture.wrapS = THREE.RepeatWrapping
    texture.anisotropy = Math.min(renderer.capabilities.getMaxAnisotropy(), 4)
    material = new THREE.MeshStandardMaterial({ map: texture, roughness: 1, metalness: 0, emissiveMap: texture, emissive: '#ffffff', emissiveIntensity: .12 })
    globe = new THREE.Mesh(new THREE.SphereGeometry(1, 48, 32), material)
    scene.add(globe)
    scene.add(new THREE.AmbientLight('#fff5ef', 1.3))
    const light = new THREE.DirectionalLight('#fff6ed', 1.5)
    light.position.set(-3, 4, 5)
    scene.add(light)
    controls = new OrbitControls(camera, canvas.value)
    controls.enablePan = false
    controls.enableZoom = false
    controls.enableDamping = !reducedMotion.matches
    controls.dampingFactor = .12
    controls.rotateSpeed = .65
    controls.addEventListener('start', onDragStart)
    controls.addEventListener('end', onDragEnd)
    // 底图属于前端打包素材；它的失败不能阻断 ready、帧循环或后续作品补图。
    const fallback = await loadPlanetImage(heroArt, lifetime.signal)
    if (destroyed) return
    if (fallback) {
      resident.push(...Array.from({ length: PLANET_SLOTS }, () => ({ url: heroArt, image: fallback })))
      paintPlanetTexture(atlas, resident.map(item => item.image))
    } else {
      // A missing decorative asset must not prevent rotation or artwork loading.
      const context = atlas.getContext('2d')
      if (context) {
        context.fillStyle = '#f5d9e2'
        context.fillRect(0, 0, atlas.width, atlas.height)
      }
    }
    texture.needsUpdate = true
    ready.value = true
    updateAnimation()
  } catch {
    ready.value = false
    disposeScene()
  } finally { initializing = false }
}

function disposeScene() {
  controls?.dispose()
  renderer?.setAnimationLoop(null)
  globe?.geometry.dispose()
  material?.dispose()
  texture?.dispose()
  renderer?.dispose()
  renderer = undefined
  controls = undefined
  resident.length = 0
  queue.length = 0
}
function onContextLost(event: Event) {
  event.preventDefault()
  lostContext = true
  ready.value = false
  updateAnimation()
}
function onContextRestored() { lostContext = false; ready.value = true; updateAnimation() }

watch(() => props.artworks, items => { deck.update(items); refill() }, { immediate: true })
onMounted(() => {
  desktop = window.matchMedia('(min-width: 769px)')
  reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
  desktop.addEventListener('change', updateAnimation)
  reducedMotion.addEventListener('change', updateAnimation)
  document.addEventListener('visibilitychange', updateAnimation)
  observer = new IntersectionObserver(entries => { inView = entries.some(entry => entry.isIntersecting); updateAnimation() })
  if (host.value) observer.observe(host.value)
})
onActivated(() => { active = true; updateAnimation() })
onDeactivated(() => { active = false; universeOrigin.value = undefined; updateAnimation() })
onBeforeUnmount(() => {
  destroyed = true
  clearTimeout(refillTimer)
  lifetime.abort()
  loads.abort()
  observer?.disconnect()
  desktop?.removeEventListener('change', updateAnimation)
  reducedMotion?.removeEventListener('change', updateAnimation)
  document.removeEventListener('visibilitychange', updateAnimation)
  disposeScene()
})
</script>

<template>
  <div ref="host" class="artwork-planet-wrap" :class="{ 'is-exploring': universeOrigin }">
    <div class="artwork-planet" :class="{ 'is-ready': ready, 'is-dragging': dragging }" :style="{ backgroundImage: `url(${heroArt})` }">
      <canvas ref="canvas" class="artwork-planet-canvas" role="img" aria-label="作品星球，可拖动旋转" tabindex="0" title="拖动旋转，双击探索星图" @dblclick="explore" @keydown.enter.prevent="explore" @keydown="rotateByKey" @webglcontextlost="onContextLost" @webglcontextrestored="onContextRestored" />
    </div>
    <button v-if="ready" class="planet-rotation-toggle" type="button" :aria-label="paused ? '继续星球自转' : '暂停星球自转'" :title="paused ? '继续自转' : '暂停自转'" :aria-pressed="paused" @click="toggleRotation">
      <IconPlayArrow v-if="paused" /><IconPause v-else />
    </button>
  </div>
  <ArtworkUniverse v-if="universeOrigin" :artworks="artworks" :origin="universeOrigin" @closed="closeUniverse" @select="selectUniverse" />
</template>

<style scoped>
.artwork-planet-wrap { position: absolute; top: 31px; left: 81px; width: 180px; height: 180px; pointer-events: auto; }
.artwork-planet-wrap.is-exploring { visibility: hidden; }
.artwork-planet { width: 100%; height: 100%; border: 1px solid color-mix(in srgb, var(--creative-accent) 28%, var(--creative-line)); border-radius: 50%; background-position: center 22%; background-size: cover; overflow: hidden; box-shadow: 8px 12px 26px color-mix(in srgb, var(--creative-accent) 10%, transparent); }
.artwork-planet.is-ready { background-image: none !important; }
.artwork-planet-canvas { display: block; width: 100%; height: 100%; opacity: 0; cursor: grab; touch-action: none; }
.is-ready .artwork-planet-canvas { opacity: 1; }
.is-dragging .artwork-planet-canvas { cursor: grabbing; }
.planet-rotation-toggle { position: absolute; right: -9px; bottom: 4px; display: grid; place-items: center; width: 28px; height: 28px; border: 1px solid var(--creative-line); border-radius: 50%; color: var(--creative-accent); background: var(--creative-paper); cursor: pointer; }
.planet-rotation-toggle svg { width: 13px; height: 13px; }
.planet-rotation-toggle:hover { background: var(--creative-soft); }
.artwork-planet-wrap:has(:focus-visible) .artwork-planet { outline: 2px solid var(--creative-accent); outline-offset: 4px; }
</style>
