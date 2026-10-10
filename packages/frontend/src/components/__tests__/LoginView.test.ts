import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import LoginView from '@/views/LoginView.vue'

const mocks = vi.hoisted(() => ({ login: vi.fn(), register: vi.fn(), push: vi.fn(), error: vi.fn(), success: vi.fn(), user: { needsEmail: false } }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ ...mocks.user, login: mocks.login }) }))
vi.mock('@/api', () => ({ authApi: { register: mocks.register } }))
vi.mock('@arco-design/web-vue', async importOriginal => ({ ...await importOriginal<typeof import('@arco-design/web-vue')>(), Message: mocks }))
const wrappers: ReturnType<typeof mount>[] = []
function create() { const wrapper = mount(LoginView, { global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } }); wrappers.push(wrapper); return wrapper }
beforeEach(() => { vi.clearAllMocks(); mocks.login.mockResolvedValue(true); mocks.register.mockResolvedValue({ code: 200 }) })
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

describe('login and register states', () => {
  it('enables the email field only for register and updates password autofill', async () => {
    const wrapper = create()
    expect(wrapper.find('#auth-email').attributes('disabled')).toBeDefined()
    expect(wrapper.find('#auth-email').attributes('required')).toBeUndefined()
    expect(wrapper.find('#auth-password').attributes('autocomplete')).toBe('current-password')
    await wrapper.findAll('.auth-mode button')[1].trigger('click')
    expect(wrapper.find('.auth-email-slot').classes()).toContain('is-open')
    expect(wrapper.find('#auth-email').attributes('disabled')).toBeUndefined()
    expect(wrapper.find('#auth-email').attributes('required')).toBeDefined()
    expect(wrapper.find('#auth-password').attributes('autocomplete')).toBe('new-password')
    await wrapper.findAll('.auth-mode button')[0].trigger('click')
    expect(wrapper.find('.auth-email-slot').classes()).not.toContain('is-open')
    expect(wrapper.find('#auth-email').attributes('disabled')).toBeDefined()
  })
  it('blocks duplicate login and mode changes during a pending request', async () => {
    let finish!: (result: boolean) => void
    mocks.login.mockImplementation(() => new Promise(resolve => { finish = resolve }))
    const wrapper = create()
    await wrapper.find('#auth-username').setValue('testuser')
    await wrapper.find('#auth-password').setValue('password')
    await wrapper.find('form').trigger('submit')
    await wrapper.find('form').trigger('submit')
    expect(mocks.login).toHaveBeenCalledTimes(1)
    expect(wrapper.find('#auth-username').attributes('disabled')).toBeDefined()
    expect(wrapper.findAll('.auth-mode button')[1].attributes('disabled')).toBeDefined()
    finish(false)
    await flushPromises()
    expect(mocks.error).toHaveBeenCalledWith('用户名或密码错误')
    expect(wrapper.find('#auth-username').attributes('disabled')).toBeUndefined()
  })
  it('returns to login after successful registration and clears the old fields', async () => {
    const wrapper = create()
    await wrapper.findAll('.auth-mode button')[1].trigger('click')
    await wrapper.find('#auth-username').setValue('creator')
    await wrapper.find('#auth-email').setValue('creator@example.com')
    await wrapper.find('#auth-password').setValue('password')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(mocks.register).toHaveBeenCalledWith('creator', 'creator@example.com', 'password')
    expect(wrapper.find('.auth-email-slot').classes()).not.toContain('is-open')
    expect((wrapper.find('#auth-username').element as HTMLInputElement).value).toBe('')
    expect(wrapper.find('#auth-password').attributes('autocomplete')).toBe('current-password')
  })
})
