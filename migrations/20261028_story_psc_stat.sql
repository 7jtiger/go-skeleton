-- Rename only. Existing pending/revoked rows are not completed by migration.
-- BuyStory completes a pending request only when its authenticated buyer retries.
ALTER TABLE str_pcs CHANGE COLUMN purchase_status psc_stat TINYINT NOT NULL DEFAULT 0
    COMMENT '구매 상태: 0=대기, 1=완료(열람 권한), 2=회수. 현재 BuyStory는 결제 없이 완료 처리';
