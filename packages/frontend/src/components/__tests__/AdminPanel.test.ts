import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AdminPanel from '../admin/AdminPanel.vue'
import AdminEmpty from '../admin/AdminEmpty.vue'
import AdminSkeleton from '../admin/AdminSkeleton.vue'
import AdminDataToolbar from '../admin/AdminDataToolbar.vue'


const mounted: { unmount: () => void }[] = []

function track<T extends { unmount: () => void }>(wrapper: T) {
  mounted.push(wrapper)
  return wrapper
}

afterEach(() => {
  mounted.splice(0).forEach((wrapper) => wrapper.unmount())
})

type PanelProps = { title: string; desc?: string; flat?: boolean; stagger?: boolean }

function mountPanel(props: PanelProps = { title: '用户管理' }, slots: Record<string, string> = {}) {
  return track(mount(AdminPanel, { props, slots }))
}

function childClasses(wrapper: ReturnType<typeof mountPanel>) {
  return Array.from(wrapper.element.children).map((el) => el.className)
}

describe('AdminPanel 向后兼容', () => {
  it('renders the legacy structure when only a title is passed', () => {
    const wrapper = mountPanel()

    expect(wrapper.classes()).toEqual(['admin-panel'])
    expect(childClasses(wrapper)).toEqual(['admin-panel-head', 'admin-panel-body'])

    const header = wrapper.find('.admin-panel-head')
    expect(header.exists()).toBe(true)
    expect(Array.from(header.element.children).map((el) => el.className)).toEqual(['admin-panel-head-text'])

    const title = wrapper.find('h2.admin-panel-title')
    expect(title.exists()).toBe(true)
    expect(title.text()).toBe('用户管理')

    expect(wrapper.find('.admin-panel-body').exists()).toBe(true)
  })

  it('renders the description only when desc is passed', () => {
    expect(mountPanel().find('.admin-panel-desc').exists()).toBe(false)

    const wrapper = mountPanel({ title: '用户管理', desc: '共 128 位用户' })
    const desc = wrapper.find('p.admin-panel-desc')
    expect(desc.exists()).toBe(true)
    expect(desc.text()).toBe('共 128 位用户')
  })

  it('keeps the actions slot inside the header and omits the wrapper when unused', () => {
    expect(mountPanel().find('.admin-panel-actions').exists()).toBe(false)
    expect(childClasses(mountPanel())).toEqual(['admin-panel-head', 'admin-panel-body'])

    const wrapper = mountPanel({ title: '用户管理' }, { actions: '<button class="legacy-action">刷新</button>' })
    const actions = wrapper.find('.admin-panel-head > .admin-panel-actions')
    expect(actions.exists()).toBe(true)
    expect(actions.find('.legacy-action').text()).toBe('刷新')
    expect(childClasses(wrapper)).toEqual(['admin-panel-head', 'admin-panel-body'])
  })

  it('renders the default slot inside the panel body', () => {
    const wrapper = mountPanel({ title: '用户管理' }, { default: '<div class="panel-content">内容</div>' })
    const body = wrapper.find('.admin-panel-body')
    expect(body.find('.panel-content').text()).toBe('内容')
  })

  it('keeps flat as a root modifier without changing the node tree', () => {
    const wrapper = mountPanel({ title: '仪表盘', desc: '站点数据总览', flat: true })
    expect(wrapper.classes()).toContain('admin-panel')
    expect(wrapper.classes()).toContain('is-flat')
    expect(childClasses(wrapper)).toEqual(['admin-panel-head', 'admin-panel-body'])
  })
})

describe('AdminPanel 新增能力', () => {
  it('renders the toolbar slot between the header and the body only when passed', () => {
    const legacy = mountPanel()
    expect(legacy.find('.admin-panel-toolbar').exists()).toBe(false)

    const wrapper = mountPanel({ title: '用户管理' }, { toolbar: '<div class="toolbar-slot">共 3 条</div>' })
    expect(childClasses(wrapper)).toEqual(['admin-panel-head', 'admin-panel-toolbar', 'admin-panel-body'])
    expect(wrapper.find('.admin-panel-toolbar .toolbar-slot').text()).toBe('共 3 条')
  })

  it('makes the head focusable and exposes focusTitle for tab switching', () => {
    const wrapper = track(mount(AdminPanel, { props: { title: '用户管理' }, attachTo: document.body }))
    const title = wrapper.find('h2.admin-panel-title')
    expect(title.attributes('tabindex')).toBe('-1')

    ;(wrapper.vm as unknown as { focusTitle: () => void }).focusTitle()
    expect(document.activeElement).toBe(title.element)
  })

  it('opts into the stagger animation only when stagger is passed', () => {
    const legacy = mountPanel()
    expect(legacy.find('.admin-panel-body').classes()).toEqual(['admin-panel-body'])

    const staggered = mountPanel({ title: '用户管理', stagger: true })
    expect(staggered.find('.admin-panel-body').classes()).toContain('adm-stagger')
  })
})

describe('AdminSkeleton', () => {
  it('renders row placeholders with a screen-reader label', () => {
    const wrapper = track(mount(AdminSkeleton, { props: { rows: 3, columns: 2 } }))

    expect(wrapper.attributes('role')).toBe('status')
    expect(wrapper.attributes('aria-busy')).toBe('true')
    expect(wrapper.find('.admin-skeleton-sr').text()).toBe('正在加载')
    expect(wrapper.findAll('.admin-skeleton-row')).toHaveLength(3)
    expect(wrapper.findAll('.admin-skeleton-row')[0]!.findAll('.admin-skeleton-cell')).toHaveLength(2)
  })

  it('renders card placeholders and clamps the count', () => {
    const wrapper = track(mount(AdminSkeleton, { props: { variant: 'cards', rows: 0 } }))

    expect(wrapper.classes()).toContain('is-cards')
    expect(wrapper.findAll('.admin-skeleton-card')).toHaveLength(1)
    expect(wrapper.find('.admin-skeleton-card .admin-skeleton-line').exists()).toBe(true)
  })
})

describe('AdminEmpty', () => {
  it('falls back to the default title and renders actions when provided', () => {
    const plain = track(mount(AdminEmpty))
    expect(plain.find('.admin-empty-title').text()).toBe('暂无数据')
    expect(plain.find('.admin-empty-desc').exists()).toBe(false)
    expect(plain.find('.admin-empty-actions').exists()).toBe(false)

    const wrapper = track(mount(AdminEmpty, {
      props: { title: '还没有内容', desc: '上传第一条内容试试', compact: true },
      slots: { default: '<span class="empty-extra">提示</span>', actions: '<button>去上传</button>' },
    }))
    expect(wrapper.classes()).toContain('is-compact')
    expect(wrapper.find('.admin-empty-title').text()).toBe('还没有内容')
    expect(wrapper.find('.admin-empty-desc').text()).toBe('上传第一条内容试试')
    expect(wrapper.find('.admin-empty-extra .empty-extra').text()).toBe('提示')
    expect(wrapper.find('.admin-empty-actions button').text()).toBe('去上传')
  })
})

describe('AdminDataToolbar', () => {
  it('shows the summary and emits refresh from the built-in button', async () => {
    const wrapper = track(mount(AdminDataToolbar, { props: { summary: '共 128 条内容', refreshable: true } }))

    expect(wrapper.find('.admin-data-toolbar-summary').text()).toBe('共 128 条内容')
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })

  it('renders no built-in button unless refreshable is set', () => {
    const wrapper = track(mount(AdminDataToolbar, { props: { summary: '共 3 条' } }))
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.find('.admin-data-toolbar-actions').exists()).toBe(true)
  })
})
