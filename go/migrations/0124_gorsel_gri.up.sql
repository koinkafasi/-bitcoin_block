-- 0124_gorsel_gri — haber görselinin listelerde siyah-beyaz gösterilmesi.
--
-- İHTİYAÇ: bazı haberlerin görseli sitenin genelinde (ana sayfa, kategori
-- listeleri, ilgili haberler, arama) siyah-beyaz görünsün; okuyucu yazıya
-- girdiğinde RENKLİ açılsın. Görsel tek kopya; değişiklik yalnız sunum
-- katmanında (CSS filtresi), dosyaya dokunulmuyor.
--
-- Varsayılan false: mevcut 3.300+ içeriğin görünümü değişmiyor.
ALTER TABLE contents ADD COLUMN IF NOT EXISTS image_grayscale boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN contents.image_grayscale IS
  'true ise kapak görseli LİSTELERDE siyah-beyaz basılır; yazı sayfasında renkli kalır';
