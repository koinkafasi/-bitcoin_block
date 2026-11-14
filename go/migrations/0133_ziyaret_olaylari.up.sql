-- 0133: Ziyaret olayları — istek başına davranış kaydı.
--
-- NEDEN: visitor_risk ziyaretçi başına TEK satır tutuyor: kim, hangi
-- ağdan, hangi puanda. "NE YAPMAYA ÇALIŞTI" sorusunun cevabı ise orada
-- yok, çünkü tek tek istekler hiç saklanmıyordu. Bir IP'nin 85 puan
-- aldığını görüyorduk ama /wp-login.php mi yokladığını, makale mi
-- kazıdığını, giriş mi denediğini bilmiyorduk. Niyet davranıştan çıkar;
-- davranış da istek dizisidir.
--
-- NEDEN AYRI TABLO: visitor_risk bilerek trafikle doğrusal büyümüyor.
-- Olay kaydı doğası gereği büyür; onu aynı tabloya koymak (JSONB dizi)
-- her istekte bütün satırı yeniden yazdırırdı. Ayrı tablo + saklama
-- süresi (14 gün, repo/ziyaret_olay.go) büyümeyi sınırlar.
--
-- `tur` ve `kaynak` YAZARKEN hesaplanır (httpapi/ziyaret_niyet.go):
-- liste sayfası 120 ziyaretçinin niyetini tek GROUP BY ile okuyabilsin.
-- Kural değişirse eski satırlar eski türüyle kalır; 14 günde kendiliğinden
-- yenilenir. Okurken hesaplamak her panel açılışında tüm yolları yeniden
-- sınıflandırmak demekti.
--
-- GİZLİLİK: sorgu dizgesi ham saklanmaz. Jeton/şifre/kod taşıyabilecek
-- parametrelerin DEĞERİ yazılmadan önce maskelenir (sorguTemizle);
-- parola sıfırlama bağlantısındaki jeton bu tabloya düşmemeli.

CREATE TABLE IF NOT EXISTS visitor_events (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- visitor_risk.anahtar ile aynı: tb_vid çerezi ya da "ip:<adres>".
    -- FK YOK: olay, risk satırından ÖNCE yazılabilir (risk 10 dk'da bir
    -- yazılıyor) ve risk satırı temizlenince olayın silinmesini istemeyiz.
    anahtar     TEXT        NOT NULL,
    ip          INET,
    zaman       TIMESTAMPTZ NOT NULL DEFAULT now(),
    yontem      TEXT        NOT NULL,
    yol         TEXT        NOT NULL,
    sorgu       TEXT,                    -- maskelenmiş, kırpılmış
    durum       INT         NOT NULL,    -- HTTP yanıt kodu
    -- tur: icerik | besleme | arama | api | giris | yonetim | zafiyet |
    --      tuzak | diger
    tur         TEXT        NOT NULL DEFAULT 'diger',
    -- kaynak: yok | ic | arama | sosyal | dis
    kaynak      TEXT        NOT NULL DEFAULT 'yok',
    referer     TEXT,
    ua          TEXT,
    sinif       TEXT,                    -- o andaki karar sınıfı
    engellendi  BOOLEAN     NOT NULL DEFAULT FALSE
);

-- Profil sayfası: "bu ziyaretçinin son istekleri".
CREATE INDEX IF NOT EXISTS ix_visitor_events_anahtar
    ON visitor_events (anahtar, zaman DESC);
-- "Bu IP'den gelen her şey" (kimlik değiştiren bot için pivot).
CREATE INDEX IF NOT EXISTS ix_visitor_events_ip
    ON visitor_events (ip, zaman DESC);
-- Saklama temizliği ve pencere sorguları.
CREATE INDEX IF NOT EXISTS ix_visitor_events_zaman
    ON visitor_events (zaman);

COMMENT ON TABLE visitor_events IS
    'İstek başına davranış kaydı (14 gün). Niyet çıkarımının ham verisi.';
COMMENT ON COLUMN visitor_events.tur IS
    'icerik | besleme | arama | api | giris | yonetim | zafiyet | tuzak | diger';
COMMENT ON COLUMN visitor_events.sorgu IS
    'Hassas parametre değerleri maskelenmiş sorgu dizgesi (en çok 200 karakter).';

-- Araç/tarayıcı kimliği ziyaretçi satırında da dursun: profil sayfası
-- olaylar silindikten sonra da "neyle geldi"yi gösterebilsin.
ALTER TABLE visitor_risk ADD COLUMN IF NOT EXISTS user_agent TEXT;
