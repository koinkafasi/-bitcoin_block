-- 0031_bik: Resmi ilanlar (BİK) — kurum/şehir/tarih + belge bağlantısı (spec 02 §217)
CREATE TABLE official_notices (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id      BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    institution  TEXT,
    city         TEXT,
    notice_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    url          TEXT,
    body         TEXT,
    sort         INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, title, notice_date)
);
CREATE INDEX ix_notices_date ON official_notices (site_id, notice_date DESC);
CREATE INDEX ix_notices_city ON official_notices (site_id, city);

CREATE TRIGGER trg_notices_updated BEFORE UPDATE ON official_notices FOR EACH ROW EXECUTE FUNCTION set_updated_at();