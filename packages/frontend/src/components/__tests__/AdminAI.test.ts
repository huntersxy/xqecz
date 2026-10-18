import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AdminAI from '../admin/AdminAI.vue'

const api = vi.hoisted(() => ({
  providers: vi.fn(),
  adminAssets: vi.fn(),
  adminTasks: vi.fn(),
  adminIdeas: vi.fn(),
  adminLedger: vi.fn(),
  adminResources: vi.fn(),
  metadata: vi.fn(),
  grantBatch: vi.fn(),
  grantUsers: vi.fn(),
  resolve: vi.fn(),
  deleteAdminTask: vi.fn(),
  deleteIdea: vi.fn(),
  deleteAdminAsset: vi.fn(),
  deleteResource: vi.fn(),
  confirm: vi.fn(),
}))
vi.mock('@/api/creative', () => ({
  aiApi: api,
  creativeApi: { adminResources: api.adminResources, delete: api.deleteResource },
}))
vi.mock('@/composables/useToast', () => ({ useConfirm: () => ({ confirm: api.confirm }) }))
const asset = {
  id: 3,
  user_id: 1,
  task_id: 0,
  title: '设定图',
  description: '',
  purpose: '',
  license: '',
  public: false,
  download_allowed: false,
  archived: false,
}
const wrappers: ReturnType<typeof mount>[] = []
function render() {
  const w = mount(AdminAI, {
    global: {
      stubs: {
        RouterLink: true,
        CreativeImage: true,
        Modal: {
          props: ['visible'],
          template: '<section v-if="visible" class="test-modal"><slot /></section>',
        },
      },
    },
  })
  wrappers.push(w)
  return w
}
function button(w: ReturnType<typeof mount>, text: string) {
  return w.findAll('button').find((b) => b.text() === text)!
}
async function edit(w: ReturnType<typeof mount>) {
  await flushPromises()
  await button(w, '角色资源').trigger('click')
  await flushPromises()
  await button(w, '编辑 / 发布 / 归档').trigger('click')
}
beforeEach(() => {
  vi.clearAllMocks()
  for (const key of [
    'providers',
    'adminTasks',
    'adminIdeas',
    'adminLedger',
    'adminResources',
  ] as const)
    api[key].mockResolvedValue([])
  api.adminAssets.mockResolvedValue([{ ...asset }])
  api.metadata.mockResolvedValue(null)
  api.resolve.mockResolvedValue(null)
  api.confirm.mockResolvedValue(true)
  api.grantBatch.mockResolvedValue({ count: 2 })
  api.grantUsers.mockResolvedValue([
    { id: 1, username: '管理员' },
    { id: 2, username: '作者' },
  ])
})
afterEach(() => wrappers.splice(0).forEach((w) => w.unmount()))

describe('AdminAI form feedback', () => {
  async function openResolution() {
    api.adminTasks.mockResolvedValue([
      {
        task: {
          id: 12,
          user_id: 1,
          prompt: '测试核查',
          status: 'unknown',
          model: 'model',
          count: 1,
          cost: 2,
          created_at: '2026-10-15T00:00:00.000Z',
        },
        assets: [],
        references: [],
      },
    ])
    const w = render()
    await flushPromises()
    await button(w, '生图任务').trigger('click')
    await flushPromises()
    await button(w, '核查结算').trigger('click')
    return w
  }
  it.each(['', '  ', '确认', '核'.repeat(301)])(
    'validates resolution notes inside the dialog without changing settlement for invalid note %s',
    async (note) => {
      const w = await openResolution()
      await w.get('textarea[aria-label="核查依据与说明"]').setValue(note)
      await button(w, '确认未消耗，释放预留').trigger('click')
      expect(api.resolve).not.toHaveBeenCalled()
      expect(w.get('.test-modal [role="alert"]').text()).toBe('核查依据与说明须填写 3–300 个字')
      expect(w.find('.test-modal').exists()).toBe(true)
    },
  )
  it('keeps server errors and the note in the resolution dialog, and resets feedback on reopen', async () => {
    api.resolve.mockRejectedValue(new Error('额度预留记录不一致'))
    const w = await openResolution()
    await w.get('textarea[aria-label="核查依据与说明"]').setValue('供应商已确认未执行')
    await button(w, '确认未消耗，释放预留').trigger('click')
    await flushPromises()
    expect(w.get('.test-modal [role="alert"]').text()).toBe('额度预留记录不一致')
    expect((w.get('textarea').element as HTMLTextAreaElement).value).toBe('供应商已确认未执行')
    await button(w, '核查结算').trigger('click')
    expect(w.find('.test-modal [role="alert"]').exists()).toBe(false)
  })
  it.each([false, true])(
    'submits a trimmed note once and closes after settlement succeeds (consumed=%s)',
    async (consumed) => {
      let finish!: () => void
      api.resolve.mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            finish = resolve
          }),
      )
      const w = await openResolution()
      await w.get('textarea[aria-label="核查依据与说明"]').setValue('  供应商已核实计费情况  ')
      if (consumed) await w.get('.test-modal input[type="checkbox"]').setValue(true)
      const submit = button(w, consumed ? '确认结算' : '确认未消耗，释放预留')
      await submit.trigger('click')
      await submit.trigger('click')
      expect(api.resolve).toHaveBeenCalledTimes(1)
      expect(api.resolve).toHaveBeenCalledWith(12, consumed, '供应商已核实计费情况')
      expect(w.find('.test-modal').exists()).toBe(true)
      finish()
      await flushPromises()
      expect(w.find('.test-modal').exists()).toBe(false)
    },
  )
  it('shows task owners, prompt disclosure, always-visible references and full image previews', async () => {
    api.adminTasks.mockResolvedValue([
      {
        task: {
          id: 11,
          user_id: 25,
          prompt: '完整提示词'.repeat(80),
          status: 'succeeded',
          model: 'image-model',
          size: '1024x1024',
          count: 1,
          cost: 2,
          created_at: '2026-10-15T00:00:00.000Z',
        },
        user: { id: 25, username: '生成作者' },
        assets: [{ id: 31 }],
        references: [{ id: 32 }, { id: 33 }],
      },
    ])
    const w = render()
    await flushPromises()
    await button(w, '生图任务').trigger('click')
    await flushPromises()
    const record = w.get('.admin-task-record')
    expect(record.get('.admin-task-identity').text()).toContain('生成作者')
    expect(record.get('.admin-task-prompt summary').text()).toContain('完整提示词')
    expect(record.get('.admin-task-prompt pre').text()).toBe('完整提示词'.repeat(80))
    expect(record.get('.admin-task-references').find('details').exists()).toBe(false)
    expect(record.get('.admin-task-references').findAll('button')).toHaveLength(2)
    await record.get('button[aria-label="查看原始参考图 1"]').trigger('click')
    const image = w.get('.admin-asset-preview').getComponent({ name: 'CreativeImage' })
    expect(image.props('id')).toBe(32)
    expect(image.props('full')).toBe(true)
  })
  it('lets an administrator remove a terminal task and its generated results', async () => {
    api.adminTasks.mockResolvedValue([
      {
        task: {
          id: 13,
          user_id: 1,
          prompt: '待删除任务',
          status: 'succeeded',
          model: 'image-model',
          size: '1024x1024',
          count: 1,
          cost: 0,
          created_at: '2026-10-15T00:00:00.000Z',
        },
        assets: [{ id: 31 }],
        references: [],
      },
    ])
    const w = render()
    await flushPromises()
    await button(w, '生图任务').trigger('click')
    await flushPromises()
    await button(w, '删除任务及结果').trigger('click')
    await flushPromises()
    expect(api.deleteAdminTask).toHaveBeenCalledWith(13)
  })
  it('requires a license before publishing and keeps the editable metadata payload narrow', async () => {
    const w = render()
    await edit(w)
    const checkbox = w
      .get('.test-modal')
      .findAll('label')
      .find((l) => l.text() === '公开素材（允许生成、二创及下载）')!
    await checkbox.get('input').setValue(true)
    await button(w, '保存').trigger('click')
    await flushPromises()
    expect(api.metadata).not.toHaveBeenCalled()
    expect(w.get('.test-modal [role="alert"]').text()).toContain('公开素材前请填写来源与授权说明')
    await w.get('.test-modal select').setValue('原创（本人创作，允许用于生成与二创）')
    await button(w, '保存').trigger('click')
    await flushPromises()
    expect(api.metadata).toHaveBeenCalledWith(3, {
      title: '设定图',
      description: '',
      purpose: '',
      license: '原创（本人创作，允许用于生成与二创）',
      public: true,
      download_allowed: true,
      archived: false,
    })
    expect(w.find('.test-modal').exists()).toBe(false)
  })
  it('shows a server rejection inside the open editor and preserves the draft', async () => {
    api.metadata.mockRejectedValue(new Error('只能将自己上传的设定图设为公开素材'))
    const w = render()
    await edit(w)
    await button(w, '保存').trigger('click')
    await flushPromises()
    expect(w.get('.test-modal [role="alert"]').text()).toBe('只能将自己上传的设定图设为公开素材')
    expect((w.get('.test-modal input').element as HTMLInputElement).value).toBe('设定图')
  })
  it('offers user selection and all users, and rejects an empty selection', async () => {
    const w = render()
    await flushPromises()
    await button(w, '用户额度 / 账本').trigger('click')
    await button(w, '发放额度').trigger('click')
    await flushPromises()
    expect(api.grantBatch).not.toHaveBeenCalled()
    expect(w.text()).toContain('请选择要发放额度的用户')
    const all = w.findAll('label').find((l) => l.text() === '全体用户')!
    await all.get('input').setValue(true)
    w.findComponent({ name: 'InputNumber' }).vm.$emit('update:modelValue', 5)
    await w.get('input[aria-label="发放原因"]').setValue('活动奖励')
    await button(w, '发放额度').trigger('click')
    await flushPromises()
    expect(api.grantBatch).toHaveBeenCalledWith([], true, 5, '活动奖励', expect.any(String))
    expect(w.text()).toContain('已为 2 位用户各增加 5 额度')
  })
})
