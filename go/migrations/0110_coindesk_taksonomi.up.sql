-- 0110: CoinDesk taksonomisi (bitcosar)
--
-- Amaç: theblock'un Türkçe haber kategorilerini kripto portalına çevirmek.
-- Ölçüm kaynağı: docs/COINDESK-SPEC.md §4 (bölümler + URL desenleri).
--
-- DAVRANIŞ:
--   · Eski kategoriler SİLİNMEZ; `status='arsiv'`, `in_header=false`,
--     `in_menu=false` yapılır. İçerikleri korunur (geri dönüş mümkün).
--   · CoinDesk kategorileri eklenir (yoksa) ve menüye alınır.
--   · Kök kategori = kripto (tüm CoinDesk bölümlerinin üstü).

-- ---- 1) Eski taksonomiyi kapat -------------------------------------------
UPDATE categories
SET status = 'arsiv', in_header = FALSE, in_menu = FALSE, in_footer = FALSE,
    updated_at = now()
WHERE status <> 'arsiv';

-- ---- 2) CoinDesk bölümleri ----------------------------------------------
INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
SELECT 1, v.name, v.slug, NULL, v.hdr, TRUE, v.ftr, 'yayinda', v.sort, v.icon
FROM (VALUES
    ('Markets',           'markets',           TRUE,  TRUE,  10, 'chart-line'),
    ('Finance',           'business',          TRUE,  TRUE,  20, 'building-bank'),
    ('Tech',              'tech',              TRUE,  TRUE,  30, 'cpu'),
    ('Policy',            'policy',            TRUE,  TRUE,  40, 'gavel'),
    ('Focus',             'focus',             TRUE,  TRUE,  50, 'target'),
    ('Opinion',           'opinion',           TRUE,  TRUE,  60, 'message-2'),
    ('News Analysis',     'news-analysis',     FALSE, TRUE,  70, 'microscope'),
    ('Press Releases',    'press-release',     FALSE, TRUE,  80, 'news'),
    ('Sponsored',         'sponsored-content', FALSE, TRUE,  90, 'ad'),
    ('Videos',            'videos',            TRUE,  TRUE, 100, 'video'),
    ('Podcasts',          'podcasts',          TRUE,  TRUE, 110, 'microphone'),
    ('Research',          'research',          TRUE,  TRUE, 120, 'file-analytics'),
    ('Prices',            'price',             TRUE,  TRUE, 130, 'currency-bitcoin'),
    ('Indices',           'index',             FALSE, TRUE, 140, 'chart-dots'),
    ('Latest News',       'latest-crypto-news',TRUE,  TRUE,   5, 'bolt')
) AS v(name, slug, hdr, ftr, sort, icon)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    in_header = EXCLUDED.in_header,
    in_menu = EXCLUDED.in_menu,
    in_footer = EXCLUDED.in_footer,
    status = 'yayinda',
    sort_order = EXCLUDED.sort_order,
    icon = EXCLUDED.icon,
    updated_at = now();

-- ---- 3) Üst kategori (kripto) -------------------------------------------
INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
SELECT 1, 'Kripto', 'kripto', NULL, FALSE, FALSE, TRUE, 'yayinda', 1, 'currency-bitcoin'
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE slug = 'kripto');

-- ---- 4) Kripto haber kaynaklarının varsayılan kategorisi ----------------
UPDATE categories SET in_header = TRUE WHERE slug = 'markets';

-- ---- 5) İçerik tipleri (CoinDesk CMS modeli) ----------------------------
INSERT INTO content_types (key, label, sort_order) VALUES
    ('haber',          'Haber',            10),
    ('analiz',         'Analiz',           20),
    ('gorus',          'Görüş',            30),
    ('canli',          'Canlı blog',       40),
    ('basin-bulteni',  'Basın bülteni',    50),
    ('sponsorlu',      'Sponsorlu',        60),
    ('video',          'Video',            70),
    ('podcast',        'Podcast',          80),
    ('arastirma',      'Araştırma',        90),
    ('fiyat',          'Fiyat sayfası',   100),
    ('endeks',         'Endeks',          110)
ON CONFLICT (key) DO UPDATE SET label = EXCLUDED.label, sort_order = EXCLUDED.sort_order;

-- ---- 6) Site kimliği: bitcosar -----------------------------------------
-- theblock şeması site_id = 1 varsayıyor; yeni kurulumda 1 numaralı site
-- bitcosar'dır. Böylece tüm sorgular değişmeden çalışır.
UPDATE sites
SET name = 'bitcosar', domain = 'bitcosar.com', brand = 'bitcosar',
    default_locale = 'tr', updated_at = now()
WHERE id = 1;
