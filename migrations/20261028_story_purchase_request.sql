-- Apply after 20261027_story_purchase_status.sql and before deploying BuyStory.
-- Stop legacy writers using MAX(idx)+1; AUTO_INCREMENT allocates IDs safely
-- for concurrent purchase requests on different stories. No rows are completed.
ALTER TABLE str_pcs MODIFY COLUMN idx INT NOT NULL AUTO_INCREMENT;
CREATE INDEX idx_str_pcs_story_buyer ON str_pcs (str_idx, pcs_uid, idx);
