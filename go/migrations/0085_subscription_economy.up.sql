-- 0085: abonelik ekonomi katmanı — dönem sonu iptal, iade, kupon, mutabakat.
ALTER TABLE users ADD COLUMN IF NOT EXISTS membership_cancel_at timestamptz;

ALTER TABLE subscription_payments ADD COLUMN IF NOT EXISTS discount numeric(10,2) NOT NULL DEFAULT 0;
ALTER TABLE subscription_payments ADD COLUMN IF NOT EXISTS coupon_code text;
ALTER TABLE subscription_payments ADD COLUMN IF NOT EXISTS refunded_at timestamptz;
ALTER TABLE subscription_payments ADD COLUMN IF NOT EXISTS note text;

CREATE TABLE IF NOT EXISTS promo_codes (
    id          bigserial PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    description text NOT NULL DEFAULT '',
    percent     int  NOT NULL DEFAULT 0,          -- 1..100
    tier_slug   text,                             -- NULL = tüm kademeler
    max_uses    int  NOT NULL DEFAULT 0,          -- 0 = sınırsız
    used_count  int  NOT NULL DEFAULT 0,
    expires_at  timestamptz,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_promo_codes_active ON promo_codes (is_active, expires_at);
