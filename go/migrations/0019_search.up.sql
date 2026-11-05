-- theblock — P4 Arama: pg_trgm dizinleri (0019)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX ix_contents_title_trgm ON contents USING gin (title gin_trgm_ops);
CREATE INDEX ix_contents_spot_trgm  ON contents USING gin (spot gin_trgm_ops);
