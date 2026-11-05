-- 0079 geri alınırsa 'audio' türüne sahip kayıtlar kısıtı ihlal eder;
-- önce bunlar 'file' türüne çevrilir, sonra eski kısıt geri kurulur.
UPDATE media_files SET kind = 'file' WHERE kind = 'audio';
ALTER TABLE media_files DROP CONSTRAINT IF EXISTS media_files_kind_check;
ALTER TABLE media_files ADD CONSTRAINT media_files_kind_check
    CHECK (kind IN ('image','video','document','file'));