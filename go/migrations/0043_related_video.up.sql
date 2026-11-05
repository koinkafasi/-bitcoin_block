-- 0043_related_video: Makale için elle seçilen "Önerilen Video" (Faz 12b).
ALTER TABLE contents
  ADD COLUMN IF NOT EXISTS related_video_id BIGINT REFERENCES contents(id) ON DELETE SET NULL;
