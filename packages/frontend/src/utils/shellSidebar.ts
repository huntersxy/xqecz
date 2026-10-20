export const SHELL_SIDEBAR_CLOSE_EVENT = 'xq:close-shell-sidebar'

export function closeShellSidebar() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(SHELL_SIDEBAR_CLOSE_EVENT))
  }
}
