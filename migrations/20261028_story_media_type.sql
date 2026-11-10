-- Existing type=2 mixed media becomes mix=3. Text-only remains txt=2.
UPDATE story SET type=3 WHERE type=2
    AND JSON_SEARCH(str_img,'one','img',NULL,'$.*.type') IS NOT NULL
    AND JSON_SEARCH(str_img,'one','vdo',NULL,'$.*.type') IS NOT NULL;

ALTER TABLE story MODIFY COLUMN type TINYINT DEFAULT 1 COMMENT '0=img, 1=vdo, 2=txt, 3=mix, 4=rsv';
