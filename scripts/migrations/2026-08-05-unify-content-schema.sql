
UPDATE contents
SET file_path = compressed_path
WHERE compressed_path IS NOT NULL
  AND compressed_path <> ''
  AND (file_path IS NULL OR file_path = '' OR file_path <> compressed_path);

UPDATE contents
SET content = CONCAT(
  IF(content IS NULL OR content = '', '', CONCAT(content, '\n\n')),
  url
)
WHERE type = 'link' AND url IS NOT NULL AND url <> '';

ALTER TABLE contents
  DROP COLUMN type,
  DROP COLUMN url,
  DROP COLUMN platform,
  DROP COLUMN og_title,
  DROP COLUMN og_image,
  DROP COLUMN compressed_path;
