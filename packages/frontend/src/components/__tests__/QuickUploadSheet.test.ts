import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { InputTag } from '@arco-design/web-vue'
import { flushPromises, mount } from '@vue/test-utils'
import QuickUploadSheet from '../QuickUploadSheet.vue'

const mocks = vi.hoisted(() => ({
  upload: vi.fn(), options: vi.fn(), push: vi.fn(), warning: vi.fn(), success: vi.fn(), error: vi.fn(),
  user: { isLoggedIn: false, user: { username: '已登录作者', email: 'author@example.com' } },
}))
vi.mock('@/api', () => ({ contentApi: { quickUpload: mocks.upload, metadataOptions: mocks.options } }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
vi.mock('@/stores/user', () => ({ useUserStore: () => mocks.user }))
vi.mock('@/composables/useToast', () => ({ toast: mocks }))

const Drawer = defineComponent({
  props: { visible: Boolean, maskClosable: Boolean, escToClose: Boolean },
  template: '<div v-if="visible"><slot /></div>',
})
const mounted: ReturnType<typeof mount>[] = []
function createSheet() {
  const wrapper = mount(QuickUploadSheet, {
    props: { open: true }, global: { stubs: { Drawer } },
  })
  mounted.push(wrapper)
  return wrapper
}
async function fillGuest(wrapper: ReturnType<typeof createSheet>) {
  await wrapper.find('input[placeholder="怎么称呼你？"]').setValue('  创作者  ')
  await wrapper.find('input[placeholder="用于后续认领作品"]').setValue('  guest@example.com  ')
  await wrapper.find('input[placeholder="给这份灵感起个名字"]').setValue('  新的故事  ')
}

afterEach(() => { mounted.splice(0).forEach(wrapper => wrapper.unmount()) })

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  mocks.user.isLoggedIn = false
  mocks.upload.mockResolvedValue({ code: 200, data: { audit_status: 'pending' } })
  mocks.options.mockResolvedValue({ code: 200, data: { available: true, ready: false, placeholder_title: '待命名作品', placeholder_tag: '待生成标签' } })
})

describe('QuickUploadSheet', () => {
  it('keeps the AI checkbox independent of auto generation and manual tags', async () => {
    const wrapper = createSheet()
    await fillGuest(wrapper)
    await wrapper.find('textarea').setValue('AI 生成的文字作品')
    wrapper.findComponent(InputTag).vm.$emit('update:modelValue', ['校园'])
    const ai = wrapper.findAll('input[type="checkbox"]')[1]!
    await ai.setValue(true)
    // The checkbox-owned marker must never be inserted into the editable manual draft.
    expect(wrapper.findComponent(InputTag).props('modelValue')).toEqual(['校园'])
    await ai.setValue(false)
    await wrapper.find('form').trigger('submit'); await flushPromises()
    expect(mocks.upload).toHaveBeenLastCalledWith(expect.objectContaining({ tags: ['校园'], is_ai_generated: false }), expect.any(Function))
  })

  it('deduplicates manually supplied AIGC and retains it with automatic metadata', async () => {
    const wrapper = createSheet()
    await fillGuest(wrapper)
    wrapper.findComponent(InputTag).vm.$emit('update:modelValue', ['校园', 'aigc', 'AIGC'])
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { configurable: true, value: [new File(['image'], 'work.png', { type: 'image/png' })] })
    await input.trigger('change'); await flushPromises()
    await wrapper.findAll('input[type="checkbox"]')[1]!.setValue(true)
    await wrapper.findAll('input[type="checkbox"]')[0]!.setValue(true)
    expect(wrapper.findComponent(InputTag).props('modelValue')).toEqual(['待生成标签', 'AIGC'])
    await wrapper.findAll('input[type="checkbox"]')[1]!.setValue(false)
    expect(wrapper.findComponent(InputTag).props('modelValue')).toEqual(['待生成标签', 'AIGC'])
    await wrapper.find('form').trigger('submit'); await flushPromises()
    expect(mocks.upload).toHaveBeenLastCalledWith(expect.objectContaining({ tags: ['待生成标签', 'AIGC'], auto_metadata: true, is_ai_generated: false }), expect.any(Function))
  })

  it('adds the checked AI marker once to manually entered tags', async () => {
    const wrapper = createSheet(); await fillGuest(wrapper)
    await wrapper.find('textarea').setValue('作品描述')
    wrapper.findComponent(InputTag).vm.$emit('update:modelValue', ['校园'])
    await wrapper.findAll('input[type="checkbox"]')[1]!.setValue(true)
    await wrapper.find('form').trigger('submit'); await flushPromises()
    expect(mocks.upload).toHaveBeenCalledWith(expect.objectContaining({ tags: ['校园', 'AIGC'], is_ai_generated: true }), expect.any(Function))
  })
  it('refreshes provider readiness when the kept-alive drawer is reopened', async () => {
    const wrapper = createSheet()
    await flushPromises()
    expect(mocks.options).toHaveBeenCalledTimes(1)
    await wrapper.setProps({ open: false })
    mocks.options.mockResolvedValue({ code: 200, data: { available: true, ready: true, placeholder_title: '待命名作品', placeholder_tag: '待生成标签' } })
    await wrapper.setProps({ open: true })
    await flushPromises()
    expect(mocks.options).toHaveBeenCalledTimes(2)
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { configurable: true, value: [new File(['image'], 'work.png', { type: 'image/png' })] })
    await input.trigger('change')
    await wrapper.find('input[type="checkbox"]').setValue(true)
    expect(wrapper.text()).not.toContain('服务尚未启用')
  })
  it('uses placeholders and restores the manual draft when switched off', async () => {
    const wrapper = createSheet()
    await fillGuest(wrapper)
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { configurable: true, value: [new File(['image'], 'work.png', { type: 'image/png' })] })
    await input.trigger('change')
    await flushPromises()
    const checkbox = wrapper.find('input[type="checkbox"]')
    await checkbox.setValue(true)
    const title = wrapper.find('input[placeholder="给这份灵感起个名字"]')
    expect((title.element as HTMLInputElement).value).toBe('待命名作品')
    expect(title.attributes('disabled')).toBeDefined()
    await checkbox.setValue(false)
    expect((title.element as HTMLInputElement).value).toBe('  新的故事  ')
    await checkbox.setValue(true)
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(mocks.upload).toHaveBeenCalledWith(expect.objectContaining({ title: '待命名作品', tags: ['待生成标签'], auto_metadata: true }), expect.any(Function))
  })

  it('resets generation when an image is removed and excludes text/video', async () => {
    const wrapper = createSheet()
    await flushPromises()
    expect(wrapper.find('input[type="checkbox"]').attributes('disabled')).toBeDefined()
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { configurable: true, value: [new File(['image'], 'work.png', { type: 'image/png' })] })
    await input.trigger('change')
    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.find('[aria-label="移除已选文件"]').trigger('click')
    expect((wrapper.find('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    Object.defineProperty(input.element, 'files', { configurable: true, value: [new File(['video'], 'work.mp4', { type: 'video/mp4' })] })
    await input.trigger('change')
    expect(wrapper.find('input[type="checkbox"]').attributes('disabled')).toBeDefined()
  })
  it('restores guest identity on the first open and keeps a file input mounted', () => {
    localStorage.setItem('xqecz_guest_identity', JSON.stringify({ nickname: '之前的作者', email: 'saved@example.com' }))
    const wrapper = createSheet()
    expect((wrapper.find('input[placeholder="怎么称呼你？"]').element as HTMLInputElement).value).toBe('之前的作者')
    expect(wrapper.find('input[type="file"]').exists()).toBe(true)
  })

  it('validates guest identity before requesting an upload', async () => {
    const wrapper = createSheet()
    await wrapper.find('form').trigger('submit')
    expect(mocks.warning).toHaveBeenCalledWith('请填写昵称')
    expect(mocks.upload).not.toHaveBeenCalled()
  })

  it('requires text or media but allows text-only works', async () => {
    const wrapper = createSheet()
    await fillGuest(wrapper)
    await wrapper.find('form').trigger('submit')
    expect(mocks.warning).toHaveBeenCalledWith('请填写描述或上传文件')
    expect(mocks.upload).not.toHaveBeenCalled()
    await wrapper.find('textarea').setValue('  一段灵感  ')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(mocks.upload).toHaveBeenCalledWith({ title: '新的故事', nickname: '创作者', email: 'guest@example.com', content: '一段灵感', file: undefined, tags: [], auto_metadata: false, is_ai_generated: false }, expect.any(Function))
    expect(wrapper.emitted('uploaded')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(mocks.push).toHaveBeenCalledWith('/')
  })

  it('keeps the same file input available after selecting and removing media', async () => {
    const wrapper = createSheet()
    const input = wrapper.find('input[type="file"]')
    const image = new File(['image'], 'summer.png', { type: 'image/png' })
    Object.defineProperty(input.element, 'files', { configurable: true, value: [image] })
    await input.trigger('change')
    expect(wrapper.find('.qus-file-name').text()).toBe('summer.png')
    expect(wrapper.find('input[type="file"]').element).toBe(input.element)
    await wrapper.find('[aria-label="移除已选文件"]').trigger('click')
    expect(wrapper.find('.qus-dropzone').exists()).toBe(true)
    expect(wrapper.find('input[type="file"]').element).toBe(input.element)
    await input.trigger('change')
    expect(wrapper.find('.qus-file-name').text()).toBe('summer.png')
  })

  it('prevents duplicate submissions and disables editing and closing during upload', async () => {
    let finish!: (value: unknown) => void
    mocks.upload.mockImplementation(() => new Promise(resolve => { finish = resolve }))
    const wrapper = createSheet()
    await fillGuest(wrapper)
    await wrapper.find('textarea').setValue('正在上传的作品')
    await wrapper.find('form').trigger('submit')
    await wrapper.find('form').trigger('submit')
    expect(mocks.upload).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[aria-label="关闭上传窗口"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('input[placeholder="给这份灵感起个名字"]').attributes('disabled')).toBeDefined()
    expect(wrapper.findComponent(Drawer).props('maskClosable')).toBe(false)
    expect(wrapper.findComponent(Drawer).props('escToClose')).toBe(false)
    finish({ code: 200, data: { audit_status: 'approved' } })
    await flushPromises()
    expect(wrapper.emitted('uploaded')).toHaveLength(1)
  })

  it('preserves the form and allows retry after a failed request', async () => {
    mocks.upload.mockRejectedValue(new Error('连接失败'))
    const wrapper = createSheet()
    await fillGuest(wrapper)
    await wrapper.find('textarea').setValue('需要保留的草稿')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(mocks.error).toHaveBeenCalledWith('连接失败')
    expect(wrapper.emitted('close')).toBeUndefined()
    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toBe('需要保留的草稿')
    expect(wrapper.find('[aria-label="关闭上传窗口"]').attributes('disabled')).toBeUndefined()
  })

  it('uses the account identity for logged-in authors', async () => {
    mocks.user.isLoggedIn = true
    const wrapper = createSheet()
    expect(wrapper.find('input[placeholder="怎么称呼你？"]').exists()).toBe(false)
    await wrapper.find('input[placeholder="给这份灵感起个名字"]').setValue('登录作品')
    await wrapper.find('textarea').setValue('创作内容')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(mocks.upload).toHaveBeenCalledWith(expect.objectContaining({ nickname: '已登录作者', email: 'author@example.com' }), expect.any(Function))
  })
})
