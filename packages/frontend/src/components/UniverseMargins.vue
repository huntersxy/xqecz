<template>
  <div class="universe-margins" aria-hidden="true">
    <div class="universe-margin universe-margin-left">
      <span class="margin-eyebrow">FIELD NOTES / 01</span>
      <svg viewBox="0 0 220 270" fill="none" focusable="false">
        <path class="margin-guide" d="M25 40h170M25 90h170M25 140h170M25 190h170M50 20v205M100 20v205M150 20v205M200 20v205" />
        <path class="margin-trail" d="m49 55 68 35 52-47-15 104-77 42-28-62 68-37 37 57" />
        <path class="margin-pencil" d="m49 48 2 5 6 2-6 2-2 6-2-6-6-2 6-2 2-5Zm120-13 2 6 6 2-6 2-2 6-2-6-6-2 6-2 2-6Zm-15 104 2 6 6 2-6 2-2 6-2-6-6-2 6-2 2-6Z" />
        <circle class="margin-dot" cx="117" cy="90" r="3" />
        <circle class="margin-dot" cx="77" cy="189" r="3" />
        <circle class="margin-dot" cx="49" cy="127" r="2" />
        <path class="margin-pencil" d="M69 225c23-8 54-6 76 1m-71 5c18-5 39-5 60-2" />
      </svg>
      <span class="margin-caption">把灵感连成星图</span>
      <span class="margin-note">somewhere, a new world begins.</span>
    </div>
    <div class="universe-margin universe-margin-right">
      <span class="margin-eyebrow">A WORK IN PROGRESS</span>
      <svg viewBox="0 0 220 245" fill="none" focusable="false">
        <path class="margin-guide" d="M23 115c-15-36 34-68 96-67 65 1 106 37 86 74-19 36-83 56-138 41" />
        <path class="margin-pencil" d="M151 39c5-8 12-10 15-7 4 3 2 10-4 18l-57 76-13-10 59-77Zm-59 77-7 16 9 7 11-13m-20 6c-17 0-19 13-31 15 15 8 36 12 40-8" />
        <path class="margin-pencil" d="M51 169c29-18 70-16 78 2 9 19-35 25-43 12-6-11 42-28 65-16 19 10 3 31-30 33" />
        <path class="margin-pencil" d="m44 61 2 7 7 2-7 2-2 7-2-7-7-2 7-2 2-7Zm139 82 2 6 6 2-6 2-2 6-2-6-6-2 6-2 2-6Z" />
        <circle class="margin-dot" cx="25" cy="115" r="3" />
      </svg>
      <span class="margin-caption">下一颗星球，正在落笔</span>
      <span class="margin-note">made of little ideas & a little courage.</span>
    </div>
  </div>
</template>

<style scoped>
/* 装饰只占据 1440px 内容区以外的页边，不参与瀑布流测量。 */
.universe-margins { display: none; position: absolute; inset: 0; z-index: -1; overflow: clip; pointer-events: none; user-select: none; }
.universe-margin { position: absolute; width: clamp(160px, calc((100vw - 1440px) / 2 - 72px), 230px); color: var(--creative-muted); text-align: center; opacity: .5; }
.universe-margin-left { left: calc((100% - 1440px) / 4); top: 210px; transform: translateX(-50%) rotate(-7deg); }
.universe-margin-right { right: calc((100% - 1440px) / 4); top: 535px; transform: translateX(50%) rotate(7deg); }
.universe-margin svg { display: block; width: 100%; stroke-linecap: round; stroke-linejoin: round; }
.margin-guide { stroke: var(--creative-line); stroke-width: 1; stroke-dasharray: 2 5; }
.margin-trail { stroke: var(--creative-muted); stroke-width: 1; opacity: .55; }
.margin-pencil { stroke: var(--creative-accent); stroke-width: 1.3; }
.margin-dot { fill: var(--creative-warm); stroke: var(--creative-accent); stroke-width: 1; }
.margin-eyebrow { font-size: 8px; letter-spacing: .2em; }
.margin-caption { display: block; margin-top: -12px; font: 17px/1.7 var(--creative-title-font); }
.margin-note { display: block; margin-top: 7px; font: italic 10px/1.6 Georgia, serif; }
@media (min-width: 2200px) { .universe-margins { display: block; } }
</style>
