import { defineComponent } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { useAIStudio } from '../useAIStudio'
const mock = vi.hoisted(() => ({
  isAdmin: false,
  config: vi.fn(),
  sessions: vi.fn(),
  assets: vi.fn(),
  account: vi.fn(),
  tasks: vi.fn(),
  saveSession: vi.fn(),
  quote: vi.fn(),
  submit: vi.fn(),
  createSession: vi.fn(),
  upload: vi.fn(),
  archive: vi.fn(),
}))
vi.mock('@/api/creative', () => ({ aiApi: mock }))
vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ user: { id: 1, is_admin: mock.isAdmin }, isLoggedIn: true }),
}))
const time = '2026-10-13T00:00:00.000Z'
function session(version = 1, draft = '{}') {
  return { id: 1, name: 'test', draft, version, created_at: time, updated_at: time }
}
async function setup() {
  let studio!: ReturnType<typeof useAIStudio>
  const wrapper = mount(
    defineComponent({
      setup() {
        studio = useAIStudio()
        return () => null
      },
    }),
  )
  await flushPromises()
  return { wrapper, studio }
}
beforeEach(() => {
  mock.isAdmin = false
  vi.useFakeTimers()
  vi.clearAllMocks()
  localStorage.clear()
  mock.config.mockResolvedValue({
    prompt_enhancement: { enabled: true, prompt: '补充画面细节' },
    providers: [
      {
        id: 1,
        revision: 1,
        owner: 0,
        enabled: true,
        has_key: true,
        name: 'test',
        sizes: ['1024x1024'],
        qualities: [],
        max_count: 1,
        max_references: 2,
        cost: 2,
        quality_costs: {},
      },
    ],
  })
  mock.sessions.mockResolvedValue([session()])
  mock.assets.mockResolvedValue([])
  mock.account.mockResolvedValue({ user_id: 1, available: 10, reserved: 0 })
  mock.tasks.mockResolvedValue([])
  mock.quote.mockResolvedValue({ cost: 2 })
  mock.saveSession.mockImplementation(
    async (_id: number, _name: string, draft: unknown, version: number) =>
      session(version + 1, JSON.stringify(draft)),
  )
  mock.submit.mockResolvedValue({ id: 1 })
  mock.createSession.mockResolvedValue({ ...session(), id: 2 })
})
afterEach(() => {
  vi.useRealTimers()
})
describe('AI studio persistence', () => {
  it('skips saves of unchanged drafts, including after an autosave', async () => {
    const { wrapper, studio } = await setup()
    await studio.save()
    expect(mock.saveSession).not.toHaveBeenCalled()
    studio.prompt.value = '新的画面'
    await flushPromises()
    await studio.save()
    await studio.save()
    expect(mock.saveSession).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(900)
    expect(mock.saveSession).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
  it('inserts a reference at a distant slot while preserving intermediate order and edit ancestry', async () => {
    const { wrapper, studio } = await setup()
    studio.references.value = [7, 8, 9].map((asset_id) => ({ asset_id, note: '' }))
    studio.parentAssetId.value = 7
    studio.moveReference(0, 2)
    expect(studio.references.value.map((r) => r.asset_id)).toEqual([8, 9, 7])
    expect(studio.parentAssetId.value).toBe(7)
    studio.moveReference(2, -2)
    expect(studio.references.value.map((r) => r.asset_id)).toEqual([7, 8, 9])
    wrapper.unmount()
  })
  it('aborts an upload on unmount and discards its late result', async () => {
    let finish!: (value: { id: number }) => void
    mock.upload.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    const { wrapper, studio } = await setup()
    const upload = studio.upload(new File(['png'], 'ref.png', { type: 'image/png' }))
    const signal = mock.upload.mock.calls[0]?.[1] as AbortSignal
    wrapper.unmount()
    expect(signal.aborted).toBe(true)
    finish({ id: 9 })
    await upload
    expect(studio.references.value).toEqual([])
    expect(studio.assets.value).toEqual([])
  })
  it('keeps a late upload in the single workspace library', async () => {
    let finish!: (value: { id: number }) => void
    mock.upload.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    const { wrapper, studio } = await setup()
    const upload = studio.upload(new File(['png'], 'ref.png', { type: 'image/png' }))
    finish({ id: 9 })
    await upload
    expect(studio.references.value).toEqual([{ asset_id: 9, note: '' }])
    expect(studio.assets.value.map((a) => a.id)).toEqual([9])
    wrapper.unmount()
  })
  it('lets administrators submit with zero available quota and a zero expected charge', async () => {
    mock.isAdmin = true
    mock.account.mockResolvedValue({ user_id: 1, available: 0, reserved: 0 })
    mock.quote.mockResolvedValue({ cost: 0 })
    const { wrapper, studio } = await setup()
    studio.prompt.value = 'A pink circle'
    expect(studio.cost.value).toBe(0)
    await studio.generate()
    expect(mock.submit).toHaveBeenCalledWith(
      expect.objectContaining({ expected_cost: 0 }),
      expect.any(AbortSignal),
    )
    wrapper.unmount()
  })
  it('restores an empty session with the default provider and automatic sizing', async () => {
    const { wrapper, studio } = await setup()
    expect(studio.providerId.value).toBe(1)
    expect(studio.size.value).toBe('auto')
    expect(studio.ready.value).toBe(true)
    wrapper.unmount()
  })
  it('defaults prompt enhancement on and keeps the toggle in the generation contract', async () => {
    const { wrapper, studio } = await setup()
    expect(studio.enhancePrompt.value).toBe(true)
    expect(studio.promptEnhancement.value.prompt).toBe('补充画面细节')
    expect(studio.input().enhance_prompt).toBe(true)
    studio.enhancePrompt.value = false
    expect(studio.input().enhance_prompt).toBe(false)
    wrapper.unmount()
  })
  it('remembers a selected size for the next new session', async () => {
    const first = await setup()
    first.studio.size.value = '1536x1024'
    await flushPromises()
    first.wrapper.unmount()

    const second = await setup()
    expect(second.studio.size.value).toBe('1536x1024')
    second.wrapper.unmount()
  })
  it('archives a library asset and removes its workspace references', async () => {
    const { wrapper, studio } = await setup()
    studio.assets.value = [{ id: 9 } as never]
    studio.references.value = [{ asset_id: 9, note: '' }]
    studio.parentAssetId.value = 9
    mock.archive.mockResolvedValue(undefined)
    await studio.archiveAsset(9)
    expect(mock.archive).toHaveBeenCalledWith(9)
    expect(studio.assets.value).toEqual([])
    expect(studio.references.value).toEqual([])
    expect(studio.parentAssetId.value).toBe(0)
    wrapper.unmount()
  })
  it('serializes autosave before persisting an idempotent submission', async () => {
    let finish!: () => void
    mock.saveSession.mockImplementationOnce(
      (_id: number, _name: string, draft: unknown, version: number) =>
        new Promise((resolve) => {
          finish = () => resolve(session(version + 1, JSON.stringify(draft)))
        }),
    )
    const { wrapper, studio } = await setup()
    studio.prompt.value = 'A pink circle'
    await vi.advanceTimersByTimeAsync(900)
    const generated = studio.generate()
    await flushPromises()
    expect(mock.submit).not.toHaveBeenCalled()
    finish()
    await generated
    expect(mock.saveSession.mock.calls[1]?.[3]).toBe(2)
    const saved = mock.saveSession.mock.calls[1]?.[2] as { pending: { idempotency_key: string } }
    expect(saved.pending.idempotency_key).toBe(mock.submit.mock.calls[0]?.[0].idempotency_key)
    expect(mock.saveSession.mock.calls[2]?.[3]).toBe(3)
    wrapper.unmount()
  })
  it('retains the same key after a lost response and stops polling after unmount', async () => {
    const { wrapper, studio } = await setup()
    studio.prompt.value = 'A pink circle'
    mock.submit.mockRejectedValueOnce(
      Object.assign(new Error('lost response'), { name: 'ApiError' }),
    )
    await expect(studio.generate()).rejects.toThrow('lost response')
    const key = studio.pending.value?.idempotency_key
    await studio.generate()
    expect(mock.submit.mock.calls[1]?.[0].idempotency_key).toBe(key)
    expect(mock.quote).toHaveBeenCalledTimes(1)
    wrapper.unmount()
    const calls = mock.tasks.mock.calls.length
    await vi.advanceTimersByTimeAsync(20000)
    expect(mock.tasks).toHaveBeenCalledTimes(calls)
  })
  it('refuses a changed quote before reserving any credit', async () => {
    const { wrapper, studio } = await setup()
    studio.prompt.value = 'A pink circle'
    mock.quote.mockResolvedValueOnce({ cost: 3 })
    await expect(studio.generate()).rejects.toThrow('价格已变化')
    expect(mock.submit).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
