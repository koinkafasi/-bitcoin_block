-- Faz 51 geri alımı
DROP INDEX IF EXISTS ix_contents_editor;
ALTER TABLE contents DROP COLUMN IF EXISTS editor_id;

DROP INDEX IF EXISTS ux_users_slug;
ALTER TABLE users
    DROP COLUMN IF EXISTS socials,
    DROP COLUMN IF EXISTS avatar_media_id,
    DROP COLUMN IF EXISTS bio,
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS slug;
