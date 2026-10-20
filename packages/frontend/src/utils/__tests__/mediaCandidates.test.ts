import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mediaCandidates, noteFailed, noteLoaded } from '../contentMedia'
import type { Content } from '@/types'

vi.mock('@/utils/imageSource', () => ({
  pickImageUrl: (local: string, mirror: string, mirror2: string, source: string | null) =>
    source === 'mirror2' ? mirror2 : source === 'mirror' ? mirror : local,
  fallbackChain: (local: string, mirror: string, _mirror2: string, source: string | null) =>
    source === 'mirror2' ? [mirror, local].filter(Boolean) : source === 'mirror' ? [local] : [],
}))

function content(overrides: Partial<Content> = {}): Content {
  return {
    id: 1, title: '', img: '/uploads/abc.png', video: '',
    mirror_img: 'https://mirror.example/uploads/abc.png',
    mirror2_img: 'https://mirror2.example/uploads/abc.png',
    user: { id: 1, username: '作者' }, tags: [], created_at: '2026-10-10T00:00:00.000Z',
    ...overrides,
  } as Content
}

beforeEach(() => localStorage.clear())

describe('mediaCandidates', () => {
  it('puts the URL that loaded last time first, even when the probe now picks another source', () => {
    noteLoaded(content(), 'https://mirror2.example/uploads/abc.png')
    expect(mediaCandidates(content(), 'mirror')).toEqual([
      'https://mirror2.example/uploads/abc.png',
      'https://mirror.example/uploads/abc.png',
      '/uploads/abc.png',
      'https://api39.xiey.work/uploads/abc.png',
    ])
  })

  it('drops a remembered URL that no longer matches the file and falls back to the probe', () => {
    noteLoaded(content(), 'https://mirror2.example/uploads/other.png')
    expect(mediaCandidates(content(), 'origin')).toEqual([
      '/uploads/abc.png',
      'https://api39.xiey.work/uploads/abc.png',
    ])
  })

  it('returns nothing until a source has been chosen', () => {
    expect(mediaCandidates(content(), null)).toEqual([])
  })

  it('forgets the remembered URL once it fails, so the next attempt starts over', () => {
    noteLoaded(content(), 'https://mirror2.example/uploads/abc.png')
    noteFailed(content(), 'https://mirror2.example/uploads/abc.png')
    expect(mediaCandidates(content(), 'origin')[0]).toBe('/uploads/abc.png')
  })
})
