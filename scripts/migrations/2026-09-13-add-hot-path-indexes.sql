-- 热点查询索引补齐（2026-09-13）
--
-- 背景：contents / comments / api_keys 此前只有主键索引，以下三类高频查询都会退化为
-- 全表扫描 + filesort（EXPLAIN 显示 type=ALL、Using filesort）：
--   1. 首页/搜索列表：WHERE deleted_at IS NULL AND audit_status IN (...) ORDER BY created_at DESC
--   2. 评论列表：WHERE content_id = ? AND parent_id IS NULL AND is_banned = 0 ORDER BY created_at
--   3. API 密钥鉴权：WHERE key_hash = ? AND is_active = 1（每次带密钥的请求都会执行）
--
-- 说明：
--   - 全部为普通二级索引，可在线添加，不影响现有查询语义；
--   - 列顺序按「等值列 → IN 列 → 排序列」排列，使 ORDER BY 能走索引顺序；
--   - contents.tags 的标签筛选走 JSON_CONTAINS，无法用 B-Tree 索引，数据量大时需另做标签表。

ALTER TABLE contents
  ADD INDEX idx_contents_list (deleted_at, audit_status, created_at),
  ADD INDEX idx_contents_user (user_id, deleted_at, audit_status, created_at);

ALTER TABLE comments
  ADD INDEX idx_comments_top (content_id, parent_id, is_banned, created_at),
  ADD INDEX idx_comments_replies (parent_id, is_banned, created_at);

ALTER TABLE api_keys
  ADD INDEX idx_api_keys_hash (key_hash, is_active);

ALTER TABLE claims
  ADD INDEX idx_claims_status_created (status, created_at);

ALTER TABLE comment_reports
  ADD INDEX idx_reports_handled_created (handled, created_at);

ALTER TABLE polls
  ADD INDEX idx_polls_deleted_created (deleted_at, created_at);

ALTER TABLE poll_votes
  ADD INDEX idx_poll_votes_option (poll_id, option_index);
