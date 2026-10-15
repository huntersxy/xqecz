import { beforeEach, describe, expect, it, vi } from 'vitest'
const mocks = vi.hoisted(() => ({ fetch: vi.fn(), error: vi.fn() }))
vi.mock('ofetch', () => ({ ofetch: { create: () => mocks.fetch } }))
vi.mock('@/composables/useToast', () => ({ toast: { error: mocks.error } }))
import { adminApi, pollApi } from '../index'
beforeEach(() => vi.clearAllMocks())
describe('owned admin request cancellation', () => {
  it('rejects cancelled requests without showing an error toast', async () => {
    const controller = new AbortController(); controller.abort()
    const error = new DOMException('left panel', 'AbortError')
    mocks.fetch.mockRejectedValue(error)
    await expect(adminApi.visionReferences(controller.signal)).rejects.toBe(error)
    expect(mocks.error).not.toHaveBeenCalled()
  })
  it('keeps ordinary transport failures visible', async () => {
    mocks.fetch.mockRejectedValue(new Error('服务不可用'))
    await expect(adminApi.visionReferences(new AbortController().signal)).rejects.toThrow('服务不可用')
    expect(mocks.error).toHaveBeenCalledWith('服务不可用')
  })
})

describe('public poll requests', () => {
  it('preserves option zero and disables automatic retries for voting', async () => {
    mocks.fetch.mockResolvedValue({ code: 200 })
    const signal = new AbortController().signal
    await pollApi.vote(7, 0, signal)
    expect(mocks.fetch).toHaveBeenCalledWith('/poll/7/vote', {
      method: 'POST', body: { option_index: 0 }, signal, retry: 0,
    })
  })
  it('passes pagination and owned cancellation to the existing list endpoint', async () => {
    mocks.fetch.mockResolvedValue({ code: 200 })
    const controller = new AbortController()
    await pollApi.list({ page: 2, page_size: 12 }, controller.signal)
    expect(mocks.fetch).toHaveBeenCalledWith('/poll/list', { query: { page: 2, page_size: 12 }, signal: controller.signal })
    controller.abort()
    mocks.fetch.mockRejectedValue(new DOMException('left poll', 'AbortError'))
    await expect(pollApi.detail(7, controller.signal)).rejects.toThrow('left poll')
    expect(mocks.error).not.toHaveBeenCalled()
  })
})
