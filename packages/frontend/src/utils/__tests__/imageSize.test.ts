import { describe, expect, it } from 'vitest'
import { formatImageRatio, formatImageSize, sameImageSize } from '@/utils/imageSize'

describe('image size labels', () => {
  it('turns common provider dimensions into readable ratio labels', () => {
    expect(formatImageSize('1024x1024')).toBe('1024×1024 · 1:1')
    expect(formatImageSize('1024x576')).toBe('1024×576 · 16:9')
    expect(formatImageRatio('1024x683')).toBe('≈3:2')
  })

  it('matches equivalent x and multiplication-sign notation', () => {
    expect(sameImageSize('1024x768', '1024×768')).toBe(true)
    expect(sameImageSize('1024x768', '768x1024')).toBe(false)
  })
})
