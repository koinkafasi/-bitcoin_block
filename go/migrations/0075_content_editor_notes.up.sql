-- 0075: editör içi notlar / satır içi yorumlar
CREATE TABLE IF NOT EXISTS content_editor_notes (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    content_id  BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    body        TEXT NOT NULL,
    quote       TEXT NOT NULL DEFAULT '',
    resolved    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_content_editor_notes_content
    ON content_editor_notes (content_id, resolved, created_at);
