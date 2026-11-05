DROP INDEX IF EXISTS ix_media_site;
DROP INDEX IF EXISTS ix_categories_site;
DROP INDEX IF EXISTS ix_contents_site;
ALTER TABLE media_files DROP COLUMN IF EXISTS site_id;
ALTER TABLE categories  DROP COLUMN IF EXISTS site_id;
ALTER TABLE contents    DROP COLUMN IF EXISTS site_id;
DROP TABLE IF EXISTS sites;
