
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
