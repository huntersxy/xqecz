// 测试环境只有 jsdom 缺失的少量浏览器 API 需要补桩；静态资源导入交给 Vite 处理。
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
