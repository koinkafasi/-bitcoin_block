-- 0093 geri alma. ai_score'a dokunulmaz: o 0093'ten önce de vardı.
DROP INDEX IF EXISTS ix_contents_triyaj;

ALTER TABLE contents
    DROP COLUMN IF EXISTS score_trend,
    DROP COLUMN IF EXISTS score_guven,
    DROP COLUMN IF EXISTS score_hesap_at;

COMMENT ON COLUMN contents.ai_score IS NULL;
