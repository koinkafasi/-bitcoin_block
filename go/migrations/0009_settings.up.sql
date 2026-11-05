-- theblock — Ayarlar: audit log + varsayılan ayar tamamlamaları (0009)
CREATE TABLE logs (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action     TEXT NOT NULL,
    detail     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_logs_created ON logs (created_at DESC);

-- çekirdek ayarlar eksiklerini tamamla (varsa dokunma)
INSERT INTO settings (section, key, value) VALUES
    ('gorsel',     'site_font',        '"system-ui, -apple-system, Segoe UI, Roboto"'::jsonb),
    ('gorsel',     'baslik_font',      '"Georgia, serif"'::jsonb),
    ('moduller',   'izinler',          '{}'::jsonb),
    ('baglantilar','meta',             '{}'::jsonb),
    ('baglantilar','adsense',          '""'::jsonb),
    ('baglantilar','robots_html',      '""'::jsonb),
    ('yapay_zeka', 'model',            '"auto"'::jsonb),
    ('yapay_zeka', 'base_url',         '""'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
