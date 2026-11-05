DROP TABLE IF EXISTS trend_fetch_runs;
DROP TABLE IF EXISTS trend_news;
DROP TABLE IF EXISTS trend_points;
DROP TABLE IF EXISTS trend_terms;
DROP TABLE IF EXISTS trend_sources;
DELETE FROM settings WHERE section = 'trend';
