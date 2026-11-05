DROP INDEX IF EXISTS idx_categories_parent_id;

DELETE FROM categories
WHERE parent_id IS NOT NULL
  AND slug IN ('piyasalar','borsa','doviz','altin','emtia','faiz','enflasyon',
               'finans','emlak','enerji','otomotiv','is-dunyasi');

ALTER TABLE categories DROP COLUMN IF EXISTS parent_id;
