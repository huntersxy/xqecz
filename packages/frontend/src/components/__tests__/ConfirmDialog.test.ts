import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ConfirmDialog from '../ConfirmDialog.vue'

const mockPendingConfirm = ref<{ title?: string; message: string } | null>(null)
// 与真实 respond 一致：回应后待确认项被清空，浮层随之收起。
const mockRespond = vi.fn((_: boolean) => { mockPendingConfirm.value = null })

vi.mock('@/composables/useToast', () => ({
  useConfirm: vi.fn(() => ({
    pendingConfirm: mockPendingConfirm,
    respond: mockRespond,
  })),
}))

async function mountDialog() {
  const wrapper = mount(ConfirmDialog)
  await nextTick()
  return wrapper
}

function queryDialog() {
  return document.body.querySelector<HTMLElement>('.site-confirm [role="dialog"]')
}

function button(label: string) {
  return [...document.body.querySelectorAll<HTMLButtonElement>('.site-confirm button')]
    .find((b) => b.textContent?.includes(label))
}

describe('ConfirmDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockPendingConfirm.value = null
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders nothing when there is no pending confirm', async () => {
    const wrapper = await mountDialog()
    expect(queryDialog()).toBeNull()
    wrapper.unmount()
  })

  it('renders title and message when a confirm is pending', async () => {
    mockPendingConfirm.value = { message: '确定要删除吗？' }
    const wrapper = await mountDialog()
    const dialog = queryDialog()
    expect(dialog).not.toBeNull()
    expect(dialog!.getAttribute('aria-modal')).toBe('true')
    expect(dialog!.textContent).toContain('确认操作')
    expect(dialog!.textContent).toContain('确定要删除吗？')
    wrapper.unmount()
  })

  it('keeps the creative theme on the teleported dialog outside the page', async () => {
    mockPendingConfirm.value = { message: '确认前请检查内容' }
    const wrapper = await mountDialog()
    const dialog = queryDialog()!
    expect(dialog.closest('.creative-theme')).not.toBeNull()
    expect(wrapper.element.contains(dialog)).toBe(false)
    wrapper.unmount()
  })

  it('calls respond with false when cancel is clicked', async () => {
    mockPendingConfirm.value = { message: '测试消息' }
    const wrapper = await mountDialog()
    button('取消')!.click()
    expect(mockRespond).toHaveBeenCalledWith(false)
    wrapper.unmount()
  })

  it('calls respond with true when confirm is clicked', async () => {
    mockPendingConfirm.value = { message: '测试消息' }
    const wrapper = await mountDialog()
    button('确认')!.click()
    await flushPromises()
    expect(mockRespond).toHaveBeenCalledWith(true)
    wrapper.unmount()
  })

  it('calls respond with false when the close button is clicked', async () => {
    mockPendingConfirm.value = { message: '测试消息' }
    const wrapper = await mountDialog()
    document.body.querySelector<HTMLButtonElement>('.site-confirm [aria-label="关闭"]')!.click()
    expect(mockRespond).toHaveBeenCalledWith(false)
    wrapper.unmount()
  })

  it('treats Escape and a mask click as cancel', async () => {
    mockPendingConfirm.value = { message: '测试消息' }
    const wrapper = await mountDialog()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(mockRespond).toHaveBeenLastCalledWith(false)

    mockPendingConfirm.value = { message: '再来一次' }
    await nextTick()
    document.body.querySelector<HTMLElement>('.site-overlay-mask')!.click()
    expect(mockRespond).toHaveBeenCalledTimes(2)
    expect(mockRespond).toHaveBeenLastCalledWith(false)
    wrapper.unmount()
  })
})
