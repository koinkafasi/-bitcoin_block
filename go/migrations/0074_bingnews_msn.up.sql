-- 0074: MSN Türkiye için Bing News RSS adaptörü
UPDATE feeds
SET kind='bingnews',
    url='https://www.bing.com/news/search?q=site%3Amsn.com%2Ftr-tr&format=rss&setlang=tr-TR&cc=TR',
    homepage='https://www.msn.com/tr-tr',
    aktif=true,
    notes='Bing News RSS üzerinden MSN Türkiye haberleri; gerçek msn.com bağlantısı çıkarılır'
WHERE name='MSN Türkiye';
