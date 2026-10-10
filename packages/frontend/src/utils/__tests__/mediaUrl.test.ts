import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

afterEach(() => { vi.unstubAllEnvs(); vi.resetModules() })

// 同源开发环境掩盖过素材误加媒体域名的问题；两种配置必须执行同一份契约。
describe.each(['', 'https://media.example.invalid'])('media URL contract (base=%s)', mediaBase => {
  beforeEach(() => { vi.stubEnv('VITE_MEDIA_BASE_URL', mediaBase); vi.resetModules() })

  it.each(['/thumbs/a_thumb.webp', '/uploads/a.png', '/images/legacy.jpg?v=2'])('resolves API media once: %s', async path => {
    const { getImageUrl, isContentMediaPath } = await import('../index')
    expect(isContentMediaPath(path)).toBe(true)
    expect(getImageUrl(path)).toBe(`${mediaBase}${path}`)
    expect(getImageUrl(getImageUrl(path))).toBe(`${mediaBase}${path}`)
  })

  it.each([
    '/assets/bg-buildhash.webp', '/src/assets/bg.webp', '/icons/user-circle.svg',
    '/thumbs-other/a.webp', '//images.example.invalid/a.webp',
    'https://external.example.invalid/thumbs/a.webp?signature=abc%2Fdef',
    'blob:https://xq.example.invalid/image-id', 'data:image/png;base64,AAAA',
  ])('preserves asset or external URL: %s', async url => {
    const { getImageUrl } = await import('../index')
    expect(getImageUrl(url)).toBe(url)
  })

  it.each([
    '/assets/missing.webp', '/icons/user-circle.svg', '/src/assets/bg.webp',
    'https://media.example.invalid/assets/missing.webp',
    'https://external.example.invalid/thumbs/a.webp', 'data:image/png;base64,AAAA',
  ])('never falls back non-owned media or frontend assets: %s', async url => {
    const { getRemoteFallbackUrl } = await import('../index')
    expect(getRemoteFallbackUrl(url, 'https://xq.example.invalid')).toBe('')
  })

  it('falls back owned media without rewriting the file name or query', async () => {
    const { getImageUrl, getRemoteFallbackUrl, REMOTE_MEDIA_BASE } = await import('../index')
    const path = '/thumbs/a_thumb.webp?v=2'
    expect(getRemoteFallbackUrl(getImageUrl(path), 'https://xq.example.invalid')).toBe(`${REMOTE_MEDIA_BASE}${path}`)
  })
})
