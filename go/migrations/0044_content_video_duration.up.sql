-- 0044_content_video_duration: Video içeriklerinde süre (saniye) — Faz 17.
-- NULL = süre bilinmiyor; kartta süre rozeti basılmaz.
ALTER TABLE contents
  ADD COLUMN IF NOT EXISTS video_seconds INT;