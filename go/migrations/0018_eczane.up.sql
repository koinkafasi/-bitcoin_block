-- theblock — P3 Nöbetçi Eczane ayarları (0018)
INSERT INTO settings (section, key, value) VALUES
    ('servis','eczane_kaynak','""'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
