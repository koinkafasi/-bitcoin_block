-- 0097: hit_daily taban satırları sentinel güne taşınır.
--
-- 0096 taban satırlarını içeriğin updated_at gününe yazıyordu. Üretimde bu
-- tarihler son bir haftaya düştüğü için panel "bugün 3534 okunma" gösterdi —
-- oysa o sayı aylardır birikmiş TOPLAMDI, bugünün okunması değil. Trend
-- grafiği ilk günlerde yanlış bir zirve çiziyordu.
--
-- Tablo 0096 ile yeni oluşturuldu ve gerçek günlük veri henüz birikmedi;
-- bu yüzden taban satırlarını yeniden yazmak veri kaybı sayılmaz (kaybedilen
-- en fazla 0096 ile bu migration arasındaki birkaç gerçek okumadır).
--
-- Sentinelin anlamı: "bu okunmalar oldu, hangi gün olduğu bilinmiyor".
-- Trend sorguları son N günü seçtiği için bu satırlar grafiğe hiç girmez.
DELETE FROM hit_daily;

INSERT INTO hit_daily (content_id, day, hits)
SELECT content_id, DATE '1990-01-01', total
FROM counters_hit
WHERE total > 0
ON CONFLICT (content_id, day) DO NOTHING;
