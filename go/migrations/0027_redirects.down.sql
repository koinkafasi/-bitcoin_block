-- 0027_redirects (down)
DROP INDEX IF EXISTS idx_redirects_source_lower;
DROP INDEX IF EXISTS idx_redirects_site;
ALTER TABLE redirects DROP CONSTRAINT IF EXISTS redirects_pkey;
ALTER TABLE redirects ADD CONSTRAINT redirects_pkey PRIMARY KEY (source_path);
ALTER TABLE redirects DROP CONSTRAINT IF EXISTS redirects_site_id_fkey;
ALTER TABLE redirects DROP COLUMN IF EXISTS note;
ALTER TABLE redirects DROP COLUMN IF EXISTS updated_at;
ALTER TABLE redirects DROP COLUMN IF EXISTS hits;
ALTER TABLE redirects DROP COLUMN IF EXISTS site_id;