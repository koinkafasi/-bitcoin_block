-- theblock P1/auth — oturum tablosu (0003)
-- RBAC kontrolü oturum üzerinden (httpOnly çerez + DB kaydı).
CREATE TABLE sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    user_agent TEXT
);

CREATE INDEX ix_sessions_user    ON sessions (user_id);
CREATE INDEX ix_sessions_expires ON sessions (expires_at);
