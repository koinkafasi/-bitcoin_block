-- SEO başlık onarım damgası.
--
-- NEDEN: 1491 yayın içeriğinin SEO başlığı Google'ın kestiği sınırın
-- üstünde (788'i 70 karakterden uzun, en uzunu 141). Hepsi eski
-- WordPress göçünden geliyor ve 782'sinde seo_title = title.
--
-- Damga iki işi birden yapar:
--   1) Aynı içeriğin tekrar tekrar onarılmasını önler. AI her çağrıda
--      biraz farklı başlık üretir; damga olmadan sonsuz döngü olurdu.
--   2) Günlük limit sayacını besler (yayındaki meta değişikliği arama
--      görünürlüğünü etkiler; kademeli gitmek gerekiyor).
ALTER TABLE contents
  ADD COLUMN IF NOT EXISTS seo_onarim_at timestamptz;

-- Kısmi indeks: onarım kuyruğu sorgusu "seo_onarim_at IS NULL AND
-- length(seo_title) > 70" biçiminde. Tam indeks 3.836 satırın tamamını
-- kapsardı; onarılmış olanlar ise küçük ve büyüyen bir alt küme.
CREATE INDEX IF NOT EXISTS contents_seo_onarim_idx
  ON contents (seo_onarim_at)
  WHERE seo_onarim_at IS NOT NULL;
