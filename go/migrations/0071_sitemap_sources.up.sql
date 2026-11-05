-- 0071: Google News sitemap'i olan kaynaklar için sitemap adaptörü
UPDATE feeds
SET kind='sitemap',
    url='https://www.birgun.net/sitemap/haberler',
    homepage='https://www.birgun.net',
    aktif=true,
    notes='Google News sitemap + JSON-LD/meta zenginleştirme'
WHERE name='BirGün';

UPDATE feeds
SET kind='sitemap',
    url='https://www.donanimhaber.com/site-sitemaps/news-sitemap',
    homepage='https://www.donanimhaber.com',
    aktif=true,
    notes='Google News sitemap + JSON-LD/meta zenginleştirme'
WHERE name='DonanımHaber';

UPDATE feeds
SET kind='passive', aktif=false,
    notes='HTML/JavaScript; özel adaptör bekliyor'
WHERE name='MSN Türkiye';
