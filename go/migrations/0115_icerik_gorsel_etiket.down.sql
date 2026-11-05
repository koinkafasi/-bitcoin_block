-- 0115 geri alımı.
DROP INDEX IF EXISTS ix_contents_image;
ALTER TABLE contents DROP COLUMN IF EXISTS image_url;
