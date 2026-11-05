-- theblock — Reklam yönetimi: yerleşimler, reklamlar, performans (0026)
-- spec 01 §E (P2-4), 05 reklam modeli
CREATE TABLE ad_slots (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    label       TEXT NOT NULL,
    width       INT NOT NULL DEFAULT 0,
    height      INT NOT NULL DEFAULT 0,
    sticky       BOOLEAN NOT NULL DEFAULT FALSE,
    status      TEXT NOT NULL DEFAULT 'aktif'
                CHECK (status IN ('aktif','pasif')),
    UNIQUE (site_id, key)
);

CREATE TABLE ads (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL
                CHECK (kind IN ('display','banner','native','article','sticky')),
    slot_key    TEXT,
    html        TEXT,
    image_media_id BIGINT REFERENCES media_files(id) ON DELETE SET NULL,
    target_url  TEXT,
    starts_at   TIMESTAMPTZ,
    ends_at     TIMESTAMPTZ,
    status      TEXT NOT NULL DEFAULT 'taslak'
                CHECK (status IN ('taslak','aktif','bitmis','pasif')),
    weight      INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_ads_active ON ads (site_id, status, starts_at, ends_at);

CREATE TABLE ad_stats (
    site_id    BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    ad_id      BIGINT NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
    stat_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    views      INT NOT NULL DEFAULT 0,
    clicks     INT NOT NULL DEFAULT 0,
    revenue_micros BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (ad_id, stat_date)
);
CREATE INDEX ix_adstats_date ON ad_stats (site_id, stat_date DESC);