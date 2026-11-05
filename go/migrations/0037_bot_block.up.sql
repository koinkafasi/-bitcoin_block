-- 0037_bot_block: Bloklu IP katmanı (bot tuzağı → otomatik blok) (spec 06 K-I)
CREATE TABLE blocked_ips (
    ip         TEXT PRIMARY KEY,
    reason     TEXT,
    hits       INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);