-- 0069 geri alımı
UPDATE feeds
SET aktif=false
WHERE name IN (
    'Hürriyet Dünya', 'DHA', 'İHA', 'NTV Spor', 'Fanatik', 'Sporx',
    'DW Türkçe', 'Technopat', 'Bigpara'
);
