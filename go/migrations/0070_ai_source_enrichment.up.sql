-- 0070: HTML kaynak öğeleri için kontrollü özet zenginleştirme
ALTER TABLE ai_source_items
    ADD COLUMN IF NOT EXISTS enriched_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS ai_source_items_enrich_idx
    ON ai_source_items (site_id, id)
    WHERE enriched_at IS NULL;
