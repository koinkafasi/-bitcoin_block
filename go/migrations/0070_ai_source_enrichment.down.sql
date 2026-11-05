-- 0070 geri alımı
DROP INDEX IF EXISTS ai_source_items_enrich_idx;
ALTER TABLE ai_source_items DROP COLUMN IF EXISTS enriched_at;
