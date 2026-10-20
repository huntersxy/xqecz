import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TagInput from '../TagInput.vue'
import SiteOverlay from '../SiteOverlay.vue'
import SiteImage from '../SiteImage.vue'
import SiteCheckbox from '../SiteCheckbox.vue'

afterEach(() => { document.body.innerHTML = '' })

describe('TagInput', () => {
  it('commits trimmed, de-duplicated tags on Enter without submitting the form', async () => {
    const wrapper = mount(TagInput, { props: { modelValue: ['校园'] } })
    const input = wrapper.get('input')
    await input.setValue('  夏日  ')
    const enter = new KeyboardEvent('keydown', { key: 'Enter', cancelable: true })
    input.element.dispatchEvent(enter)
    expect(enter.defaultPrevented).toBe(true)
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual([['校园', '夏日']])

    await input.setValue('校园')
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('update:modelValue')).toHaveLength(1)
  })

  it('ignores Enter while an IME composition is active', async () => {
    const wrapper = mount(TagInput, { props: { modelValue: [] } })
    await wrapper.get('input').setValue('ni')
    await wrapper.get('input').trigger('keydown', { key: 'Enter', isComposing: true })
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('commits pending text on blur, removes by chip and by Backspace on an empty draft', async () => {
    const wrapper = mount(TagInput, { props: { modelValue: ['a', 'b'] } })
    await wrapper.get('input').setValue('c')
    await wrapper.get('input').trigger('blur')
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual([['a', 'b', 'c']])

    await wrapper.get('[aria-label="移除标签 a"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual([['b']])

    await wrapper.get('input').trigger('keydown', { key: 'Backspace' })
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual([['a']])
  })

  it('does nothing when disabled', async () => {
    const wrapper = mount(TagInput, { props: { modelValue: ['a'], disabled: true } })
    expect(wrapper.get('input').attributes('disabled')).toBeDefined()
    await wrapper.get('[aria-label="移除标签 a"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})

describe('SiteCheckbox', () => {
  it('toggles its v-model and respects disabled', async () => {
    const wrapper = mount(SiteCheckbox, { props: { modelValue: false, 'onUpdate:modelValue': (v: boolean) => wrapper.setProps({ modelValue: v }) }, slots: { default: '同意' } })
    await wrapper.get('input').setValue(true)
    expect(wrapper.props('modelValue')).toBe(true)
    await wrapper.setProps({ disabled: true })
    expect(wrapper.get('input').attributes('disabled')).toBeDefined()
  })
})

describe('SiteOverlay', () => {
  function mountOverlay(props: Record<string, unknown> = {}) {
    return mount(SiteOverlay, {
      props: { modelValue: true, label: '测试', ...props },
      slots: { default: '<button class="inner">内部</button>' },
      attachTo: document.body,
    })
  }

  it('teleports a labelled modal dialog to body and forwards attrs to the root', async () => {
    const wrapper = mountOverlay({ class: 'custom-root' })
    const root = document.body.querySelector('.site-overlay')!
    expect(root.classList.contains('custom-root')).toBe(true)
    const dialog = root.querySelector('[role="dialog"]')!
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(dialog.getAttribute('aria-label')).toBe('测试')
    expect(wrapper.element.contains(root)).toBe(false)
    wrapper.unmount()
  })

  it('closes on Escape and mask click, but not when those are disabled', async () => {
    const wrapper = mountOverlay()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual([false])
    document.body.querySelector<HTMLElement>('.site-overlay-mask')!.click()
    expect(wrapper.emitted('update:modelValue')).toHaveLength(2)
    wrapper.unmount()

    const locked = mountOverlay({ closeOnEsc: false, closeOnMask: false })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    document.body.querySelector<HTMLElement>('.site-overlay-mask')!.click()
    expect(locked.emitted('update:modelValue')).toBeUndefined()
    locked.unmount()
  })

  it('gives Escape only to the topmost overlay', async () => {
    const below = mountOverlay()
    const above = mountOverlay()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(above.emitted('update:modelValue')).toHaveLength(1)
    expect(below.emitted('update:modelValue')).toBeUndefined()
    above.unmount()
    below.unmount()
  })

  it('moves focus into the panel on open and gives it back on close', async () => {
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    trigger.focus()
    const wrapper = mountOverlay({ modelValue: false })
    await wrapper.setProps({ modelValue: true })
    await wrapper.vm.$nextTick()
    expect(document.activeElement).toBe(document.body.querySelector('[role="dialog"]'))
    await wrapper.setProps({ modelValue: false })
    expect(document.activeElement).toBe(trigger)
    wrapper.unmount()
  })

  it('wraps Tab focus inside the panel', async () => {
    const wrapper = mountOverlay()
    const inner = document.body.querySelector<HTMLElement>('.inner')!
    inner.focus()
    const tab = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    inner.dispatchEvent(tab)
    expect(document.activeElement).toBe(inner)
    wrapper.unmount()
  })
})

describe('SiteImage', () => {
  it('shows the skeleton until loaded and falls back through the chain before showing the error', async () => {
    const wrapper = mount(SiteImage, {
      props: { src: 'https://first.example/a.png', fallbackSrc: ['https://second.example/a.png'], alt: '图' },
    })
    expect(wrapper.classes()).not.toContain('is-loaded')
    await wrapper.get('img').trigger('error')
    expect(wrapper.get('img').attributes('src')).toBe('https://second.example/a.png')
    expect(wrapper.emitted('fallback')).toEqual([['https://first.example/a.png']])
    expect(wrapper.emitted('error')).toBeUndefined()

    await wrapper.get('img').trigger('error')
    expect(wrapper.emitted('error')).toHaveLength(1)
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.get('[role="img"]').attributes('aria-label')).toBe('图')
  })

  it('marks itself loaded on load and resets when the source changes', async () => {
    const wrapper = mount(SiteImage, { props: { src: 'https://images.example/a.png' } })
    await wrapper.get('img').trigger('load')
    expect(wrapper.classes()).toContain('is-loaded')
    await wrapper.setProps({ src: 'https://images.example/b.png' })
    expect(wrapper.classes()).not.toContain('is-loaded')
  })
})
