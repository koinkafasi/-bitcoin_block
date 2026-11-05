-- 0114 geri alımı: kaynakları yeniden aç, arşivlenen içerikleri geri al.
UPDATE feeds SET aktif = TRUE, notes = ''
WHERE notes LIKE '%0114%'
  AND url NOT IN ('https://www.coinbase.com/blog/rss.xml', 'https://tether.to/en/news/feed/');

UPDATE contents
SET status = 'yayinda', updated_at = now()
WHERE status = 'arsiv'
  AND source_name IN (
    'GZT Teknoloji','GZT Oyun','NTV Teknoloji','CNN Teknoloji','Hürriyet Teknoloji',
    'Webtekno','ShiftDelete','Technopat','DonanımHaber','Chip','Tamindir','LOG'
  );
