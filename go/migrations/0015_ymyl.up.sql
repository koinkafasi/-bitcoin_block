-- theblock — P5-C Kaynak doğrulama + YMYL (0015)
ALTER TABLE contents
    ADD COLUMN is_ymyl      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN ymyl_uzman   TEXT,
    ADD COLUMN ymyl_durumu  TEXT NOT NULL DEFAULT 'dogrulanmadi';

CREATE TABLE content_sources (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    content_id  BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    title       TEXT,
    body_snippet TEXT,
    accessible  BOOLEAN NOT NULL DEFAULT FALSE,
    status_code INT,
    checked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_sources_content ON content_sources (content_id);
