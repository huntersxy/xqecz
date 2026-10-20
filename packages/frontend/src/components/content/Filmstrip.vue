<script setup lang="ts">
import { ref } from 'vue'
import { getImageUrl } from '@/utils'
import { retryOnRemote } from '@/utils/contentMedia'
import CdIcon from './CdIcon.vue'

interface Item { id: number; title: string; thumb: string; isCurrent: boolean }

defineOptions({ name: 'CdFilmstrip' })
defineProps<{ items: Item[]; hasPrev: boolean; hasNext: boolean }>()
const emit = defineEmits<{ select: [id: number]; prev: []; next: []; hover: [id: number] }>()

const root = ref<HTMLElement | null>(null)
defineExpose({ root })

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
          <img :src="getImageUrl(item.thumb)" :alt="item.title" loading="lazy" decoding="async" draggable="false" @error="onThumbError" />
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
  padding: 11px clamp(14px, 2.5vw, 30px);
  background: color-mix(in srgb, var(--creative-paper) 70%, transparent);
  border-top: 1px solid var(--creative-line); scrollbar-width: none;
  /* 两侧渐隐：滚动到尽头时不像被硬切掉 */
  mask-image: linear-gradient(90deg, transparent 0, #000 26px, #000 calc(100% - 26px), transparent 100%);
}
.cd-film::-webkit-scrollbar { display: none; }
.cd-film-group { display: flex; align-items: center; gap: 10px; width: max-content; margin-inline: auto; }
.cd-film-track { display: flex; gap: 7px; }
/* 宽屏用舞台两侧的大翻页键，这里的前后键只在窄屏出现 */
.cd-film-nav { display: none; flex-shrink: 0; place-items: center; width: 30px; height: 30px; border: 1px solid var(--creative-line); border-radius: 8px; background: transparent; color: var(--creative-muted); cursor: pointer; transition: background .15s, color .15s, border-color .15s; }
.cd-film-nav:hover { color: var(--creative-accent); border-color: var(--creative-accent); background: var(--creative-soft); }
.cd-film-thumb { flex-shrink: 0; width: 60px; height: 46px; padding: 0; overflow: hidden; border: 1px solid transparent; border-radius: 8px; background: var(--creative-canvas); cursor: pointer; transition: border-color .18s, opacity .18s, transform .18s; }
.cd-film-thumb img { display: block; width: 100%; height: 100%; object-fit: cover; border-radius: 5px; }
.cd-film-thumb:hover { border-color: color-mix(in srgb, var(--creative-accent) 60%, transparent); opacity: .9; transform: translateY(-1px); }
.cd-thumb-current { border-color: var(--creative-accent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--creative-accent) 22%, transparent); }
@media (max-width: 768px) {
  .cd-film { padding: 9px 12px; }
  .cd-film-group { gap: 8px; }
  .cd-film-nav { display: grid; width: 26px; height: 26px; }
  .cd-film-thumb { width: 52px; height: 40px; }
}
</style>
