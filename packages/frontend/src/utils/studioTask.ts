import { parseImageSize } from '@/utils/imageSize'
import type { AIAsset, AITaskView } from '@/types/schemas'

/** 「全部结果」瀑布流的一格：一张成图，或一个还没有成图的任务（排队/生成中/失败）。 */
export interface GalleryItem {
  /** 瀑布流布局与 v-for 共用的稳定键；成图用资产 id，无图任务用任务 id。 */
  key: string
  row: AITaskView
  asset: AIAsset | null
  /** 该图在所属任务内的序号（0 起）与任务图片总数，用于「2/4」角标。 */
  index: number
  total: number
}

/** 工作台任务状态文案：侧栏、全部结果、任务详情共用，避免各处各写一份。 */
export const TASK_LABELS: Record<string, string> = {
  queued: '排队中',
  running: '生成中',
  succeeded: '完成',
  failed: '失败',
  cancelled: '已取消',
  unknown: '待核查',
}

export function statusClass(value: string) {
  return value === 'succeeded'
    ? 'is-success'
    : ['failed', 'cancelled'].includes(value)
      ? 'is-danger'
      : 'is-accent'
}

export function isFailedStatus(value: string) {
  return value === 'failed' || value === 'cancelled'
}

export function isPendingStatus(value: string) {
  return value === 'queued' || value === 'running'
}

/** 无图任务在「全部结果」里的卡片标题。 */
export function stateTileLabel(value: string) {
  if (isFailedStatus(value)) return '生成失败'
  if (value === 'unknown') return '结果待核查'
  return TASK_LABELS[value] ?? value
}

export function clockOf(iso: string) {
  const date = new Date(iso)
  return Number.isNaN(date.getTime())
    ? '--:--'
    : date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })
}

export function dayOf(iso: string) {
  const date = new Date(iso)
  return Number.isNaN(date.getTime())
    ? '--.--'
    : `${String(date.getMonth() + 1).padStart(2, '0')}.${String(date.getDate()).padStart(2, '0')}`
}

/**
 * 无图任务卡片的宽高比：沿用任务请求的画幅，让「生成中」占位和最终成图在瀑布流里占相近的位置；
 * 自动尺寸取 1:1。夹在 4:5 ~ 6:5 之间，保证卡片里的提示词与按钮始终放得下，
 * 高度由宽度确定，瀑布流无需等内容撑开再量。
 */
export function stateTileRatio(size: string) {
  const parsed = parseImageSize(size)
  if (!parsed) return 1
  return Math.min(1.2, Math.max(0.8, parsed.width / parsed.height))
}
