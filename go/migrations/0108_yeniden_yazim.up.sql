-- Faz 204 — yayındaki ince içerikler için AI ile yeniden yazım önerileri.
--
-- Neden ayrı tablo: `contents` canlı metindir, `content_revisions` ise
-- KAYDEDİLMİŞ sürümlerin geçmişidir. AI önerisi ikisi de değil: yayımlanmış
-- metni değiştirmeden önce editörün göreceği TASLAK. Öneri burada durur;
-- onaylanınca `contents` güncellenir ve eski hâli `content_revisions`a düşer.
--
-- Neden `ai_jobs` yetmiyor: iş kuyruğu tek çağrının sonucunu tutar, ama
-- karar (onay/red) içerik başına ve dil başına kalıcı olmalı. Ayrıca aynı
-- içerik için ikinci kez öneri üretilmesini engelleyen TEK AKTİF satır
-- kuralı burada.
CREATE TABLE IF NOT EXISTS yeniden_yazim (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id              BIGINT NOT NULL DEFAULT 1,
    content_id           BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    -- Çeviri kökü: aynı haberin TR/EN satırlarını tek dosyada toplar.
    root_id              BIGINT NOT NULL,
    locale               TEXT NOT NULL,
    -- bekliyor | kuyrukta | hazir | hata | uygulandi | reddedildi
    status               TEXT NOT NULL DEFAULT 'bekliyor',
    -- Öneri üretilirken içeriğin sürümü. Onayda bu sürüm tutmuyorsa
    -- (editör arada kaydettiyse) yazma reddedilir: AI kimsenin değişikliğini
    -- ezmez.
    content_version      INT NOT NULL DEFAULT 0,

    old_title            TEXT NOT NULL DEFAULT '',
    old_spot             TEXT NOT NULL DEFAULT '',
    old_detail           JSONB NOT NULL DEFAULT '[]'::jsonb,
    old_seo_title        TEXT NOT NULL DEFAULT '',
    old_seo_description  TEXT NOT NULL DEFAULT '',

    new_title            TEXT NOT NULL DEFAULT '',
    new_spot             TEXT NOT NULL DEFAULT '',
    new_detail           JSONB NOT NULL DEFAULT '[]'::jsonb,
    new_seo_title        TEXT NOT NULL DEFAULT '',
    new_seo_description  TEXT NOT NULL DEFAULT '',

    -- Kalite kapısı uyarıları (JSON dizi). Boş dizi = temiz öneri.
    warnings             JSONB NOT NULL DEFAULT '[]'::jsonb,
    error                TEXT NOT NULL DEFAULT '',
    provider             TEXT NOT NULL DEFAULT '',
    model                TEXT NOT NULL DEFAULT '',
    prompt_tokens        INT NOT NULL DEFAULT 0,
    completion_tokens    INT NOT NULL DEFAULT 0,
    attempts             INT NOT NULL DEFAULT 0,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_at           TIMESTAMPTZ
);

-- Tek aktif öneri kuralı: aynı içerik için bekleyen/kuyrukta/hazır/hata
-- durumunda yalnız BİR satır olabilir. Onaylanmış/reddedilmiş satırlar
-- geçmiş olarak kalır; yeni tur onları engellemez.
CREATE UNIQUE INDEX IF NOT EXISTS yeniden_yazim_aktif_uidx
    ON yeniden_yazim (content_id)
 WHERE status IN ('bekliyor', 'kuyrukta', 'hazir', 'hata');

-- Kuyruk işçisi: en eski bekleyeni al.
CREATE INDEX IF NOT EXISTS yeniden_yazim_kuyruk_idx
    ON yeniden_yazim (status, id);

-- Panel: kök + dil bazında geçmiş.
CREATE INDEX IF NOT EXISTS yeniden_yazim_root_idx
    ON yeniden_yazim (root_id, locale, id DESC);
