import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import UserAvatar from '../UserAvatar.vue'
import defaultAvatar from '@/assets/avatar-default.svg'

const wrappers: ReturnType<typeof mount>[] = []
function create(props: { src?: string; email?: string; username?: string; size?: number } = {}) {
  const wrapper = mount(UserAvatar, { props })
  wrappers.push(wrapper)
  return wrapper
}
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

describe('UserAvatar', () => {
  it('uses a bundled default avatar when email or avatar is missing', () => {
    for (const email of [undefined, '', '   ']) {
      const wrapper = create({ email })
      expect(wrapper.find('.site-avatar-default').attributes('src')).toBe(defaultAvatar)
      expect(wrapper.find('.site-avatar-image').exists()).toBe(false)
    }
  })
  it('uses QQ avatars for QQ email and normalizes Gravatar email', () => {
    expect(create({ email: ' 12345@QQ.COM ' }).find('.site-avatar-image').attributes('src')).toBe('https://q.qlogo.cn/headimg_dl?dst_uin=12345&spec=100')
    expect(create({ email: ' Test@Example.com ' }).find('.site-avatar-image').attributes('src')).toBe('https://www.gravatar.com/avatar/55502f40dc8b7c769880b10874abc9d0?d=identicon&s=80')
  })
  it('prefers the public avatar URL over email and keeps a default while loading', async () => {
    const wrapper = create({ src: 'https://example.com/avatar.png', email: '12345@qq.com', username: '作者' })
    const image = wrapper.find('.site-avatar-image')
    expect(image.attributes('src')).toBe('https://example.com/avatar.png')
    expect(image.attributes('referrerpolicy')).toBe('no-referrer')
    expect(image.classes()).not.toContain('is-loaded')
    expect(wrapper.attributes('aria-label')).toBe('作者的头像')
    await image.trigger('load')
    expect(image.classes()).toContain('is-loaded')
  })
  it('falls back locally after an external avatar fails and retries when the source changes', async () => {
    const wrapper = create({ email: 'test@example.com' })
    await wrapper.find('.site-avatar-image').trigger('error')
    expect(wrapper.find('.site-avatar-image').exists()).toBe(false)
    expect(wrapper.find('.site-avatar-default').exists()).toBe(true)
    await wrapper.setProps({ email: '12345@qq.com' })
    expect(wrapper.find('.site-avatar-image').attributes('src')).toContain('q.qlogo.cn')
    await wrapper.setProps({ email: 'test@example.com' })
    expect(wrapper.find('.site-avatar-image').exists()).toBe(true)
  })
  it('ignores load failures from an avatar belonging to the previous user', async () => {
    const wrapper = create({ src: 'https://example.com/old.png' })
    const oldImage = wrapper.find('.site-avatar-image').element
    await wrapper.setProps({ src: 'https://example.com/new.png' })
    oldImage.dispatchEvent(new Event('error'))
    expect(wrapper.find('.site-avatar-image').attributes('src')).toBe('https://example.com/new.png')
  })
})
