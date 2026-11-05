-- 0035_kantan_fields (down)
DROP TABLE IF EXISTS content_metrics;
ALTER TABLE contents DROP COLUMN IF EXISTS audio_status;
ALTER TABLE contents DROP COLUMN IF EXISTS audio_seconds;
ALTER TABLE contents DROP COLUMN IF EXISTS audio_url;
ALTER TABLE contents DROP COLUMN IF EXISTS ai_summary;
ALTER TABLE contents DROP COLUMN IF EXISTS social_summary;
ALTER TABLE contents DROP COLUMN IF EXISTS image_credit;
ALTER TABLE categories DROP COLUMN IF EXISTS label;