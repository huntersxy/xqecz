/**
 * The studio sidebar is rendered inside PublicShell's real site sidebar. The
 * teleported content is logically owned by AIStudioView, so it cannot inject
 * PublicShell's drawer state; this event is the single close hand-off used by
 * both sides. On desktop it is a no-op because the drawer is already closed.
 */
export const SHELL_SIDEBAR_CLOSE_EVENT = 'xq:close-shell-sidebar'

export function closeShellSidebar() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(SHELL_SIDEBAR_CLOSE_EVENT))
  }
}
