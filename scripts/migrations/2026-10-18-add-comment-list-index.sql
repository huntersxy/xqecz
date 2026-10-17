-- 评论列表与回复链热点索引（2026-10-18）
--
-- 旧版 idx_comments_top 以已删除的 deleted_at 为前导列；删除软删除列时该索引
-- 会被 MySQL/TiDB 一并移除。评论列表和回复展开都按内容、父评论和屏蔽状态
-- 过滤，再按创建时间与 id 排序，补回一个不依赖 deleted_at 的复合索引。

ALTER TABLE comments
  ADD INDEX idx_comments_content_parent_visible_v2
    (content_id, parent_id, is_banned, created_at, id);
