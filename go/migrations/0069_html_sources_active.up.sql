-- 0069: HTML adaptörü olan katalog kaynaklarını taramaya aç
UPDATE feeds
SET aktif=true
WHERE name IN (
    'Hürriyet Dünya', 'DHA', 'İHA', 'NTV Spor', 'Fanatik', 'Sporx',
    'DW Türkçe', 'Technopat', 'Bigpara'
);

UPDATE feeds
SET aktif=false, kind='passive'
WHERE name IN ('MSN Türkiye', 'BirGün', 'DonanımHaber');
