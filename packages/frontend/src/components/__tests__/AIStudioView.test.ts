import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AIStudioView from '@/views/AIStudioView.vue'

const api = vi.hoisted(() => ({
  config: vi.fn(),
  sessions: vi.fn(),
  session: vi.fn(),
  createSession: vi.fn(),
  assets: vi.fn(),
  tasks: vi.fn(),
  account: vi.fn(),
  saveSession: vi.fn(),
  upload: vi.fn(),
  quote: vi.fn(),
  submit: vi.fn(),
  archiveTask: vi.fn(),
  setLiked: vi.fn(),
  confirm: vi.fn(),
}))
vi.mock('@/api/creative', () => ({ aiApi: api }))
vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ user: { id: 1, username: '小泉', email: 'a@qq.com' } }),
}))
vi.mock('@/composables/useToast', () => ({ useConfirm: () => ({ confirm: api.confirm }) }))
const wrappers: ReturnType<typeof mount>[] = []
const time = '2026-10-14T00:00:00.000Z'
const asset = (id: number, liked = false) => ({
  id,
  task_id: 1,
  user_id: 1,
  mime: 'image/png',
  width: 1024,
  height: 1024,
  title: '',
  bytes: 100,
  liked,
  created_at: time,
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
    created_at: time,
  },
  assets: [asset(id * 2), asset(id * 2 + 1)],
  references: [],
})
function session(draft = '{}', version = 1, id = 1) {
  return { id, name: '我的创作', draft, version, created_at: time, updated_at: time }
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
  return w.findAll('button').find((item) => item.text().trim() === text)!
}
function paste(w: ReturnType<typeof mount>, files: File[]) {
  const event = new Event('paste', { bubbles: true, cancelable: true })
  Object.defineProperty(event, 'clipboardData', { value: { files } })
  w.get('textarea').element.dispatchEvent(event)
  return event
}
async function openSettings(w: ReturnType<typeof mount>) {
  await w.get('[aria-label="生成设置"]').trigger('click')
  await flushPromises()
}
beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  api.confirm.mockResolvedValue(true)
  api.config.mockResolvedValue({
    prompt_enhancement: { enabled: true, prompt: '补充画面细节' },
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
  api.session.mockImplementation(async (id: number) => session('{}', 1, id))
  api.createSession.mockImplementation(async (name: string) => session('{}', 1, 2))
  api.assets.mockResolvedValue([])
  api.tasks.mockResolvedValue([])
  api.account.mockResolvedValue({ user_id: 1, available: 20, reserved: 0 })
  api.saveSession.mockImplementation(async (id, _name, draft, version) =>
    session(JSON.stringify(draft), version + 1, id),
  )
  api.upload.mockResolvedValue(asset(9))
  api.quote.mockResolvedValue({ cost: 2 })
  api.submit.mockResolvedValue({ id: 99 })
  api.archiveTask.mockResolvedValue(undefined)
  api.setLiked.mockImplementation(async (id: number, liked: boolean) => asset(id, liked))
})
afterEach(() => wrappers.splice(0).forEach((w) => w.unmount()))

describe('studio workspace and composer', () => {
  it('uses the prompt and references without exposing a mode picker or BYOK controls', async () => {
    const w = await render()
    expect(w.find('.composer-settings').exists()).toBe(false)
    expect(w.find('.composer-mode-row').exists()).toBe(false)
    expect(w.find('[aria-label="API 设置"]').exists()).toBe(false)
    expect(w.find('[aria-label="切换会话"]').exists()).toBe(false)
    expect(w.text()).not.toContain('自备 API Key')
    expect(w.text()).not.toContain('管理员 · 生成不消耗站点额度')
    expect(w.find('[aria-label="画面尺寸"]').exists()).toBe(false)
    await openSettings(w)
    expect(w.get('[aria-label="画面尺寸"]').text()).toContain('自动')
    expect(w.get('[aria-label="生成设置"]').attributes('aria-expanded')).toBe('true')
  })

  it('opens generation settings and refocuses the prompt when reusing a task prompt', async () => {
    api.tasks.mockResolvedValue([task(1)])
    const w = await render()
    await openSettings(w)
    expect(w.find('.ws-settings').exists()).toBe(true)
    await button(w, '复用提示词').trigger('click')
    await flushPromises()
    expect(document.activeElement).toBe(w.get('textarea').element)
    expect((w.get('textarea').element as HTMLTextAreaElement).value).toBe('画面 1')
  })

  it('still switches to the all-result grid when the workspace has one image', async () => {
    const single = task(1)
    single.assets = [asset(2)]
    single.task.count = 1
    api.tasks.mockResolvedValue([single])
    const w = await render()
    await button(w, '全部').trigger('click')
    const grid = w.get('.ws-canvas-grid')
    expect(grid.classes()).toContain('is-single')
    expect(grid.findAll('button')).toHaveLength(1)
    await grid.get('button').trigger('click')
    expect(w.find('.ws-canvas-grid').exists()).toBe(false)
    expect(w.get('.ws-canvas-image .test-image').attributes('data-id')).toBe('2')
  })

  it('shows every workspace result in all view and continues editing from the chosen one', async () => {
    api.tasks.mockResolvedValue([task(1), task(2, 1)])
    const w = await render()
    expect(w.findAll('.ws-history-item')).toHaveLength(2)
    expect(w.get('.ws-canvas-image .test-image').attributes('data-id')).toBe('2')
    await button(w, '全部').trigger('click')
    expect(w.findAll('.ws-canvas-grid button')).toHaveLength(4)
    await w.findAll('.ws-canvas-grid button')[2]!.trigger('click')
    expect(w.findAll('.ws-canvas-grid')).toHaveLength(0)
    expect(w.get('.ws-canvas-image .test-image').attributes('data-id')).toBe('4')
    await w.get('.ws-canvas-image').trigger('click')
    expect(w.get('.test-modal .test-image').attributes('data-id')).toBe('4')
    await w
      .get('.result-actions')
      .findAll('button')
      .find((item) => item.text() === '继续编辑')!
      .trigger('click')
    await flushPromises()
    expect(w.find('.test-modal').exists()).toBe(false)
    expect(w.get('.composer-references .test-image').attributes('data-id')).toBe('4')
    expect(document.activeElement).toBe(w.get('textarea').element)
    await w.get('textarea').setValue('继续画这张图')
    await w.get('[aria-label="开始生成"]').trigger('click')
    await flushPromises()
    expect(api.submit).toHaveBeenCalledWith(
      expect.objectContaining({ parent_asset_id: 4, references: [{ asset_id: 4, note: '' }] }),
      expect.any(AbortSignal),
    )
  })

  it('likes a selected result and lists it under the liked tab', async () => {
    api.tasks.mockResolvedValue([task(1)])
    const w = await render()
    await button(w, '喜欢').trigger('click')
    await flushPromises()
    expect(api.setLiked).toHaveBeenCalledWith(2, true)
    await button(w, '喜欢').trigger('click')
    expect(api.setLiked).toHaveBeenLastCalledWith(2, false)
  })

  it('removes failed task records while keeping successful tasks protected', async () => {
    const failed = task(3)
    failed.task.status = 'failed'
    failed.task.error = 'provider rejected the request'
    failed.assets = []
    api.tasks.mockResolvedValueOnce([failed]).mockResolvedValueOnce([])
    const w = await render()
    await w.get('.ws-history-item').trigger('click')
    await button(w, '删除记录').trigger('click')
    await flushPromises()
    expect(api.archiveTask).toHaveBeenCalledWith(3)
    expect(w.findAll('.ws-history-item')).toHaveLength(0)
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
    expect(w.findAll('.ws-history-item')).toHaveLength(1)
  })

  it('keeps plain text paste native and uploads a pasted image through the same reference path', async () => {
    const w = await render()
    expect(paste(w, []).defaultPrevented).toBe(false)
    expect(
      paste(w, [new File(['png'], 'reference.png', { type: 'image/png' })]).defaultPrevented,
    ).toBe(true)
    await flushPromises()
    expect(api.upload).toHaveBeenCalledWith(expect.any(File), expect.any(AbortSignal))
    expect(w.get('.reference-label').text()).toBe('@1')
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
    await w.get('[aria-label="预览参考图 1"]').trigger('keydown', { key: 'ArrowRight' })
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

  it('submits with Shift+Enter like the dock hint', async () => {
    const w = await render()
    await w.get('textarea').setValue('Shift 提交')
    await w.get('textarea').trigger('keydown', { key: 'Enter', shiftKey: true })
    await flushPromises()
    expect(api.submit).toHaveBeenCalledTimes(1)
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
    await openSettings(w)
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
    ).find((item) => item.textContent?.trim() === '确定')
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
    await openSettings(w)
    await w.get('textarea').setValue('让模型自行决定构图尺寸')
    await w.get('button[aria-label="画面尺寸"]').trigger('click')
    const confirmSize = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('.studio-size-dialog button'),
    ).find((item) => item.textContent?.trim() === '确定')
    expect(confirmSize).toBeTruthy()
    confirmSize!.click()
    await flushPromises()
    await w.get('[aria-label="开始生成"]').trigger('click')
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
    await w.get('.ws-composer').trigger('drop', { dataTransfer: { files: [file] } })
    await flushPromises()
    expect(api.upload).not.toHaveBeenCalled()
    expect(w.get('textarea').element.matches(':disabled')).toBe(true)
    await w.get('[aria-label="确认上次提交"]').trigger('click')
    await flushPromises()
    expect(api.submit).toHaveBeenCalledWith(
      expect.objectContaining({ idempotency_key: 'original-key' }),
      expect.any(AbortSignal),
    )
    expect(api.confirm).not.toHaveBeenCalled()
    expect(api.quote).not.toHaveBeenCalled()
  })
})
