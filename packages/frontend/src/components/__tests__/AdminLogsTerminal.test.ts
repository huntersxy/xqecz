import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AdminLogsTerminal from '../admin/AdminLogsTerminal.vue'

const api = vi.hoisted(() => ({ logs: vi.fn() }))
const toast = vi.hoisted(() => ({ success: vi.fn(), warning: vi.fn() }))

vi.mock('@/api', () => ({ adminApi: { logs: api.logs } }))
vi.mock('@/composables/useToast', () => ({ toast }))

const wrappers: ReturnType<typeof mount>[] = []

function render() {
  const wrapper = mount(AdminLogsTerminal)
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  api.logs.mockResolvedValue({
    code: 200,
    data: {
      entries: [
        {
          id: 1,
          time: '2026-10-18T01:02:03.000Z',
          level: 'ERROR',
          message: 'http request failed',
          line: '[2026-10-18 09:02:03 ERROR] http request failed  status=500',
        },
      ],
      next_cursor: 4,
      server_time: '2026-10-18T01:02:04.000Z',
    },
  })
})

afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.useRealTimers()
})

describe('AdminLogsTerminal', () => {
  it('renders the server-redacted line and normalizes uppercase levels', async () => {
    const wrapper = render()
    await flushPromises()

    expect(wrapper.get('.logs-line-message').text()).toContain('status=500')
    expect(wrapper.get('.logs-line-marker').classes()).toContain('is-error')
    expect(wrapper.text()).toContain('1 条')
  })

  it('uses the cursor for incremental polling and aborts an owned request on unmount', async () => {
    vi.useFakeTimers()
    const initial = {
      code: 200,
      data: { entries: [], next_cursor: 8, server_time: '2026-10-18T01:02:04.000Z' },
    }
    api.logs.mockResolvedValueOnce(initial).mockResolvedValueOnce({
      code: 200,
      data: {
        entries: [{ id: 9, time: '2026-10-18T01:02:05.000Z', level: 'INFO', message: 'ready' }],
        next_cursor: 9,
      },
    })
    render()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(api.logs).toHaveBeenNthCalledWith(2, { limit: 200, level: undefined, after: 8 }, expect.any(AbortSignal))

    let requestSignal!: AbortSignal
    api.logs.mockImplementationOnce((_params: unknown, signal: AbortSignal) => {
      requestSignal = signal
      return new Promise(() => {})
    })
    const pending = render()
    await flushPromises()
    pending.unmount()
    expect(requestSignal.aborted).toBe(true)
  })
})
