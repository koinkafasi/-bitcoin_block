-- theblock — Okuyucu geri bildirimi / içerik güvenliği + AI asistan geçmişi (0024)
-- spec 06 L12 (safety · feedback)
CREATE TABLE content_feedback (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    content_id  BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    site_id     BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL DEFAULT 'sikayet'
                CHECK (kind IN ('sikayet','yanlis','etik','reklam','eksik','diger')),
    detail      TEXT,
    reporter    TEXT,
    status      TEXT NOT NULL DEFAULT 'bekliyor'
                CHECK (status IN ('bekliyor','incelendi','cozuldu','reddedildi')),
    resolution  TEXT,
    handled_by  BIGINT REFERENCES users(id),
    handled_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_feedback_status ON content_feedback (site_id, status, created_at DESC);
CREATE INDEX ix_feedback_content ON content_feedback (content_id);

CREATE TABLE assistant_logs (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    content_id  BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    question    TEXT NOT NULL,
    answer      TEXT,
    provider    TEXT,
    user_id     BIGINT REFERENCES users(id),
    locale      TEXT NOT NULL DEFAULT 'tr',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_assistant_created ON assistant_logs (site_id, created_at DESC);