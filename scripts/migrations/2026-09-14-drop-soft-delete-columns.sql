-- ============================================================================
-- 弃用软删除：物理删除历史软删行，并移除 5 张表的 deleted_at 列
--
-- 背景：删除语义统一为物理删除（代码层已不再引用 deleted_at）。
-- 若直接 DROP COLUMN，原先被 deleted_at 标记的行会因失去过滤条件而"复活"
--（尤其是被禁/已注销用户），因此必须先物理清理。
--
-- 执行前请务必备份数据库；执行顺序：先部署不含 deleted_at 引用的新二进制，
-- 再执行本脚本（否则旧代码的统计 SQL 会因列不存在而报错）。
--
-- 执行前实测的软删行数：users 9、contents 0、comments 3、polls 1、api_keys 3。
-- 依赖核对：软删用户无 contents/comments 引用；软删 poll 有 1 条 poll_votes（一并清理）。
-- ============================================================================

-- 1) 物理删除已软删的行（先子表后主表，避免外键阻塞）
DELETE FROM poll_votes WHERE poll_id IN (SELECT id FROM polls WHERE deleted_at IS NOT NULL);
DELETE FROM polls       WHERE deleted_at IS NOT NULL;
DELETE FROM comments    WHERE deleted_at IS NOT NULL;
DELETE FROM api_keys    WHERE deleted_at IS NOT NULL;
DELETE FROM users       WHERE deleted_at IS NOT NULL;

-- 2) 移除软删除列
ALTER TABLE users      DROP COLUMN deleted_at;
ALTER TABLE contents   DROP COLUMN deleted_at;
ALTER TABLE comments   DROP COLUMN deleted_at;
ALTER TABLE polls      DROP COLUMN deleted_at;
ALTER TABLE api_keys   DROP COLUMN deleted_at;
