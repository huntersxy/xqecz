import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AdminVisionSettings from '../admin/AdminVisionSettings.vue'

const mocks = vi.hoisted(() => ({ settings: vi.fn(), jobs: vi.fn(), references: vi.fn(), save: vi.fn(), test: vi.fn(), retry: vi.fn(), success: vi.fn() }))
vi.mock('@/api', () => ({ adminApi: { visionSettings: mocks.settings, visionJobs: mocks.jobs, visionReferences: mocks.references, saveVisionSettings: mocks.save, testVision: mocks.test, retryVision: mocks.retry } }))
vi.mock('@/composables/useToast', () => ({ toast: { success: mocks.success } }))

const config = { enabled: false, base_url: 'https://api.siliconflow.cn/v1', model: 'Qwen/Qwen3.5-4B', prompt: '请按可见主体生成标题和短词标签', max_tags: 8, timeout_seconds: 90, disable_thinking: true, token_parameter: 'max_tokens' as const, use_references: true, has_api_key: true }
const mounted: ReturnType<typeof mount>[] = []
function create() { const w = mount(AdminVisionSettings); mounted.push(w); return w }
beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('matchMedia', vi.fn((media: string) => ({ matches: false, media, onchange: null, addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn() })))
  mocks.settings.mockResolvedValue({ code: 200, data: { ...config } })
  mocks.jobs.mockResolvedValue({ code: 200, data: [] })
  mocks.references.mockResolvedValue({ code: 200, data: [{ name: '小泉', bytes: 51819 }] })
  mocks.save.mockResolvedValue({ code: 200, data: { ...config } })
  mocks.test.mockResolvedValue({ code: 200, data: { title: '窗边的少年', tags: ['少年', '窗边'] } })
})
afterEach(() => { mounted.splice(0).forEach(w => w.unmount()); vi.unstubAllGlobals() })

describe('AdminVisionSettings', () => {
  it('shows reference names and preserves a saved key when the field is blank', async () => {
    const w = create(); await flushPromises()
    expect(w.text()).toContain('参照库：小泉')
    await w.findAll('button').find(b => b.text() === '保存配置')!.trigger('click'); await flushPromises()
    expect(mocks.save).toHaveBeenCalledWith(expect.not.objectContaining({ api_key: expect.anything() }), expect.any(AbortSignal))
    expect(mocks.save).toHaveBeenCalledWith(expect.objectContaining({ clear_api_key: false }), expect.any(AbortSignal))
  })
  it('reports soft validation failures and does not call the test endpoint after failed saving', async () => {
    mocks.save.mockResolvedValue({ code: 400, message: '启用前请配置 API Key' })
    const w = create(); await flushPromises()
    await w.findAll('button').find(b => b.text() === '保存并测试视觉接口')!.trigger('click'); await flushPromises()
    expect(w.text()).toContain('启用前请配置 API Key')
    expect(mocks.test).not.toHaveBeenCalled()
  })
  it('cancels pending requests when leaving the settings page', async () => {
    mocks.settings.mockReturnValue(new Promise(() => {}))
    const w = create(); await flushPromises()
    const signal = mocks.settings.mock.calls[0]![0] as AbortSignal
    expect(signal.aborted).toBe(false)
    w.unmount()
    expect(signal.aborted).toBe(true)
  })
})
