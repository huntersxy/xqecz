<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount, useId } from 'vue'
import { IconClose } from '@arco-design/web-vue/es/icon'
import CreativeImage from './CreativeImage.vue'
import type { GenerationInput } from '@/api/creative'

const props = defineProps<{ references: GenerationInput['references']; disabled: boolean }>()
const emit = defineEmits<{
  preview: [id: number]
  mention: [index: number]
  remove: [index: number]
  reorder: [from: number, to: number]
}>()
const rail = ref<HTMLElement>()
const helpId = useId()
const draggedId = ref(0),
  targetIndex = ref(-1),
  offset = ref(0),
  announcement = ref('')
let gesture:
  | {
      pointerId: number
      index: number
      id: number
      startX: number
      startY: number
      centerX: number
      scrollLeft: number
      maxScroll: number
      x: number
      y: number
      owner: HTMLElement
    }
  | undefined
let frame = 0,
  suppressPreview = 0

function resetGesture() {
  if (frame) cancelAnimationFrame(frame)
  frame = 0
  const previous = gesture
  gesture = undefined
  draggedId.value = 0
  targetIndex.value = -1
  offset.value = 0
  if (previous?.owner.hasPointerCapture?.(previous.pointerId))
    previous.owner.releasePointerCapture(previous.pointerId)
}
function start(event: PointerEvent, index: number) {
  if (props.disabled || props.references.length < 2 || event.button !== 0 || !event.isPrimary)
    return
  resetGesture()
  suppressPreview = 0
  const owner = event.currentTarget as HTMLElement
  const tile = owner.closest<HTMLElement>('.composer-reference')!
  const rect = tile.getBoundingClientRect()
  gesture = {
    pointerId: event.pointerId,
    index,
    id: props.references[index]!.asset_id,
    startX: event.clientX,
    startY: event.clientY,
    centerX: rect.left + rect.width / 2,
    scrollLeft: rail.value?.scrollLeft ?? 0,
    maxScroll: Math.max(0, (rail.value?.scrollWidth ?? 0) - (rail.value?.clientWidth ?? 0)),
    x: event.clientX,
    y: event.clientY,
    owner,
  }
  owner.setPointerCapture?.(event.pointerId)
}
function locateTarget() {
  if (!gesture || !rail.value) return
  const bounds = rail.value.getBoundingClientRect()
  offset.value = gesture.x - gesture.startX + rail.value.scrollLeft - gesture.scrollLeft
  if (
    gesture.y < bounds.top - 24 ||
    gesture.y > bounds.bottom + 24 ||
    gesture.x < bounds.left - 24 ||
    gesture.x > bounds.right + 24
  ) {
    targetIndex.value = -1
    return
  }
  let nearest = gesture.index,
    distance = Infinity
  Array.from(rail.value.querySelectorAll<HTMLElement>('.composer-reference')).forEach(
    (tile, index) => {
      const rect = tile.getBoundingClientRect()
      const center =
        index === gesture!.index
          ? gesture!.centerX - (rail.value!.scrollLeft - gesture!.scrollLeft)
          : rect.left + rect.width / 2
      const next = Math.abs(gesture!.x - center)
      if (next < distance) {
        distance = next
        nearest = index
      }
    },
  )
  targetIndex.value = nearest
}
function scrollAtEdge() {
  frame = 0
  if (!gesture || !draggedId.value || !rail.value) return
  const bounds = rail.value.getBoundingClientRect()
  const direction = gesture.x < bounds.left + 32 ? -1 : gesture.x > bounds.right - 32 ? 1 : 0
  const previous = rail.value.scrollLeft
  rail.value.scrollLeft = Math.max(0, Math.min(gesture.maxScroll, previous + direction * 8))
  locateTarget()
  if (direction && rail.value.scrollLeft !== previous) frame = requestAnimationFrame(scrollAtEdge)
}
function move(event: PointerEvent) {
  if (!gesture || event.pointerId !== gesture.pointerId || props.disabled) return
  gesture.x = event.clientX
  gesture.y = event.clientY
  if (
    !draggedId.value &&
    Math.hypot(event.clientX - gesture.startX, event.clientY - gesture.startY) < 6
  )
    return
  draggedId.value = gesture.id
  event.preventDefault()
  locateTarget()
  if (!frame) frame = requestAnimationFrame(scrollAtEdge)
}
function reorder(from: number, to: number) {
  if (props.disabled || from === to || to < 0 || to >= props.references.length) return
  const id = props.references[from]!.asset_id
  emit('reorder', from, to)
  announcement.value = `图片${from + 1}已移至第${to + 1}位，提示词引用已同步更新。`
  void nextTick(() =>
    rail.value
      ?.querySelector<HTMLButtonElement>(`[data-reference-id="${id}"] .reference-preview`)
      ?.focus({ preventScroll: true }),
  )
}
function finish(event: PointerEvent) {
  if (!gesture || event.pointerId !== gesture.pointerId) return
  const from = gesture.index,
    to = targetIndex.value,
    id = draggedId.value
  resetGesture()
  if (id) {
    suppressPreview = id
    reorder(from, to)
  }
}
function preview(id: number) {
  if (suppressPreview === id) {
    suppressPreview = 0
    return
  }
  if (!props.disabled) emit('preview', id)
}
function keyboard(event: KeyboardEvent, index: number) {
  if (props.disabled) return
  if (event.key === 'Escape') {
    event.preventDefault()
    resetGesture()
    return
  }
  const to =
    event.key === 'ArrowLeft'
      ? index - 1
      : event.key === 'ArrowRight'
        ? index + 1
        : event.key === 'Home'
          ? 0
          : event.key === 'End'
            ? props.references.length - 1
            : -1
  if (to < 0) return
  event.preventDefault()
  reorder(index, to)
}
watch(() => [props.disabled, props.references.map((item) => item.asset_id).join(',')], resetGesture)
onBeforeUnmount(resetGesture)
</script>

<template>
  <div
    v-if="references.length"
    ref="rail"
    class="composer-references"
    role="list"
    aria-label="已选参考图"
  >
    <span :id="helpId" class="sr-only"
      >拖动图片改变顺序；聚焦图片后可用左右方向键、Home 和 End 移动。点击角标可插入提示词引用。</span
    >
    <div
      v-for="(reference, index) in references"
      :key="reference.asset_id"
      class="composer-reference"
      role="listitem"
      :data-reference-id="reference.asset_id"
      :class="{
        'is-sorting': draggedId === reference.asset_id,
        'is-sort-target': draggedId && targetIndex === index && draggedId !== reference.asset_id,
      }"
      :style="
        draggedId === reference.asset_id ? { transform: `translateX(${offset}px)` } : undefined
      "
    >
      <button
        class="reference-preview"
        type="button"
        :disabled="disabled"
        :aria-label="`预览参考图 ${index + 1}`"
        :aria-describedby="helpId"
        :title="`图片 ${index + 1}：点击预览，拖动排序`"
        @click="preview(reference.asset_id)"
        @pointerdown="start($event, index)"
        @pointermove="move"
        @pointerup="finish"
        @pointercancel="resetGesture"
        @lostpointercapture="resetGesture"
        @keydown="keyboard($event, index)"
        @dragstart.prevent
      >
        <CreativeImage :id="reference.asset_id" />
      </button>
      <button
        type="button"
        class="reference-label reference-index"
        :disabled="disabled"
        :aria-label="`插入 @图片${index + 1} 到提示词`"
        title="插入提示词引用"
        @click="emit('mention', index)"
      >
        @{{ index + 1 }}
      </button>
      <button
        type="button"
        class="reference-remove"
        :disabled="disabled"
        :aria-label="`移除图片${index + 1}`"
        title="移除参考图"
        @click="emit('remove', index)"
      >
        <IconClose />
      </button>
    </div>
  </div>
  <span class="sr-only" role="status" aria-live="polite">{{ announcement }}</span>
</template>

<style scoped>
.composer-references {
  gap: 8px;
  padding-bottom: 4px;
}
.composer-reference {
  display: block;
  width: 56px;
  height: 56px;
}
.composer-reference .reference-preview {
  position: relative;
  display: block;
  width: 56px;
  height: 56px;
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--creative-line);
  border-radius: 10px;
  background: var(--creative-canvas);
  cursor: grab;
  touch-action: pan-y;
  user-select: none;
}
.reference-preview:active {
  cursor: grabbing;
}
.reference-preview .creative-image {
  height: 100%;
  min-height: 0;
}
.reference-preview :deep(.creative-image img) {
  object-fit: cover;
}
.composer-reference .reference-index {
  position: absolute;
  z-index: 2;
  bottom: 3px;
  left: 3px;
  box-sizing: border-box;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border: 0;
  border-radius: 999px;
  background: color-mix(in srgb, var(--creative-paper) 88%, transparent);
  color: var(--creative-accent);
  font-size: 9px;
  font-weight: 650;
  line-height: 18px;
  text-align: center;
  cursor: pointer;
}
.composer-reference .reference-remove {
  position: absolute;
  z-index: 2;
  top: 2px;
  right: 2px;
  box-sizing: border-box;
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: color-mix(in srgb, var(--creative-paper) 88%, transparent);
  color: var(--creative-muted);
  opacity: 0;
  cursor: pointer;
  transition: opacity var(--transition-fast), color var(--transition-fast);
}
.composer-reference .reference-remove svg {
  width: 10px;
  height: 10px;
}
.composer-reference:hover .reference-remove,
.reference-remove:focus-visible {
  opacity: 1;
}
.reference-remove:hover {
  color: var(--creative-accent);
}
.reference-index:disabled,
.reference-remove:disabled {
  opacity: 0;
  cursor: not-allowed;
}
@media (hover: none) {
  .reference-remove {
    opacity: 1;
  }
}
@media (max-width: 820px) {
  .composer-reference .reference-index,
  .composer-reference .reference-remove {
    width: 20px;
    height: 20px;
    line-height: 20px;
  }
  .composer-reference .reference-index {
    bottom: 2px;
    left: 2px;
    font-size: 9px;
  }
  .composer-reference .reference-remove {
    top: 2px;
    right: 2px;
    opacity: 1;
  }
  .composer-reference .reference-remove svg {
    width: 11px;
    height: 11px;
  }
}
</style>
