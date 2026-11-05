-- 0114: kripto dışı kaynakları kapat, içeriklerini arşivle
--
-- ÖLÇÜM (5 Kasım 2026, ilk toplama): theblock'tan devralınan 15 Türkçe
-- teknoloji/genel haber feed'i (GZT, NTV Teknoloji, Hürriyet, Webtekno,
-- ShiftDelete, Chip, LOG, DonanımHaber, Technopat, Tamindir…) da aktifti ve
-- tek turda 554 içeriğin ~420'si bunlardan geldi. Kripto portalında yeri yok:
-- okuyucu "Bitcoin" ararken "Chip dergisi" haberi görürse güven kaybeder.
--
-- Kaynaklar SİLİNMEZ: `aktif=false` + not yazılır (geri açmak tek UPDATE).
-- İçerikleri `arsiv` durumuna alınır (silinmez; geri dönüş mümkün).

UPDATE feeds
SET aktif = FALSE,
    notes = CASE WHEN notes = '' THEN 'kripto dışı — arşivlendi (0114)'
                 ELSE notes || ' | kripto dışı — arşivlendi (0114)' END,
    updated_at = now()
WHERE url IN (
  'https://www.gzt.com/rss?category=teknoloji',
  'https://www.gzt.com/rss?category=oyun',
  'https://www.ntv.com.tr/teknoloji.rss',
  'https://www.cnnturk.com/feed/rss/teknoloji',
  'https://www.hurriyet.com.tr/rss/teknoloji',
  'https://www.webtekno.com/rss.xml',
  'https://shiftdelete.net/feed',
  'https://www.technopat.net',
  'https://www.donanimhaber.com/site-sitemaps/news-sitemap',
  'https://www.chip.com.tr/rss',
  'https://www.tamindir.com/feed/',
  'https://www.log.com.tr/feed/'
);

-- Bu kaynaklardan gelmiş içerikleri arşivle.
UPDATE contents
SET status = 'arsiv', updated_at = now()
WHERE status IN ('taslak','yayinda')
  AND source_name IN (
    'GZT Teknoloji','GZT Oyun','NTV Teknoloji','CNN Teknoloji','Hürriyet Teknoloji',
    'Webtekno','ShiftDelete','Technopat','DonanımHaber','Chip','Tamindir','LOG'
  );

-- Çalışmayan iki kaynağı da kapat (ölçüldü: TLS reset / HTML döndü).
UPDATE feeds
SET aktif = FALSE,
    notes = 'ölçüm: 5 Kasım 2026 — TLS reset / RSS değil',
    updated_at = now()
WHERE url IN ('https://www.coinbase.com/blog/rss.xml', 'https://tether.to/en/news/feed/');
