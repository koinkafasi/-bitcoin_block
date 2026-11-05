ALTER TABLE yeniden_yazim
    DROP COLUMN IF EXISTS mod,
    DROP COLUMN IF EXISTS sources;

DROP INDEX IF EXISTS yeniden_yazim_mod_idx;
