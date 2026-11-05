-- theblock — AI sohbet, kullanım/maliyet logları ve şablonlar (0006)
CREATE TABLE ai_conversations (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT 'Yeni sohbet',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_messages (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('user','assistant','system')),
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_ai_messages_conv ON ai_messages (conversation_id, created_at);

-- Her AI çağrısı denetlenebilir: sağlayıcı/model + tahmini token/maliyet
CREATE TABLE ai_usage_logs (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id          BIGINT REFERENCES users(id) ON DELETE SET NULL,
    content_id       BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    provider         TEXT NOT NULL,
    action           TEXT NOT NULL,
    model            TEXT,
    prompt_tokens    INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    cost_est         NUMERIC(12,4) NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_ai_usage_created ON ai_usage_logs (created_at DESC);
CREATE INDEX ix_ai_usage_provider ON ai_usage_logs (provider);

CREATE TABLE ai_templates (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind       TEXT NOT NULL DEFAULT 'sohbet' CHECK (kind IN ('sohbet','sag_tik')),
    title      TEXT NOT NULL,
    prompt     TEXT NOT NULL,
    icon       TEXT NOT NULL DEFAULT '🤖',
    sort_order INT NOT NULL DEFAULT 0
);
