// Test setup file for Vitest
// Mock static assets to prevent file URL errors
vi.mock('/icons/shield.svg', () => ({
  default: '/icons/shield.svg',
}))

vi.mock('/icons/play.svg', () => ({
  default: '/icons/play.svg',
}))

vi.mock('/icons/link.svg', () => ({
  default: '/icons/link.svg',
}))

vi.mock('/icons/user-circle.svg', () => ({
  default: '/icons/user-circle.svg',
}))

vi.mock('/icons/eye.svg', () => ({
  default: '/icons/eye.svg',
}))

// Mock CSS modules
vi.mock('*.css', () => ({}))
vi.mock('*.scss', () => ({}))

// jsdom 未实现 ResizeObserver：组件（WaterfallCard / useWaterfallLayout）挂载时会创建，
// 这里提供最小桩，避免单测崩溃。
if (typeof globalThis.ResizeObserver === 'undefined') {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  globalThis.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver
}