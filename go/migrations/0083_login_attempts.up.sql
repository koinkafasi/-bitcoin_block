-- 0083: üye girişi hız sınırı denemeleri.
-- 0082 uygulandıktan sonra eklendiği için ayrı migration dosyasıdır.
CREATE TABLE IF NOT EXISTS login_attempts (
    id           bigserial PRIMARY KEY,
    email        text NOT NULL,
    success      boolean NOT NULL DEFAULT false,
    attempted_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_email ON login_attempts (email, attempted_at);
