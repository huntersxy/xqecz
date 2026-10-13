import { beforeEach, describe, expect, it, vi } from 'vitest'
const mocks = vi.hoisted(() => ({ fetch: vi.fn(), error: vi.fn() }))
vi.mock('ofetch', () => ({ ofetch: { create: () => mocks.fetch } }))
vi.mock('@/composables/useToast', () => ({ toast: { error: mocks.error } }))
import { adminApi } from '../index'
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
