import { marked } from 'marked'
import DOMPurify from 'dompurify'
import dayjs from 'dayjs'
import removeMarkdown from 'remove-markdown'
import SparkMD5 from 'spark-md5'

const MEDIA_BASE = import.meta.env.VITE_MEDIA_BASE_URL || ''

export function isContentMediaPath(path: string): boolean {
  return /^\/(thumbs|uploads|images)\//.test(path)
}

export function getImageUrl(image?: string): string {
  if (!image) return ''
  if (image.startsWith('http://') || image.startsWith('https://')) return image
  if (isContentMediaPath(image)) {
    return `${MEDIA_BASE}${image}`
  }
  return image
}

export const REMOTE_MEDIA_BASE = 'https://api39.xiey.work'

export function getRemoteFallbackUrl(url: string, origin?: string): string {
  if (!url) return ''
  const baseOrigin =
    origin ?? (typeof window !== 'undefined' ? window.location.origin : '')
  if (!baseOrigin) return ''
  let u: URL
  try {
    u = new URL(url, baseOrigin)
  } catch {
    return ''
  }
  if (!/^https?:$/.test(u.protocol) || !isContentMediaPath(u.pathname)) return ''
  const remoteHost = new URL(REMOTE_MEDIA_BASE).hostname
  let mediaHost = ''
  if (MEDIA_BASE) {
    try {
      mediaHost = new URL(MEDIA_BASE).hostname
    } catch {
      mediaHost = ''
    }
  }
  const isOwnHost =
    u.hostname === 'localhost' ||
    u.hostname === '127.0.0.1' ||
    u.hostname === remoteHost ||
    u.hostname === 'xq.xiey.work' ||
    (mediaHost !== '' && u.hostname === mediaHost) ||
    u.origin === baseOrigin
  if (!isOwnHost) return ''
  const target = `${REMOTE_MEDIA_BASE}${u.pathname}${u.search}`
  return target === url ? '' : target
}

export function getAvatarUrl(email: string, size: number = 80): string {
  if (!email.trim()) return ''
  const qqMatch = /^(\d{5,11})@qq\.com$/i.exec(email.trim())
  if (qqMatch) {
    return `https://q.qlogo.cn/headimg_dl?dst_uin=${qqMatch[1]}&spec=100`
  }
  const hash = md5Hex(email.trim().toLowerCase())
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=${size}`
}

function md5Hex(input: string): string {
  return SparkMD5.hash(input)
}

export function formatTime(ts: number | string, useLocaleDate: boolean = false): string {
  if (!ts) return ''
  const d = typeof ts === 'string' && /^\d{4}-/.test(ts) ? dayjs(ts) : dayjs.unix(typeof ts === 'string' ? Number.parseInt(ts, 10) : ts)
  return useLocaleDate ? d.format('YYYY/MM/DD') : d.format('YYYY/MM/DD HH:mm:ss')
}

export function formatFileSize(bytes?: number): string {
  if (!bytes || bytes <= 0) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

export function getUrlExtension(url?: string): string {
  if (!url) return ''
  const path = url.split('?')[0].split('#')[0]
  const base = path.slice(path.lastIndexOf('/') + 1)
  const dot = base.lastIndexOf('.')
  if (dot <= 0) return ''
  return base.slice(dot + 1).toUpperCase()
}

export function withDownloadFlag(url: string): string {
  if (!url) return ''
  return url.includes('?') ? `${url}&download=1` : `${url}?download=1`
}

export function getPreviewText(content: string, maxLength: number = 100): string {
  if (!content) return ''
  const plainText = removeMarkdown(content).replace(/\s+/g, ' ').trim()
  return plainText.length > maxLength ? plainText.substring(0, maxLength) + '...' : plainText
}

export function prefersReducedMotion(): boolean {
  return globalThis.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
}

export function scrollBehavior(): ScrollBehavior {
  return prefersReducedMotion() ? 'auto' : 'smooth'
}

export function renderMarkdown(text: string): string {
  try {
    return DOMPurify.sanitize(marked(text) as string)
  } catch {
    return DOMPurify.sanitize(text)
  }
}

async function fileMd5(file: File): Promise<string> {
  const CHUNK = 4 * 1024 * 1024
  const spark = new SparkMD5.ArrayBuffer()
  for (let offset = 0; offset < file.size; offset += CHUNK) {
    const buf = await file.slice(offset, offset + CHUNK).arrayBuffer()
    spark.append(buf)
  }
  return spark.end()
}

export async function renameFileToMd5(file: File): Promise<File> {
  const md5 = await fileMd5(file)
  const dot = file.name.lastIndexOf('.')
  const ext = dot > -1 ? file.name.slice(dot + 1).toLowerCase() : 'bin'
  return new File([file], `${md5}.${ext}`, { type: file.type, lastModified: file.lastModified })
}

export function toFormData(obj: Record<string, unknown>): FormData {
  const fd = new FormData()
  for (const [key, value] of Object.entries(obj)) {
    if (value === null || value === undefined) continue
    if (typeof value === 'string' && value === '') continue
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item !== null && item !== undefined) fd.append(key, String(item))
      }
      continue
    }
    if (value instanceof File) {
      fd.append(key, value)
    } else {
      fd.append(key, String(value))
    }
  }
  return fd
}
