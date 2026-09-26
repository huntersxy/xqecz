<script setup lang="ts">
import { ref, watch } from 'vue'
import { IconImageClose } from '@arco-design/web-vue/es/icon'
import { getImageUrl, getRemoteFallbackUrl } from '@/utils'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{ src?: string; alt?: string; fallbackSrc?: string }>(),
  { src: '', alt: '', fallbackSrc: '' },
)

const emit = defineEmits<{ error: []; fallback: [] }>()

const currentSrc = ref(getImageUrl(props.src))
const finalFailed = ref(false)
/** 是否已试过调用方给的兜底地址（源站） */
let triedFallback = false
/** 是否已试过生产服务器同路径地址 */
let triedRemote = false

watch(
  () => props.src,
  (value) => {
    currentSrc.value = getImageUrl(value)
    triedFallback = false
    triedRemote = false
    finalFailed.value = false
  },
)

/**
 * 加载失败时逐级回退，前一级没地址或已经试过才试下一级：
 *   1. 调用方给的兜底地址（通常是源站）——镜像可达性判断再准也可能出错，这里才是真兜底；
 *   2. 生产服务器同路径地址；
 *   3. 都失败才对外 emit error 并展示坏图。
 *
 * 两级都试完仍失败后才 emit error；只要用上了第 1 级就会 emit fallback，
 * 供调用方据此判定「这一侧确实取不到」并修正来源决策。
 */
function onLoadError() {
  if (finalFailed.value) return
  if (!triedFallback) {
    const fb = props.fallbackSrc
    if (fb && fb !== currentSrc.value) {
      triedFallback = true
      currentSrc.value = fb
      emit('fallback')
      return
    }
  }
  if (!triedRemote) {
    const remote = getRemoteFallbackUrl(currentSrc.value)
    if (remote && remote !== currentSrc.value) {
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
    <template #error>
      <span :ref="(el) => el && onLoadError()" />
      <div v-if="finalFailed" class="arco-image-error">
        <div class="arco-image-error-icon">
          <IconImageClose />
        </div>
        <div v-if="alt" class="arco-image-error-alt">{{ alt }}</div>
      </div>
    </template>
  </a-image>
</template>
