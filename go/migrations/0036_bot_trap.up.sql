-- 0036_bot_trap: Bot tuzağı (honeypot) isabetleri (spec 06 K-I)
CREATE TABLE bot_trap_hits (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ip         TEXT,
    user_agent TEXT,
    path       TEXT NOT NULL,
    referer    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_bottrap_created ON bot_trap_hits (created_at DESC);