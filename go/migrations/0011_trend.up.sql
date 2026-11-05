-- theblock — Trend / Ulusal Gündem (RSS agrega) (0011)
CREATE TABLE gundem_items (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source       TEXT NOT NULL,
    title        TEXT NOT NULL,
    url          TEXT NOT NULL,
    url_hash     TEXT NOT NULL UNIQUE,
    body         TEXT,
    published_at TIMESTAMPTZ,
    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_gundem_fetched ON gundem_items (fetched_at DESC);

INSERT INTO settings (section, key, value) VALUES
    ('gundem','kaynaklar','[{"name":"AA Gündem","url":"https://www.aa.com.tr/tr/rss/default?cat=gundem"},{"name":"DHA","url":"https://www.dha.com.tr/rss/kat/gundem.xml"}]'::jsonb),
    ('gundem','dakika','15'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
