-- 0086: kategori düzeyinde varsayılan abonelik katmanı.
-- Yeni içerik bu kategoride oluşturulduğunda min_tier_weight buradan gelir;
-- içerik formunda elle değiştirilebilir (0 = kategori kuralı yok).
ALTER TABLE categories ADD COLUMN IF NOT EXISTS min_tier_weight int NOT NULL DEFAULT 0;
ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_min_tier_weight_check;
ALTER TABLE categories ADD CONSTRAINT categories_min_tier_weight_check
    CHECK (min_tier_weight >= 0 AND min_tier_weight <= 5);
