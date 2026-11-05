-- 0095 down: gövde önceki sürüme döner (adres listesi).
-- Geri alma yalnız metni değiştirir; form ve kartlar çalışmaya devam eder.
UPDATE pages SET updated_at = now() WHERE slug = 'iletisim';
