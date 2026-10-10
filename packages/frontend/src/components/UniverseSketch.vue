<script setup lang="ts">
import heroArt from '@/assets/bg.webp'
</script>

<template>
  <div class="universe-sketch" aria-hidden="true">
    <span class="universe-halo"></span>
    <svg class="universe-orbits" viewBox="0 0 340 240" fill="none" focusable="false">
      <path class="universe-pencil" d="M66 156C20 133 65 67 151 48c88-20 161-4 161 35 0 38-78 88-164 99-49 7-82-2-88-20" />
      <path class="universe-guide" d="M98 44c-43 45-46 106-8 140 37 35 104 27 153-12 44-36 55-82 23-115" />
      <path class="universe-guide" d="m23 54 35-4m-15-13 2 30m249 116 26-4m-13-11 2 26" />
    </svg>
    <div class="universe-planet" :style="{ backgroundImage: 'url(' + heroArt + ')' }"></div>
    <svg class="universe-orbits universe-front" viewBox="0 0 340 240" fill="none" focusable="false">
      <path class="universe-pencil" d="M34 113c-3 30 68 42 147 18 72-21 122-51 122-77" />
      <path class="universe-pencil universe-second-stroke" d="M45 125c25 20 91 16 140 2 57-17 100-43 111-60" />
      <path class="universe-star" d="m279 34 3 10 10 3-10 3-3 11-3-11-10-3 10-3 3-10Z" />
      <path class="universe-tiny-star" d="m71 173 2 7 7 2-7 2-2 7-2-7-7-2 7-2 2-7Z" />
      <circle class="universe-satellite" cx="48" cy="107" r="5" />
    </svg>
    <span class="universe-caption">宇宙，从你的这一笔开始</span>
    <span class="universe-note">a little universe of our own</span>
  </div>
</template>

<style scoped>
.universe-sketch { position: relative; width: 340px; height: 250px; pointer-events: none; }
.universe-halo { position: absolute; inset: 5px 30px 25px; border-radius: 50%; background: radial-gradient(ellipse, color-mix(in srgb, var(--creative-warm) 38%, transparent), transparent 70%); }
.universe-orbits { position: absolute; inset: 0; width: 100%; height: 240px; stroke-linecap: round; stroke-linejoin: round; }
.universe-pencil { stroke: var(--creative-accent); stroke-width: 1.4; opacity: .6; }
.universe-guide { stroke: var(--creative-muted); stroke-width: 1; stroke-dasharray: 3 7; opacity: .3; }
.universe-planet { position: absolute; top: 39px; left: 89px; width: 165px; height: 165px; border: 1px solid color-mix(in srgb, var(--creative-accent) 28%, var(--creative-line)); border-radius: 48% 52% 51% 49% / 51% 46% 54% 49%; background-position: center 22%; background-size: cover; animation: universe-planet-light .8s ease-out both; box-shadow: inset 8px -8px 20px color-mix(in srgb, var(--creative-paper) 45%, transparent), 8px 12px 26px color-mix(in srgb, var(--creative-accent) 10%, transparent); }
.universe-planet::after { content: ''; position: absolute; inset: 0; border-radius: inherit; background: linear-gradient(135deg, color-mix(in srgb, var(--creative-paper) 25%, transparent), transparent 70%); }
.universe-second-stroke { opacity: .2; stroke-width: .8; }
.universe-star, .universe-tiny-star { stroke: var(--creative-accent); stroke-width: 1.4; fill: var(--creative-paper); }
.universe-star { transform-origin: 279px 47px; animation: universe-light-up 1s ease-out both; }
.universe-satellite { fill: var(--creative-warm); stroke: var(--creative-accent); stroke-width: 1.2; }
.universe-caption { position: absolute; bottom: 20px; left: 68px; color: var(--creative-ink); font: 17px/1.5 var(--creative-title-font); transform: rotate(-4deg); }
.universe-note { position: absolute; bottom: 0; left: 92px; color: var(--creative-muted); font: italic 10px/1.5 Georgia, serif; }
@keyframes universe-planet-light { from { opacity: .5; } to { opacity: 1; } }
@keyframes universe-light-up { from { opacity: .25; transform: scale(.65); } to { opacity: 1; transform: scale(1); } }
</style>
