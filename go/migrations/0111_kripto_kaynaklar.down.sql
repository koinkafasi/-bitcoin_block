-- 0111 geri alımı: yalnız bu migration'ın eklediği feed'ler silinir.
DELETE FROM feeds
WHERE url IN (
  'https://www.coindesk.com/arc/outboundfeeds/rss/',
  'https://www.theblock.co/rss.xml',
  'https://decrypt.co/feed',
  'https://cryptobriefing.com/feed/',
  'https://cryptoslate.com/feed/',
  'https://cointelegraph.com/rss',
  'https://bitcoinmagazine.com/feed',
  'https://beincrypto.com/feed/',
  'https://www.newsbtc.com/feed/',
  'https://bitcoinist.com/feed/',
  'https://cryptopotato.com/feed/',
  'https://ambcrypto.com/feed/',
  'https://coingape.com/feed/',
  'https://coinjournal.net/feed/',
  'https://cryptonews.com/news/feed/',
  'https://www.sec.gov/news/pressreleases.rss',
  'https://www.federalreserve.gov/feeds/press_all.xml',
  'https://www.ecb.europa.eu/rss/press.html',
  'https://www.coinbase.com/blog/rss.xml',
  'https://tether.to/en/news/feed/'
);
