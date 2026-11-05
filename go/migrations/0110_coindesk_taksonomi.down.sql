-- 0110 geri alımı.
--
-- CoinDesk kategorileri KALDIRILIR (içeriği olanlar arşivlenir), eski
-- kategoriler yeniden menüye alınır. İçerik kaybı yoktur.

UPDATE categories SET status = 'arsiv', in_header = FALSE, in_menu = FALSE
WHERE slug IN ('markets','business','tech','policy','focus','opinion',
               'news-analysis','press-release','sponsored-content',
               'videos','podcasts','research','price','index',
               'latest-crypto-news','kripto');

UPDATE categories
SET status = 'yayinda', in_header = TRUE, in_menu = TRUE, updated_at = now()
WHERE status = 'arsiv';

DELETE FROM content_types
WHERE key IN ('analiz','gorus','canli','basin-bulteni','sponsorlu',
              'video','podcast','arastirma','fiyat','endeks');
