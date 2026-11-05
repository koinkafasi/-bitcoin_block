-- 0112: bitcosar marka metinleri
--
-- NEDEN AYRI: 0110 taksonomiyi kuruyor; marka metinleri (site adı, iletişim)
-- ayrı bir karar ve tekrar uygulanabilir olmalı. Değerler JSONB olduğu için
-- jsonb_set ile YALNIZ ilgili anahtarlar güncellenir; diğer alanlar korunur.

-- ---- Site adı (tr + en) ----
UPDATE settings
SET value = '{"tr":"bitcosar","en":"bitcosar"}'::jsonb,
    updated_at = now()
WHERE section = 'genel' AND key = 'site_baslik';

-- ---- İletişim adresleri ----
UPDATE settings
SET value = jsonb_set(
              jsonb_set(
                jsonb_set(value, '{info}',  '"info@bitcosar.com"'),
                '{kvkk}', '"info@bitcosar.com"'),
              '{haber}',  '"haber@bitcosar.com"'),
    updated_at = now()
WHERE section = 'iletisim' AND key = 'ana';

-- ---- Slogan çevirisi (site.suffix) ----
INSERT INTO translations (locale, key, value)
VALUES ('tr', 'site.suffix', 'Kripto Haberleri ve Piyasa Verisi')
ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value;

INSERT INTO translations (locale, key, value)
VALUES ('en', 'site.suffix', 'Crypto News and Market Data')
ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value;
