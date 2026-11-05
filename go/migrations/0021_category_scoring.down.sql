-- theblock — 0021 geri alma
DROP TABLE IF EXISTS trends;
ALTER TABLE contents DROP COLUMN IF EXISTS ai_score;
ALTER TABLE categories
    DROP COLUMN IF EXISTS min_score,
    DROP COLUMN IF EXISTS homepage_min_score,
    DROP COLUMN IF EXISTS icon,
    DROP COLUMN IF EXISTS image_service,
    DROP COLUMN IF EXISTS name_en,
    DROP COLUMN IF EXISTS name_es;