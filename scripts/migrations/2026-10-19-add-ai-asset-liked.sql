-- AI 素材收藏（2026-10-19）
--
-- 工作台「喜欢」栏目按生成结果逐张收藏。素材本身就属于作者，只需在
-- ai_assets 上记录私有标记；不改变公开可见性与发布流程。
--
-- TiDB 不允许在同一条 ALTER 里给刚新增的列建索引（报 1072 column does not
-- exist），因此拆成两条执行。

ALTER TABLE ai_assets
  ADD COLUMN liked TINYINT(1) NOT NULL DEFAULT 0;

ALTER TABLE ai_assets
  ADD INDEX idx_ai_assets_user_liked (user_id, liked);
