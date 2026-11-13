-- Kaynak görsellerini medya kütüphanesine bağla.
--
-- DURUM (13 Kasım 2026): RSS'ten gelen görseller `source_images` tablosuna
-- ve `{depo}/kaynak/` ağacına iniyor (712 görsel, 152 MB) ama medya
-- kütüphanesine HİÇ girmiyor: `media_files` yalnız 39 kayıt tutuyor
-- (brand 30, upload 9). Yani görseller bizim diskimizde ama bizim
-- kütüphanemizde değil — düzenlenemiyor, alt metni yazılamıyor,
-- yeniden üretim için kuyruğa alınamıyor.
--
-- NEDEN BU SÜTUN: aktarımın iki kez çalışmasını önlemek ve hangi kaynak
-- görselin hangi medya kaydına karşılık geldiğini bilmek için. Eşleşmeyi
-- dosya adından türetmek kırılgan olurdu (SaveMedia rastgele ad veriyor).
ALTER TABLE source_images
  ADD COLUMN IF NOT EXISTS media_id BIGINT;

-- Aktarım kuyruğu sorgusu: "indirilmiş ama medyaya alınmamış".
-- Kısmi indeks, çünkü aktarılanlar zamanla çoğunluk olacak ve sorgu
-- yalnız NULL olanları arıyor.
CREATE INDEX IF NOT EXISTS ix_source_images_medyasiz
  ON source_images (created_at DESC)
  WHERE media_id IS NULL AND bytes > 0;

-- Ters yön: bir medya kaydının kaynağını bulmak (AI görsel üretici
-- gelince `source='internet'` olanları yeniden üretmek için).
CREATE INDEX IF NOT EXISTS ix_source_images_media_id
  ON source_images (media_id)
  WHERE media_id IS NOT NULL;
