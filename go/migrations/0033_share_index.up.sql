-- 0033_share_index: sosyal paylaşım + indeksleme (IndexNow/GSC) kayıtları (P4-7, P4-6)
CREATE TABLE share_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id    BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    content_id BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    platform   TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'gonderildi',   -- gonderildi | hata
    detail     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_share_created ON share_log (site_id, created_at DESC);

CREATE TABLE indexing_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id    BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    engine     TEXT NOT NULL,                          -- indexnow | google-indexing
    url        TEXT NOT NULL,
    status     TEXT NOT NULL,                          -- ok | hata
    detail     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_indexing_created ON indexing_log (site_id, created_at DESC);