-- 0059: kart geri bildirimi (buna benzer daha fazla / daha az)
CREATE TABLE IF NOT EXISTS content_reactions (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    visitor    TEXT NOT NULL,
    reaction   SMALLINT NOT NULL CHECK (reaction IN (-1, 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (content_id, visitor)
);
CREATE INDEX IF NOT EXISTS idx_content_reactions_content ON content_reactions(content_id);
