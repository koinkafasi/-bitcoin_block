-- 0072 geri alımı
DROP INDEX IF EXISTS idx_ai_source_items_category_check;
ALTER TABLE ai_source_items DROP COLUMN IF EXISTS category_checked_at;
ALTER TABLE ai_source_items DROP COLUMN IF EXISTS section;
