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

/** 三级地址：源站、R2、OpenList。镜像未启用时对应级为空串。 */
export function mediaTiers(c: MediaFields) {
  const isImage = !!c.img
  return {
    local: getImageUrl(c.img || c.video),
    mirror: (isImage ? c.mirror_img : c.mirror_video) || '',
    mirror2: (isImage ? c.mirror2_img : c.mirror2_video) || '',
  }
}

/**
 * 按已确定的来源解出要加载的地址与兜底链。
 * 详情页大图和邻近作品的预取必须走这同一个函数，否则预取的是别的地址，
 * 既白占带宽，又不会命中浏览器缓存。
 */
export function resolveMedia(c: MediaFields, source: MediaSource | null) {
  const { local, mirror, mirror2 } = mediaTiers(c)
  return {
    url: source ? pickImageUrl(local, mirror, mirror2, source) : '',
    fallbacks: fallbackChain(local, mirror, mirror2, source),
  }
}

/**
 * 正文是 Markdown 渲染出的 HTML，里面的 `/uploads/...` 相对路径会按页面域名解析，
 * 而本站媒体并不在页面域名下。这里把站内媒体路径补上媒体域名；外链保持原样。
 * 用 <template> 解析，不会触发图片请求。
 */
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

/**
 * <img> 加载失败时的一次性退路：站内媒体换到生产服务器同路径地址重试。
 * 外部图床、前端素材没有退路（getRemoteFallbackUrl 返回空串），保持失败。
 * 返回是否已经换了地址。
 */
export function retryOnRemote(img: HTMLImageElement): boolean {
  if (img.dataset.retried) return false
  img.dataset.retried = '1'
  const remote = getRemoteFallbackUrl(img.currentSrc || img.src)
  if (!remote) return false
  img.src = remote
  return true
}
