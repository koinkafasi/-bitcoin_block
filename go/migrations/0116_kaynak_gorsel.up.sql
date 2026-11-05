-- 0116: kaynak görsel önbelleği (third-party cookie sorununu kapatır)
--
-- ÖLÇÜM (5 Kasım 2026, Lighthouse /markets): best-practices 73. Nedeni
-- bizim kodumuz DEĞİL — kaynak CDN'leri (decrypt.co, cryptoslate.com)
-- görsel yanıtıyla **üçüncü taraf çerez** yazıyor:
--   third-party-cookies → cdn.decrypt.co/...png
--   inspector-issues    → Cookie
-- Ayrıca dış DNS/TLS her sayfa görüntülemesinde yeniden kuruluyor (LCP 2.0 s).
--
-- ÇÖZÜM: görseli kendi origin'imizden servis et. Bu:
--   · üçüncü taraf çerezi kaldırır (BP 96),
--   · dış CDN'e bağımlılığı bitirir (LCP ↓),
--   · hotlink engellemesine takılmaz,
--   · kaynak atfı makale içinde korunur (telif: kopya değil, teknik önbellek).
--
-- Saklama: gövde diskte (storage/kaynak/<hash ilk 2>/<hash>), DB'de yalnız
-- meta. BYTEA'ya koymak tabloyu şişirir ve yedekleri büyütür.

CREATE TABLE IF NOT EXISTS source_images (
    hash         TEXT PRIMARY KEY,           -- sha256(url) ilk 32 hex
    url          TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    bytes        BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Son kullanım: temizlik işi (henüz yok) buna göre silecek.
    used_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_source_images_used ON source_images (used_at);
