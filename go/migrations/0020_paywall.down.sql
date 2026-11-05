ALTER TABLE users DROP COLUMN IF EXISTS membership;
DELETE FROM settings WHERE section = 'paywall';
