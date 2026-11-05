-- Faz 190: editörden tetiklenen AI çeviri istekleri.
--
-- Neden tablo: AI çeviri 20–160 sn sürüyor (uzun haberlerde parça parça
-- çevrildiği için daha uzun). İsteği HTTP yanıtının içinde bekletmek iki
-- sınırı aşıyor:
--   * sunucu WriteTimeout 60 sn  → bağlantı kopuyor, tarayıcı HTTP 000 görür
--   * Cloudflare proxy ~100 sn    → 524 hatası
-- Bu yüzden düğme yalnız isteği kuyruğa yazar; arka plan işçisi (ceviri.Run)
-- işler; panel durumu 2 sn'de bir yoklar.
CREATE TABLE IF NOT EXISTS ceviri_istek (
    id         BIGSERIAL PRIMARY KEY,
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    locale     TEXT NOT NULL,
    -- bekliyor | calisiyor | bitti | hata
    durum      TEXT NOT NULL DEFAULT 'bekliyor',
    sonuc_id   BIGINT,
    hata       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Aynı içerik + dil için tek istek: düğmeye iki kez basmak iki çeviri
-- üretmemeli (maliyet ve kopya kayıt).
CREATE UNIQUE INDEX IF NOT EXISTS ceviri_istek_tek
    ON ceviri_istek (content_id, locale);

CREATE INDEX IF NOT EXISTS ceviri_istek_bekleyen
    ON ceviri_istek (durum, id);
