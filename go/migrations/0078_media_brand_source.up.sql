-- Marka görselleri (logo/favicon/OG) source='brand' ile kaydedilir;
-- mevcut CHECK kısıtı 'brand' türüne izin vermiyordu ve yüklemeler
-- sessizce başarısız oluyordu (bkz. e2e/tests/admin_marka.test.js).
ALTER TABLE media_files DROP CONSTRAINT IF EXISTS media_files_source_check;
ALTER TABLE media_files ADD CONSTRAINT media_files_source_check
    CHECK (source IN ('upload','library','internet','render','generate','pexels','ai','brand'));