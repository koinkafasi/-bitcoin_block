-- Faz 195 — çekmece/alt bilgi anahtarlarının BOŞ değerlerini doldur.
--
-- ÖLÇÜM (üretim, 2 Kasım 2026): bu 9 anahtarın satırı VAR ama `tr` değeri
-- BOŞ; 1 anahtar hiç yok. `T()` boş değeri "çeviri var" saydığı için
-- çekmecede etiketler kayboluyordu:
--
--   [2] "📰"      → "Akış" etiketi yok
--   [1] "✉"       → "Bülten" yok
--   [2] "💳"      → "Abonelik" yok
--   [2] "☀ / 🕌 / 📈 / ⚽ / 🔮" → hava, namaz, borsa, fikstür, astroloji yok
--   [27] "🏥 footer.nobetci-eczaneler" → ham anahtar basılıyor
--
-- KURAL: yalnız BOŞ değer doldurulur. Panelden yazılmış metin EZİLMEZ.

INSERT INTO translations (locale, key, value) VALUES
  ('tr', 'footer.akis',              'Akış'),
  ('en', 'footer.akis',              'Live Feed'),
  ('tr', 'footer.bulten',            'Bülten'),
  ('en', 'footer.bulten',            'Newsletter'),
  ('tr', 'subscription',             'Abonelik'),
  ('en', 'subscription',             'Subscription'),
  ('tr', 'footer.hava-durumu',       'Hava Durumu'),
  ('en', 'footer.hava-durumu',       'Weather'),
  ('tr', 'footer.namaz-vakitleri',   'Namaz Vakitleri'),
  ('en', 'footer.namaz-vakitleri',   'Prayer Times'),
  ('tr', 'footer.borsa',             'Borsa'),
  ('en', 'footer.borsa',             'Markets'),
  ('tr', 'footer.fikstur',           'Fikstür'),
  ('en', 'footer.fikstur',           'Fixtures'),
  ('tr', 'footer.astroloji',         'Astroloji'),
  ('en', 'footer.astroloji',         'Horoscope'),
  ('tr', 'footer.nobetci-eczaneler', 'Nöbetçi Eczaneler'),
  ('en', 'footer.nobetci-eczaneler', 'Pharmacies on Duty'),
  ('tr', 'footer.gazete-mansetleri', 'Gazete Manşetleri'),
  ('en', 'footer.gazete-mansetleri', 'Newspaper Front Pages')
ON CONFLICT (locale, key) DO UPDATE
   SET value = EXCLUDED.value
 WHERE translations.value = '';

-- Doğrulama: bu anahtarların hiçbiri boş kalmamalı.
SELECT count(*) AS bos_kalan
  FROM translations
 WHERE key IN ('footer.akis','footer.bulten','subscription','footer.hava-durumu',
               'footer.namaz-vakitleri','footer.borsa','footer.fikstur',
               'footer.astroloji','footer.nobetci-eczaneler','footer.gazete-mansetleri')
   AND value = '';
