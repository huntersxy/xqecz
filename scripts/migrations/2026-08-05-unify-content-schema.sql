-- ============================================================================
-- 内容模型统一：去掉 type 分类体系与遗留列，内容 = 文本 + 可选媒体文件
-- （贴吧/动态式：仅文本 / 仅图片 / 图文均可，不再区分类型）。
--
-- 变更内容：
--   1) file_path 归一化：历史"原图 + 压缩图"两文件模型 → file_path 即展示文件
--      （有 compressed_path 且与 file_path 不同的行，用 compressed_path 覆盖 file_path）
--   2) 遗留 link 行：把 url 折叠进 content 正文（Markdown 渲染为可点击链接）
--   3) 删除遗留列：type / url / platform / og_title / og_image / compressed_path
--
-- 执行前请先备份数据库。
-- 注意：新代码（contents 实体不含 type 列）依赖本脚本执行完成后再部署——
-- 旧 schema 的 `type` 列 NOT NULL 且无默认值，新代码插入内容时会报错。
-- ============================================================================

-- 1) 图片归一：压缩图即展示文件（无损 WebP 原图 + Tinify 压缩图的重复模型合并）
UPDATE contents
SET file_path = compressed_path
WHERE compressed_path IS NOT NULL
  AND compressed_path <> ''
  AND (file_path IS NULL OR file_path = '' OR file_path <> compressed_path);

-- 2) link 行：链接折叠进正文
UPDATE contents
SET content = CONCAT(
  IF(content IS NULL OR content = '', '', CONCAT(content, '\n\n')),
  url
)
WHERE type = 'link' AND url IS NOT NULL AND url <> '';

-- 3) 删除遗留列
ALTER TABLE contents
  DROP COLUMN type,
  DROP COLUMN url,
  DROP COLUMN platform,
  DROP COLUMN og_title,
  DROP COLUMN og_image,
  DROP COLUMN compressed_path;
