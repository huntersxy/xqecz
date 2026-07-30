<script setup lang="ts">
import { getImageUrl } from '@/utils'
import SafeImage from '@/components/SafeImage.vue'
import type { Content } from '@/types'

interface Props {
  contents: unknown[]
  loading?: boolean
}

defineProps<Props>()
const emit = defineEmits<{ click: [item: Content] }>()
</script>

<template>
  <section class="wf-recommend">
    <div class="wf-recommend-head">
      <h2>✨ 精选推荐</h2>
    </div>
    <div class="wf-recommend-scroll">
      <div
        v-for="item in contents"
        :key="(item as Content).id"
        class="wf-recommend-card"
        @click="emit('click', item as Content)"
      >
        <SafeImage
          v-if="(item as Content).type !== 'text'"
          :src="getImageUrl((item as Content).thumb)"
          :alt="(item as Content).title"
          loading="lazy"
        />
        <div v-else class="wf-rec-text">{{ (item as Content).title }}</div>
        <div class="wf-rec-overlay">
          <span>{{ (item as Content).title }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.wf-recommend {
  max-width: 1600px; margin: 0 auto; padding: 1rem 1rem 0;
}
.wf-recommend-head {
  display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.625rem;
}
.wf-recommend-head h2 { font-size: 0.9375rem; font-weight: 700; margin: 0; color: var(--theme-text); }

.wf-recommend-scroll {
  display: flex; gap: 0.625rem; overflow-x: auto;
  padding-bottom: 0.75rem; scrollbar-width: none;
}
.wf-recommend-scroll::-webkit-scrollbar { display: none; }

.wf-recommend-card {
  flex-shrink: 0; width: 140px; height: 180px; border-radius: 0.75rem;
  overflow: hidden; position: relative; cursor: pointer;
  background: var(--theme-placeholder-bg);
}
.wf-recommend-card img { width: 100%; height: 100%; object-fit: cover; transition: transform 0.3s; }
.wf-recommend-card:hover img { transform: scale(1.08); }

.wf-rec-text {
  display: flex; align-items: center; justify-content: center;
  height: 100%; padding: 0.75rem; font-size: 0.75rem;
  color: var(--theme-text-secondary); text-align: center;
}

.wf-rec-overlay {
  position: absolute; bottom: 0; left: 0; right: 0;
  padding: 1.5rem 0.5rem 0.5rem;
  background: linear-gradient(transparent, rgba(0,0,0,0.7));
  color: #fff; font-size: 0.6875rem; font-weight: 500; line-height: 1.3;
}
.wf-rec-overlay span {
  display: -webkit-box; -webkit-line-clamp: 2;
  -webkit-box-orient: vertical; overflow: hidden;
}
</style>
