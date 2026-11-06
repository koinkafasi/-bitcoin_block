-- 0118 geri alma: bandı 0117'nin başlangıç durumuna (kapalı, boş) döndürür.
UPDATE settings SET value = 'false'::jsonb, updated_at = now()
WHERE section = 'promo' AND key = 'aktif';

UPDATE settings SET value = '""'::jsonb, updated_at = now()
WHERE section = 'promo' AND key IN ('metin', 'baglanti', 'baglanti_metin', 'bitis');
