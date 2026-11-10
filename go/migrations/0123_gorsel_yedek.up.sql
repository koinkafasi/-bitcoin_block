-- 0123_gorsel_yedek — kaybolan CDN görsellerinin adresini saklar.
--
-- DURUM (10 Kasım 2026): cdn.bitcosar.com üzerindeki WordPress görselleri
-- KAYBOLDU. Sunucu artık her isteğe HTTP 403 dönüyor (Cloudflare bot
-- koruması); dosyaların kopyası da yok. `/kg/` proxy'si bu adresleri
-- indiremediği için görseller sahnede kırık çıkıyordu.
--
-- KARAR (kullanıcı): görsel kaldırılsın; yakında AI görsel motoru
-- eklenecek ve görseller YENİDEN ÜRETİLECEK. Üretim sırasında "bu içeriğin
-- orijinal görseli neydi" bilgisi gerekecek — bu yüzden adres SİLİNMEZ,
-- `image_url_yedek` sütununa taşınır.
--
-- Geri dönüş: aşağıdaki UPDATE'i tersine çevirmek yeterli (kolon durduğu
-- sürece veri kaybı yok).
ALTER TABLE contents ADD COLUMN IF NOT EXISTS image_url_yedek text;

COMMENT ON COLUMN contents.image_url_yedek IS
  'Kaybolan/kaldırılan kapak görselinin orijinal adresi (AI ile yeniden üretim için)';
