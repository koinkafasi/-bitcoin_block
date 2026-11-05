-- 0058: kategori taksonomisini SEO ve kullanım için toparla
--
-- - Üst menü sırası: Gündem, Siyaset, Ekonomi, Dünya, Spor, Yaşam,
--   Teknoloji, Eğitim-kültür. "Diğer" footer'a taşınır.
-- - Alt kategorilerde yalnızca çekirdek taksonomi + içeriği olanlar menüde
--   kalır; boş uzun kuyruk menü/footer'dan gizlenir.
-- - Boş kategori sayfaları uygulama tarafında noindex,follow olur ve
--   sitemap'e girmez.

UPDATE categories
SET sort_order = CASE slug
        WHEN 'gundem' THEN 1
        WHEN 'siyaset' THEN 2
        WHEN 'ekonomi' THEN 3
        WHEN 'dunya' THEN 4
        WHEN 'spor' THEN 5
        WHEN 'yasam' THEN 6
        WHEN 'teknoloji' THEN 7
        WHEN 'egitim-kultur' THEN 8
        WHEN 'diger' THEN 99
        ELSE sort_order END,
    in_header = CASE WHEN slug = 'diger' THEN FALSE ELSE TRUE END,
    in_menu = TRUE,
    in_footer = TRUE
WHERE parent_id IS NULL;

UPDATE categories
SET in_menu = FALSE, in_footer = FALSE
WHERE parent_id IS NOT NULL;

UPDATE categories
SET in_menu = TRUE, in_footer = TRUE,
    sort_order = CASE slug
        WHEN 'asayis' THEN 1 WHEN 'ilceler' THEN 2 WHEN 'afet-kaza' THEN 3 WHEN 'analiz' THEN 4
        WHEN 'turkiye-ekonomisi' THEN 1 WHEN 'dunya-ekonomisi' THEN 2 WHEN 'piyasalar' THEN 3
        WHEN 'borsa' THEN 4 WHEN 'doviz' THEN 5 WHEN 'altin' THEN 6 WHEN 'kripto-para' THEN 7
        WHEN 'emtia' THEN 8 WHEN 'faiz' THEN 9 WHEN 'enflasyon' THEN 10 WHEN 'bankacilik' THEN 11
        WHEN 'finans' THEN 12 WHEN 'is-dunyasi' THEN 13 WHEN 'enerji' THEN 14
        WHEN 'otomotiv' THEN 15 WHEN 'emlak' THEN 16 WHEN 'tarim' THEN 17 WHEN 'sanayi-uretim' THEN 18
        WHEN 'saglik' THEN 1 WHEN 'turizm' THEN 2 WHEN 'aktuel' THEN 3 WHEN 'kadin' THEN 4
        WHEN 'kisisel-gelisim' THEN 5 WHEN 'yeme-icme' THEN 6 WHEN 'doga-kesif' THEN 7
        WHEN 'gezi-rotalari' THEN 8 WHEN 'hobi' THEN 9 WHEN 'mizah' THEN 10
        WHEN 'futbol' THEN 1 WHEN 'basketbol' THEN 2 WHEN 'dunya-kupasi' THEN 3
        WHEN 'atletizm' THEN 4 WHEN 'diger-sporlar' THEN 5 WHEN 'golf' THEN 6
        WHEN 'gures' THEN 7 WHEN 'extreme-sporlar' THEN 8
        WHEN 'sanat' THEN 1 WHEN 'edebiyat' THEN 2 WHEN 'kitap' THEN 3 WHEN 'sinema' THEN 4
        WHEN 'tarih' THEN 5 WHEN 'mimari' THEN 6 WHEN 'muzik' THEN 7 WHEN 'dusunce-fikir' THEN 8
        WHEN 'bilim' THEN 1 WHEN 'oyun' THEN 2 WHEN 'donanim' THEN 3 WHEN 'internet' THEN 4
        WHEN 'mobil-teknolojiler' THEN 5 WHEN 'yazilim' THEN 6 WHEN 'uzay' THEN 7
        WHEN 'robotik' THEN 8 WHEN 'e-spor' THEN 9 WHEN 'oyun-haberleri' THEN 10
        WHEN 'afrika' THEN 1 WHEN 'amerika' THEN 2 WHEN 'asya' THEN 3 WHEN 'avrupa' THEN 4
        WHEN 'ortadogu' THEN 5
        ELSE sort_order END
WHERE parent_id IS NOT NULL
  AND (
    slug IN (
      'asayis','ilceler','afet-kaza','analiz',
      'turkiye-ekonomisi','dunya-ekonomisi','piyasalar','borsa','doviz','altin',
      'kripto-para','emtia','faiz','enflasyon','bankacilik','finans','is-dunyasi',
      'enerji','otomotiv','emlak','tarim','sanayi-uretim',
      'saglik','turizm','aktuel','kadin','kisisel-gelisim','yeme-icme',
      'doga-kesif','gezi-rotalari','hobi','mizah',
      'futbol','basketbol','dunya-kupasi','atletizm','diger-sporlar','golf','gures','extreme-sporlar',
      'sanat','edebiyat','kitap','sinema','tarih','mimari','muzik','dusunce-fikir',
      'bilim','oyun','donanim','internet','mobil-teknolojiler','yazilim','uzay','robotik',
      'e-spor','oyun-haberleri',
      'afrika','amerika','asya','avrupa','ortadogu'
    )
    OR EXISTS (SELECT 1 FROM contents x WHERE x.category_id = categories.id AND x.status = 'yayinda')
  );
