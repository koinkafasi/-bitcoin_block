-- Yazar / editör profilleri ve içerik editörü (Faz 51)
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS slug TEXT,
    ADD COLUMN IF NOT EXISTS title TEXT,
    ADD COLUMN IF NOT EXISTS bio TEXT,
    ADD COLUMN IF NOT EXISTS avatar_media_id BIGINT REFERENCES media_files(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS socials JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE UNIQUE INDEX IF NOT EXISTS ux_users_slug ON users (slug) WHERE slug IS NOT NULL;

ALTER TABLE contents
    ADD COLUMN IF NOT EXISTS editor_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS ix_contents_editor ON contents (editor_id) WHERE editor_id IS NOT NULL;

-- Mevcut kullanıcılar için okunabilir slug üret (Türkçe karakterler sadeleşir).
UPDATE users
SET slug = trim(both '-' from regexp_replace(
        lower(translate(name, 'çğıöşüÇĞİÖŞÜâîû', 'cgiosuCGIOSUaiu')),
        '[^a-z0-9]+', '-', 'g'))
WHERE slug IS NULL;

WITH dup AS (
    SELECT id, slug, row_number() OVER (PARTITION BY slug ORDER BY id) AS rn
    FROM users WHERE slug IS NOT NULL
)
UPDATE users u SET slug = dup.slug || '-' || u.id
FROM dup WHERE dup.id = u.id AND dup.rn > 1;
