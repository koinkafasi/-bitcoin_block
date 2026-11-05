-- theblock — Çok dilli altyapı (0004): yerelleştirme kataloğu + içerik çevirisi.
CREATE TABLE locales (
    code        TEXT PRIMARY KEY,
    native_name TEXT NOT NULL,
    rtl         BOOLEAN NOT NULL DEFAULT FALSE,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INT NOT NULL DEFAULT 0
);

-- Arayüz çevirileri (site UI): locale+key -> value
CREATE TABLE translations (
    locale     TEXT NOT NULL REFERENCES locales(code),
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (locale, key)
);

-- İçerik düzeyi: varsayılan dil + çeviri bağlantısı
ALTER TABLE contents
    ADD COLUMN locale        TEXT NOT NULL DEFAULT 'tr' REFERENCES locales(code),
    ADD COLUMN translation_of BIGINT REFERENCES contents(id) ON DELETE SET NULL;

CREATE INDEX ix_contents_locale      ON contents (locale);
CREATE INDEX ix_contents_translation ON contents (translation_of) WHERE translation_of IS NOT NULL;
