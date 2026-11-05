-- Faz 194 geri alma: yalnız indeks ve kolon düşer.
-- Üretilmiş AI sonuçları ve yazılmış içerikler SİLİNMEZ.
DROP INDEX IF EXISTS ix_ai_jobs_icerik_eylem;
DROP INDEX IF EXISTS ix_ai_jobs_uygulanmamis;
ALTER TABLE ai_jobs DROP COLUMN IF EXISTS applied_at;
