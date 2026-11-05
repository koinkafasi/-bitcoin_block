-- 0097 down: taban satırları yeniden içeriğin güncelleme gününe yazılır.
-- (0096'nın ilk sürümündeki davranış — yalnız geri alma kolaylığı için.)
DELETE FROM hit_daily WHERE day = DATE '1990-01-01';

INSERT INTO hit_daily (content_id, day, hits)
SELECT c.content_id, ct.updated_at::date, c.total
FROM counters_hit c
JOIN contents ct ON ct.id = c.content_id
WHERE c.total > 0
ON CONFLICT (content_id, day) DO NOTHING;
