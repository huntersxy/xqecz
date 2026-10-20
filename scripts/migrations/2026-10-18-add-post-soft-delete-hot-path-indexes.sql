
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
