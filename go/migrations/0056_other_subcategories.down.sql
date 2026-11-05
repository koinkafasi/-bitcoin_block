DELETE FROM categories
WHERE parent_id IS NOT NULL
  AND slug IN (
    'afrika','amerika','asya','avrupa','ortadogu',
    'donanim','internet','mobil-teknolojiler','robotik','teknoloji-haberleri','uzay','yazilim',
    'e-spor','mobil-oyunlar','oyun-haberleri','oyun-incelemeleri',
    'aktuel','genclik','kadin','kisisel-gelisim','doga-kesif','gezi-rotalari','sehir-rehberi',
    'seyahat-rehberi','yeme-icme','hayvanlar-alemi','hobi','mizah',
    'derleme-secki','dusunce-fikir','edebiyat','kitap','medya-dijital-kultur','mekan','muzik',
    'portreler','sinema','tarih','mimari'
  );
