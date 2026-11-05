-- 0088 geri alınırsa dalga formu verisi kaldırılır.
ALTER TABLE audio_transcripts DROP COLUMN IF EXISTS waveform;
