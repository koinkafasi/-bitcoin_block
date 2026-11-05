-- theblock — P4 Paywall / üyelik (0020)
ALTER TABLE users ADD COLUMN membership TEXT NOT NULL DEFAULT '';

INSERT INTO settings (section, key, value) VALUES
    ('paywall','aktif','true'::jsonb),
    ('paywall','limit','3'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
