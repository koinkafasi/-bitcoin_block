-- 0054: GZT ekonomi alt kategorileri + ana menü hiyerarşisi
--
-- 1) Ekonomi altına GZT'de olup bizde olmayan kategoriler eklenir.
-- 2) Spor altına temel spor dalları eklenir.
-- 3) Yardımcı üst kategoriler ilgili ana kategoriye taşınır; ana menü sadeleşir.

-- --- Ekonomi alt kategorileri ---
INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
FROM categories p
CROSS JOIN (VALUES
    ('Türkiye Ekonomisi', 'turkiye-ekonomisi', 13, 'chart-line'),
    ('Dünya Ekonomisi',   'dunya-ekonomisi',   14, 'globe'),
    ('Kripto Para',       'kripto-para',       15, 'currency-bitcoin'),
    ('Tarım',             'tarim',             16, 'plant'),
    ('Sanayi & Üretim',   'sanayi-uretim',     17, 'building-factory'),
    ('Bankacılık',        'bankacilik',        18, 'building-bank'),
    ('Hisse Senetleri',   'hisse-senetleri',   19, 'chart-candle'),
    ('Yatırım Fonları',   'yatirim-fonlari',   20, 'briefcase'),
    ('Banka Gişe Kurları','banka-gise-kurlari',21, 'cash'),
    ('KAP Haberleri',     'kap-haberleri',     22, 'news')
) AS v(name, slug, sort, icon)
WHERE p.slug = 'ekonomi'
ON CONFLICT (slug) DO NOTHING;

-- --- Spor alt kategorileri (GZT spor menüsü) ---
INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
FROM categories p
CROSS JOIN (VALUES
    ('Futbol',          'futbol',          1, 'ball-football'),
    ('Basketbol',       'basketbol',       2, 'ball-basketball'),
    ('Atletizm',        'atletizm',        3, 'run'),
    ('Dünya Kupası',    'dunya-kupasi',    4, 'trophy'),
    ('Golf',            'golf',            5, 'golf'),
    ('Güreş',           'gures',           6, 'karate'),
    ('Ekstrem Sporlar', 'extreme-sporlar', 7, 'bike'),
    ('Diğer Sporlar',   'diger-sporlar',   8, 'ball')
) AS v(name, slug, sort, icon)
WHERE p.slug = 'spor'
ON CONFLICT (slug) DO NOTHING;

-- --- Yardımcı kategorileri alt kategoriye taşı ---
UPDATE categories c
SET parent_id = p.id, in_header = FALSE, in_menu = TRUE
FROM categories p, (VALUES
    ('asayis', 'guncel'),
    ('ilceler', 'guncel'),
    ('bilim', 'teknoloji'),
    ('oyun', 'teknoloji'),
    ('saglik', 'yasam'),
    ('turizm', 'yasam'),
    ('sanat', 'egitim-kultur')
) AS m(child_slug, parent_slug)
WHERE c.slug = m.child_slug
  AND p.slug = m.parent_slug
  AND p.site_id = c.site_id;
