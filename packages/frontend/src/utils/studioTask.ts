import { parseImageSize } from '@/utils/imageSize'
import type { AIAsset, AITaskView } from '@/types/schemas'

export interface GalleryItem {
  key: string
  row: AITaskView
  asset: AIAsset | null
  index: number
  total: number
}

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

export function stateTileRatio(size: string) {
  const parsed = parseImageSize(size)
  if (!parsed) return 1
  return Math.min(1.2, Math.max(0.8, parsed.width / parsed.height))
}
