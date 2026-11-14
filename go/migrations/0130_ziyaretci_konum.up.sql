-- 0130: Ziyaretçi konumu (visitor_locations).
--
-- NEDEN: panelde "ziyaretçi nereden geliyor" sorusunun cevabı yoktu.
-- Elimizde yalnız içerik başına okunma sayısı (hit_daily) vardı; coğrafya
-- hiç tutulmuyordu. Yönetici artık ülke/şehir dağılımını ve (izin
-- verildiyse) gerçek koordinatı görebilecek.
--
-- İKİ KAYNAK, İKİ DOĞRULUK SEVİYESİ:
--   kaynak='ip'        → Cloudflare'in CF-IPCountry başlığı. İZİN SORMAZ,
--                        her ziyaretçi için çalışır, ama yalnız ÜLKE verir
--                        (şehir/koordinat null). Kaba ama bedava ve hızlı.
--   kaynak='tarayici'  → navigator.geolocation. Kullanıcı AÇIKÇA izin verir;
--                        enlem/boylam/doğruluk gelir. "Gerçek konum" budur.
--
-- GİZLİLİK KARARLARI (bilinçli, koda gömülü):
--   · visitor_id rastgele bir UUID'dir; kişiyi değil TARAYICIYI tanır.
--     Ad, e-posta, oturum kimliği ile BAĞLANMAZ.
--   · Koordinat 4 ondalığa yuvarlanır (~11 m). Sokak kapısı değil, "hangi
--     mahalle" doğruluğu. Yönetici için yeterli, kötüye kullanım için dar.
--   · IP saklanır (kötüye kullanım/tekrar tespiti için) ama panelde
--     MASKELENEREK gösterilir (bkz. ziyaretci.go maskeliIP).
--   · IP, ülke ve koordinat asla üçüncü tarafa gönderilmez: ülke zaten
--     Cloudflare'in bizim isteğimize eklediği başlıktan geliyor, tersine
--     bir coğrafi API çağrısı YOK.
--
-- ZAMAN: ilk_görülme/son_görülme tutulur; aynı ziyaretçi tekrar gelince
-- satır GÜNCELLENİR (UNIQUE visitor_id), tablo ziyaret sayısıyla şişmez.

CREATE TABLE IF NOT EXISTS visitor_locations (
    visitor_id   UUID        PRIMARY KEY,
    ip           INET,
    ulke         TEXT,             -- ISO 3166-1 alpha-2 (CF-IPCountry)
    sehir        TEXT,             -- yalnız tarayıcı/ileride GeoIP ile
    enlem        DOUBLE PRECISION,
    boylam       DOUBLE PRECISION,
    dogruluk_m   INTEGER,          -- metre; tarayıcının bildirdiği doğruluk
    kaynak       TEXT NOT NULL DEFAULT 'ip',
    izin         BOOLEAN NOT NULL DEFAULT FALSE, -- tarayıcı izni verildi mi
    user_agent   TEXT,
    ilk_görülme  TIMESTAMPTZ NOT NULL DEFAULT now(),
    son_görülme  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ülke dağılımı panelin en sık sorgusu: son 30 gün + ülke kırılımı.
CREATE INDEX IF NOT EXISTS ix_visitor_konum_ulke
    ON visitor_locations (ulke, son_görülme DESC);

-- "Son gelenler" listesi son_görülme DESC ile okunur.
CREATE INDEX IF NOT EXISTS ix_visitor_konum_son
    ON visitor_locations (son_görülme DESC);

-- Koordinatı olanlar (gerçek konum) ayrı listelenir.
CREATE INDEX IF NOT EXISTS ix_visitor_konum_koord
    ON visitor_locations (son_görülme DESC)
    WHERE enlem IS NOT NULL AND boylam IS NOT NULL;

COMMENT ON TABLE visitor_locations IS
    'Ziyaretçi coğrafyası: ülke (CF-IPCountry) + izinli tarayıcı koordinatı. Kişisel kimlik yok.';
COMMENT ON COLUMN visitor_locations.visitor_id IS
    'Tarayıcı çerezi (tb_vid). Kişi değil tarayıcı; oturum/kullanıcı ile bağlanmaz.';
COMMENT ON COLUMN visitor_locations.enlem IS
    '4 ondalık (~11 m) — bilinçli olarak kaba; kapı numarası değil, mahalle.';
