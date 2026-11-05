-- 0096: Günlük okunma geçmişi (hit_daily).
--
-- Neden gerekli: counters_hit yalnızca TOPLAM tutuyor, günlük kırılım yok.
-- Bu yüzden panelde gerçek bir "okunma trendi" grafiği çizilemiyordu;
-- elimizde yalnız yayın sayısı serisi vardı. Pano 2.0'ın en büyük grafiği
-- okunma akışı olduğu için günlük kova tutulmaya başlıyor.
--
-- Geriye dönük veri UYDURULMUYOR: geçmiş günler bilinmiyor. Mevcut toplam
-- tek bir SENTINEL güne yazılır (1990-01-01). Bu tarih bilinçli olarak
-- trend pencerelerinin çok dışında: "bu okunmalar bir zamanlar oldu, hangi
-- gün olduğu bilinmiyor" demek. Bugünün tarihine yazmak, panelde tek günde
-- binlerce okunma varmış gibi yanlış bir zirve üretirdi.
CREATE TABLE IF NOT EXISTS hit_daily (
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    day        DATE   NOT NULL,
    hits       BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (content_id, day)
);

CREATE INDEX IF NOT EXISTS ix_hit_daily_day ON hit_daily (day DESC);

INSERT INTO hit_daily (content_id, day, hits)
SELECT content_id, DATE '1990-01-01', total
FROM counters_hit
WHERE total > 0
ON CONFLICT (content_id, day) DO NOTHING;
