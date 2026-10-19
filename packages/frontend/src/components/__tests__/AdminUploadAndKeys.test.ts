// WP6：上传内容 / API 密钥 / API 文档三页的交互契约。
// 网络层与 store 全部打桩，只验证组件自己的校验、提交、复制与目录切换。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import AdminUploadPanel from '@/components/admin/AdminUploadPanel.vue'
import AdminApiKeys from '@/components/admin/AdminApiKeys.vue'
import AdminApiDocs from '@/components/admin/AdminApiDocs.vue'
import type { ApiKey } from '@/types'

const uploadContent = vi.hoisted(() => vi.fn(async () => true))
const listKeys = vi.hoisted(() => vi.fn())
const createKey = vi.hoisted(() => vi.fn())
const updateKey = vi.hoisted(() => vi.fn(async () => ({ code: 200, data: null })))
const deleteKey = vi.hoisted(() => vi.fn(async () => ({ code: 200, data: null })))
const metadataOptions = vi.hoisted(() => vi.fn(async () => ({
  code: 200,
  data: { available: false, ready: false, placeholder_title: '', placeholder_tag: '' },
})))
const copyToClipboard = vi.hoisted(() => vi.fn(async () => undefined))

const toast = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
}))
const confirm = vi.hoisted(() => vi.fn(async () => true))

vi.mock('@/api', () => ({
  contentApi: { metadataOptions },
  apiKeyApi: { list: listKeys, create: createKey, update: updateKey, delete: deleteKey },
}))

vi.mock('@/composables/useToast', () => ({
  toast,
  useConfirm: () => ({ confirm }),
}))

// 上传面板只需要「已登录 + 上传动作」两个外部依赖
vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ user: { id: 9, username: 'admin', role: 'admin' } }),
}))

vi.mock('@/stores/admin', () => ({
  useAdminStore: () => ({
    uploading: false,
    uploadProgress: 0,
    tags: ['风景', '人物'],
    uploadContent,
  }),
}))

// Markdown 编辑器在 jsdom 里没有 CodeMirror，用 textarea 保持 v-model 语义
vi.mock('@/components/MarkdownEditor.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      name: 'MarkdownEditorStub',
      props: { modelValue: { type: String, default: '' } },
      emits: ['update:modelValue'],
      setup(props, { emit }) {
        return () => h('textarea', {
          class: 'markdown-stub',
          value: props.modelValue,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
        })
      },
    }),
  }
})

const wrappers: VueWrapper[] = []

function mountPage(component: typeof AdminUploadPanel) {
  const wrapper = mount(component, { attachTo: document.body })
  wrappers.push(wrapper)
  return wrapper
}

function buttonByText(wrapper: VueWrapper, text: string) {
  const hit = wrapper.findAll('button').find((btn) => btn.text().trim() === text)
  if (!hit) throw new Error('button not found: ' + text)
  return hit
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('matchMedia', vi.fn((media: string) => ({
    matches: false,
    media,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })))
  // 目录点击后调 scrollIntoView，jsdom 未实现
  Element.prototype.scrollIntoView = vi.fn()
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: copyToClipboard },
  })
  listKeys.mockResolvedValue({ code: 200, data: { list: [] } })
})

afterEach(() => {
  while (wrappers.length) wrappers.pop()?.unmount()
  document.body.innerHTML = ''
})

const keyFixture = (id: number, name: string, active: boolean): ApiKey => ({
  id,
  name,
  key_prefix: 'xq_live_ab',
  permissions: ['upload'],
  is_active: active,
  last_used_at: null,
  created_at: '2026-10-01T00:00:00.000Z',
})

describe('上传内容面板', () => {
  it('必填校验先拦标题，提交动作不会触达上传接口', async () => {
    const wrapper = mountPage(AdminUploadPanel)

    await buttonByText(wrapper, '提交发布').trigger('click')

    expect(toast.warning).toHaveBeenCalledWith('请填写标题')
    expect(uploadContent).not.toHaveBeenCalled()
  })

  it('有标题但正文与媒体都为空时提示二选一', async () => {
    const wrapper = mountPage(AdminUploadPanel)

    await wrapper.find('input[placeholder="输入标题"]').setValue('测试标题')
    await buttonByText(wrapper, '提交发布').trigger('click')

    expect(toast.warning).toHaveBeenCalledWith('请填写描述正文或上传媒体文件')
    expect(uploadContent).not.toHaveBeenCalled()
  })

  it('填写完整后把标题、正文与用户 id 交给上传动作，成功后清空表单', async () => {
    const wrapper = mountPage(AdminUploadPanel)

    await wrapper.find('input[placeholder="输入标题"]').setValue('  测试标题  ')
    const editor = wrapper.find('textarea.markdown-stub')
    await editor.setValue('正文 **Markdown**')
    await buttonByText(wrapper, '提交发布').trigger('click')
    await flushPromises()

    expect(uploadContent).toHaveBeenCalledTimes(1)
    expect(uploadContent).toHaveBeenCalledWith(expect.objectContaining({
      title: '测试标题',
      content: '正文 **Markdown**',
      userId: 9,
    }))
    const titleInput = wrapper.find('input[placeholder="输入标题"]').element as HTMLInputElement
    expect(titleInput.value).toBe('')
  })
})

describe('API 密钥面板', () => {
  it('列表就绪后工具区给出总数与启用数', async () => {
    listKeys.mockResolvedValue({
      code: 200,
      data: { list: [keyFixture(1, '甲密钥', true), keyFixture(2, '乙密钥', false)] },
    })
    const wrapper = mountPage(AdminApiKeys)
    await flushPromises()

    expect(listKeys).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('共 2 个密钥，启用 1 个')
    expect(wrapper.text()).toContain('甲密钥')
  })

  it('创建后弹层展示完整密钥，复制按钮写入剪贴板并给出成功反馈', async () => {
    createKey.mockResolvedValue({
      code: 200,
      data: { id: 3, name: '新密钥', key: 'xq_完整密钥_只显示一次', permissions: ['upload'], is_active: true, created_at: '2026-10-02T00:00:00.000Z' },
    })
    const wrapper = mountPage(AdminApiKeys)
    await flushPromises()

    await buttonByText(wrapper, '新建密钥').trigger('click')
    await flushPromises()
    const nameInput = document.querySelector('.arco-modal input') as HTMLInputElement
    nameInput.value = '新密钥'
    nameInput.dispatchEvent(new Event('input'))
    await flushPromises()

    const okButton = Array.from(document.querySelectorAll('.arco-modal button'))
      .find((btn) => btn.textContent?.trim() === '创建') as HTMLButtonElement
    okButton.click()
    await flushPromises()

    expect(createKey).toHaveBeenCalledWith({ name: '新密钥', permissions: ['upload'] })
    expect(document.body.textContent).toContain('xq_完整密钥_只显示一次')
    expect(document.body.textContent).toContain('关闭后将无法再次查看')

    const copyButton = Array.from(document.querySelectorAll('button'))
      .find((btn) => btn.textContent?.trim() === '复制') as HTMLButtonElement
    copyButton.click()
    await flushPromises()

    expect(copyToClipboard).toHaveBeenCalledWith('xq_完整密钥_只显示一次')
    expect(toast.success).toHaveBeenCalledWith('已复制到剪贴板')
    // 弹层由 a-modal teleport 到 body，断言整页文本而不是组件根节点
    expect(document.body.textContent).toContain('已复制')
  })

  it('确认撤销后调用删除接口并刷新列表', async () => {
    listKeys.mockResolvedValue({ code: 200, data: { list: [keyFixture(5, '待撤销密钥', true)] } })
    const wrapper = mountPage(AdminApiKeys)
    await flushPromises()

    const revokeButton = wrapper.find('.admin-icon-btn.is-danger')
    expect(revokeButton.exists()).toBe(true)
    await revokeButton.trigger('click')
    await flushPromises()

    expect(confirm).toHaveBeenCalledWith(expect.stringContaining('确定要撤销 "待撤销密钥" 吗？'))
    expect(deleteKey).toHaveBeenCalledWith(5)
    expect(toast.success).toHaveBeenCalledWith('已撤销')
  })
})

describe('API 文档页', () => {
  it('没有 IntersectionObserver 时也能渲染，目录条数与章节一一对应', async () => {
    vi.stubGlobal('IntersectionObserver', undefined)
    const wrapper = mountPage(AdminApiDocs as unknown as typeof AdminUploadPanel)

    await flushPromises()
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(wrapper.find('.api-main').exists()).toBe(true)
    expect(wrapper.findAll('.api-toc-item')).toHaveLength(8)
    expect(wrapper.find('#sec-auth').exists()).toBe(true)
  })

  it('点击目录项高亮当前项并滚动到对应章节', async () => {
    vi.stubGlobal('IntersectionObserver', undefined)
    const wrapper = mountPage(AdminApiDocs as unknown as typeof AdminUploadPanel)
    await flushPromises()

    const items = wrapper.findAll('.api-toc-item')
    await items[2].trigger('click')
    await flushPromises()

    expect(items[2].classes()).toContain('is-active')
    expect(items[2].attributes('aria-current')).toBe('true')
    expect(Element.prototype.scrollIntoView).toHaveBeenCalled()
  })
})
