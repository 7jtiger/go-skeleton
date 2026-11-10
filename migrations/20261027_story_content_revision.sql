-- Apply to the story database before deploying the revision-aware gateway.
-- Stop old writers during rollout: every writer of cached content must increment
-- content_revision in the same UPDATE. Do not mix old and new writers.
-- Revision 0 is reserved for legacy Redis entries and is always a cache miss.
-- Existing rows start at 1; no Redis flush or timestamp precision change needed.
ALTER TABLE story
    ADD COLUMN content_revision BIGINT UNSIGNED NOT NULL DEFAULT 1;
