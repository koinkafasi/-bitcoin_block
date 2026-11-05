-- 0081: Stripe abonelik altyapısı
-- users: Stripe müşteri/abonelik kimlikleri + üyelik bitişi
-- stripe_events: webhook idempotens kaydı (aynı olay iki kez işlenmesin)
ALTER TABLE users ADD COLUMN IF NOT EXISTS stripe_customer_id text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS stripe_subscription_id text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS membership_until timestamptz;

CREATE TABLE IF NOT EXISTS stripe_events (
    id          text PRIMARY KEY,
    received_at timestamptz NOT NULL DEFAULT now()
);