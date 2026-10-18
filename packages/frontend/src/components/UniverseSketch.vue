<script setup lang="ts">
import { ref } from 'vue'
import ArtworkPlanet from '@/components/ArtworkPlanet.vue'
import type { UniverseArtwork } from '@/utils/artworkUniverse'

defineProps<{ artworks: readonly UniverseArtwork[] }>()
const emit = defineEmits<{ select: [id: number] }>()

// “双击画布”仍是星图入口，但手势不可发现：补一个看得见的按钮。
// 星图浮层归 ArtworkPlanet 持有，点按与双击走同一条 explore()。
const planet = ref<{ explore: () => void } | null>(null)
function openUniverse() {
  planet.value?.explore()
}
</script>

<template>
  <div class="universe-sketch">
    <div class="universe-stage">
      <svg class="universe-orbits" viewBox="0 0 240 170" fill="none" focusable="false" aria-hidden="true">
        <ellipse class="universe-orbit" cx="120" cy="85" rx="112" ry="64" transform="rotate(-12 120 85)" />
        <ellipse class="universe-orbit is-faint" cx="120" cy="85" rx="88" ry="49" transform="rotate(9 120 85)" stroke-dasharray="2 8" />
      </svg>
      <div class="universe-planet-box">
        <ArtworkPlanet ref="planet" :artworks="artworks" @select="emit('select', $event)" />
      </div>
    </div>
    <button class="universe-enter" type="button" @click="openUniverse">进入星图<span aria-hidden="true">↗</span></button>
  </div>
</template>

<style scoped>
.universe-sketch {
  --planet-size: clamp(108px, 9.8vw, 144px);
  display: grid;
  justify-items: center;
  gap: 1px;
  width: calc(var(--planet-size) * 1.58);
  flex: 0 0 auto;
}
.universe-stage {
  position: relative;
  display: grid;
  place-items: center;
  width: 100%;
  height: calc(var(--planet-size) * 1.34);
}
.universe-stage::before {
  content: '';
  position: absolute;
  width: calc(var(--planet-size) * 1.28);
  aspect-ratio: 1;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--creative-warm) 24%, transparent), transparent 68%);
  pointer-events: none;
}
.universe-orbits {
  position: absolute;
  top: 50%;
  left: 50%;
  width: calc(var(--planet-size) * 1.52);
  height: calc(var(--planet-size) * 1.08);
  transform: translate(-50%, -50%);
  overflow: visible;
  pointer-events: none;
}
.universe-orbit {
  stroke: color-mix(in srgb, var(--home-accent) 28%, transparent);
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}
.universe-orbit.is-faint {
  stroke: color-mix(in srgb, var(--home-accent) 16%, transparent);
}
.universe-planet-box {
  position: relative;
  z-index: 2;
  width: var(--planet-size);
  height: var(--planet-size);
}
.universe-enter {
  display: inline-flex;
  align-items: baseline;
  gap: 3px;
  padding: 2px 0;
  border: 0;
  background: none;
  color: color-mix(in srgb, var(--home-ink) 70%, var(--home-muted));
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: color .18s;
}
.universe-enter:hover { color: var(--home-accent); }
.universe-enter:focus-visible { outline: 2px solid var(--home-accent); outline-offset: 3px; border-radius: 2px; }
@media (max-width: 900px) {
  .universe-sketch { --planet-size: 108px; }
}
@media (max-width: 768px) {
  /* 手机端不初始化星球渲染，静态底图会读成一张头像；星图入口也仅桌面可用，整块收起。 */
  .universe-sketch { display: none; }
}
@media (prefers-reduced-motion: reduce) {
  .universe-enter { transition: none; }
}
</style>
