-- 0027_redirects: 301 yönlendirme yönetimi (spec 01 §D7/J2, 03 P4-9)
-- P0'da oluşan `redirects` tablosu yalnızca (source_path PK) içeriyordu.
-- Çoklu site desteği + hit sayacı + arama/güncelleme için genişletilir.

ALTER TABLE redirects ADD COLUMN IF NOT EXISTS site_id BIGINT;
UPDATE redirects SET site_id = 1 WHERE site_id IS NULL;
ALTER TABLE redirects ALTER COLUMN site_id SET NOT NULL;
ALTER TABLE redirects ADD COLUMN IF NOT EXISTS hits BIGINT NOT NULL DEFAULT 0;
ALTER TABLE redirects ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE redirects ADD COLUMN IF NOT EXISTS note TEXT;

-- site_id'yi sites(id) ile ilişkilendir (varsa eski kısıtı değiştir)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'redirects_site_id_fkey'
  ) THEN
    ALTER TABLE redirects
      ADD CONSTRAINT redirects_site_id_fkey FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE;
  END IF;
END$$;

-- PK'yı (site_id, source_path) olacak şekilde değiştir
ALTER TABLE redirects DROP CONSTRAINT IF EXISTS redirects_pkey;
ALTER TABLE redirects ADD CONSTRAINT redirects_pkey PRIMARY KEY (site_id, source_path);

CREATE INDEX IF NOT EXISTS idx_redirects_site ON redirects (site_id);
CREATE INDEX IF NOT EXISTS idx_redirects_source_lower ON redirects (site_id, lower(source_path));