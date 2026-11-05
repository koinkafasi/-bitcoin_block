-- 0093: Yayın önceliği eksenleri (spec 11 §5 ölçümünün sonucu).
--
-- Eski yapıda tek bir `ai_score` vardı ve yalnız üstveri eksiksizliğini
-- ölçüyordu. 29 Ekim 2026 ölçümü: 34.156 taslağın %87,9'u tek bir 60–79
-- bandında. Tek sayı hem "yayına hazır mı" hem "yayınlamaya değer mi"
-- sorusunu yanıtlamaya çalışıp ikisini de yanıtlayamıyordu.
--
-- Eksenler ayrıldı (internal/scoring):
--   ai_score      → TAMLIK   : yayına hazır mı (yayın kapısı burayı okur)
--   score_trend   → TREND    : şu an aranıyor mu (trend_matches)
--   score_guven   → GÜVEN    : kaynağa güveniliyor mu (feeds.trust)
--   (tazelik)     → sütun YOK: zamanın fonksiyonu, sorgu anında hesaplanır
--
-- Tazelik bilerek sütuna yazılmaz. Yazılsaydı her saat bayatlar ve 34 bin
-- satırı sürekli güncelleyen bir iş gerekirdi; iki koşum arasında sıralama
-- yanlış kalırdı. created_at zaten burada, hesap sorgu anında yapılır
-- (scoring.TazelikSQL / scoring.OncelikSQL).

ALTER TABLE contents
    ADD COLUMN IF NOT EXISTS score_trend    smallint    NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS score_guven    smallint    NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS score_hesap_at timestamptz;

COMMENT ON COLUMN contents.ai_score IS
    'TAMLIK ekseni (0-100): başlık/spot/etiket/kapak/kaynak eksiksizliği. '
    'Yayın kapısı (categories.min_score) bunu okur. Alaka ya da AI yargısı DEĞİLDİR.';
COMMENT ON COLUMN contents.score_trend IS
    'TREND ekseni (0-100): trend_matches eşleşme gücü × terimin güncelliği. Eşleşme yoksa 0.';
COMMENT ON COLUMN contents.score_guven IS
    'GÜVEN ekseni (0-100): kaynağın feeds.trust değeri; agregatör akışlarda düşürülür.';
COMMENT ON COLUMN contents.score_hesap_at IS
    'Eksenlerin en son ne zaman hesaplandığı. Tazelik buna bakmaz; o sorgu anında hesaplanır.';

-- Triyaj sıralaması site + durum içinde çalışır; tazelik sorgu anında
-- hesaplandığı için önceliğin tamamı indekslenemez. Bu indeks, sıralamanın
-- taradığı kümeyi (site+durum) daraltır ve eşitlik bozucuyu hazır tutar.
CREATE INDEX IF NOT EXISTS ix_contents_triyaj
    ON contents (site_id, status, score_trend DESC, created_at DESC);
