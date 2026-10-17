import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AIStudioView from '@/views/AIStudioView.vue'

const api = vi.hoisted(() => ({
  config: vi.fn(),
  sessions: vi.fn(),
  assets: vi.fn(),
  tasks: vi.fn(),
  account: vi.fn(),
  saveSession: vi.fn(),
  upload: vi.fn(),
  quote: vi.fn(),
  submit: vi.fn(),
  archiveTask: vi.fn(),
  confirm: vi.fn(),
}))
vi.mock('@/api/creative', () => ({ aiApi: api }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ user: { id: 1 } }) }))
vi.mock('@/composables/useToast', () => ({ useConfirm: () => ({ confirm: api.confirm }) }))
const wrappers: ReturnType<typeof mount>[] = []
const time = '2026-10-14T00:00:00.000Z'
const asset = (id: number) => ({
  id,
  task_id: 1,
  user_id: 1,
  mime: 'image/png',
  width: 1024,
  height: 1024,
  title: '',
  bytes: 100,
})
const task = (id: number, session_id = 1) => ({
  task: {
    id,
    session_id,
    prompt: `画面 ${id}`,
    status: 'succeeded',
    model: 'image-model',
    size: '1024x1024',
    count: 2,
    error: '',
  },
  assets: [asset(id * 2), asset(id * 2 + 1)],
})
function session(draft = '{}', version = 1) {
  return { id: 1, name: '我的创作', draft, version, created_at: time, updated_at: time }
}
async function render() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/studio', component: AIStudioView },
      { path: '/:pathMatch(.*)*', component: { template: '<div />' } },
    ],
  })
  await router.push('/studio')
  await router.isReady()
  const wrapper = mount(defineComponent({ template: '<router-view />' }), {
    attachTo: document.body,
    global: {
      plugins: [router],
      stubs: {
        CreativeImage: { props: ['id'], template: '<span class="test-image" :data-id="id" />' },
        CreativeLibrary: true,
        Modal: {
          props: ['visible'],
          template: '<section v-if="visible" class="test-modal"><slot /></section>',
        },
      },
    },
  })
  wrappers.push(wrapper)
  await flushPromises()
  return wrapper
}
function button(w: ReturnType<typeof mount>, text: string) {
  return w.findAll('button').find((b) => b.text() === text)!
}
function paste(w: ReturnType<typeof mount>, files: File[]) {
  const event = new Event('paste', { bubbles: true, cancelable: true })
  Object.defineProperty(event, 'clipboardData', { value: { files } })
  w.get('textarea').element.dispatchEvent(event)
  return event
}
beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  api.confirm.mockResolvedValue(true)
  api.config.mockResolvedValue({
    providers: [
      {
        id: 1,
        revision: 1,
        owner: 0,
        enabled: true,
        has_key: true,
        name: '站点',
        model: 'image-model',
        sizes: ['1024x1024', '1536x1024'],
        qualities: ['high'],
        max_count: 2,
        max_references: 2,
        cost: 2,
        quality_costs: {},
      },
    ],
  })
  api.sessions.mockResolvedValue([session()])
  api.assets.mockResolvedValue([])
  api.tasks.mockResolvedValue([])
  api.account.mockResolvedValue({ user_id: 1, available: 20, reserved: 0 })
  api.saveSession.mockImplementation(async (_id, _name, draft, version) =>
    session(JSON.stringify(draft), version + 1),
  )
  api.upload.mockResolvedValue(asset(9))
  api.quote.mockResolvedValue({ cost: 2 })
  api.submit.mockResolvedValue({ id: 99 })
  api.archiveTask.mockResolvedValue(undefined)
})
afterEach(() => wrappers.splice(0).forEach((w) => w.unmount()))

describe('studio gallery and composer', () => {
  it('uses the prompt and references without exposing a mode picker or BYOK controls', async () => {
    const w = await render()
    expect(w.find('.composer-settings').exists()).toBe(false)
    expect(w.find('.composer-mode-row').exists()).toBe(false)
    expect(w.find('[aria-label="API 设置"]').exists()).toBe(false)
    expect(w.find('[aria-label="切换会话"]').exists()).toBe(false)
    expect(w.find('[aria-label="创作会话名称"]').exists()).toBe(false)
    expect(w.text()).not.toContain('自备 API Key')
    expect(w.text()).not.toContain('管理员 · 生成不消耗站点额度')
    expect(w.get('[aria-label="画面尺寸"]').text()).toContain('自动')
  })

  it('can collapse the phone composer and reopens it when reusing a task prompt', async () => {
    api.tasks.mockResolvedValue([task(1)])
    const w = await render()
    await w.get('[aria-label="收起创作输入区"]').trigger('click')
    expect(w.get('.studio-composer').classes()).toContain('is-collapsed')
    await button(w, '复用提示词').trigger('click')
    await flushPromises()
    expect(w.get('.studio-composer').classes()).not.toContain('is-collapsed')
    expect(document.activeElement).toBe(w.get('textarea').element)
    await button(w, '生成设置').trigger('click')
    expect(button(w, '生成设置').attributes('aria-expanded')).toBe('true')
    expect(w.get('.composer-controls').classes()).toContain('is-mobile-open')
  })
  it('shows every result in the single workspace and continues editing from a completed task', async () => {
    api.tasks.mockResolvedValue([task(1), task(2, 2)])
    const w = await render()
    expect(w.findAll('.studio-task-card')).toHaveLength(2)
    expect(w.findAll('.task-images button')).toHaveLength(4)
    await w.get('[aria-label="查看任务 1 的结果 2"]').trigger('click')
    expect(w.get('.test-modal .test-image').attributes('data-id')).toBe('3')
    await w.get('.result-actions').findAll('button').find((item) => item.text() === '继续编辑')!.trigger('click')
    await flushPromises()
    expect(w.find('.test-modal').exists()).toBe(false)
    expect(w.get('.composer-references .test-image').attributes('data-id')).toBe('3')
    expect(document.activeElement).toBe(w.get('textarea').element)
    await w.get('textarea').setValue('继续画这张图')
    await button(w, '开始生成').trigger('click')
    await flushPromises()
    expect(api.submit).toHaveBeenCalledWith(
      expect.objectContaining({ parent_asset_id: 3, references: [{ asset_id: 3, note: '' }] }),
      expect.any(AbortSignal),
    )
  })
  it('removes failed task records while keeping successful tasks protected', async () => {
    const failed = task(3)
    failed.task.status = 'failed'
    failed.task.error = 'provider rejected the request'
    failed.assets = []
    api.tasks.mockResolvedValueOnce([failed]).mockResolvedValueOnce([])
    const w = await render()
    await button(w, '删除记录').trigger('click')
    await flushPromises()
    expect(api.archiveTask).toHaveBeenCalledWith(3)
    expect(w.findAll('.studio-task-card')).toHaveLength(0)
  })
  it('clears visible failed and cancelled records together', async () => {
    const failed = task(4)
    failed.task.status = 'failed'
    failed.task.error = '供应商拒绝请求'
    failed.assets = []
    const cancelled = task(5)
    cancelled.task.status = 'cancelled'
    cancelled.assets = []
    const succeeded = task(6)
    api.tasks.mockResolvedValueOnce([failed, cancelled, succeeded]).mockResolvedValueOnce([succeeded])
    const w = await render()
    const clear = w.get('[aria-label="清除失败任务"]')
    expect(clear.text()).toContain('2')
    await clear.trigger('click')
    await flushPromises()
    expect(api.archiveTask).toHaveBeenNthCalledWith(1, 4)
    expect(api.archiveTask).toHaveBeenNthCalledWith(2, 5)
    expect(w.findAll('.studio-task-card')).toHaveLength(1)
  })
  it('keeps plain text paste native and uploads a pasted image through the same reference path', async () => {
    const w = await render()
    expect(paste(w, []).defaultPrevented).toBe(false)
    expect(
      paste(w, [new File(['png'], 'reference.png', { type: 'image/png' })]).defaultPrevented,
    ).toBe(true)
    await flushPromises()
    expect(api.upload).toHaveBeenCalledWith(expect.any(File), expect.any(AbortSignal))
    expect(w.get('.reference-label').text()).toBe('@图片1')
    await w.get('.reference-label').trigger('click')
    expect((w.get('textarea').element as HTMLTextAreaElement).value).toBe('@图片1 ')
  })
  it('rejects an oversized reference batch before uploading any files', async () => {
    const w = await render()
    paste(
      w,
      Array.from({ length: 3 }, (_, i) => new File(['png'], `${i}.png`, { type: 'image/png' })),
    )
    await flushPromises()
    expect(api.upload).not.toHaveBeenCalled()
    expect(w.get('[role="alert"]').text()).toContain('参考图数量超出当前模型上限')
  })
  it('remaps reference mentions when pictures move or are removed', async () => {
    api.sessions.mockResolvedValue([
      session(
        JSON.stringify({
          prompt: '@图片1 与 @图片2',
          references: [
            { asset_id: 7, note: '' },
            { asset_id: 8, note: '' },
          ],
        }),
      ),
    ])
    const w = await render()
    await w.get('[aria-label="拖动图片1排序"]').trigger('keydown', { key: 'ArrowRight' })
    expect((w.get('textarea').element as HTMLTextAreaElement).value).toBe('@图片2 与 @图片1')
    await w.get('[aria-label="移除图片1"]').trigger('click')
    await flushPromises()
    expect(api.confirm).toHaveBeenCalled()
    expect((w.get('textarea').element as HTMLTextAreaElement).value).toBe('@图片1 与 ')
    expect(w.findAll('.composer-reference')).toHaveLength(1)
  })
  it('submits with Ctrl+Enter, preserves the text, and ignores an IME composition Enter', async () => {
    const w = await render()
    await w.get('textarea').setValue('窗边的小泉')
    await w.get('textarea').trigger('keydown', { key: 'Enter', ctrlKey: true, isComposing: true })
    expect(api.submit).not.toHaveBeenCalled()
    await w.get('textarea').trigger('keydown', { key: 'Enter', ctrlKey: true })
    await flushPromises()
    expect(api.submit).toHaveBeenCalledTimes(1)
    expect((w.get('textarea').element as HTMLTextAreaElement).value).toBe('窗边的小泉')
  })
  it('keeps one confirmation for repeated shortcuts and submits the visible output settings', async () => {
    let respond!: (accepted: boolean) => void
    api.confirm.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          respond = resolve
        }),
    )
    api.quote.mockResolvedValue({ cost: 4 })
    const w = await render()
    await w.get('textarea').setValue('两幅横向画面')
    await w.get('button[aria-label="画面尺寸"]').trigger('click')
    const ratioTab = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('.studio-size-tabs [role="tab"]'),
    ).find((tab) => tab.textContent?.includes('按比例'))
    expect(ratioTab).toBeTruthy()
    ratioTab!.click()
    await flushPromises()
    const tierOption = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('.studio-size-tiers [role="option"]'),
    ).find((option) => option.textContent?.trim().startsWith('2K'))
    expect(tierOption).toBeTruthy()
    tierOption!.click()
    const ratioOption = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('.studio-size-ratios [role="option"]'),
    ).find((option) => option.textContent?.includes('16:9'))
    expect(ratioOption).toBeTruthy()
    ratioOption!.click()
    const confirmSize = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('.studio-size-dialog button'),
    ).find((button) => button.textContent?.trim() === '确定')
    expect(confirmSize).toBeTruthy()
    confirmSize!.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await flushPromises()
    await w.get('select[aria-label="输出数量"]').setValue(2)
    await w.get('textarea').trigger('keydown', { key: 'Enter', metaKey: true })
    await w.get('textarea').trigger('keydown', { key: 'Enter', metaKey: true })
    expect(api.confirm).toHaveBeenCalledTimes(1)
    expect(api.submit).not.toHaveBeenCalled()
    respond(true)
    await flushPromises()
    expect(api.submit).toHaveBeenCalledTimes(1)
    expect(api.submit).toHaveBeenCalledWith(
      expect.objectContaining({ size: '2560x1440', count: 2, expected_cost: 4 }),
      expect.any(AbortSignal),
    )
  })
  it('submits automatic sizing without adding a concrete resolution', async () => {
    const w = await render()
    await w.get('textarea').setValue('让模型自行决定构图尺寸')
    await w.get('button[aria-label="画面尺寸"]').trigger('click')
    const confirmSize = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('.studio-size-dialog button'),
    ).find((button) => button.textContent?.trim() === '确定')
    expect(confirmSize).toBeTruthy()
    confirmSize!.click()
    await flushPromises()
    await button(w, '开始生成').trigger('click')
    await flushPromises()
    expect(api.submit).toHaveBeenCalledWith(
      expect.objectContaining({ size: 'auto' }),
      expect.any(AbortSignal),
    )
  })
  it('blocks drop and paste mutations while a lost submission is pending and reuses its original key', async () => {
    const pending = {
      session_id: 1,
      prompt: '已提交的画面',
      provider_id: 1,
      mode: 'site',
      size: '1024x1024',
      quality: '',
      count: 1,
      references: [],
      parent_asset_id: 0,
      expected_cost: 2,
      provider_revision: 1,
      character_ids: [],
      idempotency_key: 'original-key',
    }
    api.sessions.mockResolvedValue([session(JSON.stringify({ ...pending, pending }))])
    const w = await render()
    const file = new File(['png'], 'ref.png', { type: 'image/png' })
    paste(w, [file])
    await w.get('.studio-composer').trigger('drop', { dataTransfer: { files: [file] } })
    await flushPromises()
    expect(api.upload).not.toHaveBeenCalled()
    expect(w.get('textarea').element.matches(':disabled')).toBe(true)
    await button(w, '确认上次提交').trigger('click')
    await flushPromises()
    expect(api.submit).toHaveBeenCalledWith(
      expect.objectContaining({ idempotency_key: 'original-key' }),
      expect.any(AbortSignal),
    )
    expect(api.confirm).not.toHaveBeenCalled()
    expect(api.quote).not.toHaveBeenCalled()
  })
})
