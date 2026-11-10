-- Apply once after checking existing indexes and the purchase-status migration.
-- These support the page-wide comment and completed-purchase lookups.
CREATE INDEX idx_str_cmt_story_stat_time ON str_cmt (str_idx, stat, at_create, idx);
CREATE INDEX idx_str_pcs_buyer_status_story ON str_pcs (pcs_uid, purchase_status, str_idx);
