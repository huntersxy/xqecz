-- ============================================================================
-- 1) 新增 contents.compressed_at：TinyPNG 后台压缩的「已处理」标记（NULL = 未压缩）
-- 2) 清理悬空的 thumb_path：file_path 为空（纯文本内容）却残留缩略图引用
--
-- 背景：2026-08-05 内容模型统一后，旧「原图 + 压缩图」双文件模型被合并为单一
-- file_path，但部分历史行的 thumb_path 未被清理，形成指向缩略图的前向引用；
-- 同时压缩任务需要 compressed_at 判断是否已处理，避免重复消耗 API 配额。
--
-- 执行顺序：先执行本脚本，再部署引用 compressed_at 的新二进制
--（Go 侧模型已含该列，缺列会导致查询报 Unknown column）。
-- ============================================================================

ALTER TABLE contents
  ADD COLUMN compressed_at DATETIME(3) NULL DEFAULT NULL COMMENT 'TinyPNG 压缩完成时间，NULL 表示未压缩' AFTER file_size;

UPDATE contents
SET thumb_path = NULL
WHERE (file_path IS NULL OR file_path = '')
  AND thumb_path IS NOT NULL;

-- 校验：以下两条应分别只返回一行，且值符合预期
SELECT COUNT(*) AS added_column
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'contents' AND COLUMN_NAME = 'compressed_at';

SELECT COUNT(*) AS dangling_thumb_path
FROM contents
WHERE (file_path IS NULL OR file_path = '') AND thumb_path IS NOT NULL;
