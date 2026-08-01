import { describe, it, expect } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import MediaImage from '../MediaImage.vue'

async function failImage(wrapper: ReturnType<typeof mount>) {
  await nextTick()
  await wrapper.find('.arco-image-img').trigger('error')
  await nextTick()
}

describe('MediaImage', () => {
  it('renders the resolved dev URL', async () => {
    const wrapper = mount(MediaImage, { props: { src: '/thumbs/a.jpg', alt: 'A' } })
    await nextTick()
    expect(wrapper.find('.arco-image-img').attributes('src')).toBe('/thumbs/a.jpg')
    wrapper.unmount()
  })

  it('falls back to the production host when the dev URL fails', async () => {
    const wrapper = mount(MediaImage, { props: { src: '/thumbs/broken.jpg' } })
    await failImage(wrapper)
    expect(wrapper.find('.arco-image-img').attributes('src')).toBe(
      'https://xq.xiey.work/thumbs/broken.jpg'
    )
    expect(wrapper.emitted('error')).toBeUndefined()
    wrapper.unmount()
  })

  it('emits error and shows the broken placeholder when production also fails', async () => {
    const wrapper = mount(MediaImage, { props: { src: '/thumbs/broken.jpg' } })
    await failImage(wrapper)
    await failImage(wrapper)
    expect(wrapper.emitted('error')).toHaveLength(1)
    expect(wrapper.find('.arco-image-error').exists()).toBe(true)
    wrapper.unmount()
  })

  it('does not fall back for external image hosts', async () => {
    const wrapper = mount(MediaImage, { props: { src: 'https://q.qlogo.cn/headimg.png' } })
    await failImage(wrapper)
    expect(wrapper.find('.arco-image-img').attributes('src')).toBe(
      'https://q.qlogo.cn/headimg.png'
    )
    expect(wrapper.emitted('error')).toHaveLength(1)
    wrapper.unmount()
  })
})
