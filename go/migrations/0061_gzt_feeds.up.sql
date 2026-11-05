-- 0061: GZT RSS kaynaklarını AI kaynak merkezine ekle
INSERT INTO feeds (site_id, name, url, category_id, aktif, sort_order)
SELECT 1, v.name, v.url, c.id, TRUE, v.sort
FROM (VALUES
    ('GZT Gündem',   'https://www.gzt.com/rss?category=gundem',   'gundem',        1),
    ('GZT Ekonomi',  'https://www.gzt.com/rss?category=ekonomi',  'ekonomi',       2),
    ('GZT Spor',     'https://www.gzt.com/rss?category=spor',     'spor',          3),
    ('GZT Dünya',    'https://www.gzt.com/rss?category=dunya',    'dunya',         4),
    ('GZT Teknoloji','https://www.gzt.com/rss?category=teknoloji','teknoloji',     5),
    ('GZT Hayat',    'https://www.gzt.com/rss?category=hayat',    'yasam',         6),
    ('GZT Kültür',   'https://www.gzt.com/rss?category=kultur',   'egitim-kultur', 7),
    ('GZT Sinema',   'https://www.gzt.com/rss?category=sinema',   'sinema',        8),
    ('GZT Tarih',    'https://www.gzt.com/rss?category=tarih',    'tarih',         9),
    ('GZT Seyahat',  'https://www.gzt.com/rss?category=seyahat',  'turizm',       10),
    ('GZT Eğlence',  'https://www.gzt.com/rss?category=eglence',  'yasam',        11),
    ('GZT Oyun',     'https://www.gzt.com/rss?category=oyun',     'oyun',         12),
    ('GZT Mimari',   'https://www.gzt.com/rss?category=mimari',   'mimari',       13)
) AS v(name, url, cat_slug, sort)
JOIN categories c ON c.slug = v.cat_slug
WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.url = v.url);
