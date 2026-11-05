DELETE FROM categories
WHERE parent_id IS NOT NULL
  AND slug IN ('afet-kaza','analiz','guvenlik','ozgun','toplum','ramazan');

UPDATE categories
SET slug = 'guncel', name = 'Güncel'
WHERE slug = 'gundem'
  AND NOT EXISTS (SELECT 1 FROM categories WHERE slug = 'guncel');

UPDATE settings
SET value = replace(value::text, '"gundem"', '"guncel"')::jsonb
WHERE value::text LIKE '%"gundem"%';
