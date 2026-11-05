-- 0112 geri alımı: marka metinleri theblock değerlerine döner.
UPDATE settings
SET value = '{"tr":"theblock.com.tr"}'::jsonb, updated_at = now()
WHERE section = 'genel' AND key = 'site_baslik';

DELETE FROM translations
WHERE key = 'site.suffix' AND locale IN ('tr','en');
