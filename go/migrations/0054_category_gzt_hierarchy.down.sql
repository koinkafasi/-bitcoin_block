UPDATE categories
SET parent_id = NULL, in_header = TRUE, in_menu = TRUE
WHERE slug IN ('asayis','ilceler','bilim','oyun','saglik','turizm','sanat');

DELETE FROM categories
WHERE parent_id IS NOT NULL
  AND slug IN ('turkiye-ekonomisi','dunya-ekonomisi','kripto-para','tarim','sanayi-uretim',
               'bankacilik','hisse-senetleri','yatirim-fonlari','banka-gise-kurlari','kap-haberleri',
               'futbol','basketbol','atletizm','dunya-kupasi','golf','gures','extreme-sporlar','diger-sporlar');
