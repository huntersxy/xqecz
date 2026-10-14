<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount, useId } from 'vue'
import { IconDragDotVertical, IconClose } from '@arco-design/web-vue/es/icon'
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

// Pointer capture keeps mouse and touch on the same path. Cancel without changing
// the draft when capture is lost, the session changes, or submission locks it.
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
  // A transformed dragged tile can enlarge scrollWidth; clamp to the rail's
  // original extent so edge scrolling cannot chase its own moving preview.
  rail.value.scrollLeft = Math.max(0, Math.min(gesture.maxScroll, previous + direction * 8))
  locateTarget()
  // Only keep a frame loop while the rail can actually scroll at an edge.
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
      ?.querySelector<HTMLButtonElement>(`[data-reference-id="${id}"] .reference-sort-handle`)
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
      >拖动图片或排序手柄改变顺序；聚焦排序手柄后可用左右方向键、Home 和 End 移动。</span
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
        @click="preview(reference.asset_id)"
        @pointerdown="start($event, index)"
        @pointermove="move"
        @pointerup="finish"
        @pointercancel="resetGesture"
        @lostpointercapture="resetGesture"
        @dragstart.prevent
      >
        <CreativeImage :id="reference.asset_id" />
      </button>
      <button
        type="button"
        class="reference-label"
        :disabled="disabled"
        :aria-label="`插入 @图片${index + 1} 到提示词`"
        @click="emit('mention', index)"
      >
        @图片{{ index + 1 }}
      </button>
      <div class="reference-tools">
        <button
          type="button"
          class="reference-sort-handle"
          :disabled="disabled || references.length < 2"
          :aria-label="`拖动图片${index + 1}排序`"
          :aria-describedby="helpId"
          title="拖动排序，也可使用左右方向键"
          @pointerdown="start($event, index)"
          @pointermove="move"
          @pointerup="finish"
          @pointercancel="resetGesture"
          @lostpointercapture="resetGesture"
          @keydown="keyboard($event, index)"
        >
          <IconDragDotVertical />
        </button>
        <button
          type="button"
          :disabled="disabled"
          :aria-label="`移除图片${index + 1}`"
          @click="emit('remove', index)"
        >
          <IconClose />
        </button>
      </div>
    </div>
  </div>
  <span class="sr-only" role="status" aria-live="polite">{{ announcement }}</span>
</template>
