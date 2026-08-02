import { extname } from 'path'
import { stat, unlink } from 'fs/promises'
import sharp from 'sharp'

/**
 * 上传图片的本地无损 WebP 化（非 GIF）。
 *
 * 规则：
 * - GIF（动图）与已经是 WebP 的文件跳过，保持原样；
 * - 其余图片用 sharp 以 lossless 模式转出同目录同名 `.webp`，
 *   转换成功后删除源文件，返回值作为新原图（绝对路径 + 大小）；
 * - 任何失败都降级：保留源文件并返回 null，不阻塞上传（后续缩略图/压缩照常）。
 */
export interface WebpConversion {
  /** 转换后的 WebP 绝对路径（新原图） */
  absPath: string
  /** 转换后的文件大小 */
  size: number
}

export async function convertNonGifToWebp(
  absPath: string,
  mimetype?: string,
): Promise<WebpConversion | null> {
  const ext = extname(absPath).toLowerCase()
  if (ext === '.gif' || ext === '.webp') return null
  if (mimetype && !mimetype.startsWith('image/')) return null

  const webpPath = `${absPath.slice(0, absPath.length - ext.length)}.webp`
  try {
    await sharp(absPath, { failOn: 'error' })
      .webp({ lossless: true, effort: 4 })
      .toFile(webpPath)
  } catch (err) {
    console.warn(`[webp] 无损转换失败，保留原文件: ${absPath} -> ${(err as Error)?.message || err}`)
    await unlink(webpPath).catch(() => undefined)
    return null
  }

  try {
    await unlink(absPath)
  } catch (err) {
    console.warn(`[webp] 源文件删除失败（忽略）: ${absPath} -> ${(err as Error)?.message || err}`)
  }

  const size = (await stat(webpPath)).size
  return { absPath: webpPath, size }
}
