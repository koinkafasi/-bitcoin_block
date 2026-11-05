-- 0068: RSS yayınlamayan kaynaklar için HTML liste adaptörü
-- DHA, İHA, NTV Spor, Fanatik, Sporx, DW Türkçe, Technopat ve Bigpara
-- ana sayfalarından haber bağlantısı çıkarılabiliyor. Hürriyet Dünya için
-- doğrudan dünya section'ı kullanılıyor. MSN/BirGün/DonanımHaber'in ana
-- sayfaları JavaScript kabuğu döndürdüğü için pasif katalogda kalıyor.

UPDATE feeds
SET url='https://www.hurriyet.com.tr/dunya/',
    homepage='https://www.hurriyet.com.tr/dunya/',
    kind='html',
    notes='HTML section listesi + JSON-LD makale özeti'
WHERE name='Hürriyet Dünya';

UPDATE feeds
SET kind='html',
    notes='HTML ana sayfa listesi + JSON-LD makale özeti'
WHERE name IN (
    'DHA', 'İHA', 'NTV Spor', 'Fanatik', 'Sporx',
    'DW Türkçe', 'Technopat', 'Bigpara'
);

UPDATE feeds
SET kind='passive',
    notes='HTML/JavaScript; özel adaptör bekliyor'
WHERE name IN ('MSN Türkiye', 'BirGün', 'DonanımHaber');
