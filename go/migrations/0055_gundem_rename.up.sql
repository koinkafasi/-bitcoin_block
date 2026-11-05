-- 0055: Güncel -> Gündem (GZT düzeni)
--
-- Görünen ad "Gündem" olur; URL de /kategori/gundem olarak sadeleşir.
-- Eski /kategori/guncel adresi uygulama içinde 301 ile yönlendirilir.
-- GZT Gündem menüsündeki benzersiz alt kategoriler eklenir.

UPDATE categories SET name = 'Gündem' WHERE slug = 'guncel';

UPDATE categories
SET slug = 'gundem'
WHERE slug = 'guncel'
  AND NOT EXISTS (SELECT 1 FROM categories WHERE slug = 'gundem');

-- Menü/footer/sekme ayarlarındaki eski slug referanslarını güncelle.
UPDATE settings
SET value = replace(value::text, '"guncel"', '"gundem"')::jsonb
WHERE value::text LIKE '%"guncel"%';

INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
FROM categories p
CROSS JOIN (VALUES
    ('Afet & Kaza', 'afet-kaza', 13, 'alert-triangle'),
    ('Analiz',      'analiz',    14, 'chart-line'),
    ('Güvenlik',    'guvenlik',  15, 'shield'),
    ('Özgün',       'ozgun',     16, 'star'),
    ('Toplum',      'toplum',    17, 'users'),
    ('Ramazan',     'ramazan',   18, 'moon')
) AS v(name, slug, sort, icon)
WHERE p.slug = 'gundem'
ON CONFLICT (slug) DO NOTHING;
