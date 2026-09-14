import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ConfirmDialog from '../ConfirmDialog.vue'

// Mock useConfirm composable
const mockRespond = vi.fn()
const mockPendingConfirm = ref<{ title?: string; message: string } | null>(null)

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

function queryModal() {
  return document.body.querySelector('.arco-modal')
}

describe('ConfirmDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockPendingConfirm.value = null
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('hides dialog when no pending confirm', async () => {
    const wrapper = await mountDialog()
    const modal = queryModal()
    expect(modal).not.toBeNull()
    expect(modal!.style.display).toBe('none')
    wrapper.unmount()
  })

  it('renders dialog when pending confirm exists', async () => {
    mockPendingConfirm.value = {
      message: '确定要删除吗？',
    }

    const wrapper = await mountDialog()

    const dialog = queryModal()
    expect(dialog).not.toBeNull()
    expect(dialog!.style.display).not.toBe('none')
    expect(dialog!.textContent).toContain('确定要删除吗？')
    wrapper.unmount()
  })

  it('renders title and message correctly', async () => {
    mockPendingConfirm.value = {
      message: '测试消息',
    }

    const wrapper = await mountDialog()

    const title = document.body.querySelector('.arco-modal-title')
    expect(title?.textContent).toContain('确认操作')

    const body = document.body.querySelector('.arco-modal-body')
    expect(body?.textContent).toContain('测试消息')
    wrapper.unmount()
  })

  it('calls respond with false when cancel button clicked', async () => {
    mockPendingConfirm.value = {
      message: '测试消息',
    }

    const wrapper = await mountDialog()

    const cancelButton = [...document.body.querySelectorAll('.arco-modal-footer .arco-btn')]
      .find((b) => b.textContent?.includes('取消'))
    expect(cancelButton).toBeDefined()
    ;(cancelButton as HTMLButtonElement).click()
    expect(mockRespond).toHaveBeenCalledWith(false)
    wrapper.unmount()
  })

  it('calls respond with true when confirm button clicked', async () => {
    mockPendingConfirm.value = {
      message: '测试消息',
    }

    const wrapper = await mountDialog()

    const confirmButton = [...document.body.querySelectorAll('.arco-modal-footer .arco-btn')]
      .find((b) => b.textContent?.includes('确认'))
    expect(confirmButton).toBeDefined()
    ;(confirmButton as HTMLButtonElement).click()
    await flushPromises()
    expect(mockRespond).toHaveBeenCalledWith(true)
    wrapper.unmount()
  })

  it('calls respond with false when close button clicked', async () => {
    mockPendingConfirm.value = {
      message: '测试消息',
    }

    const wrapper = await mountDialog()

    const closeButton = document.body.querySelector('.arco-modal-close-btn') as HTMLButtonElement
    expect(closeButton).not.toBeNull()
    closeButton.click()
    expect(mockRespond).toHaveBeenCalledWith(false)
    wrapper.unmount()
  })
})
