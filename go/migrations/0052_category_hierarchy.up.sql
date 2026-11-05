-- 0052: kategori hiyerarşisi (üst kategori + alt kategoriler)
--
-- Üst menüde bir kategoriye tıklandığında açılan kısa panelde alt kategoriler
-- listelenir. Alt kategori sayfaları /kategori/<slug> adresinde yayınlanır ve
-- üst kategori sayfası, alt kategorilerin içeriklerini de listeler.

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);

-- Ekonomi alt kategorileri (mynet finans menüsüne benzer kısa panel).
-- Yalnızca ilk kurulumda eklenir; mevcut slug'lar korunur.
INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
FROM categories p
CROSS JOIN (VALUES
    ('Piyasalar',   'piyasalar',   1, 'chart-line'),
    ('Borsa',       'borsa',       2, 'building-bank'),
    ('Döviz',       'doviz',       3, 'currency-dollar'),
    ('Altın',       'altin',       4, 'coin'),
    ('Emtia',       'emtia',       5, 'barrel'),
    ('Faiz',        'faiz',        6, 'percent'),
    ('Enflasyon',   'enflasyon',   7, 'trending-up'),
    ('Finans',      'finans',      8, 'wallet'),
    ('Emlak',       'emlak',       9, 'home'),
    ('Enerji',      'enerji',     10, 'bolt'),
    ('Otomotiv',    'otomotiv',   11, 'car'),
    ('İş Dünyası',  'is-dunyasi', 12, 'briefcase')
) AS v(name, slug, sort, icon)
WHERE p.slug = 'ekonomi'
ON CONFLICT (slug) DO NOTHING;
