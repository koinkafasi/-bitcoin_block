-- 0042_layout_ab: Sayfa düzeni A/B testi (Faz 7).
-- A varyantı = aktif duzen (page_layouts blocks); B varyantı = b_blocks.
-- weight_b = B'nin goruntulenme yuzdesi (0-100); enabled sesizce A dondurur.
CREATE TABLE IF NOT EXISTS layout_ab (
    site_id    BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    page_key   TEXT   NOT NULL DEFAULT 'home',
    enabled    BOOLEAN NOT NULL DEFAULT false,
    weight_b   INT     NOT NULL DEFAULT 50,
    b_blocks   JSONB   NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (site_id, page_key)
);

-- Gunluk goruntuleme sayacı (A/B secimi yapildiktan sonra).
CREATE TABLE IF NOT EXISTS layout_ab_hits (
    site_id  BIGINT NOT NULL DEFAULT 1,
    page_key TEXT   NOT NULL DEFAULT 'home',
    day      DATE   NOT NULL DEFAULT CURRENT_DATE,
    variant  TEXT   NOT NULL DEFAULT 'a' CHECK (variant IN ('a','b')),
    hits     BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, page_key, day, variant)
);
