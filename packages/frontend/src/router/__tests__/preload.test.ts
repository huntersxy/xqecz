import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import router, { preloadRoute } from '../index'

const removeRoutes: (() => void)[] = []
afterEach(() => removeRoutes.splice(0).forEach(remove => remove()))

describe('navigation intent preload', () => {
  it('loads page code once without navigating or mounting it', async () => {
    const mounted = vi.fn()
    const loader = vi.fn().mockResolvedValue({ setup: mounted, template: '<div />' })
    removeRoutes.push(router.addRoute({ path: '/intent-page/:id', component: loader }))
    const before = router.currentRoute.value.fullPath
    preloadRoute('/intent-page/1')
    preloadRoute('/intent-page/2')
    await flushPromises()
    expect(loader).toHaveBeenCalledTimes(1)
    expect(mounted).not.toHaveBeenCalled()
    expect(router.currentRoute.value.fullPath).toBe(before)
  })

  it('allows navigation to retry after a failed preload', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const loader = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ template: '<div />' })
    removeRoutes.push(router.addRoute({ path: '/retry-intent', component: loader }))
    preloadRoute('/retry-intent')
    await flushPromises()
    preloadRoute('/retry-intent')
    await flushPromises()
    expect(loader).toHaveBeenCalledTimes(2)
    warn.mockRestore()
  })
})
