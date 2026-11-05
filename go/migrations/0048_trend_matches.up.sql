-- theblock — Trend Merkezi: kapsama eşleşmeleri + AI brifing + uyarılar (Faz 102-103/106)

CREATE TABLE trend_matches (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    term_id     BIGINT NOT NULL REFERENCES trend_terms(id) ON DELETE CASCADE,
    content_id  BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    match_kind  TEXT NOT NULL DEFAULT 'baslik',
    score       INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (term_id, content_id)
);
CREATE INDEX ix_trend_matches_term ON trend_matches (term_id);
CREATE INDEX ix_trend_matches_content ON trend_matches (content_id);

CREATE TABLE trend_briefs (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    term_id    BIGINT NOT NULL REFERENCES trend_terms(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    provider   TEXT NOT NULL DEFAULT '',
    user_id    BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_trend_briefs_term ON trend_briefs (term_id, created_at DESC);

CREATE TABLE trend_alerts (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    term_id     BIGINT NOT NULL REFERENCES trend_terms(id) ON DELETE CASCADE,
    score       INT NOT NULL DEFAULT 0,
    alert_day   DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    seen_at     TIMESTAMPTZ,
    UNIQUE (term_id, alert_day)
);
CREATE INDEX ix_trend_alerts_unseen ON trend_alerts (seen_at) WHERE seen_at IS NULL;

INSERT INTO settings (section, key, value) VALUES
    ('trend', 'gsc_property', '""'::jsonb),
    ('trend', 'gsc_last_sync', '""'::jsonb),
    ('trend', 'ga4_property', '""'::jsonb),
    ('trend', 'odeme_aktif', 'false'::jsonb),
    ('trend', 'bildirim_aktif', 'true'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
