-- 0111: kripto haber kaynak merkezi (feeds seed)
--
-- Kaynaklar 5 Kasım 2026'da BU MAKİNEDEN test edildi; ayrıntı ve
-- çalışmayanların listesi: docs/KAYNAK-MERKEZI.md
--
-- Kurallar:
--   · `trust` 0-100: kaynak ağırlığı (önem skoru ve sıralamada kullanılır).
--   · `aggregator = true`: içerik toplayıcı (tam metin YAYINLANMAZ, atıf
--     verilir). Resmî kurum kaynakları (SEC, Fed, ECB) aggregator = false.
--   · B grubu (TLS parmak izi engeli) `aktif = false` seed edilir; utls
--     adaptörü gelince açılır. Çalışmayan feed'i "aktif" yazmak, her turda
--     boş dönen bir kaynağı sağlıklı göstermek olurdu.

INSERT INTO feeds (site_id, name, url, aktif, sort_order, homepage, kind, trust, notes, aggregator)
SELECT 1, v.name, v.url, v.aktif, v.sort, v.home, 'rss', v.trust, v.not_, TRUE
FROM (VALUES
    -- ---- A grubu: ölçüldü, 200 + öğe döndü ----
    ('CoinDesk',        'https://www.coindesk.com/arc/outboundfeeds/rss/',            TRUE,  10, 'https://www.coindesk.com',  95, 'Arc XP feed; 25 öğe'),
    ('The Block',       'https://www.theblock.co/rss.xml',                            TRUE,  20, 'https://www.theblock.co',   90, '20 öğe; hızlı'),
    ('Decrypt',         'https://decrypt.co/feed',                                    TRUE,  30, 'https://decrypt.co',        85, '36 öğe; görselli'),
    ('CryptoBriefing',  'https://cryptobriefing.com/feed/',                           TRUE,  40, 'https://cryptobriefing.com',75, '30 öğe'),
    ('CryptoSlate',     'https://cryptoslate.com/feed/',                              TRUE,  50, 'https://cryptoslate.com',   70, '10 öğe; sayfalama sınırlı'),

    -- ---- B grubu: TLS parmak izi engeli (aktif=false) ----
    ('Cointelegraph',   'https://cointelegraph.com/rss',                              FALSE, 60, 'https://cointelegraph.com', 90, 'curl/Go TLS reset; utls bekliyor'),
    ('Bitcoin Magazine','https://bitcoinmagazine.com/feed',                           FALSE, 70, 'https://bitcoinmagazine.com',80, 'TLS reset; utls bekliyor'),
    ('BeInCrypto',      'https://beincrypto.com/feed/',                               FALSE, 80, 'https://beincrypto.com',    70, 'TLS reset; utls bekliyor'),
    ('NewsBTC',         'https://www.newsbtc.com/feed/',                              FALSE, 90, 'https://www.newsbtc.com',   60, 'bağlantı yok'),
    ('Bitcoinist',      'https://bitcoinist.com/feed/',                               FALSE,100, 'https://bitcoinist.com',    60, 'bağlantı yok'),
    ('CryptoPotato',    'https://cryptopotato.com/feed/',                             FALSE,110, 'https://cryptopotato.com',  60, 'bağlantı yok'),
    ('AMBCrypto',       'https://ambcrypto.com/feed/',                                FALSE,120, 'https://ambcrypto.com',     55, 'bağlantı yok'),
    ('CoinGape',        'https://coingape.com/feed/',                                 FALSE,130, 'https://coingape.com',      55, 'bağlantı yok'),
    ('CoinJournal',     'https://coinjournal.net/feed/',                              FALSE,140, 'https://coinjournal.net',   60, 'bağlantı yok'),
    ('Cryptonews',      'https://cryptonews.com/news/feed/',                           FALSE,150, 'https://cryptonews.com',    55, 'bağlantı yok')
) AS v(name, url, aktif, sort, home, trust, not_)
WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.url = v.url);

-- ---- Resmî kurum kaynakları: topLAYICI değil, birincil kaynak ----
INSERT INTO feeds (site_id, name, url, aktif, sort_order, homepage, kind, trust, notes, aggregator)
SELECT 1, v.name, v.url, TRUE, v.sort, v.home, 'rss', v.trust, v.not_, FALSE
FROM (VALUES
    ('SEC Basın Duyuruları', 'https://www.sec.gov/news/pressreleases.rss',              200, 'https://www.sec.gov',        95, 'UA e-posta zorunlu'),
    ('Federal Reserve',      'https://www.federalreserve.gov/feeds/press_all.xml',      210, 'https://www.federalreserve.gov', 95, ''),
    ('ECB Basın',            'https://www.ecb.europa.eu/rss/press.html',                220, 'https://www.ecb.europa.eu',  95, ''),
    ('Coinbase Blog',        'https://www.coinbase.com/blog/rss.xml',                   230, 'https://www.coinbase.com',   85, ''),
    ('Tether Duyuruları',    'https://tether.to/en/news/feed/',                         240, 'https://tether.to',          85, '')
) AS v(name, url, sort, home, trust, not_)
WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.url = v.url);

-- ---- Kaynakları Markets kategorisine bağla (varsa) ----
UPDATE feeds
SET category_id = (SELECT id FROM categories WHERE slug = 'markets' LIMIT 1)
WHERE category_id IS NULL;
