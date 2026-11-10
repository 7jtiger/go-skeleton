-- Apply to the story database before deploying the gateway.
-- Stop old gateway instances: their existence-only check bypasses this status.
-- 0 = unverified/pending, 1 = completed entitlement, 2 = revoked/refunded.
-- Existing records remain unverified, including records with qt_payed > 0.
-- Never backfill completion from record existence or amount alone.
--
-- No completion writer is enabled until checkout verification is implemented.
-- Completion must bind a verified, idempotent server-side checkout to buyer
-- and story, with the final price/discount determined by the server. For a
-- positive final price, verify settled payment for that amount; for a zero
-- final price, verify the server-authorized free/discount policy instead.
-- Persist completion atomically with checkout fulfillment; revoke entitlement
-- on refund/reversal. Client input must never set purchase_status directly.
ALTER TABLE str_pcs
    ADD COLUMN purchase_status TINYINT NOT NULL DEFAULT 0
    COMMENT '0:pending, 1:completed, 2:revoked';
