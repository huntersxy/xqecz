import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AdminReferenceLibrary from '../admin/AdminReferenceLibrary.vue'

const mocks = vi.hoisted(() => ({ list: vi.fn(), preview: vi.fn(), upload: vi.fn(), rename: vi.fn(), remove: vi.fn(), save: vi.fn(), describe: vi.fn(), confirm: vi.fn() }))
vi.mock('@/api', () => ({ adminApi: { visionReferences: mocks.list, visionReferenceImage: mocks.preview, uploadVisionReference: mocks.upload, renameVisionReference: mocks.rename, deleteVisionReference: mocks.remove, saveVisionReferenceDescription: mocks.save, describeVisionReference: mocks.describe } }))
vi.mock('@/composables/useToast', () => ({ toast: { success: vi.fn() }, useConfirm: () => ({ confirm: mocks.confirm }) }))
const role = { name: '小泉', bytes: 51819, version: 'image-v1', description: '黑色短发', needs_description: false }
const wrappers: ReturnType<typeof mount>[] = []
function create(refs = [role], hasKey = true) { const w = mount(AdminReferenceLibrary, { props: { references: refs, hasKey } }); wrappers.push(w); return w }
beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('matchMedia', vi.fn(() => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() })))
  vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:preview'), revokeObjectURL: vi.fn() })
  mocks.list.mockResolvedValue({ code: 200, data: [role] }); mocks.preview.mockResolvedValue(new Blob(['image']))
  mocks.upload.mockResolvedValue({ code: 200, data: role }); mocks.rename.mockResolvedValue({ code: 200, data: role })
  mocks.save.mockResolvedValue({ code: 200, data: role }); mocks.describe.mockResolvedValue({ code: 200, data: role })
  mocks.remove.mockResolvedValue({ code: 200 }); mocks.confirm.mockResolvedValue(true)
})
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.unstubAllGlobals() })
function button(w: ReturnType<typeof mount>, text: string) { return w.findAll('button').find(b => b.text() === text)! }
describe('admin character library', () => {
  it('uses versioned descriptions and refreshes after saving', async () => {
    const w = create(); await flushPromises()
    await w.find('textarea').setValue('蓝色眼睛，星形发饰')
    await button(w, '保存文字描述').trigger('click'); await flushPromises()
    expect(mocks.save).toHaveBeenCalledWith(role, '蓝色眼睛，星形发饰', expect.any(AbortSignal))
    expect(w.emitted('update')?.[0]).toEqual([[role]])
  })
  it('converts only missing or outdated descriptions in bulk', async () => {
    const stale = { ...role, name: '角色乙', needs_description: true }
    const w = create([role, stale]); await flushPromises()
    await button(w, '转述缺失 / 过期描述').trigger('click'); await flushPromises()
    expect(mocks.describe).toHaveBeenCalledTimes(1)
    expect(mocks.describe).toHaveBeenCalledWith(stale, expect.any(AbortSignal))
  })
  it('keeps an edited description when a stale-version save fails', async () => {
    mocks.save.mockResolvedValue({ code: 400, message: '参照图已更新，请刷新后再保存描述' })
    const w = create(); await flushPromises(); await w.find('textarea').setValue('待保存特征')
    await button(w, '保存文字描述').trigger('click'); await flushPromises()
    expect(w.text()).toContain('参照图已更新')
    expect((w.find('textarea').element as HTMLTextAreaElement).value).toBe('待保存特征')
    expect(mocks.list).not.toHaveBeenCalled()
  })
  it('uploads a selected image under an editable filename-derived name', async () => {
    const w = create([]); const file = new File(['png'], '角色甲.png', { type: 'image/png' })
    const input = w.find('input[type=file]')
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change'); await button(w, '上传参照图').trigger('click'); await flushPromises()
    expect(mocks.upload).toHaveBeenCalledWith('角色甲', file, false, expect.any(AbortSignal))
  })
  it('asks before removal, aborts requests and releases private image previews on exit', async () => {
    const w = create(); await flushPromises()
    await button(w, '移出').trigger('click'); await flushPromises()
    expect(mocks.confirm).toHaveBeenCalled(); expect(mocks.remove).toHaveBeenCalledWith('小泉', expect.any(AbortSignal))
    const signal = mocks.preview.mock.calls[0]![1] as AbortSignal
    w.unmount(); expect(signal.aborted).toBe(true); expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:preview')
  })
})
