-- 0029_newspapers (down)
DROP TRIGGER IF EXISTS trg_newspaper_updated ON newspaper_pages;
DROP TABLE IF EXISTS newspaper_pages;