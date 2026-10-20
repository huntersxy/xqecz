ALTER TABLE contents ADD COLUMN ai_enhanced TINYINT(1) NOT NULL DEFAULT 0 AFTER audit_status;
