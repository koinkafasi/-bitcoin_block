-- theblock — P5-A yayın ağı: siteler + site_id izolasyonu (0012)
CREATE TABLE sites (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name           TEXT NOT NULL,
    domain         TEXT NOT NULL UNIQUE,
    brand          TEXT,
    default_locale TEXT NOT NULL DEFAULT 'tr',
    status         TEXT NOT NULL DEFAULT 'aktif',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO sites (name, domain, brand, default_locale) VALUES ('theblock.com.tr', 'theblock.com.tr', 'theblock', 'tr');

ALTER TABLE contents    ADD COLUMN site_id BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id);
ALTER TABLE categories  ADD COLUMN site_id BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id);
ALTER TABLE media_files ADD COLUMN site_id BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id);

CREATE INDEX ix_contents_site    ON contents (site_id);
CREATE INDEX ix_categories_site  ON categories (site_id);
CREATE INDEX ix_media_site       ON media_files (site_id);
