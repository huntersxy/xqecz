import { describe, expect, it, beforeEach } from 'vitest'
import { forgetLoaded, mediaKeyOf, recallLoaded, rememberLoaded, sameMedia } from '../mediaMemory'

describe('media key', () => {
  it('keys by directory and filename, ignoring host and query', () => {
    expect(mediaKeyOf('https://mirror.example/uploads/abc.png?x=1')).toBe('uploads/abc.png')
    expect(mediaKeyOf('/thumbs/abc_thumb.png')).toBe('thumbs/abc_thumb.png')
    expect(mediaKeyOf('https://cdn.example/other/abc.png')).toBeNull()
    expect(mediaKeyOf('')).toBeNull()
  })

  it('treats the same file on different hosts as the same media', () => {
    expect(sameMedia('https://mirror.example/uploads/abc.png', '/uploads/abc.png')).toBe(true)
    expect(sameMedia('/uploads/abc.png', '/uploads/other.png')).toBe(false)
    expect(sameMedia('/thumbs/abc.png', '/uploads/abc.png')).toBe(false)
  })
})

describe('loaded memory', () => {
  beforeEach(() => localStorage.clear())

  it('recalls the URL that actually loaded for the same file', () => {
    rememberLoaded('/uploads/abc.png', 'https://mirror.example/uploads/abc.png')
    expect(recallLoaded('/uploads/abc.png')).toBe('https://mirror.example/uploads/abc.png')
  })

  it('refuses a remembered URL that points at a different file', () => {
    rememberLoaded('/uploads/abc.png', 'https://mirror.example/uploads/other.png')
    expect(recallLoaded('/uploads/abc.png')).toBeNull()
  })

  it('drops entries past the retention window', () => {
    rememberLoaded('/uploads/abc.png', 'https://mirror.example/uploads/abc.png', 1)
    expect(recallLoaded('/uploads/abc.png', 31 * 24 * 60 * 60 * 1000)).toBeNull()
  })

  it('forgets a file that failed to load', () => {
    rememberLoaded('/uploads/abc.png', 'https://mirror.example/uploads/abc.png')
    forgetLoaded('/uploads/abc.png')
    expect(recallLoaded('/uploads/abc.png')).toBeNull()
  })
})
