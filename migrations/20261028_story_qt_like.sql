-- Preserve existing like counts and column attributes.
ALTER TABLE story RENAME COLUMN qt_good TO qt_like;
