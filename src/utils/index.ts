import { marked } from 'marked'
import DOMPurify from 'dompurify'

const API_BASE_URL = 'https://xqapi.xiey.work'

export function getImageUrl(image?: string, filePath?: string): string {
  const url = image || filePath || ''
  if (!url) return ''
  if (url.startsWith('http')) return url
  return `${API_BASE_URL}/uploads/${url}`
}

export function getPreviewText(content: string, maxLength: number = 100): string {
  const text = content.replace(/<[^>]+>/g, '').replace(/\s+/g, ' ').trim()
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength) + '...'
}

export function renderMarkdown(text: string): string {
  return DOMPurify.sanitize(marked(text) as string)
}
