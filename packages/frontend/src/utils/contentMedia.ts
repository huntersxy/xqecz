import { getImageUrl, getRemoteFallbackUrl } from '@/utils'
import { fallbackChain, pickImageUrl, type MediaSource } from '@/utils/imageSource'
import type { Content } from '@/types'

export type MediaKind = 'image' | 'video' | 'text'

type MediaFields = Pick<
  Content,
  'img' | 'video' | 'mirror_img' | 'mirror_video' | 'mirror2_img' | 'mirror2_video'
>

export function mediaKindOf(c: Pick<Content, 'img' | 'video'>): MediaKind {
  if (c.img) return 'image'
  if (c.video) return 'video'
  return 'text'
}

export function mediaTiers(c: MediaFields) {
  const isImage = !!c.img
  return {
    local: getImageUrl(c.img || c.video),
    mirror: (isImage ? c.mirror_img : c.mirror_video) || '',
    mirror2: (isImage ? c.mirror2_img : c.mirror2_video) || '',
  }
}

export function resolveMedia(c: MediaFields, source: MediaSource | null) {
  const { local, mirror, mirror2 } = mediaTiers(c)
  return {
    url: source ? pickImageUrl(local, mirror, mirror2, source) : '',
    fallbacks: fallbackChain(local, mirror, mirror2, source),
  }
}

export function rewriteMediaInHtml(html: string): string {
  if (!html || !html.includes('<img')) return html
  const tpl = document.createElement('template')
  tpl.innerHTML = html
  tpl.content.querySelectorAll('img').forEach((img) => {
    const src = img.getAttribute('src')
    if (src) img.setAttribute('src', getImageUrl(src))
    img.setAttribute('loading', 'lazy')
    img.setAttribute('decoding', 'async')
  })
  return tpl.innerHTML
}

export function retryOnRemote(img: HTMLImageElement): boolean {
  if (img.dataset.retried) return false
  img.dataset.retried = '1'
  const remote = getRemoteFallbackUrl(img.currentSrc || img.src)
  if (!remote) return false
  img.src = remote
  return true
}
