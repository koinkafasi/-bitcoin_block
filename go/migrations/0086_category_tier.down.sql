-- 0086 geri alınırsa kategori katman kuralı kaldırılır.
ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_min_tier_weight_check;
ALTER TABLE categories DROP COLUMN IF EXISTS min_tier_weight;
