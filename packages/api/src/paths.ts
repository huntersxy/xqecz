import { existsSync } from 'fs'
import { dirname, join } from 'path'

// 项目根目录：从本文件位置向上查找 pnpm-workspace.yaml 或 .git 标记。
// 这样无论项目放在哪个盘、哪条绝对路径下，data 永远落在「项目根/data」，
// 是一个相对于项目的位置，不写死任何盘符或绝对路径（符合「项目中的 data」约定）。
function findProjectRoot(start: string): string {
  let dir = start
  // 最多向上 6 级，避免极端情况下的死循环
  for (let i = 0; i < 6; i++) {
    if (existsSync(join(dir, 'pnpm-workspace.yaml')) || existsSync(join(dir, '.git'))) {
      return dir
    }
    const parent = dirname(dir)
    if (parent === dir) break
    dir = parent
  }
  // 兜底：本文件预期位于 packages/api/src，向上三级即项目根
  return join(start, '..', '..', '..')
}

export const PROJECT_ROOT = findProjectRoot(__dirname)
const DATA_DIR = join(PROJECT_ROOT, 'data')
// 三目录同级：uploads（原文件，扁平存放、文件名为前端计算的 md5）/ thumbs（缩略图）/ images（压缩图）
export const UPLOAD_DIR = join(DATA_DIR, 'uploads')
export const THUMB_DIR = join(DATA_DIR, 'thumbs')
export const IMAGES_DIR = join(DATA_DIR, 'images')
