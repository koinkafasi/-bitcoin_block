-- 0072: HTML/sitemap makale bölümü ve kategori düzeltme durumu
ALTER TABLE ai_source_items
    ADD COLUMN IF NOT EXISTS section TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_source_items
    ADD COLUMN IF NOT EXISTS category_checked_at TIMESTAMPTZ;

-- Bölüm eşlemesiyle kategorisi feed varsayılanından farklılaşmış eski
-- öğeler zaten kontrol edilmiş sayılır. Feed varsayılanında kalanlar
-- zenginleştirme sırasında yeniden kontrol edilecek.
UPDATE ai_source_items i
SET category_checked_at = COALESCE(i.enriched_at, i.fetched_at)
FROM feeds f
WHERE f.id = i.feed_id
  AND f.kind IN ('html','sitemap')
  AND i.enriched_at IS NOT NULL
  AND i.category_id IS DISTINCT FROM f.category_id;

CREATE INDEX IF NOT EXISTS idx_ai_source_items_category_check
    ON ai_source_items (site_id, id)
    WHERE category_checked_at IS NULL;
