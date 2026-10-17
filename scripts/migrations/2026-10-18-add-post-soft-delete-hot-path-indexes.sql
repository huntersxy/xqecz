-- 软删除列移除后的热点索引补齐（2026-10-18）
--
-- 2026-09-13 的索引脚本以 deleted_at 作为前导列；2026-09-14 删除该列后，
-- MySQL/TiDB 会同时移除依赖该列的索引。评论、密钥、举报和投票选项索引
-- 不依赖 deleted_at，仍由旧脚本提供，这里补缺失的内容、投票与互动去重索引。
--
-- 覆盖的实际查询：
--   * 公开/管理列表：WHERE audit_status IN (...) ORDER BY created_at
--   * 我的内容：WHERE user_id = ? ORDER BY created_at
--   * 推荐降级与按热度列表：WHERE audit_status = ? ORDER BY view_count, created_at
--   * 投票列表：ORDER BY created_at DESC
--   * 内容点赞/收藏及投票去重：WHERE content_id/poll_id + user_id/visitor_id

ALTER TABLE contents
  ADD INDEX idx_contents_audit_created_v2 (audit_status, created_at),
  ADD INDEX idx_contents_user_created_v2 (user_id, created_at),
  ADD INDEX idx_contents_audit_views_created_v2 (audit_status, view_count, created_at);

ALTER TABLE polls
  ADD INDEX idx_polls_created_v2 (created_at);

ALTER TABLE content_likes
  ADD INDEX idx_content_likes_content_user_v2 (content_id, user_id);

ALTER TABLE content_favorites
  ADD INDEX idx_content_favorites_content_user_v2 (content_id, user_id);

ALTER TABLE poll_votes
  ADD INDEX idx_poll_votes_user_v2 (poll_id, user_id),
  ADD INDEX idx_poll_votes_visitor_v2 (poll_id, visitor_id);
