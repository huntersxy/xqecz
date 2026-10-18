<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const canvasRef = ref<HTMLCanvasElement | null>(null)

let animationFrame = 0
let resizeObserver: ResizeObserver | undefined
let context: CanvasRenderingContext2D | null = null
let width = 0
let height = 0
let reducedMotion = false
let colors = { accent: '#e6c48d', warm: '#efa3b2', paper: '#fffdfd' }

const PARTICLE_COUNT = 30
const LOOP_MS = 14000

function readColors() {
  const styles = getComputedStyle(canvasRef.value as HTMLCanvasElement)
  return {
    accent: styles.getPropertyValue('--orbit-accent').trim() || '#e6c48d',
    warm: styles.getPropertyValue('--orbit-warm').trim() || '#efa3b2',
    paper: styles.getPropertyValue('--orbit-paper').trim() || '#fffdfd',
  }
}

function resizeCanvas() {
  const canvas = canvasRef.value
  if (!canvas) return
  const rect = canvas.getBoundingClientRect()
  const dpr = Math.min(window.devicePixelRatio || 1, 2)
  width = Math.max(1, rect.width)
  height = Math.max(1, rect.height)
  canvas.width = Math.round(width * dpr)
  canvas.height = Math.round(height * dpr)
  context = canvas.getContext('2d')
  context?.setTransform(dpr, 0, 0, dpr, 0, 0)
  colors = readColors()
  draw(0)
}

function draw(time: number) {
  const ctx = context
  if (!ctx || width <= 0 || height <= 0) return

  const centerX = width * 0.52
  const centerY = height * 0.5
  const phase = ((time % LOOP_MS) / LOOP_MS) * Math.PI * 2
  ctx.clearRect(0, 0, width, height)

  const glow = ctx.createRadialGradient(centerX, centerY, 0, centerX, centerY, Math.min(width, height) * 0.44)
  glow.addColorStop(0, `${colors.warm}22`)
  glow.addColorStop(0.52, `${colors.accent}0b`)
  glow.addColorStop(1, 'transparent')
  ctx.fillStyle = glow
  ctx.fillRect(0, 0, width, height)

  ctx.save()
  ctx.translate(centerX, centerY)
  ctx.rotate(-0.18)
  for (let index = 0; index < 3; index += 1) {
    ctx.beginPath()
    ctx.ellipse(0, 0, width * (0.3 + index * 0.085), height * (0.14 + index * 0.04), 0, 0, Math.PI * 2)
    ctx.strokeStyle = index === 1 ? `${colors.warm}55` : `${colors.paper}2c`
    ctx.lineWidth = index === 1 ? 1.15 : 0.7
    ctx.setLineDash(index === 1 ? [] : [2, 8])
    ctx.stroke()
  }
  ctx.restore()

  for (let index = 0; index < PARTICLE_COUNT; index += 1) {
    const seed = index * 1.61803398875
    const orbit = index % 3
    const radiusX = width * (0.24 + orbit * 0.08)
    const radiusY = height * (0.12 + orbit * 0.035)
    const angle = phase * (index % 2 === 0 ? 1 : -0.74) + seed
    const x = centerX + Math.cos(angle) * radiusX
    const y = centerY + Math.sin(angle) * radiusY
    const size = index % 7 === 0 ? 2.4 : 1.1 + (index % 3) * 0.35
    ctx.beginPath()
    ctx.arc(x, y, size, 0, Math.PI * 2)
    ctx.fillStyle = index % 5 === 0 ? `${colors.warm}d9` : `${colors.paper}9c`
    ctx.fill()
  }

  const starX = centerX + Math.cos(phase * 0.8) * width * 0.31
  const starY = centerY + Math.sin(phase * 0.8) * height * 0.15
  ctx.save()
  ctx.translate(starX, starY)
  ctx.rotate(phase * 0.18)
  ctx.beginPath()
  ctx.moveTo(0, -7)
  ctx.lineTo(2, -2)
  ctx.lineTo(7, 0)
  ctx.lineTo(2, 2)
  ctx.lineTo(0, 7)
  ctx.lineTo(-2, 2)
  ctx.lineTo(-7, 0)
  ctx.lineTo(-2, -2)
  ctx.closePath()
  ctx.strokeStyle = `${colors.warm}e6`
  ctx.lineWidth = 1
  ctx.stroke()
  ctx.restore()
}

function onVisibilityChange() {
  if (document.visibilityState === 'hidden') {
    cancelAnimationFrame(animationFrame)
    animationFrame = 0
    return
  }
  if (!reducedMotion && !animationFrame) animationFrame = requestAnimationFrame(tick)
}

function tick(time: number) {
  animationFrame = 0
  draw(time)
  if (!reducedMotion && document.visibilityState !== 'hidden') {
    animationFrame = requestAnimationFrame(tick)
  }
}

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  context = canvas.getContext('2d')
  reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  resizeObserver = new ResizeObserver(resizeCanvas)
  resizeObserver.observe(canvas)
  document.addEventListener('visibilitychange', onVisibilityChange)
  resizeCanvas()
  if (!reducedMotion) animationFrame = requestAnimationFrame(tick)
})

onBeforeUnmount(() => {
  cancelAnimationFrame(animationFrame)
  resizeObserver?.disconnect()
  document.removeEventListener('visibilitychange', onVisibilityChange)
  context = null
})
</script>

<template>
  <canvas ref="canvasRef" class="hero-orbit-canvas" aria-hidden="true" />
</template>

<style scoped>
.hero-orbit-canvas {
  --orbit-accent: #e6c48d;
  --orbit-warm: #efa3b2;
  --orbit-paper: #fffdfd;
  position: absolute;
  inset: 0;
  z-index: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}
</style>
