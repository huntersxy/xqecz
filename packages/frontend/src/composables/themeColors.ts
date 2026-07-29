/**
 * 日间/暗色常量 —— 仅用于 AdminView 的 ant-design 动态 token。
 * CSS 变量的实际注入已移至 main.css 的 `:root` / `html.dark`，纯 CSS 切换，
 * 不再通过 JS 逐个 setProperty。
 */

export const LIGHT_COLORS = {
  primary: '#6366f1',
} as const

export const DARK_COLORS = {
  primary: '#818cf8',
} as const

export type ThemeColorKey = keyof typeof LIGHT_COLORS

/**
 * 切换日间/暗色：仅操作 document.documentElement 的 dark class。
 * CSS 变量值由 main.css 中的 `:root` / `html.dark` 控制。
 */
export function applyThemeColors(mode: 'light' | 'dark') {
  const root = document.documentElement
  root.classList.toggle('dark', mode === 'dark')
  root.classList.toggle('light', mode === 'light')
}
