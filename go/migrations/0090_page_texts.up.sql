-- 0090: Sayfa Tasarımcısı şablon metinleri.
--
-- Kategori başlığı/açıklaması ve yazı sayfası etiketleri gibi şablon
-- metinleri kodda sabit kalıyordu. Bu tablo, sayfa + anahtar + dil bazında
-- override saklar; kayıt yoksa koddaki varsayılan kullanılır.
--
-- page_key: home | article | kategori | kategori:<slug>
-- text_key: baslik | intro | okuma_suresi | ...
CREATE TABLE IF NOT EXISTS page_texts (
    site_id    BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    page_key   TEXT   NOT NULL,
    text_key   TEXT   NOT NULL,
    locale     TEXT   NOT NULL DEFAULT 'tr',
    value      TEXT   NOT NULL,
    updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (site_id, page_key, text_key, locale)
);

CREATE INDEX IF NOT EXISTS page_texts_page_idx
    ON page_texts (site_id, page_key, locale);

COMMENT ON TABLE page_texts IS
    'Sayfa Tasarımcısı şablon metni override''ları; boş değer varsayılana döner.';
