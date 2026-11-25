
ALTER TABLE contents
  ADD COLUMN compressed_at DATETIME(3) NULL DEFAULT NULL COMMENT 'TinyPNG 压缩完成时间，NULL 表示未压缩' AFTER file_size;

UPDATE contents
SET thumb_path = NULL
WHERE (file_path IS NULL OR file_path = '')
  AND thumb_path IS NOT NULL;

SELECT COUNT(*) AS added_column
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'contents' AND COLUMN_NAME = 'compressed_at';

SELECT COUNT(*) AS dangling_thumb_path
FROM contents
WHERE (file_path IS NULL OR file_path = '') AND thumb_path IS NOT NULL;
