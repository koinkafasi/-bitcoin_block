-- 0084: abonelik katmanları (membership_tiers), ödeme kayıtları ve
-- içerik başına asgari katman (min_tier_weight).
--
-- Referans: edirnegercek (Laravel) kademe modeli — slug, renk, ağırlık,
-- aylık/yıllık fiyat, perks (JSON), aktiflik.

CREATE TABLE IF NOT EXISTS membership_tiers (
    id            bigserial PRIMARY KEY,
    slug          text NOT NULL UNIQUE,
    name          text NOT NULL,
    description   text NOT NULL DEFAULT '',
    color_hex     text NOT NULL DEFAULT '#71717a',
    weight        int  NOT NULL DEFAULT 1,
    monthly_price numeric(10,2) NOT NULL DEFAULT 0,
    annual_price  numeric(10,2) NOT NULL DEFAULT 0,
    perks         jsonb NOT NULL DEFAULT '[]'::jsonb,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_membership_tiers_weight ON membership_tiers (weight);

-- Kullanıcı aboneliği (kademe + bitiş + fatura dönemi)
ALTER TABLE users ADD COLUMN IF NOT EXISTS membership_tier_id bigint
    REFERENCES membership_tiers(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS membership_expires_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS membership_billing_cycle text; -- monthly | annual

-- İçerik başına asgari katman ağırlığı (0 = herkese açık)
ALTER TABLE contents ADD COLUMN IF NOT EXISTS min_tier_weight int NOT NULL DEFAULT 0;
UPDATE contents SET min_tier_weight = 1 WHERE is_exclusive = TRUE AND min_tier_weight = 0;

-- Ödeme kayıtları (Stripe + iyzico ortak)
CREATE TABLE IF NOT EXISTS subscription_payments (
    id                 bigserial PRIMARY KEY,
    user_id            bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier_id            bigint REFERENCES membership_tiers(id) ON DELETE SET NULL,
    provider           text NOT NULL,              -- stripe | iyzico | manual
    conversation_id    text UNIQUE,                -- iyzico conversationId / stripe session
    token              text,                       -- iyzico checkout token (callback doğrulaması)
    provider_payment_id text,
    amount             numeric(10,2) NOT NULL DEFAULT 0,
    currency           text NOT NULL DEFAULT 'TRY',
    billing_cycle      text NOT NULL DEFAULT 'monthly',
    status             text NOT NULL DEFAULT 'pending', -- pending|success|failed|refunded
    raw_response       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_subscription_payments_user ON subscription_payments (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS ix_subscription_payments_status ON subscription_payments (status, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_subscription_payments_token ON subscription_payments (token) WHERE token IS NOT NULL;

-- Başlangıç kademeleri (edirnegercek fiyat aralığı referans alındı).
INSERT INTO membership_tiers (slug, name, description, color_hex, weight,
    monthly_price, annual_price, perks, is_active)
VALUES
 ('filiz', 'Filiz', 'Öğrenci ve yeni okuyucular için giriş seviyesi.',
  '#22c55e', 1, 150, 1500,
  '["Ayrıcalıklı haber özetleri","Reklamsız makale görünümü (kısmi)","Bülten"]'::jsonb, true),
 ('bronz', 'Bronz', 'Düzenli okuyucular için genişletilmiş erişim.',
  '#b45309', 2, 300, 3000,
  '["Filiz ayrıcalıkları","Analiz içerikleri","Okuma listesi senkronu"]'::jsonb, true),
 ('gumus', 'Gümüş', 'Ayrıcalıklı haber ve analizlerin tamamı.',
  '#94a3b8', 3, 750, 7500,
  '["Bronz ayrıcalıkları","Tüm analizler","E-gazete arşivi"]'::jsonb, true),
 ('altin', 'Altın', 'Öncelikli erişim ve reklamsız deneyim.',
  '#eab308', 4, 1200, 12000,
  '["Gümüş ayrıcalıkları","Reklamsız deneyim","Etkinlik önceliği"]'::jsonb, true),
 ('elmas', 'Elmas', 'En üst düzey destek ve sınırsız erişim.',
  '#6366f1', 5, 3000, 30000,
  '["Altın ayrıcalıkları","Sınırsız erişim","Editör buluşmaları","Yıllık özel rapor"]'::jsonb, true)
ON CONFLICT (slug) DO NOTHING;

-- Mevcut Stripe premium üyeleri en üst kademeye taşınır (erişim kaybı olmasın).
UPDATE users SET membership_tier_id = (SELECT id FROM membership_tiers WHERE slug = 'elmas')
WHERE membership = 'premium' AND membership_tier_id IS NULL;
UPDATE users SET membership_expires_at = membership_until
WHERE membership = 'premium' AND membership_expires_at IS NULL AND membership_until IS NOT NULL;
