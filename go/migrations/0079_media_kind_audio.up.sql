-- 0079: native ses yükleme desteği — media_files.kind kontrolüne 'audio' eklendi.
-- Yalnız görsel/video/doküman yüklenebiliyordu; ses dosyaları (mp3/wav/…)
-- CHECK kısıtı yüzünden sessizce kaydedilemiyordu.
ALTER TABLE media_files DROP CONSTRAINT IF EXISTS media_files_kind_check;
ALTER TABLE media_files ADD CONSTRAINT media_files_kind_check
    CHECK (kind IN ('image','video','audio','document','file'));