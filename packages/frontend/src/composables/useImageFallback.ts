import { computed, ref, watch, type Ref } from 'vue'
import { getImageUrl, getRemoteFallbackUrl } from '@/utils'

export interface ImageFallbackOptions {
  src: Ref<string | undefined>
  /** 调用方给的兜底候选，按顺序尝试；单串写法等价于只有一个候选。 */
  fallbackSrc: Ref<string | string[] | undefined>
  /** 每用上兜底链里的一项就回调，带上失败的地址，供调用方修正来源判断。 */
  onFallback?: (failedUrl: string) => void
  /** 所有兜底都失败后调用一次。 */
  onFinalError?: () => void
}

/**
 * 图片加载失败时的逐级回退，MediaImage 与 SiteImage 共用同一套规则：
 *   1. 调用方给的兜底链（OpenList → R2 → 源站），逐项尝试；
 *   2. 生产服务器同路径地址；
 *   3. 都失败才标记最终失败。
 * 已经试过的地址会被记录，兜底链里出现环（A→B→A）也不会无限重试。
 */
export function useImageFallback(options: ImageFallbackOptions) {
  const fallbacks = computed(() => {
    const raw = options.fallbackSrc.value
    const list = Array.isArray(raw) ? raw : [raw]
    return list.filter((u): u is string => typeof u === 'string' && u !== '')
  })

  const currentSrc = ref(getImageUrl(options.src.value ?? ''))
  const finalFailed = ref(false)
  let attempted: string[] = []
  let triedRemote = false

  watch(options.src, (value) => {
    currentSrc.value = getImageUrl(value ?? '')
    attempted = []
    triedRemote = false
    finalFailed.value = false
  })

  function onLoadError() {
    if (finalFailed.value) return
    const failed = currentSrc.value
    attempted.push(failed)

    for (const fb of fallbacks.value) {
      if (!attempted.includes(fb)) {
        currentSrc.value = fb
        options.onFallback?.(failed)
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
    options.onFinalError?.()
  }

  return { currentSrc, finalFailed, onLoadError }
}
