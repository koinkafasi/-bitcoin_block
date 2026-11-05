-- Medya kütüphanesi: WordPress benzeri metadata, kategori ve önizleme.
ALTER TABLE media_files
    ADD COLUMN IF NOT EXISTS title TEXT,
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS seo_title TEXT,
    ADD COLUMN IF NOT EXISTS seo_description TEXT,
    ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'genel',
    ADD COLUMN IF NOT EXISTS thumb_path TEXT,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE media_files
   SET title = filename
 WHERE title IS NULL OR btrim(title) = '';

CREATE INDEX IF NOT EXISTS idx_media_files_site_created
    ON media_files (site_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_files_site_category
    ON media_files (site_id, category, created_at DESC);
