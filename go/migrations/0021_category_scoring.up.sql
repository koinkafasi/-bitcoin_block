-- theblock — Kantan referansı: kategori AI-skor kapıları + alaka puanı + trends (0021)
-- spec 06 K-A/K-B (01 §L1, §L3)
ALTER TABLE categories
    ADD COLUMN min_score           INT  NOT NULL DEFAULT 0,   -- yayın AI alaka eşiği (0 = kapı yok)
    ADD COLUMN homepage_min_score  INT  NOT NULL DEFAULT 0,   -- anasayfa eşiği (0 = kapı yok)
    ADD COLUMN icon                TEXT NOT NULL DEFAULT 'news',
    ADD COLUMN image_service       TEXT,                       -- görsel yoksa otomatik görsel kaynağı
    ADD COLUMN name_en             TEXT,
    ADD COLUMN name_es             TEXT;

ALTER TABLE contents
    ADD COLUMN ai_score INT NOT NULL DEFAULT 0;               -- AI alaka puanı (0-100)

CREATE TABLE trends (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    tag         TEXT NOT NULL,
    post_count  INT  NOT NULL DEFAULT 0,
    trend_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    lang        TEXT NOT NULL DEFAULT 'tr',
    source      TEXT NOT NULL DEFAULT 'otomatik',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, category_id, tag, trend_date, lang)
);
CREATE INDEX ix_trends_site_date ON trends (site_id, trend_date DESC);
CREATE INDEX ix_trends_tag ON trends (tag);