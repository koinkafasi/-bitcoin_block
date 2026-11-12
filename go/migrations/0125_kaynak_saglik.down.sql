ALTER TABLE feeds
  DROP COLUMN IF EXISTS ardisik_hata,
  DROP COLUMN IF EXISTS last_ok_at,
  DROP COLUMN IF EXISTS gorsel_oran,
  DROP COLUMN IF EXISTS dil,
  DROP COLUMN IF EXISTS otomatik_kapandi;
