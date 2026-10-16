export interface ParsedImageSize {
  width: number
  height: number
}

export type ImageSizeTier = '1K' | '2K' | '4K'

export const AUTO_IMAGE_SIZE = 'auto'
export const IMAGE_SIZE_TIERS: readonly ImageSizeTier[] = ['1K', '2K', '4K']
export const IMAGE_SIZE_RATIOS = [
  { label: '1:1', value: '1:1' },
  { label: '3:2', value: '3:2' },
  { label: '2:3', value: '2:3' },
  { label: '16:9', value: '16:9' },
  { label: '9:16', value: '9:16' },
  { label: '4:3', value: '4:3' },
  { label: '3:4', value: '3:4' },
  { label: '21:9', value: '21:9' },
] as const

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

const SIZE_MULTIPLE = 16
const MAX_EDGE = 3840
const MAX_ASPECT_RATIO = 3
const MIN_PIXELS = 655_360
const MAX_PIXELS = 8_294_400
const TIER_PIXEL_BUDGET: Record<ImageSizeTier, number> = {
  '1K': 1_572_864,
  '2K': 4_194_304,
  '4K': MAX_PIXELS,
}

type PresetRatio = '1:1' | '3:2' | '2:3' | '16:9' | '9:16' | '4:3' | '3:4' | '21:9'

const COMMON_SIZE_PRESETS: Record<ImageSizeTier, Record<PresetRatio, string>> = {
  '1K': {
    '1:1': '1024x1024',
    '3:2': '1536x1024',
    '2:3': '1024x1536',
    '16:9': '1280x720',
    '9:16': '720x1280',
    '4:3': '1024x768',
    '3:4': '768x1024',
    '21:9': '1280x544',
  },
  '2K': {
    '1:1': '2048x2048',
    '3:2': '2160x1440',
    '2:3': '1440x2160',
    '16:9': '2560x1440',
    '9:16': '1440x2560',
    '4:3': '2048x1536',
    '3:4': '1536x2048',
    '21:9': '2560x1088',
  },
  '4K': {
    '1:1': '2880x2880',
    '3:2': '3456x2304',
    '2:3': '2304x3456',
    '16:9': '3840x2160',
    '9:16': '2160x3840',
    '4:3': '3200x2400',
    '3:4': '2400x3200',
    '21:9': '3840x1600',
  },
}

export function parseImageSize(value: string): ParsedImageSize | null {
  const match = /^\s*(\d+)\s*[x×]\s*(\d+)\s*$/i.exec(value)
  if (!match) return null
  const width = Number(match[1])
  const height = Number(match[2])
  return width > 0 && height > 0 ? { width, height } : null
}

export function parseImageRatio(value: string): { width: number; height: number } | null {
  const match = /^\s*(\d+(?:\.\d+)?)\s*[:x×]\s*(\d+(?:\.\d+)?)\s*$/i.exec(value)
  if (!match) return null
  const width = Number(match[1])
  const height = Number(match[2])
  return Number.isFinite(width) && Number.isFinite(height) && width > 0 && height > 0
    ? { width, height }
    : null
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
  if (value.trim().toLowerCase() === AUTO_IMAGE_SIZE) return '自动'
  const parsed = parseImageSize(value)
  if (!parsed) return value || '未设置'
  return `${parsed.width}×${parsed.height} · ${formatImageRatio(parsed)}`
}

export function sameImageSize(left: string, right: string): boolean {
  const a = parseImageSize(left)
  const b = parseImageSize(right)
  return !!a && !!b && a.width === b.width && a.height === b.height
}

function roundToMultiple(value: number, multiple: number): number {
  return Math.max(multiple, Math.round(value / multiple) * multiple)
}

function floorToMultiple(value: number, multiple: number): number {
  return Math.max(multiple, Math.floor(value / multiple) * multiple)
}

function ceilToMultiple(value: number, multiple: number): number {
  return Math.max(multiple, Math.ceil(value / multiple) * multiple)
}

function normalizeDimensions(width: number, height: number): ParsedImageSize {
  let normalizedWidth = roundToMultiple(width, SIZE_MULTIPLE)
  let normalizedHeight = roundToMultiple(height, SIZE_MULTIPLE)

  const scaleToFit = (scale: number) => {
    normalizedWidth = floorToMultiple(normalizedWidth * scale, SIZE_MULTIPLE)
    normalizedHeight = floorToMultiple(normalizedHeight * scale, SIZE_MULTIPLE)
  }
  const scaleToFill = (scale: number) => {
    normalizedWidth = ceilToMultiple(normalizedWidth * scale, SIZE_MULTIPLE)
    normalizedHeight = ceilToMultiple(normalizedHeight * scale, SIZE_MULTIPLE)
  }

  for (let i = 0; i < 4; i += 1) {
    const maxEdge = Math.max(normalizedWidth, normalizedHeight)
    if (maxEdge > MAX_EDGE) scaleToFit(MAX_EDGE / maxEdge)

    if (normalizedWidth / normalizedHeight > MAX_ASPECT_RATIO) {
      normalizedWidth = floorToMultiple(normalizedHeight * MAX_ASPECT_RATIO, SIZE_MULTIPLE)
    } else if (normalizedHeight / normalizedWidth > MAX_ASPECT_RATIO) {
      normalizedHeight = floorToMultiple(normalizedWidth * MAX_ASPECT_RATIO, SIZE_MULTIPLE)
    }

    const pixels = normalizedWidth * normalizedHeight
    if (pixels > MAX_PIXELS) scaleToFit(Math.sqrt(MAX_PIXELS / pixels))
    else if (pixels < MIN_PIXELS) scaleToFill(Math.sqrt(MIN_PIXELS / pixels))
  }
  return { width: normalizedWidth, height: normalizedHeight }
}

export function normalizeImageSize(value: string): string {
  const trimmed = value.trim()
  if (trimmed.toLowerCase() === AUTO_IMAGE_SIZE) return AUTO_IMAGE_SIZE
  const parsed = parseImageSize(trimmed)
  if (!parsed) return trimmed
  const normalized = normalizeDimensions(parsed.width, parsed.height)
  return `${normalized.width}x${normalized.height}`
}

function presetRatioKey(width: number, height: number): PresetRatio | null {
  if (!Number.isInteger(width) || !Number.isInteger(height)) return null
  const divisor = gcd(width, height)
  const key = `${width / divisor}:${height / divisor}`
  return key in COMMON_SIZE_PRESETS['1K'] ? (key as PresetRatio) : null
}

export function calculateImageSize(tier: ImageSizeTier, ratio: string): string | null {
  const parsed = parseImageRatio(ratio)
  if (!parsed) return null
  const preset = presetRatioKey(parsed.width, parsed.height)
  if (preset) return COMMON_SIZE_PRESETS[tier][preset]

  const targetRatio = parsed.width / parsed.height
  const pixelBudget = TIER_PIXEL_BUDGET[tier]
  let bestWidth = 0
  let bestHeight = 0
  let bestPixels = 0

  for (let width = SIZE_MULTIPLE; width <= MAX_EDGE; width += SIZE_MULTIPLE) {
    const idealHeight = width / targetRatio
    const candidates = [
      Math.floor(idealHeight / SIZE_MULTIPLE) * SIZE_MULTIPLE,
      Math.ceil(idealHeight / SIZE_MULTIPLE) * SIZE_MULTIPLE,
    ]
    for (const height of candidates) {
      if (height < SIZE_MULTIPLE || height > MAX_EDGE) continue
      const pixels = width * height
      if (pixels > pixelBudget || pixels < MIN_PIXELS) continue
      if (Math.max(width / height, height / width) > MAX_ASPECT_RATIO) continue
      const ratioError = Math.abs(width / height - targetRatio) / targetRatio
      if (ratioError > 0.01) continue
      if (pixels > bestPixels) {
        bestPixels = pixels
        bestWidth = width
        bestHeight = height
      }
    }
  }
  return bestPixels ? `${bestWidth}x${bestHeight}` : null
}
