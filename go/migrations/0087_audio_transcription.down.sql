-- 0087 geri alınırsa transkripsiyon tabloları ve private görünürlük kaldırılır.
DROP TRIGGER IF EXISTS trg_audio_transcripts_search ON audio_transcripts;
DROP FUNCTION IF EXISTS audio_transcripts_search_update();
DROP TABLE IF EXISTS audio_transcripts;
DROP TABLE IF EXISTS transcription_jobs;
DROP INDEX IF EXISTS idx_media_files_visibility;
ALTER TABLE media_files DROP CONSTRAINT IF EXISTS media_files_visibility_check;
ALTER TABLE media_files DROP COLUMN IF EXISTS visibility;
