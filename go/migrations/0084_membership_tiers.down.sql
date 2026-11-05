-- 0084 geri alınırsa katman/ödeme tabloları ve içerik ağırlığı kaldırılır.
DROP TABLE IF EXISTS subscription_payments;
ALTER TABLE contents DROP COLUMN IF EXISTS min_tier_weight;
ALTER TABLE users DROP COLUMN IF EXISTS membership_billing_cycle;
ALTER TABLE users DROP COLUMN IF EXISTS membership_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS membership_tier_id;
DROP TABLE IF EXISTS membership_tiers;
