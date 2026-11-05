-- 0096 down: günlük okunma tablosu kaldırılır.
-- Toplam sayaç (counters_hit) etkilenmez; yalnız günlük kırılım silinir.
DROP INDEX IF EXISTS ix_hit_daily_day;
DROP TABLE IF EXISTS hit_daily;
