DROP INDEX IF EXISTS ix_source_images_media_id;
DROP INDEX IF EXISTS ix_source_images_medyasiz;
ALTER TABLE source_images DROP COLUMN IF EXISTS media_id;
