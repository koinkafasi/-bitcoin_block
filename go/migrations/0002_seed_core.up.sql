-- theblock P0 çekirdek SEED (0002): 12 içerik türü + varsayılan ayarlar.
-- Not: kullanıcı/anahtar gibi sırlar migration'a GİRMEZ (CLI/seed komutu ile).

INSERT INTO content_types (key, label, sort_order) VALUES
    ('haber',        'Haber',        1),
    ('foto_galeri',  'Foto Galeri',  2),
    ('video',        'Video',        3),
    ('makale',       'Makale',       4),
    ('resmi_ilan',   'Resmi İlan',   5),
    ('e-gazete',     'E-Gazete',     6),
    ('roportaj',     'Röportaj',     7),
    ('anket',        'Anket',        8),
    ('biyografi',    'Biyografi',    9),
    ('firma_rehberi','Firma Rehberi',10),
    ('seri_ilan',    'Seri İlan',    11),
    ('vefat',        'Vefat',        12)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (section, key, value) VALUES
    ('genel',      'site_baslik',            '{"tr":"theblock.com.tr"}'::jsonb),
    ('genel',      'slogan',                 '{"tr":""}'::jsonb),
    ('genel',      'dil',                    '"tr"'::jsonb),
    ('genel',      'bakim_modu',             '{"aktif":false,"mesaj":""}'::jsonb),
    ('gorsel',     'varsayilan_manset_sablonu','{}'::jsonb),
    ('moduller',   'anasayfa_sirasi',        '[]'::jsonb),
    ('yapay_zeka', 'default_provider',       '"omniroute"'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
