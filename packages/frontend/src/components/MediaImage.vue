<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { IconImageClose } from '@arco-design/web-vue/es/icon'
import { getImageUrl, getRemoteFallbackUrl } from '@/utils'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{ src?: string; alt?: string; fallbackSrc?: string | string[] }>(),
  { src: '', alt: '', fallbackSrc: '' },
)

/**
 * `fallback` 会带上**失败的那个地址**，调用方据此判断是哪一级不可达
 *（OpenList 还是 R2），从而只修正那一级的结论。
 */
const emit = defineEmits<{ error: []; fallback: [failedUrl: string] }>()

/** 兜底链：调用方给的候选，按顺序尝试。单串写法等价于只有一个候选。 */
const fallbacks = computed(() => {
  const raw = props.fallbackSrc
  const list = Array.isArray(raw) ? raw : [raw]
  return list.filter((u) => typeof u === 'string' && u !== '')
})

const currentSrc = ref(getImageUrl(props.src))
const finalFailed = ref(false)
/** 已经试过并失败的地址，避免兜底链里出现环（A→B→A 无限重试）。 */
let attempted: string[] = []
/** 是否已试过生产服务器同路径地址 */
let triedRemote = false

watch(
  () => props.src,
  (value) => {
    currentSrc.value = getImageUrl(value)
    attempted = []
    triedRemote = false
    finalFailed.value = false
  },
)

/**
 * 加载失败时逐级回退，前一级没地址或已经试过才试下一级：
 *   1. 调用方给的兜底链（OpenList → R2 → 源站），逐项尝试；
 *   2. 生产服务器同路径地址；
 *   3. 都失败才对外 emit error 并展示坏图。
 *
 * 每次用上兜底链里的一项就 emit fallback 并带上失败地址，
 * 供调用方据此判定「这一级确实取不到」并修正来源决策。
 */
function onLoadError() {
  if (finalFailed.value) return
  const failed = currentSrc.value
  attempted.push(failed)

  for (const fb of fallbacks.value) {
    if (!attempted.includes(fb)) {
      currentSrc.value = fb
      emit('fallback', failed)
      return
    }
  }
  if (!triedRemote) {
    const remote = getRemoteFallbackUrl(failed)
    if (remote && !attempted.includes(remote)) {
      triedRemote = true
      currentSrc.value = remote
      return
    }
  }
  finalFailed.value = true
  emit('error')
}
</script>

<template>
  <a-image v-bind="$attrs" :src="currentSrc" :alt="alt">
    <template #extra>
      <slot name="extra" />
    </template>
    <template v-if="$slots.loader" #loader>
      <slot name="loader" />
    </template>
    <template #error>
      <span :ref="(el) => el && onLoadError()" />
      <slot v-if="finalFailed" name="error">
        <div class="arco-image-error">
          <div class="arco-image-error-icon">
            <IconImageClose />
          </div>
          <div v-if="alt" class="arco-image-error-alt">{{ alt }}</div>
        </div>
      </slot>
      <slot v-else name="loader" />
    </template>
  </a-image>
</template>
