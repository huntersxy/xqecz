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
// 以下为波次 2（WP3）新增：把各页签里重复写的列定义收成常量，供波次 3 的表格重制直接引用。
// 只新增，上方的既有常量一个不动（各页签当前的内联列定义也保持不变）。
export const TAGS_COL: TableColumnData = {
  title: '标签', slotName: 'tags', width: 180,
}

export const AUTHOR_COL: TableColumnData = {
  title: '作者', slotName: 'author', minWidth: 80,
}

export const USERNAME_COL: TableColumnData = {
  title: '用户名', slotName: 'username', minWidth: 160,
}

export const ROLE_COL: TableColumnData = {
  title: '角色', slotName: 'role', width: 110, align: 'center',
}

export const REPORTER_COL: TableColumnData = {
  title: '举报人', slotName: 'reporter', width: 110,
}

