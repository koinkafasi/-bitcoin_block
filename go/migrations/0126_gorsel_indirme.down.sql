DROP INDEX IF EXISTS ix_source_images_indirilecek;
ALTER TABLE source_images
  DROP COLUMN IF EXISTS indirme_deneme,
  DROP COLUMN IF EXISTS son_deneme_at,
  DROP COLUMN IF EXISTS son_hata;
