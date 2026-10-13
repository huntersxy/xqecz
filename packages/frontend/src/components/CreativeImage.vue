<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { aiApi } from '@/api/creative'
const props = defineProps<{ id: number; ideaId?: number; publicAsset?: boolean; full?: boolean }>()
const src = ref('')
const failed = ref(false)
let controller: AbortController | undefined
let generation = 0
watch(
  () => [props.id, props.ideaId, props.publicAsset, props.full],
  async () => {
    controller?.abort()
    const epoch = ++generation
    if (src.value) URL.revokeObjectURL(src.value)
    src.value = ''
    failed.value = false
    if (!props.id) return
    controller = new AbortController()
    try {
      const blob = props.ideaId
        ? await aiApi.ideaBlob(props.ideaId, props.id, controller.signal, !props.full)
        : props.publicAsset
          ? await aiApi.publicBlob(props.id, controller.signal, !props.full)
          : await aiApi.blob(props.id, controller.signal, !props.full)
      if (epoch !== generation) return
      src.value = URL.createObjectURL(blob)
    } catch {
      if (epoch === generation && !controller.signal.aborted) failed.value = true
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => {
  ++generation
  controller?.abort()
  if (src.value) URL.revokeObjectURL(src.value)
})
</script>
<template>
  <span class="creative-image"
    ><img v-if="src" :src="src" alt="创作图片" loading="lazy" /><span v-else role="status">{{
      failed ? '图片暂不可用' : '读取图片…'
    }}</span></span
  >
</template>
<style scoped>
.creative-image {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
  place-items: center;
  min-width: 0;
  min-height: 80px;
  width: 100%;
  height: 100%;
  color: var(--creative-muted);
  font-size: 12px;
}
.creative-image img {
  display: block;
  min-width: 0;
  min-height: 0;
  max-width: 100%;
  max-height: 100%;
  width: 100%;
  height: 100%;
  object-fit: contain;
}
</style>
