DROP INDEX IF EXISTS contents_auto_published_idx;
ALTER TABLE contents DROP COLUMN IF EXISTS auto_published_at;
