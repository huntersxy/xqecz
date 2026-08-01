import type { TableColumnData } from '@arco-design/web-vue'

export const ACTION_COL: TableColumnData = {
  title: '操作', slotName: 'actions', width: 110, align: 'center',
}

export const STATUS_COL: TableColumnData = {
  title: '状态', slotName: 'status', width: 100, align: 'center',
}

export const TIME_COL: TableColumnData = {
  title: '时间', slotName: 'time', width: 160,
}

export const CONTENT_COL: TableColumnData = {
  title: '内容', slotName: 'content', minWidth: 220,
}

export const REASON_COL: TableColumnData = {
  title: '理由', slotName: 'reason', minWidth: 160,
}

export const CLAIMER_COL: TableColumnData = {
  title: '认领者', slotName: 'claimer', width: 110,
}
