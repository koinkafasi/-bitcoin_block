-- Otomatik yayın damgası.
--
-- NEDEN AYRI SÜTUN (published_at yeterli değil): hangi içeriğin insan
-- onayı olmadan çıktığını ayırt etmek gerekiyor. İki iş yapar:
--   1) Günlük limit sayacı — ayrı bir sayaç tablosu tutmaya gerek kalmaz,
--      kayma olmaz (tek kaynak).
--   2) Denetim izi — otomatik yayının etkisi sonradan ölçülebilir ve
--      gerekirse yalnız o içerikler geri alınabilir.
ALTER TABLE contents
  ADD COLUMN IF NOT EXISTS auto_published_at timestamptz;

-- Kısmi indeks: yalnız otomatik yayınlananlar. Günlük sayaç sorgusu
-- (auto_published_at >= date_trunc('day', now())) her turda çalışıyor;
-- tam indeks 3.800+ yayın satırının tamamını kapsardı, oysa otomatik
-- yayınlananlar küçük bir alt küme.
CREATE INDEX IF NOT EXISTS contents_auto_published_idx
  ON contents (auto_published_at DESC)
  WHERE auto_published_at IS NOT NULL;
