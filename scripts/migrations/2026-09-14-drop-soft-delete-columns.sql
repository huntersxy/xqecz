
DELETE FROM poll_votes WHERE poll_id IN (SELECT id FROM polls WHERE deleted_at IS NOT NULL);
DELETE FROM polls       WHERE deleted_at IS NOT NULL;
DELETE FROM comments    WHERE deleted_at IS NOT NULL;
DELETE FROM api_keys    WHERE deleted_at IS NOT NULL;
DELETE FROM users       WHERE deleted_at IS NOT NULL;

ALTER TABLE users      DROP COLUMN deleted_at;
ALTER TABLE contents   DROP COLUMN deleted_at;
ALTER TABLE comments   DROP COLUMN deleted_at;
ALTER TABLE polls      DROP COLUMN deleted_at;
ALTER TABLE api_keys   DROP COLUMN deleted_at;
