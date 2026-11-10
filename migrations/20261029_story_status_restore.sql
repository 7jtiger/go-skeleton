-- Preserve current meaning when restoring databases that ran the previous mapping.
CREATE TABLE IF NOT EXISTS gateway_schema_migrations (name VARCHAR(128) PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB;
UPDATE story SET stat = CASE stat WHEN 1 THEN 0 WHEN 2 THEN 1 WHEN 3 THEN 2 ELSE stat END WHERE EXISTS (SELECT 1 FROM gateway_schema_migrations WHERE name='20261028_story_status');
INSERT INTO gateway_schema_migrations (name) VALUES ('20261029_story_status_restore');
ALTER TABLE story MODIFY COLUMN stat TINYINT DEFAULT 1 COMMENT 'stat : 0=PUB(public 전체공개), 1=FLW(follower only), 2=PAY(paid only), 3=PRV(Private 비공개), 4=DEL(Deleted) 5=RSV(reserved)';
