import { marked } from 'marked'
import DOMPurify from 'dompurify'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import removeMarkdown from 'remove-markdown'
import SparkMD5 from 'spark-md5'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

const MEDIA_BASE = import.meta.env.VITE_MEDIA_BASE_URL || ''

export function getImageUrl(image?: string): string {
  if (!image) return ''
  if (image.startsWith('http://') || image.startsWith('https://')) return image
  if (image.startsWith('/')) {
    return `${MEDIA_BASE}${image}`
  }
  return image
}

/**
 * 根据邮箱生成头像 URL：QQ 邮箱用 QQ 头像接口，其余回退 Gravatar。
 * @param email 用户邮箱
 * @param size 头像尺寸，默认 80
 */
export function getAvatarUrl(email: string, size: number = 80): string {
  if (!email) return ''
  // QQ 邮箱 → QQ 头像接口
  const qqMatch = /^(\d{5,11})@qq\.com$/i.exec(email.trim())
  if (qqMatch) {
    return `https://q.qlogo.cn/headimg_dl?dst_uin=${qqMatch[1]}&spec=100`
  }
  // 其余走 Gravatar
  const hash = md5Hex(email.trim().toLowerCase())
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=${size}`
}

// 简单 MD5 hex。Gravatar 需要小写 hex MD5 —— 复用顶部 import 的 SparkMD5。
function md5Hex(input: string): string {
  return SparkMD5.hash(input)
}

export function formatTime(ts: number | string, useLocaleDate: boolean = false): string {
  if (!ts) return ''
  // ISO 字符串（如 '2026-07-15T03:08:20.763Z'）直接解析；数字或纯数字字符串走 Unix 秒
  const d = typeof ts === 'string' && /^\d{4}-/.test(ts) ? dayjs(ts) : dayjs.unix(typeof ts === 'string' ? Number.parseInt(ts, 10) : ts)
  return useLocaleDate ? d.format('YYYY/MM/DD') : d.format('YYYY/MM/DD HH:mm:ss')
}

export function formatRelativeTime(ts: number | string): string {
  if (!ts) return ''
  const d = typeof ts === 'string' && /^\d{4}-/.test(ts) ? dayjs(ts) : dayjs.unix(typeof ts === 'string' ? Number.parseInt(ts, 10) : ts)
  return d.fromNow()
}

export function getPreviewText(content: string, maxLength: number = 100): string {
  if (!content) return ''
  const plainText = removeMarkdown(content).replace(/\s+/g, ' ').trim()
  return plainText.length > maxLength ? plainText.substring(0, maxLength) + '...' : plainText
}

export function renderMarkdown(text: string): string {
  try {
    return DOMPurify.sanitize(marked(text) as string)
  } catch {
    return DOMPurify.sanitize(text)
  }
}

/**
 * 计算文件内容的 MD5（spark-md5 分片读取，避免大文件一次性载入内存）。
 */
export async function fileMd5(file: File): Promise<string> {
  const CHUNK = 4 * 1024 * 1024 // 4MB 分片
  const spark = new SparkMD5.ArrayBuffer()
  for (let offset = 0; offset < file.size; offset += CHUNK) {
    const buf = await file.slice(offset, offset + CHUNK).arrayBuffer()
    spark.append(buf)
  }
  return spark.end()
}

/**
 * 上传前把文件重命名为 `<md5>.<ext>`（内容寻址）。
 * 后端按此文件名扁平落盘到 data/uploads/，同内容文件天然去重。
 */
export async function renameFileToMd5(file: File): Promise<File> {
  const md5 = await fileMd5(file)
  const dot = file.name.lastIndexOf('.')
  const ext = dot > -1 ? file.name.slice(dot + 1).toLowerCase() : 'bin'
  return new File([file], `${md5}.${ext}`, { type: file.type, lastModified: file.lastModified })
}

/**
 * 把普通对象转为 FormData：跳过 null/undefined/空串/空数组；
 * 数组按元素重复 append（适配后端同名多值字段，如 tags）；File 直接 append；
 * 其余（number/boolean）转字符串。消除各 api 里手搓的 `if (x) fd.append(...)` 样板。
 */
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
