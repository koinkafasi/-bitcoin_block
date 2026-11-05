-- 0062: kaynak bankası — site adresi, RSS türü, güven ve tarama durumu
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS homepage TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'rss';
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS trust INT NOT NULL DEFAULT 50;
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMPTZ;
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS last_status TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS last_item_count INT NOT NULL DEFAULT 0;

UPDATE feeds SET homepage='https://www.gzt.com', trust=70, notes='GZT kategori RSS'
WHERE url LIKE 'https://www.gzt.com/rss?category=%';

-- Doğrulanmış ulusal/ajans/kategori RSS kaynakları.
INSERT INTO feeds (site_id, name, url, category_id, homepage, kind, trust, notes, aktif, sort_order)
SELECT 1, v.name, v.url, c.id, v.homepage, 'rss', v.trust, v.notes, v.aktif, v.sort
FROM (VALUES
    ('AA Gündem',        'https://www.aa.com.tr/tr/rss/default?cat=gundem',  'gundem',       'https://www.aa.com.tr', 90, 'Anadolu Ajansı', TRUE, 20),
    ('AA Ekonomi',       'https://www.aa.com.tr/tr/rss/default?cat=ekonomi', 'ekonomi',      'https://www.aa.com.tr', 90, 'Anadolu Ajansı', TRUE, 21),
    ('AA Spor',          'https://www.aa.com.tr/tr/rss/default?cat=spor',    'spor',         'https://www.aa.com.tr', 90, 'Anadolu Ajansı', TRUE, 22),
    ('AA Dünya',         'https://www.aa.com.tr/tr/rss/default?cat=dunya',   'dunya',        'https://www.aa.com.tr', 90, 'Anadolu Ajansı', TRUE, 23),
    ('TRT Haber',        'https://www.trthaber.com/sondakika.rss',          'gundem',       'https://www.trthaber.com', 85, 'TRT Haber son dakika', TRUE, 30),
    ('NTV Son Dakika',   'https://www.ntv.com.tr/son-dakika.rss',           'gundem',       'https://www.ntv.com.tr', 80, 'NTV son dakika', TRUE, 40),
    ('NTV Türkiye',      'https://www.ntv.com.tr/turkiye.rss',              'gundem',       'https://www.ntv.com.tr', 80, 'NTV Türkiye', TRUE, 41),
    ('NTV Ekonomi',      'https://www.ntv.com.tr/ekonomi.rss',              'ekonomi',      'https://www.ntv.com.tr', 80, 'NTV ekonomi', TRUE, 42),
    ('NTV Dünya',        'https://www.ntv.com.tr/dunya.rss',                'dunya',        'https://www.ntv.com.tr', 80, 'NTV dünya', TRUE, 43),
    ('NTV Teknoloji',    'https://www.ntv.com.tr/teknoloji.rss',            'teknoloji',    'https://www.ntv.com.tr', 80, 'NTV teknoloji', TRUE, 44),
    ('NTV Yaşam',        'https://www.ntv.com.tr/yasam.rss',                'yasam',        'https://www.ntv.com.tr', 80, 'NTV yaşam', TRUE, 45),
    ('CNN Türk',         'https://www.cnnturk.com/feed/rss/news',           'gundem',       'https://www.cnnturk.com', 78, 'CNN Türk haber', TRUE, 50),
    ('CNN Türkiye',      'https://www.cnnturk.com/feed/rss/turkiye',        'gundem',       'https://www.cnnturk.com', 78, 'CNN Türk Türkiye', TRUE, 51),
    ('CNN Ekonomi',      'https://www.cnnturk.com/feed/rss/ekonomi',        'ekonomi',      'https://www.cnnturk.com', 78, 'CNN Türk ekonomi', TRUE, 52),
    ('CNN Spor',         'https://www.cnnturk.com/feed/rss/spor',           'spor',         'https://www.cnnturk.com', 78, 'CNN Türk spor', TRUE, 53),
    ('CNN Dünya',        'https://www.cnnturk.com/feed/rss/dunya',          'dunya',        'https://www.cnnturk.com', 78, 'CNN Türk dünya', TRUE, 54),
    ('CNN Teknoloji',    'https://www.cnnturk.com/feed/rss/teknoloji',      'teknoloji',    'https://www.cnnturk.com', 78, 'CNN Türk teknoloji', TRUE, 55),
    ('CNN Yaşam',        'https://www.cnnturk.com/feed/rss/yasam',          'yasam',        'https://www.cnnturk.com', 78, 'CNN Türk yaşam', TRUE, 56),
    ('Hürriyet',         'https://www.hurriyet.com.tr/rss/anasayfa',        'gundem',       'https://www.hurriyet.com.tr', 80, 'Hürriyet ana sayfa', TRUE, 60),
    ('Hürriyet Gündem',  'https://www.hurriyet.com.tr/rss/gundem',          'gundem',       'https://www.hurriyet.com.tr', 80, 'Hürriyet gündem', TRUE, 61),
    ('Hürriyet Ekonomi', 'https://www.hurriyet.com.tr/rss/ekonomi',         'ekonomi',      'https://www.hurriyet.com.tr', 80, 'Hürriyet ekonomi', TRUE, 62),
    ('Hürriyet Spor',    'https://www.hurriyet.com.tr/rss/spor',            'spor',         'https://www.hurriyet.com.tr', 80, 'Hürriyet spor', TRUE, 63),
    ('Hürriyet Dünya',   'https://www.hurriyet.com.tr/rss/dunya',           'dunya',        'https://www.hurriyet.com.tr', 80, 'Hürriyet dünya', TRUE, 64),
    ('Hürriyet Teknoloji','https://www.hurriyet.com.tr/rss/teknoloji',      'teknoloji',    'https://www.hurriyet.com.tr', 80, 'Hürriyet teknoloji', TRUE, 65),
    ('Milliyet Son Dakika','https://www.milliyet.com.tr/rss/rssNew/SonDakikaRss.xml','gundem','https://www.milliyet.com.tr',75,'Milliyet son dakika',TRUE,70),
    ('Sabah',            'https://www.sabah.com.tr/rss/anasayfa.xml',       'gundem',       'https://www.sabah.com.tr', 72, 'Sabah ana sayfa', TRUE, 80),
    ('Sabah Gündem',     'https://www.sabah.com.tr/rss/gundem.xml',         'gundem',       'https://www.sabah.com.tr', 72, 'Sabah gündem', TRUE, 81),
    ('Sabah Ekonomi',    'https://www.sabah.com.tr/rss/ekonomi.xml',        'ekonomi',      'https://www.sabah.com.tr', 72, 'Sabah ekonomi', TRUE, 82),
    ('Sabah Spor',       'https://www.sabah.com.tr/rss/spor.xml',           'spor',         'https://www.sabah.com.tr', 72, 'Sabah spor', TRUE, 83),
    ('Sabah Dünya',      'https://www.sabah.com.tr/rss/dunya.xml',          'dunya',        'https://www.sabah.com.tr', 72, 'Sabah dünya', TRUE, 84),
    ('Sabah Yaşam',      'https://www.sabah.com.tr/rss/yasam.xml',          'yasam',        'https://www.sabah.com.tr', 72, 'Sabah yaşam', TRUE, 85),
    ('Habertürk',        'https://www.haberturk.com/rss',                   'gundem',       'https://www.haberturk.com', 75, 'Habertürk genel RSS', TRUE, 90),
    ('Cumhuriyet Son Dakika','https://www.cumhuriyet.com.tr/rss/son_dakika.xml','gundem',   'https://www.cumhuriyet.com.tr', 75, 'Cumhuriyet son dakika', TRUE, 100),
    ('Cumhuriyet Gündem','https://www.cumhuriyet.com.tr/rss/gundem.xml',    'gundem',       'https://www.cumhuriyet.com.tr', 75, 'Cumhuriyet gündem', TRUE, 101),
    ('Cumhuriyet Ekonomi','https://www.cumhuriyet.com.tr/rss/ekonomi.xml',  'ekonomi',      'https://www.cumhuriyet.com.tr', 75, 'Cumhuriyet ekonomi', TRUE, 102),
    ('Cumhuriyet Spor',  'https://www.cumhuriyet.com.tr/rss/spor.xml',      'spor',         'https://www.cumhuriyet.com.tr', 75, 'Cumhuriyet spor', TRUE, 103),
    ('Cumhuriyet Dünya', 'https://www.cumhuriyet.com.tr/rss/dunya.xml',     'dunya',        'https://www.cumhuriyet.com.tr', 75, 'Cumhuriyet dünya', TRUE, 104),
    ('Cumhuriyet Kültür','https://www.cumhuriyet.com.tr/rss/kultur-sanat.xml','egitim-kultur','https://www.cumhuriyet.com.tr', 75, 'Cumhuriyet kültür-sanat', TRUE, 105),
    ('Sözcü Gündem',     'https://www.sozcu.com.tr/feeds-rss-category-gundem','gundem',     'https://www.sozcu.com.tr', 70, 'Sözcü gündem', TRUE, 110),
    ('Sözcü Ekonomi',    'https://www.sozcu.com.tr/feeds-rss-category-ekonomi','ekonomi',   'https://www.sozcu.com.tr', 70, 'Sözcü ekonomi', TRUE, 111),
    ('Sözcü Spor',       'https://www.sozcu.com.tr/feeds-rss-category-spor',  'spor',      'https://www.sozcu.com.tr', 70, 'Sözcü spor', TRUE, 112),
    ('Sözcü Dünya',      'https://www.sozcu.com.tr/feeds-rss-category-dunya', 'dunya',     'https://www.sozcu.com.tr', 70, 'Sözcü dünya', TRUE, 113),
    ('Sözcü Yaşam',      'https://www.sozcu.com.tr/feeds-rss-category-yasam', 'yasam',     'https://www.sozcu.com.tr', 70, 'Sözcü yaşam', TRUE, 114),
    ('BBC Türkçe',       'https://feeds.bbci.co.uk/turkce/rss.xml',          'dunya',     'https://www.bbc.com/turkce', 85, 'BBC Türkçe', TRUE, 120),
    ('Euronews Türkçe',  'https://tr.euronews.com/rss',                      'dunya',     'https://tr.euronews.com', 80, 'Euronews Türkçe', TRUE, 121),
    ('Bloomberg HT',     'https://www.bloomberght.com/rss',                  'ekonomi',   'https://www.bloomberght.com', 80, 'Bloomberg HT', TRUE, 130),
    ('Dünya',            'https://www.dunya.com/rss',                        'ekonomi',   'https://www.dunya.com', 78, 'Dünya gazetesi', TRUE, 131),
    ('Webtekno',         'https://www.webtekno.com/rss.xml',                 'teknoloji', 'https://www.webtekno.com', 70, 'Webtekno', TRUE, 140),
    ('ShiftDelete',      'https://shiftdelete.net/feed',                     'teknoloji', 'https://shiftdelete.net', 70, 'ShiftDelete.Net', TRUE, 141),
    ('Google News Türkiye','https://news.google.com/rss?hl=tr&gl=TR&ceid=TR:tr','gundem',  'https://news.google.com', 60, 'Google News toplayıcı; kaynak çeşitliliği için', TRUE, 150)
) AS v(name, url, cat_slug, homepage, trust, notes, aktif, sort)
JOIN categories c ON c.slug = v.cat_slug
WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.url = v.url);

-- MSN'in herkese açık RSS'i yok; kaynak bankasında HTML/API adaptörü olarak izlenir.
INSERT INTO feeds (site_id, name, url, category_id, homepage, kind, trust, notes, aktif, sort_order)
SELECT 1, 'MSN Türkiye', 'https://www.msn.com/tr-tr', c.id, 'https://www.msn.com/tr-tr', 'html', 60,
       'RSS yok; HTML/API adaptörü Faz 2', FALSE, 160
FROM categories c WHERE c.slug='gundem'
  AND NOT EXISTS (SELECT 1 FROM feeds f WHERE f.url='https://www.msn.com/tr-tr');
