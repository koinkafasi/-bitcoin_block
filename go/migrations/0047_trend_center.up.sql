-- theblock — Trend Merkezi: dış trend kaynakları + terimler (Faz 101)
-- Google Trends Daily RSS omurgası; kaynak sağlığı ve zaman serisi takibi.

CREATE TABLE trend_sources (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id           BIGINT NOT NULL DEFAULT 1,
    name              TEXT NOT NULL,
    kind              TEXT NOT NULL DEFAULT 'google_trends',
    url               TEXT NOT NULL,
    geo               TEXT NOT NULL DEFAULT 'TR',
    hl                TEXT NOT NULL DEFAULT 'tr-TR',
    interval_seconds  INT NOT NULL DEFAULT 900,
    aktif             BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order        INT NOT NULL DEFAULT 0,
    etag              TEXT NOT NULL DEFAULT '',
    last_modified     TEXT NOT NULL DEFAULT '',
    last_fetch_at     TIMESTAMPTZ,
    last_status       INT,
    last_error        TEXT NOT NULL DEFAULT '',
    consecutive_errors INT NOT NULL DEFAULT 0,
    next_retry_at     TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_trend_sources_site ON trend_sources (site_id, aktif, sort_order);

CREATE TABLE trend_terms (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_id       BIGINT REFERENCES trend_sources(id) ON DELETE SET NULL,
    site_id         BIGINT NOT NULL DEFAULT 1,
    term            TEXT NOT NULL,
    term_norm       TEXT NOT NULL,
    fingerprint     TEXT NOT NULL,
    geo             TEXT NOT NULL DEFAULT 'TR',
    hl              TEXT NOT NULL DEFAULT 'tr-TR',
    kind            TEXT NOT NULL DEFAULT 'top',
    approx_traffic  INT NOT NULL DEFAULT 0,
    picture         TEXT NOT NULL DEFAULT '',
    picture_source  TEXT NOT NULL DEFAULT '',
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    hidden          BOOLEAN NOT NULL DEFAULT FALSE,
    score           INT NOT NULL DEFAULT 0,
    coverage_count  INT NOT NULL DEFAULT 0,
    last_scored_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, fingerprint)
);
CREATE INDEX ix_trend_terms_score ON trend_terms (site_id, hidden, score DESC, last_seen_at DESC);
CREATE INDEX ix_trend_terms_seen ON trend_terms (last_seen_at DESC);
CREATE INDEX ix_trend_terms_norm ON trend_terms (term_norm);

CREATE TABLE trend_points (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    term_id         BIGINT NOT NULL REFERENCES trend_terms(id) ON DELETE CASCADE,
    captured_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    rank            INT NOT NULL DEFAULT 0,
    approx_traffic  INT NOT NULL DEFAULT 0,
    news_count      INT NOT NULL DEFAULT 0
);
CREATE INDEX ix_trend_points_term ON trend_points (term_id, captured_at DESC);

CREATE TABLE trend_news (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    term_id      BIGINT NOT NULL REFERENCES trend_terms(id) ON DELETE CASCADE,
    title        TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL DEFAULT '',
    url_hash     TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT '',
    picture      TEXT NOT NULL DEFAULT '',
    snippet      TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (term_id, url_hash)
);
CREATE INDEX ix_trend_news_term ON trend_news (term_id, created_at DESC);

CREATE TABLE trend_fetch_runs (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_id    BIGINT REFERENCES trend_sources(id) ON DELETE SET NULL,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ,
    status       TEXT NOT NULL DEFAULT 'calisiyor',
    http_code    INT,
    added        INT NOT NULL DEFAULT 0,
    updated      INT NOT NULL DEFAULT 0,
    error        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX ix_trend_fetch_runs_source ON trend_fetch_runs (source_id, started_at DESC);

INSERT INTO trend_sources (site_id, name, kind, url, geo, hl, interval_seconds, aktif, sort_order)
VALUES
    (1, 'Google Trends — Türkiye', 'google_trends',
     'https://trends.google.com/trending/rss?geo=TR', 'TR', 'tr-TR', 900, TRUE, 0),
    (1, 'Google News — Türkiye', 'google_news',
     'https://news.google.com/rss?hl=tr&gl=TR&ceid=TR:tr', 'TR', 'tr-TR', 1800, FALSE, 10)
ON CONFLICT DO NOTHING;

INSERT INTO settings (section, key, value) VALUES
    ('trend', 'agirliklar', '{"trafik":25,"tazelik":15,"haber":15,"alaka":25,"talep":20}'::jsonb),
    ('trend', 'uyari_esik', '70'::jsonb),
    ('trend', 'otomatik_taslak', 'false'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
