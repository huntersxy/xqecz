export interface ParsedImageSize {
  width: number
  height: number
}

const COMMON_RATIOS = [
  { width: 1, height: 1, label: '1:1' },
  { width: 4, height: 3, label: '4:3' },
  { width: 3, height: 4, label: '3:4' },
  { width: 3, height: 2, label: '3:2' },
  { width: 2, height: 3, label: '2:3' },
  { width: 16, height: 9, label: '16:9' },
  { width: 9, height: 16, label: '9:16' },
  { width: 21, height: 9, label: '21:9' },
  { width: 9, height: 21, label: '9:21' },
]

export function parseImageSize(value: string): ParsedImageSize | null {
  const match = /^\s*(\d+)\s*[x×]\s*(\d+)\s*$/i.exec(value)
  if (!match) return null
  const width = Number(match[1])
  const height = Number(match[2])
  return width > 0 && height > 0 ? { width, height } : null
}

function gcd(a: number, b: number): number {
  while (b) [a, b] = [b, a % b]
  return a
}

export function formatImageRatio(value: string | ParsedImageSize): string {
  const parsed = typeof value === 'string' ? parseImageSize(value) : value
  if (!parsed) return '自定义比例'
  const exactDivisor = gcd(parsed.width, parsed.height)
  const exactWidth = parsed.width / exactDivisor
  const exactHeight = parsed.height / exactDivisor
  if (exactWidth <= 32 && exactHeight <= 32) return `${exactWidth}:${exactHeight}`

  const actual = parsed.width / parsed.height
  let closest = COMMON_RATIOS[0]
  let distance = Math.abs(actual - closest.width / closest.height)
  for (const ratio of COMMON_RATIOS.slice(1)) {
    const nextDistance = Math.abs(actual - ratio.width / ratio.height)
    if (nextDistance < distance) {
      closest = ratio
      distance = nextDistance
    }
  }
  return `≈${closest.label}`
}

export function formatImageSize(value: string): string {
  const parsed = parseImageSize(value)
  if (!parsed) return value || '未设置'
  return `${parsed.width}×${parsed.height} · ${formatImageRatio(parsed)}`
}

export function sameImageSize(left: string, right: string): boolean {
  const a = parseImageSize(left)
  const b = parseImageSize(right)
  return !!a && !!b && a.width === b.width && a.height === b.height
}
