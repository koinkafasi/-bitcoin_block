-- 0029_newspapers: Gazete manşetleri (spec 02 §217, 01 §? gündem) — günlük ön sayfa galerisi
CREATE TABLE newspaper_pages (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id    BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    page_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    image_url  TEXT NOT NULL,
    thumb_url  TEXT,
    sort       INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, slug, page_date)
);
CREATE INDEX ix_newspaper_date ON newspaper_pages (site_id, page_date DESC);

CREATE TRIGGER trg_newspaper_updated BEFORE UPDATE ON newspaper_pages FOR EACH ROW EXECUTE FUNCTION set_updated_at();