-- 0089 geri alınır: kaynak hiyerarşisi işareti kaldırılır.
DROP INDEX IF EXISTS ix_feeds_aggregator;
ALTER TABLE feeds DROP COLUMN IF EXISTS aggregator;
