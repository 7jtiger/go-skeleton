-- Apply to the story database (sdb) before deploying the new gateway.
-- Both story and story_view_history must use InnoDB for atomic commits.
-- Policy: one increment per (story_idx, uid) per rolling 24 hours, measured
-- from the last counted view. Duplicate requests do not extend the window.
-- Legacy Redis view markers are not imported: the first view after rollout
-- starts a new window. Stop old gateway instances when switching writers;
-- mixed old/new versions do not share this deduplication ledger.
CREATE TABLE IF NOT EXISTS story_view_history (
    story_idx INT UNSIGNED NOT NULL,
    uid BIGINT UNSIGNED NOT NULL,
    last_counted_at DATETIME(6) NULL,
    PRIMARY KEY (story_idx, uid)
) ENGINE=InnoDB;
