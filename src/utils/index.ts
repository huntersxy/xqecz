import { marked } from 'marked'
import DOMPurify from 'dompurify'

export function getImageUrl(image?: string, _filePath?: string): string {
  if (image) {
    let url = image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
    if (url.startsWith('/')) {
      url = `https://xqapi.xiey.work${url}`
    }
    return url
  }
  return ''
}

export function getPreviewText(content: string, maxLength: number = 100): string {
  const text = content.replace(/<[^>]+>/g, '').replace(/\s+/g, ' ').trim()
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength) + '...'
}

export function renderMarkdown(text: string): string {
  return DOMPurify.sanitize(marked(text) as string)
}
