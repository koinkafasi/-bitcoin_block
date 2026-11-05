-- theblock — Bülten + günlük özet (/ping) + podcast serisi (0022)
-- spec 06 K-E/K-F (01 §L5, §L6)
CREATE TABLE subscribers (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    locale      TEXT NOT NULL DEFAULT 'tr',
    status      TEXT NOT NULL DEFAULT 'onay bekliyor'
                CHECK (status IN ('onay bekliyor','aktif','iptal')),
    token       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at TIMESTAMPTZ,
    UNIQUE (site_id, email, locale)
);
CREATE INDEX ix_subs_status ON subscribers (site_id, status);

CREATE TABLE digests (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id      BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    digest_date  DATE NOT NULL,
    locale       TEXT NOT NULL DEFAULT 'tr',
    kind         TEXT NOT NULL DEFAULT 'gunluk'
                 CHECK (kind IN ('gunluk','haftalik','podcast')),
    title        TEXT NOT NULL,
    summary      TEXT,
    content_ids  BIGINT[] NOT NULL DEFAULT '{}',
    word_count   INT NOT NULL DEFAULT 0,
    audio_path   TEXT,
    status       TEXT NOT NULL DEFAULT 'yayinda',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, digest_date, locale, kind)
);
CREATE INDEX ix_digests_site_date ON digests (site_id, digest_date DESC);