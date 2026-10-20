import { getImageUrl, getRemoteFallbackUrl } from '@/utils'
import { fallbackChain, pickImageUrl, type MediaSource } from '@/utils/imageSource'
import { forgetLoaded, recallLoaded, rememberLoaded, sameMedia } from '@/utils/mediaMemory'
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

export function mediaCandidates(c: MediaFields, source: MediaSource | null): string[] {
  if (!source) return []
  const { local, mirror, mirror2 } = mediaTiers(c)
  const { url, fallbacks } = resolveMedia(c, source)
  const list = [recallLoaded(local), url, ...fallbacks, getRemoteFallbackUrl(local)]
  return list.filter((u, i): u is string => !!u && list.indexOf(u) === i && (!mirror && !mirror2 || sameMedia(local, u)))
}

export function noteLoaded(c: MediaFields, loadedUrl: string): void {
  rememberLoaded(mediaTiers(c).local, loadedUrl)
}

export function noteFailed(c: MediaFields, failedUrl: string): void {
  const local = mediaTiers(c).local
  if (recallLoaded(local) === failedUrl) forgetLoaded(local)
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
