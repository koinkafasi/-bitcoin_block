-- 0035_kantan_fields: Önyüz/alan spec (07) — kategori label + içerik alanları + kaydırma metriği
ALTER TABLE categories ADD COLUMN IF NOT EXISTS label TEXT;

ALTER TABLE contents ADD COLUMN IF NOT EXISTS image_credit TEXT;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS social_summary TEXT;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS ai_summary TEXT;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS audio_url TEXT;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS audio_seconds NUMERIC(6,2);
ALTER TABLE contents ADD COLUMN IF NOT EXISTS audio_status TEXT NOT NULL DEFAULT 'yok';

CREATE TABLE IF NOT EXISTS content_metrics (
    content_id   BIGINT PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    scroll_count BIGINT NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);