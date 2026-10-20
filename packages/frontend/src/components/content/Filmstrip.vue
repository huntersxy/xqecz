<script setup lang="ts">
import { ref } from 'vue'
import { getImageUrl } from '@/utils'
import { recallLoaded, rememberLoaded } from '@/utils/mediaMemory'
import { retryOnRemote } from '@/utils/contentMedia'
import CdIcon from './CdIcon.vue'

interface Item { id: number; title: string; thumb: string; isCurrent: boolean }

defineOptions({ name: 'CdFilmstrip' })
defineProps<{ items: Item[]; hasPrev: boolean; hasNext: boolean }>()
const emit = defineEmits<{ select: [id: number]; prev: []; next: []; hover: [id: number] }>()

const root = ref<HTMLElement | null>(null)
defineExpose({ root })

function thumbUrl(thumb: string) {
  const local = getImageUrl(thumb)
  return recallLoaded(local) || local
}

function onThumbLoad(e: Event) {
  const img = e.currentTarget as HTMLImageElement
  rememberLoaded(getImageUrl(img.dataset.local || ''), img.currentSrc || img.src)
}

function onThumbError(e: Event) {
  const img = e.currentTarget as HTMLImageElement
  if (!retryOnRemote(img)) img.style.visibility = 'hidden'
}
</script>

<template>
  <div ref="root" class="cd-film">
    <div class="cd-film-group">
      <button v-if="hasPrev" class="cd-film-nav" type="button" aria-label="上一个" @click="emit('prev')">
        <CdIcon name="chevron-left" />
      </button>
      <div class="cd-film-track">
        <button
          v-for="item in items"
          :key="item.id"
          type="button"
          class="cd-film-thumb"
          :class="{ 'cd-thumb-current': item.isCurrent }"
          :aria-label="item.title || '查看作品'"
          :aria-current="item.isCurrent ? 'true' : undefined"
          @pointerenter="emit('hover', item.id)"
          @focus="emit('hover', item.id)"
          @click="emit('select', item.id)"
        >
          <img :src="thumbUrl(item.thumb)" :data-local="getImageUrl(item.thumb)" :alt="item.title" loading="lazy" decoding="async" draggable="false" @load="onThumbLoad" @error="onThumbError" />
        </button>
      </div>
      <button v-if="hasNext" class="cd-film-nav" type="button" aria-label="下一个" @click="emit('next')">
        <CdIcon name="chevron-right" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.cd-film {
  position: relative; z-index: 2; flex-shrink: 0; display: flex; overflow-x: auto;
  padding: 14px clamp(18px, 3vw, 36px) 16px;
  background: var(--creative-paper);
  border-top: 1px solid var(--creative-line); scrollbar-width: none;
  mask-image: linear-gradient(90deg, transparent 0, #000 32px, #000 calc(100% - 32px), transparent 100%);
}
.cd-film::-webkit-scrollbar { display: none; }
.cd-film-group { display: flex; align-items: center; gap: 12px; width: max-content; margin-inline: auto; }
.cd-film-track { display: flex; gap: 10px; }
.cd-film-nav { display: none; flex-shrink: 0; place-items: center; width: 30px; height: 30px; border: 1px solid var(--creative-line); border-radius: 50%; background: var(--creative-canvas); color: var(--creative-muted); cursor: pointer; transition: background .15s, color .15s, border-color .15s; }
.cd-film-nav:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-soft); }
.cd-film-thumb {
  flex-shrink: 0; width: 76px; height: 56px; padding: 0; overflow: hidden; border: 0; border-radius: 10px;
  background: var(--creative-canvas); opacity: .55; cursor: pointer;
  transition: opacity .2s, transform .2s, box-shadow .2s;
}
.cd-film-thumb img { display: block; width: 100%; height: 100%; object-fit: cover; }
.cd-film-thumb:hover { opacity: .95; transform: translateY(-2px); }
.cd-film-thumb:focus-visible { opacity: 1; outline: 2px solid var(--creative-accent); outline-offset: 2px; }
.cd-thumb-current,
.cd-thumb-current:hover {
  opacity: 1; transform: translateY(-2px);
  box-shadow: 0 0 0 2px var(--creative-paper), 0 0 0 4px var(--creative-accent), 0 8px 18px color-mix(in srgb, var(--creative-accent) 24%, transparent);
}
@media (max-width: 768px) {
  .cd-film { padding: 12px 14px 14px; }
  .cd-film-group { gap: 8px; }
  .cd-film-track { gap: 8px; }
  .cd-film-nav { display: grid; width: 28px; height: 28px; }
  .cd-film-thumb { width: 60px; height: 46px; border-radius: 8px; }
}
</style>
