-- Apply through dbtool, after the legacy revision migration.
-- Saturation avoids wraparound and permanently bypasses cache at 255.
UPDATE story SET content_revision = LEAST(content_revision, 255);
ALTER TABLE story CHANGE COLUMN content_revision cntt_rvsn TINYINT UNSIGNED NOT NULL DEFAULT 1
    COMMENT '콘텐츠 변경 버전: 본문·상태·미디어 수정 시 증가, 같은 초 변경의 캐시 검증용. 0=미확인, 1~254=버전, 255=포화 시 캐시 사용 안 함';
