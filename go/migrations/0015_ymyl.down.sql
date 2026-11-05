DROP TABLE IF EXISTS content_sources;
ALTER TABLE contents
    DROP COLUMN IF EXISTS ymyl_durumu,
    DROP COLUMN IF EXISTS ymyl_uzman,
    DROP COLUMN IF EXISTS is_ymyl;
