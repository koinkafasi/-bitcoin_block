DROP INDEX IF EXISTS ix_contents_translation;
DROP INDEX IF EXISTS ix_contents_locale;
ALTER TABLE contents DROP COLUMN IF EXISTS translation_of;
ALTER TABLE contents DROP COLUMN IF EXISTS locale;
DROP TABLE IF EXISTS translations;
DROP TABLE IF EXISTS locales;
