-- 0089: feed kaynak hiyerarşisi — agregatör işareti.
--
-- Sorun (B3): MSN Türkiye 10.764 taslakla en büyük "kaynak"tı ve bir
-- agregatördür: kendi haberi yoktur, başka yayıncıların haberlerini toplar.
-- Bot onu sıradan bir kaynak gibi çektiği için aynı haber önce MSN'den
-- "orijinal" gibi taslağa düşüyor, sonra asıl kaynaktan gelen kopya mükerrer
-- sayılıyordu — yani hiyerarşi tersine çalışıyordu.
--
-- Bu kolon bot sırasını ve mükerrer eşiğini belirler:
--   * agregatörler en SON çekilir (orijinal önce kaydedilsin),
--   * agregatör öğelerinde benzerlik eşiği daha düşüktür (kolay mükerrer).
ALTER TABLE feeds
    ADD COLUMN IF NOT EXISTS aggregator BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN feeds.aggregator IS
    'Feed başka yayıncıların haberlerini toplayan bir agregatör mü (MSN, Google News)?';

-- Bilinen agregatörleri işaretle: adı agregatör olanlar ve bingnews türü.
UPDATE feeds SET aggregator = true
WHERE kind = 'bingnews'
   OR name ILIKE '%msn%'
   OR name ILIKE '%google news%'
   OR name ILIKE '%news.google%';

CREATE INDEX IF NOT EXISTS ix_feeds_aggregator ON feeds (aggregator) WHERE aggregator;
