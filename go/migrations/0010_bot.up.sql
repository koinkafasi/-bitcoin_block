-- theblock — Bot / ajans-RSS içerik akışı (0010)
CREATE TABLE feeds (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL DEFAULT 1,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    author_id   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    aktif       BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_feeds_site ON feeds (site_id);

CREATE TABLE crawled_items (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    feed_id    BIGINT REFERENCES feeds(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    url_hash   TEXT NOT NULL UNIQUE,
    status     TEXT NOT NULL DEFAULT 'yeni',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (section, key, value) VALUES
    ('bot','kelimeler','[]'::jsonb),
    ('bot','otomatik','true'::jsonb),
    ('bot','editorde','1'::jsonb),
    ('bot','dakika','5'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
