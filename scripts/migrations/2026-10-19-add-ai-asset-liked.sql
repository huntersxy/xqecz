
ALTER TABLE ai_assets
  ADD COLUMN liked TINYINT(1) NOT NULL DEFAULT 0;

ALTER TABLE ai_assets
  ADD INDEX idx_ai_assets_user_liked (user_id, liked);
