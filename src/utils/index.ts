import { marked } from 'marked'
import DOMPurify from 'dompurify'

export function getImageUrl(image?: string, filePath?: string, contentType?: string): string {
  if (image) {
    return image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
  }
  if (!filePath) return ''
  if (contentType === 'video') {
    return `https://xqapi.xiey.work/uploads/${filePath}`
  }
  const thumbPath = filePath.includes('_thumb.') ? filePath : filePath.replace(/\.[^.]+$/, '_thumb.webp')
  return `https://xqapi.xiey.work/thumbnails/${thumbPath}`
}

export function getPreviewText(content: string, maxLength: number = 100): string {
  const text = content.replace(/<[^>]+>/g, '').replace(/\s+/g, ' ').trim()
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength) + '...'
}

export function renderMarkdown(text: string): string {
  return DOMPurify.sanitize(marked(text) as string)
}
