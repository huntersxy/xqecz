-- contents 增加「发布时是否启用提示词增强」标记。
-- 详情页据此显示「提示词增强」徽标；发布描述仍保存增强前的原始提示词。
-- TiDB/MySQL 通用：普通 TINYINT 列可带 DEFAULT，不涉及 TEXT/BLOB 限制。
ALTER TABLE contents ADD COLUMN ai_enhanced TINYINT(1) NOT NULL DEFAULT 0 AFTER audit_status;
