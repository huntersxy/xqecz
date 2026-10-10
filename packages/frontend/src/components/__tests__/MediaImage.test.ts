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
      'https://api39.xiey.work/thumbs/broken.jpg'
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

  it('shows the provided loader until load completes and restores it when the source changes', async () => {
    const wrapper = mount(MediaImage, {
      props: { src: 'https://images.example/art.png', preview: false },
      slots: { loader: '<div role="status">正在加载作品图片…</div>' },
    })
    await nextTick()
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    await wrapper.get('.arco-image-img').trigger('load')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)

    await wrapper.setProps({ src: 'https://images.example/next.png' })
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    await wrapper.get('.arco-image-img').trigger('load')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('keeps the loader visible across fallback attempts and replaces it with the final error', async () => {
    const wrapper = mount(MediaImage, {
      props: {
        src: 'https://first.example/art.png',
        fallbackSrc: ['https://second.example/art.png'],
        preview: false,
      },
      slots: {
        loader: '<div role="status">正在加载作品图片…</div>',
        error: '<div role="status">图片加载失败，请稍后重试</div>',
      },
    })
    await failImage(wrapper)
    expect(wrapper.get('.arco-image-img').attributes('src')).toBe('https://second.example/art.png')
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    expect(wrapper.emitted('error')).toBeUndefined()
    expect(wrapper.emitted('fallback')).toEqual([['https://first.example/art.png']])

    await failImage(wrapper)
    expect(wrapper.get('[role="status"]').text()).toBe('图片加载失败，请稍后重试')
    expect(wrapper.emitted('error')).toHaveLength(1)
    wrapper.unmount()
  })

})
