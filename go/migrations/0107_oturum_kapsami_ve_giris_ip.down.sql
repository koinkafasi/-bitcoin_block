DROP INDEX IF EXISTS idx_login_attempts_email_ip;

ALTER TABLE login_attempts DROP COLUMN IF EXISTS ip;

ALTER TABLE sessions DROP COLUMN IF EXISTS scope;
