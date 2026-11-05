DROP INDEX IF EXISTS idx_media_files_site_category;
DROP INDEX IF EXISTS idx_media_files_site_created;
ALTER TABLE media_files
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS thumb_path,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS seo_description,
    DROP COLUMN IF EXISTS seo_title,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS title;
