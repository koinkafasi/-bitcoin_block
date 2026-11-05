-- 0064: doğrulanmış ek RSS kaynakları (gazete, ekonomi, teknoloji, bağımsız medya)
INSERT INTO feeds (site_id, name, url, category_id, homepage, kind, trust, notes, aktif, sort_order)
SELECT 1, v.name, v.url, c.id, v.homepage, 'rss', v.trust, v.notes, TRUE, v.sort
FROM (VALUES
    ('Yeni Şafak',          'https://www.yenisafak.com/rss',                    'gundem',    'https://www.yenisafak.com', 70, 'Yeni Şafak genel RSS', 180),
    ('Yeni Şafak Gündem',   'https://www.yenisafak.com/rss?category=gundem',    'gundem',    'https://www.yenisafak.com', 70, 'Yeni Şafak gündem', 181),
    ('Akşam',               'https://www.aksam.com.tr/rss/',                    'gundem',    'https://www.aksam.com.tr', 70, 'Akşam genel RSS', 182),
    ('Türkiye Gazetesi',    'https://www.turkiyegazetesi.com.tr/rss/',          'gundem',    'https://www.turkiyegazetesi.com.tr', 65, 'Türkiye Gazetesi', 183),
    ('Gazete Duvar',        'https://www.gazeteduvar.com.tr/rss',               'gundem',    'https://www.gazeteduvar.com.tr', 70, 'Gazete Duvar', 184),
    ('Medyascope',          'https://medyascope.tv/feed/',                      'gundem',    'https://medyascope.tv', 75, 'Medyascope', 185),
    ('Artı Gerçek',         'https://artigercek.com/rss',                       'gundem',    'https://artigercek.com', 70, 'Artı Gerçek', 186),
    ('Yeniçağ',             'https://www.yenicaggazetesi.com.tr/rss',           'gundem',    'https://www.yenicaggazetesi.com.tr', 65, 'Yeniçağ', 187),
    ('OdaTV',               'https://odatv4.com/rss',                           'gundem',    'https://odatv4.com', 60, 'OdaTV', 188),
    ('Gazete Pencere',      'https://www.gazetepencere.com/feed/',              'gundem',    'https://www.gazetepencere.com', 65, 'Gazete Pencere', 189),
    ('Serbestiyet',         'https://serbestiyet.com/feed/',                    'gundem',    'https://serbestiyet.com', 70, 'Serbestiyet', 190),
    ('10Haber',             'https://10haber.net/feed/',                        'gundem',    'https://10haber.net', 70, '10Haber', 191),
    ('Yetkin Report',       'https://yetkinreport.com/feed/',                   'gundem',    'https://yetkinreport.com', 75, 'Yetkin Report', 192),
    ('Ekonomim',            'https://www.ekonomim.com/rss',                     'ekonomi',   'https://www.ekonomim.com', 75, 'Ekonomim', 193),
    ('Chip',                'https://www.chip.com.tr/rss',                      'teknoloji', 'https://www.chip.com.tr', 70, 'Chip Online', 194),
    ('Tamindir',            'https://www.tamindir.com/feed/',                   'teknoloji', 'https://www.tamindir.com', 65, 'Tamindir', 195),
    ('LOG',                 'https://www.log.com.tr/feed/',                     'teknoloji', 'https://www.log.com.tr', 65, 'LOG Bilişim', 196)
) AS v(name, url, cat_slug, homepage, trust, notes, sort)
JOIN categories c ON c.slug = v.cat_slug
WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.url = v.url);
