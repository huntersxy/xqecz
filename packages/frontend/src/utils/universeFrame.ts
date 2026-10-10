/** 空闲时不轮询，交互/贴图更新只预约一帧；每次绘制跟随屏幕刷新，无 30fps 人为上限。 */
export function createUniverseFrame(render: () => void, request = requestAnimationFrame, cancel = cancelAnimationFrame) {
  let handle = 0
  let active = false
  let disposed = false
  function invalidate() {
    if (!active || disposed || handle) return
    handle = request(() => { handle = 0; if (active && !disposed) render() })
  }
  return {
    invalidate,
    setActive(value: boolean) {
      active = value
      if (handle) cancel(handle)
      handle = 0
      if (value) invalidate()
    },
    dispose() { disposed = true; active = false; if (handle) cancel(handle); handle = 0 },
  }
}
