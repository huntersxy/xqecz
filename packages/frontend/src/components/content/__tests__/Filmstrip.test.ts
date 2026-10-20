import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Filmstrip from '../Filmstrip.vue'
import { rememberLoaded } from '@/utils/mediaMemory'

const items = [
  { id: 1, title: '作品一', thumb: '/thumbs/abc_thumb.png', isCurrent: true },
  { id: 2, title: '作品二', thumb: '/thumbs/def_thumb.png', isCurrent: false },
]

describe('Filmstrip thumbnails', () => {
  afterEach(() => localStorage.clear())

  it('loads a thumbnail from the host that served it last time', () => {
    rememberLoaded('/thumbs/abc_thumb.png', 'https://api39.xiey.work/thumbs/abc_thumb.png')
    const wrapper = mount(Filmstrip, { props: { items, hasPrev: false, hasNext: false } })
    const imgs = wrapper.findAll('img')
    expect(imgs[0].attributes('src')).toBe('https://api39.xiey.work/thumbs/abc_thumb.png')
    expect(imgs[1].attributes('src')).toBe('/thumbs/def_thumb.png')
  })
})
