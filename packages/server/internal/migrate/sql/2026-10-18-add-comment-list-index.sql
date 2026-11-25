
ALTER TABLE comments
  ADD INDEX idx_comments_content_parent_visible_v2
    (content_id, parent_id, is_banned, created_at, id);
