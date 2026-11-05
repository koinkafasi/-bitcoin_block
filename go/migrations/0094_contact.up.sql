-- 0094: İletişim formu mesajları + sayfa metinlerinde yer tutucuya geçiş.
--
-- Amaç: e-posta adresleri ve alan adı sayfa metinlerine gömülüydü. Alan adı
-- değişince tüm sayfaları elle düzeltmek gerekiyordu. Artık metinlerde
-- {domain}, {eposta_info}, {eposta_reklam}… yer tutucuları kullanılır;
-- değerler Yönetim → İletişim ekranından tek seferde yönetilir.

CREATE TABLE IF NOT EXISTS contact_messages (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id    BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL,
    phone      TEXT,
    subject    TEXT NOT NULL DEFAULT 'diger',
    message    TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'yeni' CHECK (status IN ('yeni','okundu','arsiv')),
    ip         TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS contact_messages_site_idx
    ON contact_messages (site_id, status, created_at DESC);

COMMENT ON TABLE contact_messages IS
    'İletişim formundan gelen okuyucu mesajları; yönetimden okundu/arşiv/sil.';

-- Yayındaki kurumsal sayfalarda gömülü adresleri yer tutucuya çevir.
-- Sıra önemli: önce adresler, sonra kalan çıplak alan adı.
DO $$
DECLARE
    r RECORD;
    txt TEXT;
BEGIN
    FOR r IN SELECT id, detail::text AS body FROM pages LOOP
        txt := r.body;
        txt := replace(txt, 'info@theblock.com.tr',     '{eposta_info}');
        txt := replace(txt, 'haber@theblock.com.tr',    '{eposta_haber}');
        txt := replace(txt, 'reklam@theblock.com.tr',   '{eposta_reklam}');
        txt := replace(txt, 'destek@theblock.com.tr',   '{eposta_destek}');
        txt := replace(txt, 'telif@theblock.com.tr',    '{eposta_telif}');
        txt := replace(txt, 'duzeltme@theblock.com.tr', '{eposta_duzeltme}');
        txt := replace(txt, 'kariyer@theblock.com.tr',  '{eposta_kariyer}');
        txt := replace(txt, 'theblock.com.tr',          '{domain}');
        IF txt IS DISTINCT FROM r.body THEN
            UPDATE pages SET detail = txt::jsonb, updated_at = now() WHERE id = r.id;
        END IF;
    END LOOP;
END $$;

-- Sayfa metni override'ları (Sayfa Tasarımcısı) da aynı yer tutuculara geçer.
UPDATE page_texts SET value = replace(replace(replace(replace(replace(replace(replace(
        value,
        'info@theblock.com.tr',     '{eposta_info}'),
        'haber@theblock.com.tr',    '{eposta_haber}'),
        'reklam@theblock.com.tr',   '{eposta_reklam}'),
        'destek@theblock.com.tr',   '{eposta_destek}'),
        'telif@theblock.com.tr',    '{eposta_telif}'),
        'duzeltme@theblock.com.tr', '{eposta_duzeltme}'),
        'kariyer@theblock.com.tr',  '{eposta_kariyer}')
WHERE value LIKE '%@theblock.com.tr%';

UPDATE page_texts SET value = replace(value, 'theblock.com.tr', '{domain}')
WHERE value LIKE '%theblock.com.tr%';

-- Varsayılan iletişim ayarı (alan adı + adresler). Yönetimden değiştirilebilir.
INSERT INTO settings (section, key, value)
VALUES ('iletisim', 'ana', '{
  "domain": "theblock.com.tr",
  "info": "info@theblock.com.tr",
  "haber": "haber@theblock.com.tr",
  "reklam": "reklam@theblock.com.tr",
  "destek": "destek@theblock.com.tr",
  "telif": "telif@theblock.com.tr",
  "duzeltme": "duzeltme@theblock.com.tr",
  "kvkk": "info@theblock.com.tr",
  "kariyer": "kariyer@theblock.com.tr",
  "telefon": "",
  "adres": "",
  "calisma": "Haber merkezi 7/24 açık · Diğer talepler hafta içi 09.00–18.00",
  "form_aktif": true
}'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
