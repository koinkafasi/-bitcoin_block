-- 0056: GZT'nin diğer bölümlerindeki alt kategoriler
--
-- Dünya, Teknoloji, Yaşam ve Eğitim-kültür altında GZT'de olup bizde
-- olmayan alt kategoriler eklenir. Tek seviye kuralı korunur.

INSERT INTO categories (site_id, name, slug, parent_id, in_header, in_menu, in_footer, status, sort_order, icon)
SELECT p.site_id, v.name, v.slug, p.id, FALSE, TRUE, TRUE, 'yayinda', v.sort, v.icon
FROM (VALUES
    -- Dünya
    ('Afrika',              'afrika',               'dunya', 1,  'globe'),
    ('Amerika',             'amerika',              'dunya', 2,  'world'),
    ('Asya',                'asya',                 'dunya', 3,  'map'),
    ('Avrupa',              'avrupa',               'dunya', 4,  'flag'),
    ('Orta Doğu',           'ortadogu',             'dunya', 5,  'map-pin'),
    -- Teknoloji
    ('Donanım',             'donanim',              'teknoloji', 1, 'cpu'),
    ('İnternet',            'internet',             'teknoloji', 2, 'world-www'),
    ('Mobil Teknolojiler',  'mobil-teknolojiler',   'teknoloji', 3, 'device-mobile'),
    ('Robotik',             'robotik',              'teknoloji', 4, 'robot'),
    ('Teknoloji Haberleri', 'teknoloji-haberleri',  'teknoloji', 5, 'news'),
    ('Uzay',                'uzay',                 'teknoloji', 6, 'rocket'),
    ('Yazılım',             'yazilim',              'teknoloji', 7, 'code'),
    ('E-spor',              'e-spor',               'teknoloji', 8, 'trophy'),
    ('Mobil Oyunlar',       'mobil-oyunlar',        'teknoloji', 9, 'device-gamepad'),
    ('Oyun Haberleri',      'oyun-haberleri',       'teknoloji', 10, 'news'),
    ('Oyun İncelemeleri',   'oyun-incelemeleri',    'teknoloji', 11, 'search'),
    -- Yaşam
    ('Aktüel',              'aktuel',               'yasam', 1,  'news'),
    ('Gençlik',             'genclik',              'yasam', 2,  'users'),
    ('Kadın',               'kadin',                'yasam', 3,  'woman'),
    ('Kişisel Gelişim',     'kisisel-gelisim',      'yasam', 4,  'growth'),
    ('Doğa & Keşif',        'doga-kesif',           'yasam', 5,  'trees'),
    ('Gezi Rotaları',       'gezi-rotalari',        'yasam', 6,  'route'),
    ('Şehir Rehberi',       'sehir-rehberi',        'yasam', 7,  'building-community'),
    ('Seyahat Rehberi',     'seyahat-rehberi',      'yasam', 8,  'map'),
    ('Yeme-İçme',           'yeme-icme',            'yasam', 9,  'tools-kitchen'),
    ('Hayvanlar Alemi',     'hayvanlar-alemi',      'yasam', 10, 'paw'),
    ('Hobi',                'hobi',                 'yasam', 11, 'palette'),
    ('Mizah',               'mizah',                'yasam', 12, 'mood-smile'),
    -- Eğitim-kültür
    ('Derleme-Seçki',       'derleme-secki',        'egitim-kultur', 1,  'books'),
    ('Düşünce-Fikir',       'dusunce-fikir',        'egitim-kultur', 2,  'bulb'),
    ('Edebiyat',            'edebiyat',             'egitim-kultur', 3,  'book'),
    ('Kitap',               'kitap',                'egitim-kultur', 4,  'book-2'),
    ('Medya-Dijital Kültür','medya-dijital-kultur', 'egitim-kultur', 5,  'device-tv'),
    ('Mekan',               'mekan',                'egitim-kultur', 6,  'building'),
    ('Müzik',               'muzik',                'egitim-kultur', 7,  'music'),
    ('Portreler',           'portreler',            'egitim-kultur', 8,  'user'),
    ('Sinema',              'sinema',               'egitim-kultur', 9,  'movie'),
    ('Tarih',               'tarih',                'egitim-kultur', 10, 'history'),
    ('Mimari',              'mimari',               'egitim-kultur', 11, 'building-arch')
) AS v(name, slug, parent_slug, sort, icon)
JOIN categories p ON p.slug = v.parent_slug
ON CONFLICT (slug) DO NOTHING;
