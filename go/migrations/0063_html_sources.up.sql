-- 0063: RSS'i doğrulanamayan/olmayan haber sitelerini kaynak bankasına katalogla.
-- Bu kayıtlar pasif HTML kaynaklarıdır; Faz 2'de site adaptörü eklenecek.
INSERT INTO feeds (site_id, name, url, category_id, homepage, kind, trust, notes, aktif, sort_order)
SELECT 1, v.name, v.homepage, c.id, v.homepage, 'html', v.trust, v.notes, FALSE, v.sort
FROM (VALUES
    ('DHA',           'https://www.dha.com.tr',          'gundem',    85, 'RSS yok; HTML/API adaptörü', 161),
    ('İHA',           'https://www.iha.com.tr',          'gundem',    85, 'RSS yok; HTML/API adaptörü', 162),
    ('NTV Spor',      'https://www.ntvspor.net',         'spor',      75, 'RSS yok; HTML/API adaptörü', 163),
    ('Fanatik',       'https://www.fanatik.com.tr',      'spor',      70, 'RSS yok; HTML/API adaptörü', 164),
    ('Sporx',         'https://www.sporx.com',           'spor',      70, 'RSS yok; HTML/API adaptörü', 165),
    ('BirGün',        'https://www.birgun.net',          'gundem',    65, 'RSS yok; HTML/API adaptörü', 166),
    ('DW Türkçe',     'https://www.dw.com/tr',           'dunya',     80, 'RSS yok; HTML/API adaptörü', 167),
    ('Technopat',     'https://www.technopat.net',       'teknoloji', 65, 'RSS yok; HTML/API adaptörü', 168),
    ('DonanımHaber',  'https://www.donanimhaber.com',    'teknoloji', 65, 'RSS yok; HTML/API adaptörü', 169),
    ('Bigpara',       'https://bigpara.hurriyet.com.tr', 'ekonomi',   70, 'RSS yok; HTML/API adaptörü', 170)
) AS v(name, homepage, cat_slug, trust, notes, sort)
JOIN categories c ON c.slug = v.cat_slug
WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.url = v.homepage);
