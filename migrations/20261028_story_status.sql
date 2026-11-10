-- Journal and data update are committed together so retries never remap twice.
CREATE TABLE IF NOT EXISTS gateway_schema_migrations (name VARCHAR(128) PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB;
UPDATE story SET stat = CASE stat WHEN 0 THEN 1 WHEN 1 THEN 2 WHEN 2 THEN 3 WHEN 3 THEN 2 ELSE stat END;
INSERT INTO gateway_schema_migrations (name) VALUES ('20261028_story_status');
ALTER TABLE story MODIFY COLUMN stat TINYINT DEFAULT 1 COMMENT 'stat : 0=ALL(query only), 1=PUB(public), 2=PRV(followers and owner), 3=LMT(purchase required), 4=DEL(internal), 5=RSV(internal)';
