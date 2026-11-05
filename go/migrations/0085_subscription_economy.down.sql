DROP TABLE IF EXISTS promo_codes;
ALTER TABLE subscription_payments DROP COLUMN IF EXISTS note;
ALTER TABLE subscription_payments DROP COLUMN IF EXISTS refunded_at;
ALTER TABLE subscription_payments DROP COLUMN IF EXISTS coupon_code;
ALTER TABLE subscription_payments DROP COLUMN IF EXISTS discount;
ALTER TABLE users DROP COLUMN IF EXISTS membership_cancel_at;
