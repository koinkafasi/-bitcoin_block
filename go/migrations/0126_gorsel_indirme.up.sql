-- 0126_gorsel_indirme — kaynak görsellerinde indirme takibi.
--
-- DURUM (12 Kasım 2026): `source_images` 4.918 kayıt tutuyor ama yalnız
-- 136'sı diske inmiş (%3). Kalan 4.782 görsel hâlâ UZAK kaynağa bağlı:
-- `/kg/<hash>` ilk istendiğinde indiriyor, istenmezse hiç inmiyor.
--
-- RİSK ÖLÇÜLDÜ: eski WordPress görselleri (cdn.bitcosar.com) kaynakta
-- KAYBOLDU, sunucu 403 dönüyor (bkz. 0123_gorsel_yedek). Aynı şey RSS
-- kaynakları için de olacak — haber siteleri görselleri taşır veya siler.
-- O an gelmeyen görsel bir daha gelmez.
--
-- ÇÖZÜM: arka planda önden indirme. Bu göç, o işin sonsuz döngüye
-- girmemesi için deneme sayacı ekliyor: ölü bir adres 5 denemeden sonra
-- bırakılır, her turda yeniden denenmez.
ALTER TABLE source_images
  ADD COLUMN IF NOT EXISTS indirme_deneme  integer     NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS son_deneme_at   timestamptz,
  ADD COLUMN IF NOT EXISTS son_hata        text;

COMMENT ON COLUMN source_images.indirme_deneme IS
  'Başarısız indirme sayısı; 5''e ulaşınca önden indirme bu kaydı atlar';
COMMENT ON COLUMN source_images.son_deneme_at IS
  'Son indirme denemesi — aynı adrese arka arkaya yüklenmemek için';

-- Önden indirme sırası: hiç inmemiş ve denemesi tükenmemiş kayıtlar.
CREATE INDEX IF NOT EXISTS ix_source_images_indirilecek
  ON source_images (indirme_deneme, son_deneme_at)
  WHERE bytes = 0;
