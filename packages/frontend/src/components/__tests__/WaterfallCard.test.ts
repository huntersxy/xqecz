import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import WaterfallCard from '../WaterfallCard.vue'
import { ContentSchema } from '@/types'

function textItem(overrides: Record<string, unknown> = {}) {
  return ContentSchema.parse({
    id: 1,
    title: '纯文本作品',
    text: '**加粗** 的内容摘要，用于测试纯文本卡片展示。',
    thumb: '',
    video: '',
    img: '',
    origin: '',
    file_size: 0,
    user: { id: 1, username: '作者' },
    avatar_url: '',
    tags: ['AI'],
    like_count: 3,
    created_at: 1720000000,
    ...overrides,
  })
}

describe('WaterfallCard text branch', () => {
  it('renders a markdown-stripped excerpt with a read-more hint', () => {
    const wrapper = mount(WaterfallCard, { props: { item: textItem() } })
    const excerpt = wrapper.find('.wf-card-text-excerpt')
    expect(excerpt.text()).toContain('内容摘要')
    expect(excerpt.text()).not.toContain('**')
    expect(wrapper.find('.wf-card-text-more').text()).toBe('阅读全文')
    wrapper.unmount()
  })

  it('falls back to the title when the content has no text', () => {
    const wrapper = mount(WaterfallCard, { props: { item: textItem({ text: '' }) } })
    expect(wrapper.find('.wf-card-text-excerpt').text()).toContain('纯文本作品')
    wrapper.unmount()
  })
})
