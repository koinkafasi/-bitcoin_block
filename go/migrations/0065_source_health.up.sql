-- 0065: kaynak sağlığı düzeltmeleri
UPDATE feeds
SET url='https://www.odatv.com/rss.xml', homepage='https://www.odatv.com', last_status='', last_error=''
WHERE url='https://odatv4.com/rss';

-- Hürriyet Dünya RSS'i aralıklı olarak HTML dönüyor; genel Hürriyet RSS'i yeterli.
UPDATE feeds
SET aktif=FALSE, notes='RSS aralıklı HTML dönüyor; genel Hürriyet RSS kullanılabilir'
WHERE url='https://www.hurriyet.com.tr/rss/dunya';
