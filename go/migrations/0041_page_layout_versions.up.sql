-- 0041_page_layout_versions: Sayfa düzeni sürüm geçmişi (Faz 4).
-- Her kayıt, page_layouts'ın o anki blok dizisinin bir anlık görüntüsü + not/kişi.
-- Geri alma da yeni bir sürüm yazar (geçmişe işlenir, kopyalanmaz).
CREATE TABLE IF NOT EXISTS page_layout_versions (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id    BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    page_key   TEXT   NOT NULL DEFAULT 'home',
    blocks     JSONB  NOT NULL DEFAULT '[]',
    note       TEXT   NOT NULL DEFAULT '',
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_layout_versions_site_page
  ON page_layout_versions (site_id, page_key, id DESC);
