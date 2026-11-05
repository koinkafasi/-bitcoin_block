-- 0068 geri alımı: HTML adaptörlü kaynakları pasif kataloğa döndür.
UPDATE feeds
SET url='https://www.hurriyet.com.tr',
    homepage='https://www.hurriyet.com.tr',
    kind='html'
WHERE name='Hürriyet Dünya';

UPDATE feeds
SET kind='html'
WHERE name IN (
    'DHA', 'İHA', 'NTV Spor', 'Fanatik', 'Sporx',
    'DW Türkçe', 'Technopat', 'Bigpara', 'MSN Türkiye', 'BirGün', 'DonanımHaber'
);
