-- Gazete modülü (Faz 1): site lisansı, depolama, şablon/sayı/sayfa, abonelik, yetki.

-- Site bazlı özellik lisansları (ticari paket anahtarları).
CREATE TABLE IF NOT EXISTS site_features (
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    feature_key TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (site_id, feature_key)
);

-- Site bazlı nesne depolama yapılandırması (yerel disk / Cloudflare R2 / S3).
CREATE TABLE IF NOT EXISTS site_storage (
    site_id BIGINT PRIMARY KEY REFERENCES sites(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT 'local' CHECK (provider IN ('local','r2','s3','minio')),
    endpoint TEXT NOT NULL DEFAULT '',
    bucket TEXT NOT NULL DEFAULT '',
    region TEXT NOT NULL DEFAULT 'auto',
    access_key_id TEXT NOT NULL DEFAULT '',
    secret_enc TEXT NOT NULL DEFAULT '',
    public_base_url TEXT NOT NULL DEFAULT '',
    path_prefix TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Gazete şablonları (10 sayfalık düzen tanımı, JSONB sayfa blokları).
CREATE TABLE IF NOT EXISTS gazete_templates (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    pages JSONB NOT NULL DEFAULT '[]'::jsonb,
    thumb_url TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_gazete_templates_site ON gazete_templates(site_id);

-- Gazete sayıları (baskılar): günlük/haftalık/aylık periyotta tarih damgalı.
CREATE TABLE IF NOT EXISTS gazete_issues (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    template_id BIGINT REFERENCES gazete_templates(id) ON DELETE SET NULL,
    period TEXT NOT NULL DEFAULT 'gunluk' CHECK (period IN ('gunluk','haftalik','aylik')),
    issue_date DATE NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'taslak' CHECK (status IN ('taslak','hazir','yayinda','arsiv')),
    assigned_editor_id BIGINT,
    cover_media_id BIGINT,
    pdf_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    UNIQUE (site_id, period, issue_date)
);
CREATE INDEX IF NOT EXISTS idx_gazete_issues_site_date ON gazete_issues(site_id, issue_date DESC);

-- Gazete sayı sayfaları (düzenlenebilir sayfa yerleşimi).
CREATE TABLE IF NOT EXISTS gazete_pages (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES gazete_issues(id) ON DELETE CASCADE,
    page_no INT NOT NULL CHECK (page_no BETWEEN 1 AND 50),
    page_key TEXT NOT NULL DEFAULT '',
    blocks JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (issue_id, page_no)
);

-- Gazete abonelikleri (özel bülten aboneleri).
CREATE TABLE IF NOT EXISTS gazete_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    channel TEXT NOT NULL DEFAULT 'email' CHECK (channel IN ('email','whatsapp')),
    frequency TEXT NOT NULL DEFAULT 'gunluk' CHECK (frequency IN ('gunluk','haftalik','aylik')),
    status TEXT NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif','iptal')),
    token TEXT NOT NULL DEFAULT '',
    opt_in_at TIMESTAMPTZ,
    opt_in_ip TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_gazete_subs_email ON gazete_subscriptions(site_id, lower(email)) WHERE email <> '';
CREATE UNIQUE INDEX IF NOT EXISTS uq_gazete_subs_phone ON gazete_subscriptions(site_id, phone) WHERE phone <> '';

-- Gönderim kayıtları (e-posta / WhatsApp teslim takibi).
CREATE TABLE IF NOT EXISTS gazete_deliveries (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES gazete_issues(id) ON DELETE CASCADE,
    subscription_id BIGINT NOT NULL REFERENCES gazete_subscriptions(id) ON DELETE CASCADE,
    channel TEXT NOT NULL DEFAULT 'email',
    status TEXT NOT NULL DEFAULT 'kuyrukta' CHECK (status IN ('kuyrukta','gonderildi','hata')),
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ,
    UNIQUE (issue_id, subscription_id)
);

-- Gazete ekip yetkileri (site bazlı editör ataması).
CREATE TABLE IF NOT EXISTS gazete_user_roles (
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rol TEXT NOT NULL DEFAULT 'editor' CHECK (rol IN ('sorumlu','editor','izleyici')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (site_id, user_id)
);