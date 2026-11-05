-- Faz 194 — otomatik AI ile yazım (bot hattı).
--
-- Neden: bot hattı RSS özetini olduğu gibi taslağa yazıyor (ölçüm: yayındaki
-- 986 içeriğin 982'si TEK paragraf, 598'i 200 karakterin altında). AI ile
-- yazım yolu (`AI Kaynak Merkezi → küme → taslak`) vardı ama ELLE
-- tetikleniyordu. Bu faz onu isteğe bağlı bir otomatik hatta çeviriyor:
-- ayar açıksa bot taslakları AI'dan geçip gerçek haber metnine dönüşür.
--
-- `applied_at`: AI sonucunun içeriğe YAZILDIĞI an. Olmadan işçi aynı sonucu
-- her turda yeniden uygular (version şişer, editör düzeltmeleri ezilir).
ALTER TABLE ai_jobs ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ;

-- Ayar aktifken her turda "tamamlanmış ama uygulanmamış" işler aranır.
CREATE INDEX IF NOT EXISTS ix_ai_jobs_uygulanmamis
    ON ai_jobs (status, action, id)
    WHERE applied_at IS NULL;

-- "Bu içerik için zaten AI yazımı yapıldı mı?" sorgusu için.
CREATE INDEX IF NOT EXISTS ix_ai_jobs_icerik_eylem
    ON ai_jobs (content_id, action, status);
