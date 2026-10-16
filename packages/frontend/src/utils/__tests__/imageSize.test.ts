import { describe, expect, it } from 'vitest'
import {
  calculateImageSize,
  formatImageRatio,
  formatImageSize,
  normalizeImageSize,
  parseImageRatio,
  sameImageSize,
} from '@/utils/imageSize'

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

  it('supports automatic sizing and 1K/2K/4K ratio presets', () => {
    expect(formatImageSize('auto')).toBe('自动')
    expect(calculateImageSize('1K', '16:9')).toBe('1280x720')
    expect(calculateImageSize('2K', '16:9')).toBe('2560x1440')
    expect(calculateImageSize('4K', '16:9')).toBe('3840x2160')
    expect(parseImageRatio('2.39:1')).toEqual({ width: 2.39, height: 1 })
  })

  it('normalizes custom dimensions into the safe generation range', () => {
    expect(normalizeImageSize('5000x5000')).toBe('2880x2880')
    expect(normalizeImageSize('100x4000')).toBe('480x1408')
  })
})
