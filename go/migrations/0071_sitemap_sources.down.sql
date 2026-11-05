-- 0071 geri alımı
UPDATE feeds
SET kind='passive',
    aktif=false,
    url=homepage,
    notes='HTML/JavaScript; özel adaptör bekliyor'
WHERE name IN ('BirGün', 'DonanımHaber');
